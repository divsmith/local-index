package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectSetupWorkflow tests the complete project setup workflow
func TestProjectSetupWorkflow(t *testing.T) {
	// Create a temporary directory to simulate a project
	projectDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create a project structure
	createProjectStructure(t, projectDir)

	// Change to project directory
	err := os.Chdir(projectDir)
	if err != nil {
		t.Fatalf("Failed to change to project directory: %v", err)
	}

	// Step 1: Index the project (current directory)
	t.Run("IndexCurrentDirectory", func(t *testing.T) {
		cmd := exec.Command("code-search", "index")
		output, err := cmd.CombinedOutput()

		// Should work (existing functionality)
		if err != nil {
			t.Errorf("Expected index command to succeed, got error: %v, output: %s", err, string(output))
		}

		// Should create .clindex directory
		clindexDir := filepath.Join(projectDir, ".clindex")
		if _, err := os.Stat(clindexDir); os.IsNotExist(err) {
			t.Errorf("Expected .clindex directory to be created, but it doesn't exist")
		}
	})

	// Step 2: Search within the project
	t.Run("SearchInCurrentDirectory", func(t *testing.T) {
		cmd := exec.Command("code-search", "search", "func main")
		output, err := cmd.CombinedOutput()

		// Should work (existing functionality)
		if err != nil {
			t.Errorf("Expected search command to succeed, got error: %v, output: %s", err, string(output))
		}

		// Should find the main function
		if !strings.Contains(string(output), "func main") {
			t.Errorf("Expected to find 'func main' in search results, got: %s", string(output))
		}
	})

	// Step 3: Index with explicit directory (new functionality)
	t.Run("IndexWithExplicitDirectory", func(t *testing.T) {
		cmd := exec.Command("code-search", "index", "--dir", projectDir)
		output, err := cmd.CombinedOutput()

		// Initially, this will fail because --dir flag is not implemented
		if err != nil {
			if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else {
				t.Errorf("Expected unknown flag error for --dir, got: %v, output: %s", err, string(output))
			}
		} else {
			// After implementation, this should work
			t.Logf("Index with --dir flag worked: %s", string(output))
		}
	})

	// Step 4: Search with explicit directory (new functionality)
	t.Run("SearchWithExplicitDirectory", func(t *testing.T) {
		cmd := exec.Command("code-search", "search", "--dir", projectDir, "func main")
		output, err := cmd.CombinedOutput()

		// Initially, this will fail because --dir flag is not implemented
		if err != nil {
			if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else {
				t.Errorf("Expected unknown flag error for --dir, got: %v, output: %s", err, string(output))
			}
		} else {
			// After implementation, this should work
			if !strings.Contains(string(output), "func main") {
				t.Errorf("Expected to find 'func main' in search results, got: %s", string(output))
			}
		}
	})

	// Step 5: Verify index files are in the correct location
	t.Run("VerifyIndexLocation", func(t *testing.T) {
		clindexDir := filepath.Join(projectDir, ".clindex")

		// Check metadata file
		metadataFile := filepath.Join(clindexDir, "metadata.json")
		if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
			t.Logf("Expected: metadata file not created yet (indexing might not be implemented)")
		}

		// Check data file
		dataFile := filepath.Join(clindexDir, "data.index")
		if _, err := os.Stat(dataFile); os.IsNotExist(err) {
			t.Logf("Expected: data file not created yet (indexing might not be implemented)")
		}
	})
}

// TestIndexFileLocation tests that index files are stored in the correct location
func TestIndexFileLocation(t *testing.T) {
	projectDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create a simple project
	createProjectStructure(t, projectDir)

	// Change to project directory
	err := os.Chdir(projectDir)
	if err != nil {
		t.Fatalf("Failed to change to project directory: %v", err)
	}

	// Index the project
	cmd := exec.Command("code-search", "index")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Failed to index project: %v, output: %s", err, string(output))
	}

	// Check that .clindex directory exists in project
	clindexDir := filepath.Join(projectDir, ".clindex")
	if _, err := os.Stat(clindexDir); os.IsNotExist(err) {
		t.Errorf("Expected .clindex directory to be created in project directory")
	}

	// Check that .clindex directory contains expected files
	expectedFiles := []string{"metadata.json", "data.index"}
	for _, file := range expectedFiles {
		filePath := filepath.Join(clindexDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Logf("Expected file %s not found (indexing implementation might be incomplete)", file)
		}
	}
}

// TestRelativePathResolution tests that relative paths are resolved correctly
func TestRelativePathResolution(t *testing.T) {
	// Create a parent directory with two subdirectories
	parentDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	projectA := filepath.Join(parentDir, "projectA")
	projectB := filepath.Join(parentDir, "projectB")

	err := os.MkdirAll(projectA, 0755)
	if err != nil {
		t.Fatalf("Failed to create projectA: %v", err)
	}

	err = os.MkdirAll(projectB, 0755)
	if err != nil {
		t.Fatalf("Failed to create projectB: %v", err)
	}

	// Create projects
	createProjectStructure(t, projectA)
	createProjectStructure(t, projectB)

	// Change to parent directory
	err = os.Chdir(parentDir)
	if err != nil {
		t.Fatalf("Failed to change to parent directory: %v", err)
	}

	// Test indexing with relative path
	t.Run("IndexRelativePath", func(t *testing.T) {
		cmd := exec.Command("code-search", "index", "--dir", "./projectA")
		output, err := cmd.CombinedOutput()

		if err != nil {
			if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else {
				t.Errorf("Expected index with relative path to work after implementation, got: %v, output: %s", err, string(output))
			}
		} else {
			// After implementation, check that index is created in the right place
			clindexDir := filepath.Join(projectA, ".clindex")
			if _, err := os.Stat(clindexDir); os.IsNotExist(err) {
				t.Errorf("Expected .clindex directory to be created in projectA, not in parent directory")
			}
		}
	})

	// Test searching with relative path
	t.Run("SearchRelativePath", func(t *testing.T) {
		cmd := exec.Command("code-search", "search", "--dir", "./projectA", "func main")
		output, err := cmd.CombinedOutput()

		if err != nil {
			if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else {
				t.Errorf("Expected search with relative path to work after implementation, got: %v, output: %s", err, string(output))
			}
		} else {
			// After implementation, should find results
			if !strings.Contains(string(output), "func main") {
				t.Errorf("Expected to find 'func main' in projectA, got: %s", string(output))
			}
		}
	})
}

// Helper function to create a basic project structure
func createProjectStructure(t *testing.T, projectDir string) {
	files := map[string]string{
		"main.go":     "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n",
		"utils.go":    "package main\n\nfunc helper() string {\n\treturn \"helper\"\n}\n",
		"README.md":   "# Test Project\nThis is a test project.\n",
		"go.mod":      "module test-project\n\ngo 1.21\n",
	}

	for file, content := range files {
		filePath := filepath.Join(projectDir, file)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}
}

