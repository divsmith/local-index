package main

import (
	"fmt"
	"os"
	"code-search/src/lib"
	"code-search/src/models"
)

func main() {
	// Use the existing index path
	indexPath := "/home/claude/.code-search/indexes/e9671acd244849c57167c658fa2f9697.db"

	// Check if index exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		fmt.Printf("ERROR: Index file does not exist: %s\n", indexPath)
		os.Exit(1)
	}

	// Try to load the index directly
	fmt.Printf("Loading index from: %s\n", indexPath)

	// Create a simple vector store
	vectorStore := lib.NewInMemoryVectorStore("")

	// Load the code index
	index, err := models.LoadCodeIndex(indexPath, vectorStore)
	if err != nil {
		fmt.Printf("ERROR: Failed to load index: %v\n", err)
		os.Exit(1)
	}
	defer index.Close()

	// Get index statistics
	stats := index.GetStats()
	fmt.Printf("Index stats: %+v\n", stats)

	// Get all files
	files := index.GetAllFiles()
	fmt.Printf("Total files in index: %d\n", len(files))

	// Show first few files
	if len(files) > 0 {
		fmt.Printf("First 5 files:\n")
		for i, file := range files {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d: %s (%s)\n", i+1, file.FilePath, file.Language)
		}
	}

	// Try to get some sample content
	if len(files) > 0 {
		file := files[0]
		fmt.Printf("\nSample file content for: %s\n", file.FilePath)
		content, err := file.GetContent()
		if err != nil {
			fmt.Printf("ERROR: Failed to get content: %v\n", err)
		} else {
			fmt.Printf("Content length: %d characters\n", len(content))
			if len(content) > 200 {
				fmt.Printf("First 200 chars:\n%s\n", content[:200])
			} else {
				fmt.Printf("Full content:\n%s\n", content)
			}
		}
	}
}