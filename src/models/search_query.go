package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// SearchQuery represents a search request from the user
type SearchQuery struct {
	QueryText      string            `json:"query_text"`
	MaxResults     int               `json:"max_results"`
	IncludeContext bool              `json:"include_context"`
	FileFilter     string            `json:"file_filter"`
	LanguageFilter string            `json:"language_filter"`
	Threshold      float64           `json:"threshold"`
	SearchType     SearchType        `json:"search_type"`
	Options        map[string]string `json:"options"`
	CreatedAt      time.Time         `json:"created_at"`
}

// SearchType represents the type of search to perform
type SearchType string

const (
	SearchTypeSemantic SearchType = "semantic" // Vector-based semantic search
	SearchTypeText     SearchType = "text"     // Traditional text-based search
	SearchTypeHybrid   SearchType = "hybrid"   // Combination of semantic and text
	SearchTypeRegex    SearchType = "regex"    // Regular expression search
	SearchTypeExact    SearchType = "exact"    // Exact phrase match
	SearchTypeFuzzy    SearchType = "fuzzy"    // Fuzzy string matching
)

// NewSearchQuery creates a new SearchQuery with default values
func NewSearchQuery(queryText string) *SearchQuery {
	return &SearchQuery{
		QueryText:      strings.TrimSpace(queryText),
		MaxResults:     10,               // Default max results
		IncludeContext: false,            // Default: don't include context
		FileFilter:     "",               // No file filter by default
		LanguageFilter: "",               // No language filter by default
		Threshold:      0.7,              // Default similarity threshold
		SearchType:     SearchTypeHybrid, // Default to hybrid search
		Options:        make(map[string]string),
		CreatedAt:      time.Now(),
	}
}

// Validate validates the search query
func (sq *SearchQuery) Validate() error {
	if strings.TrimSpace(sq.QueryText) == "" {
		return fmt.Errorf("query text cannot be empty")
	}

	if sq.MaxResults < 1 {
		return fmt.Errorf("max results must be >= 1, got %d", sq.MaxResults)
	}

	if sq.MaxResults > 1000 {
		return fmt.Errorf("max results cannot exceed 1000, got %d", sq.MaxResults)
	}

	if sq.Threshold < 0 || sq.Threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1, got %f", sq.Threshold)
	}

	// Validate search type
	validTypes := map[SearchType]bool{
		SearchTypeSemantic: true,
		SearchTypeText:     true,
		SearchTypeHybrid:   true,
		SearchTypeRegex:    true,
		SearchTypeExact:    true,
		SearchTypeFuzzy:    true,
	}

	if !validTypes[sq.SearchType] {
		return fmt.Errorf("invalid search type: %s", sq.SearchType)
	}

	// Validate regex if search type is regex
	if sq.SearchType == SearchTypeRegex {
		if _, err := regexp.Compile(sq.QueryText); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Validate file filter pattern
	if sq.FileFilter != "" {
		if _, err := regexp.Compile(sq.FileFilter); err != nil {
			return fmt.Errorf("invalid file filter pattern: %w", err)
		}
	}

	return nil
}

// GetProcessedQuery returns the processed query text for searching
func (sq *SearchQuery) GetProcessedQuery() string {
	query := strings.TrimSpace(sq.QueryText)

	// Apply different processing based on search type
	switch sq.SearchType {
	case SearchTypeExact:
		// Return as-is for exact matching
		return query
	case SearchTypeFuzzy:
		// Add fuzzy matching markers
		return addFuzzyMarkers(query)
	case SearchTypeSemantic, SearchTypeHybrid:
		// Clean and normalize for semantic search
		return normalizeQuery(query)
	case SearchTypeText:
		// Simple text normalization
		return strings.ToLower(query)
	case SearchTypeRegex:
		// Return as-is (already validated)
		return query
	default:
		return query
	}
}

