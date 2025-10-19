package lib

import (
	"path/filepath"
	"strings"
)

// FileFilter provides intelligent file filtering for agent-optimized search
type FileFilter struct {
	excludePatterns []string
	includeOnly     []string
}

// NewAgentFileFilter creates a file filter with agent-optimized defaults
func NewAgentFileFilter() *FileFilter {
	return &FileFilter{
		excludePatterns: []string{
			"*/test*", "*/tests/*", "*_test.go",
			"*/vendor/*", "*/node_modules/*",
			"*/docs/*", "*/doc/*",
			"*/build/*", "*/dist/*", "*/target/*",
			"*/.git/*", "*/.svn/*",
			"*/.idea/*", "*/.vscode/*",
			"*/cache/*", "*/tmp/*", "*/temp/*",
			"*.min.js", "*.min.css",
			"*/coverage/*", "*/.coverage/*",
			"*/logs/*", "*/log/*",
		},
		includeOnly: []string{
			".go", ".js", ".ts", ".jsx", ".tsx",
			".py", ".java", ".cpp", ".c", ".h", ".hpp",
			".cs", ".php", ".rb", ".swift", ".kt",
			".rs", ".scala", ".clj", ".hs", ".ml",
			".sh", ".bash", ".zsh", ".fish",
			".html", ".htm", ".css", ".scss", ".sass",
			".json", ".yaml", ".yml", ".toml", ".ini",
			".sql", ".graphql", ".proto",
			".md", ".txt", ".rst",
		},
	}
}

// ShouldInclude determines if a file should be included in search results
func (ff *FileFilter) ShouldInclude(filePath string) bool {
	// Normalize path
	filePath = filepath.ToSlash(filePath)

	// Check exclusion patterns first
	for _, pattern := range ff.excludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
			return false
		}

		// Check directory patterns
		pathParts := strings.Split(filePath, "/")
		for i, part := range pathParts {
			for _, pattern := range ff.excludePatterns {
				if matched, _ := filepath.Match(pattern, part); matched {
					return false
				}
				// Check if any parent directory matches pattern
				if i < len(pathParts)-1 {
					fullPath := strings.Join(pathParts[:i+1], "/")
					if matched, _ := filepath.Match(pattern, fullPath); matched {
						return false
					}
				}
			}
		}
	}

	// If includeOnly is specified, check against it
	if len(ff.includeOnly) > 0 {
		ext := strings.ToLower(filepath.Ext(filePath))
		for _, allowedExt := range ff.includeOnly {
			if ext == allowedExt {
				return true
			}
		}
		return false
	}

	return true
}

// AddCustomPattern adds a custom exclusion pattern
func (ff *FileFilter) AddCustomPattern(pattern string) {
	ff.excludePatterns = append(ff.excludePatterns, pattern)
}

// SetIncludeOnly sets the allowed file extensions
func (ff *FileFilter) SetIncludeOnly(extensions []string) {
	ff.includeOnly = extensions
}

// ClearIncludeOnly removes file extension restrictions
func (ff *FileFilter) ClearIncludeOnly() {
	ff.includeOnly = []string{}
}