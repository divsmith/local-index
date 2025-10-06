package models

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CodeIndex represents the indexed representation of the codebase
type CodeIndex struct {
	ID             string                `json:"id"`
	Version        string                `json:"version"`
	RepositoryPath string                `json:"repository_path"`
	LastModified   time.Time             `json:"last_modified"`
	FileEntries    map[string]*FileEntry `json:"file_entries"`
	vectorStore    VectorStore           `json:"-"` // Not serialized
	mu             sync.RWMutex          `json:"-"` // For concurrent access
}

// VectorStore interface for vector database operations
type VectorStore interface {
	Insert(id string, vector []float64, metadata map[string]interface{}) error
	Search(queryVector []float64, limit int) ([]VectorSearchResult, error)
	Delete(id string) error
	Close() error
}

// VectorSearchResult represents a result from vector search
type VectorSearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NewCodeIndex creates a new CodeIndex instance
func NewCodeIndex(repositoryPath string, vectorStore VectorStore) *CodeIndex {
	return &CodeIndex{
		ID:             generateIndexID(repositoryPath),
		Version:        "1.0.0",
		RepositoryPath: repositoryPath,
		LastModified:   time.Now(),
		FileEntries:    make(map[string]*FileEntry),
		vectorStore:    vectorStore,
	}
}

// LoadCodeIndex loads an existing index from disk
func LoadCodeIndex(indexPath string, vectorStore VectorStore) (*CodeIndex, error) {
	file, err := os.Open(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index CodeIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	index.vectorStore = vectorStore

	return &index, nil
}

// Save saves the index to disk
func (ci *CodeIndex) Save(indexPath string) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	ci.LastModified = time.Now()

	data, err := json.MarshalIndent(ci, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Write to temporary file first
	tempPath := indexPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary index file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, indexPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename index file: %w", err)
	}

	return nil
}

// AddFileEntry adds a file entry to the index
func (ci *CodeIndex) AddFileEntry(entry *FileEntry) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	relativePath, err := filepath.Rel(ci.RepositoryPath, entry.FilePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	ci.FileEntries[relativePath] = entry

	// Index all chunks in vector store
	for _, chunk := range entry.Chunks {
		metadata := map[string]interface{}{
			"file_path":  relativePath,
			"start_line": chunk.StartLine,
			"end_line":   chunk.EndLine,
			"content":    chunk.Content,
			"context":    chunk.Context,
		}

		if err := ci.vectorStore.Insert(chunk.ID, chunk.Vector, metadata); err != nil {
			return fmt.Errorf("failed to insert chunk into vector store: %w", err)
		}
	}

	return nil
}

// RemoveFileEntry removes a file entry from the index
func (ci *CodeIndex) RemoveFileEntry(filePath string) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	relativePath, err := filepath.Rel(ci.RepositoryPath, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	entry, exists := ci.FileEntries[relativePath]
	if !exists {
		return nil // File not in index, nothing to remove
	}

	// Remove chunks from vector store
	for _, chunk := range entry.Chunks {
		if err := ci.vectorStore.Delete(chunk.ID); err != nil {
			return fmt.Errorf("failed to delete chunk from vector store: %w", err)
		}
	}

	delete(ci.FileEntries, relativePath)
	return nil
}

// GetFileEntry retrieves a file entry from the index
func (ci *CodeIndex) GetFileEntry(filePath string) (*FileEntry, error) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	relativePath, err := filepath.Rel(ci.RepositoryPath, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}

	entry, exists := ci.FileEntries[relativePath]
	if !exists {
		return nil, fmt.Errorf("file not found in index: %s", relativePath)
	}

	return entry, nil
}

// GetAllFiles returns all file entries in the index
func (ci *CodeIndex) GetAllFiles() []*FileEntry {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	files := make([]*FileEntry, 0, len(ci.FileEntries))
	for _, entry := range ci.FileEntries {
		files = append(files, entry)
	}

	return files
}

// GetStats returns statistics about the index
func (ci *CodeIndex) GetStats() IndexStats {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	stats := IndexStats{
		TotalFiles:     len(ci.FileEntries),
		TotalChunks:    0,
		LastModified:   ci.LastModified,
		RepositoryPath: ci.RepositoryPath,
		FileTypes:      make(map[string]int),
	}

	for _, entry := range ci.FileEntries {
		ext := filepath.Ext(entry.FilePath)
		if ext == "" {
			ext = "no_extension"
		}
		stats.FileTypes[ext]++
		stats.TotalChunks += len(entry.Chunks)
	}

	return stats
}

// Search performs a vector search on the index
func (ci *CodeIndex) Search(queryVector []float64, limit int) ([]VectorSearchResult, error) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	if ci.vectorStore == nil {
		return nil, fmt.Errorf("vector store not initialized")
	}

	return ci.vectorStore.Search(queryVector, limit)
}

// IsEmpty returns true if the index contains no files
func (ci *CodeIndex) IsEmpty() bool {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	return len(ci.FileEntries) == 0
}

// ShouldReindex determines if the index needs to be rebuilt based on file changes
func (ci *CodeIndex) ShouldReindex() (bool, error) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	// Check if any files have been modified since last index
	for _, entry := range ci.FileEntries {
		fileInfo, err := os.Stat(entry.FilePath)
		if err != nil {
			if os.IsNotExist(err) {
				// File was deleted, should reindex
				return true, nil
			}
			return false, fmt.Errorf("failed to stat file %s: %w", entry.FilePath, err)
		}

		// Check if file was modified
		if fileInfo.ModTime().After(entry.LastModified) {
			return true, nil
		}

		// Check content hash
		currentHash, err := calculateFileHash(entry.FilePath)
		if err != nil {
			return false, fmt.Errorf("failed to calculate file hash: %w", err)
		}

		if currentHash != entry.ContentHash {
			return true, nil
		}
	}

	return false, nil
}

// Close closes the index and releases resources
func (ci *CodeIndex) Close() error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	if ci.vectorStore != nil {
		return ci.vectorStore.Close()
	}

	return nil
}

// IndexStats contains statistics about the code index
type IndexStats struct {
	TotalFiles     int            `json:"total_files"`
	TotalChunks    int            `json:"total_chunks"`
	LastModified   time.Time      `json:"last_modified"`
	RepositoryPath string         `json:"repository_path"`
	FileTypes      map[string]int `json:"file_types"`
}

// generateIndexID generates a unique ID for the index
func generateIndexID(repositoryPath string) string {
	hash := sha256.Sum256([]byte(repositoryPath + time.Now().String()))
	return fmt.Sprintf("idx_%x", hash[:16])
}

// calculateFileHash calculates SHA256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
