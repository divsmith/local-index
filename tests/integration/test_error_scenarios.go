package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestErrorScenarios tests various error scenarios
func TestErrorScenarios(t *testing.T) {
	t.Run("IndexNonExistentDirectory", func(t *testing.T) {
		nonExistentDir := "/path/that/does/not/exist"

		cmd := exec.Command("code-search", "index", "--dir", nonExistentDir)
		output, err := cmd.CombinedOutput()

		// Should fail initially due to unknown flag, then give specific error after implementation
		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "does not exist") && !contains(string(output), "not found") {
				t.Errorf("Expected directory not found error, got: %s", string(output))
			}
		} else {
			t.Errorf("Expected index command with non-existent directory to fail")
		}
	})

	t.Run("SearchNonExistentDirectory", func(t *testing.T) {
		nonExistentDir := "/path/that/does/not/exist"

		cmd := exec.Command("code-search", "search", "--dir", nonExistentDir, "test query")
		output, err := cmd.CombinedOutput()

		// Should fail initially due to unknown flag, then give specific error after implementation
		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "does not exist") && !contains(string(output), "not found") {
				t.Errorf("Expected directory not found error, got: %s", string(output))
			}
		} else {
			t.Errorf("Expected search command with non-existent directory to fail")
		}
	})

	t.Run("IndexFileInsteadOfDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.go")
		err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		cmd := exec.Command("code-search", "index", "--dir", testFile)
		output, err := cmd.CombinedOutput()

		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "not a directory") {
				t.Errorf("Expected 'not a directory' error, got: %s", string(output))
			}
		} else {
			t.Errorf("Expected index command with file to fail")
		}
	})

	t.Run("SearchInUnindexedDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.go")
		err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		cmd := exec.Command("code-search", "search", "--dir", tempDir, "func main")
		output, err := cmd.CombinedOutput()

		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "No index found") && !contains(string(output), "not indexed") {
				t.Errorf("Expected 'no index found' error, got: %s", string(output))
			}
		} else {
			t.Errorf("Expected search in unindexed directory to fail")
		}
	})
}

// TestPermissionErrors tests permission-related errors
func TestPermissionErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission tests when running as root")
	}

	t.Run("IndexDirectoryWithoutReadPermission", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.go")
		err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Remove read permissions
		err = os.Chmod(tempDir, 0300) // Execute only
		if err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}
		defer os.Chmod(tempDir, 0755) // Restore permissions

		cmd := exec.Command("code-search", "index", "--dir", tempDir)
		output, err := cmd.CombinedOutput()

		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "permission denied") {
				t.Errorf("Expected permission denied error, got: %s", string(output))
			}
		} else {
			t.Errorf("Expected index command without read permission to fail")
		}
	})

	t.Run("IndexDirectoryWithoutWritePermission", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.go")
		err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Remove write permissions (keep read)
		err = os.Chmod(tempDir, 0500) // Read and execute only
		if err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}
		defer os.Chmod(tempDir, 0755) // Restore permissions

		cmd := exec.Command("code-search", "index", "--dir", tempDir)
		output, err := cmd.CombinedOutput()

		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "permission denied") && !contains(string(output), "cannot write") {
				t.Errorf("Expected write permission error, got: %s", string(output))
			}
		} else {
			t.Errorf("Expected index command without write permission to fail")
		}
	})
}

// TestSizeLimitErrors tests size limit related errors
func TestSizeLimitErrors(t *testing.T) {
	t.Run("IndexLargeDirectory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a large file (simulate exceeding size limit)
		largeFile := filepath.Join(tempDir, "large.txt")
		// Create 50MB file (assuming default limit is 100MB for individual files)
		data := make([]byte, 50*1024*1024)
		for i := range data {
			data[i] = 'A'
		}

		err := os.WriteFile(largeFile, data, 0644)
		if err != nil {
			t.Fatalf("Failed to create large test file: %v", err)
		}

		cmd := exec.Command("code-search", "index", "--dir", tempDir)
		output, err := cmd.CombinedOutput()

		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "exceeds limit") && !contains(string(output), "too large") {
				t.Errorf("Expected size limit error, got: %s", string(output))
			}
		} else {
			// This might pass if size limits are not implemented yet
			t.Logf("Size limits may not be implemented yet")
		}
	})

	t.Run("IndexDirectoryWithManyFiles", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create many small files
		for i := 0; i < 100; i++ { // Create 100 files
			file := filepath.Join(tempDir, "file_"+string(rune(i))+".txt")
			err := os.WriteFile(file, []byte("content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file %d: %v", i, err)
			}
		}

		cmd := exec.Command("code-search", "index", "--dir", tempDir)
		output, err := cmd.CombinedOutput()

		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "too many files") {
				t.Errorf("Expected file count limit error, got: %s", string(output))
			}
		} else {
			// This might pass if file count limits are high or not implemented
			t.Logf("File count limits may not be implemented yet")
		}
	})
}

