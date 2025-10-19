package main

import (
	"fmt"
	"os"

	"code-search/src/lib"
	"code-search/src/models"
	"code-search/src/services"
)

func main() {
	// Test search service with detailed logging
	fmt.Println("=== Testing Search Service Debug ===")

	// Create components
	silentLogger := &services.SilentLogger{}
	codeParser := lib.NewSimpleCodeParser()
	vectorStore := lib.NewInMemoryVectorStore("")
	searchOptions := services.DefaultSearchOptions()

	// Create search service
	searchService := services.NewSearchService(codeParser, vectorStore, silentLogger, searchOptions)

	// Create test query
	queryText := "database"
	query := models.NewSearchQuery(queryText)
	query.MaxResults = 2
	query.SearchType = models.SearchTypeHybrid

	fmt.Printf("Query: %s\n", query.QueryText)
	fmt.Printf("Search Type: %v\n", query.SearchType)

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
		return
	}

	fmt.Printf("Index file exists, performing search...\n")

	// Perform search
	results, err := searchService.Search(query, indexPath)
	if err != nil {
		fmt.Printf("Search error: %v\n", err)
		return
	}

	if results == nil {
		fmt.Printf("Search returned nil results\n")
		return
	}

	fmt.Printf("Search completed successfully:\n")
	fmt.Printf("- Total Results: %d\n", results.TotalResults)
	fmt.Printf("- Execution Time: %s\n", results.ExecutionTime.String())
	fmt.Printf("- Results Count: %d\n", len(results.Results))

	for i, result := range results.Results {
		fmt.Printf("Result %d: %s:%d - %s\n", i+1, result.FilePath, result.StartLine, result.Content[:50])
	}
}
