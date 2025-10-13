package lib

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code-search/src/models"
)

// IncrementalIndexer handles enhanced incremental indexing with change detection
type IncrementalIndexer struct {
	metadataPath    string
	fileMetadata    map[string]*FileMetadata
	dependencyGraph map[string]*FileDependencies
	mu              sync.RWMutex
	options         IncrementalOptions
	hashPool        *BufferPool
	changeEvents    chan FileChangeEvent
	eventBuffer     []FileChangeEvent
	bufferSize       int
}

// IncrementalOptions contains options for incremental indexing
type IncrementalOptions struct {
	EnableHashing      bool          `json:"enable_hashing"`
	EnableDependencies bool          `json:"enable_dependencies"`
	HashCacheSize       int           `json:"hash_cache_size"`
	ChangeBufferSize    int           `json:"change_buffer_size"`
	DebounceDelay       time.Duration `json:"debounce_delay"`
	MaxFileSizeHash    int64         `json:"max_file_size_hash"`
	ExcludePatterns     []string      `json:"exclude_patterns"`
}

// Use models package to avoid unused import error
var _ = models.FileEntry{}

// FileMetadata contains metadata about a file for change detection
type FileMetadata struct {
	Path         string            `json:"path"`
	Size         int64             `json:"size"`
	ModTime      time.Time         `json:"mod_time"`
	Hash         string            `json:"hash"`
	LastIndexed  time.Time         `json:"last_indexed"`
	Dependencies []string          `json:"dependencies"`
	Language     string            `json:"language"`
	ChunkCount   int               `json:"chunk_count"`
	Version      int               `json:"version"`
	Attributes   map[string]string `json:"attributes"`
}

// FileDependencies represents dependencies for a file
type FileDependencies struct {
	Includes      []string `json:"includes"`
	Imports       []string `json:"imports"`
	Requires      []string `json:"requires"`
	ReferencedBy  []string `json:"referenced_by"`
	LastAnalyzed  time.Time `json:"last_analyzed"`
}

