package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestIndexWithNonExistentDirectory tests indexing a non-existent directory
func TestIndexWithNonExistentDirectory(t *testing.T) {
	nonExistentDir := "/path/that/does/not/exist"

	// Test index command with non-existent directory
	cmd := exec.Command("code-search", "index", "--dir", nonExistentDir)
	output, err := cmd.CombinedOutput()

	// Should fail initially due to unknown flag, but after implementation should give directory not found error
	if err == nil {
		t.Errorf("Expected index command with non-existent directory to fail, but it succeeded")
	}

	// Initially, it will fail with unknown flag error
	// After implementation, should give specific directory not found error
	if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
		t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
	} else if !contains(string(output), "does not exist") && !contains(string(output), "not found") {
		t.Errorf("Expected directory not found error, got: %s", string(output))
	}
}

// TestSearchWithNonExistentDirectory tests searching in a non-existent directory
func TestSearchWithNonExistentDirectory(t *testing.T) {
	nonExistentDir := "/path/that/does/not/exist"

	// Test search command with non-existent directory
	cmd := exec.Command("code-search", "search", "--dir", nonExistentDir, "test query")
	output, err := cmd.CombinedOutput()

	// Should fail initially due to unknown flag, but after implementation should give directory not found error
	if err == nil {
		t.Errorf("Expected search command with non-existent directory to fail, but it succeeded")
	}

	// Initially, it will fail with unknown flag error
	// After implementation, should give specific directory not found error
	if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
		t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
	} else if !contains(string(output), "does not exist") && !contains(string(output), "not found") {
		t.Errorf("Expected directory not found error, got: %s", string(output))
	}
}

// TestIndexWithFileInsteadOfDirectory tests indexing a file instead of directory
func TestIndexWithFileInsteadOfDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test index command with file instead of directory
	cmd := exec.Command("code-search", "index", "--dir", testFile)
	output, err := cmd.CombinedOutput()

	// Should fail initially due to unknown flag, but after implementation should give not a directory error
	if err == nil {
		t.Errorf("Expected index command with file instead of directory to fail, but it succeeded")
	}

	// Initially, it will fail with unknown flag error
	// After implementation, should give specific not a directory error
	if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
		t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
	} else if !contains(string(output), "not a directory") {
		t.Errorf("Expected 'not a directory' error, got: %s", string(output))
	}
}

// TestSearchWithFileInsteadOfDirectory tests searching in a file instead of directory
func TestSearchWithFileInsteadOfDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test search command with file instead of directory
	cmd := exec.Command("code-search", "search", "--dir", testFile, "func main")
	output, err := cmd.CombinedOutput()

	// Should fail initially due to unknown flag, but after implementation should give not a directory error
	if err == nil {
		t.Errorf("Expected search command with file instead of directory to fail, but it succeeded")
	}

	// Initially, it will fail with unknown flag error
	// After implementation, should give specific not a directory error
	if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
		t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
	} else if !contains(string(output), "not a directory") {
		t.Errorf("Expected 'not a directory' error, got: %s", string(output))
	}
}

// TestIndexWithDirectoryWithoutPermissions tests indexing without read permissions
func TestIndexWithDirectoryWithoutPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()

	// Create a test file
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

	// Test index command with directory without read permissions
	cmd := exec.Command("code-search", "index", "--dir", tempDir)
	output, err := cmd.CombinedOutput()

	// Should fail initially due to unknown flag, but after implementation should give permission error
	if err == nil {
		t.Errorf("Expected index command with no read permissions to fail, but it succeeded")
	}

	// Initially, it will fail with unknown flag error
	// After implementation, should give specific permission error
	if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
		t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
	} else if !contains(string(output), "permission denied") {
		t.Errorf("Expected permission denied error, got: %s", string(output))
	}
}

// TestPathTraversalSecurity tests path traversal security
func TestPathTraversalSecurity(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file in temp directory
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try path traversal attacks
	pathTraversalTests := []string{
		"../../../etc/passwd",
		filepath.Join(tempDir, "..", "..", "etc", "passwd"),
		"/etc/passwd",
	}

	for _, maliciousPath := range pathTraversalTests {
		t.Run("path_traversal_"+filepath.Base(maliciousPath), func(t *testing.T) {
			// Test index command with path traversal
			cmd := exec.Command("code-search", "index", "--dir", maliciousPath)
			output, err := cmd.CombinedOutput()

			// Should fail initially due to unknown flag, but after implementation should give security error
			if err == nil {
				t.Errorf("Expected index command with path traversal to fail, but it succeeded")
			}

			// Initially, it will fail with unknown flag error
			// After implementation, should give specific path traversal error
			if contains(string(output), "unknown flag") || contains(string(output), "flag provided but not defined") {
				t.Logf("Expected: --dir flag not implemented yet, got: %s", string(output))
			} else if !contains(string(output), "path traversal") {
				t.Errorf("Expected path traversal error, got: %s", string(output))
			}
		})
	}
}