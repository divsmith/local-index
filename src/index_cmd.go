package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code-search/src/lib"
	"code-search/src/services"
)

// IndexCommand implements the index command
type IndexCommand struct {
	indexingService *services.IndexingService
	logger          *services.DefaultLogger
}

// NewIndexCommand creates a new index command
func NewIndexCommand() *IndexCommand {
	// Create dependencies
	fileScanner := lib.NewFileSystemScanner()
	codeParser := lib.NewSimpleCodeParser()
	vectorStore := lib.NewInMemoryVectorStore(".code-search-index.db")
	logger := &services.DefaultLogger{}
	options := services.DefaultIndexingOptions()

	return &IndexCommand{
		indexingService: services.NewIndexingService(
			fileScanner,
			codeParser,
			vectorStore,
			logger,
			options,
		),
		logger: logger,
	}
}

// Execute executes the index command with the given arguments
func (cmd *IndexCommand) Execute(args []string) error {
	// Parse arguments
	options, err := cmd.parseIndexOptions(args)
	if err != nil {
		return fmt.Errorf("invalid index options: %w", err)
	}

	// Get repository path (current directory by default)
	repositoryPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine index path
	indexPath := cmd.getIndexPath(options.force)

	// Show progress
	progressCallback := func(current, total int, filePath string) {
		if current%10 == 0 || current == total {
			percent := float64(current) / float64(total) * 100
			fmt.Printf("\rIndexing progress: %d/%d files (%.1f%%) - %s",
				current, total, percent, filepath.Base(filePath))
		}
	}

	fmt.Printf("Indexing repository: %s\n", repositoryPath)

	// Perform indexing
	start := time.Now()
	result, err := cmd.indexingService.IndexRepository(
		repositoryPath,
		indexPath,
		options.force,
		progressCallback,
	)
	if err != nil {
		fmt.Printf("\n")
		return fmt.Errorf("indexing failed: %w", err)
	}

	// Show final progress line
	fmt.Printf("\rIndexing progress: %d/%d files (100.0%%) - Complete!\n",
		result.FilesIndexed, result.FilesIndexed+result.FilesSkipped)

	// Display results
	if err := cmd.displayIndexResult(result, start, options); err != nil {
		return fmt.Errorf("failed to display index result: %w", err)
	}

	return nil
}

// IndexOptions contains index command options
type IndexOptions struct {
	force           bool
	includeHidden   bool
	fileTypes       []string
	excludePatterns []string
	maxFileSize     int64
	verbose         bool
	quiet           bool
}

// parseIndexOptions parses command line options for index
func (cmd *IndexCommand) parseIndexOptions(args []string) (IndexOptions, error) {
	options := IndexOptions{
		force:           false,
		includeHidden:   false,
		fileTypes:       []string{"*"},
		excludePatterns: []string{},
		maxFileSize:     1024 * 1024, // 1MB
		verbose:         false,
		quiet:           false,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "--force", "-f":
			options.force = true

		case "--include-hidden", "-i":
			options.includeHidden = true

		case "--file-types", "-t":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--file-types requires a value")
			}
			// Split comma-separated file types
			types := strings.Split(args[i+1], ",")
			options.fileTypes = make([]string, len(types))
			for j, t := range types {
				options.fileTypes[j] = strings.TrimSpace(t)
			}
			i++

		case "--exclude", "-e":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--exclude requires a value")
			}
			// Split comma-separated patterns
			patterns := strings.Split(args[i+1], ",")
			options.excludePatterns = make([]string, len(patterns))
			for j, p := range patterns {
				options.excludePatterns[j] = strings.TrimSpace(p)
			}
			i++

		case "--max-file-size", "-s":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--max-file-size requires a value")
			}
			var size int64
			if _, err := fmt.Sscanf(args[i+1], "%d", &size); err != nil || size <= 0 {
				return options, fmt.Errorf("invalid max-file-size value: %s", args[i+1])
			}
			options.maxFileSize = size
			i++

		case "--verbose", "-v":
			options.verbose = true
			options.quiet = false

		case "--quiet", "-q":
			options.quiet = true
			options.verbose = false

		case "--help", "-h":
			cmd.printIndexHelp()
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
func (cmd *IndexCommand) getIndexPath(force bool) string {
	indexPath := ".code-search-index"

	// If force is true, use a test path
	if force {
		return indexPath + ".test"
	}

	return indexPath
}

