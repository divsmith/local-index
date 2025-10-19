# Agent-First Codebase Search Tool Specification

## Overview

A locally-installed CLI tool that provides semantic and symbol-aware code search capabilities for agentic coding tools. Solves the token inefficiency and noise problems of grep-only retrieval while avoiding the complexity of MCP-based solutions.

## Problem Statement

**Current Issues:**
- Claude Code's grep-only retrieval "drowns you in irrelevant matches, burns tokens, and stalls your workflow"
- Existing semantic solutions (Claude Context MCP) require Docker, complex setup, and maintenance overhead
- No simple, self-contained way for coding agents to perform semantic code search on local codebases

**Target Users:**
- Primary: AI coding agents (Claude Code, GitHub Copilot, etc.)
- Secondary: Human developers working with AI assistants

## Core Requirements

### Functional Requirements
1. **Semantic Search**: Vector embedding-based search that understands code meaning and intent
2. **Symbol-Aware Search**: Parse and index code structure (functions, classes, variables, imports)
3. **Multi-Language Support**: Handle common programming languages and file types from MVP
4. **Incremental Indexing**: Fast, real-time updates as code changes
5. **Agent-First CLI Interface**: Self-documenting commands that agents can discover via `--help`

### Non-Functional Requirements
1. **System Installation**: Single binary installed globally, not per-project
2. **Local Storage**: Index data stored in user home directory (~/.codesearch/)
3. **Zero Repository Pollution**: No binaries or index files committed to git
4. **Fast Performance**: Indexing and search should feel instantaneous for agents
5. **Self-Documenting**: Agents can discover capabilities through help commands

## Technical Architecture

### High-Level Design
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI Interface │    │   Index Engine   │    │  Search Engine  │
│   (git-style)   │────│  (Incremental)   │────│ (Semantic +     │
│                 │    │                  │    │  Symbol-aware)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                       ┌──────────────────┐    ┌─────────────────┐
                       │  Local Storage   │    │  Multi-Language │
                       │  (~/.codesearch/)│    │   Parsers       │
                       └──────────────────┘    └─────────────────┘
```

### Core Components

#### 1. CLI Interface (Git-style)
- Primary entry point: `codesearch`
- Subcommands: `index`, `search`, `status`, `config`
- Self-documenting: `codesearch --help` and `codesearch <subcommand> --help`
- Agent-friendly output: Structured text with optional JSON

#### 2. Index Engine
- **Incremental Updates**: Only re-index changed files
- **Multi-Language Parsing**: Support for common languages from MVP
- **Semantic Embeddings**: Generate vector representations for semantic search
- **Symbol Extraction**: Parse ASTs for functions, classes, variables
- **Storage**: Local file-based storage in user home directory

#### 3. Search Engine
- **Semantic Search**: Vector similarity matching with relevance scoring
- **Symbol Search**: Exact and fuzzy matching of code identifiers
- **Hybrid Results**: Combine semantic and symbol relevance
- **Context-Rich Output**: Include surrounding code, file context, relationships

#### 4. Multi-Language Support
**MVP Languages:**
- JavaScript/TypeScript
- Python
- YAML/JSON (configs)
- Markdown (docs)
- Shell scripts

**Extension Points:**
- Plugin architecture for additional languages
- Configurable file type associations
- Custom parsing rules

## CLI Interface Specification

### Primary Commands

```bash
# Initialize and index current directory
codesearch index [path] [options]

# Search with semantic understanding
codesearch search <query> [options]

# Search for specific symbols/functions
codesearch find <symbol> [options]

# Show indexing status and statistics
codesearch status [path]

# Show configuration
codesearch config [show|set <key> <value>]
```

### Search Examples

```bash
# Semantic search for authentication flow
codesearch search "how user authentication works"

# Find specific function or class
codesearch find "authenticate_user"

# Search with file type filter
codesearch search "database connection" --type python,js

# Output as JSON for agent consumption
codesearch search "error handling" --json

# Limit results and show context
codesearch search "API endpoints" --limit 5 --context 3
```

### Output Format

**Default (structured text):**
```
src/auth.py:45-62 (Score: 0.92)
├─ authenticate_user(username, password)
│  # Main authentication function that validates credentials
│  # and returns user session tokens

