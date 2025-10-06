package lib

import (
	"os"
	"path/filepath"
	"strings"

	"code-search/src/services"
)

// FileSystemScanner implements the FileScanner interface
type FileSystemScanner struct {
	ignorePatterns []string
}

// NewFileSystemScanner creates a new file system scanner
func NewFileSystemScanner() *FileSystemScanner {
	ignorePatterns := []string{
		".git/*",
		"node_modules/*",
		"*.tmp",
		"*.log",
		".DS_Store",
		"Thumbs.db",
		"*.pyc",
		"*.pyo",
		"__pycache__/*",
		".vscode/*",
		".idea/*",
		"*.swp",
		"*.swo",
		"*~",
	}

	return &FileSystemScanner{
		ignorePatterns: ignorePatterns,
	}
}

// ScanFiles scans files in the given directory according to options
func (fs *FileSystemScanner) ScanFiles(rootPath string, options services.IndexingOptions) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip hidden files unless explicitly included
		if !options.IncludeHidden && fs.isHiddenFile(path) {
			return nil
		}

		// Skip files matching exclude patterns
		if fs.shouldExcludeFile(path, options.ExcludePatterns) {
			return nil
		}

		// Check file size
		if info.Size() > options.MaxFileSize {
			return nil
		}

		// Check if file type is supported
		if !fs.isSupportedFileType(path, options.FileTypes) {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// GetFileStats returns statistics about a file
func (fs *FileSystemScanner) GetFileStats(filePath string) (services.FileStats, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return services.FileStats{}, err
	}

	// Detect language
	language := fs.detectLanguage(filePath)

	return services.FileStats{
		Path:         filePath,
		Size:         info.Size(),
		ModifiedTime: info.ModTime(),
		IsDir:        info.IsDir(),
		Language:     language,
	}, nil
}

// isHiddenFile checks if a file is hidden
func (fs *FileSystemScanner) isHiddenFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".")
}

// shouldExcludeFile checks if a file should be excluded based on patterns
func (fs *FileSystemScanner) shouldExcludeFile(path string, excludePatterns []string) bool {
	// Combine built-in patterns with user patterns
	allPatterns := append([]string{}, fs.ignorePatterns...)
	allPatterns = append(allPatterns, excludePatterns...)

	for _, pattern := range allPatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}

		// Check directory patterns
		if strings.HasSuffix(pattern, "/*") {
			dirPattern := strings.TrimSuffix(pattern, "/*")
			if strings.Contains(path, string(filepath.Separator)+dirPattern+string(filepath.Separator)) {
				return true
			}
		}
	}

	return false
}

// isSupportedFileType checks if the file type is supported
func (fs *FileSystemScanner) isSupportedFileType(path string, fileTypes []string) bool {
	// If wildcard, check against supported types
	if len(fileTypes) == 1 && fileTypes[0] == "*" {
		return fs.isGenerallySupported(path)
	}

	// Check specific file types
	for _, fileType := range fileTypes {
		if strings.HasSuffix(strings.ToLower(path), strings.ToLower(fileType)) {
			return true
		}
	}

	return false
}

// isGenerallySupported checks if a file type is generally supported for code indexing
func (fs *FileSystemScanner) isGenerallySupported(path string) bool {
	supportedExtensions := map[string]bool{
		".go":   true,
		".js":   true,
		".ts":   true,
		".jsx":  true,
		".tsx":  true,
		".py":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".cc":   true,
		".cxx":  true,
		".h":    true,
		".hpp":  true,
		".cs":   true,
		".php":  true,
		".rb":   true,
		".swift": true,
		".kt":   true,
		".rs":   true,
		".scala": true,
		".sh":   true,
		".bash": true,
		".zsh":  true,
		".fish": true,
		".ps1":  true,
		".sql":  true,
		".html": true,
		".css":  true,
		".scss": true,
		".sass": true,
		".less": true,
		".xml":  true,
		".yaml": true,
		".yml":  true,
		".json": true,
		".toml": true,
		".md":   true,
		".txt":  true,
		".dockerfile": true,
	}

	ext := strings.ToLower(filepath.Ext(path))
	return supportedExtensions[ext]
}

// detectLanguage detects the programming language based on file extension
func (fs *FileSystemScanner) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".jsx":  "JavaScript",
		".tsx":  "TypeScript",
		".py":   "Python",
		".java": "Java",
		".c":    "C",
		".cpp":  "C++",
		".cc":   "C++",
		".cxx":  "C++",
		".h":    "C/C++ Header",
		".hpp":  "C++ Header",
		".cs":   "C#",
		".php":  "PHP",
		".rb":   "Ruby",
		".swift": "Swift",
		".kt":   "Kotlin",
		".rs":   "Rust",
		".scala": "Scala",
		".sh":   "Shell",
		".bash": "Shell",
		".zsh":  "Shell",
		".fish": "Shell",
		".ps1":  "PowerShell",
		".sql":  "SQL",
		".html": "HTML",
		".css":  "CSS",
		".scss": "SCSS",
		".sass": "Sass",
		".less": "Less",
		".xml":  "XML",
		".yaml": "YAML",
		".yml":  "YAML",
		".json": "JSON",
		".toml": "TOML",
		".md":   "Markdown",
		".txt":  "Text",
		".dockerfile": "Dockerfile",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	// Check for special filenames
	filename := strings.ToLower(filepath.Base(filePath))
	switch filename {
	case "dockerfile":
		return "Dockerfile"
	case "makefile":
		return "Makefile"
	case "gemfile":
		return "Ruby"
	case "requirements.txt":
		return "Python"
	case "package.json":
		return "JavaScript/Node.js"
	case "go.mod":
		return "Go"
	case "go.sum":
		return "Go"
	}

	return "Unknown"
}