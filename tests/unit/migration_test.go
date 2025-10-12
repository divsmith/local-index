package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code-search/src/lib"
	"code-search/src/models"
)

// TestIndexMigrator_DetectLegacyIndexes tests legacy index detection
func TestIndexMigrator_DetectLegacyIndexes(t *testing.T) {
	migrator := lib.NewIndexMigrator()

	t.Run("No legacy indexes", func(t *testing.T) {
		tempDir := t.TempDir()

		legacyIndexes, err := migrator.DetectLegacyIndexes(tempDir)
		if err != nil {
			t.Errorf("Expected no error detecting legacy indexes, got: %v", err)
		}

		if len(legacyIndexes) != 0 {
			t.Errorf("Expected 0 legacy indexes, got %d", len(legacyIndexes))
		}
	})

	t.Run("Detect .code-search-index file", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyFile := filepath.Join(tempDir, ".code-search-index")

		// Create legacy index file
		if err := os.WriteFile(legacyFile, []byte("legacy index data"), 0644); err != nil {
			t.Fatalf("Failed to create legacy index file: %v", err)
		}

		legacyIndexes, err := migrator.DetectLegacyIndexes(tempDir)
		if err != nil {
			t.Errorf("Expected no error detecting legacy indexes, got: %v", err)
		}

		if len(legacyIndexes) != 1 {
			t.Errorf("Expected 1 legacy index, got %d", len(legacyIndexes))
		}

		if legacyIndexes[0] != legacyFile {
			t.Errorf("Expected legacy index path %s, got %s", legacyFile, legacyIndexes[0])
		}
	})

	t.Run("Detect .code-search-index.db file", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyFile := filepath.Join(tempDir, ".code-search-index.db")

		// Create legacy index database file
		if err := os.WriteFile(legacyFile, []byte("legacy db data"), 0644); err != nil {
			t.Fatalf("Failed to create legacy db file: %v", err)
		}

		legacyIndexes, err := migrator.DetectLegacyIndexes(tempDir)
		if err != nil {
			t.Errorf("Expected no error detecting legacy indexes, got: %v", err)
		}

		if len(legacyIndexes) != 1 {
			t.Errorf("Expected 1 legacy index, got %d", len(legacyIndexes))
		}

		if legacyIndexes[0] != legacyFile {
			t.Errorf("Expected legacy index path %s, got %s", legacyFile, legacyIndexes[0])
		}
	})

	t.Run("Detect multiple legacy variants", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create multiple legacy files
		legacyFiles := []string{
			".code-search-index",
			".code-search-index.db",
			".code-index",
		}

		for _, file := range legacyFiles {
			filePath := filepath.Join(tempDir, file)
			if err := os.WriteFile(filePath, []byte("legacy data"), 0644); err != nil {
				t.Fatalf("Failed to create legacy file %s: %v", file, err)
			}
		}

		legacyIndexes, err := migrator.DetectLegacyIndexes(tempDir)
		if err != nil {
			t.Errorf("Expected no error detecting legacy indexes, got: %v", err)
		}

		if len(legacyIndexes) != len(legacyFiles) {
			t.Errorf("Expected %d legacy indexes, got %d", len(legacyFiles), len(legacyIndexes))
		}
	})
}

// TestIndexMigrator_NeedsMigration tests migration need detection
func TestIndexMigrator_NeedsMigration(t *testing.T) {
	migrator := lib.NewIndexMigrator()

	t.Run("Fresh directory needs no migration", func(t *testing.T) {
		tempDir := t.TempDir()

		needsMigration, err := migrator.NeedsMigration(tempDir)
		if err != nil {
			t.Errorf("Expected no error checking migration need, got: %v", err)
		}

		if needsMigration {
			t.Error("Expected fresh directory to not need migration")
		}
	})

	t.Run("Directory with legacy indexes needs migration", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyFile := filepath.Join(tempDir, ".code-search-index")

		// Create legacy index file
		if err := os.WriteFile(legacyFile, []byte("legacy data"), 0644); err != nil {
			t.Fatalf("Failed to create legacy file: %v", err)
		}

		needsMigration, err := migrator.NeedsMigration(tempDir)
		if err != nil {
			t.Errorf("Expected no error checking migration need, got: %v", err)
		}

		if !needsMigration {
			t.Error("Expected directory with legacy indexes to need migration")
		}
	})

	t.Run("Directory with new index needs no migration", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create new-style index directory
		clindexDir := filepath.Join(tempDir, ".clindex")
		if err := os.MkdirAll(clindexDir, 0755); err != nil {
			t.Fatalf("Failed to create .clindex directory: %v", err)
		}

		// Create metadata file
		metadata := models.IndexMetadata{
			Version:   "2.0.0",
			CreatedAt: time.Now(),
			Migrated:  false,
		}

		metadataFile := filepath.Join(clindexDir, "metadata.json")
		metadataBytes, _ := json.Marshal(metadata)
		if err := os.WriteFile(metadataFile, metadataBytes, 0644); err != nil {
			t.Fatalf("Failed to create metadata file: %v", err)
		}

		needsMigration, err := migrator.NeedsMigration(tempDir)
		if err != nil {
			t.Errorf("Expected no error checking migration need, got: %v", err)
		}

		if needsMigration {
			t.Error("Expected directory with new index to not need migration")
		}
	})
}