tests/test_auth.py:123-140 (Score: 0.85)
├─ test_authentication_flow()
│  # Test cases for the authentication process
```

**JSON (for agent consumption):**
```json
{
  "results": [
    {
      "file": "src/auth.py",
      "start_line": 45,
      "end_line": 62,
      "score": 0.92,
      "type": "function",
      "name": "authenticate_user",
      "context": "Main authentication function...",
      "code_snippet": "def authenticate_user(username, password):..."
    }
  ],
  "query": "user authentication",
  "total_results": 2
}
```

## Data Model

### Index Management

#### File Watching and Invalidation Strategy

**Real-time File Watching:**
- **Cross-platform**: Use `notify` crate for file system events
- **Debouncing**: 500ms debounce to batch rapid file changes
- **Event Types**: Handle create, modify, delete, move operations
- **Exclusions**: Ignore temporary files, build artifacts, version control files

**Incremental Update Process:**
```rust
// Pseudo-code for incremental update
on_file_change(event) {
    match event.type {
        Create => index_file(event.path),
        Modify => update_file_index(event.path),
        Delete => remove_file_from_index(event.path),
        Move => update_file_references(event.old_path, event.new_path)
    }
}
```

**Index Invalidation Rules:**
- **File content change**: Remove old vectors, regenerate embeddings
- **File deletion**: Remove from all indices, clean up orphaned references
- **File rename**: Update path mappings, preserve existing embeddings if content unchanged
- **Directory changes**: Recursively update all affected files
- **Configuration changes**: Re-index affected file types or full reindex if needed

#### Concurrent Access Patterns

**Multi-Process Safety:**
- **File Locking**: SQLite handles concurrent read/write automatically
- **Vector Storage**: Use memory-mapped files with atomic updates
- **Index Updates**: Queue updates during active searches, apply after completion
- **Write-Ahead Logging**: SQLite WAL mode for concurrent readers/writers

**Agent Access Patterns:**
```bash
# Multiple agents working simultaneously
agent1$ codesearch search "authentication"      # Read access
agent2$ codesearch find "DatabaseConnection"    # Read access
agent3$ codesearch index . --incremental         # Write access
```

**Lock Hierarchy:**
1. **SQLite**: Built-in row-level locking for metadata
2. **Vector Files**: Atomic file replacement during updates
3. **Configuration**: File-level locking for config updates
4. **Index Status**: Shared memory flags for coordination

#### Index Corruption Recovery

**Detection Mechanisms:**
- **SQLite Integrity**: Run `PRAGMA integrity_check` on index open
- **Vector File Headers**: Magic numbers and checksums for binary files
- **Metadata Consistency**: Cross-reference SQLite metadata with file system
- **Size Validation**: Verify index sizes match expected ranges

**Recovery Strategies:**

**Level 1: Index Repair**
```bash
codesearch repair [path]
# Rebuild corrupted components while preserving valid data
# Attempt to recover from transaction logs
# Re-index only corrupted files
```

**Level 2: Partial Rebuild**
```bash
codesearch rebuild --from-scratch [path]
# Clear corrupted index, rebuild from source files
# Preserves configuration and model cache
```

**Level 3: Full Reset**
```bash
codesearch reset --all [path]
# Remove all index data, start fresh
# Last resort for catastrophic corruption
```

**Backup and Rollback:**
- **Automatic Backups**: Create index snapshots before major updates
- **Rollback Capability**: Restore from previous index version if corruption detected
- **Graceful Degradation**: Continue serving read queries during index recovery

#### Index Storage Structure

```
~/.codesearch/
├── indices/
│   └── <project-hash>/
│       ├── metadata.db          # SQLite database (metadata, symbols, files)
│       ├── vectors.bin          # Binary vector embeddings
│       ├── vectors.bin.lock     # Lock file for vector updates
│       ├── index.json           # Index configuration and status
│       └── backups/             # Automatic index snapshots
│           ├── 2024-01-15_10-30/
│           └── 2024-01-16_14-22/
├── models/
│   ├── codebert-base.onnx       # Primary embedding model
│   ├── graphcodebert.onnx       # Alternative model
│   └── model_checksums.json     # Verify model integrity
├── config/
│   ├── default.toml             # Default configuration
│   └── user_overrides.toml      # User customizations
└── logs/
    ├── indexing.log             # Indexing operations and errors
    └── search.log               # Search performance metrics
