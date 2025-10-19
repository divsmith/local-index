package main

import (
	"fmt"
	"code-search/src/lib"
)

func main() {
	fmt.Println("=== Testing Query Analyzer ===")
	
	analyzer := lib.NewQueryAnalyzer()
	queryText := "database"
	
	queryType := analyzer.AnalyzeQuery(queryText)
	
	fmt.Printf("Query: %s\n", queryText)
	fmt.Printf("Query Type: %v\n", queryType)
	
	// Map to search type
	switch queryType {
	case lib.QueryTypeExact:
		fmt.Printf("Search Type: Exact\n")
	case lib.QueryTypeRegex:
		fmt.Printf("Search Type: Text\n")
	case lib.QueryTypeSemantic:
		fmt.Printf("Search Type: Semantic\n")
	case lib.QueryTypeHybrid:
		fmt.Printf("Search Type: Hybrid\n")
	default:
		fmt.Printf("Search Type: Hybrid (default)\n")
	}
}
