package lib

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// PerformanceOptimizer handles performance optimizations for large directories
type PerformanceOptimizer struct {
	workerPool    int
	batchSize     int
	maxMemoryMB   int64
	parallelWalk  bool
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer() *PerformanceOptimizer {
	return &PerformanceOptimizer{
		workerPool:   runtime.NumCPU(),
		batchSize:    1000,
		maxMemoryMB:  512, // 512MB memory limit
		parallelWalk: true,
	}
}

// ScanOptions contains options for directory scanning
type ScanOptions struct {
	BaseDir        string        // Base directory for relative path calculations
	MaxConcurrency int           // Maximum number of concurrent goroutines
	BatchSize      int           // Number of files to process in each batch
	MaxDepth       int           // Maximum directory depth to scan (0 = unlimited)
	FollowSymlinks bool          // Follow symbolic links
	FileTypes      []string      // Specific file types to include (empty = all)
	ExcludeDirs    []string      // Directory names to exclude
	MaxFileSize    int64         // Maximum file size to include (0 = unlimited)
	Timeout        time.Duration // Timeout for scanning operation
}

// DefaultScanOptions returns default scan options optimized for performance
func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		BaseDir:        "", // Will be set by caller
		MaxConcurrency: runtime.NumCPU(),
		BatchSize:      1000,
		MaxDepth:       0, // Unlimited depth
		FollowSymlinks: false,
		FileTypes:      []string{},
		ExcludeDirs:    []string{".git", "node_modules", ".clindex", ".svn", "vendor"},
		MaxFileSize:    100 * 1024 * 1024, // 100MB
		Timeout:        30 * time.Minute,
	}
}

// ScanResult contains the results of an optimized directory scan
type ScanResult struct {
	Files         []FileInfo `json:"files"`
	TotalFiles    int64       `json:"total_files"`
	TotalSize     int64       `json:"total_size"`
	TotalDirs     int64       `json:"total_dirs"`
	SkippedFiles  int64       `json:"skipped_files"`
	SkippedDirs   int64       `json:"skipped_dirs"`
	Errors        []string    `json:"errors"`
	Duration      string      `json:"duration"`
	MemoryUsedMB  float64     `json:"memory_used_mb"`
}

// FileInfo represents information about a file during scanning
type FileInfo struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IsDir        bool      `json:"is_dir"`
	Extension    string    `json:"extension"`
	RelativePath string    `json:"relative_path"`
}

// FastDirectoryScan performs an optimized scan of a directory
func (po *PerformanceOptimizer) FastDirectoryScan(directory string, options ScanOptions) (*ScanResult, error) {
	start := time.Now()

	// Create context with timeout
	ctx := context.Background()
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	result := &ScanResult{}

	// Memory monitoring
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Use parallel walk if enabled and directory is large enough
	if po.parallelWalk && options.MaxConcurrency > 1 {
		err := po.parallelDirectoryWalk(ctx, directory, options, result)
		if err != nil {
			return nil, fmt.Errorf("parallel walk failed: %w", err)
		}
	} else {
		err := po.sequentialDirectoryWalk(ctx, directory, options, result)
		if err != nil {
			return nil, fmt.Errorf("sequential walk failed: %w", err)
		}
	}

	// Calculate memory usage
	runtime.ReadMemStats(&memAfter)
	memUsed := memAfter.Alloc - memBefore.Alloc
	result.MemoryUsedMB = float64(memUsed) / (1024 * 1024)

	result.Duration = time.Since(start).String()

	return result, nil
}

// parallelDirectoryWalk performs directory walking in parallel
func (po *PerformanceOptimizer) parallelDirectoryWalk(ctx context.Context, directory string, options ScanOptions, result *ScanResult) error {
	// Channel for file paths
	filePaths := make(chan string, options.BatchSize*2)

	// Channel for results
	results := make(chan FileInfo, options.BatchSize)

	// Channel for errors
	errors := make(chan error, 10)

	// Worker pool
	var wg sync.WaitGroup
	workers := options.MaxConcurrency

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go po.fileWorker(ctx, &wg, filePaths, results, errors, options)
	}

	// Start result collector
	go po.collectResults(ctx, results, result, options.BatchSize)

	// Walk directory and send file paths to workers
	err := po.produceFilePaths(ctx, directory, filePaths, options, result)
	if err != nil {
		return err
	}

	// Close file path channel
	close(filePaths)

	// Wait for workers to finish
	wg.Wait()
	close(results)

	// Collect any remaining errors
	close(errors)
	for err := range errors {
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return nil
}