```

#### Index Lifecycle Management

**Index Aging and Cleanup:**
- **Project Access Tracking**: Track last access time per project index
- **Automatic Cleanup**: Remove indices for projects not accessed in 90 days
- **Size Management**: Warn when total index storage exceeds configured limits
- **User Control**: Manual control over cleanup policies

**Migration Strategy:**
- **Version Compatibility**: Index versioning with automatic migration
- **Backward Compatibility**: Support reading older index formats
- **Migration Tools**: Built-in tools to migrate between major versions
- **Fallback Options**: Graceful degradation if migration fails

### Search Result Schema
```json
{
  "file": "string",
  "start_line": "number",
  "end_line": "number",
  "score": "number",
  "type": "function|class|variable|file",
  "name": "string",
  "context": "string",
  "code_snippet": "string",
  "relationships": ["related_symbols"],
  "language": "string"
}
```

## Agent Integration Patterns

### Discovery Pattern
```bash
# Agent discovers tool capabilities
codesearch --help
codesearch search --help
```

### Usage Pattern
```bash
# Agent working in new codebase
cd /path/to/project
codesearch index .                    # Quick index setup
codesearch search "how to handle X"   # Find relevant code
codesearch find "function_name"       # Locate specific implementation
```

### Dogfooding Pattern
```bash
# Tool can search its own codebase
cd /path/to/codesearch
codesearch search "indexing implementation"
codesearch find "search_engine"
```

## Implementation Phases

### Phase 1: MVP (Dogfooding Ready)
- Basic semantic search for 2-3 core languages
- Simple symbol extraction
- Local indexing with basic incremental updates
- Git-style CLI with core commands
- Search within tool's own codebase

### Phase 2: Multi-Language
- Extended language support (JS, Python, YAML, MD, Shell)
- Improved parsing accuracy
- Better relevance scoring
- Configuration system

### Phase 3: Advanced Features
- Dependency graph traversal
- Cross-file relationship mapping
- Advanced filtering and query syntax
- Performance optimizations

## Success Metrics

### Agent Experience
- **Discovery**: Agent can understand tool capabilities within 2-3 help commands
- **Performance**: Search results returned < 100ms for indexed codebases
- **Relevance**: Top 3 results contain relevant information > 80% of the time
- **Token Efficiency**: Reduce context needed vs grep by 40%+

### Developer Experience
- **Setup**: New user can install and index first project within 5 minutes
- **Maintenance**: Incremental re-indexing completes < 10 seconds for typical changes
- **Storage**: Index size < 10% of source code size
- **Compatibility**: Works across common operating systems and shells

## Error Handling and Recovery

### File System Errors

**Missing Files:**
```bash
# File indexed but deleted from disk
codesearch search "function_name"
# → Warning: File '/path/to/file.py' referenced in index but not found on disk
# → Continue with other results, offer to reindex
```

**Permission Errors:**
```bash
# File exists but not readable
codesearch index /path/to/project
# → Warning: Skipping '/path/to/secret.key' - permission denied
# → Log to indexing.log, continue with other files
# → Exit code 1 with warning if critical files skipped
```

**File Too Large:**
```bash
# File exceeds size limit (default 10MB)
codesearch index .
# → Info: Skipping 'large_model.bin' (15.2MB) - exceeds size limit
# → Option: codesearch index . --max-file-size 50MB
```

### Parsing Failures

**Syntax Errors:**
```bash
# Invalid Python syntax
codesearch index broken_file.py
# → Warning: 'broken_file.py:42' - SyntaxError: invalid syntax
# → Continue indexing other functions in same file
# → Add parsing_error flag to metadata for affected symbols
```

**Unsupported File Types:**
```bash
# Unknown file extension
codesearch search --type unknown_format
# → Error: File type 'unknown_format' not supported
# → Info: Supported types: python, javascript, typescript, yaml, markdown
# → Suggestion: Use codesearch config add_type unknown_format <parser>
```

**Tree-sitter Parse Failures:**
```bash
# Tree-sitter cannot parse file
codesearch index complex_file.js
# → Warning: 'complex_file.js' - Parse error, treating as plain text
# → Fallback: Basic text-based indexing, no symbol extraction
# → Performance: Slower search on fallback-indexed content
```

### Index Corruption

**SQLite Database Corruption:**
```bash
# Database integrity check fails
codesearch search "query"
# → Error: Index corruption detected in metadata.db
# → Auto-repair: Attempting SQLite recovery...
# → Fallback: codesearch repair --force if auto-repair fails
```

**Vector File Corruption:**
```bash
# Binary vector file corrupted
codesearch search "semantic query"
# → Error: Vector index corrupted, falling back to symbol-only search
# → Info: Rebuilding vector index...
# → Background: Regenerate embeddings for affected files
```

**Schema Mismatch:**
```bash
# Index version incompatible
codesearch index .
# → Error: Index version 2.1 incompatible with tool version 3.0
# → Solution: codesearch migrate --from-version 2.1
# → Fallback: codesearch index . --force (full rebuild)
```

### Model Loading Errors

**Missing ONNX Models:**
```bash
# Model file not found
codesearch search "query"
# → Error: Embedding model 'codebert-base.onnx' not found
# → Auto-fix: Downloading model (4.2MB)...
# → Progress: ████████████████████████████████ 100%
# → Retry: Search with loaded model
```

**Model Load Failure:**
```bash
# ONNX runtime error
codesearch search "query"
# → Error: Failed to load model - incompatible ONNX version
# → Fallback: Using text-only search (no semantic matching)
# → Fix: codesearch models download --force
```

**Memory Insufficient:**
```bash
# Not enough RAM for model
codesearch search "query"
# → Error: Insufficient memory to load embedding model (requires 1.2GB, available 800MB)
# → Option: Use smaller model: codesearch config set model codebert-small
# → Option: Close other applications and retry
```

### Network and Model Distribution Errors

**Download Failures:**
```bash
# Model download interrupted
codesearch models download
# → Error: Download failed - network timeout
# → Resume: codesearch models download --resume
# → Fallback: Manual download from https://github.com/.../models/
```

**Checksum Mismatch:**
```bash
# Model file corrupted during download
codesearch models download
# → Error: Model checksum mismatch - file corrupted
# → Auto-fix: Redownloading model...
# → Verification: SHA256 checksum validation
```

### Concurrent Access Conflicts

**Index Lock Conflicts:**
```bash
# Multiple indexing operations
codesearch index . &
codesearch index . &
# → Error: Index operation in progress (PID 12345)
# → Option: codesearch index . --force-override
# → Option: Wait and retry: codesearch index . --wait
```

**Search During Indexing:**
```bash
# Search while index updating
codesearch search "query"
# → Info: Index update in progress, using cached results
# → Performance: May return slightly stale results
# → Refresh: Results automatically update when indexing completes
```

### Error Recovery Commands

**Diagnostics:**
```bash
codesearch doctor
# Comprehensive system health check:
# ✓ Model files present and valid
# ✓ Index integrity verified
# ✓ Permissions sufficient
# ✗ SQLite database needs optimization
# → Recommendation: codesearch optimize
```

**Repair Operations:**
```bash
codesearch repair [path] [--level=1|2|3] [--dry-run]
# Level 1: Repair index metadata and corrupted entries
# Level 2: Rebuild corrupted components from source
# Level 3: Full index reconstruction (last resort)
```

**Optimization:**
```bash
codesearch optimize [path]
# SQLite VACUUM and index optimization
# Vector file compaction
# Cache cleanup
# Performance improvements
```

### Error Codes and Exit Status

**Exit Codes:**
- `0`: Success
- `1`: Warning (non-fatal issues, operation completed)
- `2`: Error (fatal issue, operation failed)
- `3`: Configuration error (invalid settings)
- `4`: Permission error (insufficient permissions)
- `5`: Network error (model download/update failure)

**Error Message Format:**
```
[ERROR] codesearch: index: Failed to parse file 'src/main.py'
Caused by: SyntaxError: invalid syntax (line 42, column 15)
  → src/main.py:42:15: return value with extra parenthesis
