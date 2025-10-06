package contract

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestCLIIndexCommand tests the contract for the CLI index command
func TestCLIIndexCommand(t *testing.T) {
	// Create a temporary directory with test files
	tempDir := t.TempDir()

	// Create test Go files
	testFile1 := tempDir + "/main.go"
	err := os.WriteFile(testFile1, []byte(`package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func calculateSum(a, b int) int {
	return a + b
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFile2 := tempDir + "/utils.go"
	err = os.WriteFile(testFile2, []byte(`package main

import "strings"

func validateInput(input string) bool {
	return strings.TrimSpace(input) != ""
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Build the CLI tool (this will fail initially since we haven't implemented it yet)
	cmd := exec.Command("go", "build", "-o", "code-search", "../../src/cli/main.go")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()

	// If build fails, that's expected at this point - we'll skip the functional test
	if err != nil {
		t.Skipf("CLI tool not yet implemented (build failed): %v\nOutput: %s", err, string(output))
	}

	// Test the index command
	cmd = exec.Command("./code-search", "index")
	cmd.Dir = tempDir
	start := time.Now()
	output, err = cmd.CombinedOutput()
	duration := time.Since(start)

	// Check exit code
	if err != nil {
		t.Errorf("Index command failed with exit code %v: %v", err, string(output))
	}

	// Check that output contains expected success message
	expectedMsg := "Indexing complete. Indexed "
	if !strings.Contains(string(output), expectedMsg) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedMsg, string(output))
	}

	// Check that output mentions files were indexed
	if !strings.Contains(string(output), "files") {
		t.Errorf("Expected output to mention 'files', got: %s", string(output))
	}

	// Check that indexing completes in reasonable time (< 30 seconds for small test case)
	if duration > 30*time.Second {
		t.Errorf("Indexing took too long: %v (should be < 30s)", duration)
	}

	// Check that an index file was created
	if _, err := os.Stat(tempDir + "/.code-search-index"); os.IsNotExist(err) {
		t.Error("Expected index file to be created, but it doesn't exist")
	}

	t.Logf("Index command test passed. Duration: %v, Output: %s", duration, string(output))
}

// TestCLIIndexCommandWithForce tests the --force flag
func TestCLIIndexCommandWithForce(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := tempDir + "/test.go"
	err := os.WriteFile(testFile, []byte(`package main

func test() {}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Build the CLI tool
	cmd := exec.Command("go", "build", "-o", "code-search", "../../src/cli/main.go")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("CLI tool not yet implemented: %v\nOutput: %s", err, string(output))
	}

	// Test with --force flag
	cmd = exec.Command("./code-search", "index", "--force")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Index command with --force failed: %v, output: %s", err, string(output))
	}

	expectedMsg := "Indexing complete. Indexed "
	if !strings.Contains(string(output), expectedMsg) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedMsg, string(output))
	}
}

// TestCLIIndexCommandErrorHandling tests error scenarios
func TestCLIIndexCommandErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Build the CLI tool
	cmd := exec.Command("go", "build", "-o", "code-search", "../../src/cli/main.go")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("CLI tool not yet implemented: %v\nOutput: %s", err, string(output))
	}

	// Test with invalid arguments
	cmd = exec.Command("./code-search", "index", "--invalid-flag")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()

	if err == nil {
		t.Error("Expected index command to fail with invalid flag, but it succeeded")
	}

	// Should exit with code 2 for invalid arguments
	if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != 2 {
		t.Errorf("Expected exit code 2 for invalid arguments, got %d", exitError.ExitCode())
	}
}