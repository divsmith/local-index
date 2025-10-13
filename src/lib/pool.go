package lib

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// PoolManager manages multiple object pools for different types
type PoolManager struct {
	pools map[string]interface{}
	mu    sync.RWMutex
	stats PoolStats
}

// PoolStats contains statistics about pool usage
type PoolStats struct {
	TotalAllocations int64 `json:"total_allocations"`
	TotalReleases   int64 `json:"total_releases"`
	ActiveObjects   int64 `json:"active_objects"`
	PoolHits        int64 `json:"pool_hits"`
	PoolMisses      int64 `json:"pool_misses"`
	LastUpdate      time.Time `json:"last_update"`
}

// VectorPool manages pooling of float64 slices for vectors
type VectorPool struct {
	pools map[int]*sync.Pool // Pools indexed by vector size
	mu    sync.RWMutex
	stats PoolStats
}

// ChunkPool manages pooling of code chunks
type ChunkPool struct {
	pool      sync.Pool
	stats     PoolStats
	created   int64
	reused    int64
	maxSize   int
}

// BufferPool manages pooling of byte buffers
type BufferPool struct {
	smallPool  sync.Pool // 1KB buffers
	mediumPool sync.Pool // 4KB buffers
	largePool  sync.Pool // 16KB buffers
	stats      PoolStats
}

// SearchResultPool manages pooling of search results
type SearchResultPool struct {
	pool      sync.Pool
	stats     PoolStats
	created   int64
	reused    int64
	maxSize   int
}

var globalPoolManager *PoolManager
var poolManagerOnce sync.Once

// GetPoolManager returns the global pool manager instance
func GetPoolManager() *PoolManager {
	poolManagerOnce.Do(func() {
		globalPoolManager = NewPoolManager()
	})
	return globalPoolManager
}

// NewPoolManager creates a new pool manager
func NewPoolManager() *PoolManager {
	pm := &PoolManager{
		pools: make(map[string]interface{}),
		stats: PoolStats{
			LastUpdate: time.Now(),
		},
	}

	// Initialize default pools
	pm.registerPool("vectors", NewVectorPool())
	pm.registerPool("chunks", NewChunkPool(1000))
	pm.registerPool("buffers", NewBufferPool())
	pm.registerPool("search_results", NewSearchResultPool(500))

	return pm
}

// registerPool registers a pool with the manager
func (pm *PoolManager) registerPool(name string, pool interface{}) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pools[name] = pool
}

// GetVectorPool returns the vector pool
func (pm *PoolManager) GetVectorPool() *VectorPool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pool, ok := pm.pools["vectors"].(*VectorPool); ok {
		return pool
	}
	return NewVectorPool()
}

// GetChunkPool returns the chunk pool
func (pm *PoolManager) GetChunkPool() *ChunkPool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pool, ok := pm.pools["chunks"].(*ChunkPool); ok {
		return pool
	}
	return NewChunkPool(1000)
}

// GetBufferPool returns the buffer pool
func (pm *PoolManager) GetBufferPool() *BufferPool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pool, ok := pm.pools["buffers"].(*BufferPool); ok {
		return pool
	}
	return NewBufferPool()
}

// GetSearchResultPool returns the search result pool
func (pm *PoolManager) GetSearchResultPool() *SearchResultPool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pool, ok := pm.pools["search_results"].(*SearchResultPool); ok {
		return pool
	}
	return NewSearchResultPool(500)
}

// GetStats returns aggregated statistics for all pools
func (pm *PoolManager) GetStats() PoolStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// This is a simplified aggregation - in production, each pool would track its own stats
	totalStats := PoolStats{
		LastUpdate: time.Now(),
	}

	for range pm.pools {
		// Each pool type would implement its own stats method
		// For now, we'll return basic stats
	}

	return totalStats
}

// NewVectorPool creates a new vector pool
func NewVectorPool() *VectorPool {
	return &VectorPool{
		pools: make(map[int]*sync.Pool),
		stats: PoolStats{
			LastUpdate: time.Now(),
		},
	}
}

// GetVector gets a vector of the specified size from the pool
func (vp *VectorPool) GetVector(size int) []float64 {
	if size <= 0 {
		return make([]float64, 0)
	}

	vp.mu.RLock()
	pool, exists := vp.pools[size]
	vp.mu.RUnlock()

	if !exists {
		vp.mu.Lock()
		// Double-check after acquiring write lock
		if pool, exists = vp.pools[size]; !exists {
			pool = &sync.Pool{
				New: func() interface{} {
					atomic.AddInt64(&vp.stats.TotalAllocations, 1)
					return make([]float64, size)
				},
			}
			vp.pools[size] = pool
		}
		vp.mu.Unlock()
	}

	vector := pool.Get().([]float64)
	atomic.AddInt64(&vp.stats.PoolHits, 1)
	atomic.AddInt64(&vp.stats.ActiveObjects, 1)

	return vector
}

