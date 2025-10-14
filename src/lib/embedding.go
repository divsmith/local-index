package lib

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// EmbeddingService defines the interface for text embedding services
type EmbeddingService interface {
	// Embed converts text into a vector representation
	Embed(text string) ([]float32, error)

	// Dimensions returns the size of the embedding vectors
	Dimensions() int

	// ModelName returns the name of the embedding model
	ModelName() string

	// Close releases any resources used by the service
	Close() error
}

// EmbeddingConfig holds configuration for embedding services
type EmbeddingConfig struct {
	ModelName        string  `json:"model_name"`
	MaxBatchSize     int     `json:"max_batch_size"`
	CacheSize        int     `json:"cache_size"`
	MemoryLimit      int64   `json:"memory_limit_mb"`
	SemanticWeight   float64 `json:"semantic_weight"`
	TextWeight       float64 `json:"text_weight"`
}

// DefaultEmbeddingConfig returns a sensible default configuration
func DefaultEmbeddingConfig() EmbeddingConfig {
	return EmbeddingConfig{
		ModelName:        "all-MiniLM-L6-v2",
		MaxBatchSize:     32,
		CacheSize:        1000,
		MemoryLimit:      200, // MB
		SemanticWeight:   0.7,
		TextWeight:       0.3,
	}
}

// MockEmbeddingService provides a simple implementation for testing
// This will be replaced with actual ONNX-based implementation
type MockEmbeddingService struct {
	config EmbeddingConfig
	cache  *EmbeddingCache
	closed bool
	mu     sync.RWMutex
}

// NewMockEmbeddingService creates a new mock embedding service
func NewMockEmbeddingService(config EmbeddingConfig) *MockEmbeddingService {
	cache := NewEmbeddingCache(
		config.CacheSize,           // L1 cache size
		config.MemoryLimit*1024*1024, // L2 cache size (MB to bytes)
		24*time.Hour,               // 24 hour TTL
		config.MemoryLimit*1024*1024, // Memory limit (MB to bytes)
	)

	return &MockEmbeddingService{
		config: config,
		cache:  cache,
		closed: false,
	}
}

// Embed generates a simple hash-based embedding (mock implementation)
func (m *MockEmbeddingService) Embed(text string) ([]float32, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, fmt.Errorf("embedding service is closed")
	}

	// Check cache first
	if cached, ok := m.cache.Get(text, m.ModelName()); ok {
		return cached, nil
	}

	// Simple mock embedding based on text hash
	// This will be replaced with actual ONNX model inference
	embedding := make([]float32, 384) // MiniLM dimensions

	// Generate pseudo-random but deterministic embedding based on text
	hash := 5381
	for _, c := range text {
		hash = ((hash << 5) + hash) + int(c)
	}

	// Use hash to generate embedding values
	for i := range embedding {
		hash = ((hash << 5) + hash) + i
		embedding[i] = float32((hash%2000-1000)) / 1000.0 // Normalize to [-1, 1]
	}

	// Cache the result
	m.cache.Put(text, m.ModelName(), embedding)

	return embedding, nil
}

// Dimensions returns the embedding dimensions
func (m *MockEmbeddingService) Dimensions() int {
	return 384 // MiniLM produces 384-dimensional vectors
}

// ModelName returns the model name
func (m *MockEmbeddingService) ModelName() string {
	return m.config.ModelName
}

// Close releases resources
func (m *MockEmbeddingService) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	if m.cache != nil {
		m.cache.Clear()
	}

	m.closed = true
	return nil
}

// EmbeddingServiceFactory creates embedding services
type EmbeddingServiceFactory struct{}

// NewEmbeddingServiceFactory creates a new factory
func NewEmbeddingServiceFactory() *EmbeddingServiceFactory {
	return &EmbeddingServiceFactory{}
}

// CreateService creates an embedding service based on configuration
func (f *EmbeddingServiceFactory) CreateService(config EmbeddingConfig) (EmbeddingService, error) {
	// For now, return mock service
	// This will be extended to support different model types
	switch config.ModelName {
	case "all-MiniLM-L6-v2", "minilm", "":
		return NewMockEmbeddingService(config), nil
	default:
		return nil, fmt.Errorf("unsupported model: %s", config.ModelName)
	}
}

// ValidateEmbeddingCompatibility checks if two embeddings are compatible
func ValidateEmbeddingCompatibility(embedding1, embedding2 []float32) bool {
	if len(embedding1) != len(embedding2) {
		return false
	}
	return true
}

// CosineSimilarity calculates cosine similarity between two vectors
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}