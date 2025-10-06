package services

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"code-search/src/models"
)

// SearchService handles search operations on indexed codebases
type SearchService struct {
	codeParser  CodeParser
	vectorStore models.VectorStore
	logger      Logger
	searchOptions SearchOptions
}

// SearchOptions contains options for search operations
type SearchOptions struct {
	DefaultMaxResults int           `json:"default_max_results"`
	DefaultThreshold  float64       `json:"default_threshold"`
	Timeout           time.Duration `json:"timeout"`
	EnableFuzzy       bool          `json:"enable_fuzzy"`
	EnableContext     bool          `json:"enable_context"`
	ContextLines      int           `json:"context_lines"`
	CacheResults      bool          `json:"cache_results"`
	CacheSize         int           `json:"cache_size"`
	CacheTTL          time.Duration `json:"cache_ttl"`
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		DefaultMaxResults: 10,
		DefaultThreshold:  0.7,
		Timeout:           30 * time.Second,
		EnableFuzzy:       true,
		EnableContext:     false,
		ContextLines:      3,
		CacheResults:      true,
		CacheSize:         100,
		CacheTTL:          10 * time.Minute,
	}
}

// SearchResultCache caches search results
type SearchResultCache struct {
	cache map[string]*CacheEntry
	ttl   time.Duration
}

// CacheEntry represents a cached search result
type CacheEntry struct {
	results    *models.SearchResults
	expiration time.Time
}

// NewSearchService creates a new SearchService
func NewSearchService(
	codeParser CodeParser,
	vectorStore models.VectorStore,
	logger Logger,
	options SearchOptions,
) *SearchService {
	return &SearchService{
		codeParser:    codeParser,
		vectorStore:   vectorStore,
		logger:        logger,
		searchOptions: options,
	}
}

// Search performs a search operation on the indexed codebase
func (ss *SearchService) Search(
	query *models.SearchQuery,
	indexPath string,
) (*models.SearchResults, error) {
	start := time.Now()

	// Validate query
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid search query: %w", err)
	}

	// Apply default values
	if query.MaxResults == 0 {
		query.MaxResults = ss.searchOptions.DefaultMaxResults
	}
	if query.Threshold == 0 {
		query.Threshold = ss.searchOptions.DefaultThreshold
	}

	ss.logger.Info("Performing search: %s", query.GetSummary())

	// Load index
	index, err := ss.loadIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}
	defer index.Close()

	// Create search results container
	results := models.NewSearchResults(query)
	results.SetSearchedFiles(len(index.GetAllFiles()))

	// Perform search based on query type
	searchResults, err := ss.performSearch(query, index)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Add results to container
	for _, result := range searchResults {
		if err := results.AddResult(result); err != nil {
			ss.logger.Warn("Failed to add search result: %v", err)
		}
	}

	// Sort results by relevance
	results.SortResults()

	// Apply limits
	results.LimitResults(query.MaxResults)

	// Set execution time
	results.SetExecutionTime(time.Since(start))

	ss.logger.Info("Search completed: %d results found in %v", results.TotalResults, results.ExecutionTime)

	return results, nil
}

// performSearch executes the actual search based on query type
func (ss *SearchService) performSearch(
	query *models.SearchQuery,
	index *models.CodeIndex,
) ([]*models.SearchResult, error) {
	switch query.SearchType {
	case models.SearchTypeSemantic:
		return ss.performSemanticSearch(query, index)
	case models.SearchTypeText:
		return ss.performTextSearch(query, index)
	case models.SearchTypeHybrid:
		return ss.performHybridSearch(query, index)
	case models.SearchTypeRegex:
		return ss.performRegexSearch(query, index)
	case models.SearchTypeExact:
		return ss.performExactSearch(query, index)
	case models.SearchTypeFuzzy:
		return ss.performFuzzySearch(query, index)
	default:
		return nil, fmt.Errorf("unsupported search type: %s", query.SearchType)
	}
}

