package lib

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"code-search/src/models"
)

// SimpleCodeParser implements the CodeParser interface
type SimpleCodeParser struct {
	supportedFileTypes []string
}

// NewSimpleCodeParser creates a new simple code parser
func NewSimpleCodeParser() *SimpleCodeParser {
	supportedTypes := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".java", ".c", ".cpp",
		".cc", ".cxx", ".h", ".hpp", ".cs", ".php", ".rb", ".swift", ".kt",
		".rs", ".scala", ".sh", ".bash", ".zsh", ".fish", ".ps1", ".sql",
		".html", ".css", ".scss", ".sass", ".less", ".xml", ".yaml", ".yml",
		".json", ".toml", ".md", ".txt", ".dockerfile",
	}

	return &SimpleCodeParser{
		supportedFileTypes: supportedTypes,
	}
}

// ParseFile parses a file into code chunks
func (p *SimpleCodeParser) ParseFile(filePath string) ([]models.CodeChunk, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)
	if contentStr == "" {
		return []models.CodeChunk{}, nil
	}

	// Detect language
	language := p.detectLanguage(filePath)

	// Split content into chunks based on language
	chunks := p.createChunks(contentStr, language)

	return chunks, nil
}

// GetEmbedding generates a vector embedding for text
func (p *SimpleCodeParser) GetEmbedding(text string) ([]float64, error) {
	// This is a simplified mock embedding implementation
	// In a real implementation, you would use a proper embedding model
	return p.generateMockEmbedding(text), nil
}

// GetSupportedFileTypes returns the list of supported file types
func (p *SimpleCodeParser) GetSupportedFileTypes() []string {
	return p.supportedFileTypes
}

// createChunks splits content into code chunks
func (p *SimpleCodeParser) createChunks(content, language string) []models.CodeChunk {
	lines := strings.Split(content, "\n")
	var chunks []models.CodeChunk

	switch language {
	case "Go":
		chunks = p.createGoChunks(lines)
	case "Python":
		chunks = p.createPythonChunks(lines)
	case "JavaScript", "TypeScript":
		chunks = p.createJSChunks(lines)
	default:
		// Generic chunking for other languages
		chunks = p.createGenericChunks(lines)
	}

	return chunks
}

// createGoChunks creates chunks for Go code
func (p *SimpleCodeParser) createGoChunks(lines []string) []models.CodeChunk {
	var chunks []models.CodeChunk
	currentChunk := ""
	startLine := 1

	inFunction := false
	inComment := false
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle multi-line comments
		if strings.HasPrefix(trimmed, "/*") {
			inComment = true
		}
		if strings.HasSuffix(trimmed, "*/") {
			inComment = false
		}
		if inComment || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Detect function/method definitions
		if p.isGoFunctionDefinition(trimmed) {
			// Save previous chunk if exists
			if currentChunk != "" {
				chunk := models.NewCodeChunk(currentChunk, startLine, i, "Go")
				chunks = append(chunks, *chunk)
			}

			currentChunk = line
			startLine = i + 1
			inFunction = true
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
		} else if inFunction {
			currentChunk += "\n" + line
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// End function when braces balance
			if braceCount <= 0 {
				chunk := models.NewCodeChunk(currentChunk, startLine, i+1, "Go")
				chunks = append(chunks, *chunk)
				currentChunk = ""
				startLine = i + 2
				inFunction = false
				braceCount = 0
			}
		} else if trimmed != "" {
			// Add non-empty lines as standalone chunks
			chunk := models.NewCodeChunk(line, i+1, i+1, "Go")
			chunks = append(chunks, *chunk)
		}
	}

	// Add final chunk if exists
	if currentChunk != "" {
		chunk := models.NewCodeChunk(currentChunk, startLine, len(lines), "Go")
		chunks = append(chunks, *chunk)
	}

	return chunks
}

// createPythonChunks creates chunks for Python code
func (p *SimpleCodeParser) createPythonChunks(lines []string) []models.CodeChunk {
	var chunks []models.CodeChunk
	currentChunk := ""
	startLine := 1
	indentLevel := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Detect function/class definitions
		if p.isPythonFunctionDefinition(trimmed) || p.isPythonClassDefinition(trimmed) {
			// Save previous chunk if exists
			if currentChunk != "" {
				chunk := models.NewCodeChunk(currentChunk, startLine, i, "Python")
				chunks = append(chunks, *chunk)
			}

			currentChunk = line
			startLine = i + 1
			indentLevel = currentIndent
		} else if currentChunk != "" {
			// Continue current chunk if same or greater indent
			if currentIndent <= indentLevel || trimmed == "" {
				currentChunk += "\n" + line
			} else {
				// End current chunk and start new one
				chunk := models.NewCodeChunk(currentChunk, startLine, i, "Python")
				chunks = append(chunks, *chunk)

				currentChunk = line
				startLine = i + 1
				indentLevel = currentIndent
			}
		} else {
			// Start new chunk
			currentChunk = line
			startLine = i + 1
			indentLevel = currentIndent
		}
	}

	// Add final chunk if exists
	if currentChunk != "" {
		chunk := models.NewCodeChunk(currentChunk, startLine, len(lines), "Python")
		chunks = append(chunks, *chunk)
	}

	return chunks
}

