package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// SearchResults represents a collection of search results
type SearchResults struct {
	Query         *SearchQuery           `json:"query"`
	Results       []*SearchResult        `json:"results"`
	TotalResults  int                    `json:"total_results"`
	ExecutionTime time.Duration          `json:"execution_time"`
	HasMore       bool                   `json:"has_more"`
	SearchedFiles int                    `json:"searched_files"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
}

// NewSearchResults creates a new SearchResults instance
func NewSearchResults(query *SearchQuery) *SearchResults {
	return &SearchResults{
		Query:         query,
		Results:       make([]*SearchResult, 0),
		TotalResults:  0,
		ExecutionTime: 0,
		HasMore:       false,
		SearchedFiles: 0,
		Metadata:      make(map[string]interface{}),
		CreatedAt:     time.Now(),
	}
}

// AddResult adds a search result to the collection
func (sr *SearchResults) AddResult(result *SearchResult) error {
	if result == nil {
		return fmt.Errorf("cannot add nil result")
	}

	if err := result.Validate(); err != nil {
		return fmt.Errorf("invalid search result: %w", err)
	}

	sr.Results = append(sr.Results, result)
	sr.TotalResults++

	return nil
}

// AddResults adds multiple search results to the collection
func (sr *SearchResults) AddResults(results []*SearchResult) error {
	for _, result := range results {
		if err := sr.AddResult(result); err != nil {
			return fmt.Errorf("failed to add result: %w", err)
		}
	}
	return nil
}

// GetResult returns a result by index
func (sr *SearchResults) GetResult(index int) (*SearchResult, error) {
	if index < 0 || index >= len(sr.Results) {
		return nil, fmt.Errorf("index out of bounds: %d (results count: %d)", index, len(sr.Results))
	}
	return sr.Results[index], nil
}

// RemoveResult removes a result by index
func (sr *SearchResults) RemoveResult(index int) error {
	if index < 0 || index >= len(sr.Results) {
		return fmt.Errorf("index out of bounds: %d (results count: %d)", index, len(sr.Results))
	}

	sr.Results = append(sr.Results[:index], sr.Results[index+1:]...)
	sr.TotalResults--

	return nil
}

// ClearResults removes all results
func (sr *SearchResults) ClearResults() {
	sr.Results = make([]*SearchResult, 0)
	sr.TotalResults = 0
}

// SortResults sorts the results by relevance score (descending)
func (sr *SearchResults) SortResults() {
	sort.Slice(sr.Results, func(i, j int) bool {
		return sr.Results[i].IsBetterThan(sr.Results[j])
	})

	// Update ranks after sorting
	for i, result := range sr.Results {
		result.Rank = i + 1
	}
}

// FilterResults filters results based on a predicate function
func (sr *SearchResults) FilterResults(predicate func(*SearchResult) bool) []*SearchResult {
	var filtered []*SearchResult

	for _, result := range sr.Results {
		if predicate(result) {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// GetResultsByFile returns results grouped by file path
func (sr *SearchResults) GetResultsByFile() map[string][]*SearchResult {
	resultsByFile := make(map[string][]*SearchResult)

	for _, result := range sr.Results {
		filePath := result.FilePath
		resultsByFile[filePath] = append(resultsByFile[filePath], result)
	}

	return resultsByFile
}

// GetResultsByLanguage returns results grouped by programming language
func (sr *SearchResults) GetResultsByLanguage() map[string][]*SearchResult {
	resultsByLanguage := make(map[string][]*SearchResult)

	for _, result := range sr.Results {
		language := result.Language
		if language == "" {
			language = "Unknown"
		}
		resultsByLanguage[language] = append(resultsByLanguage[language], result)
	}

	return resultsByLanguage
}

// GetResultsByMatchType returns results grouped by match type
func (sr *SearchResults) GetResultsByMatchType() map[MatchType][]*SearchResult {
	resultsByType := make(map[MatchType][]*SearchResult)

	for _, result := range sr.Results {
		matchType := result.MatchType
		resultsByType[matchType] = append(resultsByType[matchType], result)
	}

	return resultsByType
}

// LimitResults limits the number of results to the specified maximum
func (sr *SearchResults) LimitResults(maxResults int) {
	if maxResults >= 0 && len(sr.Results) > maxResults {
		sr.HasMore = sr.TotalResults > maxResults
		sr.Results = sr.Results[:maxResults]
	} else {
		sr.HasMore = false
	}
}

// GetTopResults returns the top N results
func (sr *SearchResults) GetTopResults(n int) []*SearchResult {
	if n <= 0 {
		return []*SearchResult{}
	}

	if n >= len(sr.Results) {
		return sr.Results
	}

	// Make a copy of the first N results
	topResults := make([]*SearchResult, n)
	copy(topResults, sr.Results[:n])

	return topResults
}

// GetAverageScore returns the average relevance score of all results
func (sr *SearchResults) GetAverageScore() float64 {
	if len(sr.Results) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, result := range sr.Results {
		totalScore += result.RelevanceScore
	}

	return totalScore / float64(len(sr.Results))
}

// GetHighestScore returns the highest relevance score
func (sr *SearchResults) GetHighestScore() float64 {
	if len(sr.Results) == 0 {
		return 0.0
	}

	highestScore := sr.Results[0].RelevanceScore
	for _, result := range sr.Results {
		if result.RelevanceScore > highestScore {
			highestScore = result.RelevanceScore
		}
	}

	return highestScore
}

// GetLowestScore returns the lowest relevance score
func (sr *SearchResults) GetLowestScore() float64 {
	if len(sr.Results) == 0 {
		return 0.0
	}

	lowestScore := sr.Results[0].RelevanceScore
	for _, result := range sr.Results {
		if result.RelevanceScore < lowestScore {
			lowestScore = result.RelevanceScore
		}
	}

	return lowestScore
}

// GetScoreDistribution returns the distribution of scores across ranges
func (sr *SearchResults) GetScoreDistribution() ScoreDistribution {
	distribution := ScoreDistribution{
		Excellent: 0, // 0.9 - 1.0
		Good:      0, // 0.7 - 0.9
		Fair:      0, // 0.5 - 0.7
		Poor:      0, // 0.3 - 0.5
		VeryPoor:  0, // 0.0 - 0.3
	}

	for _, result := range sr.Results {
		score := result.RelevanceScore
		switch {
		case score >= 0.9:
			distribution.Excellent++
		case score >= 0.7:
			distribution.Good++
		case score >= 0.5:
			distribution.Fair++
		case score >= 0.3:
			distribution.Poor++
		default:
			distribution.VeryPoor++
		}
	}

	return distribution
}

// GetStatistics returns comprehensive statistics about the search results
func (sr *SearchResults) GetStatistics() SearchStatistics {
	stats := SearchStatistics{
		Query:              sr.Query.GetSummary(),
		TotalResults:       sr.TotalResults,
		DisplayedResults:   len(sr.Results),
		ExecutionTime:      sr.ExecutionTime,
		SearchedFiles:      sr.SearchedFiles,
		AverageScore:       sr.GetAverageScore(),
		HighestScore:       sr.GetHighestScore(),
		LowestScore:        sr.GetLowestScore(),
		ScoreDistribution:  sr.GetScoreDistribution(),
		ResultsByFile:      len(sr.GetResultsByFile()),
		ResultsByLanguage:  len(sr.GetResultsByLanguage()),
		ResultsByMatchType: len(sr.GetResultsByMatchType()),
		CreatedAt:          sr.CreatedAt,
	}

	// Add file statistics
	filesByType := make(map[string]int)
	totalLines := 0
	for _, result := range sr.Results {
		ext := result.GetFileExtension()
		if ext == "" {
			ext = "no_extension"
		}
		filesByType[ext]++
		totalLines += result.GetLineCount()
	}
	stats.FileTypes = filesByType
	stats.TotalLines = totalLines

	return stats
}

// SetExecutionTime sets the execution time of the search
func (sr *SearchResults) SetExecutionTime(duration time.Duration) {
	sr.ExecutionTime = duration
}

// SetSearchedFiles sets the number of files that were searched
func (sr *SearchResults) SetSearchedFiles(count int) {
	sr.SearchedFiles = count
}

// AddMetadata adds metadata to the search results
func (sr *SearchResults) AddMetadata(key string, value interface{}) {
	if sr.Metadata == nil {
		sr.Metadata = make(map[string]interface{})
	}
	sr.Metadata[key] = value
}

// GetMetadata retrieves metadata value
func (sr *SearchResults) GetMetadata(key string) (interface{}, bool) {
	if sr.Metadata == nil {
		return nil, false
	}
	value, exists := sr.Metadata[key]
	return value, exists
}

// ToJSON converts the search results to JSON
func (sr *SearchResults) ToJSON() ([]byte, error) {
	return json.MarshalIndent(sr, "", "  ")
}

// FromJSON creates SearchResults from JSON data
func (srs *SearchResults) FromJSON(data []byte) error {
	var results SearchResults
	if err := json.Unmarshal(data, &results); err != nil {
		return fmt.Errorf("failed to unmarshal search results: %w", err)
	}

	*srs = results
	return nil
}

// ToTableFormat returns the results formatted as a table
func (sr *SearchResults) ToTableFormat() string {
	if len(sr.Results) == 0 {
		return "No results found."
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d results:", sr.TotalResults))
	lines = append(lines, "")

	for _, result := range sr.Results {
		lines = append(lines, result.GetDisplayFormat())
		lines = append(lines, "")
	}

	// Add summary
	if sr.ExecutionTime > 0 {
		lines = append(lines, fmt.Sprintf("Search completed in %v", sr.ExecutionTime))
	}

	return strings.Join(lines, "\n")
}

// ToJSONFormat returns the results formatted as JSON
func (sr *SearchResults) ToJSONFormat() (string, error) {
	data, err := json.Marshal(map[string]interface{}{
		"query":          sr.Query.QueryText,
		"results":        sr.Results,
		"total_results":  sr.TotalResults,
		"execution_time": sr.ExecutionTime.String(),
		"has_more":       sr.HasMore,
		"searched_files": sr.SearchedFiles,
		"created_at":     sr.CreatedAt,
	})

	if err != nil {
		return "", fmt.Errorf("failed to format as JSON: %w", err)
	}

	return string(data), nil
}

// Validate validates the search results
func (sr *SearchResults) Validate() error {
	if sr.Query == nil {
		return fmt.Errorf("query cannot be nil")
	}

	if err := sr.Query.Validate(); err != nil {
		return fmt.Errorf("invalid query: %w", err)
	}

	if sr.TotalResults != len(sr.Results) {
		return fmt.Errorf("total_results mismatch: expected %d, got %d",
			len(sr.Results), sr.TotalResults)
	}

	for i, result := range sr.Results {
		if err := result.Validate(); err != nil {
			return fmt.Errorf("invalid result at index %d: %w", i, err)
		}
	}

	return nil
}

// IsEmpty returns true if there are no results
func (sr *SearchResults) IsEmpty() bool {
	return len(sr.Results) == 0
}

// Clone creates a deep copy of the search results
func (sr *SearchResults) Clone() *SearchResults {
	clone := NewSearchResults(sr.Query.Clone())

	clone.TotalResults = sr.TotalResults
	clone.ExecutionTime = sr.ExecutionTime
	clone.HasMore = sr.HasMore
	clone.SearchedFiles = sr.SearchedFiles
	clone.CreatedAt = sr.CreatedAt

	// Copy metadata
	for k, v := range sr.Metadata {
		clone.Metadata[k] = v
	}

	// Copy results
	clone.Results = make([]*SearchResult, len(sr.Results))
	for i, result := range sr.Results {
		// Deep copy of result
		resultCopy := *result
		clone.Results[i] = &resultCopy
	}

	return clone
}

// ScoreDistribution represents the distribution of relevance scores
type ScoreDistribution struct {
	Excellent int `json:"excellent"` // 0.9 - 1.0
	Good      int `json:"good"`      // 0.7 - 0.9
	Fair      int `json:"fair"`      // 0.5 - 0.7
	Poor      int `json:"poor"`      // 0.3 - 0.5
	VeryPoor  int `json:"very_poor"` // 0.0 - 0.3
}

// SearchStatistics contains comprehensive statistics about search results
type SearchStatistics struct {
	Query              string            `json:"query"`
	TotalResults       int               `json:"total_results"`
	DisplayedResults   int               `json:"displayed_results"`
	ExecutionTime      time.Duration     `json:"execution_time"`
	SearchedFiles      int               `json:"searched_files"`
	AverageScore       float64           `json:"average_score"`
	HighestScore       float64           `json:"highest_score"`
	LowestScore        float64           `json:"lowest_score"`
	ScoreDistribution  ScoreDistribution `json:"score_distribution"`
	FileTypes          map[string]int    `json:"file_types"`
	TotalLines         int               `json:"total_lines"`
	ResultsByFile      int               `json:"results_by_file"`
	ResultsByLanguage  int               `json:"results_by_language"`
	ResultsByMatchType int               `json:"results_by_match_type"`
	CreatedAt          time.Time         `json:"created_at"`
}