// PutVector returns a vector to the pool
func (vp *VectorPool) PutVector(vector []float64) {
	if vector == nil || len(vector) == 0 {
		return
	}

	size := len(vector)
	vp.mu.RLock()
	pool, exists := vp.pools[size]
	vp.mu.RUnlock()

	if exists {
		// Clear the vector before returning to pool
		for i := range vector {
			vector[i] = 0
		}
		pool.Put(vector)
		atomic.AddInt64(&vp.stats.TotalReleases, 1)
		atomic.AddInt64(&vp.stats.ActiveObjects, -1)
	}
}

// GetStats returns vector pool statistics
func (vp *VectorPool) GetStats() PoolStats {
	vp.stats.LastUpdate = time.Now()
	return vp.stats
}

// NewChunkPool creates a new chunk pool
func NewChunkPool(maxSize int) *ChunkPool {
	return &ChunkPool{
		maxSize: maxSize,
		stats: PoolStats{
			LastUpdate: time.Now(),
		},
		pool: sync.Pool{
			New: func() interface{} {
				atomic.AddInt64(&ChunkPoolInstance.created, 1)
				// Create a generic interface that can be used for chunks
				return make([]interface{}, 0, 50) // Pre-allocate capacity
			},
		},
	}
}

var ChunkPoolInstance = &ChunkPool{} // Global instance for atomic counters

// GetChunk gets a chunk from the pool
func (cp *ChunkPool) GetChunk() []interface{} {
	chunk := cp.pool.Get().([]interface{})
	if len(chunk) > 0 {
		// Clear existing data
		for i := range chunk {
			chunk[i] = nil
		}
		chunk = chunk[:0] // Reset length but keep capacity
		atomic.AddInt64(&cp.reused, 1)
	} else {
		atomic.AddInt64(&cp.created, 1)
	}

	atomic.AddInt64(&cp.stats.TotalAllocations, 1)
	atomic.AddInt64(&cp.stats.ActiveObjects, 1)
	return chunk
}

// PutChunk returns a chunk to the pool
func (cp *ChunkPool) PutChunk(chunk []interface{}) {
	if chunk == nil {
		return
	}

	// Clear the chunk
	for i := range chunk {
		chunk[i] = nil
	}
	chunk = chunk[:0]

	cp.pool.Put(chunk)
	atomic.AddInt64(&cp.stats.TotalReleases, 1)
	atomic.AddInt64(&cp.stats.ActiveObjects, -1)
}

// GetStats returns chunk pool statistics
func (cp *ChunkPool) GetStats() PoolStats {
	cp.stats.LastUpdate = time.Now()
	return cp.stats
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		stats: PoolStats{
			LastUpdate: time.Now(),
		},
		smallPool: sync.Pool{
			New: func() interface{} {
				atomic.AddInt64(&BufferPoolInstance.stats.TotalAllocations, 1)
				return make([]byte, 0, 1024) // 1KB capacity
			},
		},
		mediumPool: sync.Pool{
			New: func() interface{} {
				atomic.AddInt64(&BufferPoolInstance.stats.TotalAllocations, 1)
				return make([]byte, 0, 4096) // 4KB capacity
			},
		},
		largePool: sync.Pool{
			New: func() interface{} {
				atomic.AddInt64(&BufferPoolInstance.stats.TotalAllocations, 1)
				return make([]byte, 0, 16384) // 16KB capacity
			},
		},
	}
}

var BufferPoolInstance = &BufferPool{} // Global instance for atomic counters

// GetBuffer gets a buffer of appropriate size from the pool
func (bp *BufferPool) GetBuffer(size int) []byte {
	var buffer []byte

	switch {
	case size <= 1024:
		buffer = bp.smallPool.Get().([]byte)
		atomic.AddInt64(&bp.stats.PoolHits, 1)
	case size <= 4096:
		buffer = bp.mediumPool.Get().([]byte)
		atomic.AddInt64(&bp.stats.PoolHits, 1)
	case size <= 16384:
		buffer = bp.largePool.Get().([]byte)
		atomic.AddInt64(&bp.stats.PoolHits, 1)
	default:
		// For very large buffers, allocate directly
		buffer = make([]byte, 0, size)
		atomic.AddInt64(&bp.stats.PoolMisses, 1)
	}

	if len(buffer) > 0 {
		// Clear existing data
		buffer = buffer[:0]
	}

	atomic.AddInt64(&bp.stats.ActiveObjects, 1)
	return buffer
}

