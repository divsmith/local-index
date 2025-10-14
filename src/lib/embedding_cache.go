package lib

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EmbeddingCache provides efficient caching for text embeddings
type EmbeddingCache struct {
	l1Cache      *sync.Map                    // In-memory cache for frequently used embeddings
	l2Cache      *PersistentEmbeddingCache    // Disk-based cache for larger embeddings
	statistics   EmbeddingCacheStatistics
	maxL1Size    int
	maxL2Size    int64 // Maximum disk size in bytes
	ttl          time.Duration
	memoryLimit  int64 // Memory limit in bytes
	currentSize  int64 // Current memory usage
	mu           sync.RWMutex
}

// EmbeddingCacheStatistics tracks embedding cache performance
type EmbeddingCacheStatistics struct {
	L1Hits          int64
	L1Misses        int64
	L2Hits          int64
	L2Misses        int64
	TotalRequests   int64
	CacheSize       int64
	MemoryUsage     int64
	DiskUsage       int64
	LastCleanup     time.Time
	AverageHitTime  time.Duration
	mu              sync.RWMutex
}

// EmbeddingCacheEntry represents a cached embedding with metadata
type EmbeddingCacheEntry struct {
	Embedding    []float32           `json:"embedding"`
	TextHash     string              `json:"text_hash"`
	ModelName    string              `json:"model_name"`
	CreatedAt    time.Time           `json:"created_at"`
	LastAccess   time.Time           `json:"last_access"`
	AccessCount  int64               `json:"access_count"`
	Size         int64               `json:"size"`         // Size in bytes
	Expiration   time.Time           `json:"expiration"`
	Tags         []string            `json:"tags,omitempty"` // For categorization
}

// PersistentEmbeddingCache manages disk-based embedding storage
type PersistentEmbeddingCache struct {
	basePath   string
	maxSize    int64
	ttl        time.Duration
	indexFile  string
	mu         sync.RWMutex
	index      map[string]*EmbeddingIndexEntry
}

