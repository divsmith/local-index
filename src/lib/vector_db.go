package lib

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"code-search/src/models"
)

// InMemoryVectorStore is an in-memory implementation of VectorStore
// This is a simplified implementation for demonstration purposes
type InMemoryVectorStore struct {
	vectors       map[string]*VectorEntry
	mu            sync.RWMutex
	path          string    // For persistence
	batchBuffer   []*VectorEntry // Buffer for batch operations
	batchSize     int
	batchMutex    sync.Mutex
	transactionID int64
	transactions  map[int64]*Transaction
	transMutex    sync.RWMutex
	poolManager   *PoolManager
	vectorPool    *VectorPool
}

// VectorEntry represents a vector entry with metadata
type VectorEntry struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector"`
	Metadata map[string]interface{} `json:"metadata"`
	Created  time.Time              `json:"created"`
}

// BatchOperation represents a batch operation
type BatchOperation struct {
	Type      string      `json:"type"` // "insert", "update", "delete"
	ID        string      `json:"id"`
	Vector    []float64   `json:"vector,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Transaction represents a transaction for atomic operations
type Transaction struct {
	ID         int64           `json:"id"`
	Operations []BatchOperation `json:"operations"`
	Status     string          `json:"status"` // "active", "committed", "rolled_back"
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	snapshot   map[string]*VectorEntry // Snapshot for rollback
	store      *InMemoryVectorStore
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	Errors       []string `json:"errors"`
	Duration     time.Duration `json:"duration"`
}

// NewInMemoryVectorStore creates a new in-memory vector store
func NewInMemoryVectorStore(indexPath string) *InMemoryVectorStore {
	// Initialize pool manager
	poolManager := GetPoolManager()

	store := &InMemoryVectorStore{
		vectors:      make(map[string]*VectorEntry),
		path:         indexPath,
		batchBuffer:  make([]*VectorEntry, 0),
		batchSize:    100,
		transactions: make(map[int64]*Transaction),
		poolManager:  poolManager,
		vectorPool:   poolManager.GetVectorPool(),
	}

	// Try to load existing data
	if indexPath != "" {
		store.loadFromFile()
	}

	return store
}

// Insert inserts a vector with metadata
func (s *InMemoryVectorStore) Insert(id string, vector []float64, metadata map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(vector) == 0 {
		return fmt.Errorf("vector cannot be empty")
	}

	// Use vector pool to allocate storage
	pooledVector := s.vectorPool.GetVector(len(vector))
	defer s.vectorPool.PutVector(pooledVector) // Return to pool when done

	entry := &VectorEntry{
		ID:       id,
		Vector:   make([]float64, len(vector)), // Still allocate for persistent storage
		Metadata: make(map[string]interface{}),
		Created:  time.Now(),
	}

	// Copy vector and metadata
	copy(entry.Vector, vector)
	for k, v := range metadata {
		entry.Metadata[k] = v
	}

	s.vectors[id] = entry

	// Persist to file if path is set
	if s.path != "" {
		return s.saveToFile()
	}

	return nil
}

// BatchInsert inserts multiple vectors atomically
func (s *InMemoryVectorStore) BatchInsert(entries []VectorEntry) (*BatchResult, error) {
	start := time.Now()
	result := &BatchResult{
		Errors: make([]string, 0),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Pre-allocate temporary vectors from pool for processing
	tempVectors := make([][]float64, 0, len(entries))
	defer func() {
		// Return all temporary vectors to pool
		for _, vec := range tempVectors {
			s.vectorPool.PutVector(vec)
		}
	}()

	for _, entry := range entries {
		if len(entry.Vector) == 0 {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("vector cannot be empty for ID: %s", entry.ID))
			continue
		}

		// Get temporary vector from pool for processing
		tempVec := s.vectorPool.GetVector(len(entry.Vector))
		tempVectors = append(tempVectors, tempVec)
		copy(tempVec, entry.Vector)

		// Create a copy of the entry
		newEntry := &VectorEntry{
			ID:       entry.ID,
			Vector:   make([]float64, len(entry.Vector)), // Persistent storage
			Metadata: make(map[string]interface{}),
			Created:  time.Now(),
		}

		copy(newEntry.Vector, entry.Vector)
		for k, v := range entry.Metadata {
			newEntry.Metadata[k] = v
		}

		s.vectors[entry.ID] = newEntry
		result.SuccessCount++
	}

	// Persist to file if path is set
	if s.path != "" {
		if err := s.saveToFile(); err != nil {
			return result, fmt.Errorf("failed to persist batch insert: %w", err)
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// BeginTransaction starts a new transaction
func (s *InMemoryVectorStore) BeginTransaction() *Transaction {
	transID := atomic.AddInt64(&s.transactionID, 1)

	// Create snapshot of current state
	s.mu.RLock()
	snapshot := make(map[string]*VectorEntry)
	for id, entry := range s.vectors {
		entryCopy := &VectorEntry{
			ID:       entry.ID,
			Vector:   make([]float64, len(entry.Vector)),
			Metadata: make(map[string]interface{}),
			Created:  entry.Created,
		}
		copy(entryCopy.Vector, entry.Vector)
		for k, v := range entry.Metadata {
			entryCopy.Metadata[k] = v
		}
		snapshot[id] = entryCopy
	}
	s.mu.RUnlock()

	trans := &Transaction{
		ID:         transID,
		Operations: make([]BatchOperation, 0),
		Status:     "active",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		snapshot:   snapshot,
		store:      s,
	}

	s.transMutex.Lock()
	s.transactions[transID] = trans
	s.transMutex.Unlock()

	return trans
}

// CommitTransaction commits a transaction
func (s *InMemoryVectorStore) CommitTransaction(transID int64) error {
	s.transMutex.Lock()
	trans, exists := s.transactions[transID]
	if !exists {
		s.transMutex.Unlock()
		return fmt.Errorf("transaction not found: %d", transID)
	}

	if trans.Status != "active" {
		s.transMutex.Unlock()
		return fmt.Errorf("transaction not active: %s", trans.Status)
	}

	trans.Status = "committed"
	trans.UpdatedAt = time.Now()
	s.transMutex.Unlock()

	// Execute all operations atomically
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range trans.Operations {
		switch op.Type {
		case "insert":
			entry := &VectorEntry{
				ID:       op.ID,
				Vector:   make([]float64, len(op.Vector)),
				Metadata: make(map[string]interface{}),
				Created:  op.Timestamp,
			}
			copy(entry.Vector, op.Vector)
			for k, v := range op.Metadata {
				entry.Metadata[k] = v
			}
			s.vectors[op.ID] = entry

		case "update":
			if existing, found := s.vectors[op.ID]; found {
				entryCopy := &VectorEntry{
					ID:       existing.ID,
					Vector:   make([]float64, len(op.Vector)),
					Metadata: make(map[string]interface{}),
					Created:  existing.Created,
				}
				copy(entryCopy.Vector, op.Vector)
				for k, v := range op.Metadata {
					entryCopy.Metadata[k] = v
				}
				s.vectors[op.ID] = entryCopy
			}

		case "delete":
			delete(s.vectors, op.ID)
		}
	}

	// Persist to file if path is set
	if s.path != "" {
		if err := s.saveToFile(); err != nil {
			return fmt.Errorf("failed to persist transaction: %w", err)
		}
	}

	// Clean up transaction
	s.transMutex.Lock()
	delete(s.transactions, transID)
	s.transMutex.Unlock()

	return nil
}

// RollbackTransaction rolls back a transaction
func (s *InMemoryVectorStore) RollbackTransaction(transID int64) error {
	s.transMutex.Lock()
	trans, exists := s.transactions[transID]
	if !exists {
		s.transMutex.Unlock()
		return fmt.Errorf("transaction not found: %d", transID)
	}

	if trans.Status != "active" {
		s.transMutex.Unlock()
		return fmt.Errorf("transaction not active: %s", trans.Status)
	}

	trans.Status = "rolled_back"
	trans.UpdatedAt = time.Now()
	s.transMutex.Unlock()

	// Restore snapshot
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear current vectors and restore snapshot
	s.vectors = make(map[string]*VectorEntry)
	for id, entry := range trans.snapshot {
		entryCopy := &VectorEntry{
			ID:       entry.ID,
			Vector:   make([]float64, len(entry.Vector)),
			Metadata: make(map[string]interface{}),
			Created:  entry.Created,
		}
		copy(entryCopy.Vector, entry.Vector)
		for k, v := range entry.Metadata {
			entryCopy.Metadata[k] = v
		}
		s.vectors[id] = entryCopy
	}

	// Clean up transaction
	s.transMutex.Lock()
	delete(s.transactions, transID)
	s.transMutex.Unlock()

	return nil
}

// Search performs vector similarity search
func (s *InMemoryVectorStore) Search(queryVector []float64, limit int) ([]models.VectorSearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	var results []models.VectorSearchResult

	for id, entry := range s.vectors {
		// Calculate cosine similarity
		similarity := cosineSimilarity(queryVector, entry.Vector)

		result := models.VectorSearchResult{
			ID:       id,
			Score:    similarity,
			Metadata: entry.Metadata,
		}

		results = append(results, result)
	}

	// Sort by similarity (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Delete removes a vector by ID
func (s *InMemoryVectorStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.vectors, id)

	// Persist to file if path is set
	if s.path != "" {
		return s.saveToFile()
	}

	return nil
}

// Close closes the vector store
func (s *InMemoryVectorStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path != "" {
		return s.saveToFile()
	}

	return nil
}

// GetStats returns statistics about the vector store
func (s *InMemoryVectorStore) GetStats() VectorStoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := VectorStoreStats{
		VectorCount: len(s.vectors),
		Dimensions:  0,
		TotalSize:   0,
	}

	// Calculate dimensions and size
	for _, entry := range s.vectors {
		if len(entry.Vector) > stats.Dimensions {
			stats.Dimensions = len(entry.Vector)
		}
		stats.TotalSize += len(entry.Vector) * 8 // 8 bytes per float64
	}

	return stats
}

// GetPoolStats returns memory pool statistics
func (s *InMemoryVectorStore) GetPoolStats() PoolStats {
	if s.vectorPool != nil {
		return s.vectorPool.GetStats()
	}
	return PoolStats{}
}

// CleanupPools performs cleanup on memory pools
func (s *InMemoryVectorStore) CleanupPools() {
	if s.poolManager != nil {
		s.poolManager.Cleanup()
	}
}

// saveToFile saves the vector store to a file
func (s *InMemoryVectorStore) saveToFile() error {
	if s.path == "" {
		return nil
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(s.vectors, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vectors: %w", err)
	}

	// Write to temporary file first
	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, s.path); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// loadFromFile loads the vector store from a file
func (s *InMemoryVectorStore) loadFromFile() error {
	if s.path == "" {
		return nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, that's OK
		}
		return fmt.Errorf("failed to read vector store file: %w", err)
	}

	var vectors map[string]*VectorEntry
	if err := json.Unmarshal(data, &vectors); err != nil {
		return fmt.Errorf("failed to unmarshal vectors: %w", err)
	}

	s.vectors = vectors
	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// VectorStoreStats contains statistics about the vector store
type VectorStoreStats struct {
	VectorCount int    `json:"vector_count"`
	Dimensions  int    `json:"dimensions"`
	TotalSize   int    `json:"total_size"`
	Path        string `json:"path,omitempty"`
}

// Transaction Methods

// Insert adds an insert operation to the transaction
func (t *Transaction) Insert(id string, vector []float64, metadata map[string]interface{}) error {
	if t.Status != "active" {
		return fmt.Errorf("transaction not active: %s", t.Status)
	}

	op := BatchOperation{
		Type:      "insert",
		ID:        id,
		Vector:    make([]float64, len(vector)),
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	copy(op.Vector, vector)
	for k, v := range metadata {
		op.Metadata[k] = v
	}

	t.Operations = append(t.Operations, op)
	t.UpdatedAt = time.Now()

	return nil
}

// Update adds an update operation to the transaction
func (t *Transaction) Update(id string, vector []float64, metadata map[string]interface{}) error {
	if t.Status != "active" {
		return fmt.Errorf("transaction not active: %s", t.Status)
	}

	op := BatchOperation{
		Type:      "update",
		ID:        id,
		Vector:    make([]float64, len(vector)),
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	copy(op.Vector, vector)
	for k, v := range metadata {
		op.Metadata[k] = v
	}

	t.Operations = append(t.Operations, op)
	t.UpdatedAt = time.Now()

	return nil
}

// Delete adds a delete operation to the transaction
func (t *Transaction) Delete(id string) error {
	if t.Status != "active" {
		return fmt.Errorf("transaction not active: %s", t.Status)
	}

	op := BatchOperation{
		Type:      "delete",
		ID:        id,
		Timestamp: time.Now(),
	}

	t.Operations = append(t.Operations, op)
	t.UpdatedAt = time.Now()

	return nil
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	return t.store.CommitTransaction(t.ID)
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	return t.store.RollbackTransaction(t.ID)
}

// GetOperationCount returns the number of operations in the transaction
func (t *Transaction) GetOperationCount() int {
	return len(t.Operations)
}

// GetStatus returns the current status of the transaction
func (t *Transaction) GetStatus() string {
	return t.Status
}

// MockVectorStore is a mock implementation for testing
type MockVectorStore struct {
	vectors map[string]*VectorEntry
}

// NewMockVectorStore creates a new mock vector store
func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		vectors: make(map[string]*VectorEntry),
	}
}

// Insert implements VectorStore interface
func (m *MockVectorStore) Insert(id string, vector []float64, metadata map[string]interface{}) error {
	m.vectors[id] = &VectorEntry{
		ID:       id,
		Vector:   append([]float64{}, vector...),
		Metadata: metadata,
		Created:  time.Now(),
	}
	return nil
}

// Search implements VectorStore interface
func (m *MockVectorStore) Search(queryVector []float64, limit int) ([]models.VectorSearchResult, error) {
	var results []models.VectorSearchResult

	for id, entry := range m.vectors {
		// Simple mock similarity calculation
		similarity := 0.8 // Mock high similarity

		result := models.VectorSearchResult{
			ID:       id,
			Score:    similarity,
			Metadata: entry.Metadata,
		}

		results = append(results, result)
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Delete implements VectorStore interface
func (m *MockVectorStore) Delete(id string) error {
	delete(m.vectors, id)
	return nil
}

// Close implements VectorStore interface
func (m *MockVectorStore) Close() error {
	return nil
}

// AddMockData adds some mock data for testing
func (m *MockVectorStore) AddMockData() {
	mockData := []struct {
		id       string
		vector   []float64
		metadata map[string]interface{}
	}{
		{
			id:     "test1",
			vector: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
			metadata: map[string]interface{}{
				"file_path":  "test.go",
				"start_line": 1.0,
				"end_line":   5.0,
				"content":    "func test() {}",
				"language":   "Go",
			},
		},
		{
			id:     "test2",
			vector: []float64{0.6, 0.7, 0.8, 0.9, 1.0},
			metadata: map[string]interface{}{
				"file_path":  "main.go",
				"start_line": 10.0,
				"end_line":   15.0,
				"content":    "func main() {}",
				"language":   "Go",
			},
		},
	}

	for _, data := range mockData {
		m.Insert(data.id, data.vector, data.metadata)
	}
}