Help: Fix syntax error or exclude file with --exclude-pattern
Documentation: https://github.com/codesearch/docs/troubleshooting
```

## Technical Considerations

### Performance
- **Indexing Strategy**: Incremental with file change detection
- **Storage**: Efficient binary formats for vectors, JSON for metadata
- **Caching**: Embedding caching to avoid recomputation
- **Memory**: Lazy loading of indices to handle large codebases

### Extensibility
- **Plugin Architecture**: Language parsers as plugins
- **Configuration**: User-customizable file associations and search preferences
- **API Design**: Clean separation between CLI and core search engine


### Security
- **Local Only**: No network dependencies or external API calls
- **Privacy**: All data stays on user's machine
- **Sandboxing**: Tool operates with standard file permissions

## Technical Recommendations (Resolved)

### 1. Vector Embedding Strategy: **Local Generation (ONNX)**
**Decision**: Use pre-trained ONNX models optimized for code (CodeBERT, GraphCodeBERT)

**Rationale:**
- **Privacy is non-negotiable** - developers won't send code to third parties
- **Agent experience** - zero network latency is crucial for interactive coding
- **Dogfooding requirement** - tool must work on its own codebase without external dependencies
- **Model quality is good enough** - Code-specific models work well for code similarity

**Tradeoffs Accepted**: 100MB-1GB binary size increase, CPU-based generation (slower than GPU APIs)

### Model Distribution and Management

#### Distribution Strategy

**Bundled Models:**
- **Primary Model**: `codebert-base` (420MB) - included in binary distribution
- **Alternative Model**: `graphcodebert-base` (445MB) - optional download
- **Lightweight Model**: `codebert-small` (145MB) - for memory-constrained environments

**Binary Bundling:**
```rust
// Models embedded in binary using include_bytes! macro
static MODEL_CODEBERT_BASE: &[u8] = include_bytes!("models/codebert-base.onnx");
static MODEL_CODEBERT_SMALL: &[u8] = include_bytes!("models/codebert-small.onnx");
```

**Distribution Methods:**

**Method 1: Single Binary (Recommended for MVP)**
- **Pros**: Zero setup, models always available
- **Cons**: Larger binary size (600MB-1GB)
- **Use Case**: Developer machines with sufficient storage

**Method 2: Binary + Download (Recommended for Production)**
- **Base Binary**: ~50MB with small model included
- **Large Models**: Downloaded on-demand
- **Advantages**: Smaller initial download, choice of models

#### Model Storage Structure

```
~/.codesearch/
├── models/
│   ├── bundled/
│   │   ├── codebert-small.onnx      # Embedded in binary, extracted on first use
│   │   └── model_manifest.json      # Bundled model metadata
│   ├── downloaded/
│   │   ├── codebert-base.onnx       # Downloaded on demand
│   │   ├── graphcodebert-base.onnx  # Downloaded on demand
│   │   └── downloads.json           # Download history and checksums
│   └── registry.json                # Available models and versions
```

#### Model Download System

**Automatic Downloads:**
```bash
# First use of better model
codesearch search "semantic query" --model codebert-base
# → Model 'codebert-base' not found locally
# → Downloading model (420MB)...
# → Progress: ████████████████████████████████ 100%
# → Model extracted and verified
# → Search continuing with loaded model
```

**Download Configuration:**
```toml
[models]
auto_download = true              # Automatically download missing models
preferred_model = "codebert-base" # Default model choice
max_total_size_mb = 2000         # Maximum total model storage
download_timeout = 300            # Download timeout in seconds

