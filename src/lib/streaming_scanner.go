package lib

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

// StreamingScanner provides streaming directory traversal with buffered channels
type StreamingScanner struct {
	rootPath    string
	options     ScannerOptions
	fileChan    chan StreamingScanResult
	errorChan   chan error
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.RWMutex
	stats       ScannerStats
	fileCounter int64
	dirCounter  int64
	pool        *BufferPool
}

// ScannerOptions contains options for streaming scanning
type ScannerOptions struct {
	BufferSize        int           `json:"buffer_size"`
	MaxWorkers       int           `json:"max_workers"`
	ExcludePatterns  []string      `json:"exclude_patterns"`
	IncludeHidden    bool          `json:"include_hidden"`
	FollowSymlinks    bool          `json:"follow_symlinks"`
	MaxFileSize      int64         `json:"max_file_size"`
	EnableMonitoring bool          `json:"enable_monitoring"`
	ReadTimeout      time.Duration `json:"read_timeout"`
	IOPriority       int           `json:"io_priority"`
}

// StreamingScanResult represents a scanned file or directory
type StreamingScanResult struct {
	Path       string     `json:"path"`
	IsDir      bool       `json:"is_dir"`
	Size       int64      `json:"size"`
	ModTime    time.Time  `json:"mod_time"`
	Mode       fs.FileMode `json:"mode"`
	Info       fs.FileInfo `json:"info"`
	Error      error      `json:"error,omitempty"`
}

// ScannerStats contains statistics about the scanning process
type ScannerStats struct {
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	Duration       time.Duration `json:"duration"`
	FilesScanned   int64         `json:"files_scanned"`
	DirsScanned    int64         `json:"dirs_scanned"`
	FilesSkipped   int64         `json:"files_skipped"`
	BytesScanned   int64         `json:"bytes_scanned"`
	Errors         int64         `json:"errors"`
	Throughput     float64       `json:"throughput_mbps"`
	AvgFileSize    float64       `json:"avg_file_size"`
}

// DefaultScannerOptions returns default options for streaming scanning
func DefaultScannerOptions() ScannerOptions {
	return ScannerOptions{
		BufferSize:       1000,
	MaxWorkers:      runtime.NumCPU(),
	ExcludePatterns:  []string{".git/*", "node_modules/*", "*.tmp", "*.log", "*.cache"},
	IncludeHidden:    false,
		FollowSymlinks:    false,
		MaxFileSize:      50 * 1024 * 1024, // 50MB
	EnableMonitoring: true,
	ReadTimeout:      5 * time.Second,
		IOPriority:       0, // Normal I/O priority
	}
}

// NewStreamingScanner creates a new streaming scanner
func NewStreamingScanner(rootPath string, options ScannerOptions) *StreamingScanner {
	ctx, cancel := context.WithCancel(context.Background())

	return &StreamingScanner{
		rootPath:  rootPath,
		options:   options,
		fileChan:  make(chan StreamingScanResult, options.BufferSize),
		errorChan: make(chan error, 100),
		ctx:       ctx,
		cancel:    cancel,
		stats:     ScannerStats{StartTime: time.Now()},
		pool:      GetPoolManager().GetBufferPool(),
	}
}

// Scan starts the streaming scan process
func (ss *StreamingScanner) Scan() (<-chan StreamingScanResult, <-chan error) {
	// Validate root path
	if _, err := os.Stat(ss.rootPath); err != nil {
		ss.errorChan <- fmt.Errorf("invalid root path: %w", err)
		close(ss.fileChan)
		close(ss.errorChan)
		return ss.fileChan, ss.errorChan
	}

	// Start scan goroutine
	ss.wg.Add(1)
	go ss.scanDirectory()

	// Start stats collection if enabled
	if ss.options.EnableMonitoring {
		ss.wg.Add(1)
		go ss.collectStats()
	}

	return ss.fileChan, ss.errorChan
}

// Stop stops the scanning process
func (ss *StreamingScanner) Stop() {
	ss.cancel()
	ss.wg.Wait()
	close(ss.fileChan)
	close(ss.errorChan)
}

