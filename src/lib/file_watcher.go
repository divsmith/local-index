package lib

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code-search/src/models"
)

// FileWatcher monitors file system changes for real-time index updates
type FileWatcher struct {
	watchedDirs    map[string]*WatchContext
	eventChan      chan FileEvent
	stopChan       chan struct{}
	mu             sync.RWMutex
	config         *WatcherConfig
	indexingService IndexingService
	logger         Logger
}

// WatchContext represents a watched directory context
type WatchContext struct {
	DirPath      string
	LastModified map[string]time.Time
	FileHashes   map[string]string
	Ignore       []string
	IsWatching   bool
	StopChan     chan struct{}
}

// FileEvent represents a file system event
type FileEvent struct {
	Path     string
	Op       FileOperation
	Modified time.Time
	Size     int64
	IsDir    bool
}

// FileOperation represents the type of file operation
type FileOperation string

const (
	FileCreated  FileOperation = "created"
	FileModified FileOperation = "modified"
	FileDeleted  FileOperation = "deleted"
	FileRenamed  FileOperation = "renamed"
)

// WatcherConfig contains configuration for the file watcher
type WatcherConfig struct {
	PollInterval    time.Duration `json:"poll_interval"`    // Interval for polling file changes
	BatchDelay      time.Duration `json:"batch_delay"`      // Delay to batch rapid file changes
	MaxBatchSize    int           `json:"max_batch_size"`    // Maximum events per batch
	DebounceDelay   time.Duration `json:"debounce_delay"`   // Delay to debounce rapid changes
	MaxConcurrency  int           `json:"max_concurrency"`  // Maximum concurrent file processing
	BufferSize      int           `json:"buffer_size"`      // Event channel buffer size
	EnableBatching  bool          `json:"enable_batching"`  // Enable event batching
	EnableDebouncing bool         `json:"enable_debouncing"` // Enable event debouncing
}

// DefaultWatcherConfig returns default watcher configuration
func DefaultWatcherConfig() *WatcherConfig {
	return &WatcherConfig{
		PollInterval:     500 * time.Millisecond,
		BatchDelay:       2 * time.Second,
		MaxBatchSize:     100,
		DebounceDelay:    1 * time.Second,
		MaxConcurrency:   4,
		BufferSize:       1000,
		EnableBatching:   true,
		EnableDebouncing: true,
	}
}

// IndexingService interface for updating the index
type IndexingService interface {
	IndexFile(filePath string) error
	RemoveFromIndex(filePath string) error
	UpdateIndexFile(filePath string) error
	GetIndexStats() (*models.IndexStats, error)
}

// Logger interface for logging events
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(config *WatcherConfig, indexingService IndexingService, logger Logger) *FileWatcher {
	if config == nil {
		config = DefaultWatcherConfig()
	}

	return &FileWatcher{
		watchedDirs:    make(map[string]*WatchContext),
		eventChan:      make(chan FileEvent, config.BufferSize),
		stopChan:       make(chan struct{}),
		config:         config,
		indexingService: indexingService,
		logger:         logger,
	}
}

// WatchDirectory starts watching a directory for changes
func (fw *FileWatcher) WatchDirectory(dirPath string, ignorePatterns []string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Check if already watching
	if _, exists := fw.watchedDirs[dirPath]; exists {
		return fmt.Errorf("directory %s is already being watched", dirPath)
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if directory exists
	if stat, err := os.Stat(absPath); err != nil || !stat.IsDir() {
		return fmt.Errorf("directory %s does not exist or is not a directory", absPath)
	}

	// Create watch context
	watchCtx := &WatchContext{
		DirPath:      absPath,
		LastModified: make(map[string]time.Time),
		FileHashes:   make(map[string]string),
		Ignore:       ignorePatterns,
		IsWatching:   true,
		StopChan:     make(chan struct{}),
	}

	// Scan initial directory state
	if err := fw.scanDirectory(watchCtx); err != nil {
		return fmt.Errorf("failed to scan initial directory state: %w", err)
	}

	fw.watchedDirs[absPath] = watchCtx

	// Start monitoring goroutine
	go fw.monitorDirectory(watchCtx)

	fw.logger.Info("Started watching directory: %s", absPath)
	return nil
}

// StopWatching stops watching a directory
func (fw *FileWatcher) StopWatching(dirPath string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	watchCtx, exists := fw.watchedDirs[absPath]
	if !exists {
		return fmt.Errorf("directory %s is not being watched", absPath)
	}

	// Stop monitoring
	watchCtx.IsWatching = false
	close(watchCtx.StopChan)

	// Remove from watched directories
	delete(fw.watchedDirs, absPath)

	fw.logger.Info("Stopped watching directory: %s", absPath)
	return nil
}

// Stop stops all file watching
func (fw *FileWatcher) Stop() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Stop all watchers
	for dirPath, watchCtx := range fw.watchedDirs {
		watchCtx.IsWatching = false
		close(watchCtx.StopChan)
		fw.logger.Info("Stopped watching directory: %s", dirPath)
	}

	fw.watchedDirs = make(map[string]*WatchContext)

	// Stop event processing
	close(fw.stopChan)
}

