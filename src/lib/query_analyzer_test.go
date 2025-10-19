// ABOUTME: Test the query analyzer functionality
package lib

import (
	"testing"
)

func TestQueryAnalyzer_AnalyzeQuery(t *testing.T) {
	analyzer := NewQueryAnalyzer()

	tests := []struct {
		name     string
		query    string
		expected QueryType
	}{
		// Exact queries
		{"TODO keyword", "TODO", QueryTypeExact},
		{"FIXME keyword", "FIXME", QueryTypeExact},
		{"Quoted phrase", `"exact phrase"`, QueryTypeExact},
		{"TODO with colon", "TODO: implement this", QueryTypeExact},

		// Regex queries
		{"Regex pattern", "func.*error", QueryTypeRegex},
		{"Character class", "[a-z]+", QueryTypeRegex},
		{"Quantifier", "test{2,5}", QueryTypeRegex},

		// Semantic queries
		{"Authentication", "user authentication", QueryTypeSemantic},
		{"Database query", "database connection", QueryTypeSemantic},
		{"API endpoint", "api endpoint", QueryTypeSemantic},
		{"Error handling", "error handling", QueryTypeSemantic},

		// Hybrid queries (default fallback)
		{"Simple function", "calculate", QueryTypeHybrid},
		{"Single word", "simple", QueryTypeHybrid},
		{"Empty query", "", QueryTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.AnalyzeQuery(tt.query)
			if result != tt.expected {
				t.Errorf("AnalyzeQuery(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestQueryAnalyzer_HelperMethods(t *testing.T) {
	analyzer := NewQueryAnalyzer()

	// Test ShouldUseSemanticSearch
	if !analyzer.ShouldUseSemanticSearch("user authentication") {
		t.Error("ShouldUseSemanticSearch failed for semantic query")
	}
	if analyzer.ShouldUseSemanticSearch("TODO") {
		t.Error("ShouldUseSemanticSearch should not return true for TODO")
	}

	// Test ShouldUseExactMatch
	if !analyzer.ShouldUseExactMatch("TODO") {
		t.Error("ShouldUseExactMatch failed for TODO")
	}
	if analyzer.ShouldUseExactMatch("user authentication") {
		t.Error("ShouldUseExactMatch should not return true for semantic query")
	}

	// Test ShouldUseRegex
	if !analyzer.ShouldUseRegex("func.*error") {
		t.Error("ShouldUseRegex failed for regex pattern")
	}
	if analyzer.ShouldUseRegex("TODO") {
		t.Error("ShouldUseRegex should not return true for TODO")
	}
}

func TestQueryAnalyzer_GetQueryTypeString(t *testing.T) {
	analyzer := NewQueryAnalyzer()

	tests := map[QueryType]string{
		QueryTypeExact:    "exact",
		QueryTypeRegex:    "regex",
		QueryTypeSemantic: "semantic",
		QueryTypeHybrid:   "hybrid",
		QueryTypeUnknown:  "unknown",
	}

	for qType, expected := range tests {
		result := analyzer.GetQueryTypeString(qType)
		if result != expected {
			t.Errorf("GetQueryTypeString(%v) = %q, want %q", qType, result, expected)
		}
	}
}

func TestQueryAnalyzer_GetRecommendedSearchType(t *testing.T) {
	analyzer := NewQueryAnalyzer()

	tests := map[string]string{
		"TODO":               "exact",
		"func.*error":        "regex",
		"user authentication": "semantic",
		"database":           "semantic", // database is a semantic keyword
		"calculate":          "hybrid",
	}

	for query, expected := range tests {
		result := analyzer.GetRecommendedSearchType(query)
		if result != expected {
			t.Errorf("GetRecommendedSearchType(%q) = %q, want %q", query, result, expected)
		}
	}
}