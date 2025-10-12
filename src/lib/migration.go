package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code-search/src/models"
)

// IndexMigrator handles migration of index files between different versions
type IndexMigrator struct {
	fileUtils *FileUtilities
}

// NewIndexMigrator creates a new index migrator
func NewIndexMigrator() *IndexMigrator {
	return &IndexMigrator{
		fileUtils: NewFileUtilities(),
	}
}

// LegacyIndexMetadata represents metadata from old index files
type LegacyIndexMetadata struct {
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	FileCount   int       `json:"file_count"`
	ChunkCount  int       `json:"chunk_count"`
	Directory   string    `json:"directory"`
	IndexerType string    `json:"indexer_type"`
}

// MigrationResult represents the result of a migration operation
type MigrationResult struct {
	Success        bool     `json:"success"`
	MigratedFiles  []string `json:"migrated_files"`
	Errors         []string `json:"errors"`
	SourcePath     string   `json:"source_path"`
	TargetPath     string   `json:"target_path"`
	FilesMigrated  int      `json:"files_migrated"`
	BytesMigrated  int64    `json:"bytes_migrated"`
	Duration       string   `json:"duration"`
}

// DetectLegacyIndexes scans for legacy index files that need migration
func (m *IndexMigrator) DetectLegacyIndexes(directory string) ([]string, error) {
	var legacyIndexes []string

	// Check for legacy .code-search-index file
	legacyIndexFile := filepath.Join(directory, ".code-search-index")
	if m.fileUtils.FileExists(legacyIndexFile) {
		legacyIndexes = append(legacyIndexes, legacyIndexFile)
	}

	// Check for legacy .code-search-index.db file
	legacyIndexDB := filepath.Join(directory, ".code-search-index.db")
	if m.fileUtils.FileExists(legacyIndexDB) {
		legacyIndexes = append(legacyIndexes, legacyIndexDB)
	}

	// Check for other legacy variants
	legacyVariants := []string{
		".code-search",
		".code-index",
		".search-index",
	}

	for _, variant := range legacyVariants {
		variantPath := filepath.Join(directory, variant)
		if m.fileUtils.FileExists(variantPath) {
			legacyIndexes = append(legacyIndexes, variantPath)
		}
	}

	return legacyIndexes, nil
}

// NeedsMigration checks if a directory contains legacy indexes that need migration
func (m *IndexMigrator) NeedsMigration(directory string) (bool, error) {
	// If new-style index already exists, no migration needed
	newIndexLoc := m.fileUtils.CreateIndexLocation(directory)
	if m.fileUtils.DirectoryExists(newIndexLoc.IndexDir) {
		// Check if new index has content
		entries, err := os.ReadDir(newIndexLoc.IndexDir)
		if err == nil && len(entries) > 0 {
			return false, nil // New index already exists and has content
		}
	}

	// Check for legacy indexes
	legacyIndexes, err := m.DetectLegacyIndexes(directory)
	if err != nil {
		return false, fmt.Errorf("failed to detect legacy indexes: %w", err)
	}

	return len(legacyIndexes) > 0, nil
}

// MigrateIndex migrates legacy index files to the new directory structure
func (m *IndexMigrator) MigrateIndex(directory string, force bool) (*MigrationResult, error) {
	start := time.Now()
	result := &MigrationResult{
		SourcePath: directory,
		Success:    true,
	}

	// Resolve directory path
	absDir, err := m.fileUtils.ResolvePath(directory)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to resolve directory: %v", err))
		return result, err
	}

	result.SourcePath = absDir

	// Check if migration is needed
	needsMigration, err := m.NeedsMigration(absDir)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to check migration need: %v", err))
		return result, err
	}

	if !needsMigration {
		return result, nil // Nothing to migrate
	}

	// Detect legacy indexes
	legacyIndexes, err := m.DetectLegacyIndexes(absDir)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to detect legacy indexes: %v", err))
		return result, err
	}

	if len(legacyIndexes) == 0 {
		return result, nil // Nothing to migrate
	}

	// Create new index location
	newIndexLoc := m.fileUtils.CreateIndexLocation(absDir)

	// Ensure target directory exists
	if err := m.fileUtils.EnsureDirectory(newIndexLoc.IndexDir); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to create target directory: %v", err))
		return result, err
	}

	result.TargetPath = newIndexLoc.IndexDir

	// Check if target already has content and we're not forcing
	if !force {
		entries, err := os.ReadDir(newIndexLoc.IndexDir)
		if err == nil && len(entries) > 0 {
			result.Success = false
			result.Errors = append(result.Errors, "target index directory already exists and has content")
			return result, fmt.Errorf("target index directory already exists and has content (use --force to overwrite)")
		}
	}

	// Migrate each legacy index file
	for _, legacyFile := range legacyIndexes {
		if err := m.migrateLegacyFile(legacyFile, newIndexLoc, result); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("failed to migrate %s: %v", legacyFile, err))
			continue
		}
		result.MigratedFiles = append(result.MigratedFiles, legacyFile)
	}

	// Create metadata file for the migrated index
	if err := m.createMigrationMetadata(newIndexLoc, legacyIndexes, result); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to create migration metadata: %v", err))
	}

	result.Duration = time.Since(start).String()

	// Cleanup legacy files if migration was successful
	if result.Success && len(result.MigratedFiles) > 0 {
		if err := m.cleanupLegacyFiles(legacyIndexes); err != nil {
			// Don't fail the migration, but record the error
			result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to cleanup legacy files: %v", err))
		}
	}

	return result, nil
}

