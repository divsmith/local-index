# Code Search - Local CLI Codebase Search

A fast, locally-run command-line tool for searching codebases. Built with Go for performance and minimal resource usage.

## 🚀 Features

- **Fast Search**: Quick indexing and searching of codebases
- **Multiple Search Types**: Text, semantic, regex, exact, fuzzy, and hybrid search
- **Memory Efficient**: Minimal memory usage for typical repositories
- **File Type Support**: Supports major programming languages and configuration files
- **Zero External Dependencies**: Single binary with only Go standard library
- **Privacy First**: Everything runs locally, your code never leaves your machine

## 📦 Installation

### From Source

```bash
git clone https://github.com/your-org/code-search.git
cd code-search
make build
sudo mv bin/code-search /usr/local/bin/
```

### Development Setup

```bash
make setup    # Install development dependencies
make build    # Build the binary
make test     # Run tests
make install  # Install to GOPATH/bin
```

## 🎯 Quick Start

```bash
# Index your current directory
code-search index

# Search for code patterns
code-search search "user authentication"

# Search with specific options
code-search search "database query" --file-pattern "*.go" --max-results 5

# Semantic search (find similar concepts)
code-search search "error handling" --semantic

# Fuzzy search (find similar strings)
code-search search "UsreAuth" --fuzzy
```

## 📖 Usage

### Indexing

Index your codebase before searching:

```bash
# Basic indexing
code-search index

# Force reindexing (rebuild entire index)
code-search index --force

# Quiet mode (less output)
code-search index --quiet

# Include hidden files
code-search index --include-hidden

# Verbose mode
code-search index --verbose

# Specify file types
code-search index --file-types "*.go,*.js,*.py"

# Exclude patterns
code-search index --exclude "*.min.js,node_modules/*"

# Help
code-search index --help
```

**What gets indexed:**
- Supported file types in current directory and subdirectories
- Automatically excludes .git/, node_modules/, build artifacts, and temporary files
- Files up to 1MB in size by default
- Hidden files excluded by default

### Searching

#### Basic Search

```bash
# Simple text search
code-search search "function"

# Search in specific files
code-search search "config" --file-pattern "*.yaml"

# Limit results
code-search search "api" --max-results 10
```

#### Search Types

```bash
# Exact phrase matching
code-search search "UserAuthentication" --exact

# Fuzzy string matching (forgives typos)
code-search search "UsreAuthenication" --fuzzy

# Regular expression search
code-search search "func.*Error" --regex

# Semantic search (find similar concepts)
code-search search "data validation" --semantic

# Hybrid search (default - combines semantic and text)
code-search search "payment processing"
```

#### Output Formats

```bash
# Table format (default, human-readable)
code-search search "database"

# JSON format (for scripting)
code-search search "database" --format json

# Raw format (file:line:content)
code-search search "database" --format raw
```

#### Advanced Options

```bash
# Include context lines around matches
code-search search "function" --with-context

# Adjust similarity threshold (0.0-1.0)
code-search search "algorithm" --threshold 0.8
```

## 📋 Command Reference

### `code-search index`

Index the current directory for searching.

```bash
code-search index [options]

Options:
  -f, --force                 Force re-indexing even if index exists
  -i, --include-hidden        Include hidden files and directories
  -t, --file-types <types>    Specify file types to include (comma-separated)
  -e, --exclude <patterns>   Exclude patterns (comma-separated)
  -s, --max-file-size <size> Maximum file size in bytes (default: 1MB)
  -v, --verbose              Show detailed progress and statistics
  -q, --quiet                Suppress progress output
  -h, --help                 Show help message
```

### `code-search search`

Search the indexed codebase.

```bash
code-search search <query> [options]

Arguments:
  <query>         The search query text

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
  -h, --help              Show help message
```

## 🤝 Contributing

### Development Setup

```bash
git clone https://github.com/your-org/code-search.git
cd code-search

# Install development dependencies
make setup

# Run tests
make test

# Build locally
make build

# Run development version
./bin/code-search search "test"
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.