// TestIndexMigrator_MigrateIndex tests the migration process
func TestIndexMigrator_MigrateIndex(t *testing.T) {
	migrator := lib.NewIndexMigrator()

	t.Run("Migrate single legacy file", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyFile := filepath.Join(tempDir, ".code-search-index")
		legacyContent := []byte("legacy index content")

		// Create legacy index file
		if err := os.WriteFile(legacyFile, legacyContent, 0644); err != nil {
			t.Fatalf("Failed to create legacy file: %v", err)
		}

		// Perform migration
		result, err := migrator.MigrateIndex(tempDir, false)
		if err != nil {
			t.Errorf("Expected no error migrating index, got: %v", err)
		}

		if !result.Success {
			t.Error("Expected migration to be successful")
		}

		if len(result.MigratedFiles) != 1 {
			t.Errorf("Expected 1 migrated file, got %d", len(result.MigratedFiles))
		}

		if result.FilesMigrated != 1 {
			t.Errorf("Expected 1 file migrated, got %d", result.FilesMigrated)
		}

		if result.BytesMigrated != int64(len(legacyContent)) {
			t.Errorf("Expected %d bytes migrated, got %d", len(legacyContent), result.BytesMigrated)
		}

		// Check that new index file exists
		newIndexFile := filepath.Join(tempDir, ".clindex", ".code-search-index")
		if !fileExists(newIndexFile) {
			t.Error("Expected new index file to exist after migration")
		}

		// Check that content was copied correctly
		newContent, err := os.ReadFile(newIndexFile)
		if err != nil {
			t.Fatalf("Failed to read migrated file: %v", err)
		}

		if string(newContent) != string(legacyContent) {
			t.Error("Expected migrated content to match original")
		}
	})

	t.Run("Migrate multiple legacy files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create multiple legacy files
		legacyFiles := map[string][]byte{
			".code-search-index":     []byte("index data"),
			".code-search-index.db":  []byte("database data"),
			".code-index":            []byte("old index data"),
		}

		for fileName, content := range legacyFiles {
			filePath := filepath.Join(tempDir, fileName)
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				t.Fatalf("Failed to create legacy file %s: %v", fileName, err)
			}
		}

		// Perform migration
		result, err := migrator.MigrateIndex(tempDir, false)
		if err != nil {
			t.Errorf("Expected no error migrating multiple files, got: %v", err)
		}

		if !result.Success {
			t.Error("Expected migration to be successful")
		}

		if len(result.MigratedFiles) != len(legacyFiles) {
			t.Errorf("Expected %d migrated files, got %d", len(legacyFiles), len(result.MigratedFiles))
		}

		// Check that all files were migrated
		clindexDir := filepath.Join(tempDir, ".clindex")
		for fileName := range legacyFiles {
			migratedFile := filepath.Join(clindexDir, fileName)
			if !fileExists(migratedFile) {
				t.Errorf("Expected migrated file %s to exist", migratedFile)
			}
		}
	})

	t.Run("Migration with force flag", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyFile := filepath.Join(tempDir, ".code-search-index")

		// Create legacy file
		if err := os.WriteFile(legacyFile, []byte("legacy"), 0644); err != nil {
			t.Fatalf("Failed to create legacy file: %v", err)
		}

		// Create existing new index directory
		clindexDir := filepath.Join(tempDir, ".clindex")
		if err := os.MkdirAll(clindexDir, 0755); err != nil {
			t.Fatalf("Failed to create .clindex directory: %v", err)
		}

		// Create existing index file
		existingFile := filepath.Join(clindexDir, ".code-search-index")
		if err := os.WriteFile(existingFile, []byte("existing"), 0644); err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}

		// Try migration without force (should fail)
		_, err := migrator.MigrateIndex(tempDir, false)
		if err == nil {
			t.Error("Expected error when migrating without force to existing directory")
		}

		// Try migration with force (should succeed)
		result, err := migrator.MigrateIndex(tempDir, true)
		if err != nil {
			t.Errorf("Expected no error when migrating with force, got: %v", err)
		}

		if !result.Success {
			t.Error("Expected forced migration to be successful")
		}
	})

	t.Run("Migration creates metadata", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyFile := filepath.Join(tempDir, ".code-search-index")

		// Create legacy file
		if err := os.WriteFile(legacyFile, []byte("legacy"), 0644); err != nil {
			t.Fatalf("Failed to create legacy file: %v", err)
		}

		// Perform migration
		result, err := migrator.MigrateIndex(tempDir, false)
		if err != nil {
			t.Errorf("Expected no error migrating, got: %v", err)
		}

		// Check that metadata file exists
		metadataFile := filepath.Join(tempDir, ".clindex", "metadata.json")
		if !fileExists(metadataFile) {
			t.Error("Expected metadata file to exist after migration")
		}

		// Check metadata content
		metadataBytes, err := os.ReadFile(metadataFile)
		if err != nil {
			t.Fatalf("Failed to read metadata file: %v", err)
		}

		var metadata models.IndexMetadata
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			t.Fatalf("Failed to unmarshal metadata: %v", err)
		}

		if !metadata.Migrated {
			t.Error("Expected metadata to indicate migration occurred")
		}

		if metadata.Version != "2.0.0" {
			t.Errorf("Expected version 2.0.0, got %s", metadata.Version)
		}

		if len(metadata.LegacyFiles) != 1 {
			t.Errorf("Expected 1 legacy file in metadata, got %d", len(metadata.LegacyFiles))
		}
	})
}

