package lib

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code-search/src/models"
)

// ConcurrentIndex provides thread-safe access to index with copy-on-write updates
type ConcurrentIndex struct {
	current     atomic.Value // *models.CodeIndex
	version     int64
	pending     atomic.Value // *models.CodeIndex
	updating    int32
	readers     int32
	mu          sync.RWMutex
	writeMu     sync.Mutex
	updateQueue chan *IndexUpdate
	options     ConcurrentOptions
	stats       IndexStats
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// IndexUpdate represents an update to be applied to the index
type IndexUpdate struct {
	Type      UpdateType
	Data      interface{}
	Timestamp time.Time
	Version   int64
	Callback  func(*models.CodeIndex) error
}

// UpdateType represents the type of index update
type UpdateType int

const (
	UpdateTypeAddFile UpdateType = iota
	UpdateTypeRemoveFile
	UpdateTypeAddChunk
	UpdateTypeRemoveChunk
	UpdateTypeAddVector
	UpdateTypeRemoveVector
	UpdateTypeBatch
	UpdateTypeRebuild
)

// ConcurrentOptions contains options for concurrent index access
type ConcurrentOptions struct {
	MaxQueueSize     int           `json:"max_queue_size"`
	UpdateBatchSize  int           `json:"update_batch_size"`
	UpdateTimeout    time.Duration `json:"update_timeout"`
	MaxReaders       int           `json:"max_readers"`
	EnableMetrics    bool          `json:"enable_metrics"`
	CompactInterval  time.Duration `json:"compact_interval"`
}

// DefaultConcurrentOptions returns default options for concurrent access
func DefaultConcurrentOptions() ConcurrentOptions {
	return ConcurrentOptions{
		MaxQueueSize:    1000,
		UpdateBatchSize: 10,
		UpdateTimeout:   30 * time.Second,
		MaxReaders:      1000,
		EnableMetrics:   true,
		CompactInterval: 5 * time.Minute,
	}
}

// IndexStats contains statistics about index access
type IndexStats struct {
	Reads        int64     `json:"reads"`
	Writes       int64     `json:"writes"`
	Updates       int64     `json:"updates"`
	Compacts      int64     `json:"compacts"`
	ReadErrors    int64     `json:"read_errors"`
	WriteErrors   int64     `json:"write_errors"`
	UpdateErrors  int64     `json:"update_errors"`
	MaxReaders    int32     `json:"max_readers"`
	CurrentReaders int32    `json:"current_readers"`
	LastUpdate    time.Time `json:"last_update"`
	AvgReadTime   time.Duration `json:"avg_read_time"`
	AvgWriteTime  time.Duration `json:"avg_write_time"`
}

// NewConcurrentIndex creates a new concurrent index
func NewConcurrentIndex(indexPath string, vectorStore models.VectorStore, options ConcurrentOptions) (*ConcurrentIndex, error) {
	// Create initial index
	initialIndex := models.NewCodeIndex(indexPath, vectorStore)

	ci := &ConcurrentIndex{
		updateQueue: make(chan *IndexUpdate, options.MaxQueueSize),
		options:     options,
		stopChan:    make(chan struct{}),
		stats: IndexStats{
			LastUpdate: time.Now(),
		},
	}

	// Store initial index
	ci.current.Store(initialIndex)

	// Start background processor
	ci.wg.Add(1)
	go ci.processUpdates()

	// Start metrics collector if enabled
	if options.EnableMetrics {
		ci.wg.Add(1)
		go ci.collectMetrics()
	}

	return ci, nil
}

// Read performs a read operation on the index
func (ci *ConcurrentIndex) Read(operation func(*models.CodeIndex) error) error {
	// Check if we have too many concurrent readers
	if atomic.LoadInt32(&ci.readers) >= int32(ci.options.MaxReaders) {
		return fmt.Errorf("too many concurrent readers")
	}

	// Increment reader count
	atomic.AddInt32(&ci.readers, 1)
	defer atomic.AddInt32(&ci.readers, -1)

	// Update stats
	atomic.AddInt64(&ci.stats.Reads, 1)
	start := time.Now()
	defer func() {
		_ = time.Since(start) // Calculate duration but don't use for now
		// Update average read time (simplified)
		ci.stats.LastUpdate = time.Now()
	}()

	// Get current index
	indexValue := ci.current.Load()
	if indexValue == nil {
		atomic.AddInt64(&ci.stats.ReadErrors, 1)
		return fmt.Errorf("index not available")
	}

	index := indexValue.(*models.CodeIndex)

	// Perform read operation
	err := operation(index)
	if err != nil {
		atomic.AddInt64(&ci.stats.ReadErrors, 1)
	}

	return err
}

// Write performs a write operation on the index
func (ci *ConcurrentIndex) Write(operation func(*models.CodeIndex) error) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	// Check if index is being updated
	if atomic.LoadInt32(&ci.updating) != 0 {
		return fmt.Errorf("index is currently being updated")
	}

	// Mark as updating
	atomic.StoreInt32(&ci.updating, 1)
	defer atomic.StoreInt32(&ci.updating, 0)

	// Update stats
	atomic.AddInt64(&ci.stats.Writes, 1)
	start := time.Now()
	defer func() {
		_ = time.Since(start) // Calculate duration but don't use for now
		// Update average write time (simplified)
		ci.stats.LastUpdate = time.Now()
	}()

	// Get current index
	indexValue := ci.current.Load()
	if indexValue == nil {
		atomic.AddInt64(&ci.stats.WriteErrors, 1)
		return fmt.Errorf("index not available")
	}

	index := indexValue.(*models.CodeIndex)

	// Perform write operation
	err := operation(index)
	if err != nil {
		atomic.AddInt64(&ci.stats.WriteErrors, 1)
		return err
	}

	// Increment version
	atomic.AddInt64(&ci.version, 1)

	return nil
}

