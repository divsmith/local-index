package services

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"code-search/src/models"
)

// IndexingService handles codebase indexing operations
type IndexingService struct {
	fileScanner  FileScanner
	codeParser   CodeParser
	vectorStore  models.VectorStore
	logger       Logger
	indexOptions IndexingOptions
	mu           sync.RWMutex
}

// IndexingOptions contains options for the indexing process
type IndexingOptions struct {
	IncludeHidden     bool          `json:"include_hidden"`
	FileTypes         []string      `json:"file_types"`
	ExcludePatterns   []string      `json:"exclude_patterns"`
	MaxFileSize       int64         `json:"max_file_size"`
	ChunkSize         int           `json:"chunk_size"`
	ChunkOverlap      int           `json:"chunk_overlap"`
	MaxConcurrency    int           `json:"max_concurrency"`
	Timeout           time.Duration `json:"timeout"`
	EnableIncremental bool          `json:"enable_incremental"`
}

// DefaultIndexingOptions returns default indexing options
func DefaultIndexingOptions() IndexingOptions {
	return IndexingOptions{
		IncludeHidden:     false,
		FileTypes:         []string{"*"}, // All supported types
		ExcludePatterns:   []string{"*.tmp", "*.log", "node_modules/*", ".git/*"},
		MaxFileSize:       1024 * 1024, // 1MB
		ChunkSize:         500,         // 500 characters per chunk
		ChunkOverlap:      50,          // 50 characters overlap
		MaxConcurrency:    4,
		Timeout:           30 * time.Minute,
		EnableIncremental: true,
	}
}

// Logger interface for logging operations
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// FileScanner interface for scanning files
type FileScanner interface {
	ScanFiles(rootPath string, options IndexingOptions) ([]string, error)
	GetFileStats(filePath string) (FileStats, error)
}

// CodeParser interface for parsing code
type CodeParser interface {
	ParseFile(filePath string) ([]models.CodeChunk, error)
	GetEmbedding(text string) ([]float64, error)
	GetSupportedFileTypes() []string
}

// FileStats contains file statistics
type FileStats struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time"`
	IsDir        bool      `json:"is_dir"`
	Language     string    `json:"language"`
}

// ProgressCallback is called during indexing to report progress
type ProgressCallback func(current, total int, filePath string)

// IndexingResult contains the result of an indexing operation
type IndexingResult struct {
	Success        bool          `json:"success"`
	FilesIndexed   int           `json:"files_indexed"`
	FilesSkipped   int           `json:"files_skipped"`
	ChunksCreated  int           `json:"chunks_created"`
	Errors         []string      `json:"errors"`
	Duration       time.Duration `json:"duration"`
	IndexPath      string        `json:"index_path"`
	RepositoryPath string        `json:"repository_path"`
}

// NewIndexingService creates a new IndexingService
func NewIndexingService(
	fileScanner FileScanner,
	codeParser CodeParser,
	vectorStore models.VectorStore,
	logger Logger,
	options IndexingOptions,
) *IndexingService {
	return &IndexingService{
		fileScanner:  fileScanner,
		codeParser:   codeParser,
		vectorStore:  vectorStore,
		logger:       logger,
		indexOptions: options,
	}
}