// StartEventProcessing starts the event processing loop
func (fw *FileWatcher) StartEventProcessing(ctx context.Context) {
	if fw.config.EnableBatching {
		go fw.processBatchedEvents(ctx)
	} else {
		go fw.processIndividualEvents(ctx)
	}
}

// scanDirectory scans a directory and builds initial state
func (fw *FileWatcher) scanDirectory(watchCtx *WatchContext) error {
	return filepath.Walk(watchCtx.DirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		// Skip ignored files/directories
		if fw.shouldIgnore(path, watchCtx.Ignore) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Store file information
		if !info.IsDir() {
			watchCtx.LastModified[path] = info.ModTime()
			if hash, err := fw.calculateFileHash(path); err == nil {
				watchCtx.FileHashes[path] = hash
			}
		}

		return nil
	})
}

// monitorDirectory monitors a directory for changes
func (fw *FileWatcher) monitorDirectory(watchCtx *WatchContext) {
	ticker := time.NewTicker(fw.config.PollInterval)
	defer ticker.Stop()

	var pendingEvents []FileEvent
	var debounceTimer *time.Timer

	for {
		select {
		case <-ticker.C:
			if !watchCtx.IsWatching {
				return
			}

			// Scan for changes
			newEvents := fw.detectChanges(watchCtx)

			if len(newEvents) > 0 {
				if fw.config.EnableDebouncing {
					pendingEvents = append(pendingEvents, newEvents...)

					// Reset debounce timer
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(fw.config.DebounceDelay, func() {
						fw.flushPendingEvents(pendingEvents)
						pendingEvents = nil
					})
				} else {
					// Send events immediately
					for _, event := range newEvents {
						select {
						case fw.eventChan <- event:
						case <-watchCtx.StopChan:
							return
						}
					}
				}
			}

		case <-watchCtx.StopChan:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			// Flush any pending events before stopping
			if len(pendingEvents) > 0 {
				fw.flushPendingEvents(pendingEvents)
			}
			return
		}
	}
}

// detectChanges detects file changes in a directory
func (fw *FileWatcher) detectChanges(watchCtx *WatchContext) []FileEvent {
	var events []FileEvent
	currentFiles := make(map[string]time.Time)
	currentHashes := make(map[string]string)

	// Scan current directory state
	filepath.Walk(watchCtx.DirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if fw.shouldIgnore(path, watchCtx.Ignore) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			currentFiles[path] = info.ModTime()
			if hash, err := fw.calculateFileHash(path); err == nil {
				currentHashes[path] = hash
			}
		}

		return nil
	})

	// Detect deleted files
	for path := range watchCtx.LastModified {
		if _, exists := currentFiles[path]; !exists {
			events = append(events, FileEvent{
				Path:     path,
				Op:       FileDeleted,
				Modified: time.Now(),
			})
		}
	}

	// Detect new and modified files
	for path, modTime := range currentFiles {
		if lastMod, exists := watchCtx.LastModified[path]; !exists {
			// New file
			events = append(events, FileEvent{
				Path:     path,
				Op:       FileCreated,
				Modified: modTime,
			})
		} else if modTime.After(lastMod) {
			// Check if content actually changed (avoid false positives from metadata-only changes)
			if currentHash, currentExists := currentHashes[path]; currentExists {
				if oldHash, oldExists := watchCtx.FileHashes[path]; oldExists {
					if currentHash != oldHash {
						// Content actually changed
						events = append(events, FileEvent{
							Path:     path,
							Op:       FileModified,
							Modified: modTime,
						})
					}
				} else {
					// No old hash, assume modified
					events = append(events, FileEvent{
						Path:     path,
						Op:       FileModified,
						Modified: modTime,
					})
				}
			}
		}
	}

	// Update watch context state
	watchCtx.LastModified = currentFiles
	watchCtx.FileHashes = currentHashes

	return events
}