// createJSChunks creates chunks for JavaScript/TypeScript code
func (p *SimpleCodeParser) createJSChunks(lines []string) []models.CodeChunk {
	var chunks []models.CodeChunk
	currentChunk := ""
	startLine := 1

	inFunction := false
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}

		// Detect function definitions
		if p.isJSFunctionDefinition(trimmed) {
			// Save previous chunk if exists
			if currentChunk != "" {
				chunk := models.NewCodeChunk(currentChunk, startLine, i, "JavaScript")
				chunks = append(chunks, *chunk)
			}

			currentChunk = line
			startLine = i + 1
			inFunction = true
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
		} else if inFunction {
			currentChunk += "\n" + line
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// End function when braces balance
			if braceCount <= 0 {
				chunk := models.NewCodeChunk(currentChunk, startLine, i+1, "JavaScript")
				chunks = append(chunks, *chunk)
				currentChunk = ""
				startLine = i + 2
				inFunction = false
				braceCount = 0
			}
		} else if trimmed != "" {
			// Add significant lines as standalone chunks
			if p.isSignificantLine(trimmed) {
				chunk := models.NewCodeChunk(line, i+1, i+1, "JavaScript")
				chunks = append(chunks, *chunk)
			}
		}
	}

	// Add final chunk if exists
	if currentChunk != "" {
		chunk := models.NewCodeChunk(currentChunk, startLine, len(lines), "JavaScript")
		chunks = append(chunks, *chunk)
	}

	return chunks
}

// createGenericChunks creates chunks for generic text/code
func (p *SimpleCodeParser) createGenericChunks(lines []string) []models.CodeChunk {
	var chunks []models.CodeChunk

	// Create chunks of reasonable size (around 10-20 lines)
	chunkSize := 15
	for i := 0; i < len(lines); i += chunkSize {
		end := i + chunkSize
		if end > len(lines) {
			end = len(lines)
		}

		chunkContent := strings.Join(lines[i:end], "\n")
		if strings.TrimSpace(chunkContent) != "" {
			chunk := models.NewCodeChunk(chunkContent, i+1, end, "Unknown")
			chunks = append(chunks, *chunk)
		}
	}

	return chunks
}

// Helper methods for detecting patterns

func (p *SimpleCodeParser) isGoFunctionDefinition(line string) bool {
	line = strings.TrimSpace(line)

	// Go function patterns
	patterns := []string{
		`^func\s+\w+\s*\(`,          // func name(
		`^func\s*\(\w+\s+\*?\w+\)\s*\w+\s*\(`, // func (r *Receiver) name(
		`^func\s+\w+\s*[^{]*\{`,      // func name(args) {
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

func (p *SimpleCodeParser) isPythonFunctionDefinition(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "def ") && strings.Contains(line, "(")
}

func (p *SimpleCodeParser) isPythonClassDefinition(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "class ")
}

func (p *SimpleCodeParser) isJSFunctionDefinition(line string) bool {
	line = strings.TrimSpace(line)

	patterns := []string{
		`^function\s+\w+\s*\(`,      // function name(
		`^const\s+\w+\s*=\s*\(`,      // const name = (
		`^let\s+\w+\s*=\s*\(`,        // let name = (
		`^var\s+\w+\s*=\s*\(`,        // var name = (
		`^\w+\s*:\s*\([^)]*\)\s*=>`, // name: (args) =>
		`^\w+\s*\([^)]*\)\s*{`,       // name(args) {
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

func (p *SimpleCodeParser) isSignificantLine(line string) bool {
	line = strings.TrimSpace(line)

	// Skip simple lines
	if line == "" ||
	   strings.HasPrefix(line, "//") ||
	   strings.HasPrefix(line, "/*") ||
	   strings.HasPrefix(line, "*") ||
	   strings.HasPrefix(line, "}") ||
	   strings.HasPrefix(line, "]") ||
	   strings.HasPrefix(line, ")") {
		return false
	}

	// Check for significant patterns
	significantPatterns := []string{
		`^var\s+`, `^let\s+`, `^const\s+`,  // Variable declarations
		`^if\s+`, `^else`, `^for\s+`, `^while\s+`, // Control flow
		`^function\s+`, `=>`, `return\s+`,       // Functions
		`^class\s+`, `^extends\s+`,              // Classes
		`^import\s+`, `^export\s+`,              // Modules
		`\{$`, `\}`,                            // Braces
	}

	for _, pattern := range significantPatterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

// detectLanguage detects the programming language based on file extension
func (p *SimpleCodeParser) detectLanguage(filePath string) string {
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

// generateMockEmbedding creates a mock embedding vector for text
// This is a simplified implementation for demonstration
func (p *SimpleCodeParser) generateMockEmbedding(text string) []float64 {
	// Create a deterministic but pseudo-random vector based on text hash
	hash := p.simpleHash(text)

	// Generate a 128-dimensional vector
	dimensions := 128
	vector := make([]float64, dimensions)

	for i := 0; i < dimensions; i++ {
		// Use different bits of the hash to generate values
		shift := (i * 7) % 64
		value := float64((hash>>shift)&0xFF) / 255.0

		// Add some variation based on position
		value = value*0.7 + float64(i%10)/10.0*0.3

		vector[i] = value
	}

	// Normalize the vector
	magnitude := 0.0
	for _, v := range vector {
		magnitude += v * v
	}
	magnitude = math.Sqrt(magnitude)

	if magnitude > 0 {
		for i := range vector {
			vector[i] = vector[i] / magnitude
		}
	}

	return vector
}

// simpleHash creates a simple hash of the input text
func (p *SimpleCodeParser) simpleHash(text string) uint64 {
	hash := uint64(5381)
	for _, c := range text {
		hash = ((hash << 5) + hash) + uint64(c)
	}
	return hash
}