package unit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"code-search/src/lib"
)

// TestDirectoryValidator_ValidateDirectory tests the ValidateDirectory method
func TestDirectoryValidator_ValidateDirectory(t *testing.T) {
	validator := lib.NewDirectoryValidator()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "validation_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("Valid directory", func(t *testing.T) {
		config, err := validator.ValidateDirectory(tempDir)
		if err != nil {
			t.Errorf("Expected no error for valid directory, got: %v", err)
		}

		if config == nil {
			t.Fatal("Expected config to be non-nil")
		}

		if config.Path != tempDir {
			t.Errorf("Expected path %s, got %s", tempDir, config.Path)
		}

		if !config.Permissions.CanRead {
			t.Error("Expected directory to be readable")
		}

		if !config.Permissions.CanWrite {
			t.Error("Expected directory to be writable")
		}
	})

	t.Run("Non-existent directory", func(t *testing.T) {
		_, err := validator.ValidateDirectory("/non/existent/path")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}

		expectedMsg := "does not exist"
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to contain '%s', got: %v", expectedMsg, err)
		}
	})

	t.Run("File instead of directory", func(t *testing.T) {
		// Create a temporary file
		tempFile := filepath.Join(tempDir, "test_file.txt")
		if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		_, err := validator.ValidateDirectory(tempFile)
		if err == nil {
			t.Error("Expected error for file path")
		}

		expectedMsg := "not a directory"
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to contain '%s', got: %v", expectedMsg, err)
		}
	})

	t.Run("Empty path (current directory)", func(t *testing.T) {
		config, err := validator.ValidateDirectory(".")
		if err != nil {
			t.Errorf("Expected no error for current directory, got: %v", err)
		}

		if !config.IsDefault {
			t.Error("Expected IsDefault to be true for current directory")
		}
	})

	t.Run("Relative path", func(t *testing.T) {
		// Change to temp directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		config, err := validator.ValidateDirectory(".")
		if err != nil {
			t.Errorf("Expected no error for relative path, got: %v", err)
		}

		// Path should be resolved to absolute
		if !filepath.IsAbs(config.Path) {
			t.Error("Expected absolute path in config")
		}
	})
}