// flushPendingEvents flushes pending events to the event channel
func (fw *FileWatcher) flushPendingEvents(events []FileEvent) {
	for _, event := range events {
		select {
		case fw.eventChan <- event:
		default:
			fw.logger.Warn("Event channel full, dropping event: %s", event.Path)
		}
	}
}

// processIndividualEvents processes events individually
func (fw *FileWatcher) processIndividualEvents(ctx context.Context) {
	for {
		select {
		case event := <-fw.eventChan:
			fw.handleEvent(event)
		case <-fw.stopChan:
		case <-ctx.Done():
			return
		}
	}
}

// processBatchedEvents processes events in batches
func (fw *FileWatcher) processBatchedEvents(ctx context.Context) {
	ticker := time.NewTicker(fw.config.BatchDelay)
	defer ticker.Stop()

	batch := make([]FileEvent, 0, fw.config.MaxBatchSize)

	for {
		select {
		case event := <-fw.eventChan:
			batch = append(batch, event)

			// Process batch if it reaches max size
			if len(batch) >= fw.config.MaxBatchSize {
				fw.processBatch(batch)
				batch = make([]FileEvent, 0, fw.config.MaxBatchSize)
			}

		case <-ticker.C:
			// Process batch on timer
			if len(batch) > 0 {
				fw.processBatch(batch)
				batch = make([]FileEvent, 0, fw.config.MaxBatchSize)
			}

		case <-fw.stopChan:
		case <-ctx.Done():
			// Process remaining events before exiting
			if len(batch) > 0 {
				fw.processBatch(batch)
			}
			return
		}
	}
}

// processBatch processes a batch of events
func (fw *FileWatcher) processBatch(events []FileEvent) {
	fw.logger.Debug("Processing batch of %d file events", len(events))

	// Group events by operation type
	createEvents := make([]FileEvent, 0)
	modifyEvents := make([]FileEvent, 0)
	deleteEvents := make([]FileEvent, 0)

	for _, event := range events {
		switch event.Op {
		case FileCreated:
			createEvents = append(createEvents, event)
		case FileModified:
			modifyEvents = append(modifyEvents, event)
		case FileDeleted:
			deleteEvents = append(deleteEvents, event)
		}
	}

	// Process events in order of priority
	// Deletes first to avoid conflicts with creates/modifies
	fw.processDeleteEvents(deleteEvents)
	fw.processCreateEvents(createEvents)
	fw.processModifyEvents(modifyEvents)
}

// processCreateEvents processes create events
func (fw *FileWatcher) processCreateEvents(events []FileEvent) {
	semaphore := make(chan struct{}, fw.config.MaxConcurrency)
	var wg sync.WaitGroup

	for _, event := range events {
		wg.Add(1)
		go func(evt FileEvent) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fw.handleCreateEvent(evt)
		}(event)
	}

	wg.Wait()
}

// processModifyEvents processes modify events
func (fw *FileWatcher) processModifyEvents(events []FileEvent) {
	semaphore := make(chan struct{}, fw.config.MaxConcurrency)
	var wg sync.WaitGroup

	for _, event := range events {
		wg.Add(1)
		go func(evt FileEvent) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fw.handleModifyEvent(evt)
		}(event)
	}

	wg.Wait()
}

// processDeleteEvents processes delete events
func (fw *FileWatcher) processDeleteEvents(events []FileEvent) {
	for _, event := range events {
		fw.handleDeleteEvent(event)
	}
}