// GetKeywords extracts keywords from the query
func (sq *SearchQuery) GetKeywords() []string {
	query := strings.ToLower(sq.QueryText)

	// Remove common stop words
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true, "but": true, "by": true,
		"for": true, "if": true, "in": true, "into": true, "is": true, "it": true, "no": true, "not": true,
		"of": true, "on": true, "or": true, "such": true, "that": true, "the": true, "their": true, "then": true,
		"there": true, "these": true, "they": true, "this": true, "to": true, "was": true, "will": true,
		"with": true, "have": true, "has": true, "had": true, "what": true, "when": true, "where": true,
		"who": true, "why": true, "how": true, "can": true, "could": true, "should": true, "would": true,
	}

	// Split query into words
	words := strings.Fields(query)

	var keywords []string
	seen := make(map[string]bool)

	for _, word := range words {
		// Clean the word
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if word == "" {
			continue
		}

		// Skip stop words
		if stopWords[word] {
			continue
		}

		// Skip very short words
		if len(word) < 2 {
			continue
		}

		// Add to keywords if not already seen
		if !seen[word] {
			keywords = append(keywords, word)
			seen[word] = true
		}
	}

	return keywords
}

// GetBoostTerms returns terms that should be boosted in search
func (sq *SearchQuery) GetBoostTerms() []string {
	query := strings.ToLower(sq.QueryText)
	boostTerms := []string{}

	// Look for programming-specific patterns
	patterns := map[string]string{
		`\b(func|function|def|method)\s+(\w+)`: "function_definition",
		`\b(class|struct|interface)\s+(\w+)`:   "class_definition",
		`\b(import|include|require)\s+(\w+)`:   "import_statement",
		`\b(var|let|const)\s+(\w+)`:            "variable_declaration",
		`\b(return)\b`:                         "return_statement",
		`\b(if|else|for|while|switch)\b`:       "control_flow",
		`\b(try|catch|throw|raise)\b`:          "error_handling",
		`\b(test|spec|mock|assert)\b`:          "testing",
		`\b(api|endpoint|route|handler)\b`:     "api_related",
		`\b(database|db|query|sql)\b`:          "database_related",
		`\b(http|request|response)\b`:          "http_related",
	}

	for pattern, termType := range patterns {
		if matched, _ := regexp.MatchString(pattern, query); matched {
			boostTerms = append(boostTerms, termType)
		}
	}

	// Look for exact phrases in quotes
	quoteRegex := regexp.MustCompile(`"([^"]+)"`)
	matches := quoteRegex.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		if len(match) > 1 {
			boostTerms = append(boostTerms, strings.TrimSpace(match[1]))
		}
	}

	return boostTerms
}

// ShouldIncludeFile checks if a file should be included based on filters
func (sq *SearchQuery) ShouldIncludeFile(filePath, language string) bool {
	// Check file filter
	if sq.FileFilter != "" {
		matched, err := regexp.MatchString(sq.FileFilter, filePath)
		if err != nil || !matched {
			return false
		}
	}

	// Check language filter
	if sq.LanguageFilter != "" {
		if !strings.EqualFold(language, sq.LanguageFilter) {
			return false
		}
	}

	return true
}

// GetEstimatedComplexity returns an estimate of query complexity
func (sq *SearchQuery) GetEstimatedComplexity() QueryComplexity {
	complexity := QueryComplexity{
		Score:      0.0,
		Factors:    []string{},
		Processing: "simple",
	}

	// Base complexity from query length
	queryLength := len(sq.QueryText)
	if queryLength > 100 {
		complexity.Score += 0.3
		complexity.Factors = append(complexity.Factors, "long_query")
	} else if queryLength > 50 {
		complexity.Score += 0.1
		complexity.Factors = append(complexity.Factors, "medium_query")
	}

	// Complexity from search type
	switch sq.SearchType {
	case SearchTypeSemantic:
		complexity.Score += 0.4
		complexity.Factors = append(complexity.Factors, "semantic_search")
		complexity.Processing = "vector_embedding"
	case SearchTypeHybrid:
		complexity.Score += 0.3
		complexity.Factors = append(complexity.Factors, "hybrid_search")
		complexity.Processing = "combined"
	case SearchTypeRegex:
		complexity.Score += 0.2
		complexity.Factors = append(complexity.Factors, "regex_matching")
		complexity.Processing = "pattern_matching"
	case SearchTypeFuzzy:
		complexity.Score += 0.2
		complexity.Factors = append(complexity.Factors, "fuzzy_matching")
		complexity.Processing = "approximate_matching"
	}

	// Complexity from filters
	if sq.FileFilter != "" {
		complexity.Score += 0.1
		complexity.Factors = append(complexity.Factors, "file_filtering")
	}

	if sq.LanguageFilter != "" {
		complexity.Score += 0.1
		complexity.Factors = append(complexity.Factors, "language_filtering")
	}

	// Complexity from options
	if len(sq.Options) > 0 {
		complexity.Score += 0.1
		complexity.Factors = append(complexity.Factors, "custom_options")
	}

	// Determine processing level
	if complexity.Score >= 0.7 {
		complexity.Processing = "complex"
	} else if complexity.Score >= 0.4 {
		complexity.Processing = "moderate"
	}

	// Cap score at 1.0
	if complexity.Score > 1.0 {
		complexity.Score = 1.0
	}

	return complexity
}