[models.repositories]
primary = "https://github.com/codesearch/models/releases"
fallback = "https://cdn.codesearch.ai/models"
```

#### Model Versioning and Updates

**Version Management:**
- **Semantic Versioning**: Models follow `major.minor.patch` versioning
- **Compatibility Matrix**: Tool version specifies compatible model versions
- **Automatic Updates**: Optional background model updates

**Update Process:**
```bash
# Check for model updates
codesearch models check-updates
# → New model version available: codebert-base v2.1.0 (current: v2.0.3)
# → Release notes: Improved Python semantic understanding
# → Download: codesearch models update codebert-base

# Auto-update configuration
codesearch config set models.auto_update true
```

**Rollback Support:**
```bash
# Rollback to previous model version
codesearch models rollback codebert-base --to-version 2.0.3
# → Previous version kept locally for rollback capability
# → Automatic rollback if model causes issues
```

#### Model Integrity and Security

**Checksum Verification:**
```json
{
  "model": "codebert-base",
  "version": "2.1.0",
  "size_bytes": 420453987,
  "sha256": "a7b9c3d5e8f2...",
  "url": "https://github.com/codesearch/models/releases/v2.1.0/codebert-base.onnx",
  "signature": "MEUCIQD..."
}
```

**Security Measures:**
- **Digital Signatures**: Models signed with project private key
- **Checksum Validation**: SHA256 verification on download
- **Secure Downloads**: HTTPS only, certificate pinning
- **Model Verification**: Runtime validation of model format

#### Model Performance Characteristics

**Model Comparison:**
| Model | Size | Memory | Speed | Quality |
|-------|------|--------|-------|---------|
| codebert-small | 145MB | 800MB | Fast | Good |
| codebert-base | 420MB | 1.2GB | Medium | Excellent |
| graphcodebert | 445MB | 1.3GB | Medium | Excellent* |

*GraphCodeBERT excels at code structure understanding, data flow analysis

**Performance Benchmarks:**
```bash
# Model performance comparison
codesearch benchmark --models all
# Results:
# codebert-small: 45ms avg search time, 78% relevance score
# codebert-base: 82ms avg search time, 89% relevance score
# graphcodebert: 95ms avg search time, 91% relevance score
```

#### Memory Management

**Model Loading Strategy:**
```rust
// Lazy loading with LRU cache
struct ModelManager {
    loaded_models: LruCache<String, OrtSession>,
    max_memory_mb: usize,
    current_memory_mb: usize,
}