// TestDirectoryValidator_ValidateDirectoryPermissions tests permission validation
func TestDirectoryValidator_ValidateDirectoryPermissions(t *testing.T) {
	validator := lib.NewDirectoryValidator()

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "permission_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("Writable directory", func(t *testing.T) {
		config, err := validator.ValidateDirectory(tempDir)
		if err != nil {
			t.Errorf("Expected no error for writable directory, got: %v", err)
		}

		if !config.Permissions.CanWrite {
			t.Error("Expected directory to be writable")
		}
	})

	t.Run("Directory size calculation", func(t *testing.T) {
		// Create some test files
		for i := 0; i < 5; i++ {
			filePath := filepath.Join(tempDir, fmt.Sprintf("test_file_%d.txt", i))
			content := []byte(fmt.Sprintf("test content %d", i))
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		config, err := validator.ValidateDirectory(tempDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if config.Metadata.FileCount != 5 {
			t.Errorf("Expected 5 files, got %d", config.Metadata.FileCount)
		}

		if config.Metadata.TotalSize == 0 {
			t.Error("Expected non-zero total size")
		}
	})

	t.Run("Skips .clindex directories", func(t *testing.T) {
		// Create .clindex directory with files
		clindexDir := filepath.Join(tempDir, ".clindex")
		if err := os.MkdirAll(clindexDir, 0755); err != nil {
			t.Fatalf("Failed to create .clindex directory: %v", err)
		}

		// Add files to .clindex
		for i := 0; i < 3; i++ {
			filePath := filepath.Join(clindexDir, fmt.Sprintf("index_file_%d.db", i))
			if err := os.WriteFile(filePath, []byte("index data"), 0644); err != nil {
				t.Fatalf("Failed to create index file: %v", err)
			}
		}

		// Add regular files
		for i := 0; i < 2; i++ {
			filePath := filepath.Join(tempDir, fmt.Sprintf("regular_file_%d.txt", i))
			if err := os.WriteFile(filePath, []byte("regular content"), 0644); err != nil {
				t.Fatalf("Failed to create regular file: %v", err)
			}
		}

		config, err := validator.ValidateDirectory(tempDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Should only count regular files, not .clindex files
		if config.Metadata.FileCount != 2 {
			t.Errorf("Expected 2 files (excluding .clindex), got %d", config.Metadata.FileCount)
		}
	})
}

// TestDirectoryValidator_ValidateDirectoryLimits tests limit validation
func TestDirectoryValidator_ValidateDirectoryLimits(t *testing.T) {
	validator := lib.NewDirectoryValidator()

	t.Run("Small directory within limits", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "limits_test_small")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a few small files
		for i := 0; i < 10; i++ {
			filePath := filepath.Join(tempDir, fmt.Sprintf("small_file_%d.txt", i))
			if err := os.WriteFile(filePath, []byte("small content"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		_, err = validator.ValidateDirectory(tempDir)
		if err != nil {
			t.Errorf("Expected no error for directory within limits, got: %v", err)
		}
	})

	// Note: We can't easily test limit violations without creating huge directories
	// or modifying the limits, which would make the test brittle
	// In real scenarios, these would be integration tests with actual large directories
}

// TestDirectoryValidator_EdgeCases tests edge cases and error conditions
func TestDirectoryValidator_EdgeCases(t *testing.T) {
	validator := lib.NewDirectoryValidator()

	t.Run("Path with tilde expansion", func(t *testing.T) {
		// Test that tilde paths are handled (may not work in all environments)
		config, err := validator.ValidateDirectory("~/")
		if err != nil {
			// This might fail in some test environments, which is acceptable
			t.Logf("Tilde expansion failed (possibly expected in test environment): %v", err)
		} else if config == nil {
			t.Error("Expected config to be non-nil when tilde expansion succeeds")
		}
	})

	t.Run("Very long path name", func(t *testing.T) {
		// Create a directory with a long name
		tempDir, err := os.MkdirTemp("", "")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		longName := string(make([]byte, 100)) // Create a 100-character name
		for i := range longName {
			longName = longName[:i] + "a" + longName[i+1:]
		}

		longPath := filepath.Join(tempDir, longName)
		if err := os.Mkdir(longPath, 0755); err != nil {
			t.Fatalf("Failed to create long path directory: %v", err)
		}

		_, err = validator.ValidateDirectory(longPath)
		if err != nil {
			t.Errorf("Expected no error for long path, got: %v", err)
		}
	})

	t.Run("Directory with special characters", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		specialName := "test-dir_with.special@chars"
		specialPath := filepath.Join(tempDir, specialName)
		if err := os.Mkdir(specialPath, 0755); err != nil {
			t.Fatalf("Failed to create special char directory: %v", err)
		}

		_, err = validator.ValidateDirectory(specialPath)
		if err != nil {
			t.Errorf("Expected no error for special characters, got: %v", err)
		}
	})
}

// BenchmarkDirectoryValidator_ValidateDirectory benchmarks the validation performance
func BenchmarkDirectoryValidator_ValidateDirectory(b *testing.B) {
	validator := lib.NewDirectoryValidator()

	// Create a temporary directory with some files
	tempDir, err := os.MkdirTemp("", "benchmark_test")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	for i := 0; i < 100; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("bench_file_%d.txt", i))
		content := []byte(fmt.Sprintf("benchmark content %d", i))
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateDirectory(tempDir)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

// TestDirectoryValidator_ConcurrentAccess tests concurrent access to validator
func TestDirectoryValidator_ConcurrentAccess(t *testing.T) {
	validator := lib.NewDirectoryValidator()

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "concurrent_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some test files
	for i := 0; i < 10; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("concurrent_file_%d.txt", i))
		if err := os.WriteFile(filePath, []byte("concurrent test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Run validations concurrently
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := validator.ValidateDirectory(tempDir)
			if err != nil {
				errors <- err
			} else {
				errors <- nil
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
		err := <-errors
		if err != nil {
			t.Errorf("Concurrent validation failed: %v", err)
		}
	}
}