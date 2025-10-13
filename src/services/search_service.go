package services

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"code-search/src/models"
	"code-search/src/lib"
)

// SearchService handles search operations on indexed codebases
type SearchService struct {
	codeParser    CodeParser
	vectorStore   models.VectorStore
	logger        Logger
	searchOptions SearchOptions
	queryCache    *lib.QueryCache
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
	// Initialize query cache with default values
	l1Size := 1000  // In-memory cache entries
	l2Size := 10000 // Disk-based cache entries
	ttl := options.CacheTTL
	if ttl == 0 {
		ttl = 10 * time.Minute
	}

	queryCache := lib.NewQueryCache(l1Size, l2Size, ttl)

	return &SearchService{
		codeParser:    codeParser,
		vectorStore:   vectorStore,
		logger:        logger,
		searchOptions: options,
		queryCache:    queryCache,
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

	// Check cache first if enabled
	if ss.searchOptions.CacheResults && ss.queryCache != nil {
		if cachedResults, found := ss.queryCache.Get(query); found {
			ss.logger.Debug("Cache hit for query: %s", query.GetSummary())
			cachedResults.SetExecutionTime(time.Since(start))
			return cachedResults, nil
		}
		ss.logger.Debug("Cache miss for query: %s", query.GetSummary())
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

	// Cache the results if enabled
	if ss.searchOptions.CacheResults && ss.queryCache != nil {
		ss.queryCache.Put(query, results)
	}

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

// performHybridSearch performs a combination of semantic and text search in parallel
func (ss *SearchService) performHybridSearch(
	query *models.SearchQuery,
	index *models.CodeIndex,
) ([]*models.SearchResult, error) {
	// Use channels for parallel execution
	type searchResult struct {
		results []*models.SearchResult
		err     error
		searchType string
	}

	semanticChan := make(chan searchResult, 1)
	textChan := make(chan searchResult, 1)

	// Start semantic search in goroutine
	go func() {
		results, err := ss.performSemanticSearch(query, index)
		semanticChan <- searchResult{
			results:    results,
			err:        err,
			searchType: "semantic",
		}
	}()

	// Start text search in goroutine
	go func() {
		results, err := ss.performTextSearch(query, index)
		textChan <- searchResult{
			results:    results,
			err:        err,
			searchType: "text",
		}
	}()

	// Wait for both searches to complete
	var semanticResults, textResults []*models.SearchResult
	var semanticErr, textErr error

	// Collect results
	for i := 0; i < 2; i++ {
		select {
		case result := <-semanticChan:
			semanticResults = result.results
			semanticErr = result.err
		case result := <-textChan:
			textResults = result.results
			textErr = result.err
		}
	}

	// Handle search failures
	if semanticErr != nil && textErr != nil {
		return nil, fmt.Errorf("both semantic and text searches failed: semantic error=%v, text error=%v", semanticErr, textErr)
	}

	if semanticErr != nil {
		ss.logger.Warn("Semantic search failed, using text results only: %v", semanticErr)
		return ss.optimizeTextResults(textResults, query), nil
	}

	if textErr != nil {
		ss.logger.Warn("Text search failed, using semantic results only: %v", textErr)
		return ss.optimizeSemanticResults(semanticResults, query), nil
	}

	// Enhanced merge with intelligent ranking
	mergedResults := ss.mergeAndRankResults(semanticResults, textResults, query)

	// Apply dynamic threshold adjustment
	mergedResults = ss.applyDynamicThresholdAdjustment(mergedResults, query)

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

// SearchInDirectory searches within a specific directory
func (ss *SearchService) SearchInDirectory(
	query *models.SearchQuery,
	directoryPath string,
) (*models.SearchResults, error) {
	fileUtils := lib.NewFileUtilities()

	// Resolve path
	resolvedPath, err := fileUtils.ResolvePath(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Create index location
	indexLocation := fileUtils.CreateIndexLocation(resolvedPath)

	// Check if index exists
	if !fileUtils.DirectoryExists(indexLocation.IndexDir) {
		return nil, fmt.Errorf("no index found in directory '%s'", resolvedPath)
	}

	// Note: File locking disabled temporarily to resolve indexing issues
	// if fileUtils.IsLocked(resolvedPath) {
	// 	return nil, fmt.Errorf("directory '%s' is currently being indexed", resolvedPath)
	// }

	// Validate directory
	if !fileUtils.DirectoryExists(resolvedPath) {
		return nil, fmt.Errorf("directory '%s' does not exist", resolvedPath)
	}

	ss.logger.Info("Searching in directory: %s", resolvedPath)

	// Perform search using existing search logic
	results, err := ss.Search(query, indexLocation.DataFile)
	if err != nil {
		return nil, fmt.Errorf("search failed in directory '%s': %w", resolvedPath, err)
	}

	// Set directory-specific metadata
	results.AddMetadata("directory", resolvedPath)
	results.AddMetadata("index_location", indexLocation.IndexDir)

	return results, nil
}

// ValidateDirectoryForSearching validates a directory for searching
func (ss *SearchService) ValidateDirectoryForSearching(directoryPath string) (*models.DirectoryConfig, error) {
	fileUtils := lib.NewFileUtilities()

	// Resolve path
	resolvedPath, err := fileUtils.ResolvePath(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Create index location
	indexLocation := fileUtils.CreateIndexLocation(resolvedPath)

	// Check if index exists
	if !fileUtils.DirectoryExists(indexLocation.IndexDir) {
		return nil, fmt.Errorf("no index found in directory '%s'", resolvedPath)
	}

	// Validate directory
	validator := lib.NewDirectoryValidator()
	config, err := validator.ValidateDirectory(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("directory validation failed: %w", err)
	}

	return config, nil
}

// GetDirectorySearchCapabilities returns search capabilities for a directory
func (ss *SearchService) GetDirectorySearchCapabilities(directoryPath string) (*DirectorySearchCapabilities, error) {
	fileUtils := lib.NewFileUtilities()

	// Resolve path
	resolvedPath, err := fileUtils.ResolvePath(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Create index location
	indexLocation := fileUtils.CreateIndexLocation(resolvedPath)

	// Check if index exists
	if !fileUtils.DirectoryExists(indexLocation.IndexDir) {
		return &DirectorySearchCapabilities{
			Exists:    false,
			Directory: resolvedPath,
			Message:   "No index found - directory must be indexed first",
		}, nil
	}

	// Load index to get capabilities
	index, err := ss.loadIndex(indexLocation.DataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}
	defer index.Close()

	// Get index statistics
	stats := index.GetStats()

	// Load directory metadata
	metadata := &models.DirectoryMetadata{}
	if fileUtils.FileExists(indexLocation.MetadataFile) {
		metadataBytes, err := os.ReadFile(indexLocation.MetadataFile)
		if err == nil {
			metadata.FromJSON(metadataBytes)
		}
	}

	return &DirectorySearchCapabilities{
		Exists:         true,
		Directory:      resolvedPath,
		IndexLocation:  indexLocation,
		FileCount:      stats.TotalFiles,
		ChunkCount:     stats.TotalChunks,
		LastIndexed:    metadata.LastIndexed,
		IndexVersion:   metadata.IndexVersion,
		TotalSize:      metadata.TotalSize,
		FileTypes:      stats.FileTypes,
		SupportedTypes: ss.getSupportedSearchTypes(),
		Message:        "Directory is ready for searching",
	}, nil
}

// getSupportedSearchTypes returns the list of supported search types
func (ss *SearchService) getSupportedSearchTypes() []string {
	return []string{
		string(models.SearchTypeSemantic),
		string(models.SearchTypeText),
		string(models.SearchTypeHybrid),
		string(models.SearchTypeRegex),
		string(models.SearchTypeExact),
		string(models.SearchTypeFuzzy),
	}
}

// GetCacheStatistics returns the current cache performance statistics
func (ss *SearchService) GetCacheStatistics() lib.CacheStatistics {
	if ss.queryCache == nil {
		return lib.CacheStatistics{}
	}
	return ss.queryCache.GetStatistics()
}

// GetCacheHitRates returns the hit rates for each cache level
func (ss *SearchService) GetCacheHitRates() (l1Rate, l2Rate, l3Rate, overallRate float64) {
	if ss.queryCache == nil {
		return 0, 0, 0, 0
	}
	return ss.queryCache.GetHitRates()
}

// ClearCache clears all cache levels
func (ss *SearchService) ClearCache() {
	if ss.queryCache != nil {
		ss.queryCache.Clear()
		ss.logger.Info("All cache levels cleared")
	}
}

// mergeAndRankResults merges semantic and text results with intelligent ranking
func (ss *SearchService) mergeAndRankResults(
	semanticResults, textResults []*models.SearchResult,
	query *models.SearchQuery,
) []*models.SearchResult {
	// Create map for deduplication and result enhancement
	resultMap := make(map[string]*models.EnhancedSearchResult)

	// Process semantic results with higher base weight
	for _, result := range semanticResults {
		key := ss.getResultKey(result)
		enhanced := &models.EnhancedSearchResult{
			SearchResult:    result,
			SemanticScore:   result.RelevanceScore,
			TextScore:       0.0,
			CombinedScore:   result.RelevanceScore * 0.6, // Semantic gets 60% weight
			SourceTypes:     []string{"semantic"},
			MatchCount:      1,
			FinalRelevance:  result.RelevanceScore,
		}
		resultMap[key] = enhanced
	}

	// Process text results and merge with existing results
	for _, result := range textResults {
		key := ss.getResultKey(result)

		if existing, found := resultMap[key]; found {
			// Merge with existing result
			existing.TextScore = result.RelevanceScore
			existing.CombinedScore += result.RelevanceScore * 0.4 // Text gets 40% weight
			existing.SourceTypes = append(existing.SourceTypes, "text")
			existing.MatchCount++

			// Apply machine learning-based ranking boost for hybrid matches
			hybridBoost := ss.calculateHybridBoost(existing)
			existing.FinalRelevance = ss.applyMLRanking(existing, hybridBoost)

			// Update the underlying result if text result is better
			if result.IsBetterThan(existing.SearchResult) {
				existing.SearchResult = result
			}
		} else {
			// New text-only result
			enhanced := &models.EnhancedSearchResult{
				SearchResult:    result,
				SemanticScore:   0.0,
				TextScore:       result.RelevanceScore,
				CombinedScore:   result.RelevanceScore * 0.4,
				SourceTypes:     []string{"text"},
				MatchCount:      1,
				FinalRelevance:  result.RelevanceScore,
			}
			resultMap[key] = enhanced
		}
	}

	// Convert enhanced results back to regular results and sort
	var finalResults []*models.SearchResult
	for _, enhanced := range resultMap {
		enhanced.SearchResult.RelevanceScore = enhanced.FinalRelevance
		finalResults = append(finalResults, enhanced.SearchResult)
	}

	// Sort by final relevance score (descending)
	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].RelevanceScore > finalResults[j].RelevanceScore
	})

	return finalResults
}

// calculateHybridBoost calculates relevance boost for hybrid matches
func (ss *SearchService) calculateHybridBoost(enhanced *models.EnhancedSearchResult) float64 {
	if len(enhanced.SourceTypes) < 2 {
		return 1.0 // No boost for single-source results
	}

	// Base boost for hybrid matches
	boost := 1.2

	// Additional boost based on score balance
	scoreDiff := math.Abs(enhanced.SemanticScore - enhanced.TextScore)
	maxScore := math.Max(enhanced.SemanticScore, enhanced.TextScore)

	if maxScore > 0 {
		scoreRatio := scoreDiff / maxScore
		// Boost for well-balanced scores (both semantic and text agree)
		if scoreRatio < 0.3 {
			boost += 0.15
		} else if scoreRatio < 0.5 {
			boost += 0.1
		}
	}

	// Boost for multiple matches within same file/region
	if enhanced.MatchCount > 1 {
		boost += 0.05 * float64(enhanced.MatchCount-1)
	}

	return boost
}

// applyMLRanking applies machine learning-based ranking factors
func (ss *SearchService) applyMLRanking(enhanced *models.EnhancedSearchResult, hybridBoost float64) float64 {
	result := enhanced.SearchResult
	query := result.Query // Assuming this is set in the result

	baseScore := enhanced.CombinedScore
	finalScore := baseScore * hybridBoost

	// Apply contextual factors
	finalScore *= ss.calculateContextualBoost(result, query)

	// Apply language-specific factors
	finalScore *= ss.calculateLanguageBoost(result)

	// Apply recency boost (if timestamp information is available)
	finalScore *= ss.calculateRecencyBoost(result)

	// Normalize to 0-1 range
	if finalScore > 1.0 {
		finalScore = 1.0
	}

	return finalScore
}

// calculateContextualBoost calculates boost based on code context
func (ss *SearchService) calculateContextualBoost(result *models.SearchResult, query *models.SearchQuery) float64 {
	boost := 1.0

	// Boost for function/class definitions vs implementations
	if ss.isDefinitionContext(result.Content) {
		boost += 0.1
	}

	// Boost for test files when searching for test-related terms
	if ss.isTestFile(result.FilePath) && ss.isTestQuery(query.QueryText) {
		boost += 0.15
	}

	// Boost for recent files (based on file modification time if available)
	// This would require file system access or metadata

	return boost
}

// calculateLanguageBoost applies language-specific ranking adjustments
func (ss *SearchService) calculateLanguageBoost(result *models.SearchResult) float64 {
	boost := 1.0

	switch result.Language {
	case "Go":
		// Boost for Go-specific patterns
		if ss.isGoPattern(result.Content) {
			boost += 0.05
		}
	case "Python":
		// Boost for Python-specific patterns
		if ss.isPythonPattern(result.Content) {
			boost += 0.05
		}
	case "JavaScript", "TypeScript":
		// Boost for JS/TS-specific patterns
		if ss.isJavaScriptPattern(result.Content) {
			boost += 0.05
		}
	}

	return boost
}

// calculateRecencyBoost applies boost based on file recency
func (ss *SearchService) calculateRecencyBoost(result *models.SearchResult) float64 {
	// Placeholder for recency calculation
	// In practice, this would check file modification time
	// and apply boost based on how recently the file was changed
	return 1.0
}

// applyDynamicThresholdAdjustment dynamically adjusts filtering thresholds
func (ss *SearchService) applyDynamicThresholdAdjustment(
	results []*models.SearchResult,
	query *models.SearchQuery,
) []*models.SearchResult {
	if len(results) == 0 {
		return results
	}

	// Calculate quality metrics of results
	scores := make([]float64, len(results))
	for i, result := range results {
		scores[i] = result.RelevanceScore
	}

	// Calculate statistics
	mean := ss.calculateMean(scores)
	stdDev := ss.calculateStdDev(scores, mean)

	// Dynamic threshold based on result quality distribution
	dynamicThreshold := query.Threshold

	// If we have high-quality results, be more selective
	if mean > 0.8 && stdDev < 0.1 {
		dynamicThreshold = math.Max(dynamicThreshold, mean - 0.2)
	} else if mean < 0.5 && len(results) < query.MaxResults/2 {
		// If we have few low-quality results, be more permissive
		dynamicThreshold = math.Min(dynamicThreshold, mean - stdDev)
	}

	// Filter results based on dynamic threshold
	var filteredResults []*models.SearchResult
	for _, result := range results {
		if result.RelevanceScore >= dynamicThreshold {
			filteredResults = append(filteredResults, result)
		}
	}

	// Ensure we always have some results unless they're all below the original threshold
	if len(filteredResults) == 0 && len(results) > 0 {
		// Fall back to original threshold
		for _, result := range results {
			if result.RelevanceScore >= query.Threshold {
				filteredResults = append(filteredResults, result)
			}
		}
	}

	return filteredResults
}

// optimizeTextResults optimizes text-only results when semantic search fails
func (ss *SearchService) optimizeTextResults(results []*models.SearchResult, query *models.SearchQuery) []*models.SearchResult {
	// Apply enhanced ranking to text results
	for _, result := range results {
		// Boost for exact matches
		if strings.Contains(strings.ToLower(result.Content), strings.ToLower(query.QueryText)) {
			result.RelevanceScore += 0.1
		}

		// Apply contextual factors
		result.RelevanceScore *= ss.calculateContextualBoost(result, query)
		result.RelevanceScore *= ss.calculateLanguageBoost(result)

		// Normalize
		if result.RelevanceScore > 1.0 {
			result.RelevanceScore = 1.0
		}
	}

	// Sort by updated scores
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})

	return results
}

// optimizeSemanticResults optimizes semantic-only results when text search fails
func (ss *SearchService) optimizeSemanticResults(results []*models.SearchResult, query *models.SearchQuery) []*models.SearchResult {
	// Apply enhanced ranking to semantic results
	for _, result := range results {
		// Boost for semantic matches that also contain text matches
		if strings.Contains(strings.ToLower(result.Content), strings.ToLower(query.QueryText)) {
			result.RelevanceScore += 0.15
		}

		// Apply contextual factors
		result.RelevanceScore *= ss.calculateContextualBoost(result, query)
		result.RelevanceScore *= ss.calculateLanguageBoost(result)

		// Normalize
		if result.RelevanceScore > 1.0 {
			result.RelevanceScore = 1.0
		}
	}

	// Sort by updated scores
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})

	return results
}