// sequentialDirectoryWalk performs directory walking sequentially
func (po *PerformanceOptimizer) sequentialDirectoryWalk(ctx context.Context, directory string, options ScanOptions, result *ScanResult) error {
	return filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error accessing %s: %v", path, err))
			return nil // Continue walking
		}

		// Get relative path
		relPath, err := filepath.Rel(directory, path)
		if err != nil {
			relPath = path
		}

		if d.IsDir() {
			// Check if directory should be excluded
			if po.shouldExcludeDir(d.Name(), options.ExcludeDirs) {
				if path != directory { // Don't skip the root directory
					result.SkippedDirs++
					return filepath.SkipDir
				}
			}

			result.TotalDirs++
			return nil
		}

		// File processing
		info, err := d.Info()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error getting info for %s: %v", path, err))
			return nil
		}

		// Check file size
		if options.MaxFileSize > 0 && info.Size() > options.MaxFileSize {
			result.SkippedFiles++
			return nil
		}

		// Check file type
		if len(options.FileTypes) > 0 && !po.shouldIncludeFile(path, options.FileTypes) {
			result.SkippedFiles++
			return nil
		}

		// Add file info
		result.Files = append(result.Files, FileInfo{
			Path:         path,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			IsDir:        false,
			Extension:    filepath.Ext(path),
			RelativePath: relPath,
		})

		result.TotalFiles++
		result.TotalSize += info.Size()

		return nil
	})
}

// produceFilePaths walks directory and sends file paths to workers
func (po *PerformanceOptimizer) produceFilePaths(ctx context.Context, directory string, filePaths chan<- string, options ScanOptions, result *ScanResult) error {
	return filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error accessing %s: %v", path, err))
			return nil
		}

		if d.IsDir() {
			// Check if directory should be excluded
			if po.shouldExcludeDir(d.Name(), options.ExcludeDirs) {
				if path != directory { // Don't skip the root directory
					result.SkippedDirs++
					return filepath.SkipDir
				}
			}

			result.TotalDirs++
			return nil
		}

		// Quick file checks before sending to worker
		info, err := d.Info()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error getting info for %s: %v", path, err))
			return nil
		}

		// Check file size
		if options.MaxFileSize > 0 && info.Size() > options.MaxFileSize {
			result.SkippedFiles++
			return nil
		}

		// Check file type
		if len(options.FileTypes) > 0 && !po.shouldIncludeFile(path, options.FileTypes) {
			result.SkippedFiles++
			return nil
		}

		// Send file path to worker
		select {
		case filePaths <- path:
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})
}

// fileWorker processes files from the filePaths channel
func (po *PerformanceOptimizer) fileWorker(ctx context.Context, wg *sync.WaitGroup, filePaths <-chan string, results chan<- FileInfo, errors chan<- error, options ScanOptions) {
	defer wg.Done()

	for path := range filePaths {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Get file info
		info, err := os.Stat(path)
		if err != nil {
			select {
			case errors <- fmt.Errorf("error stating %s: %w", path, err):
			case <-ctx.Done():
				return
			}
			return
		}

		// Get relative path
		relPath, err := filepath.Rel(options.BaseDir, path)
		if err != nil {
			relPath = path
		}

		// Create file info
		fileInfo := FileInfo{
			Path:         path,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			IsDir:        false,
			Extension:    filepath.Ext(path),
			RelativePath: relPath,
		}

		select {
		case results <- fileInfo:
		case <-ctx.Done():
			return
		}
	}
}