// scanDirectory performs the actual directory scanning
func (ss *StreamingScanner) scanDirectory() {
	defer ss.wg.Done()

	// Use worker pool for concurrent processing
	workerPool := NewWorkerPool(PoolOptions{
		MinWorkers:    1,
		MaxWorkers:    ss.options.MaxWorkers,
		QueueSize:     ss.options.BufferSize,
		EnableMetrics: ss.options.EnableMonitoring,
	})

	defer workerPool.Close()

	// Walk directory and submit path processing tasks
	filepath.WalkDir(ss.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			ss.errorChan <- fmt.Errorf("error accessing %s: %w", path, err)
			return nil
		}

		// Check if context is cancelled
		select {
		case <-ss.ctx.Done():
			return fmt.Errorf("scan cancelled")
		default:
		}

		// Check if we should skip this path
		if ss.shouldSkip(path, d) {
			return nil
		}

		// Submit to worker pool
		future := workerPool.Submit(func() (interface{}, error) {
			return ss.processPath(path, d)
		})

		// Handle result asynchronously
		go func() {
			result, err := future.Get()
			if err != nil {
				ss.errorChan <- err
				return
			}

			if scanResult, ok := result.(StreamingScanResult); ok {
				select {
				case ss.fileChan <- scanResult:
					// Update counters
					if scanResult.IsDir {
						atomic.AddInt64(&ss.dirCounter, 1)
					} else {
						atomic.AddInt64(&ss.fileCounter, 1)
						atomic.AddInt64(&ss.stats.BytesScanned, scanResult.Size)
					}
				case <-ss.ctx.Done():
					return
				}
			}
		}()

		return nil
	})
}

// processPath processes a single path (file or directory)
func (ss *StreamingScanner) processPath(path string, d fs.DirEntry) (StreamingScanResult, error) {
	info, err := d.Info()
	if err != nil {
		return StreamingScanResult{Path: path, Error: err}, err
	}

	result := StreamingScanResult{
		Path:    path,
		IsDir:   d.IsDir(),
		Size:    info.Size(),
		ModTime:  info.ModTime(),
		Mode:    info.Mode(),
		Info:    info,
	}

	// Set I/O priority for this thread if supported
	if ss.options.IOPriority > 0 {
		_ = unix.Setpriority(0, 0, ss.options.IOPriority)
	}

	// Check file size limit
	if !d.IsDir() && info.Size() > ss.options.MaxFileSize {
		result.Error = fmt.Errorf("file too large: %d bytes", info.Size())
		atomic.AddInt64(&ss.stats.FilesSkipped, 1)
	}

	return result, nil
}

// shouldSkip determines if a path should be skipped during scanning
func (ss *StreamingScanner) shouldSkip(path string, d fs.DirEntry) bool {
	// Skip hidden files unless explicitly included
	if !ss.options.IncludeHidden && strings.HasPrefix(filepath.Base(path), ".") {
		atomic.AddInt64(&ss.stats.FilesSkipped, 1)
		return true
	}

	// Check exclude patterns
	relativePath, err := filepath.Rel(ss.rootPath, path)
	if err == nil {
		for _, pattern := range ss.options.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relativePath); matched {
				atomic.AddInt64(&ss.stats.FilesSkipped, 1)
				return true
			}
		}
	}

	// Skip directories that are symbolic links unless following symlinks
	if d.IsDir() {
		if info, err := d.Info(); err == nil {
			if info.Mode()&os.ModeSymlink != 0 && !ss.options.FollowSymlinks {
				atomic.AddInt64(&ss.stats.FilesSkipped, 1)
				return true
			}
		}
	}

	return false
}

// collectStats periodically updates scanning statistics
func (ss *StreamingScanner) collectStats() {
	defer ss.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ss.updateStats()
		case <-ss.ctx.Done():
			ss.updateStats()
			return
		}
	}
}

// updateStats updates the current statistics
func (ss *StreamingScanner) updateStats() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.stats.EndTime = time.Now()
	ss.stats.Duration = ss.stats.EndTime.Sub(ss.stats.StartTime)
	ss.stats.FilesScanned = atomic.LoadInt64(&ss.fileCounter)
	ss.stats.DirsScanned = atomic.LoadInt64(&ss.dirCounter)

	// Calculate throughput
	if ss.stats.Duration > 0 {
		ss.stats.Throughput = float64(ss.stats.BytesScanned) / ss.stats.Duration.Seconds() / (1024 * 1024) // MB/s
	}

	// Calculate average file size
	if ss.stats.FilesScanned > 0 {
		ss.stats.AvgFileSize = float64(ss.stats.BytesScanned) / float64(ss.stats.FilesScanned)
	}
}