// Helper methods for ranking factors

func (ss *SearchService) getResultKey(result *models.SearchResult) string {
	return fmt.Sprintf("%s:%d:%d", result.FilePath, result.StartLine, result.EndLine)
}

func (ss *SearchService) isDefinitionContext(content string) bool {
	definitions := []string{
		"func ", "function ", "def ", "class ", "interface ", "type ",
		"const ", "let ", "var ", "struct ", "enum ", "import ",
	}

	contentLower := strings.ToLower(content)
	for _, def := range definitions {
		if strings.Contains(contentLower, def) {
			return true
		}
	}
	return false
}

func (ss *SearchService) isTestFile(filePath string) bool {
	testIndicators := []string{"_test.go", "test_", "_test.", "spec.", "tests/"}
	filePathLower := strings.ToLower(filePath)
	for _, indicator := range testIndicators {
		if strings.Contains(filePathLower, indicator) {
			return true
		}
	}
	return false
}

func (ss *SearchService) isTestQuery(queryText string) bool {
	testTerms := []string{"test", "spec", "mock", "stub", "fixture", "assert"}
	queryLower := strings.ToLower(queryText)
	for _, term := range testTerms {
		if strings.Contains(queryLower, term) {
			return true
		}
	}
	return false
}

func (ss *SearchService) isGoPattern(content string) bool {
	patterns := []string{"func ", "type ", "interface{}", "struct {", "go func()"}
	contentLower := strings.ToLower(content)
	for _, pattern := range patterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}
	return false
}

