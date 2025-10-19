package main

import (
	"fmt"
	"os"
	"code-search/src/lib"
	"code-search/src/models"
	"code-search/src/services"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run debug_search.go <query>")
		os.Exit(1)
	}

	queryText := os.Args[1]

	// Create a silent logger
	silentLogger := &services.SilentLogger{}

	// Create base search service
	baseSearchService := services.NewSearchService(
		lib.NewSimpleCodeParser(),
		lib.NewInMemoryVectorStore(""),
		silentLogger,
		services.DefaultSearchOptions(),
	)

	// Use the existing index path
	indexPath := "/home/claude/.code-search/indexes/e9671acd244849c57167c658fa2f9697.db"

	fmt.Printf("Searching for: %s\n", queryText)
	fmt.Printf("Index path: %s\n", indexPath)

	// Check if index exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		fmt.Printf("ERROR: Index file does not exist: %s\n", indexPath)
		os.Exit(1)
	}

	// Create search query
	query := models.NewSearchQuery(queryText)
	query.SearchType = models.SearchTypeText // Simple text search
	query.MaxResults = 5

	fmt.Printf("Query: %+v\n", query)

	// Perform search
	fmt.Printf("Performing search...\n")
	results, err := baseSearchService.Search(query, indexPath)
	fmt.Printf("Search results: %v, error: %v\n", results != nil, err)

	if err != nil {
		fmt.Printf("Search error: %v\n", err)
		os.Exit(1)
	}

	if results == nil {
		fmt.Printf("ERROR: Search returned nil results\n")
		os.Exit(1)
	}

	fmt.Printf("Found %d results\n", len(results.Results))
	for i, result := range results.Results {
		fmt.Printf("%d: %s:%d - %s\n", i+1, result.FilePath, result.StartLine, result.Content)
	}
}