// GetStats returns current scanning statistics
func (ss *StreamingScanner) GetStats() ScannerStats {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.stats
}

// BufferedScanner provides buffered file scanning with backpressure control
type BufferedScanner struct {
	*StreamingScanner
	resultBuffer  []StreamingScanResult
	errorBuffer   []error
	maxBuffer     int
	flushInterval  time.Duration
	lastFlush     time.Time
	autoFlush     bool
}

// NewBufferedScanner creates a new buffered scanner
func NewBufferedScanner(rootPath string, options ScannerOptions, maxBuffer int) *BufferedScanner {
	streaming := NewStreamingScanner(rootPath, options)

	return &BufferedScanner{
		StreamingScanner: streaming,
		resultBuffer:     make([]StreamingScanResult, 0, maxBuffer),
		errorBuffer:      make([]error, 0, 100),
		maxBuffer:        maxBuffer,
		flushInterval:    100 * time.Millisecond,
		lastFlush:       time.Now(),
		autoFlush:        true,
	}
}

// Start starts the buffered scanning with automatic flushing
func (bs *BufferedScanner) Start() {
	fileChan, errorChan := bs.StreamingScanner.Scan()

	// Start result collector
	bs.wg.Add(1)
	go bs.collectResults(fileChan, errorChan)
}

// collectResults collects results from the streaming scanner
func (bs *BufferedScanner) collectResults(fileChan <-chan StreamingScanResult, errorChan <-chan error) {
	defer bs.wg.Done()

	flushTicker := time.NewTicker(bs.flushInterval)
	defer flushTicker.Stop()

	for {
		select {
		case result := <-fileChan:
			bs.resultBuffer = append(bs.resultBuffer, result)
			if bs.autoFlush && len(bs.resultBuffer) >= bs.maxBuffer {
				bs.flushResults()
			}

		case err := <-errorChan:
			bs.errorBuffer = append(bs.errorBuffer, err)
			if bs.autoFlush {
				bs.flushResults()
			}

		case <-flushTicker.C:
			if bs.autoFlush && time.Since(bs.lastFlush) >= bs.flushInterval {
				bs.flushResults()
			}

		case <-bs.ctx.Done():
			bs.flushResults()
			return
		}
	}
}

// flushResults flushes buffered results to registered handlers
func (bs *BufferedScanner) flushResults() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if len(bs.resultBuffer) == 0 && len(bs.errorBuffer) == 0 {
		return
	}

	// In a real implementation, this would call registered result handlers
	// For now, we just clear the buffers
	bs.resultBuffer = bs.resultBuffer[:0]
	bs.errorBuffer = bs.errorBuffer[:0]
	bs.lastFlush = time.Now()
}

// GetResults returns a copy of the buffered results
func (bs *BufferedScanner) GetResults() []StreamingScanResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	results := make([]StreamingScanResult, len(bs.resultBuffer))
	copy(results, bs.resultBuffer)
	return results
}

// GetErrors returns a copy of the buffered errors
func (bs *BufferedScanner) GetErrors() []error {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	errors := make([]error, len(bs.errorBuffer))
	copy(errors, bs.errorBuffer)
	return errors
}

// SetAutoFlush enables or disables automatic flushing
func (bs *BufferedScanner) SetAutoFlush(enabled bool) {
	bs.autoFlush = enabled
}

// ForceFlush manually flushes the buffers
func (bs *BufferedScanner) ForceFlush() {
	bs.flushResults()
}

// FastScanner provides high-performance scanning with optimizations
type FastScanner struct {
	*StreamingScanner
	directories map[string]*DirectoryInfo
	dirMu        sync.RWMutex
	enableCache  bool
}

// DirectoryInfo contains cached information about a directory
type DirectoryInfo struct {
	Path       string       `json:"path"`
	ModTime    time.Time    `json:"mod_time"`
	FileCount  int          `json:"file_count"`
	TotalSize  int64        `json:"total_size"`
	Files      []CachedFileInfo  `json:"files"`
	ScannedAt  time.Time    `json:"scanned_at"`
}

