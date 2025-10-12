package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"code-search/src/models"
)

// FileUtilities provides file system utility functions
type FileUtilities struct{}

// NewFileUtilities creates a new FileUtilities instance
func NewFileUtilities() *FileUtilities {
	return &FileUtilities{}
}

// ResolvePath resolves a path to an absolute path with validation
func (fu *FileUtilities) ResolvePath(path string) (string, error) {
	if path == "" {
		return os.Getwd()
	}

	// Expand tilde to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path '%s': %w", path, err)
	}

	// Clean the path
	absPath = filepath.Clean(absPath)

	return absPath, nil
}

// ValidatePathSecurity performs security checks on a path
func (fu *FileUtilities) ValidatePathSecurity(path string, allowedBasePaths []string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		// Resolve the path and check if it stays within allowed bounds
		cleanPath := filepath.Clean(path)
		for _, basePath := range allowedBasePaths {
			if strings.HasPrefix(cleanPath, basePath) {
				return nil // Path is within allowed bounds
			}
		}
		return fmt.Errorf("path traversal detected: '%s'", path)
	}

	// Additional security checks can be added here
	return nil
}

// GetDirectorySize calculates the total size of a directory
func (fu *FileUtilities) GetDirectorySize(path string) (int64, int, error) {
	var totalSize int64
	var fileCount int

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .clindex directories
		if info.IsDir() && info.Name() == ".clindex" {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}

		return nil
	})

	if err != nil {
		return 0, 0, fmt.Errorf("failed to calculate directory size: %w", err)
	}

	return totalSize, fileCount, nil
}

// EnsureDirectory creates a directory if it doesn't exist
func (fu *FileUtilities) EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// DirectoryExists checks if a directory exists
func (fu *FileUtilities) DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FileExists checks if a file exists
func (fu *FileUtilities) FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// IsAccessible checks if a directory is accessible (read + execute)
func (fu *FileUtilities) IsAccessible(path string) bool {
	// Try to open the directory
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	file.Close()

	// Check if we can list contents
	_, err = os.ReadDir(path)
	return err == nil
}

// CanWrite checks if we can write to a directory
func (fu *FileUtilities) CanWrite(path string) bool {
	testFile := filepath.Join(path, ".clindex_write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)
	return true
}

// CreateIndexLocation creates an IndexLocation for a directory
func (fu *FileUtilities) CreateIndexLocation(baseDirectory string) *models.IndexLocation {
	return models.NewIndexLocation(baseDirectory)
}

// GetSafeRelativePath gets a relative path safely
func (fu *FileUtilities) GetSafeRelativePath(basePath, targetPath string) (string, error) {
	relPath, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Security check: ensure the relative path doesn't escape the base
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("target path is outside base directory")
	}

	return relPath, nil
}

// GetFilePermissions gets file permissions in a readable format
func (fu *FileUtilities) GetFilePermissions(path string) (models.DirectoryPerms, error) {
	info, err := os.Stat(path)
	if err != nil {
		return models.DirectoryPerms{}, fmt.Errorf("failed to stat file: %w", err)
	}

	mode := info.Mode()

	// Check read permission
	canRead := mode.Perm()&0400 != 0

	// Check write permission
	canWrite := mode.Perm()&0200 != 0

	// Check execute permission
	canExec := mode.Perm()&0100 != 0

	return models.DirectoryPerms{
		CanRead:  canRead,
		CanWrite: canWrite,
		CanExec:  canExec,
	}, nil
}

// GetIndexLocation gets the index location for a directory
func (fu *FileUtilities) GetIndexLocation(directory string) (string, error) {
	// Resolve the directory path
	absDir, err := fu.ResolvePath(directory)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Create index location
	indexLoc := fu.CreateIndexLocation(absDir)

	// Ensure the .clindex directory exists
	if err := fu.EnsureDirectory(indexLoc.IndexDir); err != nil {
		return "", fmt.Errorf("failed to create index directory: %w", err)
	}

	return indexLoc.IndexFile, nil
}

// AcquireSharedLock acquires a shared file lock for read access
func (fu *FileUtilities) AcquireSharedLock(baseDirectory string) (*os.File, error) {
	// Create index location to determine lock file path
	indexLoc := fu.CreateIndexLocation(baseDirectory)
	lockFile := indexLoc.LockFile

	// Ensure directory exists
	lockDir := filepath.Dir(lockFile)
	if err := fu.EnsureDirectory(lockDir); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Create/open lock file
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire shared lock (for read access)
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to acquire shared lock: %w", err)
	}

	return file, nil
}

// AcquireLock acquires a file lock for exclusive access
func (fu *FileUtilities) AcquireLock(baseDirectory string) (*os.File, error) {
	// Create index location to determine lock file path
	indexLoc := fu.CreateIndexLocation(baseDirectory)
	lockFile := indexLoc.LockFile

	// Ensure directory exists
	lockDir := filepath.Dir(lockFile)
	if err := fu.EnsureDirectory(lockDir); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Create/open lock file
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire exclusive lock (for write access)
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to acquire exclusive lock: %w", err)
	}

	return file, nil
}

// ReleaseLock releases a file lock
func (fu *FileUtilities) ReleaseLock(lockFile *os.File) error {
	if lockFile == nil {
		return nil
	}

	// Release the lock
	err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Close the file
	err = lockFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close lock file: %w", err)
	}

	return nil
}

// IsLocked checks if a directory is locked
func (fu *FileUtilities) IsLocked(baseDirectory string) bool {
	indexLoc := fu.CreateIndexLocation(baseDirectory)
	lockFile := indexLoc.LockFile

	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return true // Assume locked if we can't open
	}
	defer file.Close()

	// Try to acquire lock non-blocking
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return true // Locked by another process
	}

	// Immediately release if we got the lock
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	return false
}

// CleanupIndexFiles removes index files from a directory
func (fu *FileUtilities) CleanupIndexFiles(indexLocation *models.IndexLocation) error {
	if indexLocation == nil {
		return fmt.Errorf("index location is nil")
	}

	// Remove the entire .clindex directory
	if fu.DirectoryExists(indexLocation.IndexDir) {
		err := os.RemoveAll(indexLocation.IndexDir)
		if err != nil {
			return fmt.Errorf("failed to remove index directory: %w", err)
		}
	}

	return nil
}

// ListIndexedDirectories lists all directories that have indexes
func (fu *FileUtilities) ListIndexedDirectories() ([]string, error) {
	var indexedDirs []string

	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check current directory for index
	if fu.FileExists(filepath.Join(currentDir, ".code-search-index")) {
		indexedDirs = append(indexedDirs, currentDir)
	}

	// Check for .clindex subdirectories
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return indexedDirs, nil // Return what we have, don't fail
	}

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == ".clindex" {
			indexedDirs = append(indexedDirs, currentDir)
			break
		}
	}

	return indexedDirs, nil
}

// GetIndexInfo gets information about an index
func (fu *FileUtilities) GetIndexInfo(directory string) (*models.IndexInfo, error) {
	indexPath, err := fu.GetIndexLocation(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to get index location: %w", err)
	}

	info := &models.IndexInfo{
		Directory: directory,
		IndexPath: indexPath,
		Exists:    fu.FileExists(indexPath),
		Locked:    fu.IsLocked(directory),
	}

	if info.Exists {
		if stat, err := os.Stat(indexPath); err == nil {
			info.Size = stat.Size()
			info.Modified = stat.ModTime()
		}
	}

	return info, nil
}

// FormatBytes formats bytes into human readable string
func (fu *FileUtilities) FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}