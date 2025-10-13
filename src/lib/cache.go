package lib

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"code-search/src/models"
)

// QueryCache represents a multi-level query caching system
type QueryCache struct {
	l1Cache     *sync.Map              // In-memory LRU cache
	l2Cache     *PersistentCache       // Disk-based cache
	l3Cache     *PatternCache          // Pre-computed patterns
	statistics  CacheStatistics
	maxL1Size   int
	maxL2Size   int
	ttl         time.Duration
	mu          sync.RWMutex
}

// CacheStatistics tracks cache performance statistics
type CacheStatistics struct {
	L1Hits       int64
	L1Misses     int64
	L2Hits       int64
	L2Misses     int64
	L3Hits       int64
	L3Misses     int64
	TotalQueries int64
	LastCleanup  time.Time
	mu           sync.RWMutex
}

// CacheEntry represents a cached search result with metadata
type CacheEntry struct {
	Results    *models.SearchResults `json:"results"`
	QueryHash  string               `json:"query_hash"`
	Expiration time.Time            `json:"expiration"`
	CreatedAt  time.Time            `json:"created_at"`
	AccessCount int64               `json:"access_count"`
	LastAccess  time.Time           `json:"last_access"`
}

// PersistentCache manages disk-based caching
type PersistentCache struct {
	basePath string
	maxSize  int
	ttl      time.Duration
	mu       sync.RWMutex
}

// PatternCache manages pre-computed common query patterns
type PatternCache struct {
	patterns map[string]*models.SearchResults
	mu       sync.RWMutex
}

// NewQueryCache creates a new multi-level query cache
func NewQueryCache(l1Size, l2Size int, ttl time.Duration) *QueryCache {
	cache := &QueryCache{
		l1Cache:    &sync.Map{},
		maxL1Size:  l1Size,
		maxL2Size:  l2Size,
		ttl:        ttl,
		statistics: CacheStatistics{},
	}

	// Initialize L2 cache
	cache.l2Cache = NewPersistentCache(l2Size, ttl)

	// Initialize L3 cache
	cache.l3Cache = NewPatternCache()

	// Start cleanup goroutine
	go cache.startCleanupRoutine()

	return cache
}

// NewPersistentCache creates a new disk-based cache
func NewPersistentCache(maxSize int, ttl time.Duration) *PersistentCache {
	// Create cache directory
	cacheDir := filepath.Join(os.TempDir(), "code_search_cache")
	os.MkdirAll(cacheDir, 0755)

	return &PersistentCache{
		basePath: cacheDir,
		maxSize:  maxSize,
		ttl:      ttl,
	}
}

// NewPatternCache creates a new pattern cache for common queries
func NewPatternCache() *PatternCache {
	patterns := make(map[string]*models.SearchResults)

	// Initialize with common patterns
	// This could be expanded based on usage analytics
	commonPatterns := []string{
		"function",
		"class",
		"import",
		"error",
		"TODO",
		"FIXME",
		"return",
	}

	for _, pattern := range commonPatterns {
		patterns[pattern] = nil // Will be populated on first use
	}

	return &PatternCache{
		patterns: patterns,
	}
}

// Get retrieves cached results for a query
func (qc *QueryCache) Get(query *models.SearchQuery) (*models.SearchResults, bool) {
	qc.statistics.mu.Lock()
	qc.statistics.TotalQueries++
	qc.statistics.mu.Unlock()

	queryHash := qc.hashQuery(query)

	// Try L1 cache (in-memory)
	if entry, found := qc.getFromL1(queryHash); found {
		qc.incrementL1Hits()
		return entry.Results, true
	}
	qc.incrementL1Misses()

	// Try L2 cache (disk-based)
	if entry, found := qc.getFromL2(queryHash); found {
		qc.incrementL2Hits()
		// Promote to L1
		qc.putToL1(queryHash, entry)
		return entry.Results, true
	}
	qc.incrementL2Misses()

	// Try L3 cache (pattern matching)
	if result, found := qc.getFromL3(query); found {
		qc.incrementL3Hits()
		return result, true
	}
	qc.incrementL3Misses()

	return nil, false
}

// Put stores search results in the cache
func (qc *QueryCache) Put(query *models.SearchQuery, results *models.SearchResults) {
	queryHash := qc.hashQuery(query)

	entry := &CacheEntry{
		Results:     results,
		QueryHash:   queryHash,
		Expiration:  time.Now().Add(qc.ttl),
		CreatedAt:   time.Now(),
		AccessCount: 1,
		LastAccess:  time.Now(),
	}

	// Store in L1 and L2
	qc.putToL1(queryHash, entry)
	qc.putToL2(queryHash, entry)
}

