package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestSearchWithDirFlag tests the search command with --dir flag
// This test should fail initially as the --dir flag is not implemented
func TestSearchWithDirFlag(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create some test files
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test search command with --dir flag
	// This should fail initially because the --dir flag is not implemented
	cmd := exec.Command("code-search", "search", "--dir", tempDir, "func main")
	output, err := cmd.CombinedOutput()

	// Initially, this should fail with an "unknown flag" error
	if err == nil {
		t.Errorf("Expected search command with --dir flag to fail initially, but it succeeded")
	}

	// The error should indicate that the --dir flag is not recognized
	if !contains(string(output), "unknown flag") && !contains(string(output), "flag provided but not defined") {
		t.Errorf("Expected 'unknown flag' error for --dir flag, got: %s", string(output))
	}
}

// TestSearchWithRelativePath tests searching with relative paths
func TestSearchWithRelativePath(t *testing.T) {
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

	// Test search command with relative path
	cmd := exec.Command("code-search", "search", "--dir", ".", "func main")
	output, err := cmd.CombinedOutput()

	// Initially, this should fail because --dir flag is not implemented
	if err == nil {
		t.Errorf("Expected search command with --dir flag to fail initially, but it succeeded")
	}

	if !contains(string(output), "unknown flag") && !contains(string(output), "flag provided but not defined") {
		t.Errorf("Expected 'unknown flag' error for --dir flag, got: %s", string(output))
	}
}

// TestSearchWithAbsolutePath tests searching with absolute paths
func TestSearchWithAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test search command with absolute path
	cmd := exec.Command("code-search", "search", "--dir", tempDir, "func main")
	output, err := cmd.CombinedOutput()

	// Initially, this should fail because --dir flag is not implemented
	if err == nil {
		t.Errorf("Expected search command with --dir flag to fail initially, but it succeeded")
	}

	if !contains(string(output), "unknown flag") && !contains(string(output), "flag provided but not defined") {
		t.Errorf("Expected 'unknown flag' error for --dir flag, got: %s", string(output))
	}
}

// TestSearchWithoutDirFlagMaintainsDefaultBehavior ensures backward compatibility
func TestSearchWithoutDirFlagMaintainsDefaultBehavior(t *testing.T) {
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

	// Test search command without --dir flag (should still work)
	cmd := exec.Command("code-search", "search", "func main")
	output, err := cmd.CombinedOutput()

	// This should work (backward compatibility)
	if err != nil {
		t.Errorf("Expected search command without --dir flag to work, got error: %v, output: %s", err, string(output))
	}
}

// TestSearchWithFormatAndDirFlags tests search with both --format and --dir flags
func TestSearchWithFormatAndDirFlags(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test search command with both flags
	cmd := exec.Command("code-search", "search", "--format", "json", "--dir", tempDir, "func main")
	output, err := cmd.CombinedOutput()

	// Initially, this should fail because --dir flag is not implemented
	if err == nil {
		t.Errorf("Expected search command with --dir flag to fail initially, but it succeeded")
	}

	if !contains(string(output), "unknown flag") && !contains(string(output), "flag provided but not defined") {
		t.Errorf("Expected 'unknown flag' error for --dir flag, got: %s", string(output))
	}
}

// TestSearchWithMaxResultsAndDirFlags tests search with both --max-results and --dir flags
func TestSearchWithMaxResultsAndDirFlags(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test search command with both flags
	cmd := exec.Command("code-search", "search", "--max-results", "10", "--dir", tempDir, "func main")
	output, err := cmd.CombinedOutput()

	// Initially, this should fail because --dir flag is not implemented
	if err == nil {
		t.Errorf("Expected search command with --dir flag to fail initially, but it succeeded")
	}

	if !contains(string(output), "unknown flag") && !contains(string(output), "flag provided but not defined") {
		t.Errorf("Expected 'unknown flag' error for --dir flag, got: %s", string(output))
	}
}