// performSemanticSearch performs vector-based semantic search
func (ss *SearchService) performSemanticSearch(
	query *models.SearchQuery,
	index *models.CodeIndex,
) ([]*models.SearchResult, error) {
	// Generate query embedding
	queryEmbedding, err := ss.codeParser.GetEmbedding(query.GetProcessedQuery())
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Perform vector search
	vectorResults, err := index.Search(queryEmbedding, query.MaxResults*2) // Get more results for filtering
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Convert vector results to search results
	var results []*models.SearchResult
	for _, vectorResult := range vectorResults {
		// Apply threshold
		if vectorResult.Score < query.Threshold {
			continue
		}

		// Extract metadata
		filePath, ok := vectorResult.Metadata["file_path"].(string)
		if !ok {
			continue
		}

		startLine, ok := vectorResult.Metadata["start_line"].(float64)
		if !ok {
			continue
		}

		endLine, ok := vectorResult.Metadata["end_line"].(float64)
		if !ok {
			continue
		}

		content, ok := vectorResult.Metadata["content"].(string)
		if !ok {
			continue
		}

		// Check file and language filters
		language, _ := vectorResult.Metadata["language"].(string)
		if !query.ShouldIncludeFile(filePath, language) {
			continue
		}

		// Create search result
		result := models.FromVectorResult(
			vectorResult,
			filePath,
			int(startLine),
			int(endLine),
			content,
		)

		result.Language = language

		// Add context if requested
		if query.IncludeContext {
			context, err := result.CalculateContext(ss.searchOptions.ContextLines)
			if err == nil {
				result.SetContext(context)
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// performTextSearch performs traditional text-based search
func (ss *SearchService) performTextSearch(
	query *models.SearchQuery,
	index *models.CodeIndex,
) ([]*models.SearchResult, error) {
	var results []*models.SearchResult
	searchTerms := strings.Fields(strings.ToLower(query.GetProcessedQuery()))

	// Get all file entries
	fileEntries := index.GetAllFiles()

	for _, fileEntry := range fileEntries {
		// Check file filter
		if !query.ShouldIncludeFile(fileEntry.FilePath, fileEntry.Language) {
			continue
		}

		// Read file content
		content, err := fileEntry.GetContent()
		if err != nil {
			ss.logger.Warn("Failed to read file content: %s", fileEntry.FilePath)
			continue
		}

		// Search for terms in content
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lineLower := strings.ToLower(line)

			// Check if all search terms are present in this line
			allTermsFound := true
			for _, term := range searchTerms {
				if !strings.Contains(lineLower, term) {
					allTermsFound = false
					break
				}
			}

			if allTermsFound {
				// Create search result
				result := models.NewSearchResult(
					fileEntry.FilePath,
					i+1, i+1, // Line numbers are 1-based
					strings.TrimSpace(line),
				)

				result.Language = fileEntry.Language
				result.MatchType = models.MatchTypeExact
				result.RelevanceScore = ss.calculateTextRelevanceScore(line, searchTerms)

				// Add context if requested
				if query.IncludeContext {
					context := ss.getLineContext(lines, i, ss.searchOptions.ContextLines)
					result.SetContext(context)
				}

				results = append(results, result)
			}
		}
	}

	return results, nil
}

// performHybridSearch performs a combination of semantic and text search
func (ss *SearchService) performHybridSearch(
	query *models.SearchQuery,
	index *models.CodeIndex,
) ([]*models.SearchResult, error) {
	// Perform both semantic and text searches
	semanticResults, err := ss.performSemanticSearch(query, index)
	if err != nil {
		ss.logger.Warn("Semantic search failed, falling back to text search: %v", err)
		return ss.performTextSearch(query, index)
	}

	textResults, err := ss.performTextSearch(query, index)
	if err != nil {
		ss.logger.Warn("Text search failed, using semantic results only: %v", err)
		return semanticResults, nil
	}

	// Merge and deduplicate results
	mergedResults := ss.mergeSearchResults(semanticResults, textResults)

	return mergedResults, nil
}

// performRegexSearch performs regular expression search
func (ss *SearchService) performRegexSearch(
	query *models.SearchQuery,
	index *models.CodeIndex,
) ([]*models.SearchResult, error) {
	// Compile regex pattern
	pattern, err := regexp.Compile(query.QueryText)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var results []*models.SearchResult
	fileEntries := index.GetAllFiles()

	for _, fileEntry := range fileEntries {
		// Check file filter
		if !query.ShouldIncludeFile(fileEntry.FilePath, fileEntry.Language) {
			continue
		}

		// Read file content
		content, err := fileEntry.GetContent()
		if err != nil {
			ss.logger.Warn("Failed to read file content: %s", fileEntry.FilePath)
			continue
		}

		// Search for regex matches
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 0 {
				// Create search result
				result := models.NewSearchResult(
					fileEntry.FilePath,
					i+1, i+1,
					strings.TrimSpace(line),
				)

				result.Language = fileEntry.Language
				result.MatchType = models.MatchTypeRegex
				result.RelevanceScore = ss.calculateRegexRelevanceScore(line, pattern)

				// Add highlights for matched groups
				for _, match := range matches[1:] { // Skip full match
					if match != "" {
						result.AddHighlight(match)
					}
				}

				results = append(results, result)
			}
		}
	}

	return results, nil
}

// performExactSearch performs exact phrase matching
func (ss *SearchService) performExactSearch(
	query *models.SearchQuery,
	index *models.CodeIndex,
) ([]*models.SearchResult, error) {
	searchPhrase := strings.ToLower(query.QueryText)
	var results []*models.SearchResult
	fileEntries := index.GetAllFiles()

	for _, fileEntry := range fileEntries {
		// Check file filter
		if !query.ShouldIncludeFile(fileEntry.FilePath, fileEntry.Language) {
			continue
		}

		// Read file content
		content, err := fileEntry.GetContent()
		if err != nil {
			ss.logger.Warn("Failed to read file content: %s", fileEntry.FilePath)
			continue
		}

		// Search for exact phrase
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), searchPhrase) {
				// Create search result
				result := models.NewSearchResult(
					fileEntry.FilePath,
					i+1, i+1,
					strings.TrimSpace(line),
				)

				result.Language = fileEntry.Language
				result.MatchType = models.MatchTypeExact
				result.RelevanceScore = ss.calculateExactRelevanceScore(line, searchPhrase)
				result.AddHighlight(query.QueryText)

				results = append(results, result)
			}
		}
	}

	return results, nil
}

