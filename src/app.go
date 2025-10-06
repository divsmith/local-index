package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// CLI represents the main CLI application
type CLI struct {
	searchCommand *SearchCommand
	indexCommand  *IndexCommand
}

// NewCLI creates a new CLI application
func NewCLI() *CLI {
	return &CLI{
		searchCommand: NewSearchCommand(),
		indexCommand:  NewIndexCommand(),
	}
}

// Run executes the CLI with the given arguments
func (cli *CLI) Run(args []string) error {
	if len(args) == 0 {
		cli.printMainHelp()
		return nil
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "search":
		return cli.searchCommand.Execute(commandArgs)

	case "index":
		return cli.indexCommand.Execute(commandArgs)

	case "help", "--help", "-h":
		cli.printMainHelp()
		return nil

	case "version", "--version", "-v":
		cli.printVersion()
		return nil

	default:
		return fmt.Errorf("unknown command: %s\n\nUse 'code-search help' for available commands", command)
	}
}

// printMainHelp prints the main help message
func (cli *CLI) printMainHelp() {
	fmt.Printf(`Code Search - High Performance Local CLI Vectorized Codebase Search

USAGE:
    code-search <command> [command options]

COMMANDS:
    search      Search the indexed codebase
    index       Index the current directory for searching
    help        Show this help message
    version     Show version information

EXAMPLES:
    # Index your current directory
    code-search index

    # Search for code patterns
    code-search search "user authentication"
    code-search search "calculate tax" --file-pattern "*.go"

    # Search with specific options
    code-search search "database query" --max-results 5 --with-context
    code-search search "function.*error" --semantic --format json

OPTIONS:
    Use 'code-search <command> --help' for command-specific options

FILE TYPES:
    Supports most programming languages and configuration files:
    Go, JavaScript, TypeScript, Python, Java, C/C++, C#, Ruby,
    Swift, Kotlin, Rust, Shell scripts, HTML, CSS, JSON, YAML,
    Markdown, and more.

PERFORMANCE:
    - Indexes repositories up to 1M lines of code
    - Search response under 2 seconds for 100k lines
    - Memory efficient (<500MB for typical usage)
    - Incremental indexing for fast updates

EXIT CODES:
    0    Success
    1    Error
    2    Invalid arguments
    3    Index not found (for search command)

For more information, visit: https://github.com/your-repo/code-search
`)
}

// printVersion prints version information
func (cli *CLI) printVersion() {
	fmt.Printf("code-search version 1.0.0\n")
	fmt.Printf("Built with Go 1.21+\n")
}

// GetDefaultIndexPath returns the default index file path for the current directory
func GetDefaultIndexPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return ".code-search-index"
	}
	return filepath.Join(wd, ".code-search-index")
}
