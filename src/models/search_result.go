package models

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// SearchResult represents a single match found during search
type SearchResult struct {
	FilePath        string                 `json:"file_path"`
	StartLine       int                    `json:"start_line"`
	EndLine         int                    `json:"end_line"`
	Content         string                 `json:"content"`
	Context         string                 `json:"context"`
	RelevanceScore  float64                `json:"relevance_score"`
	VectorDistance  float64                `json:"vector_distance"`
	Language        string                 `json:"language"`
	MatchType       MatchType              `json:"match_type"`
	Highlights      []string               `json:"highlights"`
	Metadata        map[string]interface{} `json:"metadata"`
	Rank            int                    `json:"rank"`
	FoundAt         time.Time              `json:"found_at"`
}

// MatchType represents the type of match found
type MatchType string

const (
	MatchTypeExact      MatchType = "exact"      // Exact string match
	MatchTypeSemantic   MatchType = "semantic"   // Vector similarity match
	MatchTypeFuzzy      MatchType = "fuzzy"      // Fuzzy string match
	MatchTypeRegex      MatchType = "regex"      // Regular expression match
	MatchTypePartial    MatchType = "partial"    // Partial word match
	MatchTypeSynonym    MatchType = "synonym"    // Synonym match
	MatchTypeHybrid     MatchType = "hybrid"     // Combination of multiple types
)

// NewSearchResult creates a new SearchResult with default values
func NewSearchResult(filePath string, startLine, endLine int, content string) *SearchResult {
	return &SearchResult{
		FilePath:       filePath,
		StartLine:      startLine,
		EndLine:        endLine,
		Content:        strings.TrimSpace(content),
		Context:        "",                 // Will be populated separately
		RelevanceScore: 0.0,                // Will be calculated
		VectorDistance:  0.0,                // Will be calculated for semantic matches
		Language:       "",                 // Will be detected
		MatchType:      MatchTypePartial,    // Default match type
		Highlights:     make([]string, 0),
		Metadata:       make(map[string]interface{}),
		Rank:           0,                   // Will be set during ranking
		FoundAt:        time.Now(),
	}
}

// FromVectorResult creates a SearchResult from a vector search result
func FromVectorResult(vectorResult VectorSearchResult, filePath string, startLine, endLine int, content string) *SearchResult {
	result := NewSearchResult(filePath, startLine, endLine, content)
	result.VectorDistance = vectorResult.Score
	result.RelevanceScore = calculateRelevanceScore(vectorResult.Score)
	result.MatchType = MatchTypeSemantic

	// Add metadata from vector search
	if vectorResult.Metadata != nil {
		for k, v := range vectorResult.Metadata {
			result.Metadata[k] = v
		}
	}

	return result
}

// Validate validates the search result
func (sr *SearchResult) Validate() error {
	if sr.FilePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	if sr.StartLine < 1 {
		return fmt.Errorf("start line must be >= 1, got %d", sr.StartLine)
	}

	if sr.EndLine < sr.StartLine {
		return fmt.Errorf("end line must be >= start line, got start=%d, end=%d",
			sr.StartLine, sr.EndLine)
	}

	if sr.Content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	if sr.RelevanceScore < 0 || sr.RelevanceScore > 1 {
		return fmt.Errorf("relevance score must be between 0 and 1, got %f", sr.RelevanceScore)
	}

	if sr.VectorDistance < 0 || sr.VectorDistance > 1 {
		return fmt.Errorf("vector distance must be between 0 and 1, got %f", sr.VectorDistance)
	}

	// Validate match type
	validTypes := map[MatchType]bool{
		MatchTypeExact:   true,
		MatchTypeSemantic: true,
		MatchTypeFuzzy:   true,
		MatchTypeRegex:   true,
		MatchTypePartial: true,
		MatchTypeSynonym: true,
		MatchTypeHybrid:  true,
	}

	if !validTypes[sr.MatchType] {
		return fmt.Errorf("invalid match type: %s", sr.MatchType)
	}

	return nil
}

// GetLineCount returns the number of lines in the result
func (sr *SearchResult) GetLineCount() int {
	return sr.EndLine - sr.StartLine + 1
}

// GetFileName returns the file name without path
func (sr *SearchResult) GetFileName() string {
	return filepath.Base(sr.FilePath)
}

// GetFileExtension returns the file extension
func (sr *SearchResult) GetFileExtension() string {
	return filepath.Ext(sr.FilePath)
}

// GetRelativePath returns the relative path from a base directory
func (sr *SearchResult) GetRelativePath(baseDir string) (string, error) {
	relPath, err := filepath.Rel(baseDir, sr.FilePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}
	return relPath, nil
}

// AddHighlight adds a highlighted portion of the content
func (sr *SearchResult) AddHighlight(highlight string) {
	highlight = strings.TrimSpace(highlight)
	if highlight != "" {
		sr.Highlights = append(sr.Highlights, highlight)
	}
}

// ClearHighlights clears all highlights
func (sr *SearchResult) ClearHighlights() {
	sr.Highlights = make([]string, 0)
}

// GetHighlightedContent returns the content with highlights applied
func (sr *SearchResult) GetHighlightedContent() string {
	if len(sr.Highlights) == 0 {
		return sr.Content
	}

	highlighted := sr.Content
	for _, highlight := range sr.Highlights {
		// Simple highlighting - wrap with marks
		highlighted = strings.ReplaceAll(highlighted, highlight, ">>>"+highlight+"<<<")
	}
	return highlighted
}

// AddMetadata adds metadata to the result
func (sr *SearchResult) AddMetadata(key string, value interface{}) {
	if sr.Metadata == nil {
		sr.Metadata = make(map[string]interface{})
	}
	sr.Metadata[key] = value
}

