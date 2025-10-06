package integration

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestEnvironment sets up a test directory with the CLI binary
func setupTestEnvironment(t *testing.T, testDirName string) string {
	// Create test resource directory - use absolute path to ensure it works from any test directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	resourceDir := filepath.Join(wd, "resources", testDirName)

	// Clean up any existing test directory
	if err := os.RemoveAll(resourceDir); err != nil {
		t.Fatalf("Failed to clean up existing test directory: %v", err)
	}

	// Create the test directory
	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Copy the CLI binary to the test directory
	srcBinary := filepath.Join(wd, "..", "..", "bin", "code-search")
	dstBinary := filepath.Join(resourceDir, "code-search")

	// Ensure the source binary exists
	if _, err := os.Stat(srcBinary); os.IsNotExist(err) {
		t.Fatalf("CLI binary not found at %s. Run 'make build' first.", srcBinary)
	}

	// Copy the binary
	input, err := os.ReadFile(srcBinary)
	if err != nil {
		t.Fatalf("Failed to read CLI binary: %v", err)
	}

	if err := os.WriteFile(dstBinary, input, 0755); err != nil {
		t.Fatalf("Failed to copy CLI binary: %v", err)
	}

	return resourceDir
}