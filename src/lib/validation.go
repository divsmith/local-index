package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"code-search/src/models"
)

// DirectoryValidator handles validation of directory paths
type DirectoryValidator struct {
	fileUtils *FileUtilities
}

// NewDirectoryValidator creates a new directory validator
func NewDirectoryValidator() *DirectoryValidator {
	return &DirectoryValidator{
		fileUtils: NewFileUtilities(),
	}
}

// GetFileUtilities returns the file utilities instance
func (v *DirectoryValidator) GetFileUtilities() *FileUtilities {
	return v.fileUtils
}

// ValidateDirectory validates a directory path for indexing or searching
func (v *DirectoryValidator) ValidateDirectory(path string) (*models.DirectoryConfig, error) {
	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path '%s': %w", path, err)
	}

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory '%s' does not exist", absPath)
		}
		return nil, fmt.Errorf("failed to access directory '%s': %w", absPath, err)
	}

	// Check if it's actually a directory
	if !info.IsDir() {
		return nil, fmt.Errorf("path '%s' is not a directory", absPath)
	}

	// Check permissions
	perms, err := v.checkPermissions(absPath)
	if err != nil {
		return nil, fmt.Errorf("permission check failed for directory '%s': %w", absPath, err)
	}

	// Get metadata
	metadata, err := v.getDirectoryMetadata(absPath, info)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for directory '%s': %w", absPath, err)
	}

	// Apply limits
	limits := getDefaultLimits()
	if err := v.validateLimits(metadata, limits); err != nil {
		return nil, err
	}

	// Create config
	config := &models.DirectoryConfig{
		Path:         absPath,
		OriginalPath: path,
		IsDefault:    path == "." || path == "",
		Permissions:  *perms,
		Limits:       *limits,
		Metadata:     *metadata,
	}

	return config, nil
}

// checkPermissions checks directory permissions
func (v *DirectoryValidator) checkPermissions(path string) (*models.DirectoryPerms, error) {
	// Get file info for the directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access directory: %w", err)
	}

	mode := info.Mode()

	// Check read permissions by trying to open the directory
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory: %w", err)
	}
	file.Close()

	// Use file mode permissions instead of creating test files to avoid race conditions
	perms := &models.DirectoryPerms{
		CanRead:  mode.Perm()&0400 != 0, // Owner read
		CanWrite: mode.Perm()&0200 != 0, // Owner write
		CanExec:  mode.Perm()&0100 != 0, // Owner execute
	}

	return perms, nil
}

// getDirectoryMetadata gathers information about the directory
func (v *DirectoryValidator) getDirectoryMetadata(path string, info os.FileInfo) (*models.DirectoryMetadata, error) {
	// Use performance optimizer for large directories
	optimizer := NewPerformanceOptimizer()

	// Quick estimation to determine if we need optimization
	_, estimatedCount, err := optimizer.EstimateDirectorySize(path)
	if err != nil {
		// Fall back to traditional walk if estimation fails
		return v.traditionalDirectoryScan(path, info)
	}

	// If directory is large, use optimized scanning
	if estimatedCount > 10000 {
		return v.optimizedDirectoryScan(path, info, optimizer)
	}

	// Use traditional scan for smaller directories
	return v.traditionalDirectoryScan(path, info)
}

// optimizedDirectoryScan uses performance optimizations for large directories
func (v *DirectoryValidator) optimizedDirectoryScan(path string, info os.FileInfo, optimizer *PerformanceOptimizer) (*models.DirectoryMetadata, error) {
	// Optimize for large directory
	optimizer.OptimizeForLargeDirectory(50000) // Assume large if we're here

	// Configure scan options for validation
	options := DefaultScanOptions()
	options.BaseDir = path
	options.ExcludeDirs = append(options.ExcludeDirs, ".clindex")
	options.MaxFileSize = getDefaultLimits().MaxFileSize

	// Perform optimized scan
	result, err := optimizer.FastDirectoryScan(path, options)
	if err != nil {
		return nil, fmt.Errorf("optimized directory scan failed: %w", err)
	}

	metadata := &models.DirectoryMetadata{
		FileCount:    result.TotalFiles,
		TotalSize:    result.TotalSize,
		CreatedAt:    info.ModTime(),
		ModifiedAt:   info.ModTime(),
		IndexVersion: "1.0.0",
		ScanDuration: result.Duration,
		MemoryUsed:   result.MemoryUsedMB,
	}

	return metadata, nil
}

// traditionalDirectoryScan performs traditional directory walking for smaller directories
func (v *DirectoryValidator) traditionalDirectoryScan(path string, info os.FileInfo) (*models.DirectoryMetadata, error) {
	var totalSize int64
	var fileCount int64

	// Walk directory to count files and calculate size
	err := filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .clindex directories
		if fileInfo.IsDir() && fileInfo.Name() == ".clindex" {
			return filepath.SkipDir
		}

		// Skip permission test files to avoid race conditions
		if strings.HasPrefix(fileInfo.Name(), ".clindex_perm_test_") {
			return nil
		}

		if !fileInfo.IsDir() {
			fileCount++
			totalSize += fileInfo.Size()
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	metadata := &models.DirectoryMetadata{
		FileCount:    fileCount,
		TotalSize:    totalSize,
		CreatedAt:    info.ModTime(),
		ModifiedAt:   info.ModTime(),
		IndexVersion: "1.0.0",
	}

	return metadata, nil
}

// validateLimits checks if directory exceeds configured limits
func (v *DirectoryValidator) validateLimits(metadata *models.DirectoryMetadata, limits *models.DirectoryLimits) error {
	if metadata.TotalSize > limits.MaxDirectorySize {
		return fmt.Errorf("directory size (%s) exceeds limit (%s)",
			formatBytes(metadata.TotalSize), formatBytes(limits.MaxDirectorySize))
	}

	if metadata.FileCount > int64(limits.MaxFileCount) {
		return fmt.Errorf("directory contains %d files, limit is %d",
			metadata.FileCount, limits.MaxFileCount)
	}

	return nil
}

// formatBytes formats byte size into human-readable string
func formatBytes(bytes int64) string {
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

// getDefaultLimits returns default directory limits
func getDefaultLimits() *models.DirectoryLimits {
	return &models.DirectoryLimits{
		MaxDirectorySize: 1024 * 1024 * 1024, // 1GB
		MaxFileCount:     10000,              // 10,000 files
		MaxFileSize:      100 * 1024 * 1024,  // 100MB
	}
}

