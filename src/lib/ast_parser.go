package lib

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"regexp"
	"strings"

	"code-search/src/models"
)

// ASTCodeParser implements advanced AST-based code chunking
type ASTCodeParser struct {
	simpleParser *SimpleCodeParser
	chunkConfig  *ChunkingConfig
}

// ChunkingConfig contains configuration for code chunking strategy
type ChunkingConfig struct {
	BaseChunkSize       int     `json:"base_chunk_size"`        // Base size for chunks (lines)
	MaxChunkSize       int     `json:"max_chunk_size"`        // Maximum chunk size
	OverlapLines        int     `json:"overlap_lines"`         // Overlap between chunks
	MinContextLines     int     `json:"min_context_lines"`     // Minimum context lines
	AdaptiveSizeFactor  float64 `json:"adaptive_size_factor"`  // Factor for adaptive sizing
	ComplexityWeight    float64 `json:"complexity_weight"`     // Weight for complexity-based sizing
}

// DefaultChunkingConfig returns default chunking configuration
func DefaultChunkingConfig() *ChunkingConfig {
	return &ChunkingConfig{
		BaseChunkSize:      20,
		MaxChunkSize:      100,
		OverlapLines:       5,
		MinContextLines:   3,
		AdaptiveSizeFactor: 1.5,
		ComplexityWeight:   0.3,
	}
}

// NewASTCodeParser creates a new AST-based code parser
func NewASTCodeParser(config *ChunkingConfig) *ASTCodeParser {
	if config == nil {
		config = DefaultChunkingConfig()
	}

	return &ASTCodeParser{
		simpleParser: NewSimpleCodeParser(),
		chunkConfig:  config,
	}
}

// ParseFile parses a file using AST-based chunking when possible
func (p *ASTCodeParser) ParseFile(filePath string) ([]models.CodeChunk, error) {
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
	language := p.simpleParser.detectLanguage(filePath)

	// Use AST-based chunking for supported languages
	switch language {
	case "Go":
		return p.parseGoAST(filePath, contentStr)
	case "Python":
		return p.parsePythonSmart(contentStr)
	case "JavaScript", "TypeScript":
		return p.parseJavaScriptSmart(contentStr)
	default:
		// Fall back to enhanced chunking for other languages
		return p.createEnhancedChunks(contentStr, language)
	}
}

// parseGoAST uses Go's AST parser for intelligent chunking
func (p *ASTCodeParser) parseGoAST(filePath, content string) ([]models.CodeChunk, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		// Fall back to enhanced chunking if AST parsing fails
		return p.createEnhancedChunks(content, "Go")
	}

	var chunks []models.CodeChunk
	lines := strings.Split(content, "\n")

	// Process AST nodes to create chunks
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			chunks = append(chunks, p.createFunctionChunk(x, fset, lines, "Go"))
		case *ast.TypeSpec:
			chunks = append(chunks, p.createTypeChunk(x, fset, lines, "Go"))
		case *ast.GenDecl:
			if x.Tok == token.IMPORT {
				chunks = append(chunks, p.createImportChunk(x, fset, lines, "Go"))
			}
		}
		return true
	})

	// Add non-AST chunks for code not covered by declarations
	chunks = append(chunks, p.createNonStructuralChunks(content, chunks, "Go")...)

	return chunks, nil
}

// createFunctionChunk creates a chunk from a Go function declaration
func (p *ASTCodeParser) createFunctionChunk(fn *ast.FuncDecl, fset *token.FileSet, lines []string, language string) models.CodeChunk {
	start := fset.Position(fn.Pos())
	end := fset.Position(fn.End())

	startLine := start.Line
	endLine := end.Line

	// Calculate complexity for adaptive sizing
	complexity := p.calculateGoFunctionComplexity(fn)
	adaptiveSize := int(float64(p.chunkConfig.BaseChunkSize) * (1.0 + complexity*p.chunkConfig.ComplexityWeight))

	// Add context lines around the function
	contextStart := p.max(1, startLine-p.chunkConfig.MinContextLines)
	contextEnd := p.min(len(lines), endLine+p.chunkConfig.MinContextLines)

	// Extract the function content
	var contentLines []string
	for i := contextStart; i <= contextEnd; i++ {
		contentLines = append(contentLines, lines[i-1]) // Lines are 1-indexed
	}

	content := strings.Join(contentLines, "\n")

	chunk := models.NewCodeChunk(content, startLine, endLine, language)

	// Add metadata about the function
	metadata := map[string]interface{}{
		"function_name":  fn.Name.Name,
		"complexity":     complexity,
		"start_line":     startLine,
		"end_line":       endLine,
		"is_receiver":    fn.Recv != nil,
		"chunk_type":     "function",
		"adaptive_size":  adaptiveSize,
	}

	chunk.Metadata = metadata
	return *chunk
}

