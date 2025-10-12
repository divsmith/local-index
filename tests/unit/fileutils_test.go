package unit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"code-search/src/lib"
)

// TestFileUtilities_ResolvePath tests path resolution functionality
func TestFileUtilities_ResolvePath(t *testing.T) {
	fileUtils := lib.NewFileUtilities()

	t.Run("Empty path returns current directory", func(t *testing.T) {
		path, err := fileUtils.ResolvePath("")
		if err != nil {
			t.Errorf("Expected no error for empty path, got: %v", err)
		}

		if !filepath.IsAbs(path) {
			t.Error("Expected absolute path for empty input")
		}
	})

	t.Run("Absolute path is unchanged", func(t *testing.T) {
		tempDir := t.TempDir()
		resolved, err := fileUtils.ResolvePath(tempDir)
		if err != nil {
			t.Errorf("Expected no error for absolute path, got: %v", err)
		}

		if resolved != tempDir {
			t.Errorf("Expected %s, got %s", tempDir, resolved)
		}
	})

	t.Run("Relative path is resolved", func(t *testing.T) {
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		resolved, err := fileUtils.ResolvePath(".")
		if err != nil {
			t.Errorf("Expected no error for relative path, got: %v", err)
		}

		if !filepath.IsAbs(resolved) {
			t.Error("Expected absolute path for relative input")
		}

		if resolved != tempDir {
			t.Errorf("Expected %s, got %s", tempDir, resolved)
		}
	})

	t.Run("Path with .. is cleaned", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		resolved, err := fileUtils.ResolvePath(subDir + "/..")
		if err != nil {
			t.Errorf("Expected no error for path with .., got: %v", err)
		}

		if resolved != tempDir {
			t.Errorf("Expected %s after cleaning .., got %s", tempDir, resolved)
		}
	})
}

// TestFileUtilities_ValidatePathSecurity tests path security validation
func TestFileUtilities_ValidatePathSecurity(t *testing.T) {
	fileUtils := lib.NewFileUtilities()

	t.Run("Safe path passes validation", func(t *testing.T) {
		tempDir := t.TempDir()
		safePath := filepath.Join(tempDir, "safe", "path")

		err := fileUtils.ValidatePathSecurity(safePath, []string{tempDir})
		if err != nil {
			t.Errorf("Expected no error for safe path, got: %v", err)
		}
	})

	t.Run("Path traversal is detected", func(t *testing.T) {
		tempDir := t.TempDir()
		traversalPath := filepath.Join(tempDir, "safe", "..", "unsafe")

		err := fileUtils.ValidatePathSecurity(traversalPath, []string{tempDir})
		if err == nil {
			t.Error("Expected error for path traversal attempt")
		}

		expectedMsg := "path traversal detected"
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to contain '%s', got: %v", expectedMsg, err)
		}
	})

	t.Run("Path within allowed base passes", func(t *testing.T) {
		tempDir := t.TempDir()
		allowedPath := filepath.Join(tempDir, "allowed")

		err := fileUtils.ValidatePathSecurity(allowedPath, []string{tempDir})
		if err != nil {
			t.Errorf("Expected no error for path within allowed base, got: %v", err)
		}
	})

	t.Run("Path outside allowed base fails", func(t *testing.T) {
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()

		outsidePath := filepath.Join(tempDir2, "outside")
		err := fileUtils.ValidatePathSecurity(outsidePath, []string{tempDir1})
		if err == nil {
			t.Error("Expected error for path outside allowed base")
		}
	})
}