// FileChangeEvent represents a file system change event
type FileChangeEvent struct {
	Type      ChangeType `json:"type"`
	Path      string     `json:"path"`
	OldPath   string     `json:"old_path,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
	Size      int64      `json:"size"`
	Hash      string     `json:"hash,omitempty"`
}

// ChangeType represents the type of file change
type ChangeType int

const (
	ChangeTypeCreate ChangeType = iota
	ChangeTypeModify
	ChangeTypeDelete
	ChangeTypeRename
	ChangeTypeMove
)

// ChangeSet represents a set of changes to be processed
type ChangeSet struct {
	FilesToIndex    []string `json:"files_to_index"`
	FilesToDelete    []string `json:"files_to_delete"`
	FilesToUpdate    []string `json:"files_to_update"`
	Dependencies     []string `json:"dependencies"`
	Timestamp       time.Time `json:"timestamp"`
}

// DefaultIncrementalOptions returns default options for incremental indexing
func DefaultIncrementalOptions() IncrementalOptions {
	return IncrementalOptions{
		EnableHashing:      true,
		EnableDependencies: true,
		HashCacheSize:       10000,
		ChangeBufferSize:    1000,
		DebounceDelay:       100 * time.Millisecond,
		MaxFileSizeHash:    10 * 1024 * 1024, // 10MB
		ExcludePatterns:     []string{".git/*", "node_modules/*", "*.tmp", "*.log"},
	}
}

// NewIncrementalIndexer creates a new incremental indexer
func NewIncrementalIndexer(indexPath string, options IncrementalOptions) *IncrementalIndexer {
	return &IncrementalIndexer{
		metadataPath:    filepath.Join(indexPath, "metadata.json"),
		fileMetadata:    make(map[string]*FileMetadata),
		dependencyGraph: make(map[string]*FileDependencies),
		options:         options,
		hashPool:        GetPoolManager().GetBufferPool(),
		changeEvents:    make(chan FileChangeEvent, options.ChangeBufferSize),
		eventBuffer:     make([]FileChangeEvent, 0, options.ChangeBufferSize),
		bufferSize:       options.ChangeBufferSize,
	}
}

// LoadMetadata loads existing metadata from disk
func (ii *IncrementalIndexer) LoadMetadata() error {
	ii.mu.Lock()
	defer ii.mu.Unlock()

	data, err := os.ReadFile(ii.metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing metadata, that's OK
		}
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata struct {
		Files       map[string]*FileMetadata    `json:"files"`
		Dependencies map[string]*FileDependencies `json:"dependencies"`
		Version     string                         `json:"version"`
		LastSaved   time.Time                      `json:"last_saved"`
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	ii.fileMetadata = metadata.Files
	ii.dependencyGraph = metadata.Dependencies

	return nil
}

// SaveMetadata saves current metadata to disk
func (ii *IncrementalIndexer) SaveMetadata() error {
	ii.mu.RLock()
	defer ii.mu.RUnlock()

	metadata := struct {
		Files       map[string]*FileMetadata    `json:"files"`
		Dependencies map[string]*FileDependencies `json:"dependencies"`
		Version     string                         `json:"version"`
		LastSaved   time.Time                      `json:"last_saved"`
	}{
		Files:       ii.fileMetadata,
		Dependencies: ii.dependencyGraph,
		Version:     "1.0",
		LastSaved:   time.Now(),
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to temporary file first
	tempPath := ii.metadataPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Atomic rename
	return os.Rename(tempPath, ii.metadataPath)
}

// DetectChanges scans for changes since last indexing
func (ii *IncrementalIndexer) DetectChanges(rootPath string) (*ChangeSet, error) {
	ii.mu.Lock()
	defer ii.mu.Unlock()

	changeSet := &ChangeSet{
		Timestamp: time.Now(),
	}

	// Get current files
	currentFiles, err := ii.scanDirectory(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Detect deletions
	for path := range ii.fileMetadata {
		if _, exists := currentFiles[path]; !exists {
			changeSet.FilesToDelete = append(changeSet.FilesToDelete, path)
			delete(ii.fileMetadata, path)
		}
	}

	// Detect modifications and additions
	for path, info := range currentFiles {
		if ii.shouldExcludeFile(path) {
			continue
		}

		existingMetadata, exists := ii.fileMetadata[path]
		currentMetadata := ii.createFileMetadata(path, info)

		needsUpdate := false

		if !exists {
			// New file
			changeSet.FilesToIndex = append(changeSet.FilesToIndex, path)
			needsUpdate = true
		} else {
			// Check for changes
			if ii.hasFileChanged(existingMetadata, currentMetadata) {
				changeSet.FilesToUpdate = append(changeSet.FilesToUpdate, path)
				needsUpdate = true
			}
		}

		if needsUpdate {
			ii.fileMetadata[path] = currentMetadata

			// Update dependencies if enabled
			if ii.options.EnableDependencies {
				deps := ii.analyzeDependencies(path)
				ii.dependencyGraph[path] = deps
			}
		}
	}

	// Add dependent files that need re-indexing
	if ii.options.EnableDependencies {
		dependentFiles := ii.getDependentFiles(changeSet.FilesToUpdate)
		for _, dep := range dependentFiles {
			// Avoid duplicates
			if !ii.containsString(changeSet.FilesToUpdate, dep) && !ii.containsString(changeSet.FilesToIndex, dep) {
				changeSet.FilesToUpdate = append(changeSet.FilesToUpdate, dep)
			}
		}
	}

	return changeSet, nil
}

// scanDirectory scans a directory and returns file information
func (ii *IncrementalIndexer) scanDirectory(rootPath string) (map[string]fs.FileInfo, error) {
	files := make(map[string]fs.FileInfo)

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if ii.shouldExcludeFile(path) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		files[path] = info
		return nil
	})

	return files, err
}

// shouldExcludeFile checks if a file should be excluded from scanning
func (ii *IncrementalIndexer) shouldExcludeFile(path string) bool {
	for _, pattern := range ii.options.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}
	return false
}

// createFileMetadata creates metadata for a file
func (ii *IncrementalIndexer) createFileMetadata(path string, info fs.FileInfo) *FileMetadata {
	metadata := &FileMetadata{
		Path:        path,
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		LastIndexed: time.Now(),
		Language:    ii.detectLanguage(path),
		Attributes:  make(map[string]string),
		Version:     1,
	}

	// Calculate hash if enabled and file size is reasonable
	if ii.options.EnableHashing && info.Size() <= ii.options.MaxFileSizeHash {
		if hash, err := ii.calculateFileHash(path); err == nil {
			metadata.Hash = hash
		}
	}

	return metadata
}

// calculateFileHash calculates SHA-256 hash of a file
func (ii *IncrementalIndexer) calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	buffer := ii.hashPool.GetBuffer(4096)
	defer ii.hashPool.PutBuffer(buffer)

	_, err = io.CopyBuffer(hasher, file, buffer)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// hasFileChanged checks if a file has changed since last indexing
func (ii *IncrementalIndexer) hasFileChanged(old, new *FileMetadata) bool {
	// Check modification time
	if !new.ModTime.Equal(old.ModTime) {
		return true
	}

	// Check size
	if new.Size != old.Size {
		return true
	}

	// Check hash if available
	if ii.options.EnableHashing && old.Hash != "" && new.Hash != "" {
		return old.Hash != new.Hash
	}

	return false
}

// detectLanguage detects the programming language of a file
func (ii *IncrementalIndexer) detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	// Simple language detection based on extension
	languages := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".py":   "Python",
		".java": "Java",
		".cpp":  "C++",
		".c":    "C",
		".h":    "C/C++ Header",
		".cs":   "C#",
		".php":  "PHP",
		".rb":   "Ruby",
		".rs":   "Rust",
		".kt":   "Kotlin",
		".swift": "Swift",
		".scala": "Scala",
		".sh":   "Shell",
		".sql":  "SQL",
		".html": "HTML",
		".css":  "CSS",
		".json": "JSON",
		".xml":  "XML",
		".yaml": "YAML",
		".yml":  "YAML",
		".md":   "Markdown",
	}

	if lang, exists := languages[ext]; exists {
		return lang
	}

	return "Unknown"
}

// analyzeDependencies analyzes dependencies of a file
func (ii *IncrementalIndexer) analyzeDependencies(path string) *FileDependencies {
	deps := &FileDependencies{
		Includes:     []string{},
		Imports:      []string{},
		Requires:     []string{},
		ReferencedBy: []string{},
		LastAnalyzed: time.Now(),
	}

	// This is a simplified implementation
	// In a real system, you'd parse the file and extract actual dependencies
	language := ii.detectLanguage(path)

	switch language {
	case "Go":
		deps.Imports = ii.extractGoImports(path)
	case "Python":
		deps.Imports = ii.extractPythonImports(path)
	case "JavaScript", "TypeScript":
		deps.Imports = ii.extractJSImports(path)
	case "C", "C++":
		deps.Includes = ii.extractCIncludes(path)
	}

	return deps
}

// extractGoImports extracts imports from Go files
func (ii *IncrementalIndexer) extractGoImports(path string) []string {
	// Simplified Go import extraction
	// In a real implementation, you'd use go/parser
	return []string{}
}

// extractPythonImports extracts imports from Python files
func (ii *IncrementalIndexer) extractPythonImports(path string) []string {
	// Simplified Python import extraction
	return []string{}
}

// extractJSImports extracts imports from JavaScript/TypeScript files
func (ii *IncrementalIndexer) extractJSImports(path string) []string {
	// Simplified JS/TS import extraction
	return []string{}
}

// extractCIncludes extracts includes from C/C++ files
func (ii *IncrementalIndexer) extractCIncludes(path string) []string {
	// Simplified C/C++ include extraction
	return []string{}
}

// getDependentFiles returns files that depend on the given files
func (ii *IncrementalIndexer) getDependentFiles(changedFiles []string) []string {
	dependents := make(map[string]bool)

	for _, changedFile := range changedFiles {
		for filePath, deps := range ii.dependencyGraph {
			// Check if this file depends on the changed file
			if ii.dependsOn(deps, changedFile) {
				dependents[filePath] = true
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(dependents))
	for dep := range dependents {
		result = append(result, dep)
	}

	return result
}

// dependsOn checks if dependencies contain the target file
func (ii *IncrementalIndexer) dependsOn(deps *FileDependencies, target string) bool {
	// Check imports
	for _, imp := range deps.Imports {
		if strings.Contains(imp, target) {
			return true
		}
	}

	// Check includes
	for _, inc := range deps.Includes {
		if strings.Contains(inc, target) {
			return true
		}
	}

	// Check requires
	for _, req := range deps.Requires {
		if strings.Contains(req, target) {
			return true
		}
	}

	return false
}

// containsString checks if a string slice contains a specific string
func (ii *IncrementalIndexer) containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// ProcessChangeSet processes a set of changes
func (ii *IncrementalIndexer) ProcessChangeSet(changeSet *ChangeSet) error {
	// Process deletions
	for _, path := range changeSet.FilesToDelete {
		ii.deleteFileMetadata(path)
	}

	// Process updates and additions
	// (In a real implementation, this would trigger re-indexing)
	for _, path := range append(changeSet.FilesToUpdate, changeSet.FilesToIndex...) {
		ii.updateFileMetadata(path)
	}

	// Save updated metadata
	return ii.SaveMetadata()
}

// deleteFileMetadata removes metadata for a deleted file
func (ii *IncrementalIndexer) deleteFileMetadata(path string) {
	delete(ii.fileMetadata, path)
	delete(ii.dependencyGraph, path)
}

// updateFileMetadata updates metadata for a file
func (ii *IncrementalIndexer) updateFileMetadata(path string) {
	if metadata, exists := ii.fileMetadata[path]; exists {
		metadata.LastIndexed = time.Now()
		metadata.Version++
	}
}

// GetFileMetadata returns metadata for a specific file
func (ii *IncrementalIndexer) GetFileMetadata(path string) (*FileMetadata, bool) {
	ii.mu.RLock()
	defer ii.mu.RUnlock()
	metadata, exists := ii.fileMetadata[path]
	return metadata, exists
}

// GetAllMetadata returns all file metadata
func (ii *IncrementalIndexer) GetAllMetadata() map[string]*FileMetadata {
	ii.mu.RLock()
	defer ii.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	result := make(map[string]*FileMetadata)
	for k, v := range ii.fileMetadata {
		result[k] = v
	}
	return result
}

// GetStats returns statistics about the incremental indexer
func (ii *IncrementalIndexer) GetStats() IncrementalStats {
	ii.mu.RLock()
	defer ii.mu.RUnlock()

	stats := IncrementalStats{
		TotalFiles:      len(ii.fileMetadata),
		Dependencies:    len(ii.dependencyGraph),
		LastSaved:       time.Now(),
		FilesByLanguage: make(map[string]int),
		Languages:       make([]string, 0),
	}

	// Count files by language
	langCount := make(map[string]int)
	for _, metadata := range ii.fileMetadata {
		langCount[metadata.Language]++
	}

	for lang, count := range langCount {
		stats.FilesByLanguage[lang] = count
		stats.Languages = append(stats.Languages, lang)
	}

	return stats
}

// IncrementalStats contains statistics about the incremental indexer
type IncrementalStats struct {
	TotalFiles      int               `json:"total_files"`
	Dependencies    int               `json:"dependencies"`
	LastSaved       time.Time         `json:"last_saved"`
	FilesByLanguage map[string]int    `json:"files_by_language"`
	Languages       []string          `json:"languages"`
}

// Cleanup removes stale metadata entries
func (ii *IncrementalIndexer) Cleanup(maxAge time.Duration) error {
	ii.mu.Lock()
	defer ii.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for path, metadata := range ii.fileMetadata {
		if metadata.LastIndexed.Before(cutoff) {
			delete(ii.fileMetadata, path)
			delete(ii.dependencyGraph, path)
			cleaned++
		}
	}

	if cleaned > 0 {
		return ii.SaveMetadata()
	}

	return nil
}