// EmbeddingIndexEntry tracks metadata for cached embeddings
type EmbeddingIndexEntry struct {
	FileName    string    `json:"file_name"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	LastAccess  time.Time `json:"last_access"`
	AccessCount int64     `json:"access_count"`
	ModelName   string    `json:"model_name"`
	TextHash    string    `json:"text_hash"`
}

// NewEmbeddingCache creates a new embedding cache with specified limits
func NewEmbeddingCache(l1Size int, l2Size int64, ttl time.Duration, memoryLimit int64) *EmbeddingCache {
	cache := &EmbeddingCache{
		l1Cache:     &sync.Map{},
		maxL1Size:   l1Size,
		maxL2Size:   l2Size,
		ttl:         ttl,
		memoryLimit: memoryLimit,
		statistics: EmbeddingCacheStatistics{},
	}

	// Initialize L2 cache
	cache.l2Cache = NewPersistentEmbeddingCache(l2Size, ttl)

	// Start cleanup routine
	go cache.startCleanupRoutine()

	return cache
}

// NewPersistentEmbeddingCache creates a new disk-based embedding cache
func NewPersistentEmbeddingCache(maxSize int64, ttl time.Duration) *PersistentEmbeddingCache {
	cacheDir := filepath.Join(os.TempDir(), "code_search_embeddings")
	os.MkdirAll(cacheDir, 0755)

	indexFile := filepath.Join(cacheDir, "embedding_index.json")

	cache := &PersistentEmbeddingCache{
		basePath:  cacheDir,
		maxSize:   maxSize,
		ttl:       ttl,
		indexFile: indexFile,
		index:     make(map[string]*EmbeddingIndexEntry),
	}

	// Load existing index
	cache.loadIndex()

	return cache
}

// Get retrieves a cached embedding for the given text and model
func (ec *EmbeddingCache) Get(text, modelName string) ([]float32, bool) {
	start := time.Now()

	ec.statistics.mu.Lock()
	ec.statistics.TotalRequests++
	ec.statistics.mu.Unlock()

	textHash := ec.hashText(text)
	cacheKey := fmt.Sprintf("%s:%s", modelName, textHash)

	// Try L1 cache (in-memory)
	if entry, found := ec.getFromL1(cacheKey); found {
		ec.recordL1Hit(start)
		return entry.Embedding, true
	}
	ec.recordL1Miss()

	// Try L2 cache (disk-based)
	if entry, found := ec.getFromL2(cacheKey); found {
		ec.recordL2Hit(start)
		// Promote to L1 if space allows
		ec.putToL1(cacheKey, entry)
		return entry.Embedding, true
	}
	ec.recordL2Miss()

	return nil, false
}

// Put stores an embedding in the cache
func (ec *EmbeddingCache) Put(text, modelName string, embedding []float32) {
	textHash := ec.hashText(text)
	cacheKey := fmt.Sprintf("%s:%s", modelName, textHash)

	entry := &EmbeddingCacheEntry{
		Embedding:   embedding,
		TextHash:    textHash,
		ModelName:   modelName,
		CreatedAt:   time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 1,
		Size:        int64(len(embedding) * 4), // float32 = 4 bytes
		Expiration:  time.Now().Add(ec.ttl),
		Tags:        ec.generateTags(text),
	}

	// Store in both levels
	ec.putToL1(cacheKey, entry)
	ec.putToL2(cacheKey, entry)

	// Update statistics
	ec.updateMemoryUsage()
}

// GetMultiple retrieves multiple cached embeddings efficiently
func (ec *EmbeddingCache) GetMultiple(texts []string, modelName string) ([][]float32, []bool) {
	results := make([][]float32, len(texts))
	found := make([]bool, len(texts))

	for i, text := range texts {
		if embedding, ok := ec.Get(text, modelName); ok {
			results[i] = embedding
			found[i] = true
		}
	}

	return results, found
}

// PutMultiple stores multiple embeddings efficiently
func (ec *EmbeddingCache) PutMultiple(texts []string, modelName string, embeddings [][]float32) {
	for i, text := range texts {
		if i < len(embeddings) {
			ec.Put(text, modelName, embeddings[i])
		}
	}
}

// GetByPattern retrieves embeddings that match a pattern
func (ec *EmbeddingCache) GetByPattern(pattern, modelName string) map[string][]float32 {
	results := make(map[string][]float32)

	// Search through L1 cache
	ec.l1Cache.Range(func(key, value interface{}) bool {
		cacheKey := key.(string)
		if ec.matchesModel(cacheKey, modelName) {
			entry := value.(*EmbeddingCacheEntry)
			if ec.matchesPattern(entry, pattern) {
				// Extract text hash from cache key
				parts := fmt.Sprintf("%s", cacheKey)
				if len(parts) > len(modelName)+1 {
					textHash := parts[len(modelName)+1:]
					results[textHash] = entry.Embedding
				}
			}
		}
		return true
	})

	// Add L2 cache results if needed
	// This would require disk scanning, which is more expensive
	// For now, we only return L1 results

	return results
}

// Clear removes all cached embeddings
func (ec *EmbeddingCache) Clear() {
	// Clear L1 cache
	ec.l1Cache = &sync.Map{}

	// Clear L2 cache
	ec.l2Cache.Clear()

	// Reset statistics
	ec.mu.Lock()
	ec.currentSize = 0
	ec.mu.Unlock()

	ec.statistics = EmbeddingCacheStatistics{}
}

// GetStatistics returns cache performance statistics
func (ec *EmbeddingCache) GetStatistics() EmbeddingCacheStatistics {
	ec.statistics.mu.RLock()
	defer ec.statistics.mu.RUnlock()

	// Update current statistics
	stats := ec.statistics
	ec.updateMemoryUsage()
	stats.MemoryUsage = ec.currentSize
	stats.DiskUsage = ec.l2Cache.getCurrentSize()

	return stats
}

// Private helper methods

func (ec *EmbeddingCache) hashText(text string) string {
	h := md5.New()
	h.Write([]byte(text))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (ec *EmbeddingCache) getFromL1(cacheKey string) (*EmbeddingCacheEntry, bool) {
	if value, found := ec.l1Cache.Load(cacheKey); found {
		entry := value.(*EmbeddingCacheEntry)
		if time.Now().Before(entry.Expiration) {
			entry.AccessCount++
			entry.LastAccess = time.Now()
			return entry, true
		}
		// Entry expired, remove it
		ec.l1Cache.Delete(cacheKey)
		ec.mu.Lock()
		ec.currentSize -= entry.Size
		ec.mu.Unlock()
	}
	return nil, false
}

func (ec *EmbeddingCache) putToL1(cacheKey string, entry *EmbeddingCacheEntry) {
	// Check if we need to evict entries
	ec.evictL1IfNeeded()

	ec.l1Cache.Store(cacheKey, entry)
	ec.mu.Lock()
	ec.currentSize += entry.Size
	ec.mu.Unlock()
}

func (ec *EmbeddingCache) evictL1IfNeeded() {
	ec.mu.RLock()
	needsEviction := ec.currentSize > ec.memoryLimit || ec.getL1Count() > ec.maxL1Size
	ec.mu.RUnlock()

	if !needsEviction {
		return
	}

	// Collect entries for eviction
	var entries []*cacheEntryInfo
	ec.l1Cache.Range(func(key, value interface{}) bool {
		entry := value.(*EmbeddingCacheEntry)
		entries = append(entries, &cacheEntryInfo{
			key:    key.(string),
			entry:  entry,
			score:  ec.calculateEvictionScore(entry),
		})
		return true
	})

	// Sort by eviction score (lowest score gets evicted first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].score > entries[j].score {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Evict entries until we have enough space
	targetSize := ec.memoryLimit * 3 / 4 // Leave 25% headroom
	targetCount := ec.maxL1Size * 3 / 4

	evictedSize := int64(0)
	evictedCount := 0

	for _, entryInfo := range entries {
		if (ec.currentSize - evictedSize) <= targetSize && (ec.getL1Count() - evictedCount) <= targetCount {
			break
		}

		ec.l1Cache.Delete(entryInfo.key)
		evictedSize += entryInfo.entry.Size
		evictedCount++
	}

	ec.mu.Lock()
	ec.currentSize -= evictedSize
	ec.mu.Unlock()
}

func (ec *EmbeddingCache) getL1Count() int {
	count := 0
	ec.l1Cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func (ec *EmbeddingCache) calculateEvictionScore(entry *EmbeddingCacheEntry) float64 {
	// Lower score = more likely to be evicted
	age := time.Since(entry.LastAccess)
	frequency := float64(entry.AccessCount)

	// Score favors recently accessed and frequently used entries
	score := frequency / (age.Seconds() + 1)

	// Boost score for entries with tags (likely important)
	if len(entry.Tags) > 0 {
		score *= 1.5
	}

	return score
}

func (ec *EmbeddingCache) getFromL2(cacheKey string) (*EmbeddingCacheEntry, bool) {
	return ec.l2Cache.Get(cacheKey)
}

func (ec *EmbeddingCache) putToL2(cacheKey string, entry *EmbeddingCacheEntry) {
	ec.l2Cache.Put(cacheKey, entry)
}

func (ec *EmbeddingCache) generateTags(text string) []string {
	tags := make([]string, 0)

	// Add length-based tags
	if len(text) < 50 {
		tags = append(tags, "short")
	} else if len(text) > 500 {
		tags = append(tags, "long")
	}

	// Add pattern-based tags
	if containsCodePattern(text) {
		tags = append(tags, "code")
	}
	if containsDocumentation(text) {
		tags = append(tags, "documentation")
	}

	return tags
}

func (ec *EmbeddingCache) matchesPattern(entry *EmbeddingCacheEntry, pattern string) bool {
	for _, tag := range entry.Tags {
		if tag == pattern {
			return true
		}
	}
	return false
}

func (ec *EmbeddingCache) matchesModel(cacheKey, modelName string) bool {
	return len(cacheKey) > len(modelName) && cacheKey[:len(modelName)] == modelName
}

func (ec *EmbeddingCache) updateMemoryUsage() {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Recalculate current memory usage
	var totalSize int64
	ec.l1Cache.Range(func(key, value interface{}) bool {
		entry := value.(*EmbeddingCacheEntry)
		totalSize += entry.Size
		return true
	})
	ec.currentSize = totalSize
}

func (ec *EmbeddingCache) recordL1Hit(start time.Time) {
	ec.statistics.mu.Lock()
	defer ec.statistics.mu.Unlock()

	ec.statistics.L1Hits++
	hitTime := time.Since(start)

	// Update average hit time
	if ec.statistics.AverageHitTime == 0 {
		ec.statistics.AverageHitTime = hitTime
	} else {
		ec.statistics.AverageHitTime = (ec.statistics.AverageHitTime + hitTime) / 2
	}
}

func (ec *EmbeddingCache) recordL1Miss() {
	ec.statistics.mu.Lock()
	defer ec.statistics.mu.Unlock()
	ec.statistics.L1Misses++
}

func (ec *EmbeddingCache) recordL2Hit(start time.Time) {
	ec.statistics.mu.Lock()
	defer ec.statistics.mu.Unlock()

	ec.statistics.L2Hits++
	hitTime := time.Since(start)

	// Update average hit time
	if ec.statistics.AverageHitTime == 0 {
		ec.statistics.AverageHitTime = hitTime
	} else {
		ec.statistics.AverageHitTime = (ec.statistics.AverageHitTime + hitTime) / 2
	}
}

func (ec *EmbeddingCache) recordL2Miss() {
	ec.statistics.mu.Lock()
	defer ec.statistics.mu.Unlock()
	ec.statistics.L2Misses++
}

func (ec *EmbeddingCache) startCleanupRoutine() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ec.cleanup()
	}
}

func (ec *EmbeddingCache) cleanup() {
	now := time.Now()

	// Clean L1 cache
	ec.l1Cache.Range(func(key, value interface{}) bool {
		entry := value.(*EmbeddingCacheEntry)
		if now.After(entry.Expiration) {
			ec.l1Cache.Delete(key)
			ec.mu.Lock()
			ec.currentSize -= entry.Size
			ec.mu.Unlock()
		}
		return true
	})

	// Clean L2 cache
	ec.l2Cache.cleanup()

	// Update statistics
	ec.statistics.mu.Lock()
	ec.statistics.LastCleanup = now
	ec.statistics.mu.Unlock()
}

// Helper types and functions

type cacheEntryInfo struct {
	key   string
	entry *EmbeddingCacheEntry
	score float64
}

func containsCodePattern(text string) bool {
	codePatterns := []string{"func ", "function ", "class ", "def ", "import ", "var ", "let ", "const "}
	for _, pattern := range codePatterns {
		if contains(text, pattern) {
			return true
		}
	}
	return false
}

func containsDocumentation(text string) bool {
	docPatterns := []string{"TODO:", "FIXME:", "NOTE:", "/// ", "*", "# "}
	for _, pattern := range docPatterns {
		if contains(text, pattern) {
			return true
		}
	}
	return false
}

func contains(text, pattern string) bool {
	return len(text) >= len(pattern) && text[:len(pattern)] == pattern
}

// PersistentEmbeddingCache methods

func (pec *PersistentEmbeddingCache) Get(cacheKey string) (*EmbeddingCacheEntry, bool) {
	pec.mu.RLock()
	defer pec.mu.RUnlock()

	if indexEntry, found := pec.index[cacheKey]; found {
		filePath := filepath.Join(pec.basePath, indexEntry.FileName)

		data, err := os.ReadFile(filePath)
		if err != nil {
			// Remove corrupted index entry
			delete(pec.index, cacheKey)
			return nil, false
		}

		var entry EmbeddingCacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			os.Remove(filePath)
			delete(pec.index, cacheKey)
			return nil, false
		}

		// Check if entry is still valid
		if time.Now().After(entry.Expiration) {
			os.Remove(filePath)
			delete(pec.index, cacheKey)
			return nil, false
		}

		// Update access information
		entry.LastAccess = time.Now()
		entry.AccessCount++
		indexEntry.LastAccess = time.Now()
		indexEntry.AccessCount++

		// Save updated index
		pec.saveIndex()

		return &entry, true
	}

	return nil, false
}

func (pec *PersistentEmbeddingCache) Put(cacheKey string, entry *EmbeddingCacheEntry) {
	pec.mu.Lock()
	defer pec.mu.Unlock()

	// Check if we need to clean up
	pec.cleanupIfNeeded()

	// Generate filename
	filename := fmt.Sprintf("%s_%d.embedding", cacheKey, time.Now().Unix())
	filePath := filepath.Join(pec.basePath, filename)

	// Save embedding to disk
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return
	}

	// Update index
	pec.index[cacheKey] = &EmbeddingIndexEntry{
		FileName:    filename,
		Size:        entry.Size,
		CreatedAt:   time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 1,
		ModelName:   entry.ModelName,
		TextHash:    entry.TextHash,
	}

	pec.saveIndex()
}

func (pec *PersistentEmbeddingCache) Clear() {
	pec.mu.Lock()
	defer pec.mu.Unlock()

	// Remove all files
	for _, indexEntry := range pec.index {
		filePath := filepath.Join(pec.basePath, indexEntry.FileName)
		os.Remove(filePath)
	}

	// Clear index
	pec.index = make(map[string]*EmbeddingIndexEntry)
	pec.saveIndex()
}

func (pec *PersistentEmbeddingCache) getCurrentSize() int64 {
	pec.mu.RLock()
	defer pec.mu.RUnlock()

	var totalSize int64
	for _, indexEntry := range pec.index {
		totalSize += indexEntry.Size
	}
	return totalSize
}

func (pec *PersistentEmbeddingCache) loadIndex() {
	data, err := os.ReadFile(pec.indexFile)
	if err != nil {
		return // Index doesn't exist yet
	}

	var index map[string]*EmbeddingIndexEntry
	if err := json.Unmarshal(data, &index); err != nil {
		return // Corrupted index, start fresh
	}

	pec.index = index
}

func (pec *PersistentEmbeddingCache) saveIndex() {
	data, err := json.MarshalIndent(pec.index, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(pec.indexFile, data, 0644)
}

func (pec *PersistentEmbeddingCache) cleanupIfNeeded() {
	currentSize := pec.getCurrentSize()
	if currentSize <= pec.maxSize {
		return
	}

	// Sort entries by access time (oldest first)
	var entries []*EmbeddingIndexEntry
	for _, entry := range pec.index {
		entries = append(entries, entry)
	}

	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].LastAccess.After(entries[j].LastAccess) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove oldest entries until we're under the limit
	targetSize := pec.maxSize * 3 / 4 // Leave 25% headroom
	removedSize := int64(0)

	for _, entry := range entries {
		if currentSize-removedSize <= targetSize {
			break
		}

		filePath := filepath.Join(pec.basePath, entry.FileName)
		os.Remove(filePath)

		// Remove from index
		for key, indexEntry := range pec.index {
			if indexEntry.FileName == entry.FileName {
				delete(pec.index, key)
				break
			}
		}

		removedSize += entry.Size
	}

	pec.saveIndex()
}

func (pec *PersistentEmbeddingCache) cleanup() {
	pec.mu.Lock()
	defer pec.mu.Unlock()

	now := time.Now()
	var toDelete []string

	// Find expired entries
	for key, indexEntry := range pec.index {
		filePath := filepath.Join(pec.basePath, indexEntry.FileName)

		data, err := os.ReadFile(filePath)
		if err != nil {
			toDelete = append(toDelete, key)
			continue
		}

		var entry EmbeddingCacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			toDelete = append(toDelete, key)
			os.Remove(filePath)
			continue
		}

		if now.After(entry.Expiration) {
			toDelete = append(toDelete, key)
			os.Remove(filePath)
		}
	}

	// Remove expired entries from index
	for _, key := range toDelete {
		delete(pec.index, key)
	}

	if len(toDelete) > 0 {
		pec.saveIndex()
	}
}