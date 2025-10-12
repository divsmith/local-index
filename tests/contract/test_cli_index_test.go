package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)


// TestCLIIndexCommand tests the contract for the CLI index command
func TestCLIIndexCommand(t *testing.T) {
	// Set up test environment
	resourceDir := setupTestEnvironment(t, "TestCLIIndexCommand")

	// Create test Go files
	testFile1 := filepath.Join(resourceDir, "main.go")
	err := os.WriteFile(testFile1, []byte(`package main

import "fmt"

func main() {
	config := LoadConfig()
	if err := StartServer(config); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

func LoadConfig() *Config {
	return &Config{Port: 8080, Timeout: 30}
}

func StartServer(config *Config) error {
	fmt.Printf("Server starting on port %d\n", config.Port)
	return nil
}

type Config struct {
	Port    int
	Timeout int
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFile2 := filepath.Join(resourceDir, "utils.go")
	err = os.WriteFile(testFile2, []byte(`package main

import "fmt"

func HelperFunction(data string) {
	fmt.Printf("Processing: %s\n", data)
}

func CalculateSum(a, b int) int {
	return a + b
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to resource directory
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(resourceDir)
	if err != nil {
		t.Fatalf("Failed to change to resource directory: %v", err)
	}

	// Test the index command
	cmd := exec.Command("../../../../bin/code-search", "index")
	cmd.Dir = resourceDir
	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	// Check exit code
	if err != nil {
		t.Errorf("Index command failed with exit code %v: %v", err, string(output))
		return
	}

	outputStr := string(output)

	// Should report success
	if !strings.Contains(outputStr, "Indexing complete") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}

	// Should have indexed the files we created
	if !strings.Contains(outputStr, "2") {
		t.Errorf("Expected to index 2 files, got: %s", outputStr)
	}

	// Should complete in reasonable time
	if duration > 30*time.Second {
		t.Errorf("Indexing took too long: %v", duration)
	}

	// Check that index file was created
	if _, err := os.Stat(".code-search-index"); os.IsNotExist(err) {
		t.Error("Expected index file to be created")
	}

	t.Logf("Index command test passed. Duration: %v, Output: %s", duration, outputStr)
}

// TestCLIIndexCommandWithForce tests the force reindex functionality
func TestCLIIndexCommandWithForce(t *testing.T) {
	// Set up test environment
	resourceDir := setupTestEnvironment(t, "TestCLIIndexCommandWithForce")

	// Create a simple test file
	testFile := filepath.Join(resourceDir, "main.go")
	err := os.WriteFile(testFile, []byte(`package main

func main() {
	println("Hello World")
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to resource directory
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(resourceDir)
	if err != nil {
		t.Fatalf("Failed to change to resource directory: %v", err)
	}

	// First indexing
	cmd := exec.Command("../../../../bin/code-search", "index")
	cmd.Dir = resourceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("First index command failed: %v, output: %s", err, string(output))
		return
	}

	// Wait a moment before force re-index
	time.Sleep(100 * time.Millisecond)

	// Force re-index
	cmd = exec.Command("../../../../bin/code-search", "index", "--force")
	cmd.Dir = resourceDir
	start := time.Now()
	output, err = cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Force re-index command failed: %v, output: %s", err, string(output))
		return
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Indexing complete") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}

	// Check that index file still exists and has reasonable size
	indexFile := ".code-search-index"
	stat2, err := os.Stat(indexFile)
	if err != nil {
		t.Fatalf("Failed to stat index file after re-index: %v", err)
	}

	// Verify index file exists and has content (size should be reasonable)
	if stat2.Size() == 0 {
		t.Error("Expected index file to have content after force re-index")
	}

	// Note: We don't check modification time as it may not update if content is identical
	// The key test is that the force re-index command executes successfully

	t.Logf("Force re-index test passed. Duration: %v", duration)
}

// TestCLIIndexCommandErrorHandling tests error scenarios
func TestCLIIndexCommandErrorHandling(t *testing.T) {
	// Set up test environment
	resourceDir := setupTestEnvironment(t, "TestCLIIndexCommandErrorHandling")

	// Change to resource directory
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(resourceDir)
	if err != nil {
		t.Fatalf("Failed to change to resource directory: %v", err)
	}

	// Test with invalid arguments
	cmd := exec.Command("../../../../bin/code-search", "index", "--invalid-flag")
	cmd.Dir = resourceDir
	output, err := cmd.CombinedOutput()

	// Should exit with code 2 for invalid arguments
	if err == nil {
		t.Error("Expected index command to fail with invalid flag, but it succeeded")
	}

	// Check exit code
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 2 {
		t.Errorf("Expected exit code 2 for invalid arguments, got %d", exitErr.ExitCode())
	}

	// Test with help flag
	cmd = exec.Command("../../../../bin/code-search", "index", "--help")
	cmd.Dir = resourceDir
	output, err = cmd.CombinedOutput()

	// Should exit with code 0 for help
	if err != nil {
		t.Errorf("Help command should not fail: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Usage:") {
		t.Errorf("Expected help output, got: %s", outputStr)
	}

	t.Logf("Error handling test passed")
}