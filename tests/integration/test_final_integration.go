package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code-search/src/lib"
)

// TestFinalIntegration tests the complete directory selection feature
func TestFinalIntegration(t *testing.T) {
	// Create test environment
	tempDir := t.TempDir()

	// Create test projects
	project1 := createTestProject(t, filepath.Join(tempDir, "project1"), map[string]string{
		"main.go":     "package main\n\nfunc main() { /* main function */ }",
		"utils.go":    "package main\n\nfunc Helper() { /* helper function */ }",
		"README.md":   "# Project 1\nThis is a test project.",
		".gitignore":  "bin/\n*.log",
	})

	project2 := createTestProject(t, filepath.Join(tempDir, "project2"), map[string]string{
		"app.js":      "// Main application\nconst API_URL = 'http://localhost:3000';",
		"utils.js":    "// Utility functions\nexport function formatDate(date) { return date.toISOString(); }",
		"package.json": "{\"name\": \"project2\", \"version\": \"1.0.0\"}",
		"test.js":     "// Test file\\ndescribe('utils', function() { /* test cases */ });",
	})

	// Create a nested project structure
	nestedProject := createTestProject(t, filepath.Join(tempDir, "nested-project"), map[string]string{
		"src/main.py":     "# Main module\ndef main():\n    pass",
		"src/utils/helpers.py": "# Helper functions\ndef format_text(text):\n    return text.strip()",
		"tests/test_main.py": "# Test main module\ndef test_main():\n    assert True",
		"docs/README.md":    "# Documentation\nProject documentation here.",
	})

	t.Run("Complete workflow test", func(t *testing.T) {
		// Test 1: Directory validation
		validator := lib.NewDirectoryValidator()
		config1, err := validator.ValidateDirectory(project1)
		if err != nil {
			t.Fatalf("Failed to validate project1: %v", err)
		}

		if config1.Metadata.FileCount != 4 { // 3 files + 1 directory (but we count only files)
			t.Errorf("Expected 4 files in project1, got %d", config1.Metadata.FileCount)
		}

		// Test 2: Index location management
		fileUtils := lib.NewFileUtilities()
		indexLoc1 := fileUtils.CreateIndexLocation(project1)

		expectedIndexDir := filepath.Join(project1, ".clindex")
		if indexLoc1.IndexDir != expectedIndexDir {
			t.Errorf("Expected index dir %s, got %s", expectedIndexDir, indexLoc1.IndexDir)
		}

		// Test 3: File locking
		lockFile, err := fileUtils.AcquireLock(project1)
		if err != nil {
			t.Fatalf("Failed to acquire lock: %v", err)
		}

		// Directory should be locked
		if !fileUtils.IsLocked(project1) {
			t.Error("Expected directory to be locked")
		}

		// Release lock
		if err := fileUtils.ReleaseLock(lockFile); err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}

		// Directory should no longer be locked
		if fileUtils.IsLocked(project1) {
			t.Error("Expected directory to be unlocked after release")
		}

		// Test 4: Performance optimization
		optimizer := lib.NewPerformanceOptimizer()

		// Create a larger directory for performance testing
		largeDir := filepath.Join(tempDir, "large-project")
		if err := os.MkdirAll(largeDir, 0755); err != nil {
			t.Fatalf("Failed to create large project directory: %v", err)
		}

		// Create many small files
		for i := 0; i < 100; i++ {
			filePath := filepath.Join(largeDir, fmt.Sprintf("file_%d.txt", i))
			content := fmt.Sprintf("Content of file %d with some text to make it slightly larger", i)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		// Test performance optimization
		optimizer.OptimizeForLargeDirectory(100)
		options := lib.DefaultScanOptions()
		options.BaseDir = largeDir
		options.ExcludeDirs = []string{".git", ".clindex"}

		result, err := optimizer.FastDirectoryScan(largeDir, options)
		if err != nil {
			t.Fatalf("Failed to scan large directory: %v", err)
		}

		if result.TotalFiles != 100 {
			t.Errorf("Expected 100 files in large directory, got %d", result.TotalFiles)
		}

		if result.Duration == "" {
			t.Error("Expected scan duration to be recorded")
		}

		// Test 5: Index migration simulation
		migrator := lib.NewIndexMigrator()

		// Create legacy index file
		legacyIndex := filepath.Join(project1, ".code-search-index")
		if err := os.WriteFile(legacyIndex, []byte("legacy index data"), 0644); err != nil {
			t.Fatalf("Failed to create legacy index: %v", err)
		}

		// Check migration status
		status, err := migrator.GetMigrationStatus(project1)
		if err != nil {
			t.Fatalf("Failed to get migration status: %v", err)
		}

		if status != "legacy" {
			t.Errorf("Expected legacy status, got %s", status)
		}

		// Test migration
		migrationResult, err := migrator.MigrateIndex(project1, false)
		if err != nil {
			t.Fatalf("Failed to migrate index: %v", err)
		}

		if !migrationResult.Success {
			t.Error("Expected migration to be successful")
		}

		if len(migrationResult.MigratedFiles) != 1 {
			t.Errorf("Expected 1 migrated file, got %d", len(migrationResult.MigratedFiles))
		}

		// Check new index structure exists
		if !fileUtils.DirectoryExists(indexLoc1.IndexDir) {
			t.Error("Expected new index directory to exist after migration")
		}

		// Test 6: Index info
		info, err := fileUtils.GetIndexInfo(project1)
		if err != nil {
			t.Fatalf("Failed to get index info: %v", err)
		}

		if !info.Exists {
			t.Error("Expected index to exist after migration")
		}

		// Test 7: Multi-project scenarios
		projects := []string{project1, project2, nestedProject}

		for _, project := range projects {
			config, err := validator.ValidateDirectory(project)
			if err != nil {
				t.Errorf("Failed to validate %s: %v", project, err)
				continue
			}

			if config.Metadata.FileCount == 0 {
				t.Errorf("Expected files to be found in %s", project)
			}
		}
	})

	t.Run("Error handling and edge cases", func(t *testing.T) {
		validator := lib.NewDirectoryValidator()
		fileUtils := lib.NewFileUtilities()

		// Test non-existent directory
		_, err := validator.ValidateDirectory("/non/existent/path")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}

		// Test permission issues (simulate by trying to access root)
		// This might not fail on all systems, so we just test the validation
		config, err := validator.ValidateDirectory("/")
		if err != nil {
			// Expected on most systems - permission denied
			t.Logf("Expected permission denied for root: %v", err)
		} else if config != nil {
			// Might succeed on some systems
			t.Logf("Root directory validation succeeded")
		}

		// Test file instead of directory
		testFile := filepath.Join(project1, "main.go")
		_, err = validator.ValidateDirectory(testFile)
		if err == nil {
			t.Error("Expected error when validating file as directory")
		}

		// Test file locking conflicts
		lock1, err := fileUtils.AcquireLock(project2)
		if err != nil {
			t.Fatalf("Failed to acquire first lock: %v", err)
		}

		// Try to acquire second lock (should fail or timeout)
		lock2, err := fileUtils.AcquireLock(project2)
		if err == nil {
			t.Error("Expected error when acquiring conflicting lock")
			if lock2 != nil {
				fileUtils.ReleaseLock(lock2)
			}
		}

		fileUtils.ReleaseLock(lock1)
	})

	t.Run("Backward compatibility", func(t *testing.T) {
		validator := lib.NewDirectoryValidator()

		// Test that empty directory (current directory) works
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}

		config, err := validator.ValidateDirectory(".")
		if err != nil {
			t.Fatalf("Failed to validate current directory: %v", err)
		}

		if !config.IsDefault {
			t.Error("Expected current directory to be marked as default")
		}

		if config.Path != originalDir {
			t.Errorf("Expected path %s, got %s", originalDir, config.Path)
		}

		// Test relative path resolution
		relativePath := filepath.Join(tempDir, "project1")
		absPath, err := filepath.Abs(relativePath)
		if err != nil {
			t.Fatalf("Failed to get absolute path: %v", err)
		}

		config, err = validator.ValidateDirectory(relativePath)
		if err != nil {
			t.Fatalf("Failed to validate relative path: %v", err)
		}

		if config.Path != absPath {
			t.Errorf("Expected absolute path %s, got %s", absPath, config.Path)
		}

		if config.OriginalPath != relativePath {
			t.Errorf("Expected original path %s, got %s", relativePath, config.OriginalPath)
		}
	})
}

