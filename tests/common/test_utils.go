package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// SetupTestEnvironment sets up a test directory for testing
// Returns the absolute path to the test resource directory
func SetupTestEnvironment(t *testing.T, testDirName string) string {
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

	return resourceDir
}

// CreateTestDirectory creates a test directory with specified files
func CreateTestDirectory(t *testing.T, name string, files map[string]string) string {
	tempDir := t.TempDir()

	testDir := filepath.Join(tempDir, name)
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	for filename, content := range files {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	return testDir
}

// CleanupTestDirectory removes a test directory and all its contents
func CleanupTestDirectory(t *testing.T, dirPath string) {
	if err := os.RemoveAll(dirPath); err != nil {
		t.Logf("Warning: Failed to cleanup test directory %s: %v", dirPath, err)
	}
}

// AssertFileExists checks if a file exists
func AssertFileExists(t *testing.T, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist, but it doesn't", filePath)
	}
}

// AssertFileNotExists checks if a file does not exist
func AssertFileNotExists(t *testing.T, filePath string) {
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Expected file %s to not exist, but it does", filePath)
	}
}

// CreateTempFile creates a temporary file with the given content
func CreateTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test-*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}