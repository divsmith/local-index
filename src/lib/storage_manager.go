// ABOUTME: Manages centralized storage for code search indexes and embeddings

package lib

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// StorageManager handles centralized storage for project-scoped indexes and embeddings
type StorageManager struct {
	baseDir string
}

// NewStorageManager creates a new storage manager with centralized storage
func NewStorageManager() *StorageManager {
	homeDir, _ := os.UserHomeDir()
	return &StorageManager{
		baseDir: filepath.Join(homeDir, ".code-search"),
	}
}

// GetProjectIndexPath returns the index file path for a given project
func (sm *StorageManager) GetProjectIndexPath(projectPath string) string {
	// Create project identifier (hash of full path)
	projectID := sm.hashProjectPath(projectPath)
	return filepath.Join(sm.baseDir, "indexes", projectID+".db")
}

// GetProjectEmbeddingPath returns the embedding file path for a given project
func (sm *StorageManager) GetProjectEmbeddingPath(projectPath string) string {
	projectID := sm.hashProjectPath(projectPath)
	return filepath.Join(sm.baseDir, "embeddings", projectID+".cache")
}

// GetModelsDir returns the directory for storing ONNX models
func (sm *StorageManager) GetModelsDir() string {
	return filepath.Join(sm.baseDir, "models")
}

// EnsureDirectories creates all necessary directories if they don't exist
func (sm *StorageManager) EnsureDirectories() error {
	dirs := []string{
		sm.baseDir,
		filepath.Join(sm.baseDir, "indexes"),
		filepath.Join(sm.baseDir, "embeddings"),
		filepath.Join(sm.baseDir, "models"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// hashProjectPath creates a consistent hash identifier for a project path
func (sm *StorageManager) hashProjectPath(projectPath string) string {
	// Convert to absolute path to ensure consistency
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		absPath = projectPath
	}

	hash := sha256.Sum256([]byte(absPath))
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 characters for shorter filenames
}

// ProjectExists checks if an index already exists for the given project
func (sm *StorageManager) ProjectExists(projectPath string) bool {
	indexPath := sm.GetProjectIndexPath(projectPath)
	_, err := os.Stat(indexPath)
	return err == nil
}

// RemoveProject removes all stored data for a project
func (sm *StorageManager) RemoveProject(projectPath string) error {
	indexPath := sm.GetProjectIndexPath(projectPath)
	embeddingPath := sm.GetProjectEmbeddingPath(projectPath)

	// Remove index file
	if _, err := os.Stat(indexPath); err == nil {
		if removeErr := os.Remove(indexPath); removeErr != nil {
			return fmt.Errorf("failed to remove index file: %w", removeErr)
		}
	}

	// Remove embedding file
	if _, err := os.Stat(embeddingPath); err == nil {
		if removeErr := os.Remove(embeddingPath); removeErr != nil {
			return fmt.Errorf("failed to remove embedding file: %w", removeErr)
		}
	}

	return nil
}

// ListProjects returns a list of all project paths that have stored indexes
func (sm *StorageManager) ListProjects() ([]string, error) {
	indexesDir := filepath.Join(sm.baseDir, "indexes")

	entries, err := os.ReadDir(indexesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read indexes directory: %w", err)
	}

	projects := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".db" {
			projects = append(projects, entry.Name())
		}
	}

	return projects, nil
}