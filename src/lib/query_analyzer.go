// ABOUTME: Analyzes search queries to determine optimal search strategies
package lib

import (
	"regexp"
	"strings"
)

// QueryType represents the type of query based on pattern analysis
type QueryType int

const (
	QueryTypeUnknown QueryType = iota
	QueryTypeExact      // "TODO", "FIXME" - exact strings
	QueryTypeRegex      // "func.*error" - patterns
	QueryTypeSemantic   // "user authentication" - concepts
	QueryTypeHybrid     // "calculate tax" - function + concept
)

// QueryAnalyzer analyzes queries to determine optimal search strategies
type QueryAnalyzer struct {
	exactPatterns   []*regexp.Regexp
	regexPatterns   []*regexp.Regexp
	semanticKeywords []string
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer() *QueryAnalyzer {
	return &QueryAnalyzer{
		exactPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^\b(TODO|FIXME|HACK|XXX|BUG|NOTE)\b$`),
			regexp.MustCompile(`(?i)^\b(TODO|FIXME|HACK|XXX|BUG|NOTE):\s`),
			regexp.MustCompile(`^"[^"]+"$`), // Exact phrases in quotes
		},
		regexPatterns: []*regexp.Regexp{
			regexp.MustCompile(`[\.\*\+\?\|\[\]\(\)\{\}\^\$\\]`), // Regex metacharacters
			regexp.MustCompile(`\[.*\]`),                         // Character classes
			regexp.MustCompile(`\{.*\}`),                         // Quantifiers
			regexp.MustCompile(`\(.+\)`),                         // Groups
		},
		semanticKeywords: []string{
			"authentication", "authorization", "login", "logout", "user", "account",
			"database", "query", "insert", "update", "delete", "select", "sql",
			"api", "endpoint", "route", "request", "response", "http", "rest",
			"function", "method", "class", "interface", "object", "variable",
			"error", "exception", "try", "catch", "throw", "handling",
			"test", "mock", "spec", "assert", "verify", "fixture",
			"config", "setting", "parameter", "option", "property",
			"util", "helper", "common", "shared", "base", "abstract",
			"cache", "memory", "performance", "optimize", "speed",
			"security", "encrypt", "decrypt", "hash", "token", "key",
			"file", "directory", "path", "stream", "buffer", "io",
			"validation", "check", "verify", "ensure", "guard",
			"logging", "debug", "trace", "monitor", "metric", "log",
			"event", "listener", "observer", "publisher", "subscriber",
			"service", "component", "module", "package", "library",
		},
	}
}

// AnalyzeQuery determines the type of search query based on its content
func (qa *QueryAnalyzer) AnalyzeQuery(query string) QueryType {
	query = strings.TrimSpace(query)

	// Empty query
	if query == "" {
		return QueryTypeUnknown
	}

	// Check for exact patterns first (highest priority)
	if qa.isExactQuery(query) {
		return QueryTypeExact
	}

	// Check for regex patterns
	if qa.isRegexQuery(query) {
		return QueryTypeRegex
	}

	// Check for semantic queries
	if qa.isSemanticQuery(query) {
		return QueryTypeSemantic
	}

	// Default to hybrid for ambiguous queries
	return QueryTypeHybrid
}

// isExactQuery checks if the query should use exact matching
func (qa *QueryAnalyzer) isExactQuery(query string) bool {
	// Check against exact patterns
	for _, pattern := range qa.exactPatterns {
		if pattern.MatchString(query) {
			return true
		}
	}

	// Check for common exact-match keywords
	exactKeywords := []string{
		"TODO", "FIXME", "HACK", "XXX", "BUG", "NOTE", "DEPRECATED",
		"main", "init", "setup", "cleanup", "destroy",
	}

	upperQuery := strings.ToUpper(query)
	for _, keyword := range exactKeywords {
		if upperQuery == keyword {
			return true
		}
	}

	// Check for quoted phrases (remove quotes for checking)
	unquoted := strings.Trim(query, `"`)
	if unquoted != query && len(unquoted) > 0 {
		// This was a quoted phrase, treat as exact
		return true
	}

	return false
}