// migrateLegacyFile migrates a single legacy index file
func (m *IndexMigrator) migrateLegacyFile(legacyFile string, newIndexLoc *models.IndexLocation, result *MigrationResult) error {
	// Get file info
	info, err := os.Stat(legacyFile)
	if err != nil {
		return fmt.Errorf("failed to stat legacy file: %w", err)
	}

	// Determine target file name
	targetFile := filepath.Join(newIndexLoc.IndexDir, filepath.Base(legacyFile))
	if filepath.Base(legacyFile) == ".code-search-index" {
		targetFile = newIndexLoc.IndexFile
	} else if filepath.Base(legacyFile) == ".code-search-index.db" {
		targetFile = newIndexLoc.MetadataFile
	}

	// Copy the file
	if err := m.copyFile(legacyFile, targetFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	result.FilesMigrated++
	result.BytesMigrated += info.Size()

	return nil
}

// createMigrationMetadata creates metadata file for the migrated index
func (m *IndexMigrator) createMigrationMetadata(newIndexLoc *models.IndexLocation, legacyFiles []string, result *MigrationResult) error {
	metadata := &models.IndexMetadata{
		Version:     "2.0.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Migrated:    true,
		MigrationDate: time.Now(),
		LegacyFiles: legacyFiles,
	}

	// Try to extract metadata from legacy files
	for _, legacyFile := range legacyFiles {
		if err := m.extractLegacyMetadata(legacyFile, metadata); err != nil {
			// Don't fail, just record the error
			result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to extract metadata from %s: %v", legacyFile, err))
		}
	}

	// Write metadata file
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(newIndexLoc.MetadataFile, metadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// extractLegacyMetadata attempts to extract metadata from a legacy index file
func (m *IndexMigrator) extractLegacyMetadata(legacyFile string, metadata *models.IndexMetadata) error {
	// Try to read as JSON metadata file
	if filepath.Ext(legacyFile) == ".json" || filepath.Base(legacyFile) == ".code-search-index" {
		file, err := os.Open(legacyFile)
		if err != nil {
			return err
		}
		defer file.Close()

		var legacyMeta LegacyIndexMetadata
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&legacyMeta); err != nil {
			return err // Not a valid JSON metadata file
		}

		// Extract relevant information
		metadata.FileCount = legacyMeta.FileCount
		metadata.ChunkCount = legacyMeta.ChunkCount
		metadata.Directory = legacyMeta.Directory
		metadata.IndexerType = legacyMeta.IndexerType
		metadata.LegacyVersion = legacyMeta.Version
	}

	return nil
}

// copyFile copies a file from source to destination
func (m *IndexMigrator) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// cleanupLegacyFiles removes legacy index files after successful migration
func (m *IndexMigrator) cleanupLegacyFiles(legacyFiles []string) error {
	var errors []string

	for _, legacyFile := range legacyFiles {
		if err := os.Remove(legacyFile); err != nil {
			errors = append(errors, fmt.Sprintf("failed to remove %s: %v", legacyFile, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// RollbackMigration attempts to rollback a migration by restoring legacy files
func (m *IndexMigrator) RollbackMigration(directory string) error {
	// This is a complex operation that would require backup of legacy files
	// For now, we'll just remove the new index directory
	newIndexLoc := m.fileUtils.CreateIndexLocation(directory)

	if m.fileUtils.DirectoryExists(newIndexLoc.IndexDir) {
		return os.RemoveAll(newIndexLoc.IndexDir)
	}

	return nil
}

// GetMigrationStatus returns the migration status of a directory
func (m *IndexMigrator) GetMigrationStatus(directory string) (string, error) {
	absDir, err := m.fileUtils.ResolvePath(directory)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory: %w", err)
	}

	newIndexLoc := m.fileUtils.CreateIndexLocation(absDir)

	// Check if new index exists
	if m.fileUtils.FileExists(newIndexLoc.MetadataFile) {
		// Read metadata to check if it's migrated
		file, err := os.Open(newIndexLoc.MetadataFile)
		if err != nil {
			return "unknown", fmt.Errorf("failed to read metadata: %w", err)
		}
		defer file.Close()

		var metadata models.IndexMetadata
		if err := json.NewDecoder(file).Decode(&metadata); err != nil {
			return "unknown", fmt.Errorf("failed to decode metadata: %w", err)
		}

		if metadata.Migrated {
			return "migrated", nil
		}
		return "new", nil
	}

	// Check for legacy indexes
	needsMigration, err := m.NeedsMigration(absDir)
	if err != nil {
		return "unknown", err
	}

	if needsMigration {
		return "legacy", nil
	}

	return "none", nil
}