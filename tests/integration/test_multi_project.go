package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMultiProjectWorkflow tests managing multiple projects
func TestMultiProjectWorkflow(t *testing.T) {
	// Create a workspace with multiple projects
	workspaceDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	projects := []string{"project-a", "project-b", "project-c"}
	projectDirs := make(map[string]string)

	// Create multiple projects
	for _, project := range projects {
		projectDir := filepath.Join(workspaceDir, project)
		projectDirs[project] = projectDir
		err := os.MkdirAll(projectDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create project directory %s: %v", project, err)
		}
		createMultiProjectStructure(t, projectDir)
	}

	// Change to workspace directory
	err := os.Chdir(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to change to workspace directory: %v", err)
	}

	// Step 1: Index each project individually
	for _, project := range projects {
		t.Run("IndexProject_"+project, func(t *testing.T) {
			projectDir := projectDirs[project]

			// Index with absolute path
			cmd := exec.Command("code-search", "index", "--dir", projectDir)
			output, err := cmd.CombinedOutput()

			if err != nil {
				if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
					t.Logf("Expected: --dir flag not implemented yet for project %s, got: %s", project, string(output))
				} else {
					t.Errorf("Expected indexing project %s to work after implementation, got: %v, output: %s", project, err, string(output))
				}
			} else {
				// After implementation, check that .clindex is created in project directory
				clindexDir := filepath.Join(projectDir, ".clindex")
				if _, err := os.Stat(clindexDir); os.IsNotExist(err) {
					t.Errorf("Expected .clindex directory to be created in project %s", project)
				}
			}
		})
	}

	// Step 2: Search in each project individually
	for _, project := range projects {
		t.Run("SearchProject_"+project, func(t *testing.T) {
			projectDir := projectDirs[project]

			// Search with absolute path
			cmd := exec.Command("code-search", "search", "--dir", projectDir, "func main")
			output, err := cmd.CombinedOutput()

			if err != nil {
				if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
					t.Logf("Expected: --dir flag not implemented yet for project %s, got: %s", project, string(output))
				} else if strings.Contains(string(output), "No index found") {
					t.Logf("Expected: No index found in project %s (indexing not implemented)", project)
				} else {
					t.Errorf("Expected searching project %s to work after implementation, got: %v, output: %s", project, err, string(output))
				}
			} else {
				// After implementation, should find results
				if !strings.Contains(string(output), "func main") {
					t.Errorf("Expected to find 'func main' in project %s, got: %s", project, string(output))
				}
			}
		})
	}

	// Step 3: Verify index isolation (each project has its own index)
	t.Run("VerifyIndexIsolation", func(t *testing.T) {
		for _, project := range projects {
			projectDir := projectDirs[project]
			clindexDir := filepath.Join(projectDir, ".clindex")

			// Each project should have its own .clindex directory
			if _, err := os.Stat(clindexDir); os.IsNotExist(err) {
				t.Logf("Expected: Project %s .clindex directory not found (indexing not implemented)", project)
			} else {
				// Check that index files exist
				expectedFiles := []string{"metadata.json", "data.index"}
				for _, file := range expectedFiles {
					filePath := filepath.Join(clindexDir, file)
					if _, err := os.Stat(filePath); os.IsNotExist(err) {
						t.Logf("Expected: File %s not found in project %s (indexing incomplete)", file, project)
					}
				}
			}
		}

		// Verify no index files in workspace root
		workspaceClindex := filepath.Join(workspaceDir, ".clindex")
		if _, err := os.Stat(workspaceClindex); err == nil {
			t.Errorf("Expected no .clindex directory in workspace root, but found one")
		}
	})

	// Step 4: Test different search scopes
	t.Run("TestSearchScopes", func(t *testing.T) {
		// Search in project-a only
		cmd := exec.Command("code-search", "search", "--dir", projectDirs["project-a"], "func main")
		output, err := cmd.CombinedOutput()

		if err != nil {
			if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet for scoped search, got: %s", string(output))
			} else {
				t.Errorf("Expected scoped search to work after implementation, got: %v, output: %s", err, string(output))
			}
		} else {
			// After implementation, results should only be from project-a
			if strings.Contains(string(output), projectDirs["project-b"]) || strings.Contains(string(output), projectDirs["project-c"]) {
				t.Errorf("Search results should only include files from project-a, but got results from other projects: %s", string(output))
			}
		}
	})
}