// SetOption sets a custom option
func (sq *SearchQuery) SetOption(key, value string) {
	if sq.Options == nil {
		sq.Options = make(map[string]string)
	}
	sq.Options[key] = value
}

// GetOption gets a custom option
func (sq *SearchQuery) GetOption(key string) (string, bool) {
	if sq.Options == nil {
		return "", false
	}
	value, exists := sq.Options[key]
	return value, exists
}

// Clone creates a copy of the search query
func (sq *SearchQuery) Clone() *SearchQuery {
	clone := *sq
	clone.Options = make(map[string]string)
	for k, v := range sq.Options {
		clone.Options[k] = v
	}
	return &clone
}

// ToJSON converts the search query to JSON
func (sq *SearchQuery) ToJSON() ([]byte, error) {
	return json.MarshalIndent(sq, "", "  ")
}

// FromJSON creates a SearchQuery from JSON data
func (sq *SearchQuery) FromJSON(data []byte) error {
	var query SearchQuery
	if err := json.Unmarshal(data, &query); err != nil {
		return fmt.Errorf("failed to unmarshal search query: %w", err)
	}

	*sq = query
	return nil
}

// GetSummary returns a human-readable summary of the query
func (sq *SearchQuery) GetSummary() string {
	summary := fmt.Sprintf("Query: %q", sq.QueryText)

	if sq.MaxResults != 10 {
		summary += fmt.Sprintf(" (max %d results)", sq.MaxResults)
	}

	if sq.SearchType != SearchTypeHybrid {
		summary += fmt.Sprintf(" [%s]", sq.SearchType)
	}

	if sq.FileFilter != "" {
		summary += fmt.Sprintf(" (files: %s)", sq.FileFilter)
	}

	if sq.LanguageFilter != "" {
		summary += fmt.Sprintf(" (language: %s)", sq.LanguageFilter)
	}

	if sq.IncludeContext {
		summary += " (with context)"
	}

	return summary
}

// QueryComplexity represents the complexity analysis of a search query
type QueryComplexity struct {
	Score      float64  `json:"score"`      // 0.0 to 1.0
	Factors    []string `json:"factors"`    // Complexity factors
	Processing string   `json:"processing"` // Processing level
}

// Helper functions

// normalizeQuery normalizes query text for semantic search
func normalizeQuery(query string) string {
	// Convert to lowercase
	query = strings.ToLower(query)

	// Remove extra whitespace
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")

	// Remove special characters but keep programming symbols
	query = regexp.MustCompile(`[^\w\s\(\)\[\]\{\}\.\,\;\:\+\-\*\/\=\!\<\>\&\|]`).ReplaceAllString(query, " ")

	// Trim whitespace
	query = strings.TrimSpace(query)

	return query
}

// addFuzzyMarkers adds markers for fuzzy matching
func addFuzzyMarkers(query string) string {
	// This is a simplified implementation
	// In a real system, you'd use proper fuzzy search algorithms
	return "~" + query + "~"
}
