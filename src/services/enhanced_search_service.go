package services

import (
	"fmt"
	"os"
	"strings"
	"time"

	"code-search/src/lib"
	"code-search/src/models"
)

// EnhancedSearchService wraps the existing SearchService with embedding capabilities
type EnhancedSearchService struct {
	*SearchService                    // Embedded existing service
	embeddingService lib.EmbeddingService // New embedding service
	modelFactory     *lib.EmbeddingServiceFactory
}

// NewEnhancedSearchService creates a new enhanced search service with embedding capabilities
func NewEnhancedSearchService(
	originalService *SearchService,
	embeddingService lib.EmbeddingService,
) *EnhancedSearchService {
	return &EnhancedSearchService{
	SearchService:    originalService,
	embeddingService: embeddingService,
		modelFactory:     lib.NewEmbeddingServiceFactory(),
	}
}

// NewEnhancedSearchServiceWithConfig creates a new enhanced search service with custom embedding config
func NewEnhancedSearchServiceWithConfig(
	originalService *SearchService,
	embeddingConfig lib.EmbeddingConfig,
) (*EnhancedSearchService, error) {
	embeddingService, err := lib.NewEmbeddingServiceFactory().CreateService(embeddingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding service: %w", err)
	}

	return &EnhancedSearchService{
	SearchService:    originalService,
		embeddingService: embeddingService,
		modelFactory:     lib.NewEmbeddingServiceFactory(),
	}, nil
}

// EmbeddingCodeParser implements the CodeParser interface using our embedding service
type EmbeddingCodeParser struct {
	embeddingService lib.EmbeddingService
	originalParser    CodeParser
}

// NewEmbeddingCodeParser creates a new code parser that uses embedding services
func NewEmbeddingCodeParser(embeddingService lib.EmbeddingService, originalParser CodeParser) *EmbeddingCodeParser {
	return &EmbeddingCodeParser{
	embeddingService: embeddingService,
		originalParser:    originalParser,
	}
}

// GetEmbedding generates embeddings for text using the embedding service
func (e *EmbeddingCodeParser) GetEmbedding(text string) ([]float64, error) {
	embedding, err := e.embeddingService.Embed(text)
	if err != nil {
		return nil, err
	}
	// Convert float32 to float64
	result := make([]float64, len(embedding))
	for i, v := range embedding {
		result[i] = float64(v)
	}
	return result, nil
}

// Wrap the original parser methods to maintain compatibility
func (e *EmbeddingCodeParser) ParseFile(filePath string) ([]models.CodeChunk, error) {
	return e.originalParser.ParseFile(filePath)
}

func (e *EmbeddingCodeParser) GetSupportedFileTypes() []string {
	return e.originalParser.GetSupportedFileTypes()
}

// Search implements the SearchServiceInterface with embedding support
func (ess *EnhancedSearchService) Search(
	query *models.SearchQuery,
	indexPath string,
) (*models.SearchResults, error) {
	fmt.Fprintf(os.Stderr, "DEBUG ENHANCED: Search called with query: %s, indexPath: %s\n", query.QueryText, indexPath)

	// Update the search service to use embedding-capable parser
	originalParser := ess.SearchService.codeParser
	ess.SearchService.codeParser = NewEmbeddingCodeParser(ess.embeddingService, ess.SearchService.codeParser)
	defer func() {
		ess.SearchService.codeParser = originalParser // Restore original parser
		fmt.Fprintf(os.Stderr, "DEBUG ENHANCED: Parser restored\n")
	}()

	fmt.Fprintf(os.Stderr, "DEBUG ENHANCED: Calling base search service...\n")
	// Use the underlying search service with embedding capability
	results, err := ess.SearchService.Search(query, indexPath)
	fmt.Fprintf(os.Stderr, "DEBUG ENHANCED: Base search returned - results: %v, err: %v\n", results != nil, err)
	fmt.Fprintf(os.Stderr, "DEBUG ENHANCED: About to return - results: %v, err: %v\n", results != nil, err)

	return results, err
}

// SemanticSearch performs pure semantic search using embeddings
func (ess *EnhancedSearchService) SemanticSearch(
	queryText string,
	indexPath string,
	maxResults int,
) (*models.SearchResults, error) {
	// Create search query
	query := &models.SearchQuery{
		QueryText:     queryText,
		SearchType:    models.SearchTypeSemantic,
		MaxResults:    maxResults,
		Threshold:     0.5, // Default threshold for semantic search
	}

	return ess.Search(query, indexPath)
}

// HybridSearch performs hybrid search combining semantic and text search
func (ess *EnhancedSearchService) HybridSearch(
	queryText string,
	indexPath string,
	maxResults int,
	semanticWeight float64,
	textWeight float64,
) (*models.SearchResults, error) {
	// Create search query
	query := &models.SearchQuery{
		QueryText:     queryText,
		SearchType:    models.SearchTypeHybrid,
		MaxResults:    maxResults,
		Threshold:     0.3, // Lower threshold for hybrid search
	}

	// Add hybrid search configuration
	query.SetOption("semantic_weight", fmt.Sprintf("%.2f", semanticWeight))
	query.SetOption("text_weight", fmt.Sprintf("%.2f", textWeight))

	return ess.Search(query, indexPath)
}