impl ModelManager {
    fn load_model(&mut self, model_name: &str) -> Result<&OrtSession> {
        // Unload least recently used model if memory limit exceeded
        if self.current_memory_mb + model_size > self.max_memory_mb {
            self.unload_lru_model();
        }
        // Load requested model
    }
}
```

**Memory Optimization:**
- **Model Caching**: Keep frequently used models in memory
- **Lazy Loading**: Load models on first use
- **Memory Limits**: Configurable memory constraints
- **Model Unloading**: Automatic cleanup when memory pressure detected

#### Multi-Model Support

**Model Selection:**
```bash
# Automatic model selection based on file type
codesearch search "authentication"
# → Uses python-specific model for Python files
# → Falls back to general model for mixed languages

# Manual model selection
codesearch search "data structure" --model graphcodebert
# → Forces use of GraphCodeBERT for structure understanding
```

**Ensemble Search:**
```bash
# Use multiple models for better results
codesearch search "complex algorithm" --ensemble
# → Combines results from multiple models
# → Improves recall, increases search time
```

### 2. AST Parser Choice: **Tree-sitter**
**Decision**: Unified tree-sitter parsing with language-specific grammars

**Rationale:**
- **Multi-language requirement** - Tree-sitter handles this elegantly across 40+ languages
- **Agent-friendly consistency** - Same query patterns across all languages
- **Maintenance simplicity** - One system to learn and maintain vs N language-specific parsers
- **Error recovery** - Graceful handling of syntax errors in incomplete/broken code

**Tradeoffs Accepted**: Less language-aware than dedicated parsers, requires WASM runtime

### 3. Index Storage Format: **SQLite + Custom Binary Hybrid**
**Decision**: SQLite for metadata + custom binary format for vectors

**Rationale:**
- **Performance**: Custom binary format optimizes vector similarity search (memory-mapped access)
- **Flexibility**: SQLite handles complex queries on metadata easily
- **Maintainability**: SQLite handles data integrity and versioning automatically
- **Debugging**: Both formats can be inspected with appropriate tools

**Structure:**
- **SQLite**: File metadata, symbol index, project configuration, indexing status
- **Custom Binary**: Vector embeddings optimized for fast similarity search

### 4. Configuration Management: **Simple JSON**
**Decision**: Simple JSON configuration for MVP

**MVP Approach:**
- **Quick start**: Minimal dependencies, easy to implement
- **Agent discovery**: Agents can easily `cat` and understand config
- **Simplicity**: Most configuration needs are basic initially

**Configuration Hierarchy:**
1. Global defaults (built into binary)
2. System config (`~/.codesearch/config.json`)
3. Project config (`.codesearch.json` in project root)
4. Command line flags (session-specific overrides)

### MVP Configuration Schema

**Simple JSON Configuration (`~/.codesearch/config.json`):**
```json
{
  "max_file_size_mb": 10,
  "exclude_patterns": [".git", "node_modules", "__pycache__"],
  "search_limit": 20,
  "context_lines": 3,
  "model": "codebert-base",
  "auto_download": true
}
```

**Project Configuration (`.codesearch.json`):**
```json
{
  "exclude_patterns": ["coverage", "*.min.js", "dist"],
  "search_limit": 15
}
```

### 5. Updated Technical Stack

**Core Technologies:**
- **Language**: Rust (for performance and single-binary deployment)
- **AST Parsing**: Tree-sitter with WASM runtime
- **Vector Embeddings**: ONNX runtime with CodeBERT/GraphCodeBERT models
- **Metadata Storage**: SQLite
- **Vector Storage**: Custom binary format with memory mapping
- **Configuration**: TOML (production) with JSON fallback (MVP)

**Performance Targets:**
- **Indexing**: < 30 seconds for typical medium project (1000 files)
- **Search**: < 100ms for semantic search queries
- **Memory**: < 500MB for typical usage (including index loading)
- **Storage**: Index size < 10% of source code size


## Technical Validation Requirements

### Critical Validation Areas

The review committee has identified two critical technical areas that require validation before architecture approval:

1. **Vector Search Performance**: Validate CPU-based similarity can meet 100ms targets
2. **Cross-platform Compatibility**: Verify WASM + ONNX runtime behavior

### Validation Test Plan

#### Test 1: Vector Search Performance Validation

**Objective**: Validate that CPU-based vector similarity search can meet the 100ms search target on realistic codebase sizes.

**Test Methodology:**
```bash
# Performance validation test suite
codesearch validate performance --test-type vector-search