// TestIndexMigrator_GetMigrationStatus tests migration status detection
func TestIndexMigrator_GetMigrationStatus(t *testing.T) {
	migrator := lib.NewIndexMigrator()

	t.Run("Fresh directory status", func(t *testing.T) {
		tempDir := t.TempDir()

		status, err := migrator.GetMigrationStatus(tempDir)
		if err != nil {
			t.Errorf("Expected no error getting migration status, got: %v", err)
		}

		if status != "none" {
			t.Errorf("Expected status 'none', got '%s'", status)
		}
	})

	t.Run("Legacy directory status", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyFile := filepath.Join(tempDir, ".code-search-index")

		// Create legacy file
		if err := os.WriteFile(legacyFile, []byte("legacy"), 0644); err != nil {
			t.Fatalf("Failed to create legacy file: %v", err)
		}

		status, err := migrator.GetMigrationStatus(tempDir)
		if err != nil {
			t.Errorf("Expected no error getting migration status, got: %v", err)
		}

		if status != "legacy" {
			t.Errorf("Expected status 'legacy', got '%s'", status)
		}
	})

	t.Run("Migrated directory status", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create new-style index with migration metadata
		clindexDir := filepath.Join(tempDir, ".clindex")
		if err := os.MkdirAll(clindexDir, 0755); err != nil {
			t.Fatalf("Failed to create .clindex directory: %v", err)
		}

		metadata := models.IndexMetadata{
			Version:       "2.0.0",
			CreatedAt:     time.Now(),
			Migrated:      true,
			MigrationDate: time.Now(),
			LegacyFiles:   []string{".code-search-index"},
		}

		metadataFile := filepath.Join(clindexDir, "metadata.json")
		metadataBytes, _ := json.Marshal(metadata)
		if err := os.WriteFile(metadataFile, metadataBytes, 0644); err != nil {
			t.Fatalf("Failed to create metadata file: %v", err)
		}

		status, err := migrator.GetMigrationStatus(tempDir)
		if err != nil {
			t.Errorf("Expected no error getting migration status, got: %v", err)
		}

		if status != "migrated" {
			t.Errorf("Expected status 'migrated', got '%s'", status)
		}
	})

	t.Run("New directory status", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create new-style index without migration flag
		clindexDir := filepath.Join(tempDir, ".clindex")
		if err := os.MkdirAll(clindexDir, 0755); err != nil {
			t.Fatalf("Failed to create .clindex directory: %v", err)
		}

		metadata := models.IndexMetadata{
			Version:   "2.0.0",
			CreatedAt: time.Now(),
			Migrated:  false,
		}

		metadataFile := filepath.Join(clindexDir, "metadata.json")
		metadataBytes, _ := json.Marshal(metadata)
		if err := os.WriteFile(metadataFile, metadataBytes, 0644); err != nil {
			t.Fatalf("Failed to create metadata file: %v", err)
		}

		status, err := migrator.GetMigrationStatus(tempDir)
		if err != nil {
			t.Errorf("Expected no error getting migration status, got: %v", err)
		}

		if status != "new" {
			t.Errorf("Expected status 'new', got '%s'", status)
		}
	})
}

