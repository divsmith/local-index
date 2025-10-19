package main

import (
	"fmt"
	"os"

	"code-search/src/lib"
	"code-search/src/models"
	"code-search/src/services"
)

func main() {
	fmt.Println("=== Testing Enhanced Search Service ===")

	// Create base search service
	silentLogger := &services.SilentLogger{}
	codeParser := lib.NewSimpleCodeParser()
	vectorStore := lib.NewInMemoryVectorStore("")
	searchOptions := services.DefaultSearchOptions()

	baseSearchService := services.NewSearchService(codeParser, vectorStore, silentLogger, searchOptions)

	// Create embedding config
	embeddingConfig := lib.EmbeddingConfig{
		ModelName:        "all-mpnet-base-v2",
		MaxBatchSize:     32,
		CacheSize:        100,
		MemoryLimit:      100,
		SemanticWeight:   0.7,
		TextWeight:       0.3,
	}

	fmt.Printf("Creating enhanced search service...\n")
	
	// Create enhanced search service
	enhancedService, err := services.NewEnhancedSearchServiceWithConfig(baseSearchService, embeddingConfig)
	if err != nil {
		fmt.Printf("Error creating enhanced service: %v\n", err)
		return
	}

	fmt.Printf("Enhanced service created successfully\n")

	// Get project index path
	storageManager := lib.NewStorageManager()
	projectDetector := lib.NewProjectDetector()

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

	// Check if index exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		fmt.Printf("Index file does not exist: %s\n", indexPath)
		fmt.Printf("Testing search with missing index...\n")
	} else {
		fmt.Printf("Index file exists, proceeding with search test...\n")
	}

	// Create test query
	queryText := "database"
	query := models.NewSearchQuery(queryText)
	query.MaxResults = 2
	query.SearchType = models.SearchTypeSemantic

	fmt.Printf("Query: %s\n", query.QueryText)
	fmt.Printf("Search Type: %v\n", query.SearchType)

	// Perform search with enhanced service
	results, err := enhancedService.Search(query, indexPath)
	if err != nil {
		fmt.Printf("Enhanced service search error: %v\n", err)
		return
	}

	if results == nil {
		fmt.Printf("Enhanced service returned nil results\n")
		return
	}

	fmt.Printf("Enhanced service search completed:\n")
	fmt.Printf("- Total Results: %d\n", results.TotalResults)
	fmt.Printf("- Execution Time: %s\n", results.ExecutionTime.String())
}