// createTypeChunk creates a chunk from a Go type declaration
func (p *ASTCodeParser) createTypeChunk(typeSpec *ast.TypeSpec, fset *token.FileSet, lines []string, language string) models.CodeChunk {
	start := fset.Position(typeSpec.Pos())
	end := fset.Position(typeSpec.End())

	startLine := start.Line
	endLine := end.Line

	// Extract type content with context
	contextStart := p.max(1, startLine-2)
	contextEnd := p.min(len(lines), endLine+3)

	var contentLines []string
	for i := contextStart; i <= contextEnd; i++ {
		contentLines = append(contentLines, lines[i-1])
	}

	content := strings.Join(contentLines, "\n")

	chunk := models.NewCodeChunk(content, startLine, endLine, language)

	metadata := map[string]interface{}{
		"type_name":    typeSpec.Name.Name,
		"start_line":   startLine,
		"end_line":     endLine,
		"chunk_type":   "type",
	}

	chunk.Metadata = metadata
	return *chunk
}

// createImportChunk creates a chunk from Go import declarations
func (p *ASTCodeParser) createImportChunk(decl *ast.GenDecl, fset *token.FileSet, lines []string, language string) models.CodeChunk {
	start := fset.Position(decl.Pos())
	end := fset.Position(decl.End())

	startLine := start.Line
	endLine := end.Line

	// Extract import block
	var contentLines []string
	for i := startLine; i <= endLine; i++ {
		if i <= len(lines) {
			contentLines = append(contentLines, lines[i-1])
		}
	}

	content := strings.Join(contentLines, "\n")

	chunk := models.NewCodeChunk(content, startLine, endLine, language)

	metadata := map[string]interface{}{
		"start_line": startLine,
		"end_line":   endLine,
		"chunk_type": "imports",
	}

	chunk.Metadata = metadata
	return *chunk
}

// calculateGoFunctionComplexity calculates a complexity score for a Go function
func (p *ASTCodeParser) calculateGoFunctionComplexity(fn *ast.FuncDecl) float64 {
	complexity := 1.0

	// Increase complexity based on function signature
	if fn.Recv != nil {
		complexity += 0.5 // Receiver method
	}

	// Count parameters and return values
	if fn.Type.Params != nil {
		complexity += float64(len(fn.Type.Params.List)) * 0.2
	}
	if fn.Type.Results != nil {
		complexity += float64(len(fn.Type.Results.List)) * 0.2
	}

	// Basic body complexity calculation
	// In a full implementation, you'd traverse the function body
	// and count control structures, etc.

	return math.Min(complexity, 3.0) // Cap at 3x complexity
}

// parsePythonSmart implements smart Python chunking with language-specific heuristics
func (p *ASTCodeParser) parsePythonSmart(content string) ([]models.CodeChunk, error) {
	lines := strings.Split(content, "\n")
	var chunks []models.CodeChunk

	// Python-specific patterns
	functionPattern := regexp.MustCompile(`^\s*def\s+(\w+)\s*\(`)
	classPattern := regexp.MustCompile(`^\s*class\s+(\w+)`)
	methodPattern := regexp.MustCompile(`^\s*def\s+(\w+)\s*\(`)

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			i++
			continue
		}

		// Detect function definitions
		if functionPattern.MatchString(line) {
			chunk, nextLine := p.createPythonFunctionChunk(lines, i, functionPattern)
			chunks = append(chunks, chunk)
			i = nextLine
		} else if classPattern.MatchString(line) { // Detect class definitions
			chunk, nextLine := p.createPythonClassChunk(lines, i, classPattern, methodPattern)
			chunks = append(chunks, chunk)
			i = nextLine
		} else {
			// Handle non-structural code
			chunk, nextLine := p.createPythonStandaloneChunk(lines, i)
			if chunk != nil {
				chunks = append(chunks, *chunk)
			}
			i = nextLine
		}
	}

	return chunks, nil
}

