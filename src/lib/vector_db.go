package lib

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"code-search/src/models"
)

// InMemoryVectorStore is an in-memory implementation of VectorStore
// This is a simplified implementation for demonstration purposes
type InMemoryVectorStore struct {
	vectors map[string]*VectorEntry
	mu      sync.RWMutex
	path    string // For persistence
}

// VectorEntry represents a vector entry with metadata
type VectorEntry struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector"`
	Metadata map[string]interface{} `json:"metadata"`
	Created  time.Time              `json:"created"`
}

// NewInMemoryVectorStore creates a new in-memory vector store
func NewInMemoryVectorStore(indexPath string) *InMemoryVectorStore {
	store := &InMemoryVectorStore{
		vectors: make(map[string]*VectorEntry),
		path:    indexPath,
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

	entry := &VectorEntry{
		ID:       id,
		Vector:   make([]float64, len(vector)),
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
