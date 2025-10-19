// ABOUTME: Detects project root directories for proper code search scoping

package lib

import (
	"os"
	"path/filepath"
)

// ProjectDetector handles detection of project boundaries
type ProjectDetector struct{}

// NewProjectDetector creates a new project detector
func NewProjectDetector() *ProjectDetector {
	return &ProjectDetector{}
}

// DetectProjectRoot finds the project root directory starting from the given path
func (pd *ProjectDetector) DetectProjectRoot(startPath string) (string, error) {
	return pd.DetectProjectRootWithOptions(startPath, true)
}

// DetectProjectRootWithOptions finds the project root with options to control walking behavior
func (pd *ProjectDetector) DetectProjectRootWithOptions(startPath string, allowWalkUp bool) (string, error) {
	// Convert to absolute path first
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		absPath = startPath
	}

	// Check if the start path itself is a project root
	if pd.isDirectoryProjectRoot(absPath) {
		return absPath, nil
	}

	// Only walk up the tree if allowed
	if allowWalkUp {
		// First, try to find .git directory by walking up from start path
		if gitRoot := pd.findGitRoot(absPath); gitRoot != "" {
			return gitRoot, nil
		}

		// Fallback: look for other project indicators by walking up from start path
		if projectRoot := pd.findProjectRootByMarkers(absPath); projectRoot != "" {
			return projectRoot, nil
		}
	}

	// Final fallback: use the absolute path as-is
	return absPath, nil
}

// findGitRoot walks up the directory tree to find the .git directory
func (pd *ProjectDetector) findGitRoot(startPath string) string {
	current := startPath

	for {
		gitPath := filepath.Join(current, ".git")
		if stat, err := os.Stat(gitPath); err == nil {
			if stat.IsDir() {
				return current
			}
			// Handle .git as a file (git worktree)
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root directory
			return ""
		}

		current = parent
	}
}

// findProjectRootByMarkers looks for common project root markers
func (pd *ProjectDetector) findProjectRootByMarkers(startPath string) string {
	current := startPath

	// Common markers that indicate a project root
	projectMarkers := []string{
		"go.mod",           // Go modules
		"package.json",     // Node.js projects
		"setup.py",         // Python projects
		"pyproject.toml",   // Python projects
		"Cargo.toml",       // Rust projects
		"pom.xml",          // Maven projects
		"build.gradle",     // Gradle projects
		"Makefile",         // Make-based projects
		"CMakeLists.txt",   // CMake projects
		".project",         // Eclipse projects
	}

	for {
		// Check for any project marker in current directory
		for _, marker := range projectMarkers {
			markerPath := filepath.Join(current, marker)
			if _, err := os.Stat(markerPath); err == nil {
				return current
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root directory
			return ""
		}

		current = parent
	}
}

// IsSameProject checks if two paths are within the same project
func (pd *ProjectDetector) IsSameProject(path1, path2 string) (bool, error) {
	root1, err := pd.DetectProjectRoot(path1)
	if err != nil {
		return false, err
	}

	root2, err := pd.DetectProjectRoot(path2)
	if err != nil {
		return false, err
	}

	return root1 == root2, nil
}

// GetProjectRelativePath returns the path relative to the project root
func (pd *ProjectDetector) GetProjectRelativePath(filePath string) (string, error) {
	root, err := pd.DetectProjectRoot(filePath)
	if err != nil {
		return "", err
	}

	relPath, err := filepath.Rel(root, filePath)
	if err != nil {
		return "", err
	}

	return relPath, nil
}

// isDirectoryProjectRoot checks if the given directory is already a project root
func (pd *ProjectDetector) isDirectoryProjectRoot(dirPath string) bool {
	// Check for .git directory
	if gitPath := filepath.Join(dirPath, ".git"); pd.pathExists(gitPath) {
		return true
	}

	// Check for project markers in this specific directory
	projectMarkers := []string{
		"go.mod",           // Go modules
		"package.json",     // Node.js projects
		"setup.py",         // Python projects
		"pyproject.toml",   // Python projects
		"Cargo.toml",       // Rust projects
		"pom.xml",          // Maven projects
		"build.gradle",     // Gradle projects
		"Makefile",         // Make-based projects
		"CMakeLists.txt",   // CMake projects
		".project",         // Eclipse projects
	}

	for _, marker := range projectMarkers {
		markerPath := filepath.Join(dirPath, marker)
		if pd.pathExists(markerPath) {
			return true
		}
	}

	// If no markers found, treat this directory as its own project
	// This ensures explicitly specified directories are respected
	return true
}

// pathExists checks if a path exists
func (pd *ProjectDetector) pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}