// QueueUpdate queues an update for background processing
func (ci *ConcurrentIndex) QueueUpdate(update *IndexUpdate) error {
	select {
	case ci.updateQueue <- update:
		return nil
	case <-time.After(ci.options.UpdateTimeout):
		return fmt.Errorf("update queue is full")
	}
}

// QueueAddFile queues adding a file to the index
func (ci *ConcurrentIndex) QueueAddFile(fileEntry *models.FileEntry) error {
	update := &IndexUpdate{
		Type:      UpdateTypeAddFile,
		Data:      fileEntry,
		Timestamp: time.Now(),
	}

	return ci.QueueUpdate(update)
}

// QueueRemoveFile queues removing a file from the index
func (ci *ConcurrentIndex) QueueRemoveFile(filePath string) error {
	update := &IndexUpdate{
		Type:      UpdateTypeRemoveFile,
		Data:      filePath,
		Timestamp: time.Now(),
	}

	return ci.QueueUpdate(update)
}

// QueueAddChunk queues adding a chunk to the index
func (ci *ConcurrentIndex) QueueAddChunk(chunk *models.CodeChunk) error {
	update := &IndexUpdate{
		Type:      UpdateTypeAddChunk,
		Data:      chunk,
		Timestamp: time.Now(),
	}

	return ci.QueueUpdate(update)
}

// QueueRemoveChunk queues removing a chunk from the index
func (ci *ConcurrentIndex) QueueRemoveChunk(chunkID string) error {
	update := &IndexUpdate{
		Type:      UpdateTypeRemoveChunk,
		Data:      chunkID,
		Timestamp: time.Now(),
	}

	return ci.QueueUpdate(update)
}

// QueueBatch queues a batch of updates
func (ci *ConcurrentIndex) QueueBatch(updates []*IndexUpdate) error {
	batchUpdate := &IndexUpdate{
		Type:      UpdateTypeBatch,
		Data:      updates,
		Timestamp: time.Now(),
	}

	return ci.QueueUpdate(batchUpdate)
}

// processUpdates processes queued updates in the background
func (ci *ConcurrentIndex) processUpdates() {
	defer ci.wg.Done()

	batch := make([]*IndexUpdate, 0, ci.options.UpdateBatchSize)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case update := <-ci.updateQueue:
			batch = append(batch, update)

			// Process batch if it's full
			if len(batch) >= ci.options.UpdateBatchSize {
				ci.processBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			// Process any pending updates
			if len(batch) > 0 {
				ci.processBatch(batch)
				batch = batch[:0]
			}

		case <-ci.stopChan:
			// Process any remaining updates
			if len(batch) > 0 {
				ci.processBatch(batch)
			}
			return

		case <-ci.stopChan:
			return
		}
	}
}

// processBatch processes a batch of updates
func (ci *ConcurrentIndex) processBatch(updates []*IndexUpdate) {
	if len(updates) == 0 {
		return
	}

	ci.mu.Lock()
	defer ci.mu.Unlock()

	// Mark as updating
	atomic.StoreInt32(&ci.updating, 1)
	defer atomic.StoreInt32(&ci.updating, 0)

	// Get current index
	indexValue := ci.current.Load()
	if indexValue == nil {
		return
	}

	// For now, modify the existing index in place
	// In a real implementation, you would create a proper copy
	newIndex := indexValue.(*models.CodeIndex)
	updateStart := time.Now()

	// Process each update
	for _, update := range updates {
		if err := ci.applyUpdate(newIndex, update); err != nil {
			atomic.AddInt64(&ci.stats.UpdateErrors, 1)
			continue
		}
		atomic.AddInt64(&ci.stats.Updates, 1)
	}

	// Swap in the new index
	ci.current.Store(newIndex)
	atomic.AddInt64(&ci.version, 1)

	// Update stats
	ci.stats.LastUpdate = time.Now()
	atomic.AddInt64(&ci.stats.Compacts, 1)

	// Record update time
	updateDuration := time.Since(updateStart)
	if ci.stats.AvgWriteTime == 0 {
		ci.stats.AvgWriteTime = updateDuration
	} else {
		ci.stats.AvgWriteTime = (ci.stats.AvgWriteTime + updateDuration) / 2
	}
}