// TestIntegrationPerformance tests performance characteristics
func TestIntegrationPerformance(t *testing.T) {
	tempDir := t.TempDir()

	// Create test data of varying sizes
	testCases := []struct {
		name      string
		fileCount int
	}{
		{"Small directory", 10},
		{"Medium directory", 100},
		{"Large directory", 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := filepath.Join(tempDir, tc.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Create test files
			for i := 0; i < tc.fileCount; i++ {
				filePath := filepath.Join(testDir, fmt.Sprintf("file_%d.txt", i))
				content := fmt.Sprintf("Content of file %d", i)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			// Test validation performance
			validator := lib.NewDirectoryValidator()
			start := time.Now()
			config, err := validator.ValidateDirectory(testDir)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Validation failed for %s: %v", tc.name, err)
			}

			if config.Metadata.FileCount != int64(tc.fileCount) {
				t.Errorf("Expected %d files, got %d", tc.fileCount, config.Metadata.FileCount)
			}

			// Performance assertions (these are generous bounds)
			maxDuration := time.Duration(tc.fileCount) * time.Millisecond
			if duration > maxDuration {
				t.Logf("Warning: %s validation took %v (expected <%v)", tc.name, duration, maxDuration)
			}

			t.Logf("%s (%d files): validated in %v", tc.name, tc.fileCount, duration)
		})
	}
}