// TestIndexMigrator_RollbackMigration tests migration rollback
func TestIndexMigrator_RollbackMigration(t *testing.T) {
	migrator := lib.NewIndexMigrator()

	t.Run("Rollback migrated index", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create new-style index directory
		clindexDir := filepath.Join(tempDir, ".clindex")
		if err := os.MkdirAll(clindexDir, 0755); err != nil {
			t.Fatalf("Failed to create .clindex directory: %v", err)
		}

		// Create some index files
		indexFile := filepath.Join(clindexDir, "index.db")
		if err := os.WriteFile(indexFile, []byte("index data"), 0644); err != nil {
			t.Fatalf("Failed to create index file: %v", err)
		}

		// Verify directory exists
		if !dirExists(clindexDir) {
			t.Error("Expected .clindex directory to exist before rollback")
		}

		// Perform rollback
		if err := migrator.RollbackMigration(tempDir); err != nil {
			t.Errorf("Expected no error rolling back migration, got: %v", err)
		}

		// Verify directory was removed
		if dirExists(clindexDir) {
			t.Error("Expected .clindex directory to be removed after rollback")
		}
	})

	t.Run("Rollback non-existent index", func(t *testing.T) {
		tempDir := t.TempDir()

		// Should not error when trying to rollback non-existent index
		if err := migrator.RollbackMigration(tempDir); err != nil {
			t.Errorf("Expected no error rolling back non-existent index, got: %v", err)
		}
	})
}

// TestIndexMigrator_ErrorHandling tests error handling scenarios
func TestIndexMigrator_ErrorHandling(t *testing.T) {
	migrator := lib.NewIndexMigrator()

	t.Run("Migrate non-existent directory", func(t *testing.T) {
		nonExistentDir := "/non/existent/directory"

		result, err := migrator.MigrateIndex(nonExistentDir, false)
		if err == nil {
			t.Error("Expected error when migrating non-existent directory")
		}

		if result.Success {
			t.Error("Expected migration result to indicate failure")
		}

		if len(result.Errors) == 0 {
			t.Error("Expected errors in migration result")
		}
	})

	t.Run("Detect indexes in non-existent directory", func(t *testing.T) {
		nonExistentDir := "/non/existent/directory"

		_, err := migrator.DetectLegacyIndexes(nonExistentDir)
		if err != nil {
			// This might not error on all systems, which is acceptable
			t.Logf("Detecting indexes in non-existent directory failed: %v", err)
		}
	})
}

// BenchmarkIndexMigrator_MigrateIndex benchmarks migration performance
func BenchmarkIndexMigrator_MigrateIndex(b *testing.B) {
	migrator := lib.NewIndexMigrator()

	// Create a temporary directory with large legacy files
	tempDir := b.TempDir()

	// Create a large legacy index file
	legacyFile := filepath.Join(tempDir, ".code-search-index")
	content := make([]byte, 1024*1024) // 1MB
	for i := range content {
		content[i] = byte(i % 256)
	}

	if err := os.WriteFile(legacyFile, content, 0644); err != nil {
		b.Fatalf("Failed to create legacy file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clean up previous migration
		clindexDir := filepath.Join(tempDir, ".clindex")
		os.RemoveAll(clindexDir)

		_, err := migrator.MigrateIndex(tempDir, false)
		if err != nil {
			b.Fatalf("Migration failed: %v", err)
		}
	}
}

// Helper functions
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}