// hashQuery creates a deterministic hash for a query
func (qc *QueryCache) hashQuery(query *models.SearchQuery) string {
	h := md5.New()

	// Include all relevant query fields
	h.Write([]byte(query.QueryText))
	h.Write([]byte(query.SearchType))
	h.Write([]byte(fmt.Sprintf("%f", query.Threshold)))
	h.Write([]byte(fmt.Sprintf("%d", query.MaxResults)))

	// Include file filter
	if query.FileFilter != "" {
		h.Write([]byte(query.FileFilter))
	}

	// Include language filter
	if query.LanguageFilter != "" {
		h.Write([]byte(query.LanguageFilter))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// L1 Cache Methods (In-memory)

func (qc *QueryCache) getFromL1(hash string) (*CacheEntry, bool) {
	if value, found := qc.l1Cache.Load(hash); found {
		entry := value.(*CacheEntry)
		if time.Now().Before(entry.Expiration) {
			entry.AccessCount++
			entry.LastAccess = time.Now()
			return entry, true
		}
		// Entry expired, remove it
		qc.l1Cache.Delete(hash)
	}
	return nil, false
}

func (qc *QueryCache) putToL1(hash string, entry *CacheEntry) {
	// Check if we need to evict entries
	qc.evictL1IfNeeded()

	qc.l1Cache.Store(hash, entry)
}

func (qc *QueryCache) evictL1IfNeeded() {
	// Simple size-based eviction - remove oldest entries if cache is full
	count := 0
	qc.l1Cache.Range(func(key, value interface{}) bool {
		count++
		return count < qc.maxL1Size
	})

	if count >= qc.maxL1Size {
		// Remove the oldest 25% of entries
		toRemove := qc.maxL1Size / 4
		removed := 0

		qc.l1Cache.Range(func(key, value interface{}) bool {
			if removed < toRemove {
				entry := value.(*CacheEntry)
				// Don't remove recently accessed entries
				if time.Since(entry.LastAccess) > time.Minute {
					qc.l1Cache.Delete(key)
					removed++
				}
			}
			return removed < toRemove
		})
	}
}

// L2 Cache Methods (Disk-based)

func (qc *QueryCache) getFromL2(hash string) (*CacheEntry, bool) {
	return qc.l2Cache.Get(hash)
}

func (qc *QueryCache) putToL2(hash string, entry *CacheEntry) {
	qc.l2Cache.Put(hash, entry)
}

// L3 Cache Methods (Pattern matching)

func (qc *QueryCache) getFromL3(query *models.SearchQuery) (*models.SearchResults, bool) {
	return qc.l3Cache.Get(query)
}

// PersistentCache Methods

func (pc *PersistentCache) Get(hash string) (*CacheEntry, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	filePath := filepath.Join(pc.basePath, hash+".cache")

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		os.Remove(filePath) // Remove corrupted cache file
		return nil, false
	}

	// Check if entry is still valid
	if time.Now().After(entry.Expiration) {
		os.Remove(filePath)
		return nil, false
	}

	entry.AccessCount++
	entry.LastAccess = time.Now()

	return &entry, true
}

func (pc *PersistentCache) Put(hash string, entry *CacheEntry) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	filePath := filepath.Join(pc.basePath, hash+".cache")

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	// Clean up if needed
	pc.cleanupIfNeeded()

	// Write to disk
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to write cache entry: %v\n", err)
	}
}

func (pc *PersistentCache) cleanupIfNeeded() {
	// Check current cache size
	files, err := filepath.Glob(filepath.Join(pc.basePath, "*.cache"))
	if err != nil {
		return
	}

	if len(files) < pc.maxSize {
		return
	}

	// Remove oldest files
	fileInfos := make([]os.FileInfo, 0, len(files))
	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			fileInfos = append(fileInfos, info)
		}
	}

	// Sort by modification time (oldest first)
	// Simple selection sort for small number of files
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].ModTime().Before(fileInfos[j].ModTime()) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// Remove oldest 25% of files
	toRemove := pc.maxSize / 4
	for i := 0; i < len(fileInfos) && i < toRemove; i++ {
		filePath := filepath.Join(pc.basePath, fileInfos[i].Name())
		os.Remove(filePath)
	}
}

// PatternCache Methods