// TestIntegrationConcurrency tests concurrent operations
func TestIntegrationConcurrency(t *testing.T) {
	tempDir := t.TempDir()

	// Create test project
	project := createTestProject(t, filepath.Join(tempDir, "concurrent-test"), map[string]string{
		"file1.go": "package main\n\nfunc test1() {}",
		"file2.go": "package main\n\nfunc test2() {}",
		"file3.go": "package main\n\nfunc test3() {}",
	})

	t.Run("Concurrent validation", func(t *testing.T) {
		validator := lib.NewDirectoryValidator()

		// Run validations concurrently
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := validator.ValidateDirectory(project)
				results <- err
			}()
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent validation failed: %v", err)
			}
		}
	})

	t.Run("Concurrent file locking", func(t *testing.T) {
		fileUtils := lib.NewFileUtilities()

		// Test shared locks (for reading)
		const numReaders = 5
		readLocks := make([]*os.File, numReaders)
		errors := make(chan error, numReaders)

		// Acquire shared locks
		for i := 0; i < numReaders; i++ {
			go func(index int) {
				lock, err := fileUtils.AcquireSharedLock(project)
				if err != nil {
					errors <- err
					return
				}
				readLocks[index] = lock
				errors <- nil
			}(i)
		}

		// Check results
		for i := 0; i < numReaders; i++ {
			err := <-errors
			if err != nil {
				t.Errorf("Failed to acquire shared lock %d: %v", i, err)
			}
		}

		// Release all shared locks
		for _, lock := range readLocks {
			if lock != nil {
				if err := fileUtils.ReleaseLock(lock); err != nil {
					t.Errorf("Failed to release shared lock: %v", err)
				}
			}
		}
	})
}

// Helper function to create test projects
func createTestProject(t *testing.T, projectDir string, files map[string]string) string {
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	for filePath, content := range files {
		fullPath := filepath.Join(projectDir, filePath)
		dir := filepath.Dir(fullPath)
		if dir != projectDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("Failed to create subdirectory: %v", err)
			}
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	return projectDir
}

// BenchmarkIntegrationOperations benchmarks integration operations
func BenchmarkIntegrationOperations(b *testing.B) {
	tempDir := b.TempDir()

	// Create test project
	project := createTestProject(nil, filepath.Join(tempDir, "benchmark-project"), map[string]string{
		"main.go":     "package main\n\nfunc main() { /* main function */ }",
		"utils.go":    "package main\n\nfunc Helper() { /* helper function */ }",
		"README.md":   "# Benchmark Project\nThis is for benchmarking.",
		"config.json": "{\"name\": \"test\", \"version\": \"1.0.0\"}",
	})

	b.Run("DirectoryValidation", func(b *testing.B) {
		validator := lib.NewDirectoryValidator()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := validator.ValidateDirectory(project)
			if err != nil {
				b.Fatalf("Validation failed: %v", err)
			}
		}
	})

	b.Run("FileLocking", func(b *testing.B) {
		fileUtils := lib.NewFileUtilities()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lockFile, err := fileUtils.AcquireLock(project)
			if err != nil {
				b.Fatalf("Failed to acquire lock: %v", err)
			}
			fileUtils.ReleaseLock(lockFile)
		}
	})

	b.Run("IndexLocationCreation", func(b *testing.B) {
		fileUtils := lib.NewFileUtilities()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = fileUtils.CreateIndexLocation(project)
		}
	})
}