// TestFileUtilities_DirectoryOperations tests directory-related operations
func TestFileUtilities_DirectoryOperations(t *testing.T) {
	fileUtils := lib.NewFileUtilities()

	t.Run("DirectoryExists", func(t *testing.T) {
		tempDir := t.TempDir()

		if !fileUtils.DirectoryExists(tempDir) {
			t.Error("Expected DirectoryExists to return true for existing directory")
		}

		if fileUtils.DirectoryExists("/non/existent/path") {
			t.Error("Expected DirectoryExists to return false for non-existent directory")
		}

		// Test with file path
		tempFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		if fileUtils.DirectoryExists(tempFile) {
			t.Error("Expected DirectoryExists to return false for file path")
		}
	})

	t.Run("FileExists", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "test.txt")

		if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		if !fileUtils.FileExists(tempFile) {
			t.Error("Expected FileExists to return true for existing file")
		}

		if fileUtils.FileExists("/non/existent/file.txt") {
			t.Error("Expected FileExists to return false for non-existent file")
		}

		if fileUtils.FileExists(tempDir) {
			t.Error("Expected FileExists to return false for directory path")
		}
	})

	t.Run("EnsureDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		nestedDir := filepath.Join(tempDir, "nested", "deep", "directory")

		if err := fileUtils.EnsureDirectory(nestedDir); err != nil {
			t.Errorf("Expected no error creating nested directory, got: %v", err)
		}

		if !fileUtils.DirectoryExists(nestedDir) {
			t.Error("Expected nested directory to exist after EnsureDirectory")
		}
	})

	t.Run("IsAccessible", func(t *testing.T) {
		tempDir := t.TempDir()

		if !fileUtils.IsAccessible(tempDir) {
			t.Error("Expected temp directory to be accessible")
		}
	})

	t.Run("CanWrite", func(t *testing.T) {
		tempDir := t.TempDir()

		if !fileUtils.CanWrite(tempDir) {
			t.Error("Expected temp directory to be writable")
		}
	})
}

// TestFileUtilities_Locking tests file locking functionality
func TestFileUtilities_Locking(t *testing.T) {
	fileUtils := lib.NewFileUtilities()

	t.Run("Acquire and release lock", func(t *testing.T) {
		tempDir := t.TempDir()

		lockFile, err := fileUtils.AcquireLock(tempDir)
		if err != nil {
			t.Errorf("Expected no error acquiring lock, got: %v", err)
		}
		defer fileUtils.ReleaseLock(lockFile)

		if lockFile == nil {
			t.Error("Expected lock file to be non-nil")
		}

		// Directory should appear locked
		if !fileUtils.IsLocked(tempDir) {
			t.Error("Expected directory to be locked while holding lock")
		}

		// Release the lock
		if err := fileUtils.ReleaseLock(lockFile); err != nil {
			t.Errorf("Expected no error releasing lock, got: %v", err)
		}

		// Directory should no longer be locked
		if fileUtils.IsLocked(tempDir) {
			t.Error("Expected directory to be unlocked after releasing lock")
		}
	})

	t.Run("Shared lock for read access", func(t *testing.T) {
		tempDir := t.TempDir()

		lockFile, err := fileUtils.AcquireSharedLock(tempDir)
		if err != nil {
			t.Errorf("Expected no error acquiring shared lock, got: %v", err)
		}
		defer fileUtils.ReleaseLock(lockFile)

		if lockFile == nil {
			t.Error("Expected lock file to be non-nil")
		}

		// Should be able to acquire multiple shared locks (in theory)
		secondLock, err := fileUtils.AcquireSharedLock(tempDir)
		if err != nil {
			// This might fail on some systems, which is acceptable
			t.Logf("Second shared lock failed (possibly expected): %v", err)
		} else {
			fileUtils.ReleaseLock(secondLock)
		}
	})

	t.Run("Lock conflict detection", func(t *testing.T) {
		tempDir := t.TempDir()

		// Acquire first lock
		firstLock, err := fileUtils.AcquireLock(tempDir)
		if err != nil {
			t.Fatalf("Failed to acquire first lock: %v", err)
		}
		defer fileUtils.ReleaseLock(firstLock)

		// Try to acquire second exclusive lock (should fail)
		secondLock, err := fileUtils.AcquireLock(tempDir)
		if err == nil {
			t.Error("Expected error when trying to acquire conflicting lock")
			if secondLock != nil {
				fileUtils.ReleaseLock(secondLock)
			}
		}
	})
}