// IndexRepository indexes an entire repository
func (is *IndexingService) IndexRepository(
	repositoryPath string,
	indexPath string,
	forceReindex bool,
	progressCallback ProgressCallback,
) (*IndexingResult, error) {
	start := time.Now()

	result := &IndexingResult{
		Success:        false,
		FilesIndexed:   0,
		FilesSkipped:   0,
		ChunksCreated:  0,
		Errors:         make([]string, 0),
		RepositoryPath: repositoryPath,
		IndexPath:      indexPath,
	}

	is.logger.Info("Starting indexing for repository: %s", repositoryPath)

	// Validate repository path
	if err := is.validateRepositoryPath(repositoryPath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid repository path: %v", err))
		return result, err
	}

	// Load existing index if not forcing reindex
	var existingIndex *models.CodeIndex
	if !forceReindex && is.indexOptions.EnableIncremental {
		var err error
		existingIndex, err = is.loadExistingIndex(indexPath)
		if err == nil {
			is.logger.Info("Loaded existing index with %d files", len(existingIndex.GetAllFiles()))
		} else {
			is.logger.Debug("No existing index found, creating new one: %v", err)
		}
	}

	// Create or update index
	codeIndex, err := is.createOrUpdateIndex(repositoryPath, existingIndex)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create index: %v", err))
		return result, err
	}

	// Scan files
	files, err := is.fileScanner.ScanFiles(repositoryPath, is.indexOptions)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to scan files: %v", err))
		return result, err
	}

	is.logger.Info("Found %d files to process", len(files))

	// Process files
	totalFiles := len(files)
	processedFiles := 0

	// Create channel for concurrent processing
	fileChan := make(chan string, is.indexOptions.MaxConcurrency)
	resultChan := make(chan FileProcessingResult, is.indexOptions.MaxConcurrency)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < is.indexOptions.MaxConcurrency; i++ {
		wg.Add(1)
		go is.processFileWorker(fileChan, resultChan, codeIndex, &wg)
	}

	// Send files to workers
	go func() {
		for _, file := range files {
			fileChan <- file
		}
		close(fileChan)
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results
	for fileResult := range resultChan {
		processedFiles++

		if fileResult.Error != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to process %s: %v", fileResult.FilePath, fileResult.Error))
			result.FilesSkipped++
		} else if fileResult.Skipped {
			result.FilesSkipped++
			is.logger.Debug("Skipped file: %s (%s)", fileResult.FilePath, fileResult.SkipReason)
		} else {
			result.FilesIndexed++
			result.ChunksCreated += fileResult.ChunkCount
			is.logger.Debug("Processed file: %s (%d chunks)", fileResult.FilePath, fileResult.ChunkCount)
		}

		// Report progress
		if progressCallback != nil {
			progressCallback(processedFiles, totalFiles, fileResult.FilePath)
		}
	}

	// Save the index
	if err := codeIndex.Save(indexPath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to save index: %v", err))
		return result, err
	}

	result.Duration = time.Since(start)
	result.Success = true

	is.logger.Info("Indexing completed successfully: %d files indexed, %d chunks created in %v",
		result.FilesIndexed, result.ChunksCreated, result.Duration)

	return result, nil
}

// FileProcessingResult contains the result of processing a single file
type FileProcessingResult struct {
	FilePath   string
	Error      error
	Skipped    bool
	SkipReason string
	ChunkCount int
}

// processFileWorker processes files from the file channel
func (is *IndexingService) processFileWorker(
	fileChan <-chan string,
	resultChan chan<- FileProcessingResult,
	codeIndex *models.CodeIndex,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for filePath := range fileChan {
		result := is.processFile(filePath, codeIndex)
		resultChan <- result
	}
}

// processFile processes a single file
func (is *IndexingService) processFile(filePath string, codeIndex *models.CodeIndex) FileProcessingResult {
	result := FileProcessingResult{
		FilePath: filePath,
	}

	// Check if file should be skipped
	shouldSkip, skipReason := is.shouldSkipFile(filePath, codeIndex)
	if shouldSkip {
		result.Skipped = true
		result.SkipReason = skipReason
		return result
	}

	// Parse file into chunks
	chunks, err := is.codeParser.ParseFile(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse file: %w", err)
		return result
	}

	if len(chunks) == 0 {
		result.Skipped = true
		result.SkipReason = "no chunks created"
		return result
	}

	// Generate embeddings for chunks
	for i := range chunks {
		embedding, err := is.codeParser.GetEmbedding(chunks[i].Content)
		if err != nil {
			result.Error = fmt.Errorf("failed to generate embedding for chunk: %w", err)
			return result
		}

		if err := chunks[i].SetVector(embedding); err != nil {
			result.Error = fmt.Errorf("failed to set vector: %w", err)
			return result
		}
	}

	// Create file entry
	fileEntry, err := models.NewFileEntry(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to create file entry: %w", err)
		return result
	}

	// Add chunks to file entry
	for _, chunk := range chunks {
		fileEntry.AddChunk(chunk)
	}

	// Add file entry to index
	if err := codeIndex.AddFileEntry(fileEntry); err != nil {
		result.Error = fmt.Errorf("failed to add file entry to index: %w", err)
		return result
	}

	result.ChunkCount = len(chunks)
	return result
}

// shouldSkipFile determines if a file should be skipped during indexing
func (is *IndexingService) shouldSkipFile(filePath string, codeIndex *models.CodeIndex) (bool, string) {
	// Check if file already exists in index and hasn't changed
	if codeIndex != nil {
		if existingEntry, err := codeIndex.GetFileEntry(filePath); err == nil {
			if fileInfo, err := os.Stat(filePath); err == nil {
				if !fileInfo.ModTime().After(existingEntry.LastModified) {
					return true, "file unchanged since last indexing"
				}
			}
		}
	}

	// Check file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return true, fmt.Sprintf("cannot stat file: %v", err)
	}

	if fileInfo.Size() > is.indexOptions.MaxFileSize {
		return true, fmt.Sprintf("file too large: %d bytes", fileInfo.Size())
	}

	// Check if file type is supported
	if !is.isFileTypeSupported(filePath) {
		return true, "unsupported file type"
	}

	return false, ""
}

