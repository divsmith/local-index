package main

import (
	"fmt"
	"os"
	
	"code-search/src/lib"
)

func main() {
	storageManager := lib.NewStorageManager()
	projectDetector := lib.NewProjectDetector()
	
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	projectRoot, err := projectDetector.DetectProjectRoot(currentDir)
	if err != nil {
		fmt.Printf("Error detecting project root: %v\n", err)
		return
	}
	
	indexPath := storageManager.GetProjectIndexPath(projectRoot)
	fmt.Printf("Current dir: %s\n", currentDir)
	fmt.Printf("Project root: %s\n", projectRoot)
	fmt.Printf("Index path: %s\n", indexPath)
}