// createPythonFunctionChunk creates a chunk for a Python function
func (p *ASTCodeParser) createPythonFunctionChunk(lines []string, startIdx int, pattern *regexp.Regexp) (models.CodeChunk, int) {
	startLine := startIdx + 1
	baseIndent := len(lines[startIdx]) - len(strings.TrimLeft(lines[startIdx], " \t"))

	// Find the end of the function
	endIdx := startIdx + 1
	for endIdx < len(lines) {
		line := lines[endIdx]
		if strings.TrimSpace(line) == "" {
			endIdx++
			continue
		}

		currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
		if currentIndent <= baseIndent && strings.TrimSpace(line) != "" {
			break
		}
		endIdx++
	}

	endLine := endIdx

	// Add context
	contextStart := p.max(0, startIdx-p.chunkConfig.MinContextLines)
	contextEnd := p.min(len(lines), endIdx+p.chunkConfig.MinContextLines)

	var contentLines []string
	for i := contextStart; i < contextEnd; i++ {
		contentLines = append(contentLines, lines[i])
	}

	content := strings.Join(contentLines, "\n")

	// Extract function name
	matches := pattern.FindStringSubmatch(lines[startIdx])
	functionName := ""
	if len(matches) > 1 {
		functionName = matches[1]
	}

	chunk := models.NewCodeChunk(content, startLine, endLine, "Python")
	chunk.Metadata = map[string]interface{}{
		"function_name": functionName,
		"start_line":    startLine,
		"end_line":      endLine,
		"chunk_type":    "function",
		"indent_level":  baseIndent,
	}

	return *chunk, endIdx
}

// createPythonClassChunk creates a chunk for a Python class
func (p *ASTCodeParser) createPythonClassChunk(lines []string, startIdx int, classPattern, methodPattern *regexp.Regexp) (models.CodeChunk, int) {
	startLine := startIdx + 1
	baseIndent := len(lines[startIdx]) - len(strings.TrimLeft(lines[startIdx], " \t"))

	// Find the end of the class
	endIdx := startIdx + 1
	for endIdx < len(lines) {
		line := lines[endIdx]
		if strings.TrimSpace(line) == "" {
			endIdx++
			continue
		}

		currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
		if currentIndent <= baseIndent && strings.TrimSpace(line) != "" {
			break
		}
		endIdx++
	}

	endLine := endIdx

	// Add context
	contextStart := p.max(0, startIdx-p.chunkConfig.MinContextLines)
	contextEnd := p.min(len(lines), endIdx+p.chunkConfig.MinContextLines)

	var contentLines []string
	for i := contextStart; i < contextEnd; i++ {
		contentLines = append(contentLines, lines[i])
	}

	content := strings.Join(contentLines, "\n")

	// Extract class name
	matches := classPattern.FindStringSubmatch(lines[startIdx])
	className := ""
	if len(matches) > 1 {
		className = matches[1]
	}

	chunk := models.NewCodeChunk(content, startLine, endLine, "Python")
	chunk.Metadata = map[string]interface{}{
		"class_name":   className,
		"start_line":   startLine,
		"end_line":     endLine,
		"chunk_type":   "class",
		"indent_level": baseIndent,
	}

	return *chunk, endIdx
}

// createPythonStandaloneChunk creates chunks for non-structural Python code
func (p *ASTCodeParser) createPythonStandaloneChunk(lines []string, startIdx int) (*models.CodeChunk, int) {
	// Create chunks of adaptive size based on content
	adaptiveSize := p.calculateAdaptiveChunkSize(lines, startIdx, "Python")
	endIdx := p.min(startIdx+adaptiveSize, len(lines))

	// Find a good breaking point (e.g., empty line or logical boundary)
	breakIdx := p.findOptimalBreakPoint(lines, startIdx, endIdx)
	if breakIdx > startIdx {
		endIdx = breakIdx
	}

	contentLines := lines[startIdx:endIdx]
	content := strings.Join(contentLines, "\n")

	if strings.TrimSpace(content) == "" {
		return nil, endIdx
	}

	chunk := models.NewCodeChunk(content, startIdx+1, endIdx, "Python")
	chunk.Metadata = map[string]interface{}{
		"start_line": startIdx + 1,
		"end_line":   endIdx,
		"chunk_type": "standalone",
	}

	return chunk, endIdx
}