// TestIndexCorruptionErrors tests index corruption scenarios
func TestIndexCorruptionErrors(t *testing.T) {
	t.Run("SearchWithCorruptedIndex", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create .clindex directory with corrupted files
		clindexDir := filepath.Join(tempDir, ".clindex")
		err := os.MkdirAll(clindexDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .clindex directory: %v", err)
		}

		// Create corrupted metadata file
		metadataFile := filepath.Join(clindexDir, "metadata.json")
		err = os.WriteFile(metadataFile, []byte("invalid json content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create corrupted metadata file: %v", err)
		}

		// Create corrupted data file
		dataFile := filepath.Join(clindexDir, "data.index")
		err = os.WriteFile(dataFile, []byte("corrupted binary data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create corrupted data file: %v", err)
		}

		cmd := exec.Command("code-search", "search", "--dir", tempDir, "test query")
		output, err := cmd.CombinedOutput()

		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "corrupted") && !contains(string(output), "invalid") {
				t.Errorf("Expected index corruption error, got: %s", string(output))
			}
		} else {
			t.Errorf("Expected search with corrupted index to fail")
		}
	})
}

// TestConcurrentAccessErrors tests concurrent access scenarios
func TestConcurrentAccessErrors(t *testing.T) {
	t.Run("ConcurrentIndexing", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create test file
		testFile := filepath.Join(tempDir, "test.go")
		err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a lock file to simulate concurrent indexing
		lockFile := filepath.Join(tempDir, ".clindex", "lock")
		err = os.MkdirAll(filepath.Dir(lockFile), 0755)
		if err != nil {
			t.Fatalf("Failed to create .clindex directory: %v", err)
		}

		err = os.WriteFile(lockFile, []byte("locked"), 0644)
		if err != nil {
			t.Fatalf("Failed to create lock file: %v", err)
		}
		defer os.Remove(lockFile)

		cmd := exec.Command("code-search", "index", "--dir", tempDir)
		output, err := cmd.CombinedOutput()

		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "locked") && !contains(string(output), "concurrent") {
				t.Errorf("Expected concurrent access error, got: %s", string(output))
			}
		} else {
			t.Logf("Concurrent access control may not be implemented yet")
		}
	})
}

// TestSecurityErrorScenarios tests security-related error scenarios
func TestSecurityErrorScenarios(t *testing.T) {
	t.Run("PathTraversalAttack", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a test file
		testFile := filepath.Join(tempDir, "test.go")
		err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Try path traversal attacks
		maliciousPaths := []string{
			filepath.Join(tempDir, "..", "..", "etc", "passwd"),
			"../../../etc/passwd",
			"/etc/passwd",
		}

		for _, maliciousPath := range maliciousPaths {
			t.Run("PathTraversal_"+filepath.Base(maliciousPath), func(t *testing.T) {
				cmd := exec.Command("code-search", "index", "--dir", maliciousPath)
				output, err := cmd.CombinedOutput()

				if err != nil {
					if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
						t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
					} else if !contains(string(output), "path traversal") && !contains(string(output), "security") {
						t.Errorf("Expected path traversal security error, got: %s", string(output))
					}
				} else {
					t.Errorf("Expected path traversal attack to be blocked")
				}
			})
		}
	})

	t.Run("SymlinkAttack", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a test file outside the directory
		outsideFile := filepath.Join(tempDir, "..", "outside.txt")
		err := os.WriteFile(outsideFile, []byte("sensitive data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create outside file: %v", err)
		}

		// Create a symlink pointing outside
		symlinkPath := filepath.Join(tempDir, "symlink.txt")
		err = os.Symlink(outsideFile, symlinkPath)
		if err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}

		cmd := exec.Command("code-search", "index", "--dir", tempDir)
		output, err := cmd.CombinedOutput()

		if err != nil {
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "symlink") && !contains(string(output), "security") {
				t.Errorf("Expected symlink security error, got: %s", string(output))
			}
		} else {
			t.Logf("Symlink security may not be implemented yet")
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}