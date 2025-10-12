package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestIndexWithDirFlag tests the index command with --dir flag
// This test should fail initially as the --dir flag is not implemented
func TestIndexWithDirFlag(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create some test files
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test index command with --dir flag
	// This should fail initially because the --dir flag is not implemented
	cmd := exec.Command("code-search", "index", "--dir", tempDir)
	output, err := cmd.CombinedOutput()

	// Initially, this should fail with an "unknown flag" error
	if err == nil {
		t.Errorf("Expected index command with --dir flag to fail initially, but it succeeded")
	}

	// The error should indicate that the --dir flag is not recognized
	if !contains(string(output), "unknown flag") && !contains(string(output), "flag provided but not defined") {
		t.Errorf("Expected 'unknown flag' error for --dir flag, got: %s", string(output))
	}
}

// TestIndexWithRelativePath tests indexing with relative paths
func TestIndexWithRelativePath(t *testing.T) {
	// Create a temporary directory and change to it
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a test file
	testFile := "test.go"
	err = os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test index command with relative path
	cmd := exec.Command("code-search", "index", "--dir", ".")
	output, err := cmd.CombinedOutput()

	// Initially, this should fail because --dir flag is not implemented
	if err == nil {
		t.Errorf("Expected index command with --dir flag to fail initially, but it succeeded")
	}

	if !contains(string(output), "unknown flag") && !contains(string(output), "flag provided but not defined") {
		t.Errorf("Expected 'unknown flag' error for --dir flag, got: %s", string(output))
	}
}

// TestIndexWithAbsolutePath tests indexing with absolute paths
func TestIndexWithAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test index command with absolute path
	cmd := exec.Command("code-search", "index", "--dir", tempDir)
	output, err := cmd.CombinedOutput()

	// Initially, this should fail because --dir flag is not implemented
	if err == nil {
		t.Errorf("Expected index command with --dir flag to fail initially, but it succeeded")
	}

	if !contains(string(output), "unknown flag") && !contains(string(output), "flag provided but not defined") {
		t.Errorf("Expected 'unknown flag' error for --dir flag, got: %s", string(output))
	}
}

// TestIndexWithoutDirFlagMaintainsDefaultBehavior ensures backward compatibility
func TestIndexWithoutDirFlagMaintainsDefaultBehavior(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir := t.TempDir()
	err := os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a test file
	testFile := "test.go"
	err = os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test index command without --dir flag (should still work)
	cmd := exec.Command("code-search", "index")
	output, err := cmd.CombinedOutput()

	// This should work (backward compatibility)
	if err != nil {
		t.Errorf("Expected index command without --dir flag to work, got error: %v, output: %s", err, string(output))
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())))
}