// parseJavaScriptSmart implements smart JavaScript/TypeScript chunking
func (p *ASTCodeParser) parseJavaScriptSmart(content string) ([]models.CodeChunk, error) {
	lines := strings.Split(content, "\n")
	var chunks []models.CodeChunk

	// JavaScript/TypeScript patterns
	functionPattern := regexp.MustCompile(`^\s*(function\s+\w+\s*\(|\w+\s*:\s*function\s*\(|\w+\s*\([^)]*\)\s*=>|\w+\s*\([^)]*\)\s*\{)`)
	classPattern := regexp.MustCompile(`^\s*(class\s+\w+|export\s+class\s+\w+)`)
	varPattern := regexp.MustCompile(`^\s*(const|let|var)\s+\w+\s*=`)

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			i++
			continue
		}

		// Detect function definitions
		if functionPattern.MatchString(line) {
			chunk, nextLine := p.createJavaScriptFunctionChunk(lines, i, functionPattern)
			chunks = append(chunks, chunk)
			i = nextLine
		} else if classPattern.MatchString(line) { // Detect class definitions
			chunk, nextLine := p.createJavaScriptClassChunk(lines, i, classPattern)
			chunks = append(chunks, chunk)
			i = nextLine
		} else if varPattern.MatchString(line) { // Detect variable declarations
			chunk, nextLine := p.createJavaScriptVariableChunk(lines, i, varPattern)
			chunks = append(chunks, chunk)
			i = nextLine
		} else {
			// Handle other code
			chunk, nextLine := p.createJavaScriptStandaloneChunk(lines, i)
			if chunk != nil {
				chunks = append(chunks, *chunk)
			}
			i = nextLine
		}
	}

	return chunks, nil
}

// createJavaScriptFunctionChunk creates a chunk for JavaScript functions
func (p *ASTCodeParser) createJavaScriptFunctionChunk(lines []string, startIdx int, pattern *regexp.Regexp) (models.CodeChunk, int) {
	startLine := startIdx + 1

	// Find function end by counting braces
	braceCount := 0
	endIdx := startIdx

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		braceCount += strings.Count(line, "{")
		braceCount -= strings.Count(line, "}")

		endIdx = i + 1
		if braceCount <= 0 && strings.Contains(line, "}") {
			break
		}
	}

	// Add context
	contextStart := p.max(0, startIdx-p.chunkConfig.MinContextLines)
	contextEnd := p.min(len(lines), endIdx+p.chunkConfig.MinContextLines)

	var contentLines []string
	for i := contextStart; i < contextEnd; i++ {
		contentLines = append(contentLines, lines[i])
	}

	content := strings.Join(contentLines, "\n")

	chunk := models.NewCodeChunk(content, startLine, endIdx, "JavaScript")
	chunk.Metadata = map[string]interface{}{
		"start_line": startLine,
		"end_line":   endIdx,
		"chunk_type": "function",
	}

	return *chunk, endIdx
}

// createJavaScriptClassChunk creates a chunk for JavaScript classes
func (p *ASTCodeParser) createJavaScriptClassChunk(lines []string, startIdx int, pattern *regexp.Regexp) (models.CodeChunk, int) {
	startLine := startIdx + 1

	// Find class end by counting braces
	braceCount := 0
	endIdx := startIdx

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		braceCount += strings.Count(line, "{")
		braceCount -= strings.Count(line, "}")

		endIdx = i + 1
		if braceCount <= 0 && strings.Contains(line, "}") {
			break
		}
	}

	// Add context
	contextStart := p.max(0, startIdx-p.chunkConfig.MinContextLines)
	contextEnd := p.min(len(lines), endIdx+p.chunkConfig.MinContextLines)

	var contentLines []string
	for i := contextStart; i < contextEnd; i++ {
		contentLines = append(contentLines, lines[i])
	}

	content := strings.Join(contentLines, "\n")

	chunk := models.NewCodeChunk(content, startLine, endIdx, "JavaScript")
	chunk.Metadata = map[string]interface{}{
		"start_line": startLine,
		"end_line":   endIdx,
		"chunk_type": "class",
	}

	return *chunk, endIdx
}