// GetMetadata retrieves metadata value
func (sr *SearchResult) GetMetadata(key string) (interface{}, bool) {
	if sr.Metadata == nil {
		return nil, false
	}
	value, exists := sr.Metadata[key]
	return value, exists
}

// UpdateScore updates the relevance score and recalculates derived values
func (sr *SearchResult) UpdateScore(newScore float64) {
	sr.RelevanceScore = newScore
	sr.updateDerivedScores()
}

// updateDerivedScores updates scores derived from the main relevance score
func (sr *SearchResult) updateDerivedScores() {
	// Update vector distance based on relevance score for semantic matches
	if sr.MatchType == MatchTypeSemantic {
		sr.VectorDistance = 1.0 - sr.RelevanceScore
	}
}

// CalculateContext generates surrounding context for the result
func (sr *SearchResult) CalculateContext(contextLines int) (string, error) {
	if contextLines <= 0 {
		return "", nil
	}

	// This would typically read the actual file and extract context
	// For now, we'll use the existing context or generate a placeholder
	if sr.Context != "" {
		return sr.Context, nil
	}

	// Placeholder implementation
	context := fmt.Sprintf("Context around lines %d-%d in %s",
		sr.StartLine, sr.EndLine, sr.GetFileName())
	return context, nil
}

// GetMatchInfo returns detailed information about the match
func (sr *SearchResult) GetMatchInfo() MatchInfo {
	return MatchInfo{
		FilePath:        sr.FilePath,
		FileName:        sr.GetFileName(),
		Extension:       sr.GetFileExtension(),
		Language:        sr.Language,
		StartLine:       sr.StartLine,
		EndLine:         sr.EndLine,
		LineCount:       sr.GetLineCount(),
		RelevanceScore:  sr.RelevanceScore,
		VectorDistance:  sr.VectorDistance,
		MatchType:       sr.MatchType,
		HighlightCount:  len(sr.Highlights),
		HasContext:      sr.Context != "",
		Rank:            sr.Rank,
	}
}

// IsBetterThan compares this result with another and returns true if it's better
func (sr *SearchResult) IsBetterThan(other *SearchResult) bool {
	if sr == nil {
		return false
	}
	if other == nil {
		return true
	}

	// Higher relevance score is better
	if sr.RelevanceScore != other.RelevanceScore {
		return sr.RelevanceScore > other.RelevanceScore
	}

	// Lower vector distance is better for semantic matches
	if sr.MatchType == MatchTypeSemantic && other.MatchType == MatchTypeSemantic {
		if sr.VectorDistance != other.VectorDistance {
			return sr.VectorDistance < other.VectorDistance
		}
	}

	// Prefer exact matches over partial matches
	if sr.MatchType == MatchTypeExact && other.MatchType != MatchTypeExact {
		return true
	}
	if sr.MatchType != MatchTypeExact && other.MatchType == MatchTypeExact {
		return false
	}

	// Prefer shorter, more focused results
	srLineCount := sr.GetLineCount()
	otherLineCount := other.GetLineCount()
	if srLineCount != otherLineCount {
		return srLineCount < otherLineCount
	}

	// Finally, prefer results that were found earlier
	return sr.FoundAt.Before(other.FoundAt)
}

// ToJSON converts the search result to JSON
func (sr *SearchResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(sr, "", "  ")
}

// FromJSON creates a SearchResult from JSON data
func (sr *SearchResult) FromJSON(data []byte) error {
	var result SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to unmarshal search result: %w", err)
	}

	*sr = result
	return nil
}

// SetContext sets the context for the search result
func (sr *SearchResult) SetContext(context string) {
	sr.Context = context
}

// GetDisplayFormat returns the result formatted for display
func (sr *SearchResult) GetDisplayFormat() string {
	lines := []string{
		fmt.Sprintf("%d. %s:%d-%d", sr.Rank, sr.FilePath, sr.StartLine, sr.EndLine),
		fmt.Sprintf("   Score: %.3f | Type: %s", sr.RelevanceScore, sr.MatchType),
	}

	// Add content (truncated if too long)
	content := sr.Content
	if len(content) > 200 {
		content = content[:197] + "..."
	}
	lines = append(lines, fmt.Sprintf("   %s", content))

	// Add highlights if any
	if len(sr.Highlights) > 0 {
		lines = append(lines, fmt.Sprintf("   Highlights: %s", strings.Join(sr.Highlights, "; ")))
	}

	// Add context if available
	if sr.Context != "" {
		context := sr.Context
		if len(context) > 150 {
			context = context[:147] + "..."
		}
		lines = append(lines, fmt.Sprintf("   Context: %s", context))
	}

	return strings.Join(lines, "\n")
}

// MatchInfo contains detailed information about a match
type MatchInfo struct {
	FilePath       string    `json:"file_path"`
	FileName       string    `json:"file_name"`
	Extension      string    `json:"extension"`
	Language       string    `json:"language"`
	StartLine      int       `json:"start_line"`
	EndLine        int       `json:"end_line"`
	LineCount      int       `json:"line_count"`
	RelevanceScore float64   `json:"relevance_score"`
	VectorDistance float64   `json:"vector_distance"`
	MatchType      MatchType `json:"match_type"`
	HighlightCount int       `json:"highlight_count"`
	HasContext     bool      `json:"has_context"`
	Rank           int       `json:"rank"`
}

// Helper functions

// calculateRelevanceScore calculates relevance score from vector distance
func calculateRelevanceScore(vectorDistance float64) float64 {
	if vectorDistance <= 0 {
		return 1.0
	}
	if vectorDistance >= 1 {
		return 0.0
	}
	return 1.0 - vectorDistance
}