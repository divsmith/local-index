package models

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileEntry represents a single file in the indexed codebase
type FileEntry struct {
	FilePath     string      `json:"file_path"`
	LastModified time.Time   `json:"last_modified"`
	ContentHash  string      `json:"content_hash"`
	Chunks       []CodeChunk `json:"chunks"`
	Size         int64       `json:"size"`
	Language     string      `json:"language"`
}

// NewFileEntry creates a new FileEntry from a file path
func NewFileEntry(filePath string) (*FileEntry, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// Calculate content hash
	contentHash, err := calculateFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate content hash: %w", err)
	}

	// Detect language
	language := detectLanguage(filePath)

	return &FileEntry{
		FilePath:     filePath,
		LastModified: fileInfo.ModTime(),
		ContentHash:  contentHash,
		Chunks:       make([]CodeChunk, 0),
		Size:         fileInfo.Size(),
		Language:     language,
	}, nil
}

// AddChunk adds a code chunk to the file entry
func (fe *FileEntry) AddChunk(chunk CodeChunk) {
	fe.Chunks = append(fe.Chunks, chunk)
}

// GetContent reads and returns the file content
func (fe *FileEntry) GetContent() (string, error) {
	file, err := os.Open(fe.FilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file content: %w", err)
	}

	return string(content), nil
}

// GetLineCount returns the number of lines in the file
func (fe *FileEntry) GetLineCount() (int, error) {
	content, err := fe.GetContent()
	if err != nil {
		return 0, err
	}

	if content == "" {
		return 0, nil
	}

	return len(strings.Split(content, "\n")), nil
}

// GetChunkForLine returns the chunk that contains the specified line
func (fe *FileEntry) GetChunkForLine(lineNumber int) (*CodeChunk, error) {
	for _, chunk := range fe.Chunks {
		if lineNumber >= chunk.StartLine && lineNumber <= chunk.EndLine {
			return &chunk, nil
		}
	}

	return nil, fmt.Errorf("no chunk found for line %d", lineNumber)
}