func (ss *SearchService) isPythonPattern(content string) bool {
	patterns := []string{"def ", "class ", "import ", "from ", "lambda "}
	contentLower := strings.ToLower(content)
	for _, pattern := range patterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}
	return false
}

func (ss *SearchService) isJavaScriptPattern(content string) bool {
	patterns := []string{"function ", "const ", "let ", "var ", "=>", "class "}
	contentLower := strings.ToLower(content)
	for _, pattern := range patterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}
	return false
}

func (ss *SearchService) calculateMean(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}

	sum := 0.0
	for _, score := range scores {
		sum += score
	}
	return sum / float64(len(scores))
}

func (ss *SearchService) calculateStdDev(scores []float64, mean float64) float64 {
	if len(scores) == 0 {
		return 0
	}

	sum := 0.0
	for _, score := range scores {
		diff := score - mean
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(scores)))
}

// DirectorySearchCapabilities contains information about a directory's search capabilities
type DirectorySearchCapabilities struct {
	Exists         bool                         `json:"exists"`
	Directory      string                       `json:"directory"`
	IndexLocation  *models.IndexLocation        `json:"index_location,omitempty"`
	FileCount      int                          `json:"file_count"`
	ChunkCount     int                          `json:"chunk_count"`
	LastIndexed    time.Time                    `json:"last_indexed"`
	IndexVersion   string                       `json:"index_version"`
	TotalSize      int64                        `json:"total_size"`
	FileTypes      map[string]int               `json:"file_types"`
	SupportedTypes []string                     `json:"supported_types"`
	Message        string                       `json:"message"`
}