// createJavaScriptVariableChunk creates a chunk for JavaScript variable declarations
func (p *ASTCodeParser) createJavaScriptVariableChunk(lines []string, startIdx int, pattern *regexp.Regexp) (models.CodeChunk, int) {
	startLine := startIdx + 1

	// Group consecutive variable declarations
	endIdx := startIdx + 1
	for endIdx < len(lines) {
		line := strings.TrimSpace(lines[endIdx])
		if !pattern.MatchString(line) && line != "" {
			break
		}
		endIdx++
	}

	// Add context
	contextStart := p.max(0, startIdx-p.chunkConfig.OverlapLines)
	contextEnd := p.min(len(lines), endIdx+p.chunkConfig.OverlapLines)

	var contentLines []string
	for i := contextStart; i < contextEnd; i++ {
		contentLines = append(contentLines, lines[i])
	}

	content := strings.Join(contentLines, "\n")

	chunk := models.NewCodeChunk(content, startLine, endIdx, "JavaScript")
	chunk.Metadata = map[string]interface{}{
		"start_line": startLine,
		"end_line":   endIdx,
		"chunk_type": "variables",
	}

	return *chunk, endIdx
}

// createJavaScriptStandaloneChunk creates chunks for other JavaScript code
func (p *ASTCodeParser) createJavaScriptStandaloneChunk(lines []string, startIdx int) (*models.CodeChunk, int) {
	adaptiveSize := p.calculateAdaptiveChunkSize(lines, startIdx, "JavaScript")
	endIdx := p.min(startIdx+adaptiveSize, len(lines))

	breakIdx := p.findOptimalBreakPoint(lines, startIdx, endIdx)
	if breakIdx > startIdx {
		endIdx = breakIdx
	}

	contentLines := lines[startIdx:endIdx]
	content := strings.Join(contentLines, "\n")

	if strings.TrimSpace(content) == "" {
		return nil, endIdx
	}

	chunk := models.NewCodeChunk(content, startIdx+1, endIdx, "JavaScript")
	chunk.Metadata = map[string]interface{}{
		"start_line": startIdx + 1,
		"end_line":   endIdx,
		"chunk_type": "standalone",
	}

	return chunk, endIdx
}

// createEnhancedChunks creates enhanced chunks for generic languages
func (p *ASTCodeParser) createEnhancedChunks(content, language string) ([]models.CodeChunk, error) {
	lines := strings.Split(content, "\n")
	var chunks []models.CodeChunk

	i := 0
	for i < len(lines) {
		adaptiveSize := p.calculateAdaptiveChunkSize(lines, i, language)
		endIdx := p.min(i+adaptiveSize, len(lines))

		// Find optimal breaking point
		breakIdx := p.findOptimalBreakPoint(lines, i, endIdx)
		if breakIdx > i {
			endIdx = breakIdx
		}

		// Create chunk with overlap
		contentStart := p.max(0, i-p.chunkConfig.OverlapLines)
		contentEnd := p.min(len(lines), endIdx+p.chunkConfig.OverlapLines)

		var contentLines []string
		for j := contentStart; j < contentEnd; j++ {
			contentLines = append(contentLines, lines[j])
		}

		content := strings.Join(contentLines, "\n")

		if strings.TrimSpace(content) != "" {
			chunk := models.NewCodeChunk(content, i+1, endIdx, language)
			chunk.Metadata = map[string]interface{}{
				"start_line":  i + 1,
				"end_line":    endIdx,
				"chunk_type":  "enhanced",
				"chunk_size":  endIdx - i,
			}
			chunks = append(chunks, *chunk)
		}

		i = endIdx
	}

	return chunks, nil
}

// calculateAdaptiveChunkSize calculates chunk size based on content complexity
func (p *ASTCodeParser) calculateAdaptiveChunkSize(lines []string, startIdx int, language string) int {
	baseSize := p.chunkConfig.BaseChunkSize

	// Analyze content complexity for adaptive sizing
	windowSize := p.min(baseSize*2, len(lines)-startIdx)
	contentWindow := lines[startIdx : p.min(startIdx+windowSize, len(lines))]

	complexity := p.calculateContentComplexity(contentWindow, language)

	// Adjust size based on complexity
	adaptiveSize := int(float64(baseSize) * (1.0 + complexity*p.chunkConfig.ComplexityWeight))

	// Ensure size is within bounds
	return p.max(p.chunkConfig.MinContextLines, p.min(adaptiveSize, p.chunkConfig.MaxChunkSize))
}

