package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ModelMetadata stores information about the embedding model used for an index
type ModelMetadata struct {
	ModelName      string    `json:"model_name"`
	ModelVersion   string    `json:"model_version"`
	VectorDim      int       `json:"vector_dim"`
	ModelType      string    `json:"model_type"`      // "semantic", "text", "hybrid"
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	EmbeddingModel string    `json:"embedding_model"` // Hash or identifier
	Configuration  EmbeddingConfig `json:"configuration"`
}

// IndexMetadata combines model metadata with general index information
type IndexMetadata struct {
	ModelMetadata                `json:"model"`
	IndexVersion   string      `json:"index_version"`
	FileCount      int         `json:"file_count"`
	ChunkCount     int         `json:"chunk_count"`
	IndexedSize    int64       `json:"indexed_size_bytes"`
	IndexType      string      `json:"index_type"` // "hnsw", "brute-force", etc.
	BuildTime      time.Time   `json:"build_time"`
	BuildDuration  string      `json:"build_duration"`
}

// NewModelMetadata creates default model metadata
func NewModelMetadata(modelName string, vectorDim int) ModelMetadata {
	return ModelMetadata{
		ModelName:      modelName,
		ModelVersion:   "1.0.0",
		VectorDim:      vectorDim,
		ModelType:      "semantic",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		EmbeddingModel: generateModelHash(modelName),
		Configuration:  DefaultEmbeddingConfig(),
	}
}

// NewIndexMetadata creates new index metadata
func NewIndexMetadata(modelMetadata ModelMetadata) IndexMetadata {
	return IndexMetadata{
		ModelMetadata: modelMetadata,
		IndexVersion:  "1.0.0",
		IndexType:     "hnsw",
		BuildTime:     time.Now(),
	}
}

// SaveMetadata saves metadata to a file
func (m *IndexMetadata) SaveMetadata(indexPath string) error {
	metadataPath := filepath.Join(indexPath, "metadata.json")

	// Ensure directory exists
	if err := os.MkdirAll(indexPath, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// LoadMetadata loads metadata from a file
func LoadMetadata(indexPath string) (*IndexMetadata, error) {
	// Try to load metadata from various possible locations
	var metadataPath string

	// Check if indexPath is a directory
	if info, err := os.Stat(indexPath); err == nil && info.IsDir() {
		metadataPath = filepath.Join(indexPath, "metadata.json")
	} else {
		// For centralized storage, look for metadata next to the index file
		indexDir := filepath.Dir(indexPath)
		metadataPath = filepath.Join(indexDir, "metadata.json")
	}

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file at %s: %w", metadataPath, err)
	}

	var metadata IndexMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// ValidateCompatibility checks if this model is compatible with another
func (m *ModelMetadata) ValidateCompatible(otherModel EmbeddingService) error {
	if m.ModelName != otherModel.ModelName() {
		return fmt.Errorf("model mismatch: index created with %s, but using %s",
			m.ModelName, otherModel.ModelName())
	}

	if m.VectorDim != otherModel.Dimensions() {
		return fmt.Errorf("dimension mismatch: index created with %dD, but model produces %dD",
			m.VectorDim, otherModel.Dimensions())
	}

	return nil
}

// UpdateMetadata updates the metadata with current information
func (m *IndexMetadata) UpdateMetadata(fileCount, chunkCount int, indexedSize int64, duration time.Duration) {
	m.FileCount = fileCount
	m.ChunkCount = chunkCount
	m.IndexedSize = indexedSize
	m.UpdatedAt = time.Now()
	m.BuildDuration = duration.String()
}

// IsExpired checks if the index is older than the specified duration
func (m *IndexMetadata) IsExpired(maxAge time.Duration) bool {
	return time.Since(m.BuildTime) > maxAge
}

// GetSummary returns a human-readable summary of the metadata
func (m *IndexMetadata) GetSummary() string {
	return fmt.Sprintf("Index: %s (%s) - %d files, %d chunks, %s, built in %s",
		m.ModelName, m.IndexVersion, m.FileCount, m.ChunkCount,
		formatIndexBytes(m.IndexedSize), m.BuildDuration)
}

// generateModelHash creates a deterministic hash for a model
func generateModelHash(modelName string) string {
	// Simple hash function - in a real implementation, you might use
	// a proper cryptographic hash of the model file
	hash := 5381
	for _, c := range modelName {
		hash = ((hash << 5) + hash) + int(c)
	}
	return fmt.Sprintf("model_%08x", hash%0xffffffff)
}

// formatIndexBytes formats bytes into human readable string for index metadata
func formatIndexBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ModelCompatibilityError represents a model compatibility issue
type ModelCompatibilityError struct {
	IndexModel   string
	CurrentModel string
	Reason       string
}

func (e *ModelCompatibilityError) Error() string {
	return fmt.Sprintf("model compatibility error: index created with %s, but using %s - %s",
		e.IndexModel, e.CurrentModel, e.Reason)
}

// NewModelCompatibilityError creates a new compatibility error
func NewModelCompatibilityError(indexModel, currentModel, reason string) *ModelCompatibilityError {
	return &ModelCompatibilityError{
		IndexModel:   indexModel,
		CurrentModel: currentModel,
		Reason:       reason,
	}
}