// TestFileUtilities_IndexLocation tests index location management
func TestFileUtilities_IndexLocation(t *testing.T) {
	fileUtils := lib.NewFileUtilities()

	t.Run("GetIndexLocation creates directory structure", func(t *testing.T) {
		tempDir := t.TempDir()

		indexPath, err := fileUtils.GetIndexLocation(tempDir)
		if err != nil {
			t.Errorf("Expected no error getting index location, got: %v", err)
		}

		if indexPath == "" {
			t.Error("Expected non-empty index path")
		}

		// Check that .clindex directory was created
		clindexDir := filepath.Join(tempDir, ".clindex")
		if !fileUtils.DirectoryExists(clindexDir) {
			t.Error("Expected .clindex directory to be created")
		}

		expectedIndexPath := filepath.Join(clindexDir, "index.db")
		if indexPath != expectedIndexPath {
			t.Errorf("Expected index path %s, got %s", expectedIndexPath, indexPath)
		}
	})

	t.Run("CreateIndexLocation", func(t *testing.T) {
		tempDir := t.TempDir()

		indexLoc := fileUtils.CreateIndexLocation(tempDir)
		if indexLoc == nil {
			t.Error("Expected non-nil IndexLocation")
		}

		if indexLoc.BaseDirectory != tempDir {
			t.Errorf("Expected base directory %s, got %s", tempDir, indexLoc.BaseDirectory)
		}

		expectedIndexDir := filepath.Join(tempDir, ".clindex")
		if indexLoc.IndexDir != expectedIndexDir {
			t.Errorf("Expected index dir %s, got %s", expectedIndexDir, indexLoc.IndexDir)
		}
	})

	t.Run("GetIndexInfo", func(t *testing.T) {
		tempDir := t.TempDir()

		// Get index location to create directory structure
		indexPath, err := fileUtils.GetIndexLocation(tempDir)
		if err != nil {
			t.Fatalf("Failed to get index location: %v", err)
		}

		// Create a dummy index file
		if err := os.WriteFile(indexPath, []byte("dummy index data"), 0644); err != nil {
			t.Fatalf("Failed to create dummy index file: %v", err)
		}

		info, err := fileUtils.GetIndexInfo(tempDir)
		if err != nil {
			t.Errorf("Expected no error getting index info, got: %v", err)
		}

		if info == nil {
			t.Error("Expected non-nil index info")
		}

		if !info.Exists {
			t.Error("Expected index to exist")
		}

		if info.Directory != tempDir {
			t.Errorf("Expected directory %s, got %s", tempDir, info.Directory)
		}
	})
}

// TestFileUtilities_DirectorySize tests directory size calculation
func TestFileUtilities_DirectorySize(t *testing.T) {
	fileUtils := lib.NewFileUtilities()

	t.Run("Empty directory", func(t *testing.T) {
		tempDir := t.TempDir()

		size, count, err := fileUtils.GetDirectorySize(tempDir)
		if err != nil {
			t.Errorf("Expected no error for empty directory, got: %v", err)
		}

		if size != 0 {
			t.Errorf("Expected size 0 for empty directory, got %d", size)
		}

		if count != 0 {
			t.Errorf("Expected count 0 for empty directory, got %d", count)
		}
	})

	t.Run("Directory with files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create test files
		contents := []string{"small", "medium content", "larger content with more text"}
		expectedSize := int64(0)

		for i, content := range contents {
			filePath := filepath.Join(tempDir, fmt.Sprintf("file_%d.txt", i))
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			expectedSize += int64(len(content))
		}

		size, count, err := fileUtils.GetDirectorySize(tempDir)
		if err != nil {
			t.Errorf("Expected no error for directory with files, got: %v", err)
		}

		if size != expectedSize {
			t.Errorf("Expected size %d, got %d", expectedSize, size)
		}

		if count != len(contents) {
			t.Errorf("Expected count %d, got %d", len(contents), count)
		}
	})

	t.Run("Skips .clindex directories", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create .clindex directory with files
		clindexDir := filepath.Join(tempDir, ".clindex")
		if err := os.MkdirAll(clindexDir, 0755); err != nil {
			t.Fatalf("Failed to create .clindex directory: %v", err)
		}

		// Add files to .clindex (should be ignored)
		for i := 0; i < 3; i++ {
			filePath := filepath.Join(clindexDir, fmt.Sprintf("ignore_%d.db", i))
			if err := os.WriteFile(filePath, []byte("ignore me"), 0644); err != nil {
				t.Fatalf("Failed to create ignored file: %v", err)
			}
		}

		// Add regular files (should be counted)
		for i := 0; i < 2; i++ {
			filePath := filepath.Join(tempDir, fmt.Sprintf("count_%d.txt", i))
			if err := os.WriteFile(filePath, []byte("count me"), 0644); err != nil {
				t.Fatalf("Failed to create counted file: %v", err)
			}
		}

		size, count, err := fileUtils.GetDirectorySize(tempDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Should only count regular files, not .clindex files
		if count != 2 {
			t.Errorf("Expected count 2 (excluding .clindex), got %d", count)
		}

		if size == 0 {
			t.Error("Expected non-zero size for regular files")
		}
	})
}