// GetEmbeddingInfo returns information about the current embedding service
func (ess *EnhancedSearchService) GetEmbeddingInfo() EmbeddingInfo {
	return EmbeddingInfo{
		ModelName:   ess.embeddingService.ModelName(),
		Dimensions:  ess.embeddingService.Dimensions(),
		ModelType:   "sentence-transformer",
		IsLoaded:   true, // For now, assume always loaded
		Config:     ess.searchOptions, // Include search options
	}
}

// UpdateEmbeddingService updates the embedding service used for searches
func (ess *EnhancedSearchService) UpdateEmbeddingService(config lib.EmbeddingConfig) error {
	// Close existing service
	if err := ess.embeddingService.Close(); err != nil {
		ess.logger.Warn("Failed to close existing embedding service: %v", err)
	}

	// Create new service
	newService, err := ess.modelFactory.CreateService(config)
	if err != nil {
		return fmt.Errorf("failed to create new embedding service: %w", err)
	}

	ess.embeddingService = newService
	return nil
}

// ValidateIndexWithEmbedding checks if an index is compatible with the current embedding service
func (ess *EnhancedSearchService) ValidateIndexWithEmbedding(indexPath string) (*IndexValidationResult, error) {
	// Try to load metadata
	metadata, err := lib.LoadMetadata(indexPath)
	if err != nil {
		return &IndexValidationResult{
			IsValid:   false,
			Message:   fmt.Sprintf("Failed to load index metadata: %v", err),
			IndexPath: indexPath,
		}, nil
	}

	// Check compatibility
	validationError := metadata.ValidateCompatible(ess.embeddingService)
	if validationError != nil {
		return &IndexValidationResult{
			IsValid:       false,
			Message:       validationError.Error(),
			IndexPath:     indexPath,
			IndexModel:    metadata.ModelName,
			CurrentModel:  ess.embeddingService.ModelName(),
			Recommendation: ess.getReindexationRecommendation(validationError),
		}, nil
	}

	return &IndexValidationResult{
		IsValid:      true,
		Message:      "Index is compatible with current embedding model",
		IndexPath:    indexPath,
		IndexModel:   metadata.ModelName,
		CurrentModel:  ess.embeddingService.ModelName(),
	}, nil
}

// ReindexWithCurrentModel re-indexes a directory with the current embedding model
func (ess *EnhancedSearchService) ReindexWithCurrentModel(
	directoryPath string,
	indexingService IndexingService,
) (*ReindexResult, error) {
	// This would integrate with the indexing service to re-index with the new model
	// For now, return a placeholder result

	start := time.Now()

	// In a real implementation, this would:
	// 1. Validate the directory
	// 2. Call the indexing service to re-index with current model
	// 3. Save new metadata with model info
	// 4. Return results

	return &ReindexResult{
		Success:       true,
		Message:       "Re-indexing with current embedding model",
		Directory:     directoryPath,
		ModelIndex:    ess.embeddingService.ModelName(),
		Duration:      time.Since(start),
		Recommendation: "Test search functionality after re-indexing",
	}, nil
}

// getReindexationRecommendation provides recommendations for fixing model incompatibility
func (ess *EnhancedSearchService) getReindexationRecommendation(err error) string {
	if strings.Contains(err.Error(), "model mismatch") {
		return fmt.Sprintf("Re-index with model: %s", ess.embeddingService.ModelName())
	}
	if strings.Contains(err.Error(), "dimension mismatch") {
		return fmt.Sprintf("Re-index with model that produces %dD vectors", ess.embeddingService.Dimensions())
	}
	return "Re-index with compatible embedding model"
}

// EmbeddingInfo contains information about the current embedding service
type EmbeddingInfo struct {
	ModelName   string           `json:"model_name"`
	Dimensions int              `json:"dimensions"`
	ModelType   string           `json:"model_type"`
	IsLoaded   bool             `json:"is_loaded"`
	Config     SearchOptions    `json:"config"`
}

// IndexValidationResult contains the result of index validation
type IndexValidationResult struct {
	IsValid       bool    `json:"is_valid"`
	Message       string  `json:"message"`
	IndexPath     string  `json:"index_path"`
	IndexModel    string  `json:"index_model"`
	CurrentModel  string  `json:"current_model"`
	Recommendation string  `json:"recommendation,omitempty"`
}

// ReindexResult contains the result of a re-indexing operation
type ReindexResult struct {
	Success       bool          `json:"success"`
	Message       string        `json:"message"`
	Directory     string        `json:"directory"`
	ModelIndex    string        `json:"model_index"`
	Duration      time.Duration `json:"duration"`
	Recommendation string        `json:"recommendation,omitempty"`
}