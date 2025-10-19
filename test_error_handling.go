package main

import (
	"fmt"
	"strings"
)

func main() {
	// Test the exact error handling logic from search_cmd.go
	fmt.Println("=== Testing Error Handling ===")

	// Simulate the exact error message from loadIndex
	err := fmt.Errorf("failed to load index: index file does not exist: /home/claude/.code-search/indexes/240808bc7296ea4cbc4fa9ef209b14b8.db")
	
	fmt.Printf("Error: %s\n", err.Error())
	
	// Check if this is an index not found error (exact logic from search_cmd.go)
	if strings.Contains(err.Error(), "index not found") || strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "not exist") {
		fmt.Printf("✅ Detected as NotFound error\n")
	} else {
		fmt.Printf("❌ Not detected as NotFound error\n")
	}
}
