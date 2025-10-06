package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"code-search/src/lib"
	"code-search/src/models"
	"code-search/src/services"
)

// SearchCommand implements the search command
type SearchCommand struct {
	searchService *services.SearchService
	logger        *services.DefaultLogger
}

// NewSearchCommand creates a new search command
func NewSearchCommand() *SearchCommand {
	return &SearchCommand{
		searchService: services.NewSearchService(
			lib.NewSimpleCodeParser(),
			lib.NewInMemoryVectorStore(""),
			&services.DefaultLogger{},
			services.DefaultSearchOptions(),
		),
		logger: &services.DefaultLogger{},
	}
}

// Execute executes the search command with the given arguments
func (cmd *SearchCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("search query is required")
	}

	// Parse arguments
	queryText := args[0]
	options, err := cmd.parseSearchOptions(args[1:])
	if err != nil {
		return fmt.Errorf("invalid search options: %w", err)
	}

	// Create search query
	query := models.NewSearchQuery(queryText)
	query.MaxResults = options.maxResults
	query.IncludeContext = options.withContext
	query.FileFilter = options.filePattern
	query.Threshold = options.threshold

	// Set search type based on options
	if options.semantic {
		query.SearchType = models.SearchTypeSemantic
	} else if options.exact {
		query.SearchType = models.SearchTypeExact
	} else if options.fuzzy {
		query.SearchType = models.SearchTypeFuzzy
	}

	// Determine index path
	indexPath := cmd.getIndexPath(options.force)

	// Perform search
	start := time.Now()
	results, err := cmd.searchService.Search(query, indexPath)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Display results based on output format
	switch options.format {
	case "json":
		return cmd.displayJSONResults(results)
	case "raw":
		return cmd.displayRawResults(results)
	default:
		return cmd.displayTableResults(results, start)
	}
}

// SearchOptions contains search command options
type SearchOptions struct {
	maxResults  int
	filePattern string
	withContext bool
	force       bool
	format      string
	threshold   float64
	semantic    bool
	exact       bool
	fuzzy       bool
}

// parseSearchOptions parses command line options for search
func (cmd *SearchCommand) parseSearchOptions(args []string) (SearchOptions, error) {
	options := SearchOptions{
		maxResults:  10,
		filePattern: "",
		withContext: false,
		force:       false,
		format:      "table",
		threshold:   0.7,
		semantic:    false,
		exact:       false,
		fuzzy:       false,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "--max-results", "-m":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--max-results requires a value")
			}
			var maxResults int
			if _, err := fmt.Sscanf(args[i+1], "%d", &maxResults); err != nil || maxResults < 1 {
				return options, fmt.Errorf("invalid max-results value: %s", args[i+1])
			}
			options.maxResults = maxResults
			i++

		case "--file-pattern", "-f":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--file-pattern requires a value")
			}
			options.filePattern = args[i+1]
			i++

		case "--with-context", "-c":
			options.withContext = true

		case "--force", "-F":
			options.force = true

		case "--format":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--format requires a value")
			}
			format := strings.ToLower(args[i+1])
			if format != "table" && format != "json" && format != "raw" {
				return options, fmt.Errorf("invalid format: %s (supported: table, json, raw)", format)
			}
			options.format = format
			i++

		case "--threshold", "-t":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--threshold requires a value")
			}
			var threshold float64
			if _, err := fmt.Sscanf(args[i+1], "%f", &threshold); err != nil || threshold < 0 || threshold > 1 {
				return options, fmt.Errorf("invalid threshold value: %s (must be between 0 and 1)", args[i+1])
			}
			options.threshold = threshold
			i++

		case "--semantic", "-s":
			options.semantic = true

		case "--exact", "-e":
			options.exact = true

		case "--fuzzy", "-z":
			options.fuzzy = true

		case "--help", "-h":
			cmd.printSearchHelp()
			os.Exit(0)

		default:
			if strings.HasPrefix(arg, "-") {
				return options, fmt.Errorf("unknown option: %s", arg)
			}
		}
	}

	return options, nil
}

