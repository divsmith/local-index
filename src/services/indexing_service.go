package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code-search/src/models"
	"code-search/src/lib"
)

// IndexingService handles codebase indexing operations
type IndexingService struct {
	fileScanner    FileScanner
	codeParser     CodeParser
	vectorStore    models.VectorStore
	logger         Logger
	indexOptions   models.IndexingOptions
	workerPool     *lib.WorkerPool
	storageManager *lib.StorageManager
	projectDetector *lib.ProjectDetector
	mu             sync.RWMutex
}

// DefaultIndexingOptions returns default indexing options
func DefaultIndexingOptions() models.IndexingOptions {
	return models.IndexingOptions{
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
	ScanFiles(rootPath string, options models.IndexingOptions) ([]string, error)
	GetFileStats(filePath string) (models.FileStats, error)
}

// CodeParser interface for parsing code
type CodeParser interface {
	ParseFile(filePath string) ([]models.CodeChunk, error)
	GetEmbedding(text string) ([]float64, error)
	GetSupportedFileTypes() []string
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
	options models.IndexingOptions,
) *IndexingService {
	// Create dynamic worker pool with optimal configuration
	poolOptions := lib.DefaultPoolOptions()

	// Calculate optimal worker counts
	minWorkers := 1
	if options.MaxConcurrency > 2 {
		minWorkers = options.MaxConcurrency / 2
	}

	maxWorkers := 2
	if options.MaxConcurrency > 1 {
		maxWorkers = options.MaxConcurrency * 2
	}

	poolOptions.MinWorkers = minWorkers
	poolOptions.MaxWorkers = maxWorkers
	poolOptions.QueueSize = 1000
	poolOptions.EnableMetrics = true

	workerPool := lib.NewWorkerPool(poolOptions)

	// Initialize storage manager and project detector
	storageManager := lib.NewStorageManager()
	projectDetector := lib.NewProjectDetector()

	// Ensure centralized storage directories exist
	if err := storageManager.EnsureDirectories(); err != nil {
		logger.Error("Failed to create storage directories: %v", err)
	}

	return &IndexingService{
		fileScanner:     fileScanner,
		codeParser:      codeParser,
		vectorStore:     vectorStore,
		logger:          logger,
		indexOptions:    options,
		workerPool:      workerPool,
		storageManager:  storageManager,
		projectDetector: projectDetector,
	}
}

// IndexProject indexes a project using centralized storage
func (is *IndexingService) IndexProject(
	startPath string,
	forceReindex bool,
	progressCallback ProgressCallback,
) (*IndexingResult, error) {
	// Detect project root
	projectRoot, err := is.projectDetector.DetectProjectRoot(startPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project root: %w", err)
	}

	// Get centralized index path
	indexPath := is.storageManager.GetProjectIndexPath(projectRoot)

	is.logger.Info("Indexing project at root: %s", projectRoot)
	is.logger.Info("Using centralized storage: %s", indexPath)

	return is.IndexRepository(projectRoot, indexPath, forceReindex, progressCallback)
}

// GetProjectIndexStatus returns the index status for a project using centralized storage
func (is *IndexingService) GetProjectIndexStatus(startPath string) (*IndexingStatus, error) {
	// Detect project root
	projectRoot, err := is.projectDetector.DetectProjectRoot(startPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project root: %w", err)
	}

	// Get centralized index path
	indexPath := is.storageManager.GetProjectIndexPath(projectRoot)

	return is.GetIndexingStatus(indexPath)
}

// DeleteProjectIndex removes a project's index from centralized storage
func (is *IndexingService) DeleteProjectIndex(startPath string) error {
	// Detect project root
	projectRoot, err := is.projectDetector.DetectProjectRoot(startPath)
	if err != nil {
		return fmt.Errorf("failed to detect project root: %w", err)
	}

	// Remove project from centralized storage
	return is.storageManager.RemoveProject(projectRoot)
}

// ListIndexedProjects returns a list of all indexed projects
func (is *IndexingService) ListIndexedProjects() ([]string, error) {
	return is.storageManager.ListProjects()
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

	// Process files using BatchProcessor for memory efficiency
	totalFiles := len(files)
	processedFiles := 0

	// Process files in batches for memory efficiency
	batchSize := is.indexOptions.ChunkSize // Use chunk size as batch size for files
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	// Convert string file paths to FileInfo for BatchProcessor
	var fileInfos []lib.FileInfo
	for _, filePath := range files {
		if info, err := os.Stat(filePath); err == nil {
			relPath, _ := filepath.Rel(repositoryPath, filePath)
			fileInfos = append(fileInfos, lib.FileInfo{
				Path:         filePath,
				Size:         info.Size(),
				ModTime:      info.ModTime(),
				IsDir:        false,
				Extension:    filepath.Ext(filePath),
				RelativePath: relPath,
			})
		}
	}

	batchProcessor := lib.NewBatchProcessor(batchSize, func(fileBatch []lib.FileInfo) error {
		// Convert FileInfo back to strings for processing
		var filePaths []string
		for _, fileInfo := range fileBatch {
			filePaths = append(filePaths, fileInfo.Path)
		}

		// Process each batch with controlled concurrency
		return is.processFileBatch(filePaths, codeIndex, result, &processedFiles, totalFiles, progressCallback)
	})

	// Process all files in batches
	if err := batchProcessor.ProcessFiles(fileInfos); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Batch processing failed: %v", err))
		return result, err
	}

	// Save the index
	if err := codeIndex.Save(indexPath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to save index: %v", err))
		return result, err
	}

	// Save metadata for enhanced search compatibility
	if err := is.saveIndexMetadata(indexPath, codeIndex, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to save metadata: %v", err))
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

// processFileBatch processes a batch of files using the dynamic worker pool
func (is *IndexingService) processFileBatch(
	fileBatch []string,
	codeIndex *models.CodeIndex,
	result *IndexingResult,
	processedFiles *int,
	totalFiles int,
	progressCallback ProgressCallback,
) error {
	// Create tasks for the worker pool
	tasks := make([]func() (interface{}, error), len(fileBatch))

	for i, filePath := range fileBatch {
		// Capture the file path in a closure
		filePath := filePath
		tasks[i] = func() (interface{}, error) {
			return is.processFileWithPool(filePath, codeIndex)
		}
	}

	// Submit all tasks to worker pool
	futures := is.workerPool.SubmitBatch(tasks)

	// Process results as they complete
	for i, future := range futures {
		fileResult, err := future.Get()
		*processedFiles++

		var processingResult FileProcessingResult
		if err != nil {
			processingResult = FileProcessingResult{
				FilePath: fileBatch[i],
				Error:    fmt.Errorf("worker pool error: %w", err),
			}
		} else {
			processingResult = fileResult.(FileProcessingResult)
		}

		// Update result statistics
		if processingResult.Error != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to process %s: %v", processingResult.FilePath, processingResult.Error))
			result.FilesSkipped++
		} else if processingResult.Skipped {
			result.FilesSkipped++
			is.logger.Debug("Skipped file: %s (%s)", processingResult.FilePath, processingResult.SkipReason)
		} else {
			result.FilesIndexed++
			result.ChunksCreated += processingResult.ChunkCount
			is.logger.Debug("Processed file: %s (%d chunks)", processingResult.FilePath, processingResult.ChunkCount)
		}

		// Report progress
		if progressCallback != nil {
			progressCallback(*processedFiles, totalFiles, processingResult.FilePath)
		}
	}

	return nil
}

// processFileWithPool processes a single file for use with the worker pool
func (is *IndexingService) processFileWithPool(filePath string, codeIndex *models.CodeIndex) (interface{}, error) {
	return is.processFile(filePath, codeIndex), nil
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

// IndexDirectory indexes a specific directory with validation
func (is *IndexingService) IndexDirectory(
	directoryPath string,
	forceReindex bool,
	progressCallback ProgressCallback,
) (*IndexingResult, error) {
	// Validate directory configuration
	validator := lib.NewDirectoryValidator()
	fileUtils := lib.NewFileUtilities()

	// Resolve path
	resolvedPath, err := fileUtils.ResolvePath(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Validate directory
	config, err := validator.ValidateDirectory(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("directory validation failed: %w", err)
	}

	// Create index location
	indexLocation := fileUtils.CreateIndexLocation(config.Path)

	// Note: File locking disabled temporarily to resolve indexing issues
	// if fileUtils.IsLocked(config.Path) {
	// 	return nil, fmt.Errorf("directory '%s' is currently being indexed by another process", config.Path)
	// }

	// Note: File locking disabled temporarily to resolve indexing issues
	// lockFile, err := fileUtils.AcquireLock(config.Path)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to acquire lock for directory '%s': %w", config.Path, err)
	// }
	// defer fileUtils.ReleaseLock(lockFile)

	// Ensure index directory exists
	if err := fileUtils.EnsureDirectory(indexLocation.IndexDir); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	is.logger.Info("Starting directory indexing for: %s", config.Path)

	// Update directory metadata
	config.Metadata.MarkIndexed()

	// Save directory metadata
	metadataBytes, err := config.Metadata.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize directory metadata: %w", err)
	}

	if err := os.WriteFile(indexLocation.MetadataFile, metadataBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to save directory metadata: %w", err)
	}

	// Index the directory using existing repository indexing logic
	result, err := is.IndexRepository(config.Path, indexLocation.DataFile, forceReindex, progressCallback)
	if err != nil {
		return result, err
	}

	// Update result with directory-specific information
	result.RepositoryPath = config.Path
	result.IndexPath = indexLocation.IndexDir

	is.logger.Info("Directory indexing completed successfully for: %s", config.Path)
	return result, nil
}

// ValidateDirectoryForIndexing validates a directory for indexing
func (is *IndexingService) ValidateDirectoryForIndexing(directoryPath string) (*models.DirectoryConfig, error) {
	validator := lib.NewDirectoryValidator()
	fileUtils := lib.NewFileUtilities()

	// Resolve path
	resolvedPath, err := fileUtils.ResolvePath(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Validate directory
	config, err := validator.ValidateDirectory(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("directory validation failed: %w", err)
	}

	return config, nil
}

// GetDirectoryIndexStatus returns the index status for a specific directory
func (is *IndexingService) GetDirectoryIndexStatus(directoryPath string) (*DirectoryIndexStatus, error) {
	fileUtils := lib.NewFileUtilities()

	// Resolve path
	resolvedPath, err := fileUtils.ResolvePath(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Create index location
	indexLocation := fileUtils.CreateIndexLocation(resolvedPath)

	// Check if index directory exists
	if !fileUtils.DirectoryExists(indexLocation.IndexDir) {
		return &DirectoryIndexStatus{
			Exists:    false,
			Directory: resolvedPath,
			Message:   "No index found in directory",
		}, nil
	}

	// Check if locked
	if fileUtils.IsLocked(resolvedPath) {
		return &DirectoryIndexStatus{
			Exists:    true,
			Directory: resolvedPath,
			Locked:    true,
			Message:   "Index is currently locked (being updated)",
		}, nil
	}

	// Load directory metadata
	metadata := &models.DirectoryMetadata{}
	if fileUtils.FileExists(indexLocation.MetadataFile) {
		metadataBytes, err := os.ReadFile(indexLocation.MetadataFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory metadata: %w", err)
		}

		if err := metadata.FromJSON(metadataBytes); err != nil {
			return nil, fmt.Errorf("failed to parse directory metadata: %w", err)
		}
	}

	// Get index status
	indexStatus, err := is.GetIndexingStatus(indexLocation.DataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get index status: %w", err)
	}

	return &DirectoryIndexStatus{
		Exists:         true,
		Directory:      resolvedPath,
		IndexLocation:  indexLocation,
		IndexStatus:    indexStatus,
		DirectoryMeta:  *metadata,
		Locked:         false,
		Message:        "Index found and accessible",
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

// DeleteDirectoryIndex removes a directory's index
func (is *IndexingService) DeleteDirectoryIndex(directoryPath string) error {
	fileUtils := lib.NewFileUtilities()

	// Resolve path
	resolvedPath, err := fileUtils.ResolvePath(directoryPath)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Create index location
	indexLocation := fileUtils.CreateIndexLocation(resolvedPath)

	// Note: File locking disabled temporarily to resolve indexing issues
	// if fileUtils.IsLocked(resolvedPath) {
	// 	return fmt.Errorf("cannot delete index: directory '%s' is currently being indexed", resolvedPath)
	// }

	// Remove index directory
	if err := fileUtils.CleanupIndexFiles(indexLocation); err != nil {
		return fmt.Errorf("failed to cleanup index files: %w", err)
	}

	is.logger.Info("Directory index deleted: %s", resolvedPath)
	return nil
}

// GetWorkerPoolStats returns statistics about the worker pool
func (is *IndexingService) GetWorkerPoolStats() lib.WorkerPoolStats {
	if is.workerPool == nil {
		return lib.WorkerPoolStats{}
	}
	return is.workerPool.GetStats()
}

// Close gracefully shuts down the indexing service
func (is *IndexingService) Close(timeout time.Duration) error {
	if is.workerPool != nil {
		is.logger.Info("Shutting down worker pool...")
		return is.workerPool.Stop(timeout)
	}
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

// saveIndexMetadata saves metadata alongside the index file for enhanced search compatibility
func (is *IndexingService) saveIndexMetadata(indexPath string, codeIndex *models.CodeIndex, result *IndexingResult) error {
	// Get model information from the code parser
	var modelName string
	var vectorDim int

	// Try to get model info from the code parser if it has embedding capabilities
	if parserWithEmbedding, ok := is.codeParser.(interface{ GetEmbeddingService() interface{} }); ok {
		if embeddingService := parserWithEmbedding.GetEmbeddingService(); embeddingService != nil {
			if modelGetter, ok := embeddingService.(interface{ ModelName() string }); ok {
				modelName = modelGetter.ModelName()
			}
			if dimGetter, ok := embeddingService.(interface{ Dimensions() int }); ok {
				vectorDim = dimGetter.Dimensions()
			}
		}
	}

	// Fallback to defaults if we couldn't get model info
	if modelName == "" {
		modelName = "all-mpnet-base-v2"
	}
	if vectorDim == 0 {
		vectorDim = 768
	}

	// Create model metadata
	modelMetadata := lib.NewModelMetadata(modelName, vectorDim)

	// Create index metadata
	indexMetadata := lib.NewIndexMetadata(modelMetadata)
	indexMetadata.UpdateMetadata(result.FilesIndexed, result.ChunksCreated, 0, result.Duration)

	// Save metadata to the same directory as the index file
	indexDir := filepath.Dir(indexPath)
	if err := indexMetadata.SaveMetadata(indexDir); err != nil {
		return fmt.Errorf("failed to save index metadata: %w", err)
	}

	is.logger.Debug("Saved index metadata with model: %s (%dD)", modelName, vectorDim)
	return nil
}

// DirectoryIndexStatus contains information about a directory's index
type DirectoryIndexStatus struct {
	Exists        bool                 `json:"exists"`
	Directory     string               `json:"directory"`
	IndexLocation *models.IndexLocation `json:"index_location,omitempty"`
	IndexStatus   *IndexingStatus      `json:"index_status,omitempty"`
	DirectoryMeta models.DirectoryMetadata `json:"directory_meta"`
	Locked        bool                 `json:"locked"`
	Message       string               `json:"message"`
}
