# CodeSearch - Agent-First Python Implementation

A minimal, pure-Python codebase search tool specifically designed for AI agents to discover and use easily. Provides intelligent text-based search functionality for code discovery.

## Features

- **🤖 Agent-First Design**: Discoverable through simple help commands, built for AI agents
- **📚 Comprehensive Help**: Full documentation accessible via `help` command
- **⚡ Fast Search**: Intelligent keyword matching with relevance scoring
- **🔍 Smart Indexing**: Automatically detects file types and excludes build artifacts
- **📊 Statistics**: Track indexing progress and database metrics
- **🩺 Diagnostics**: Built-in health checks and troubleshooting
- **🛠️ Pure Python**: No external dependencies required (uses built-in SQLite)
- **📁 File Support**: Supports 20+ programming languages and config formats

## Quick Start for AI Agents

```bash
# 1. Discover tool capabilities
python3 codesearch.py --help     # Basic usage help
python3 codesearch.py help       # Comprehensive agent documentation

# 2. Check tool health
python3 codesearch.py doctor     # Run diagnostics
python3 codesearch.py status     # Check indexing status

# 3. Index a codebase
python3 codesearch.py index .    # Index current directory
python3 codesearch.py index . --verbose --stats  # Detailed indexing

# 4. Search for code
python3 codesearch.py search "class Database"
python3 codesearch.py search "def authenticate_user"
python3 codesearch.py search "import requests" --limit 5
```

## Discovery Pattern for AI Agents

This tool is designed to be incrementally discoverable:

1. **Discovery**: Run `python3 codesearch.py --help` to understand basic commands
2. **Learning**: Use `python3 codesearch.py help` for comprehensive documentation
3. **Validation**: Run `python3 codesearch.py doctor` to verify tool health
4. **Usage**: Index directories and search code
5. **Dogfooding**: Search the tool's own codebase to understand implementation

```bash
# Example: Understanding how the tool works
python3 codesearch.py search "SimpleCodeIndex class"
python3 codesearch.py search "relevance scoring"
python3 codesearch.py search "chunk_content"
```

## How it Works

1. **Indexing**: Splits files into chunks and stores them in SQLite
2. **Search**: Uses keyword matching with relevance scoring
3. **Results**: Shows matching code with line numbers and context

## Commands

```bash
index <directory>    - Index all files in directory for searching
search <query>       - Search indexed code using intelligent keyword matching
status               - Show current index status and detailed statistics
help                 - Show comprehensive help documentation for agents
doctor               - Run diagnostic checks on tool and database
```

## Command Options

### Indexing Options
```bash
--verbose           - Show detailed indexing progress for each file
--stats             - Display index statistics after indexing completes
--db <path>         - Custom database path (default: ~/.codesearch/index.db)
```

### Search Options
```bash
--limit <number>    - Maximum results to return (default: 10)
--db <path>         - Custom database path for multiple indexes
```

## Search Examples

### Basic Usage
```bash
# Find class definitions
python3 codesearch.py search "class UserManager"

# Find function implementations
python3 codesearch.py search "def authenticate"

# Find configuration
python3 codesearch.py search "database config"

# Search specific language constructs
python3 codesearch.py search "import requests"
python3 codesearch.py search "async def"
python3 codesearch.py search "const MAX_SIZE"
```

### Advanced Usage
```bash
# Verbose indexing with statistics
python3 codesearch.py index /path/to/project --verbose --stats

# Search tool's own implementation
python3 codesearch.py search "SimpleCodeIndex class"

# Check indexing status
python3 codesearch.py status

# Run health diagnostics
python3 codesearch.py doctor
```

## Scoring Algorithm

The search uses an intelligent relevance scoring system:
- +1.0 for each keyword match
- +0.5 bonus for whole-word matches
- +1.0 bonus for matches in function/class definitions
- Results sorted by relevance score

## File Support

Automatically detects and indexes common code file types:
- Python (.py, .pyi, .pyx)
- JavaScript (.js, .jsx, .mjs)
- TypeScript (.ts, .tsx)
- Rust (.rs), Go (.go), Java (.java)
- C/C++ (.c, .cpp, .h, .hpp)
- Configuration (.yml, .yaml, .json, .toml)
- Documentation (.md, .rst, .txt)
- And more...

## Performance

- Indexing: ~432 files in < 30 seconds
- Search: Typically < 100ms
- Storage: SQLite database in `~/.codesearch/`

## Usage Examples

```bash
# Find class definitions
python3 codesearch.py search "class UserManager"

# Find function implementations
python3 codesearch.py search "def authenticate"

# Find configuration
python3 codesearch.py search "database config"

# Search in specific contexts
python3 codesearch.py search "import requests"
python3 codesearch.py search "async def"
```

## Architecture Notes

This is a minimal implementation focused on simplicity and immediate usefulness. The code is intentionally straightforward:

- **No external dependencies**: Uses only Python standard library
- **Simple search algorithm**: Basic keyword matching (no embeddings yet)
- **SQLite storage**: Reliable and portable
- **Chunking**: Breaks large files into searchable pieces

## Dogfooding

This tool was created to be immediately useful for AI coding agents. It can search its own codebase:

```bash
# Search the tool's own implementation
python3 codesearch.py search "SimpleCodeIndex class"
python3 codesearch.py search "def search"
python3 codesearch.py search "chunk_content"
```

## Future Improvements

- Add semantic search with embeddings
- Symbol-aware parsing for better code understanding
- Incremental file watching for live updates
- More sophisticated ranking algorithms
- Cross-reference discovery between related code

## License

This code is provided as-is for experimentation and dogfooding purposes.