// calculateContentComplexity calculates complexity score for content
func (p *ASTCodeParser) calculateContentComplexity(lines []string, language string) float64 {
	complexity := 0.0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}

		// Add complexity for various constructs
		if strings.Contains(line, "if ") || strings.Contains(line, "else") || strings.Contains(line, "elif ") {
			complexity += 0.2
		}
		if strings.Contains(line, "for ") || strings.Contains(line, "while ") {
			complexity += 0.3
		}
		if strings.Contains(line, "try ") || strings.Contains(line, "catch ") || strings.Contains(line, "except ") {
			complexity += 0.4
		}
		if strings.Contains(line, "switch ") || strings.Contains(line, "case ") {
			complexity += 0.3
		}
		if strings.Contains(line, "async ") || strings.Contains(line, "await ") {
			complexity += 0.2
		}

		// Count brackets/parentheses (proxy for nested structures)
		complexity += float64(strings.Count(line, "{")) * 0.1
		complexity += float64(strings.Count(line, "(")) * 0.05
	}

	return math.Min(complexity, 2.0) // Cap at 2x complexity
}

// findOptimalBreakPoint finds the best place to break content into chunks
func (p *ASTCodeParser) findOptimalBreakPoint(lines []string, startIdx, endIdx int) int {
	bestBreak := endIdx
	bestScore := 0.0

	for i := endIdx - 5; i <= endIdx && i > startIdx; i++ {
		if i >= len(lines) {
			continue
		}

		line := strings.TrimSpace(lines[i])
		score := 0.0

		// Prefer breaking at empty lines
		if line == "" {
			score += 10.0
		}

		// Prefer breaking after logical boundaries
		if strings.HasSuffix(line, "}") || strings.HasSuffix(line, ")") {
			score += 8.0
		}

		// Prefer breaking before function/class definitions
		if strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "def ") ||
		   strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "function ") {
			score += 6.0
		}

		// Prefer breaking after comments or docstrings
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			score += 4.0
		}

		if score > bestScore {
			bestScore = score
			bestBreak = i
		}
	}

	return bestBreak
}

// createNonStructuralChunks creates chunks for code not covered by AST nodes
func (p *ASTCodeParser) createNonStructuralChunks(content string, existingChunks []models.CodeChunk, language string) []models.CodeChunk {
	lines := strings.Split(content, "\n")

	// Mark lines covered by existing chunks
	covered := make([]bool, len(lines))
	for _, chunk := range existingChunks {
		for i := chunk.StartLine - 1; i < chunk.EndLine && i < len(covered); i++ {
			covered[i] = true
		}
	}

	var chunks []models.CodeChunk

	// Create chunks for uncovered regions
	startIdx := -1
	for i := 0; i < len(covered); i++ {
		if !covered[i] && startIdx == -1 {
			startIdx = i
		} else if covered[i] && startIdx != -1 {
			// Create chunk for uncovered region
			endIdx := i
			if endIdx > startIdx {
				contentLines := lines[startIdx:endIdx]
				content := strings.Join(contentLines, "\n")

				if strings.TrimSpace(content) != "" {
					chunk := models.NewCodeChunk(content, startIdx+1, endIdx, language)
					chunk.Metadata = map[string]interface{}{
						"start_line": startIdx + 1,
						"end_line":   endIdx,
						"chunk_type": "non_structural",
					}
					chunks = append(chunks, *chunk)
				}
			}
			startIdx = -1
		}
	}

	// Handle final uncovered region
	if startIdx != -1 && startIdx < len(lines) {
		contentLines := lines[startIdx:]
		content := strings.Join(contentLines, "\n")

		if strings.TrimSpace(content) != "" {
			chunk := models.NewCodeChunk(content, startIdx+1, len(lines), language)
			chunk.Metadata = map[string]interface{}{
				"start_line": startIdx + 1,
				"end_line":   len(lines),
				"chunk_type": "non_structural",
			}
			chunks = append(chunks, *chunk)
		}
	}

	return chunks
}

// GetEmbedding generates a vector embedding for text (delegates to simple parser)
func (p *ASTCodeParser) GetEmbedding(text string) ([]float64, error) {
	return p.simpleParser.GetEmbedding(text)
}

// GetSupportedFileTypes returns the list of supported file types
func (p *ASTCodeParser) GetSupportedFileTypes() []string {
	return p.simpleParser.GetSupportedFileTypes()
}

// max returns the maximum of two integers
func (p *ASTCodeParser) max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two integers
func (p *ASTCodeParser) min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