// isFileTypeSupported checks if the file type is supported for indexing
func (is *IndexingService) isFileTypeSupported(filePath string) bool {
	supportedTypes := is.codeParser.GetSupportedFileTypes()

	if len(is.indexOptions.FileTypes) == 1 && is.indexOptions.FileTypes[0] == "*" {
		// All supported types
		for _, supportedType := range supportedTypes {
			if strings.HasSuffix(strings.ToLower(filePath), strings.ToLower(supportedType)) {
				return true
			}
		}
		return false
	}

	// Specific file types
	for _, fileType := range is.indexOptions.FileTypes {
		if strings.HasSuffix(strings.ToLower(filePath), strings.ToLower(fileType)) {
			return true
		}
	}

	return false
}

// validateRepositoryPath validates the repository path
func (is *IndexingService) validateRepositoryPath(repositoryPath string) error {
	if repositoryPath == "" {
		return fmt.Errorf("repository path cannot be empty")
	}

	// Check if path exists
	fileInfo, err := os.Stat(repositoryPath)
	if err != nil {
		return fmt.Errorf("repository path does not exist: %w", err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("repository path must be a directory")
	}

	// Check if path is readable
	if _, err := os.Open(repositoryPath); err != nil {
		return fmt.Errorf("repository path is not readable: %w", err)
	}

	return nil
}

// loadExistingIndex loads an existing index from disk
func (is *IndexingService) loadExistingIndex(indexPath string) (*models.CodeIndex, error) {
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("index file does not exist")
	}

	return models.LoadCodeIndex(indexPath, is.vectorStore)
}

// createOrUpdateIndex creates a new index or updates an existing one
func (is *IndexingService) createOrUpdateIndex(repositoryPath string, existingIndex *models.CodeIndex) (*models.CodeIndex, error) {
	if existingIndex != nil && existingIndex.RepositoryPath == repositoryPath {
		// Check if reindexing is needed
		shouldReindex, err := existingIndex.ShouldReindex()
		if err != nil {
			return nil, fmt.Errorf("failed to check if reindexing is needed: %w", err)
		}

		if !shouldReindex {
			is.logger.Info("Index is up to date, no reindexing needed")
			return existingIndex, nil
		}

		is.logger.Info("Updating existing index")
		return existingIndex, nil
	}

	is.logger.Info("Creating new index")
	return models.NewCodeIndex(repositoryPath, is.vectorStore), nil
}

// GetIndexingStatus returns the current indexing status
func (is *IndexingService) GetIndexingStatus(indexPath string) (*IndexingStatus, error) {
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return &IndexingStatus{
			Exists:    false,
			Message:   "No index found",
			CreatedAt: time.Time{},
		}, nil
	}

	// Load index to get status
	index, err := is.loadExistingIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load index for status: %w", err)
	}

	stats := index.GetStats()
	shouldReindex, _ := index.ShouldReindex()

	return &IndexingStatus{
		Exists:         true,
		RepositoryPath: stats.RepositoryPath,
		FileCount:      stats.TotalFiles,
		ChunkCount:     stats.TotalChunks,
		LastModified:   stats.LastModified,
		CreatedAt:      stats.LastModified, // Approximation
		ShouldReindex:  shouldReindex,
		Message:        "Index found and valid",
	}, nil
}

// DeleteIndex removes an index from disk
func (is *IndexingService) DeleteIndex(indexPath string) error {
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return nil // Index doesn't exist, nothing to delete
	}

	if err := os.Remove(indexPath); err != nil {
		return fmt.Errorf("failed to delete index file: %w", err)
	}

	is.logger.Info("Index deleted: %s", indexPath)
	return nil
}

// IndexingStatus contains information about an index
type IndexingStatus struct {
	Exists         bool      `json:"exists"`
	RepositoryPath string    `json:"repository_path"`
	FileCount      int       `json:"file_count"`
	ChunkCount     int       `json:"chunk_count"`
	LastModified   time.Time `json:"last_modified"`
	CreatedAt      time.Time `json:"created_at"`
	ShouldReindex  bool      `json:"should_reindex"`
	Message        string    `json:"message"`
}

// DefaultLogger provides a simple console logger
type DefaultLogger struct{}

// Info logs an info message
func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg, args...)
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[DEBUG] "+msg, args...)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[WARN] "+msg, args...)
}

// SilentLogger is a logger that discards all log messages
type SilentLogger struct{}

// Info discards info messages
func (l *SilentLogger) Info(msg string, args ...interface{}) {
	// Silent - no output
}

// Error discards error messages
func (l *SilentLogger) Error(msg string, args ...interface{}) {
	// Silent - no output
}

// Debug discards debug messages
func (l *SilentLogger) Debug(msg string, args ...interface{}) {
	// Silent - no output
}

// Warn discards warning messages
func (l *SilentLogger) Warn(msg string, args ...interface{}) {
	// Silent - no output
}