// collectResults collects results from the results channel
func (po *PerformanceOptimizer) collectResults(ctx context.Context, results <-chan FileInfo, result *ScanResult, batchSize int) {
	var batch []FileInfo

	for {
		select {
		case fileInfo, ok := <-results:
			if !ok {
				// Flush remaining batch
				result.Files = append(result.Files, batch...)
				return
			}

			batch = append(batch, fileInfo)
			result.TotalFiles++
			result.TotalSize += fileInfo.Size

			// Flush batch when it reaches the desired size
			if len(batch) >= batchSize {
				result.Files = append(result.Files, batch...)
				batch = batch[:0] // Reset batch
			}

		case <-ctx.Done():
			return
		}
	}
}

// shouldExcludeDir checks if a directory should be excluded
func (po *PerformanceOptimizer) shouldExcludeDir(dirName string, excludeDirs []string) bool {
	for _, exclude := range excludeDirs {
		if dirName == exclude {
			return true
		}
	}
	return false
}

// shouldIncludeFile checks if a file should be included based on file types
func (po *PerformanceOptimizer) shouldIncludeFile(path string, fileTypes []string) bool {
	ext := filepath.Ext(path)
	for _, fileType := range fileTypes {
		if fileType == ext || fileType == "*"+ext || fileType == path {
			return true
		}
	}
	return false
}

// OptimizeForLargeDirectory adjusts optimization parameters for large directories
func (po *PerformanceOptimizer) OptimizeForLargeDirectory(estimatedFileCount int64) {
	if estimatedFileCount > 100000 { // 100k+ files
		po.workerPool = runtime.NumCPU() * 2
		po.batchSize = 5000
		po.parallelWalk = true
	} else if estimatedFileCount > 10000 { // 10k-100k files
		po.workerPool = runtime.NumCPU()
		po.batchSize = 2000
		po.parallelWalk = true
	} else { // < 10k files
		po.workerPool = runtime.NumCPU() / 2
		po.batchSize = 1000
		po.parallelWalk = false
	}
}

// GetMemoryUsage returns current memory usage statistics
func (po *PerformanceOptimizer) GetMemoryUsage() (float64, float64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / (1024 * 1024)
	sysMB := float64(m.Sys) / (1024 * 1024)

	return allocMB, sysMB
}

// EstimateDirectorySize provides a quick estimate of directory size without full scan
func (po *PerformanceOptimizer) EstimateDirectorySize(directory string) (int64, int64, error) {
	var totalSize, fileCount int64
	sampleSize := int64(0)
	sampleCount := int64(0)

	err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors for estimation
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		fileCount++
		totalSize += info.Size()

		// Sample every 100th file for quick estimation
		if fileCount%100 == 0 {
			sampleCount++
			sampleSize += info.Size()
		}

		// Stop after 1000 samples for quick estimation
		if fileCount >= 100000 {
			return fmt.Errorf("estimation limit reached")
		}

		return nil
	})

	if err != nil && err.Error() != "estimation limit reached" {
		return 0, 0, err
	}

	// If we have a sample, use it to estimate total
	if sampleCount > 0 && fileCount > 1000 {
		avgFileSize := float64(sampleSize) / float64(sampleCount)
		estimatedTotal := float64(totalSize) * (float64(fileCount) / 1000.0)
		// Use avgFileSize in the calculation to avoid unused variable warning
		estimatedTotal = avgFileSize * float64(fileCount)
		return int64(estimatedTotal), fileCount, nil
	}

	return totalSize, fileCount, nil
}

// BatchProcessor processes files in batches for memory efficiency
type BatchProcessor struct {
	batchSize    int
	processor    func([]FileInfo) error
	memoryLimit  int64
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, processor func([]FileInfo) error) *BatchProcessor {
	return &BatchProcessor{
		batchSize:   batchSize,
		processor:   processor,
		memoryLimit: 100 * 1024 * 1024, // 100MB default
	}
}

// ProcessFiles processes files in batches
func (bp *BatchProcessor) ProcessFiles(files []FileInfo) error {
	for i := 0; i < len(files); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		if err := bp.processor(batch); err != nil {
			return fmt.Errorf("batch processing failed at batch %d: %w", i/bp.batchSize, err)
		}

		// Optional: Force garbage collection if memory usage is high
		if bp.memoryLimit > 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.Alloc > uint64(bp.memoryLimit) {
				runtime.GC()
			}
		}
	}

	return nil
}