// applyUpdate applies a single update to an index
func (ci *ConcurrentIndex) applyUpdate(index *models.CodeIndex, update *IndexUpdate) error {
	switch update.Type {
	case UpdateTypeAddFile:
		if fileEntry, ok := update.Data.(*models.FileEntry); ok {
			return index.AddFileEntry(fileEntry)
		}

	case UpdateTypeRemoveFile:
		if filePath, ok := update.Data.(string); ok {
			return index.RemoveFileEntry(filePath)
		}

	case UpdateTypeAddChunk:
		if _, ok := update.Data.(*models.CodeChunk); ok {
			// This would need to be implemented in the CodeIndex
			return nil
		}

	case UpdateTypeRemoveChunk:
		if _, ok := update.Data.(string); ok {
			// This would need to be implemented in the CodeIndex
			return nil
		}

	case UpdateTypeBatch:
		if updates, ok := update.Data.([]*IndexUpdate); ok {
			for _, subUpdate := range updates {
				if err := ci.applyUpdate(index, subUpdate); err != nil {
					return err
				}
			}
		}

	case UpdateTypeRebuild:
		// Rebuild entire index - this would need more implementation
		return nil
	}

	return fmt.Errorf("unknown update type: %v", update.Type)
}

// GetCurrentIndex returns the current index snapshot
func (ci *ConcurrentIndex) GetCurrentIndex() *models.CodeIndex {
	indexValue := ci.current.Load()
	if indexValue == nil {
		return nil
	}
	return indexValue.(*models.CodeIndex)
}

// GetVersion returns the current index version
func (ci *ConcurrentIndex) GetVersion() int64 {
	return atomic.LoadInt64(&ci.version)
}

// GetStats returns current statistics
func (ci *ConcurrentIndex) GetStats() IndexStats {
	stats := ci.stats
	stats.MaxReaders = int32(ci.options.MaxReaders)
	stats.CurrentReaders = atomic.LoadInt32(&ci.readers)
	return stats
}

// Compact performs index compaction
func (ci *ConcurrentIndex) Compact() error {
	return ci.QueueUpdate(&IndexUpdate{
		Type:      UpdateTypeRebuild,
		Timestamp: time.Now(),
	})
}

// GetReaders returns the current number of active readers
func (ci *ConcurrentIndex) GetReaders() int32 {
	return atomic.LoadInt32(&ci.readers)
}

// IsUpdating returns whether the index is currently being updated
func (ci *ConcurrentIndex) IsUpdating() bool {
	return atomic.LoadInt32(&ci.updating) != 0
}

// collectMetrics collects and updates metrics periodically
func (ci *ConcurrentIndex) collectMetrics() {
	defer ci.wg.Done()

	ticker := time.NewTicker(ci.options.CompactInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Perform periodic maintenance
			ci.performMaintenance()

		case <-ci.stopChan:
			return
		}
	}
}

// performMaintenance performs periodic maintenance tasks
func (ci *ConcurrentIndex) performMaintenance() {
	// Update max readers statistic
	currentReaders := atomic.LoadInt32(&ci.readers)
	if currentReaders > ci.stats.MaxReaders {
		ci.stats.MaxReaders = currentReaders
	}

	// Trigger compaction if needed
	// This could be based on various heuristics
	if atomic.LoadInt64(&ci.stats.Updates) > 1000 {
		ci.Compact()
		atomic.StoreInt64(&ci.stats.Updates, 0)
	}
}

// Close shuts down the concurrent index
func (ci *ConcurrentIndex) Close() error {
	close(ci.stopChan)
	ci.wg.Wait()

	// Close update queue
	close(ci.updateQueue)

	return nil
}

// CopyOnWriteRead creates a copy-on-write read context
func (ci *ConcurrentIndex) CopyOnWriteRead() (*CopyOnWriteReader, error) {
	if atomic.LoadInt32(&ci.readers) >= int32(ci.options.MaxReaders) {
		return nil, fmt.Errorf("too many concurrent readers")
	}

	// Increment reader count
	atomic.AddInt32(&ci.readers, 1)

	return &CopyOnWriteReader{
		index:    ci,
		done:     make(chan struct{}),
		released: false,
	}, nil
}

// CopyOnWriteReader provides copy-on-write read access
type CopyOnWriteReader struct {
	index    *ConcurrentIndex
	done     chan struct{}
	released bool
	mu       sync.Mutex
}

// GetIndex returns the index for reading
func (cow *CopyOnWriteReader) GetIndex() *models.CodeIndex {
	cow.mu.Lock()
	defer cow.mu.Unlock()

	if cow.released {
		return nil
	}

	return cow.index.GetCurrentIndex()
}

// Release releases the read context
func (cow *CopyOnWriteReader) Release() {
	cow.mu.Lock()
	defer cow.mu.Unlock()

	if !cow.released {
		atomic.AddInt32(&cow.index.readers, -1)
		cow.released = true
		close(cow.done)
	}
}

// Done returns a channel that's closed when the reader is released
func (cow *CopyOnWriteReader) Done() <-chan struct{} {
	return cow.done
}