package main

import (
	"fmt"
	"os"
	"strings"
	"code-search/src/lib"
	"code-search/src/models"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run debug_text_search.go <query>")
		os.Exit(1)
	}

	queryText := os.Args[1]

	// Use the existing index path
	indexPath := "/home/claude/.code-search/indexes/e9671acd244849c57167c658fa2f9697.db"

	// Check if index exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		fmt.Printf("ERROR: Index file does not exist: %s\n", indexPath)
		os.Exit(1)
	}

	
	// Load the index directly to debug
	fmt.Printf("Loading index from: %s\n", indexPath)
	vectorStore := lib.NewInMemoryVectorStore("")
	index, err := models.LoadCodeIndex(indexPath, vectorStore)
	if err != nil {
		fmt.Printf("ERROR: Failed to load index: %v\n", err)
		os.Exit(1)
	}
	defer index.Close()

	// Get all files and manually search
	files := index.GetAllFiles()
	fmt.Printf("Searching %d files for: %s\n", len(files), queryText)

	searchTerms := strings.Fields(strings.ToLower(queryText))
	fmt.Printf("Search terms: %v\n", searchTerms)

	foundFiles := 0
	for _, fileEntry := range files {
		fmt.Printf("Checking file: %s\n", fileEntry.FilePath)

		// Read file content
		content, err := fileEntry.GetContent()
		if err != nil {
			fmt.Printf("  Failed to read content: %v\n", err)
			continue
		}

		// Search for terms in content
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lineLower := strings.ToLower(line)

			// Check if all search terms are present in this line
			allTermsFound := true
			for _, term := range searchTerms {
				if !strings.Contains(lineLower, term) {
					allTermsFound = false
					break
				}
			}

			if allTermsFound {
				fmt.Printf("  MATCH: Line %d: %s\n", i+1, strings.TrimSpace(line))
				foundFiles++
				break // Just show one match per file
			}
		}
	}

	fmt.Printf("Found matches in %d files\n", foundFiles)
}