// performFuzzySearch performs fuzzy string matching
func (ss *SearchService) performFuzzySearch(
	query *models.SearchQuery,
	index *models.CodeIndex,
) ([]*models.SearchResult, error) {
	// For now, implement a simple fuzzy search using text search with relaxed criteria
	// In a production system, you'd use more sophisticated fuzzy matching algorithms

	// Split query into terms and allow for some mismatches
	searchTerms := strings.Fields(strings.ToLower(query.GetProcessedQuery()))

	var results []*models.SearchResult
	fileEntries := index.GetAllFiles()

	for _, fileEntry := range fileEntries {
		// Check file filter
		if !query.ShouldIncludeFile(fileEntry.FilePath, fileEntry.Language) {
			continue
		}

		// Read file content
		content, err := fileEntry.GetContent()
		if err != nil {
			ss.logger.Warn("Failed to read file content: %s", fileEntry.FilePath)
			continue
		}

		// Search with fuzzy matching
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lineLower := strings.ToLower(line)

			// Calculate fuzzy match score
			matchScore := ss.calculateFuzzyMatchScore(lineLower, searchTerms)
			if matchScore > query.Threshold {
				// Create search result
				result := models.NewSearchResult(
					fileEntry.FilePath,
					i+1, i+1,
					strings.TrimSpace(line),
				)

				result.Language = fileEntry.Language
				result.MatchType = models.MatchTypeFuzzy
				result.RelevanceScore = matchScore

				results = append(results, result)
			}
		}
	}

	return results, nil
}

// Helper methods

// loadIndex loads an index from disk
func (ss *SearchService) loadIndex(indexPath string) (*models.CodeIndex, error) {
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("index file does not exist: %s", indexPath)
	}

	return models.LoadCodeIndex(indexPath, ss.vectorStore)
}

