package main

import (
	"fmt"
	"os"
	"strings"

	"code-search/src/lib"
	"code-search/src/models"
	"code-search/src/services"
)

// SearchServiceInterface defines the interface for search services
type SearchServiceInterface interface {
	Search(query *models.SearchQuery, indexPath string) (*models.SearchResults, error)
}

// Simplified version of the search command logic
func main() {
	fmt.Println("=== Testing Full Search Flow ===")

	// Create components (same as search command)
	silentLogger := &services.SilentLogger{}
	codeParser := lib.NewSimpleCodeParser()
	vectorStore := lib.NewInMemoryVectorStore("")
	searchOptions := services.DefaultSearchOptions()

	baseSearchService := services.NewSearchService(codeParser, vectorStore, silentLogger, searchOptions)
	storageManager := lib.NewStorageManager()
	projectDetector := lib.NewProjectDetector()

	// Get query and analyze it (same as search command)
	queryText := "database"
	analyzer := lib.NewQueryAnalyzer()
	queryType := analyzer.AnalyzeQuery(queryText)
	
	query := models.NewSearchQuery(queryText)
	query.MaxResults = 2
	query.SearchType = models.SearchTypeHybrid

	fmt.Printf("Query: %s\n", query.QueryText)
	fmt.Printf("Query Type: %v\n", queryType)
	fmt.Printf("Search Type: %v\n", query.SearchType)

	// Create embedding config (same as search command)
	embeddingConfig := lib.EmbeddingConfig{
		ModelName:        "all-mpnet-base-v2",
		MaxBatchSize:     32,
		CacheSize:        100,
		MemoryLimit:      100,
		SemanticWeight:   0.7,
		TextWeight:       0.3,
	}

	// Create enhanced search service if needed (same as search command)
	var searchService SearchServiceInterface = baseSearchService
	if query.SearchType == models.SearchTypeSemantic || query.SearchType == models.SearchTypeHybrid {
		fmt.Printf("Creating enhanced search service...\n")
		enhancedService, err := services.NewEnhancedSearchServiceWithConfig(baseSearchService, embeddingConfig)
		if err != nil {
			fmt.Printf("Error creating enhanced search service: %v\n", err)
			return
		}
		searchService = enhancedService
		fmt.Printf("Enhanced search service created\n")
	}

	// Get project index path (same as search command)
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current dir: %v\n", err)
		return
	}

	projectRoot, err := projectDetector.DetectProjectRoot(currentDir)
	if err != nil {
		fmt.Printf("Error detecting project root: %v\n", err)
		return
	}

	indexPath := storageManager.GetProjectIndexPath(projectRoot)
	fmt.Printf("Index Path: %s\n", indexPath)

	// Perform search (same as search command)
	results, err := searchService.Search(query, indexPath)
	if err != nil {
		// Check if this is an index not found error (same logic as search command)
		if strings.Contains(err.Error(), "index not found") || strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "not exist") {
			fmt.Printf("✅ NotFoundError detected: %v\n", err)
			return
		}
		fmt.Printf("❌ GeneralError: %v\n", err)
		return
	}

	// This should not be reached if index doesn't exist
	if results == nil {
		fmt.Printf("❌ Search returned nil results (this is the bug!)\n")
		return
	}

	fmt.Printf("Search completed successfully: %d results\n", results.TotalResults)
}