// isRegexQuery checks if the query contains regex patterns
func (qa *QueryAnalyzer) isRegexQuery(query string) bool {
	// Check against regex patterns
	for _, pattern := range qa.regexPatterns {
		if pattern.MatchString(query) {
			return true
		}
	}

	// Check for common regex constructs
	regexConstructs := []string{
		".*", ".+", ".?", ".*?", "[a-z]", "[0-9]", "\\d", "\\w", "\\s",
		"^", "$", "|", "(", ")", "*", "+", "?", "{", "}",
	}

	for _, construct := range regexConstructs {
		if strings.Contains(query, construct) {
			// Make sure it's not a simple file pattern
			if !qa.isFilePattern(query) {
				return true
			}
		}
	}

	return false
}

// isSemanticQuery checks if the query contains semantic/conceptual terms
func (qa *QueryAnalyzer) isSemanticQuery(query string) bool {
	lowerQuery := strings.ToLower(query)

	// Check for semantic keywords
	for _, keyword := range qa.semanticKeywords {
		if strings.Contains(lowerQuery, keyword) {
			return true
		}
	}

	// Check for multi-word conceptual queries (indicates semantic intent)
	words := strings.Fields(lowerQuery)
	if len(words) >= 2 {
		// Look for concept combinations
		conceptPairs := [][]string{
			{"user", "auth"}, {"user", "login"}, {"account", "create"},
			{"database", "connection"}, {"data", "access"}, {"query", "performance"},
			{"api", "endpoint"}, {"http", "request"}, {"rest", "service"},
			{"error", "handling"}, {"exception", "management"}, {"log", "level"},
			{"file", "upload"}, {"file", "processing"}, {"stream", "data"},
			{"config", "management"}, {"setting", "update"}, {"parameter", "validation"},
			{"test", "coverage"}, {"unit", "test"}, {"integration", "test"},
			{"cache", "invalidation"}, {"memory", "usage"}, {"performance", "optimization"},
			{"security", "check"}, {"access", "control"}, {"permission", "manage"},
		}

		for _, pair := range conceptPairs {
			if strings.Contains(lowerQuery, pair[0]) && strings.Contains(lowerQuery, pair[1]) {
				return true
			}
		}
	}

	// Check for question-like queries (semantic intent)
	questionWords := []string{"how", "what", "where", "when", "why", "which", "find", "search", "look"}
	for _, word := range questionWords {
		if strings.HasPrefix(lowerQuery, word+" ") {
			return true
		}
	}

	return false
}

// isFilePattern checks if the query is likely a file pattern rather than regex
func (qa *QueryAnalyzer) isFilePattern(query string) bool {
	// Common file patterns with simple wildcards
	filePatternRegex := regexp.MustCompile(`^[\w\-\./\\\*]+\.(go|js|ts|py|java|cpp|c|h|cs|php|rb|swift|kt|rs|scala|clj|hs|ml|fs|vim|sh|bat|ps1|json|xml|yaml|yml|toml|ini|cfg|conf|md|txt|log|sql|html|css|scss|sass|less)$`)
	return filePatternRegex.MatchString(query)
}

// GetQueryTypeString returns a string representation of the query type
func (qa *QueryAnalyzer) GetQueryTypeString(qType QueryType) string {
	switch qType {
	case QueryTypeExact:
		return "exact"
	case QueryTypeRegex:
		return "regex"
	case QueryTypeSemantic:
		return "semantic"
	case QueryTypeHybrid:
		return "hybrid"
	default:
		return "unknown"
	}
}

// ShouldUseSemanticSearch determines if semantic search should be used
func (qa *QueryAnalyzer) ShouldUseSemanticSearch(query string) bool {
	qType := qa.AnalyzeQuery(query)
	return qType == QueryTypeSemantic || qType == QueryTypeHybrid
}

// ShouldUseExactMatch determines if exact matching should be used
func (qa *QueryAnalyzer) ShouldUseExactMatch(query string) bool {
	return qa.AnalyzeQuery(query) == QueryTypeExact
}

// ShouldUseRegex determines if regex search should be used
func (qa *QueryAnalyzer) ShouldUseRegex(query string) bool {
	return qa.AnalyzeQuery(query) == QueryTypeRegex
}

// GetRecommendedSearchType returns the recommended search type for the query
func (qa *QueryAnalyzer) GetRecommendedSearchType(query string) string {
	qType := qa.AnalyzeQuery(query)
	return qa.GetQueryTypeString(qType)
}