// mergeSearchResults merges and deduplicates search results from different sources
func (ss *SearchService) mergeSearchResults(resultSets ...[]*models.SearchResult) []*models.SearchResult {
	seen := make(map[string]bool)
	var merged []*models.SearchResult

	for _, results := range resultSets {
		for _, result := range results {
			// Create a unique key for deduplication
			key := fmt.Sprintf("%s:%d:%d", result.FilePath, result.StartLine, result.EndLine)

			if !seen[key] {
				seen[key] = true
				merged = append(merged, result)
			} else {
				// Update existing result if this one is better
				for i, existing := range merged {
					existingKey := fmt.Sprintf("%s:%d:%d", existing.FilePath, existing.StartLine, existing.EndLine)
					if existingKey == key && result.IsBetterThan(existing) {
						merged[i] = result
						break
					}
				}
			}
		}
	}

	// Sort merged results by relevance
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].IsBetterThan(merged[j])
	})

	return merged
}

// calculateTextRelevanceScore calculates relevance score for text search
func (ss *SearchService) calculateTextRelevanceScore(line string, searchTerms []string) float64 {
	lineLower := strings.ToLower(line)
	score := 0.0

	for _, term := range searchTerms {
		if strings.Contains(lineLower, term) {
			score += 0.5
			// Boost score for exact word matches
			words := strings.Fields(lineLower)
			for _, word := range words {
				if word == term {
					score += 0.3
				}
			}
		}
	}

	// Normalize score
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateRegexRelevanceScore calculates relevance score for regex search
func (ss *SearchService) calculateRegexRelevanceScore(line string, pattern *regexp.Regexp) float64 {
	matches := pattern.FindAllString(line, -1)
	if len(matches) == 0 {
		return 0.0
	}

	// Score based on number and length of matches
	totalMatchLength := 0
	for _, match := range matches {
		totalMatchLength += len(match)
	}

	score := float64(totalMatchLength) / float64(len(line))
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateExactRelevanceScore calculates relevance score for exact phrase search
func (ss *SearchService) calculateExactRelevanceScore(line, phrase string) float64 {
	lineLower := strings.ToLower(line)
	phraseLower := strings.ToLower(phrase)

	if !strings.Contains(lineLower, phraseLower) {
		return 0.0
	}

	// Score based on phrase length relative to line length
	score := float64(len(phrase)) / float64(len(line))

	// Boost score for exact matches
	if strings.Contains(lineLower, phraseLower) {
		score += 0.2
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateFuzzyMatchScore calculates fuzzy match score
func (ss *SearchService) calculateFuzzyMatchScore(line string, searchTerms []string) float64 {
	if len(searchTerms) == 0 {
		return 0.0
	}

	matchedTerms := 0.0
	for _, term := range searchTerms {
		if strings.Contains(line, term) {
			matchedTerms++
		} else {
			// Check for partial matches (simplified Levenshtein distance)
			if ss.isPartialMatch(line, term) {
				matchedTerms += 0.5
			}
		}
	}

	return float64(matchedTerms) / float64(len(searchTerms))
}

// isPartialMatch checks if term is a partial match of line (simplified)
func (ss *SearchService) isPartialMatch(line, term string) bool {
	// Simple implementation: check if any part of the term appears in the line
	if len(term) <= 3 {
		return strings.Contains(line, term)
	}

	// For longer terms, check if at least half the characters match
	halflen := len(term) / 2
	for i := 0; i <= len(term)-halflen; i++ {
		substr := term[i : i+halflen]
		if strings.Contains(line, substr) {
			return true
		}
	}

	return false
}

// getLineContext extracts context lines around a specific line
func (ss *SearchService) getLineContext(lines []string, lineIndex, contextLines int) string {
	start := lineIndex - contextLines
	if start < 0 {
		start = 0
	}

	end := lineIndex + contextLines
	if end >= len(lines) {
		end = len(lines) - 1
	}

	var contextLines_slice []string
	for i := start; i <= end; i++ {
		prefix := "   "
		if i == lineIndex {
			prefix = ">> "
		}
		contextLines_slice = append(contextLines_slice, prefix+lines[i])
	}

	return strings.Join(contextLines_slice, "\n")
}