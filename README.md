# Code Search - Local CLI Codebase Search

A fast, locally-run command-line tool for searching codebases. Built with Go for performance and minimal resource usage.

## 🚀 Features

- **Fast Search**: Quick indexing and searching of codebases
- **Multiple Search Types**: Text, semantic, regex, exact, fuzzy, and hybrid search
- **AI-Powered Semantic Search**: Built-in embedding models for concept-based searching
- **Advanced Caching**: Multi-level caching (L1 memory + L2 disk) for optimal performance
- **Memory Efficient**: Configurable memory limits and intelligent cache management
- **File Type Support**: Supports major programming languages and configuration files
- **Multiple Embedding Models**: Support for custom ONNX models and built-in MiniLM
- **Zero External Dependencies**: Single binary with embedded models
- **Privacy First**: Everything runs locally, your code never leaves your machine

## 📦 Installation

### From Source

```bash
git clone https://github.com/your-org/code-search.git
cd code-search
make build
make install
```

### Development Setup

```bash
make setup    # Install development dependencies (golangci-lint, goimports)
make build    # Build the binary
make test     # Run all tests (unit + contract + integration)
make test-fast # Run fast tests only (skip integration)
make test-coverage # Run tests with coverage report
make install  # Install to GOPATH/bin
make clean    # Clean build artifacts
make fmt      # Format code
make lint     # Run linter
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

# Index a specific directory
code-search index --dir /path/to/project

# Help
code-search index --help
```

**What gets indexed:**
- Supported file types in current directory and subdirectories
- Automatically excludes .git/, node_modules/, build artifacts, and temporary files
- Files up to 1MB in size by default
- Hidden files excluded by default
- Index saved as `.code-search-index.db` in current directory

### Searching

#### Basic Search

```bash
# Simple text search
code-search search "function"

# Search in specific files
code-search search "config" --file-pattern "*.yaml"

# Limit results
code-search search "api" --max-results 10

# Search in specific directory
code-search search "controller" --dir /path/to/project
```

#### Search Types

```bash
# Exact phrase matching
code-search search "UserAuthentication" --exact

# Fuzzy string matching (forgives typos)
code-search search "UsreAuthenication" --fuzzy

# Regular expression search
code-search search "func.*Error" --regex

# Semantic search (find similar concepts using AI embeddings)
code-search search "data validation" --semantic

# Hybrid search (default - combines semantic and text search)
code-search search "payment processing"

# Search with custom embedding model
code-search search "machine learning" --semantic --model custom-model --embedding-path /path/to/model.onnx

# Force search (use test index)
code-search search "test function" --force
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

# Configure embedding cache size
code-search search "authentication" --semantic --cache-size 2000

# Set memory limit for embeddings (in MB)
code-search search "database" --semantic --memory-limit 500

# Search in specific directory
code-search search "react component" --dir /path/to/frontend --semantic

# Force search to use test index
code-search search "debug" --force --format json
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
  -d, --dir <directory>      Specify directory to index (default: current directory)
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
  -d, --dir <directory>   Specify directory to search (default: current directory)
  -M, --model <name>       Embedding model name (default: all-MiniLM-L6-v2)
      --embedding-path     Path to external embedding model file
      --cache-size <n>     Embedding cache size (default: 1000)
      --memory-limit <mb>  Memory limit for embeddings in MB (default: 200)
  -h, --help              Show help message
```

## 🧠 Embedding & Semantic Search

### Overview

Code search includes advanced semantic search capabilities powered by AI embeddings. This allows you to search for code based on concepts and meaning rather than just exact text matches.

### Built-in Embedding Model

The tool includes the **all-MiniLM-L6-v2** model embedded directly in the binary:

- **Model**: Multilingual MiniLM (384-dimensional vectors)
- **Size**: Embedded in binary (model file: all-MiniLM-L6-v2.onnx)
- **Performance**: Optimized for code search scenarios
- **Privacy**: Runs entirely locally, no external API calls

### Semantic Search Examples

```bash
# Find code related to user authentication (even if it doesn't contain those exact words)
code-search search "user login validation" --semantic

# Search for error handling patterns
code-search search "exception management" --semantic

# Find database connection code
code-search search "database connectivity" --semantic

# Search for React component patterns
code-search search "UI component state management" --semantic --file-pattern "*.jsx"
```

### Hybrid Search

By default, the tool uses hybrid search that combines:

- **Semantic Search** (70% weight): Finds conceptually similar code
- **Text Search** (30% weight): Finds exact text matches

This provides the best of both worlds - conceptual understanding with precise matching.

### Custom Embedding Models

You can use custom ONNX models for specialized domains:

```bash
# Use a custom model for a specific programming language
code-search search "algorithm" --semantic --model code-bert --embedding-path ./models/code-bert.onnx

# Configure larger cache for better performance
code-search search "data structure" --semantic --cache-size 5000 --memory-limit 1000
```

### Performance Optimization

#### Multi-Level Caching

The embedding system uses advanced caching for optimal performance:

- **L1 Cache** (Memory): Fast access to frequently used embeddings
- **L2 Cache** (Disk): Persistent storage for larger datasets
- **TTL**: 24-hour expiration for fresh results
- **Eviction**: Intelligent LRU-based cache management

#### Memory Management

```bash
# Limit memory usage for embeddings (default: 200MB)
code-search search "large dataset" --semantic --memory-limit 100

# Increase cache size for better hit rates
code-search search "frequent query" --semantic --cache-size 2000
```

#### Cache Statistics

The system tracks comprehensive cache metrics:
- L1/L2 hit rates
- Memory usage
- Disk usage
- Average access times
- Eviction statistics

### Supported File Types for Semantic Search

Semantic search works best with code files and includes enhanced understanding for:

- **Languages**: Go, Python, JavaScript, TypeScript, Java, C++, Rust, and more
- **Config**: JSON, YAML, TOML, XML configuration files
- **Documentation**: Markdown, text files with code examples
- **Web**: HTML, CSS, JSX, Vue components

### Model Compatibility

The embedding system ensures model compatibility:

- **Metadata Storage**: Model information stored in index files
- **Version Tracking**: Automatic model compatibility validation
- **Migration**: Seamless handling of model updates
- **Fallback**: Graceful degradation to text search if needed

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `--model` | all-MiniLM-L6-v2 | Embedding model name |
| `--embedding-path` | - | Path to external ONNX model |
| `--cache-size` | 1000 | L1 cache entry limit |
| `--memory-limit` | 200MB | Maximum memory for embeddings |
| `--threshold` | 0.7 | Similarity threshold (0.0-1.0) |
| `--dir` | current directory | Target directory for search/index |

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