# Test across different index sizes
codesearch validate performance --index-sizes small,medium,large

# Test with different vector dimensions
codesearch validate performance --dimensions 384,768,1024
```

**Test Datasets:**
```yaml
vector_search_validation:
  test_cases:
    - name: "small_index"
      file_count: 100
      vector_count: 1500
      dimensions: 768
      target_ms: 30

    - name: "medium_index"
      file_count: 1000
      vector_count: 15000
      dimensions: 768
      target_ms: 80

    - name: "large_index"
      file_count: 10000
      vector_count: 150000
      dimensions: 768
      target_ms: 150

    - name: "xlarge_index"
      file_count: 50000
      vector_count: 750000
      dimensions: 768
      target_ms: 300
```

**Success Criteria:**
- **Small/Medium Indices** (<10K vectors): < 50ms average search time
- **Large Indices** (10K-100K vectors): < 100ms average search time
- **Very Large Indices** (>100K vectors): < 200ms average search time
- **Memory Usage**: < 1GB for typical usage scenarios

**Fallback Strategies if Validation Fails:**
1. **Approximate Search**: Implement HNSW or other approximate nearest neighbor algorithms
2. **Vector Quantization**: Use product quantization to reduce memory footprint
3. **Hybrid Approach**: Combine exact search for recent files with approximate for historical files
4. **Caching Layer**: Implement intelligent result caching for common queries

#### Test 2: Cross-platform WASM + ONNX Compatibility

**Objective**: Verify that Tree-sitter WASM and ONNX runtime work correctly across all target platforms.

**Test Matrix:**
```yaml
platform_validation:
  operating_systems:
    - windows: ["windows-latest", "windows-2019"]
    - macos: ["macos-latest", "macos-13"]
    - linux: ["ubuntu-latest", "ubuntu-20.04"]

  architectures:
    - x86_64
    - arm64 (Apple Silicon)
    - x86 (32-bit where applicable)

  test_scenarios:
    - name: "tree_sitter_parsing"
      description: "Verify Tree-sitter can parse all supported languages"

    - name: "onnx_model_loading"
      description: "Verify ONNX models load and execute correctly"

    - name: "wasm_compilation"
      description: "Verify WASM modules compile and run"

    - name: "memory_constraints"
      description: "Verify reasonable memory usage under constraints"

    - name: "concurrent_execution"
      description: "Verify multiple operations can run concurrently"
```

**Platform-Specific Validation Tests:**
```rust
// Cross-platform validation test suite
pub struct PlatformValidator;

impl PlatformValidator {
    pub fn validate_all() -> ValidationReport {
        let mut report = ValidationReport::new();

        // Test 1: Tree-sitter WASM functionality
        report.add_result("tree_sitter_wasm", Self::test_tree_sitter_wasm());

        // Test 2: ONNX runtime compatibility
        report.add_result("onnx_runtime", Self::test_onnx_runtime());

        // Test 3: Memory management
        report.add_result("memory_management", Self::test_memory_management());

        // Test 4: Concurrency
        report.add_result("concurrent_operations", Self::test_concurrency());

        report
    }