// TestFileUtilities_UtilityFunctions tests miscellaneous utility functions
func TestFileUtilities_UtilityFunctions(t *testing.T) {
	fileUtils := lib.NewFileUtilities()

	t.Run("GetSafeRelativePath", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		relPath, err := fileUtils.GetSafeRelativePath(tempDir, subDir)
		if err != nil {
			t.Errorf("Expected no error for safe relative path, got: %v", err)
		}

		if relPath != "subdir" {
			t.Errorf("Expected relative path 'subdir', got '%s'", relPath)
		}
	})

	t.Run("GetSafeRelativePath - path traversal prevention", func(t *testing.T) {
		tempDir := t.TempDir()
		outsidePath := filepath.Dir(tempDir) // Parent directory

		_, err := fileUtils.GetSafeRelativePath(tempDir, outsidePath)
		if err == nil {
			t.Error("Expected error for path outside base directory")
		}
	})

	t.Run("GetFilePermissions", func(t *testing.T) {
		tempDir := t.TempDir()

		perms, err := fileUtils.GetFilePermissions(tempDir)
		if err != nil {
			t.Errorf("Expected no error getting file permissions, got: %v", err)
		}

		// Temp directory should be readable, writable, and executable
		if !perms.CanRead {
			t.Error("Expected temp directory to be readable")
		}

		if !perms.CanWrite {
			t.Error("Expected temp directory to be writable")
		}

		if !perms.CanExec {
			t.Error("Expected temp directory to be executable")
		}
	})

	t.Run("FormatBytes", func(t *testing.T) {
		tests := []struct {
			bytes    int64
			expected string
		}{
			{500, "500 B"},
			{1024, "1.0 KB"},
			{1536, "1.5 KB"},
			{1048576, "1.0 MB"},
			{1073741824, "1.0 GB"},
		}

		for _, test := range tests {
			result := fileUtils.FormatBytes(test.bytes)
			if result != test.expected {
				t.Errorf("FormatBytes(%d): expected '%s', got '%s'", test.bytes, test.expected, result)
			}
		}
	})

	t.Run("CleanupIndexFiles", func(t *testing.T) {
		tempDir := t.TempDir()
		indexLoc := fileUtils.CreateIndexLocation(tempDir)

		// Create index directory structure
		if err := fileUtils.EnsureDirectory(indexLoc.IndexDir); err != nil {
			t.Fatalf("Failed to create index directory: %v", err)
		}

		// Create some index files
		files := []string{"index.db", "metadata.json", "lock"}
		for _, file := range files {
			filePath := filepath.Join(indexLoc.IndexDir, file)
			if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		// Verify files exist
		if !fileUtils.DirectoryExists(indexLoc.IndexDir) {
			t.Error("Expected index directory to exist")
		}

		// Cleanup
		if err := fileUtils.CleanupIndexFiles(indexLoc); err != nil {
			t.Errorf("Expected no error cleaning up index files, got: %v", err)
		}

		// Verify cleanup
		if fileUtils.DirectoryExists(indexLoc.IndexDir) {
			t.Error("Expected index directory to be removed after cleanup")
		}
	})
}

// BenchmarkFileUtilities_GetDirectorySize benchmarks directory size calculation
func BenchmarkFileUtilities_GetDirectorySize(b *testing.B) {
	fileUtils := lib.NewFileUtilities()

	// Create a temporary directory with many files
	tempDir := b.TempDir()

	for i := 0; i < 1000; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("bench_file_%d.txt", i))
		content := []byte(fmt.Sprintf("benchmark content %d", i))
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := fileUtils.GetDirectorySize(tempDir)
		if err != nil {
			b.Fatalf("GetDirectorySize failed: %v", err)
		}
	}
}