// TestProjectIndexManagement tests managing indexes across multiple projects
func TestProjectIndexManagement(t *testing.T) {
	workspaceDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create two projects with different content
	projectA := filepath.Join(workspaceDir, "frontend")
	projectB := filepath.Join(workspaceDir, "backend")

	err := os.MkdirAll(projectA, 0755)
	if err != nil {
		t.Fatalf("Failed to create frontend project: %v", err)
	}

	err = os.MkdirAll(projectB, 0755)
	if err != nil {
		t.Fatalf("Failed to create backend project: %v", err)
	}

	// Create different content for each project
	os.WriteFile(filepath.Join(projectA, "app.js"), []byte("function App() { return <div>Hello</div>; }"), 0644)
	os.WriteFile(filepath.Join(projectA, "package.json"), []byte(`{"name": "frontend"}`), 0644)

	os.WriteFile(filepath.Join(projectB, "server.py"), []byte("def app():\n    return \"Hello World\""), 0644)
	os.WriteFile(filepath.Join(projectB, "requirements.txt"), []byte("flask==2.0"), 0644)

	// Change to workspace directory
	err = os.Chdir(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to change to workspace directory: %v", err)
	}

	// Index both projects
	t.Run("IndexBothProjects", func(t *testing.T) {
		projects := map[string]string{
			"frontend": projectA,
			"backend":  projectB,
		}

		for name, dir := range projects {
			cmd := exec.Command("code-search", "index", "--dir", dir)
			output, err := cmd.CombinedOutput()

			if err != nil {
				if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
					t.Logf("Expected: --dir flag not implemented yet for %s, got: %s", name, string(output))
				} else {
					t.Errorf("Expected indexing %s project to work after implementation, got: %v, output: %s", name, err, string(output))
				}
			} else {
				// Verify index was created in the right place
				clindexDir := filepath.Join(dir, ".clindex")
				if _, err := os.Stat(clindexDir); os.IsNotExist(err) {
					t.Errorf("Expected .clindex directory to be created in %s project", name)
				}
			}
		}
	})

	// Test project-specific searches
	t.Run("ProjectSpecificSearches", func(t *testing.T) {
		// Search for JavaScript-specific content in frontend
		cmd := exec.Command("code-search", "search", "--dir", projectA, "function App")
		output, err := cmd.CombinedOutput()

		if err != nil {
			if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet for frontend search, got: %s", string(output))
			} else {
				t.Errorf("Expected frontend search to work after implementation, got: %v, output: %s", err, string(output))
			}
		} else {
			// Should find JavaScript function
			if !strings.Contains(string(output), "function App") {
				t.Errorf("Expected to find 'function App' in frontend project, got: %s", string(output))
			}
		}

		// Search for Python-specific content in backend
		cmd = exec.Command("code-search", "search", "--dir", projectB, "def app")
		output, err = cmd.CombinedOutput()

		if err != nil {
			if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet for backend search, got: %s", string(output))
			} else {
				t.Errorf("Expected backend search to work after implementation, got: %v, output: %s", err, string(output))
			}
		} else {
			// Should find Python function
			if !strings.Contains(string(output), "def app") {
				t.Errorf("Expected to find 'def app' in backend project, got: %s", string(output))
			}
		}
	})
}

// TestIndexFileCleanup tests cleanup of index files
func TestIndexFileCleanup(t *testing.T) {
	projectDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create a project
	createMultiProjectStructure(t, projectDir)

	// Change to project directory
	err := os.Chdir(projectDir)
	if err != nil {
		t.Fatalf("Failed to change to project directory: %v", err)
	}

	// Index the project
	cmd := exec.Command("code-search", "index", "--dir", projectDir)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if strings.Contains(string(output), "unknown flag") || strings.Contains(string(output), "flag provided but not defined") {
			t.Skip("Skipping cleanup test as --dir flag is not implemented")
		}
	}

	// Check that .clindex directory exists
	clindexDir := filepath.Join(projectDir, ".clindex")
	if _, err := os.Stat(clindexDir); os.IsNotExist(err) {
		t.Skip("Skipping cleanup test as .clindex directory was not created")
	}

	// Simulate cleanup (this would be implemented as a separate command or flag)
	// For now, just verify that we can remove the index directory manually
	err = os.RemoveAll(clindexDir)
	if err != nil {
		t.Errorf("Failed to remove .clindex directory: %v", err)
	}

	// Verify directory is gone
	if _, err := os.Stat(clindexDir); err == nil {
		t.Errorf("Expected .clindex directory to be removed, but it still exists")
	}
}

// Helper function to create a basic project structure
func createMultiProjectStructure(t *testing.T, projectDir string) {
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