// CachedFileInfo contains cached information about a file
type CachedFileInfo struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Hash    string    `json:"hash"`
}

// NewFastScanner creates a new fast scanner with caching
func NewFastScanner(rootPath string, options ScannerOptions, enableCache bool) *FastScanner {
	streaming := NewStreamingScanner(rootPath, options)

	fs := &FastScanner{
	StreamingScanner: streaming,
		directories:      make(map[string]*DirectoryInfo),
		enableCache:      enableCache,
	}

	return fs
}

// ScanDirectoryFast performs fast scanning of a directory with caching
func (fs *FastScanner) ScanDirectoryFast(dirPath string) ([]StreamingScanResult, error) {
	// Check cache first
	if fs.enableCache {
		if cached := fs.getCachedDirectory(dirPath); cached != nil {
			results := make([]StreamingScanResult, len(cached.Files))
			for i, file := range cached.Files {
				results[i] = StreamingScanResult{
					Path:    file.Path,
					IsDir:   false,
					Size:    file.Size,
					ModTime: file.ModTime,
					// Note: Mode and Info would need to be reconstructed
				}
			}
			return results, nil
		}
	}

	// Perform actual scan
	var results []StreamingScanResult
	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if fs.shouldSkip(path, d) {
			return nil
		}

		result, err := fs.processPath(path, d)
		if err != nil {
			return nil
		}

		results = append(results, result)
		return nil
	})

	// Cache the results
	if fs.enableCache && len(results) > 0 {
		fs.cacheDirectory(dirPath, results)
	}

	return results, nil
}

// getCachedDirectory retrieves cached directory information
func (fs *FastScanner) getCachedDirectory(dirPath string) *DirectoryInfo {
	fs.dirMu.RLock()
	defer fs.dirMu.RUnlock()

	dirInfo, exists := fs.directories[dirPath]
	if !exists {
		return nil
	}

	// Check if cache is still valid
	if info, err := os.Stat(dirPath); err == nil {
		if info.ModTime().Equal(dirInfo.ModTime) {
			return dirInfo
		}
	}

	// Cache is stale, remove it
	delete(fs.directories, dirPath)
	return nil
}

// cacheDirectory caches directory information
func (fs *FastScanner) cacheDirectory(dirPath string, results []StreamingScanResult) {
	fs.dirMu.Lock()
	defer fs.dirMu.Unlock()

	// Get directory info
	info, err := os.Stat(dirPath)
	if err != nil {
		return
	}

	// Create directory info
	dirInfo := &DirectoryInfo{
		Path:      dirPath,
		ModTime:   info.ModTime(),
		FileCount: len(results),
		TotalSize:  0,
		Files:     make([]CachedFileInfo, len(results)),
		ScannedAt: time.Now(),
	}

	// Populate files
	for i, result := range results {
		dirInfo.Files[i] = CachedFileInfo{
			Path:    result.Path,
			Size:    result.Size,
			ModTime: result.ModTime,
			Hash:    "", // Would be calculated if needed
		}
		dirInfo.TotalSize += result.Size
	}

	// Cache the directory info
	fs.directories[dirPath] = dirInfo

	// Clean up old cache entries
	fs.cleanupCache()
}

// cleanupCache removes old entries from the cache
func (fs *FastScanner) cleanupCache() {
	// Keep only recent cache entries (simplified)
	const maxCacheSize = 1000

	if len(fs.directories) > maxCacheSize {
	// Remove oldest entries (simplified implementation)
	// In a real implementation, you'd use LRU or similar
		count := 0
		for key := range fs.directories {
			count++
			if count > maxCacheSize {
				delete(fs.directories, key)
			}
		}
	}
}

// GetCacheStats returns statistics about the directory cache
func (fs *FastScanner) GetCacheStats() map[string]interface{} {
	fs.dirMu.RLock()
	defer fs.dirMu.RUnlock()

	return map[string]interface{}{
		"cached_directories": len(fs.directories),
		"max_cache_size":    1000, // Could be configurable
		"cache_enabled":     fs.enableCache,
	}
}