// displayIndexResult displays the result of indexing
func (cmd *IndexCommand) displayIndexResult(result *services.IndexingResult, start time.Time, options IndexOptions) error {
	if !options.quiet {
		fmt.Printf("\n") // New line after progress bar
	}

	if result.Success {
		fmt.Printf("Indexing complete. Indexed %d files in %v.\n",
			result.FilesIndexed, result.Duration)

		if result.FilesSkipped > 0 {
			fmt.Printf("Skipped %d files.\n", result.FilesSkipped)
		}

		fmt.Printf("Created %d code chunks.\n", result.ChunksCreated)
		fmt.Printf("Index saved to: %s\n", result.IndexPath)

		// Show index size
		if indexInfo, err := os.Stat(result.IndexPath); err == nil {
			sizeMB := float64(indexInfo.Size()) / (1024 * 1024)
			fmt.Printf("Index size: %.2f MB\n", sizeMB)
		}

		// Show errors if any
		if len(result.Errors) > 0 && options.verbose {
			fmt.Printf("\nErrors encountered:\n")
			for _, errorMsg := range result.Errors {
				fmt.Printf("  - %s\n", errorMsg)
			}
		}
	} else {
		fmt.Printf("Indexing failed.\n")
		if len(result.Errors) > 0 {
			fmt.Printf("Errors:\n")
			for _, errorMsg := range result.Errors {
				fmt.Printf("  - %s\n", errorMsg)
			}
		}
		return fmt.Errorf("indexing failed")
	}

	// Show performance statistics
	if options.verbose {
		filesPerSecond := float64(result.FilesIndexed) / result.Duration.Seconds()
		chunksPerSecond := float64(result.ChunksCreated) / result.Duration.Seconds()

		fmt.Printf("\nPerformance:\n")
		fmt.Printf("  Files processed: %.1f files/sec\n", filesPerSecond)
		fmt.Printf("  Chunks created: %.1f chunks/sec\n", chunksPerSecond)
		fmt.Printf("  Average time per file: %v\n",
			time.Duration(int64(result.Duration.Nanoseconds())/int64(result.FilesIndexed)))
	}

	return nil
}

// printIndexHelp prints help for the index command
func (cmd *IndexCommand) printIndexHelp() {
	fmt.Printf(`Usage: code-search index [options]

Options:
  -f, --force                 Force re-indexing even if index exists
  -i, --include-hidden        Include hidden files and directories
  -t, --file-types <types>     Specify file types to include (comma-separated)
  -e, --exclude <patterns>    Exclude patterns (comma-separated)
  -s, --max-file-size <size>  Maximum file size in bytes (default: 1MB)
  -v, --verbose               Show detailed progress and statistics
  -q, --quiet                 Suppress progress output
  -h, --help                  Show this help message

File Types:
  By default, all supported file types are included. Supported types include:
  - Go: .go
  - JavaScript: .js, .jsx, .mjs
  - TypeScript: .ts, .tsx
  - Python: .py
  - Java: .java
  - C/C++: .c, .cpp, .cc, .cxx, .h, .hpp
  - C#: .cs
  - Ruby: .rb
  - Swift: .swift
  - Kotlin: .kt
  - Rust: .rs
  - Shell: .sh, .bash, .zsh, .fish, .ps1
  - Web: .html, .css, .scss, .sass, .less
  - Config: .json, .yaml, .yml, .toml, .xml
  - Docs: .md, .txt

Exclude Patterns:
  Common patterns are automatically excluded:
  - .git/*
  - node_modules/*
  - *.tmp, *.log
  - .DS_Store, Thumbs.db
  - *.pyc, __pycache__/*
  - Build artifacts and IDE files

Examples:
  code-search index
  code-search index --force
  code-search index --include-hidden --verbose
  code-search index --file-types "*.go,*.js,*.py"
  code-search index --exclude "*.min.js,*.test.go"
  code-search index --max-file-size 2048000

Exit Codes:
  0        Indexing completed successfully
  1        Error during indexing
  2        Invalid arguments

Index File:
  The index is saved as '.code-search-index' in the current directory.
  Use --force to overwrite an existing index.
`)
}

// GetHelp returns help text for the index command
func (cmd *IndexCommand) GetHelp() string {
	return `index [options] - Index the current directory for searching

Use 'code-search index --help' for detailed usage information.`
}
