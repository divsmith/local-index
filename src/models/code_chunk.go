package models

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CodeChunk represents a segment of code that has been vectorized for search
type CodeChunk struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	StartLine int       `json:"start_line"`
	EndLine   int       `json:"end_line"`
	Vector    []float64 `json:"vector"`
	Context   string    `json:"context"`
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"created_at"`
}

// NewCodeChunk creates a new CodeChunk
func NewCodeChunk(content string, startLine, endLine int, language string) *CodeChunk {
	return &CodeChunk{
		ID:        generateChunkID(content, startLine, endLine),
		Content:   strings.TrimSpace(content),
		StartLine: startLine,
		EndLine:   endLine,
		Vector:    make([]float64, 0), // Will be populated by indexing service
		Context:   "",                 // Will be populated by indexing service
		Language:  language,
		CreatedAt: time.Now(),
	}
}

// SetVector sets the vector representation of the chunk
func (cc *CodeChunk) SetVector(vector []float64) error {
	if len(vector) == 0 {
		return fmt.Errorf("vector cannot be empty")
	}

	cc.Vector = make([]float64, len(vector))
	copy(cc.Vector, vector)

	return nil
}

// SetContext sets the context for the chunk
func (cc *CodeChunk) SetContext(context string) {
	cc.Context = strings.TrimSpace(context)
}

// GetLineCount returns the number of lines in the chunk
func (cc *CodeChunk) GetLineCount() int {
	return cc.EndLine - cc.StartLine + 1
}

// GetSize returns the size of the chunk in characters
func (cc *CodeChunk) GetSize() int {
	return len(cc.Content)
}

// GetWordCount returns the number of words in the chunk
func (cc *CodeChunk) GetWordCount() int {
	if cc.Content == "" {
		return 0
	}

	// Simple word count - split by whitespace
	words := strings.Fields(cc.Content)
	return len(words)
}

// ContainsKeyword checks if the chunk contains a specific keyword
func (cc *CodeChunk) ContainsKeyword(keyword string) bool {
	if keyword == "" {
		return false
	}

	return strings.Contains(strings.ToLower(cc.Content), strings.ToLower(keyword))
}

// GetKeywords extracts important keywords from the chunk
func (cc *CodeChunk) GetKeywords() []string {
	// Simple keyword extraction based on common programming patterns
	content := strings.ToLower(cc.Content)

	// Common programming keywords
	programmingKeywords := []string{
		"function", "func", "def", "class", "method", "interface",
		"import", "package", "module", "library", "api", "service",
		"database", "query", "select", "insert", "update", "delete",
		"error", "exception", "try", "catch", "throw", "return",
		"if", "else", "for", "while", "loop", "break", "continue",
		"var", "let", "const", "int", "string", "bool", "float",
		"array", "list", "map", "dict", "object", "struct",
		"http", "request", "response", "json", "xml", "html",
		"test", "mock", "assert", "verify", "validate",
	}

	var keywords []string
	seen := make(map[string]bool)

	for _, keyword := range programmingKeywords {
		if strings.Contains(content, keyword) && !seen[keyword] {
			keywords = append(keywords, keyword)
			seen[keyword] = true
		}
	}

	// Extract function names, variable names, etc. (simplified)
	lines := strings.Split(cc.Content, "\n")
	for _, line := range lines {
		// Look for function definitions
		if strings.Contains(line, "func ") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "func" && i+1 < len(parts) {
					funcName := strings.TrimSuffix(parts[i+1], "(")
					if funcName != "" && !seen[funcName] {
						keywords = append(keywords, funcName)
						seen[funcName] = true
					}
				}
			}
		}

		// Look for variable declarations
		if strings.Contains(line, "var ") || strings.Contains(line, "let ") || strings.Contains(line, "const ") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if (part == "var" || part == "let" || part == "const") && i+1 < len(parts) {
					varName := strings.TrimSuffix(parts[i+1], "=")
					varName = strings.TrimSuffix(varName, ";")
					varName = strings.TrimSpace(varName)
					if varName != "" && !seen[varName] {
						keywords = append(keywords, varName)
						seen[varName] = true
					}
				}
			}
		}
	}

	return keywords
}

// GetRelevanceScore calculates a relevance score for the chunk based on content characteristics
func (cc *CodeChunk) GetRelevanceScore() float64 {
	score := 0.0

	// Base score for having content
	if cc.Content != "" {
		score += 0.1
	}

	// Score based on line count (optimal range: 5-20 lines)
	lineCount := cc.GetLineCount()
	if lineCount >= 5 && lineCount <= 20 {
		score += 0.3
	} else if lineCount > 0 {
		score += 0.1
	}

	// Score based on size (optimal range: 100-1000 characters)
	size := cc.GetSize()
	if size >= 100 && size <= 1000 {
		score += 0.2
	} else if size > 0 {
		score += 0.1
	}

	// Score based on keyword density
	keywords := cc.GetKeywords()
	keywordScore := float64(len(keywords)) * 0.05
	if keywordScore > 0.3 {
		keywordScore = 0.3
	}
	score += keywordScore

	// Score for having context
	if cc.Context != "" {
		score += 0.1
	}

	// Score for having vector representation
	if len(cc.Vector) > 0 {
		score += 0.2
	}

	// Ensure score is between 0 and 1
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// Validate validates the code chunk
func (cc *CodeChunk) Validate() error {
	if cc.ID == "" {
		return fmt.Errorf("chunk ID cannot be empty")
	}

	if cc.Content == "" {
		return fmt.Errorf("chunk content cannot be empty")
	}

	if cc.StartLine < 1 {
		return fmt.Errorf("start line must be >= 1, got %d", cc.StartLine)
	}

	if cc.EndLine < cc.StartLine {
		return fmt.Errorf("end line must be >= start line, got start=%d, end=%d",
			cc.StartLine, cc.EndLine)
	}

	if len(cc.Vector) == 0 {
		return fmt.Errorf("chunk vector cannot be empty")
	}

	// Check vector values are valid
	for i, v := range cc.Vector {
		if v != v { // NaN check
			return fmt.Errorf("vector contains NaN at index %d", i)
		}
	}

	return nil
}

// MergeWith merges this chunk with another chunk if they are adjacent
func (cc *CodeChunk) MergeWith(other *CodeChunk) (*CodeChunk, error) {
	if other == nil {
		return nil, fmt.Errorf("cannot merge with nil chunk")
	}

	// Check if chunks are from the same language
	if cc.Language != other.Language {
		return nil, fmt.Errorf("cannot merge chunks from different languages: %s vs %s",
			cc.Language, other.Language)
	}

	// Check if chunks are adjacent or overlapping
	merged := false
	var newStartLine, newEndLine int
	var newContent string

	if cc.EndLine+1 == other.StartLine {
		// cc comes before other
		newStartLine = cc.StartLine
		newEndLine = other.EndLine
		newContent = cc.Content + "\n" + other.Content
		merged = true
	} else if other.EndLine+1 == cc.StartLine {
		// other comes before cc
		newStartLine = other.StartLine
		newEndLine = cc.EndLine
		newContent = other.Content + "\n" + cc.Content
		merged = true
	} else if cc.StartLine <= other.EndLine && other.StartLine <= cc.EndLine {
		// Chunks overlap - merge their content ranges
		newStartLine = min(cc.StartLine, other.StartLine)
		newEndLine = max(cc.EndLine, other.EndLine)
		newContent = cc.Content // Use the larger chunk's content
		if len(other.Content) > len(cc.Content) {
			newContent = other.Content
		}
		merged = true
	}

	if !merged {
		return nil, fmt.Errorf("chunks are not adjacent or overlapping")
	}

	// Create merged chunk
	mergedChunk := NewCodeChunk(newContent, newStartLine, newEndLine, cc.Language)

	// Merge contexts
	if cc.Context != "" && other.Context != "" {
		mergedChunk.SetContext(cc.Context + "\n" + other.Context)
	} else if cc.Context != "" {
		mergedChunk.SetContext(cc.Context)
	} else if other.Context != "" {
		mergedChunk.SetContext(other.Context)
	}

	// Vector will need to be recalculated
	mergedChunk.Vector = []float64{}

	return mergedChunk, nil
}

// Split splits the chunk into smaller chunks of maximum line count
func (cc *CodeChunk) Split(maxLines int) []*CodeChunk {
	if maxLines < 1 {
		return []*CodeChunk{cc}
	}

	lineCount := cc.GetLineCount()
	if lineCount <= maxLines {
		return []*CodeChunk{cc}
	}

	var chunks []*CodeChunk
	lines := strings.Split(cc.Content, "\n")

	for i := 0; i < len(lines); i += maxLines {
		end := i + maxLines
		if end > len(lines) {
			end = len(lines)
		}

		chunkContent := strings.Join(lines[i:end], "\n")
		startLine := cc.StartLine + i
		endLine := cc.StartLine + end - 1

		chunk := NewCodeChunk(chunkContent, startLine, endLine, cc.Language)
		chunk.SetContext(cc.Context) // Inherit context from parent
		chunks = append(chunks, chunk)
	}

	return chunks
}

// ToJSON converts the chunk to JSON
func (cc *CodeChunk) ToJSON() ([]byte, error) {
	return json.MarshalIndent(cc, "", "  ")
}

// FromJSON creates a CodeChunk from JSON data
func (cc *CodeChunk) FromJSON(data []byte) error {
	var chunk CodeChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return fmt.Errorf("failed to unmarshal code chunk: %w", err)
	}

	*cc = chunk
	return nil
}

// GetStats returns statistics about the chunk
func (cc *CodeChunk) GetStats() CodeChunkStats {
	return CodeChunkStats{
		ID:           cc.ID,
		LineCount:    cc.GetLineCount(),
		Size:         cc.GetSize(),
		WordCount:    cc.GetWordCount(),
		Language:     cc.Language,
		KeywordCount: len(cc.GetKeywords()),
		HasVector:    len(cc.Vector) > 0,
		HasContext:   cc.Context != "",
		CreatedAt:    cc.CreatedAt,
	}
}

// CodeChunkStats contains statistics about a code chunk
type CodeChunkStats struct {
	ID           string    `json:"id"`
	LineCount    int       `json:"line_count"`
	Size         int       `json:"size"`
	WordCount    int       `json:"word_count"`
	Language     string    `json:"language"`
	KeywordCount int       `json:"keyword_count"`
	HasVector    bool      `json:"has_vector"`
	HasContext   bool      `json:"has_context"`
	CreatedAt    time.Time `json:"created_at"`
}

// generateChunkID generates a unique ID for the chunk
func generateChunkID(content string, startLine, endLine int) string {
	hashInput := fmt.Sprintf("%s:%d:%d:%d", content, startLine, endLine, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(hashInput))
	return fmt.Sprintf("chunk_%x", hash[:16])
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