// getIndexPath returns the path to the index file
func (cmd *SearchCommand) getIndexPath(force bool) string {
	// Default index path
	indexPath := ".code-search-index"

	// If force is true, use a temporary path for testing
	if force {
		return indexPath + ".test"
	}

	return indexPath
}

// displayTableResults displays search results in table format
func (cmd *SearchCommand) displayTableResults(results *models.SearchResults, start time.Time) error {
	if results.IsEmpty() {
		fmt.Println("No results found.")
		return nil
	}

	fmt.Printf("Found %d results:\n\n", results.TotalResults)

	for _, result := range results.Results {
		fmt.Printf("%d. %s:%d-%d\n", result.Rank, result.FilePath, result.StartLine, result.EndLine)

		// Display content with syntax highlighting (simplified)
		content := result.Content
		if len(content) > 100 {
			content = content[:97] + "..."
		}
		fmt.Printf("   %s\n", content)

		// Display highlights if any
		if len(result.Highlights) > 0 {
			highlights := strings.Join(result.Highlights, "; ")
			if len(highlights) > 80 {
				highlights = highlights[:77] + "..."
			}
			fmt.Printf("   Highlights: %s\n", highlights)
		}

		// Display context if available
		if result.Context != "" && len(result.Context) < 200 {
			contextLines := strings.Split(result.Context, "\n")
			for _, line := range contextLines {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("   %s\n", line)
				}
			}
		}

		fmt.Printf("   Score: %.3f | Type: %s\n", result.RelevanceScore, result.MatchType)
		fmt.Println()
	}

	// Display summary
	fmt.Printf("Search completed in %v\n", results.ExecutionTime)
	if len(results.Results) != results.TotalResults {
		fmt.Printf("Showing %d of %d results\n", len(results.Results), results.TotalResults)
	}

	return nil
}

// displayJSONResults displays search results in JSON format
func (cmd *SearchCommand) displayJSONResults(results *models.SearchResults) error {
	// Create a clean JSON structure for output
	output := map[string]interface{}{
		"query":          results.Query.QueryText,
		"total_results":  results.TotalResults,
		"displayed":      len(results.Results),
		"execution_time": results.ExecutionTime.String(),
		"has_more":       results.HasMore,
		"results":        results.Results,
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate JSON output: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// displayRawResults displays search results in raw format
func (cmd *SearchCommand) displayRawResults(results *models.SearchResults) error {
	for _, result := range results.Results {
		fmt.Printf("%s:%d:%d:%s\n", result.FilePath, result.StartLine, result.EndLine, result.Content)
	}
	return nil
}

// printSearchHelp prints help for the search command
func (cmd *SearchCommand) printSearchHelp() {
	fmt.Printf(`Usage: code-search search <query> [options]

Arguments:
  <query>                  The search query text

Options:
  -m, --max-results <n>    Maximum number of results to return (default: 10)
  -f, --file-pattern <p>   Filter results by file pattern (e.g., "*.go")
  -c, --with-context       Include code context in results
  -F, --force              Force search (use test index)
      --format <fmt>       Output format: table, json, raw (default: table)
  -t, --threshold <t>      Similarity threshold (0.0-1.0, default: 0.7)
  -s, --semantic          Use semantic search
  -e, --exact             Use exact matching
  -z, --fuzzy             Use fuzzy matching
  -h, --help              Show this help message

Examples:
  code-search search "user authentication"
  code-search search "calculate tax" --file-pattern "*.go" --max-results 5
  code-search search "database query" --with-context --format json
  code-search search "function.*error" --semantic --threshold 0.8

Output Formats:
  table    Human-readable table format (default)
  json     Machine-readable JSON format
  raw      Simple file:line:line:content format

Search Types:
  semantic Vector-based semantic search (default for combined search)
  exact    Exact phrase matching
  fuzzy    Fuzzy string matching

Exit Codes:
  0        Search completed successfully
  1        Error during search
  2        Invalid arguments
  3        Index not found (run 'code-search index' first)
`)
}

// GetHelp returns help text for the search command
func (cmd *SearchCommand) GetHelp() string {
	return `search <query> [options] - Search the indexed codebase

Use 'code-search search --help' for detailed usage information.`
}