// GetChunksInRange returns all chunks that overlap with the specified line range
func (fe *FileEntry) GetChunksInRange(startLine, endLine int) []CodeChunk {
	var chunks []CodeChunk

	for _, chunk := range fe.Chunks {
		// Check if chunk overlaps with the range
		if chunk.StartLine <= endLine && chunk.EndLine >= startLine {
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// Validate validates the file entry
func (fe *FileEntry) Validate() error {
	if fe.FilePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	if _, err := os.Stat(fe.FilePath); err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}

	if fe.ContentHash == "" {
		return fmt.Errorf("content hash cannot be empty")
	}

	// Verify content hash matches current file content
	currentHash, err := calculateFileHash(fe.FilePath)
	if err != nil {
		return fmt.Errorf("failed to verify content hash: %w", err)
	}

	if currentHash != fe.ContentHash {
		return fmt.Errorf("content hash mismatch: file has been modified")
	}

	// Validate chunks
	for i, chunk := range fe.Chunks {
		if err := chunk.Validate(); err != nil {
			return fmt.Errorf("chunk %d validation failed: %w", i, err)
		}

		// Check if chunk line numbers are within file bounds
		lineCount, err := fe.GetLineCount()
		if err != nil {
			return fmt.Errorf("failed to get line count for chunk validation: %w", err)
		}

		if chunk.StartLine < 1 || chunk.EndLine > lineCount {
			return fmt.Errorf("chunk %d line numbers out of bounds: %d-%d (file has %d lines)",
				i, chunk.StartLine, chunk.EndLine, lineCount)
		}
	}

	return nil
}

// GetStats returns statistics about the file entry
func (fe *FileEntry) GetStats() FileEntryStats {
	stats := FileEntryStats{
		FilePath:     fe.FilePath,
		LastModified: fe.LastModified,
		Size:         fe.Size,
		Language:     fe.Language,
		ChunkCount:   len(fe.Chunks),
	}

	// Calculate total lines covered by chunks
	totalChunkLines := 0
	for _, chunk := range fe.Chunks {
		totalChunkLines += (chunk.EndLine - chunk.StartLine + 1)
	}
	stats.TotalChunkLines = totalChunkLines

	// Calculate coverage if possible
	if lineCount, err := fe.GetLineCount(); err == nil {
		stats.LineCount = lineCount
		if lineCount > 0 {
			stats.CoveragePercent = float64(totalChunkLines) / float64(lineCount) * 100
		}
	}

	return stats
}

// ShouldIndex determines if the file should be indexed based on various criteria
func (fe *FileEntry) ShouldIndex() bool {
	// Skip binary files
	if isBinaryFile(fe.FilePath) {
		return false
	}

	// Skip very large files (>1MB)
	if fe.Size > 1024*1024 {
		return false
	}

	// Skip hidden files and directories
	if strings.HasPrefix(filepath.Base(fe.FilePath), ".") {
		return false
	}

	// Check if file type is supported
	return isSupportedFileType(fe.FilePath)
}

// ToJSON converts the file entry to JSON
func (fe *FileEntry) ToJSON() ([]byte, error) {
	return json.MarshalIndent(fe, "", "  ")
}

// FromJSON creates a FileEntry from JSON data
func (fe *FileEntry) FromJSON(data []byte) error {
	var entry FileEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal file entry: %w", err)
	}

	*fe = entry
	return nil
}

// FileEntryStats contains statistics about a file entry
type FileEntryStats struct {
	FilePath        string    `json:"file_path"`
	LastModified    time.Time `json:"last_modified"`
	Size            int64     `json:"size"`
	Language        string    `json:"language"`
	LineCount       int       `json:"line_count"`
	ChunkCount      int       `json:"chunk_count"`
	TotalChunkLines int       `json:"total_chunk_lines"`
	CoveragePercent float64   `json:"coverage_percent"`
}

// detectLanguage detects the programming language based on file extension
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":         "Go",
		".js":         "JavaScript",
		".ts":         "TypeScript",
		".jsx":        "JavaScript",
		".tsx":        "TypeScript",
		".py":         "Python",
		".java":       "Java",
		".c":          "C",
		".cpp":        "C++",
		".cc":         "C++",
		".cxx":        "C++",
		".h":          "C/C++ Header",
		".hpp":        "C++ Header",
		".cs":         "C#",
		".php":        "PHP",
		".rb":         "Ruby",
		".swift":      "Swift",
		".kt":         "Kotlin",
		".rs":         "Rust",
		".scala":      "Scala",
		".sh":         "Shell",
		".bash":       "Shell",
		".zsh":        "Shell",
		".fish":       "Shell",
		".ps1":        "PowerShell",
		".sql":        "SQL",
		".html":       "HTML",
		".css":        "CSS",
		".scss":       "SCSS",
		".sass":       "Sass",
		".less":       "Less",
		".xml":        "XML",
		".yaml":       "YAML",
		".yml":        "YAML",
		".json":       "JSON",
		".toml":       "TOML",
		".md":         "Markdown",
		".txt":        "Text",
		".dockerfile": "Dockerfile",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	// Check for special filenames
	filename := strings.ToLower(filepath.Base(filePath))
	switch filename {
	case "dockerfile":
		return "Dockerfile"
	case "makefile":
		return "Makefile"
	case "gemfile":
		return "Ruby"
	case "requirements.txt":
		return "Python"
	case "package.json":
		return "JavaScript/Node.js"
	case "go.mod":
		return "Go"
	case "go.sum":
		return "Go"
	}

	return "Unknown"
}

// isBinaryFile checks if a file is likely to be binary
func isBinaryFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false // Assume not binary if we can't check
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes in the first 512 bytes
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}

	return false
}

// isSupportedFileType checks if the file type is supported for indexing
func isSupportedFileType(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	supportedExtensions := map[string]bool{
		".go":         true,
		".js":         true,
		".ts":         true,
		".jsx":        true,
		".tsx":        true,
		".py":         true,
		".java":       true,
		".c":          true,
		".cpp":        true,
		".cc":         true,
		".cxx":        true,
		".h":          true,
		".hpp":        true,
		".cs":         true,
		".php":        true,
		".rb":         true,
		".swift":      true,
		".kt":         true,
		".rs":         true,
		".scala":      true,
		".sh":         true,
		".bash":       true,
		".zsh":        true,
		".fish":       true,
		".ps1":        true,
		".sql":        true,
		".html":       true,
		".css":        true,
		".scss":       true,
		".sass":       true,
		".less":       true,
		".xml":        true,
		".yaml":       true,
		".yml":        true,
		".json":       true,
		".toml":       true,
		".md":         true,
		".txt":        true,
		".dockerfile": true,
	}

	if supportedExtensions[ext] {
		return true
	}

	// Check special filenames
	filename := strings.ToLower(filepath.Base(filePath))
	specialFiles := map[string]bool{
		"dockerfile": true,
		"makefile":   true,
		"gemfile":    true,
		"readme":     true,
		"license":    true,
	}

	return specialFiles[filename]
}