    fn test_onnx_runtime() -> TestResult {
        // Test ONNX session creation
        let environment = ort::Environment::builder()
            .with_log_level(ort::LoggingLevel::Warning)
            .build()
            .map_err(|e| TestError::OnnxError(e.to_string()))?;

        // Test model loading (use small test model)
        let model_data = include_bytes!("../models/test_model.onnx");
        let session = environment.new_session_builder()
            .map_err(|e| TestError::OnnxError(e.to_string()))?
            .with_optimization_level(ort::GraphOptimizationLevel::Level1)
            .map_err(|e| TestError::OnnxError(e.to_string()))?
            .with_model_from_memory(model_data)
            .map_err(|e| TestError::OnnxError(e.to_string()))?;

        // Test inference
        let input_tensor = ort::Value::from_array(([1, 768], vec![0.0f32; 768]))
            .map_err(|e| TestError::OnnxError(e.to_string()))?;

        let outputs = session.run(vec![input_tensor])
            .map_err(|e| TestError::OnnxError(e.to_string()))?;

        if outputs.is_empty() {
            return Err(TestError::OnnxError("No outputs from model".to_string()));
        }

        Ok(())
    }
}
```

### Validation Success Criteria

#### Vector Search Performance
- ✅ **Small indices** (<1K vectors): < 20ms average search time
- ✅ **Medium indices** (1K-10K vectors): < 50ms average search time
- ✅ **Large indices** (10K-100K vectors): < 100ms average search time
- ✅ **Memory efficiency**: < 1GB for typical workloads
- ✅ **Scalability**: Linear or sub-linear performance degradation

#### Cross-platform Compatibility
- ✅ **All target platforms**: Windows, macOS, Linux (x86_64)
- ✅ **ARM64 support**: macOS (Apple Silicon), Linux
- ✅ **WASM functionality**: Tree-sitter works across all platforms
- ✅ **ONNX inference**: Models load and execute correctly
- ✅ **Memory management**: No crashes or excessive memory usage
- ✅ **Concurrent operations**: Multiple threads work safely

### Validation Timeline

**Phase 1: Prototype Validation (2 weeks)**
- Implement basic vector search with sample data
- Test Tree-sitter + ONNX on development machine
- Validate core functionality works

**Phase 2: Cross-platform Testing (1 week)**
- Set up CI/CD pipeline with matrix testing
- Test on Windows, macOS, Linux across architectures
- Identify and fix platform-specific issues

**Phase 3: Performance Validation (1 week)**
- Benchmark vector search at scale
- Test memory usage under various loads
- Optimize based on results

**Phase 4: Integration Validation (1 week)**
- End-to-end testing of complete system
- Validate dogfooding capability
- Performance regression testing

### Risk Mitigation

**If Vector Search Performance Fails:**
1. **Immediate**: Implement approximate search algorithms
2. **Short-term**: Optimize vector storage and access patterns
3. **Long-term**: Consider GPU acceleration for large-scale deployments

**If Cross-platform Compatibility Fails:**
1. **Platform-specific builds**: Use native libraries where WASM fails
2. **Graceful degradation**: Disable problematic features on affected platforms
3. **Alternative runtimes**: Consider different WASM or ONNX runtime implementations

### Validation Deliverables

1. **Performance Benchmark Report**: Detailed metrics across all test scenarios
2. **Cross-platform Compatibility Matrix**: Pass/fail status for each platform/architecture
3. **Memory Usage Analysis**: Peak and average memory consumption patterns
4. **Concurrency Test Results**: Thread safety and performance under load
5. **Risk Assessment**: Identified risks and mitigation strategies
6. **Go/No-Go Recommendation**: Clear recommendation for proceeding to architecture phase

This comprehensive validation plan ensures we identify and address technical risks early, providing confidence that the chosen technical stack will meet the performance and compatibility requirements before significant architecture investment.

## Conclusion

This tool fills a critical gap in the agentic coding ecosystem by providing efficient, semantic code search that's simple to install and use. By focusing on agent-first design and dogfooding from day one, we ensure the tool solves real problems for both AI agents and their human collaborators.

The specification prioritizes practical utility over comprehensive feature coverage, ensuring the tool becomes valuable quickly while maintaining a clear path for future enhancement.