// handleEvent handles a single file event
func (fw *FileWatcher) handleEvent(event FileEvent) {
	switch event.Op {
	case FileCreated:
		fw.handleCreateEvent(event)
	case FileModified:
		fw.handleModifyEvent(event)
	case FileDeleted:
		fw.handleDeleteEvent(event)
	}
}

// handleCreateEvent handles file creation
func (fw *FileWatcher) handleCreateEvent(event FileEvent) {
	fw.logger.Debug("File created: %s", event.Path)

	// Add file to index
	if err := fw.indexingService.IndexFile(event.Path); err != nil {
		fw.logger.Error("Failed to index created file %s: %v", event.Path, err)
	}
}

// handleModifyEvent handles file modification
func (fw *FileWatcher) handleModifyEvent(event FileEvent) {
	fw.logger.Debug("File modified: %s", event.Path)

	// Update index
	if err := fw.indexingService.UpdateIndexFile(event.Path); err != nil {
		fw.logger.Error("Failed to update index for modified file %s: %v", event.Path, err)
	}
}

// handleDeleteEvent handles file deletion
func (fw *FileWatcher) handleDeleteEvent(event FileEvent) {
	fw.logger.Debug("File deleted: %s", event.Path)

	// Remove from index
	if err := fw.indexingService.RemoveFromIndex(event.Path); err != nil {
		fw.logger.Error("Failed to remove deleted file %s from index: %v", event.Path, err)
	}
}

// shouldIgnore checks if a file should be ignored
func (fw *FileWatcher) shouldIgnore(path string, ignorePatterns []string) bool {
	base := filepath.Base(path)

	// Default ignore patterns
	defaultIgnores := []string{
		".git", ".svn", ".hg",
		"node_modules", ".node_modules",
		".vscode", ".idea",
		"build", "dist", "target",
		".cache", "tmp", "temp",
		".DS_Store", "Thumbs.db",
		"*.tmp", "*.swp", "*.swo",
		"*.log", "*.pid",
	}

	// Check default ignores
	for _, ignore := range defaultIgnores {
		if strings.Contains(path, ignore) || base == ignore {
			return true
		}
	}

	// Check custom ignore patterns
	for _, pattern := range ignorePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

// calculateFileHash calculates a simple hash of a file for change detection
func (fw *FileWatcher) calculateFileHash(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// Simple hash based on file size and modification time
	return fmt.Sprintf("%d_%d", info.Size(), info.ModTime().UnixNano()), nil
}

// GetWatchedDirectories returns the list of currently watched directories
func (fw *FileWatcher) GetWatchedDirectories() []string {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	dirs := make([]string, 0, len(fw.watchedDirs))
	for dirPath := range fw.watchedDirs {
		dirs = append(dirs, dirPath)
	}

	return dirs
}

// IsWatching checks if a directory is being watched
func (fw *FileWatcher) IsWatching(dirPath string) bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return false
	}

	_, exists := fw.watchedDirs[absPath]
	return exists
}

// GetStats returns statistics about the file watcher
func (fw *FileWatcher) GetStats() map[string]interface{} {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	stats := map[string]interface{}{
		"watched_directories": len(fw.watchedDirs),
		"event_channel_size":  len(fw.eventChan),
		"config":             fw.config,
	}

	totalFiles := 0
	for _, watchCtx := range fw.watchedDirs {
		totalFiles += len(watchCtx.LastModified)
	}
	stats["total_watched_files"] = totalFiles

	return stats
}

// FileWatcherStats contains detailed statistics about the file watcher
type FileWatcherStats struct {
	WatchedDirectories int                `json:"watched_directories"`
	TotalWatchedFiles int                `json:"total_watched_files"`
	EventChannelSize  int                `json:"event_channel_size"`
	Config            *WatcherConfig     `json:"config"`
	DirectoryStats    []DirectoryStats   `json:"directory_stats"`
}

// DirectoryStats contains statistics for a single watched directory
type DirectoryStats struct {
	Path      string `json:"path"`
	FileCount int    `json:"file_count"`
	IsWatching bool  `json:"is_watching"`
}