func (pc *PatternCache) Get(query *models.SearchQuery) (*models.SearchResults, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	// Check if query matches a known pattern
	for pattern := range pc.patterns {
		if queryMatchesPattern(query, pattern) {
			if result := pc.patterns[pattern]; result != nil {
				return result, true
			}
		}
	}

	return nil, false
}

func (pc *PatternCache) Put(pattern string, results *models.SearchResults) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.patterns[pattern] = results
}

// queryMatchesPattern checks if a query matches a pattern
func queryMatchesPattern(query *models.SearchQuery, pattern string) bool {
	// Simple pattern matching - this could be made more sophisticated
	return query.QueryText == pattern ||
		   len(query.QueryText) > len(pattern) &&
		   query.QueryText[:len(pattern)] == pattern
}

// Statistics Methods

func (qc *QueryCache) incrementL1Hits() {
	qc.statistics.mu.Lock()
	defer qc.statistics.mu.Unlock()
	qc.statistics.L1Hits++
}

func (qc *QueryCache) incrementL1Misses() {
	qc.statistics.mu.Lock()
	defer qc.statistics.mu.Unlock()
	qc.statistics.L1Misses++
}

func (qc *QueryCache) incrementL2Hits() {
	qc.statistics.mu.Lock()
	defer qc.statistics.mu.Unlock()
	qc.statistics.L2Hits++
}

func (qc *QueryCache) incrementL2Misses() {
	qc.statistics.mu.Lock()
	defer qc.statistics.mu.Unlock()
	qc.statistics.L2Misses++
}

func (qc *QueryCache) incrementL3Hits() {
	qc.statistics.mu.Lock()
	defer qc.statistics.mu.Unlock()
	qc.statistics.L3Hits++
}

func (qc *QueryCache) incrementL3Misses() {
	qc.statistics.mu.Lock()
	defer qc.statistics.mu.Unlock()
	qc.statistics.L3Misses++
}

// GetStatistics returns cache performance statistics
func (qc *QueryCache) GetStatistics() CacheStatistics {
	qc.statistics.mu.RLock()
	defer qc.statistics.mu.RUnlock()
	return qc.statistics
}

// GetHitRates returns the hit rates for each cache level
func (qc *QueryCache) GetHitRates() (l1Rate, l2Rate, l3Rate, overallRate float64) {
	stats := qc.GetStatistics()

	if stats.TotalQueries == 0 {
		return 0, 0, 0, 0
	}

	l1Rate = float64(stats.L1Hits) / float64(stats.TotalQueries)
	l2Rate = float64(stats.L2Hits) / float64(stats.TotalQueries)
	l3Rate = float64(stats.L3Hits) / float64(stats.TotalQueries)

	totalHits := stats.L1Hits + stats.L2Hits + stats.L3Hits
	overallRate = float64(totalHits) / float64(stats.TotalQueries)

	return l1Rate, l2Rate, l3Rate, overallRate
}

// startCleanupRoutine starts a background goroutine to clean up expired entries
func (qc *QueryCache) startCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		qc.cleanup()
	}
}

// cleanup removes expired entries from all cache levels
func (qc *QueryCache) cleanup() {
	now := time.Now()

	// Clean L1 cache
	qc.l1Cache.Range(func(key, value interface{}) bool {
		entry := value.(*CacheEntry)
		if now.After(entry.Expiration) {
			qc.l1Cache.Delete(key)
		}
		return true
	})

	// Clean L2 cache
	qc.l2Cache.cleanup()

	// Update statistics
	qc.statistics.mu.Lock()
	qc.statistics.LastCleanup = now
	qc.statistics.mu.Unlock()
}

// cleanup removes expired entries from persistent cache
func (pc *PersistentCache) cleanup() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(pc.basePath, "*.cache"))
	if err != nil {
		return
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			os.Remove(file)
			continue
		}

		var entry CacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			os.Remove(file)
			continue
		}

		if time.Now().After(entry.Expiration) {
			os.Remove(file)
		}
	}
}

// Clear empties all cache levels
func (qc *QueryCache) Clear() {
	// Clear L1
	qc.l1Cache = &sync.Map{}

	// Clear L2
	qc.l2Cache.Clear()

	// Reset statistics
	qc.statistics = CacheStatistics{}
}

// Clear clears the persistent cache
func (pc *PersistentCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(pc.basePath, "*.cache"))
	if err != nil {
		return
	}

	for _, file := range files {
		os.Remove(file)
	}
}