// PutBuffer returns a buffer to the appropriate pool
func (bp *BufferPool) PutBuffer(buffer []byte) {
	if buffer == nil {
		return
	}

	capacity := cap(buffer)

	// Clear the buffer
	for i := range buffer {
		buffer[i] = 0
	}
	buffer = buffer[:0]

	switch {
	case capacity == 1024:
		bp.smallPool.Put(buffer)
	case capacity == 4096:
		bp.mediumPool.Put(buffer)
	case capacity == 16384:
		bp.largePool.Put(buffer)
	default:
		// Don't pool buffers with non-standard capacities
		return
	}

	atomic.AddInt64(&bp.stats.TotalReleases, 1)
	atomic.AddInt64(&bp.stats.ActiveObjects, -1)
}

// GetStats returns buffer pool statistics
func (bp *BufferPool) GetStats() PoolStats {
	bp.stats.LastUpdate = time.Now()
	return bp.stats
}

// NewSearchResultPool creates a new search result pool
func NewSearchResultPool(maxSize int) *SearchResultPool {
	return &SearchResultPool{
		maxSize: maxSize,
		stats: PoolStats{
			LastUpdate: time.Now(),
		},
		pool: sync.Pool{
			New: func() interface{} {
				atomic.AddInt64(&SearchResultPoolInstance.created, 1)
				// Return a generic interface that can be used for search results
				return make([]interface{}, 0, 20) // Pre-allocate capacity
			},
		},
	}
}

var SearchResultPoolInstance = &SearchResultPool{} // Global instance for atomic counters

// GetSearchResults gets a search results slice from the pool
func (srp *SearchResultPool) GetSearchResults() []interface{} {
	results := srp.pool.Get().([]interface{})
	if len(results) > 0 {
		// Clear existing data
		for i := range results {
			results[i] = nil
		}
		results = results[:0] // Reset length but keep capacity
		atomic.AddInt64(&srp.reused, 1)
	} else {
		atomic.AddInt64(&srp.created, 1)
	}

	atomic.AddInt64(&srp.stats.TotalAllocations, 1)
	atomic.AddInt64(&srp.stats.ActiveObjects, 1)
	return results
}

// PutSearchResults returns search results to the pool
func (srp *SearchResultPool) PutSearchResults(results []interface{}) {
	if results == nil {
		return
	}

	// Clear the results
	for i := range results {
		results[i] = nil
	}
	results = results[:0]

	srp.pool.Put(results)
	atomic.AddInt64(&srp.stats.TotalReleases, 1)
	atomic.AddInt64(&srp.stats.ActiveObjects, -1)
}

// GetStats returns search result pool statistics
func (srp *SearchResultPool) GetStats() PoolStats {
	srp.stats.LastUpdate = time.Now()
	return srp.stats
}

// MemoryMonitor provides memory usage monitoring
type MemoryMonitor struct {
	lastGC      time.Time
	gcThreshold uint64 // Memory threshold in bytes
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor() *MemoryMonitor {
	return &MemoryMonitor{
		lastGC:      time.Now(),
		gcThreshold: 100 * 1024 * 1024, // 100MB default
	}
}

// ShouldGC determines if garbage collection should be triggered
func (mm *MemoryMonitor) ShouldGC() bool {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Trigger GC if memory usage exceeds threshold or if enough time has passed
	memoryUsage := m.Alloc
	timeSinceLastGC := time.Since(mm.lastGC)

	return memoryUsage > mm.gcThreshold || timeSinceLastGC > 5*time.Minute
}

// TriggerGC triggers garbage collection and updates tracking
func (mm *MemoryMonitor) TriggerGC() {
	runtime.GC()
	mm.lastGC = time.Now()
}

// GetMemoryUsage returns current memory usage statistics
func (mm *MemoryMonitor) GetMemoryUsage() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// Cleanup performs cleanup operations on all pools
func (pm *PoolManager) Cleanup() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Trigger garbage collection if needed
	monitor := NewMemoryMonitor()
	if monitor.ShouldGC() {
		monitor.TriggerGC()
	}

	// Additional cleanup operations could be added here
	// For example, shrinking pools that haven't been used recently
}