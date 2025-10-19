# Agent-First Codebase Search Tool - Comprehensive Implementation Plan

## Overview

This document provides a complete implementation roadmap for building the agent-first codebase search tool. It's designed for a skilled engineer with minimal context about this codebase or problem domain.

**Project Vision**: A locally-installed CLI tool that provides semantic and symbol-aware code search capabilities for AI coding agents, solving the token inefficiency and noise problems of grep-only retrieval while avoiding MCP complexity.

**Target Engineer Profile**:
- Skilled developer (Rust experience preferred)
- Limited knowledge of search/vector technologies
- Needs detailed guidance on testing, file structure, and domain-specific concepts
- Follows TDD, frequent commits, YAGNI principles

---

## Table of Contents

1. [Project Context & Goals](#project-context--goals)
2. [Technical Architecture Overview](#technical-architecture-overview)
3. [Development Environment Setup](#development-environment-setup)
4. [Implementation Phases](#implementation-phases)
5. [Phase 0: Foundation & Validation](#phase-0-foundation--validation)
6. [Phase 1: MVP Implementation](#phase-1-mvp-implementation)
7. [Phase 2: Multi-Language Support](#phase-2-multi-language-support)
8. [Phase 3: Advanced Features](#phase-3-advanced-features)
9. [Testing Strategy](#testing-strategy)
10. [Development Workflow](#development-workflow)
11. [File Structure Guide](#file-structure-guide)
12. [Common Pitfalls & Solutions](#common-pitfalls--solutions)

---

## Project Context & Goals

### Problem Statement
- **Current Issue**: Claude Code's grep-only retrieval "drowns you in irrelevant matches, burns tokens, and stalls your workflow"
- **Complexity Barrier**: Existing semantic solutions require Docker, complex setup, and maintenance overhead
- **Missing Tool**: No simple, self-contained way for coding agents to perform semantic code search on local codebases

### Success Metrics
- **Agent Discovery**: Tool capabilities understood within 2-3 help commands
- **Performance**: Search results < 100ms for indexed codebases
- **Relevance**: Top 3 results contain relevant information > 80% of the time
- **Token Efficiency**: Reduce context needed vs grep by 40%+
- **Developer Experience**: Setup within 5 minutes, incremental indexing < 10 seconds

### Core Value Proposition
- **Agent-First**: Designed specifically for AI coding agents
- **Simple**: Single binary, zero configuration for basic use
- **Local**: All data stays on user machine, no network dependencies
- **Fast**: Sub-second search on typical codebases
- **Smart**: Semantic understanding goes beyond pattern matching

---

## Technical Architecture Overview

### Core Technologies
- **Language**: Rust (performance, single-binary deployment)
- **AST Parsing**: Tree-sitter with WASM runtime (multi-language support)
- **Vector Embeddings**: ONNX runtime with CodeBERT models (semantic search)
- **Metadata Storage**: SQLite (file metadata, symbols, configuration)
- **Vector Storage**: Custom binary format with memory mapping (performance)
- **Configuration**: JSON for MVP, TOML for production

### High-Level Architecture
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

### Key Design Decisions
1. **Local-First**: No network dependencies, all processing on user machine
2. **Incremental**: Fast updates, only process changed files
3. **Self-Contained**: Single binary with embedded models, zero setup complexity
4. **Agent-Friendly**: Git-style CLI, structured output, comprehensive help

---

## Development Environment Setup

### Prerequisites
- Rust 1.70+ (stable toolchain)
- Git (for version control)
- Basic familiarity with CLI tools and testing

### Initial Setup Commands
```bash
# 1. Clone and initialize repository
git clone <repository-url>
cd local-index

# 2. Install Rust toolchain (if not already installed)
rustup update stable
rustup component add rustfmt clippy

# 3. Verify environment
rustc --version
cargo --version

# 4. Set up development tools
cargo install cargo-watch  # For auto-reloading during development
cargo install cargo-nextest # For better test runner
```

### Project Structure (to be created)
```
local-index/
├── Cargo.toml                 # Project dependencies
├── Cargo.lock                 # Lock file (generated)
├── README.md                  # Project documentation
├── AGENT_FIRST_CODEBASE_SEARCH_SPEC.md  # Existing specification
├── src/                       # Source code
│   ├── main.rs               # CLI entry point
│   ├── lib.rs                # Library interface
│   ├── cli/                  # CLI module
│   ├── index/                # Index engine
│   ├── search/               # Search engine
│   ├── models/               # Model management
│   ├── storage/              # Data storage
│   ├── parsers/              # Language parsers
│   └── config/               # Configuration management
├── tests/                     # Integration tests
├── benches/                   # Performance benchmarks
├── models/                    # ML models (embedded)
└── docs/                      # Documentation
```

---

## Implementation Phases

### Phase Overview
- **Phase 0 (1-2 weeks)**: Foundation & Technical Validation
- **Phase 1 (3-4 weeks)**: MVP - Basic semantic search for 2-3 languages
- **Phase 2 (2-3 weeks)**: Multi-language support and improved parsing
- **Phase 3 (2-3 weeks)**: Advanced features and optimization

### Success Criteria for Each Phase
- **Phase 0**: Technical validation complete, architecture decisions confirmed
- **Phase 1**: Tool can search its own codebase (dogfooding), basic CLI works
- **Phase 2**: Supports JS, Python, YAML, MD, Shell with good accuracy
- **Phase 3**: Performance targets met, advanced features implemented

---

## Phase 0: Foundation & Validation

### Goal: Validate core technical assumptions before full implementation

**Critical Risk Areas**:
1. **Vector Search Performance**: Can CPU-based similarity meet 100ms targets?
2. **Cross-Platform Compatibility**: Do WASM + ONNX runtimes work reliably?

### Tasks (Estimated: 1-2 weeks)

#### Task 0.1: Project Foundation Setup
**Files to Create**:
- `Cargo.toml` - Project configuration and dependencies
- `src/main.rs` - Basic CLI structure
- `src/lib.rs` - Library interface
- `.gitignore` - Ignore build artifacts and index data
- `README.md` - Basic project documentation

**Dependencies to Add**:
```toml
[dependencies]
# CLI framework
clap = { version = "4.0", features = ["derive"] }
# Error handling
anyhow = "1.0"
thiserror = "1.0"
# Serialization
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
# Database
rusqlite = { version = "0.28", features = ["bundled"] }
# File system
notify = "6.0"
walkdir = "2.0"
# Async runtime
tokio = { version = "1.0", features = ["full"] }
# Logging
tracing = "0.1"
tracing-subscriber = "0.3"
# Configuration
config = "0.13"
# Vector operations (placeholder)
ndarray = "0.15"

[dev-dependencies]
tempfile = "3.0"
criterion = "0.5"
```

**Implementation Steps**:
1. Initialize Cargo project
2. Set up basic CLI with `clap`
3. Create modular project structure
4. Add comprehensive error handling
5. Set up logging and tracing
6. Create basic tests for CLI structure

**Testing Requirements**:
- Unit tests for CLI argument parsing
- Integration tests for basic command structure
- Verify project builds cleanly on target platforms

**Commit Strategy**:
- Initial commit: Project foundation
- Second commit: CLI structure working
- Third commit: Basic tests passing

#### Task 0.2: Vector Search Performance Validation
**Files to Create**:
- `tests/vector_search_validation.rs` - Performance test suite
- `benches/vector_benchmark.rs` - Performance benchmarks
- `src/validation/` - Validation module

**Implementation Steps**:
1. Create sample vector data sets (1K, 10K, 100K vectors)
2. Implement basic similarity search using CPU operations
3. Benchmark search times across different data sizes
4. Test memory usage patterns
5. Validate against performance targets (<100ms for typical cases)

**Sample Test Structure**:
```rust
// tests/vector_search_validation.rs
#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Instant;

    #[test]
    fn test_small_index_performance() {
        let vectors = generate_test_vectors(1000, 768);
        let query = generate_query_vector(768);

        let start = Instant::now();
        let results = search_similar(&query, &vectors, 10);
        let duration = start.elapsed();

        assert!(duration.as_millis() < 50, "Small index search should be <50ms, got {:?}", duration);
        assert_eq!(results.len(), 10);
    }

    #[test]
    fn test_medium_index_performance() {
        // Similar test for 10K vectors, target <100ms
    }

    #[test]
    fn test_memory_usage() {
        // Test memory doesn't exceed 1GB for typical workloads
    }
}
```

**Success Criteria**:
- Small indices (<1K vectors): <20ms search time
- Medium indices (1K-10K vectors): <50ms search time
- Large indices (10K-100K vectors): <100ms search time
- Memory usage <1GB for typical workloads

**Fallback Plans if Validation Fails**:
1. Implement approximate nearest neighbor algorithms (HNSW)
2. Use vector quantization to reduce memory footprint
3. Implement hybrid search (exact for recent, approximate for historical)

#### Task 0.3: Cross-Platform Compatibility Validation
**Files to Create**:
- `tests/platform_validation.rs` - Cross-platform test suite
- `src/validation/platform.rs` - Platform validation logic

**Implementation Steps**:
1. Test Tree-sitter WASM compilation and execution
2. Test ONNX runtime model loading and inference
3. Verify memory management across platforms
4. Test concurrent operations
5. Validate file system operations

**Sample Validation Tests**:
```rust
// tests/platform_validation.rs
#[cfg(test)]
mod tests {
    #[test]
    fn test_tree_sitter_wasm_functionality() {
        // Test that Tree-sitter can parse basic code snippets
        let parser = tree_sitter::Parser::new();
        // Verify parsing works across platforms
    }

    #[test]
    fn test_onnx_runtime_compatibility() {
        // Test ONNX session creation and basic inference
        // Verify models load correctly on all platforms
    }

    #[test]
    fn test_memory_management() {
        // Test memory usage stays reasonable
        // Verify no memory leaks or crashes
    }
}
```

**Testing Matrix**:
- Operating Systems: Windows 10/11, macOS (Intel/Apple Silicon), Linux (Ubuntu/CentOS)
- Architectures: x86_64, ARM64 (where applicable)
- Memory Constraints: Test with 512MB, 1GB, 2GB available memory

**Success Criteria**:
- All target platforms can load and run models
- No platform-specific crashes or panics
- Reasonable memory usage across platforms
- Concurrent operations work safely

#### Task 0.4: Architecture Decision Documentation
**Files to Create**:
- `docs/architecture_decisions.md` - Record all technical decisions
- `docs/validation_report.md` - Summary of validation results

**Implementation Steps**:
1. Document all validation results
2. Record final architecture decisions
3. Create risk assessment and mitigation strategies
4. Document any changes to original specifications

**Decision Documentation Format**:
```markdown
## AD-001: Vector Search Algorithm Decision

**Status**: Approved
**Date**: [Date]
**Context**: Performance validation for CPU-based similarity search

**Decision**: Use exact similarity search with optimization
- Small/Medium indices: Exact search (meets performance targets)
- Large indices: Implement HNSW approximate search if needed
- Memory: Use memory-mapped files for vector storage

**Consequences**:
- Simpler implementation for typical use cases
- Good performance for most codebases (<100K vectors)
- Fallback strategy ready for very large repositories

**Validation Results**:
- Small indices (1K vectors): 15ms average
- Medium indices (10K vectors): 45ms average
- Large indices (100K vectors): 120ms average (needs optimization)
```

---

## Phase 1: MVP Implementation

### Goal: Build minimal viable product that can search its own codebase

**MVP Scope**:
- Support 2-3 languages (Rust, Python, Markdown)
- Basic semantic search functionality
- Simple CLI with core commands (index, search, status)
- Local storage with basic incremental updates
- Tool can search its own source code (dogfooding)

### Tasks (Estimated: 3-4 weeks)

#### Task 1.1: Core CLI Framework
**Files to Create/Modify**:
- `src/main.rs` - Main CLI entry point
- `src/cli/` - CLI command modules
- `src/cli/commands.rs` - Command definitions
- `src/cli/index.rs` - Index command implementation
- `src/cli/search.rs` - Search command implementation
- `src/cli/status.rs` - Status command implementation

**Implementation Steps**:
1. Define CLI structure using `clap` derive macros
2. Implement subcommand routing
3. Add comprehensive help text and examples
4. Create structured output formats (text and JSON)
5. Add error handling for CLI operations

**CLI Structure Design**:
```rust
// src/cli/commands.rs
use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "codesearch")]
#[command(about = "Agent-first semantic code search tool")]
#[command(version)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,

    /// Output results in JSON format
    #[arg(short, long, global = true)]
    pub json: bool,

    /// Maximum number of results to return
    #[arg(short, long, global = true, default_value = "20")]
    pub limit: usize,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Initialize and index a directory
    Index {
        /// Path to index (default: current directory)
        #[arg(default_value = ".")]
        path: PathBuf,

        /// Force full reindex instead of incremental
        #[arg(long)]
        force: bool,
    },

    /// Search for code using semantic queries
    Search {
        /// Search query
        query: String,

        /// Filter by file type
        #[arg(short = 't', long)]
        r#type: Option<String>,
    },

    /// Find specific symbols, functions, or classes
    Find {
        /// Symbol name to search for
        symbol: String,

        /// Search for exact matches only
        #[arg(long)]
        exact: bool,
    },

    /// Show indexing status and statistics
    Status {
        /// Path to check (default: current directory)
        #[arg(default_value = ".")]
        path: PathBuf,
    },
}
```

**Testing Requirements**:
- Unit tests for CLI argument parsing
- Integration tests for each command
- Help text validation (agents can discover capabilities)
- Error handling tests for invalid inputs

**Example Tests**:
```rust
// tests/cli_tests.rs
#[cfg(test)]
mod tests {
    use assert_cmd::Command;
    use predicates::prelude::*;

    #[test]
    fn test_cli_help_discovery() {
        let cmd = Command::cargo_bin("codesearch").unwrap();
        cmd.arg("--help")
            .assert()
            .success()
            .stdout(predicate::str::contains("Agent-first semantic code search"));
    }

    #[test]
    fn test_search_command_help() {
        let cmd = Command::cargo_bin("codesearch").unwrap();
        cmd.args(["search", "--help"])
            .assert()
            .success()
            .stdout(predicate::str::contains("Search for code using semantic queries"));
    }

    #[test]
    fn test_index_command_validation() {
        let cmd = Command::cargo_bin("codesearch").unwrap();
        cmd.args(["index", "/nonexistent/path"])
            .assert()
            .failure()
            .stderr(predicate::str::contains("Path does not exist"));
    }
}
```

#### Task 1.2: File System Integration
**Files to Create/Modify**:
- `src/filesystem/` - File system operations module
- `src/filesystem/watcher.rs` - File watching functionality
- `src/filesystem/scanner.rs` - File scanning and filtering
- `src/filesystem/validator.rs` - File validation and metadata

**Implementation Steps**:
1. Implement file discovery and filtering
2. Add file watching with debouncing
3. Create file validation logic (size limits, permissions)
4. Implement exclusion pattern handling
5. Add metadata extraction

**File System Operations**:
```rust
// src/filesystem/scanner.rs
use std::path::PathBuf;
use walkdir::WalkDir;

pub struct FileScanner {
    exclude_patterns: Vec<String>,
    max_file_size: u64,
    supported_extensions: Vec<String>,
}

impl FileScanner {
    pub fn scan_directory(&self, path: &Path) -> Result<Vec<PathBuf>, ScanError> {
        let mut files = Vec::new();

        for entry in WalkDir::new(path)
            .follow_links(false)
            .into_iter()
            .filter_entry(|e| !self.is_excluded(e.path()))
        {
            let entry = entry.map_err(ScanError::Walkdir)?;
            if entry.file_type().is_file() {
                if self.should_index_file(entry.path())? {
                    files.push(entry.path().to_path_buf());
                }
            }
        }

        Ok(files)
    }

    fn should_index_file(&self, path: &Path) -> Result<bool, ScanError> {
        // Check file size
        let metadata = std::fs::metadata(path)
            .map_err(ScanError::Io)?;
        if metadata.len() > self.max_file_size {
            return Ok(false);
        }

        // Check file extension
        if let Some(extension) = path.extension().and_then(|s| s.to_str()) {
            Ok(self.supported_extensions.contains(&extension.to_string()))
        } else {
            Ok(false)
        }
    }

    fn is_excluded(&self, path: &Path) -> bool {
        let path_str = path.to_string_lossy();
        self.exclude_patterns.iter()
            .any(|pattern| path_str.contains(pattern))
    }
}
```

**Testing Requirements**:
- Unit tests for file scanning logic
- Tests for exclusion pattern handling
- Performance tests for large directories
- File watching integration tests

#### Task 1.3: Basic Language Parsing (Tree-sitter Integration)
**Files to Create/Modify**:
- `src/parsers/` - Language parsing module
- `src/parsers/mod.rs` - Parser trait and registry
- `src/parsers/rust.rs` - Rust language parser
- `src/parsers/python.rs` - Python language parser
- `src/parsers/markdown.rs` - Markdown parser

**Implementation Steps**:
1. Set up Tree-sitter with required language grammars
2. Implement parser trait for consistent interface
3. Create parsers for MVP languages
4. Add symbol extraction (functions, classes, variables)
5. Implement error handling for parsing failures

**Parser Interface Design**:
```rust
// src/parsers/mod.rs
use tree_sitter::{Language, Parser, Tree};
use std::path::Path;

pub trait LanguageParser: Send + Sync {
    fn language(&self) -> Language;
    fn file_extensions(&self) -> Vec<&'static str>;
    fn parse_file(&self, content: &str) -> Result<ParseResult, ParseError>;
    fn extract_symbols(&self, tree: &Tree, content: &str) -> Vec<Symbol>;
}

#[derive(Debug)]
pub struct ParseResult {
    pub tree: Tree,
    pub symbols: Vec<Symbol>,
    pub embeddings: Vec<Vec<f32>>, // Will be populated later
}

#[derive(Debug, Clone)]
pub struct Symbol {
    pub name: String,
    pub kind: SymbolKind,
    pub start_line: usize,
    pub end_line: usize,
    pub start_byte: usize,
    pub end_byte: usize,
    pub parent: Option<String>,
}

#[derive(Debug, Clone)]
pub enum SymbolKind {
    Function,
    Class,
    Variable,
    Import,
    Module,
    // Add more as needed
}
```

**Rust Parser Implementation**:
```rust
// src/parsers/rust.rs
use tree_sitter_rust::language;
use super::*;

pub struct RustParser;

impl LanguageParser for RustParser {
    fn language(&self) -> Language {
        language()
    }

    fn file_extensions(&self) -> Vec<&'static str> {
        vec!["rs"]
    }

    fn parse_file(&self, content: &str) -> Result<ParseResult, ParseError> {
        let mut parser = Parser::new();
        parser.set_language(self.language())
            .map_err(|_| ParseError::LanguageSetup)?;

        let tree = parser.parse(content, None)
            .ok_or(ParseError::ParseFailed)?;

        let symbols = self.extract_symbols(&tree, content);

        Ok(ParseResult {
            tree,
            symbols,
            embeddings: Vec::new(), // Will be added in Task 1.4
        })
    }

    fn extract_symbols(&self, tree: &Tree, content: &str) -> Vec<Symbol> {
        let mut symbols = Vec::new();
        let mut cursor = tree.walk();

        // Walk the AST and extract symbols
        // This is simplified - real implementation would be more sophisticated
        for node in tree.root_node().children(&mut cursor) {
            match node.kind() {
                "function_item" => {
                    if let Some(name_node) = node.child_by_field_name("name") {
                        let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
                        symbols.push(Symbol {
                            name,
                            kind: SymbolKind::Function,
                            start_line: node.start_position().row + 1,
                            end_line: node.end_position().row + 1,
                            start_byte: node.start_byte(),
                            end_byte: node.end_byte(),
                            parent: None,
                        });
                    }
                }
                // Add more symbol types...
                _ => {}
            }
        }

        symbols
    }
}
```

**Testing Requirements**:
- Unit tests for each language parser
- Tests for symbol extraction accuracy
- Error handling tests for malformed code
- Performance tests for large files

#### Task 1.4: Model Integration and Vector Generation
**Files to Create/Modify**:
- `src/models/` - Model management module
- `src/models/manager.rs` - Model loading and caching
- `src/models/embeddings.rs` - Vector embedding generation
- `src/models/registry.rs` - Model registry and metadata
- `models/` - Embedded ONNX models directory

**Implementation Steps**:
1. Integrate ONNX runtime for model loading
2. Implement embedding generation for code snippets
3. Create model management system (loading, caching, unloading)
4. Add model integrity verification
5. Implement fallback for model loading failures

**Model Manager Design**:
```rust
// src/models/manager.rs
use std::collections::HashMap;
use std::sync::Mutex;

pub struct ModelManager {
    loaded_models: Mutex<HashMap<String, LoadedModel>>,
    config: ModelConfig,
}

#[derive(Debug)]
pub struct LoadedModel {
    pub session: OrtSession,
    pub metadata: ModelMetadata,
    pub memory_usage_mb: usize,
}

impl ModelManager {
    pub fn new(config: ModelConfig) -> Result<Self, ModelError> {
        Ok(Self {
            loaded_models: Mutex::new(HashMap::new()),
            config,
        })
    }

    pub fn generate_embeddings(&self, code_snippets: &[String]) -> Result<Vec<Vec<f32>>, ModelError> {
        let model_name = &self.config.default_model;
        let mut models = self.loaded_models.lock().unwrap();

        let model = models.entry(model_name.clone())
            .or_insert_with(|| self.load_model(model_name).unwrap());

        // Preprocess code snippets
        let inputs = self.preprocess_inputs(code_snippets, &model.metadata)?;

        // Run inference
        let outputs = model.session.run(inputs)?;

        // Postprocess outputs to get embeddings
        self.extract_embeddings(outputs)
    }

    fn load_model(&self, model_name: &str) -> Result<LoadedModel, ModelError> {
        // Load model from embedded data or download if needed
        let model_data = self.get_model_data(model_name)?;

        let environment = ort::Environment::builder()
            .with_log_level(ort::LoggingLevel::Warning)
            .build()?;

        let session = environment.new_session_builder()?
            .with_optimization_level(ort::GraphOptimizationLevel::Level1)?
            .with_model_from_memory(&model_data)?;

        let metadata = self.load_model_metadata(model_name)?;

        Ok(LoadedModel {
            session,
            metadata,
            memory_usage_mb: model_data.len() / (1024 * 1024),
        })
    }
}
```

**Embedding Generation**:
```rust
// src/models/embeddings.rs
pub struct EmbeddingGenerator {
    model_manager: ModelManager,
    tokenizer: CodeTokenizer,
}

impl EmbeddingGenerator {
    pub fn generate_file_embeddings(&self, file_content: &str) -> Result<Vec<FileEmbedding>, EmbeddingError> {
        // Split file into meaningful chunks (functions, classes, etc.)
        let chunks = self.chunk_code(file_content)?;

        // Generate embeddings for each chunk
        let chunk_texts: Vec<String> = chunks.iter()
            .map(|chunk| chunk.extract_text(file_content))
            .collect();

        let embeddings = self.model_manager.generate_embeddings(&chunk_texts)?;

        // Combine chunk embeddings with metadata
        let mut file_embeddings = Vec::new();
        for (chunk, embedding) in chunks.into_iter().zip(embeddings.into_iter()) {
            file_embeddings.push(FileEmbedding {
                chunk,
                embedding,
                timestamp: chrono::Utc::now(),
            });
        }

        Ok(file_embeddings)
    }

    fn chunk_code(&self, content: &str) -> Result<Vec<CodeChunk>, EmbeddingError> {
        // Intelligent code chunking based on AST
        // Split at function boundaries, class definitions, etc.
        // This is simplified - real implementation would be more sophisticated
        let mut chunks = Vec::new();
        let lines: Vec<&str> = content.lines().collect();
        let mut current_chunk_start = 0;

        for (i, line) in lines.iter().enumerate() {
            if line.trim().starts_with("fn ") || line.trim().starts_with("def ") || line.trim().starts_with("class ") {
                if i > current_chunk_start + 1 {
                    chunks.push(CodeChunk {
                        start_line: current_chunk_start,
                        end_line: i - 1,
                        chunk_type: ChunkType::Code,
                    });
                }
                current_chunk_start = i;
            }
        }

        // Add final chunk
        if current_chunk_start < lines.len() {
            chunks.push(CodeChunk {
                start_line: current_chunk_start,
                end_line: lines.len() - 1,
                chunk_type: ChunkType::Code,
            });
        }

        Ok(chunks)
    }
}
```

**Testing Requirements**:
- Unit tests for model loading and unloading
- Tests for embedding generation quality
- Memory usage tests for model operations
- Fallback behavior tests when models fail

#### Task 1.5: Storage System Implementation
**Files to Create/Modify**:
- `src/storage/` - Storage module
- `src/storage/metadata.rs` - SQLite metadata storage
- `src/storage/vectors.rs` - Vector storage in custom binary format
- `src/storage/index.rs` - Index management and operations
- `src/storage/migration.rs` - Schema migration handling

**Implementation Steps**:
1. Set up SQLite database for metadata
2. Implement custom binary format for vectors
3. Create index operations (create, read, update, delete)
4. Add transaction handling and concurrency safety
5. Implement backup and recovery mechanisms

**Database Schema Design**:
```sql
-- src/storage/schema.sql
CREATE TABLE projects (
    id INTEGER PRIMARY KEY,
    path TEXT UNIQUE NOT NULL,
    hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE files (
    id INTEGER PRIMARY KEY,
    project_id INTEGER NOT NULL,
    path TEXT NOT NULL,
    hash TEXT NOT NULL,
    size INTEGER NOT NULL,
    language TEXT,
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_modified DATETIME,
    FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE
);

CREATE TABLE symbols (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    start_byte INTEGER NOT NULL,
    end_byte INTEGER NOT NULL,
    parent_symbol_id INTEGER,
    FOREIGN KEY (file_id) REFERENCES files (id) ON DELETE CASCADE,
    FOREIGN KEY (parent_symbol_id) REFERENCES symbols (id)
);

CREATE TABLE chunks (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    chunk_type TEXT NOT NULL,
    vector_offset INTEGER NOT NULL,  -- Offset in binary vector file
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (file_id) REFERENCES files (id) ON DELETE CASCADE
);

CREATE TABLE index_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX idx_files_project_path ON files (project_id, path);
CREATE INDEX idx_symbols_file_name ON symbols (file_id, name);
CREATE INDEX idx_symbols_kind ON symbols (kind);
CREATE INDEX idx_chunks_file_lines ON chunks (file_id, start_line);
```

**Vector Storage Implementation**:
```rust
// src/storage/vectors.rs
use std::fs::{File, OpenOptions};
use std::io::{Read, Write, Seek, SeekFrom};
use std::path::Path;

pub struct VectorStorage {
    file: File,
    header: VectorFileHeader,
}

#[derive(Debug, Clone)]
pub struct VectorFileHeader {
    magic: [u8; 4],
    version: u32,
    vector_count: u32,
    vector_dimension: u32,
    checksum: u64,
}

impl VectorStorage {
    pub fn create(file_path: &Path, vector_dimension: u32) -> Result<Self, VectorStorageError> {
        let file = OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .truncate(true)
            .open(file_path)?;

        let header = VectorFileHeader {
            magic: *b"CSV\0", // CodeSearch Vector
            version: 1,
            vector_count: 0,
            vector_dimension,
            checksum: 0,
        };

        let mut storage = Self { file, header };
        storage.write_header()?;
        Ok(storage)
    }

    pub fn append_vector(&mut self, vector: &[f32]) -> Result<u32, VectorStorageError> {
        if vector.len() != self.header.vector_dimension as usize {
            return Err(VectorStorageError::DimensionMismatch);
        }

        // Seek to end of file
        self.file.seek(SeekFrom::End(0))?;

        // Write vector data
        for &value in vector {
            self.file.write_all(&value.to_le_bytes())?;
        }

        // Update header
        self.header.vector_count += 1;
        self.write_header()?;

        Ok(self.header.vector_count - 1) // Return offset of new vector
    }

    pub fn get_vector(&mut self, offset: u32) -> Result<Vec<f32>, VectorStorageError> {
        if offset >= self.header.vector_count {
            return Err(VectorStorageError::InvalidOffset);
        }

        let vector_size = std::mem::size_of::<f32>() * self.header.vector_dimension as usize;
        let file_offset = std::mem::size_of::<VectorFileHeader>() + offset as usize * vector_size;

        self.file.seek(SeekFrom::Start(file_offset as u64))?;

        let mut buffer = vec![0u8; vector_size];
        self.file.read_exact(&mut buffer)?;

        let vector: Vec<f32> = buffer.chunks_exact(4)
            .map(|chunk| f32::from_le_bytes([chunk[0], chunk[1], chunk[2], chunk[3]]))
            .collect();

        Ok(vector)
    }

    fn write_header(&mut self) -> Result<(), VectorStorageError> {
        self.file.seek(SeekFrom::Start(0))?;

        // Write magic bytes
        self.file.write_all(&self.header.magic)?;

        // Write version
        self.file.write_all(&self.header.version.to_le_bytes())?;

        // Write vector count
        self.file.write_all(&self.header.vector_count.to_le_bytes())?;

        // Write vector dimension
        self.file.write_all(&self.header.vector_dimension.to_le_bytes())?;

        // Write checksum (placeholder for now)
        self.file.write_all(&self.header.checksum.to_le_bytes())?;

        self.file.flush()?;
        Ok(())
    }
}
```

**Testing Requirements**:
- Unit tests for database operations
- Tests for vector storage integrity
- Concurrency tests for multiple accessors
- Performance tests for large datasets

#### Task 1.6: Search Engine Implementation
**Files to Create/Modify**:
- `src/search/` - Search engine module
- `src/search/engine.rs` - Main search orchestration
- `src/search/semantic.rs` - Semantic similarity search
- `src/search/symbol.rs` - Symbol-based search
- `src/search/hybrid.rs` - Combined semantic + symbol search
- `src/scoring.rs` - Relevance scoring algorithms

**Implementation Steps**:
1. Implement vector similarity search
2. Add symbol-based exact and fuzzy search
3. Create hybrid search combining both approaches
4. Implement relevance scoring and ranking
5. Add result pagination and context extraction

**Search Engine Design**:
```rust
// src/search/engine.rs
use std::sync::Arc;

pub struct SearchEngine {
    storage: Arc<Storage>,
    model_manager: Arc<ModelManager>,
    config: SearchConfig,
}

#[derive(Debug)]
pub struct SearchResult {
    pub file_path: String,
    pub start_line: usize,
    pub end_line: usize,
    pub score: f32,
    pub result_type: SearchResultType,
    pub context: String,
    pub code_snippet: String,
    pub symbols: Vec<String>,
}

#[derive(Debug)]
pub enum SearchResultType {
    SemanticMatch(f32),      // Similarity score
    ExactSymbolMatch,        // Exact symbol name match
    FuzzySymbolMatch(f32),   // Fuzzy match score
    HybridMatch(f32),        // Combined score
}

impl SearchEngine {
    pub fn new(storage: Arc<Storage>, model_manager: Arc<ModelManager>) -> Self {
        Self {
            storage,
            model_manager,
            config: SearchConfig::default(),
        }
    }

    pub async fn search(&self, query: &SearchQuery) -> Result<Vec<SearchResult>, SearchError> {
        match query.query_type {
            QueryType::Semantic => self.semantic_search(query).await,
            QueryType::Symbol => self.symbol_search(query).await,
            QueryType::Hybrid => self.hybrid_search(query).await,
        }
    }

    async fn semantic_search(&self, query: &SearchQuery) -> Result<Vec<SearchResult>, SearchError> {
        // Generate embedding for query
        let query_embedding = self.model_manager.generate_embeddings(&[query.text.clone()])?
            .into_iter().next()
            .ok_or(SearchError::EmbeddingGeneration)?;

        // Get candidate chunks from database
        let candidates = self.storage.get_candidate_chunks(&query.filters)?;

        // Calculate similarity scores
        let mut results = Vec::new();
        for candidate in candidates {
            let chunk_embedding = self.storage.get_vector(candidate.vector_offset)?;
            let similarity = cosine_similarity(&query_embedding, &chunk_embedding);

            if similarity >= self.config.semantic_threshold {
                results.push(SearchResult {
                    file_path: candidate.file_path,
                    start_line: candidate.start_line,
                    end_line: candidate.end_line,
                    score: similarity,
                    result_type: SearchResultType::SemanticMatch(similarity),
                    context: self.extract_context(candidate, 3),
                    code_snippet: self.extract_code_snippet(candidate),
                    symbols: candidate.symbols,
                });
            }
        }

        // Sort by score and apply limit
        results.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap());
        results.truncate(query.limit);

        Ok(results)
    }

    async fn symbol_search(&self, query: &SearchQuery) -> Result<Vec<SearchResult>, SearchError> {
        // Search for exact symbol matches
        let exact_matches = self.storage.find_symbols_exact(&query.text, &query.filters)?;

        // Search for fuzzy symbol matches if needed
        let fuzzy_matches = if exact_matches.len() < query.limit {
            self.storage.find_symbols_fuzzy(&query.text, &query.filters)?
        } else {
            Vec::new()
        };

        // Convert matches to search results
        let mut results = Vec::new();

        for symbol_match in exact_matches {
            results.push(SearchResult {
                file_path: symbol_match.file_path,
                start_line: symbol_match.start_line,
                end_line: symbol_match.end_line,
                score: 1.0, // Perfect match
                result_type: SearchResultType::ExactSymbolMatch,
                context: self.extract_context(&symbol_match, 3),
                code_snippet: self.extract_symbol_code(&symbol_match),
                symbols: vec![symbol_match.name],
            });
        }

        for symbol_match in fuzzy_matches {
            results.push(SearchResult {
                file_path: symbol_match.file_path,
                start_line: symbol_match.start_line,
                end_line: symbol_match.end_line,
                score: symbol_match.similarity,
                result_type: SearchResultType::FuzzySymbolMatch(symbol_match.similarity),
                context: self.extract_context(&symbol_match, 3),
                code_snippet: self.extract_symbol_code(&symbol_match),
                symbols: vec![symbol_match.name],
            });
        }

        // Sort and limit results
        results.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap());
        results.truncate(query.limit);

        Ok(results)
    }

    async fn hybrid_search(&self, query: &SearchQuery) -> Result<Vec<SearchResult>, SearchError> {
        // Run both semantic and symbol searches in parallel
        let (semantic_results, symbol_results) = tokio::try_join!(
            self.semantic_search(query),
            self.symbol_search(query)
        )?;

        // Combine and deduplicate results
        let mut combined_results = semantic_results;
        combined_results.extend(symbol_results);

        // Remove duplicates (same file and line range)
        combined_results.sort_by(|a, b| a.file_path.cmp(&b.file_path)
            .then(a.start_line.cmp(&b.start_line)));
        combined_results.dedup_by(|a, b| a.file_path == b.file_path
            && a.start_line == b.start_line
            && a.end_line == b.end_line);

        // Apply hybrid scoring
        for result in &mut combined_results {
            result.score = self.calculate_hybrid_score(result);
            result.result_type = SearchResultType::HybridMatch(result.score);
        }

        // Sort and limit
        combined_results.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap());
        combined_results.truncate(query.limit);

        Ok(combined_results)
    }

    fn calculate_hybrid_score(&self, result: &SearchResult) -> f32 {
        match &result.result_type {
            SearchResultType::SemanticMatch(similarity) => {
                similarity * self.config.semantic_weight
            }
            SearchResultType::ExactSymbolMatch => {
                1.0 * self.config.symbol_weight
            }
            SearchResultType::FuzzySymbolMatch(similarity) => {
                similarity * self.config.symbol_weight
            }
            SearchResultType::HybridMatch(score) => *score,
        }
    }
}

fn cosine_similarity(a: &[f32], b: &[f32]) -> f32 {
    let dot_product: f32 = a.iter().zip(b.iter()).map(|(x, y)| x * y).sum();
    let magnitude_a: f32 = a.iter().map(|x| x * x).sum::<f32>().sqrt();
    let magnitude_b: f32 = b.iter().map(|x| x * x).sum::<f32>().sqrt();

    if magnitude_a == 0.0 || magnitude_b == 0.0 {
        0.0
    } else {
        dot_product / (magnitude_a * magnitude_b)
    }
}
```

**Testing Requirements**:
- Unit tests for each search type
- Integration tests for hybrid search
- Performance tests for query response times
- Relevance scoring validation tests

#### Task 1.7: Integration and Dogfooding
**Files to Create/Modify**:
- `src/lib.rs` - Public library interface
- `tests/integration_tests.rs` - End-to-end integration tests
- `tests/dogfooding_tests.rs` - Tool searching its own codebase

**Implementation Steps**:
1. Integrate all components into working system
2. Implement end-to-end workflows (index -> search)
3. Add comprehensive error handling and recovery
4. Create integration tests for complete workflows
5. Test tool searching its own source code

**Integration Test Example**:
```rust
// tests/integration_tests.rs
use codesearch::*;
use std::path::PathBuf;
use tempfile::TempDir;

#[tokio::test]
async fn test_full_workflow() -> Result<(), Box<dyn std::error::Error>> {
    // Create temporary directory with test code
    let temp_dir = TempDir::new()?;
    create_test_code_files(temp_dir.path())?;

    // Initialize search engine
    let storage = Storage::new(temp_dir.path().join(".codesearch"))?;
    let model_manager = ModelManager::new(ModelConfig::default())?;
    let search_engine = SearchEngine::new(Arc::new(storage), Arc::new(model_manager));

    // Index the directory
    let indexer = Indexer::new(temp_dir.path(), search_engine.clone());
    indexer.index_directory().await?;

    // Search for code
    let query = SearchQuery {
        text: "function that calculates fibonacci".to_string(),
        query_type: QueryType::Hybrid,
        filters: SearchFilters::default(),
        limit: 10,
    };

    let results = search_engine.search(&query).await?;

    // Verify results
    assert!(!results.is_empty(), "Should find search results");
    assert!(results[0].score > 0.5, "First result should have good relevance");

    // Verify result contains expected content
    assert!(results[0].code_snippet.contains("fibonacci"),
           "Result should contain fibonacci function");

    Ok(())
}

#[tokio::test]
async fn test_dogfooding() -> Result<(), Box<dyn std::error::Error>> {
    // Use the tool's own source code
    let project_root = PathBuf::from(env!("CARGO_MANIFEST_DIR"));

    let storage = Storage::new(project_root.join(".codesearch_test"))?;
    let model_manager = ModelManager::new(ModelConfig::default())?;
    let search_engine = SearchEngine::new(Arc::new(storage), Arc::new(model_manager));

    // Index the tool's own code
    let indexer = Indexer::new(&project_root, search_engine.clone());
    indexer.index_directory().await?;

    // Search for tool's own functions
    let query = SearchQuery {
        text: "search engine implementation".to_string(),
        query_type: QueryType::Hybrid,
        filters: SearchFilters {
            file_types: vec!["rs".to_string()],
            ..Default::default()
        },
        limit: 5,
    };

    let results = search_engine.search(&query).await?;

    // Should find the search engine module
    assert!(!results.is_empty(), "Should find search engine code");

    let search_engine_found = results.iter()
        .any(|r| r.file_path.contains("search") && r.code_snippet.contains("SearchEngine"));
    assert!(search_engine_found, "Should find SearchEngine implementation");

    Ok(())
}

fn create_test_code_files(dir: &Path) -> Result<(), std::io::Error> {
    use std::fs;

    // Create test Python file
    fs::write(dir.join("math.py"), r#"
def fibonacci(n):
    """Calculate the nth Fibonacci number."""
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

def factorial(n):
    """Calculate factorial of n."""
    if n <= 1:
        return 1
    return n * factorial(n-1)
"#)?;

    // Create test Rust file
    fs::write(dir.join("utils.rs"), r#"
pub fn add_two(a: i32) -> i32 {
    a + 2
}

pub fn multiply(a: i32, b: i32) -> i32 {
    a * b
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_add_two() {
        assert_eq!(add_two(2), 4);
    }
}
"#)?;

    Ok(())
}
```

**Dogfooding Validation**:
1. Index the tool's own source code
2. Search for core functionality (search engine, indexing, parsing)
3. Verify semantic understanding works for code patterns
4. Test performance with tool's own codebase size
5. Validate all CLI commands work on the tool itself

**Success Criteria for Phase 1**:
- ✅ All core commands (index, search, find, status) work
- ✅ Tool can index and search its own source code
- ✅ Basic semantic search provides relevant results
- ✅ Symbol search finds functions, classes, variables
- ✅ Incremental indexing works for file changes
- ✅ Performance meets basic targets (<1s for small projects)
- ✅ Comprehensive test coverage (>80%)

---

## Phase 2: Multi-Language Support

### Goal: Extend support to additional languages and improve parsing accuracy

**Target Languages**:
- JavaScript/TypeScript
- YAML/JSON (configuration files)
- Shell scripts
- Improved Python and Rust parsing

### Tasks (Estimated: 2-3 weeks)

#### Task 2.1: JavaScript/TypeScript Support
**Files to Create/Modify**:
- `src/parsers/javascript.rs` - JavaScript parser
- `src/parsers/typescript.rs` - TypeScript parser
- `src/parsers/node_modules.rs` - Node.js dependency handling
- `tests/parsers/js_tests.rs` - JavaScript parsing tests

**Implementation Steps**:
1. Add Tree-sitter grammars for JavaScript and TypeScript
2. Implement symbol extraction for JS/TS constructs
3. Handle import/export statements and module resolution
4. Add special handling for node_modules exclusion
5. Create comprehensive test coverage

**JavaScript Parser Implementation**:
```rust
// src/parsers/javascript.rs
use tree_sitter_javascript::language;
use super::*;

pub struct JavaScriptParser;

impl LanguageParser for JavaScriptParser {
    fn language(&self) -> Language {
        language()
    }

    fn file_extensions(&self) -> Vec<&'static str> {
        vec!["js", "jsx", "mjs"]
    }

    fn extract_symbols(&self, tree: &Tree, content: &str) -> Vec<Symbol> {
        let mut symbols = Vec::new();
        let mut cursor = tree.walk();

        // Walk the AST and extract JavaScript-specific symbols
        for node in tree.root_node().children(&mut cursor) {
            match node.kind() {
                "function_declaration" => {
                    self.extract_function_symbol(&node, content, &mut symbols);
                }
                "class_declaration" => {
                    self.extract_class_symbol(&node, content, &mut symbols);
                }
                "variable_declaration" => {
                    self.extract_variable_symbols(&node, content, &mut symbols);
                }
                "import_statement" => {
                    self.extract_import_symbols(&node, content, &mut symbols);
                }
                "export_statement" => {
                    self.extract_export_symbols(&node, content, &mut symbols);
                }
                _ => {}
            }
        }

        symbols
    }
}

impl JavaScriptParser {
    fn extract_function_symbol(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol {
                name,
                kind: SymbolKind::Function,
                start_line: node.start_position().row + 1,
                end_line: node.end_position().row + 1,
                start_byte: node.start_byte(),
                end_byte: node.end_byte(),
                parent: None,
            });
        }
    }

    fn extract_class_symbol(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol {
                name,
                kind: SymbolKind::Class,
                start_line: node.start_position().row + 1,
                end_line: node.end_position().row + 1,
                start_byte: node.start_byte(),
                end_byte: node.end_byte(),
                parent: None,
            });

            // Extract methods within the class
            self.extract_class_methods(node, content, symbols);
        }
    }
}
```

#### Task 2.2: Configuration File Support (YAML/JSON)
**Files to Create/Modify**:
- `src/parsers/yaml.rs` - YAML parser
- `src/parsers/json.rs` - JSON parser
- `src/parsers/config_files.rs` - Configuration file handling
- `tests/parsers/config_tests.rs` - Config file tests

**Implementation Considerations**:
- Configuration files often have nested structures
- Keys and values are both searchable
- Need to understand common patterns (databases, API configs, etc.)
- File size limits especially important for large JSON files

#### Task 2.3: Shell Script Support
**Files to Create/Modify**:
- `src/parsers/shell.rs` - Shell script parser
- `src/parsers/bash.rs` - Bash-specific parsing
- `tests/parsers/shell_tests.rs` - Shell parsing tests

**Implementation Considerations**:
- Shell scripts often have dynamic content
- Function definitions and variable assignments
- Command invocations and pipeline definitions
- Shebang and environment handling

#### Task 2.4: Enhanced Symbol Extraction
**Files to Create/Modify**:
- `src/symbols/` - Enhanced symbol handling module
- `src/symbols/relationships.rs` - Symbol relationship mapping
- `src/symbols/resolution.rs` - Cross-file symbol resolution
- `tests/symbols/relationship_tests.rs` - Symbol relationship tests

**Implementation Steps**:
1. Improve symbol relationship detection
2. Add cross-file symbol resolution
3. Implement import/export tracking
4. Create symbol dependency graphs
5. Add inheritance and composition detection

#### Task 2.5: Language-Specific Query Patterns
**Files to Create/Modify**:
- `src/queries/` - Query pattern module
- `src/queries/language_patterns.rs` - Language-specific patterns
- `src/queries/common_patterns.rs` - Cross-language patterns
- `tests/queries/pattern_tests.rs` - Query pattern tests

**Implementation Steps**:
1. Define common query patterns for each language
2. Implement semantic understanding of language idioms
3. Add framework-specific pattern recognition
4. Create cross-language concept mapping
5. Improve query expansion and interpretation

**Testing Requirements**:
- Comprehensive parsing tests for each language
- Symbol extraction accuracy tests
- Cross-language relationship tests
- Performance tests for parsing large files

---

## Phase 3: Advanced Features

### Goal: Add advanced features and optimize performance for production use

**Advanced Features**:
- Dependency graph traversal
- Cross-file relationship mapping
- Advanced filtering and query syntax
- Performance optimizations
- Enhanced CLI features

### Tasks (Estimated: 2-3 weeks)

#### Task 3.1: Dependency Graph and Relationships
**Files to Create/Modify**:
- `src/dependencies/` - Dependency analysis module
- `src/dependencies/graph.rs` - Dependency graph construction
- `src/dependencies/traversal.rs` - Graph traversal algorithms
- `src/dependencies/analysis.rs` - Dependency analysis
- `tests/dependencies/graph_tests.rs` - Dependency graph tests

**Implementation Steps**:
1. Build dependency graphs from import/export statements
2. Implement bidirectional relationship tracking
3. Add cyclic dependency detection
4. Create graph traversal for impact analysis
5. Implement dependency-based result ranking

#### Task 3.2: Advanced Query Syntax
**Files to Create/Modify**:
- `src/query/` - Advanced query parsing
- `src/query/parser.rs` - Query language parser
- `src/query/optimizer.rs` - Query optimization
- `src/query/filters.rs` - Advanced filtering
- `tests/query/advanced_tests.rs` - Advanced query tests

**Advanced Query Examples**:
```bash
# Find all functions that use a specific dependency
codesearch search "functions using database" --uses DatabaseConnection

# Find code that hasn't been modified recently
codesearch search "authentication logic" --older-than 30d

# Find related code across files
codesearch search "User class" --include-dependencies --depth 2

# Complex boolean queries
codesearch search "(authentication OR login) AND database" --type js,ts

# Semantic search with specific constraints
codesearch search "error handling patterns" --in-functions --with-try-catch
```

#### Task 3.3: Performance Optimizations
**Files to Create/Modify**:
- `src/performance/` - Performance optimization module
- `src/performance/caching.rs` - Result caching system
- `src/performance/prefetching.rs` - Intelligent prefetching
- `src/performance/parallel.rs` - Parallel processing
- `benches/` - Performance benchmark suite

**Optimization Strategies**:
1. Implement intelligent result caching
2. Add parallel processing for large operations
3. Optimize vector search with approximate algorithms
4. Implement lazy loading for large indices
5. Add query result prefetching

#### Task 3.4: Enhanced CLI Features
**Files to Create/Modify**:
- `src/cli/advanced.rs` - Advanced CLI commands
- `src/cli/interactive.rs` - Interactive search mode
- `src/cli/export.rs` - Result export functionality
- `src/cli/analyze.rs` - Code analysis commands

**New CLI Commands**:
```bash
# Interactive search mode
codesearch interactive

# Analyze codebase structure
codesearch analyze --dependencies --complexity

# Export search results
codesearch search "API endpoints" --export json --file results.json

# Find similar code
codesearch similar --file src/auth.py --line 45

# Code review assistance
codesearch review --since main --include-tests

# Documentation generation
codesearch docs --output docs/ --format markdown
```

**Testing Requirements**:
- Performance regression tests
- Memory usage validation
- Concurrency and thread safety tests
- Large-scale integration tests

---

## Testing Strategy

### Testing Philosophy
- **TDD Approach**: Write failing tests first, then implement code
- **Comprehensive Coverage**: Unit tests for all components, integration tests for workflows
- **Performance Testing**: Continuous benchmarking to ensure performance targets
- **Cross-Platform Testing**: Validate on Windows, macOS, Linux

### Test Structure

#### Unit Tests (80%+ coverage target)
**Location**: `src/*/tests.rs` or inline `#[cfg(test)]` modules

**Focus Areas**:
- CLI argument parsing and validation
- Language parsing and symbol extraction
- Model loading and embedding generation
- Storage operations and data integrity
- Search algorithms and scoring
- Configuration management

#### Integration Tests
**Location**: `tests/` directory

**Test Categories**:
- **Workflow Tests**: End-to-end indexing and search workflows
- **Multi-Language Tests**: Cross-language functionality
- **Performance Tests**: Query response time validation
- **Error Handling Tests**: Graceful failure and recovery
- **Dogfooding Tests**: Tool searching its own codebase

#### Performance Benchmarks
**Location**: `benches/` directory

**Benchmark Categories**:
- **Indexing Performance**: Time to index various codebase sizes
- **Search Performance**: Query response time validation
- **Memory Usage**: Peak and average memory consumption
- **Storage Efficiency**: Index size relative to source code

### Test Data Strategy

#### Sample Codebases
Create representative test codebases:
```bash
test_data/
├── small_rust_project/          # ~50 files, basic Rust project
├── medium_nodejs_app/           # ~500 files, Node.js application
├── large_python_codebase/       # ~5000 files, Python project
├── mixed_language_project/      # ~1000 files, multiple languages
├── config_heavy_project/        # Many YAML/JSON configuration files
└── edge_cases/                  # Unusual code patterns and edge cases
```

#### Test Query Suite
Create comprehensive test queries:
```yaml
# tests/test_queries.yaml
basic_searches:
  - query: "user authentication"
    expected_min_results: 3
    expected_keywords: ["login", "auth", "user"]

  - query: "database connection"
    expected_min_results: 2
    expected_keywords: ["database", "connection", "db"]

symbol_searches:
  - symbol: "authenticate_user"
    type: "function"
    expected_exact_match: true

  - symbol: "DatabaseConnection"
    type: "class"
    expected_exact_match: true

language_specific:
  javascript:
    - query: "React component"
      expected_keywords: ["component", "React", "render"]

  python:
    - query: "Django model"
      expected_keywords: ["model", "Django", "class"]

  rust:
    - query: "error handling Result"
      expected_keywords: ["Result", "error", "match"]
```

### Continuous Testing

#### CI/CD Pipeline
```yaml
# .github/workflows/test.yml
name: Test Suite
on: [push, pull_request]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
        rust: [stable, beta]

    steps:
      - uses: actions/checkout@v3
      - uses: actions-rs/toolchain@v1
        with:
          toolchain: ${{ matrix.rust }}

      - name: Run unit tests
        run: cargo test --lib

      - name: Run integration tests
        run: cargo test --test '*'

      - name: Run benchmarks
        run: cargo bench

      - name: Check code coverage
        run: cargo tarpaulin --out Xml

      - name: Performance regression tests
        run: cargo test --release --performance

      - name: Cross-platform validation
        run: cargo test --all-targets --all-features
```

#### Performance Monitoring
```rust
// tests/performance_tests.rs
#[cfg(test)]
mod performance_tests {
    use std::time::Instant;

    #[tokio::test]
    async fn test_search_performance_targets() {
        let engine = setup_test_search_engine().await;

        // Test small codebase performance
        let start = Instant::now();
        let results = engine.search(&create_test_query("small search")).await.unwrap();
        let duration = start.elapsed();

        assert!(duration.as_millis() < 50,
               "Small codebase search should be <50ms, got {:?}", duration);
        assert!(!results.is_empty(), "Should return results");

        // Test medium codebase performance
        let medium_engine = setup_medium_search_engine().await;
        let start = Instant::now();
        let results = medium_engine.search(&create_test_query("medium search")).await.unwrap();
        let duration = start.elapsed();

        assert!(duration.as_millis() < 100,
               "Medium codebase search should be <100ms, got {:?}", duration);
    }
}
```

---

## Development Workflow

### Commit Strategy
Follow frequent, descriptive commits with clear messages:

```bash
# Feature implementation
git commit -m "feat(parser): implement JavaScript symbol extraction

- Add function, class, and variable extraction
- Handle import/export statements
- Add comprehensive test coverage
- Support JSX and modern ES6+ syntax

Closes #123"

# Bug fixes
git commit -m "fix(search): resolve vector storage concurrency issue

- Add proper file locking for vector operations
- Implement atomic write operations
- Add regression tests for concurrent access

Fixes #145"

# Performance improvements
git commit -m "perf(indexing): optimize large file processing

- Implement streaming file parsing
- Add memory usage monitoring
- Reduce peak memory usage by 40%
- Add performance benchmarks

Improves #167"
```

### Branch Strategy
```bash
main                    # Production-ready code
├── develop            # Integration branch
├── feature/mvp        # MVP implementation
├── feature/multi-lang # Multi-language support
├── feature/advanced   # Advanced features
├── hotfix/critical    # Critical fixes
└── release/v1.0       # Release preparation
```

### Code Quality Standards

#### Rust Code Style
```rust
// Use rustfmt for consistent formatting
cargo fmt

// Use clippy for linting
cargo clippy -- -D warnings

// Documentation comments for public APIs
/// Generates semantic embeddings for code snippets.
///
/// # Arguments
///
/// * `code_snippets` - Vector of code strings to embed
///
/// # Returns
///
/// Vector of embedding vectors, one per input snippet
///
/// # Errors
///
/// Returns `EmbeddingError` if model loading or inference fails
///
/// # Examples
///
/// ```
/// use codesearch::models::*;
///
/// let generator = EmbeddingGenerator::new()?;
/// let embeddings = generator.generate_embeddings(&["fn test() {}"])?;
/// assert_eq!(embeddings.len(), 1);
/// ```
pub async fn generate_embeddings(&self, code_snippets: &[String]) -> Result<Vec<Vec<f32>>, EmbeddingError> {
    // Implementation
}
```

#### Error Handling Pattern
```rust
// Use thiserror for custom error types
use thiserror::Error;

#[derive(Error, Debug)]
pub enum SearchError {
    #[error("Model loading failed: {0}")]
    ModelLoadError(#[from] ModelError),

    #[error("Vector storage error: {0}")]
    StorageError(#[from] StorageError),

    #[error("Query parsing failed: {message}")]
    QueryParseError { message: String },

    #[error("Search timed out after {duration_ms}ms")]
    Timeout { duration_ms: u64 },
}

// Use anyhow for application-level error handling
pub async fn search_with_fallback(query: &SearchQuery) -> Result<Vec<SearchResult>> {
    let results = search_engine.search(query).await
        .context("Primary search failed")?;

    if results.is_empty() {
        // Fallback to simpler search
        search_engine.symbol_search(query).await
            .context("Fallback search failed")
    } else {
        Ok(results)
    }
}
```

### Testing Workflow

#### Test-Driven Development Cycle
1. **Write Failing Test**: Create test that clearly defines expected behavior
2. **Run Test**: Confirm test fails with meaningful error
3. **Implement Code**: Write minimum code to make test pass
4. **Run Test**: Confirm test passes
5. **Refactor**: Improve code while keeping tests green
6. **Repeat**: Continue with next feature

#### Daily Development Routine
```bash
# Morning setup
git checkout develop
git pull origin develop
git checkout -b feature/new-feature

# Development cycle
cargo watch -x 'test'                    # Auto-run tests on changes
cargo watch -x 'check'                   # Auto-check compilation
cargo watch -x 'clippy'                  # Auto-run linter

# Before committing
cargo fmt                               # Format code
cargo clippy -- -D warnings             # Check for issues
cargo test --all                        # Run all tests
cargo test --release                    # Run release mode tests

# Commit and push
git add .
git commit -m "feat: implement new feature with tests"
git push origin feature/new-feature
```

---

## File Structure Guide

### Complete Project Structure
```
local-index/
├── Cargo.toml                 # Project dependencies and metadata
├── Cargo.lock                 # Dependency lock file (generated)
├── README.md                  # Project documentation
├── CHANGELOG.md               # Version history
├── LICENSE                    # Open source license
├── .gitignore                 # Git ignore patterns
├── .github/                   # GitHub configuration
│   ├── workflows/             # CI/CD pipeline definitions
│   │   ├── test.yml          # Test automation
│   │   ├── benchmark.yml     # Performance benchmarks
│   │   └── release.yml       # Release automation
│   ├── ISSUE_TEMPLATE/       # Issue templates
│   └── PULL_REQUEST_TEMPLATE.md # PR template
├── src/                       # Source code
│   ├── main.rs               # CLI entry point
│   ├── lib.rs                # Public library interface
│   ├── cli/                  # CLI command modules
│   │   ├── mod.rs            # CLI module definition
│   │   ├── commands.rs       # Command definitions and routing
│   │   ├── index.rs          # Index command implementation
│   │   ├── search.rs         # Search command implementation
│   │   ├── status.rs         # Status command implementation
│   │   ├── config.rs         # Config command implementation
│   │   └── output.rs         # Output formatting utilities
│   ├── index/                # Index engine module
│   │   ├── mod.rs            # Index module definition
│   │   ├── indexer.rs        # Main indexing orchestration
│   │   ├── incremental.rs    # Incremental update logic
│   │   ├── scheduler.rs      # Background indexing tasks
│   │   └── validation.rs     # Index validation and repair
│   ├── search/               # Search engine module
│   │   ├── mod.rs            # Search module definition
│   │   ├── engine.rs         # Main search orchestration
│   │   ├── semantic.rs       # Semantic similarity search
│   │   ├── symbol.rs         # Symbol-based search
│   │   ├── hybrid.rs         # Hybrid search algorithms
│   │   ├── query.rs          # Query parsing and optimization
│   │   └── scoring.rs        # Relevance scoring algorithms
│   ├── models/               # Model management module
│   │   ├── mod.rs            # Models module definition
│   │   ├── manager.rs        # Model loading and caching
│   │   ├── embeddings.rs     # Vector embedding generation
│   │   ├── registry.rs       # Model registry and metadata
│   │   ├── download.rs       # Model download management
│   │   └── validation.rs     # Model integrity verification
│   ├── storage/              # Data storage module
│   │   ├── mod.rs            # Storage module definition
│   │   ├── metadata.rs       # SQLite metadata storage
│   │   ├── vectors.rs        # Vector storage (binary format)
│   │   ├── index.rs          # Index management operations
│   │   ├── migration.rs      # Schema migration handling
│   │   └── backup.rs         # Backup and recovery
│   ├── parsers/              # Language parsing module
│   │   ├── mod.rs            # Parsers module definition
│   │   ├── registry.rs       # Parser registry and dispatch
│   │   ├── rust.rs           # Rust language parser
│   │   ├── python.rs         # Python language parser
│   │   ├── javascript.rs     # JavaScript language parser
│   │   ├── typescript.rs     # TypeScript language parser
│   │   ├── yaml.rs           # YAML configuration parser
│   │   ├── json.rs           # JSON configuration parser
│   │   ├── shell.rs          # Shell script parser
│   │   └── markdown.rs       # Markdown documentation parser
│   ├── filesystem/           # File system operations
│   │   ├── mod.rs            # Filesystem module definition
│   │   ├── scanner.rs        # File discovery and filtering
│   │   ├── watcher.rs        # File change monitoring
│   │   ├── validator.rs      # File validation and metadata
│   │   └── permissions.rs    # Permission handling
│   ├── config/               # Configuration management
│   │   ├── mod.rs            # Config module definition
│   │   ├── manager.rs        # Configuration loading and saving
│   │   ├── schema.rs         # Configuration schema validation
│   │   ├── defaults.rs       # Default configuration values
│   │   └── migration.rs      # Configuration format migration
│   ├── query/                # Advanced query processing
│   │   ├── mod.rs            # Query module definition
│   │   ├── parser.rs         # Query language parser
│   │   ├── optimizer.rs      # Query optimization
│   │   ├── filters.rs        # Advanced filtering
│   │   └── patterns.rs       # Query pattern matching
│   ├── symbols/              # Symbol relationship handling
│   │   ├── mod.rs            # Symbols module definition
│   │   ├── relationships.rs  # Symbol relationship mapping
│   │   ├── resolution.rs     # Cross-file symbol resolution
│   │   └── indexing.rs       # Symbol index construction
│   ├── dependencies/         # Dependency analysis
│   │   ├── mod.rs            # Dependencies module definition
│   │   ├── graph.rs          # Dependency graph construction
│   │   ├── traversal.rs      # Graph traversal algorithms
│   │   └── analysis.rs       # Dependency analysis
│   ├── performance/          # Performance optimization
│   │   ├── mod.rs            # Performance module definition
│   │   ├── caching.rs        # Result caching system
│   │   ├── prefetching.rs    # Intelligent prefetching
│   │   ├── parallel.rs       # Parallel processing
│   │   └── monitoring.rs     # Performance monitoring
│   ├── validation/           # Technical validation
│   │   ├── mod.rs            # Validation module definition
│   │   ├── performance.rs    # Performance validation tests
│   │   ├── platform.rs       # Cross-platform validation
│   │   └── compatibility.rs  # Compatibility validation
│   └── utils/                # Utility functions
│       ├── mod.rs            # Utils module definition
│       ├── hash.rs           # Hash and checksum utilities
│       ├── paths.rs          # Path manipulation utilities
│       ├── strings.rs        # String processing utilities
│       └── math.rs           # Mathematical utilities
├── tests/                    # Integration tests
│   ├── integration_tests.rs  # End-to-end workflow tests
│   ├── dogfooding_tests.rs   # Tool searching its own codebase
│   ├── cli_tests.rs          # CLI functionality tests
│   ├── performance_tests.rs  # Performance validation tests
│   ├── cross_platform_tests.rs # Cross-platform compatibility
│   └── test_data/            # Test data and fixtures
│       ├── small_project/    # Small test codebase
│       ├── medium_project/   # Medium test codebase
│       ├── large_project/    # Large test codebase
│       └── edge_cases/       # Edge case test files
├── benches/                  # Performance benchmarks
│   ├── indexing_benchmark.rs # Indexing performance tests
│   ├── search_benchmark.rs   # Search performance tests
│   ├── memory_benchmark.rs   # Memory usage benchmarks
│   └── storage_benchmark.rs  # Storage performance tests
├── models/                   # Embedded ML models
│   ├── codebert-small.onnx   # Lightweight embedding model
│   ├── model_manifest.json   # Model metadata and checksums
│   └── test_model.onnx       # Small model for testing
├── docs/                     # Documentation
│   ├── plans/                # Implementation plans
│   │   └── comprehensive_implementation_plan.md # This file
│   ├── architecture/         # Architecture documentation
│   ├── api/                  # API documentation
│   ├── user_guide/           # User guide and tutorials
│   └── developer_guide/      # Developer documentation
├── scripts/                  # Build and utility scripts
│   ├── build.sh             # Build script for all platforms
│   ├── test.sh              # Test runner script
│   ├── benchmark.sh         # Benchmark runner
│   └── release.sh           # Release automation script
└── tools/                    # Development tools
    ├── generate_test_data.rs # Test data generation
    ├── validate_models.rs    # Model validation tool
    └── performance_profiler.rs # Performance profiling tool
```

### Key File Responsibilities

#### Core Application Files
- **`src/main.rs`**: CLI entry point, argument parsing, command routing
- **`src/lib.rs`**: Public library API, module re-exports
- **`Cargo.toml`**: Dependencies, build configuration, metadata

#### CLI Implementation
- **`src/cli/commands.rs`**: Command structure using `clap` derive macros
- **`src/cli/*.rs`**: Individual command implementations with error handling

#### Core Engine Components
- **`src/search/engine.rs`**: Main search orchestration and query processing
- **`src/index/indexer.rs`**: File indexing and update management
- **`src/storage/`**: Data persistence with SQLite + custom binary format

#### Language Support
- **`src/parsers/*.rs`**: Tree-sitter based language parsers
- **`src/parsers/registry.rs`**: Parser registration and language detection

#### Model and AI Integration
- **`src/models/manager.rs`**: ONNX model loading, caching, and lifecycle
- **`src/models/embeddings.rs`**: Vector generation from code snippets

### Testing File Organization

#### Unit Tests
```rust
// Inline tests in each module
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_functionality() {
        // Test implementation
    }
}
```

#### Integration Tests
```rust
// tests/integration_tests.rs
use codesearch::*;
use tempfile::TempDir;

#[tokio::test]
async fn test_complete_workflow() {
    // End-to-end test implementation
}
```

#### Benchmark Tests
```rust
// benches/search_benchmark.rs
use criterion::{black_box, criterion_group, criterion_main, Criterion};

fn search_benchmark(c: &mut Criterion) {
    c.bench_function("search_1000_vectors", |b| {
        b.iter(|| {
            // Benchmark implementation
        })
    });
}
```

---

## Common Pitfalls & Solutions

### Technical Pitfalls

#### 1. Vector Search Performance Issues
**Problem**: CPU-based similarity search too slow for large indices

**Symptoms**:
- Search queries taking >500ms
- High CPU usage during searches
- Poor user experience for agents

**Solutions**:
1. **Implement Approximate Search**: Use HNSW or other ANN algorithms
2. **Vector Quantization**: Reduce memory footprint and improve cache locality
3. **Hybrid Approach**: Exact search for recent files, approximate for historical
4. **Caching Layer**: Cache common query results

```rust
// Example: Implement HNSW fallback
if vector_count > EXACT_SEARCH_THRESHOLD {
    // Use approximate search for large indices
    hnsw_index.search(query_embedding, k)
} else {
    // Use exact search for small indices
    exact_search(query_embedding, k)
}
```

#### 2. Model Loading and Memory Issues
**Problem**: ONNX models consume too much memory or fail to load

**Symptoms**:
- Out-of-memory errors on machines with <4GB RAM
- Model loading failures on some platforms
- Slow application startup

**Solutions**:
1. **Lazy Loading**: Load models only when needed
2. **Model Selection**: Offer smaller models for memory-constrained environments
3. **Memory Management**: Implement LRU cache for loaded models
4. **Graceful Degradation**: Fallback to text-only search if models fail

```rust
// Example: Memory-aware model loading
pub fn load_model_with_memory_limit(&self, model_name: &str, memory_limit_mb: usize) -> Result<Model> {
    let model_size = self.get_model_size(model_name)?;

    if model_size > memory_limit_mb {
        // Suggest smaller model
        return Err(ModelError::InsufficientMemory {
            required: model_size,
            available: memory_limit_mb,
            alternative: "codebert-small",
        });
    }

    self.load_model(model_name)
}
```

#### 3. Cross-Platform Compatibility Issues
**Problem**: WASM or ONNX runtime behavior differs across platforms

**Symptoms**:
- Tree-sitter parsing fails on specific OS versions
- ONNX models load but produce different results
- File locking issues on Windows vs Unix

**Solutions**:
1. **Platform-Specific Testing**: Comprehensive CI matrix
2. **Graceful Fallbacks**: Alternative implementations when WASM fails
3. **Comprehensive Validation**: Test each component individually
4. **Error Isolation**: Clear error messages with platform-specific guidance

```rust
// Example: Platform-specific error handling
#[cfg(target_os = "windows")]
fn handle_file_locking() -> Result<()> {
    // Windows-specific file locking
}

#[cfg(not(target_os = "windows"))]
fn handle_file_locking() -> Result<()> {
    // Unix file locking (flock)
}
```

### Development Pitfalls

#### 1. Test Coverage Gaps
**Problem**: Critical functionality not properly tested

**Symptoms**:
- Production bugs in supposedly tested areas
- Low confidence in refactoring
- Test suite passes but software has bugs

**Solutions**:
1. **TDD Enforcement**: Write tests before implementation
2. **Coverage Requirements**: Minimum 80% line coverage
3. **Integration Testing**: Test complete workflows, not just units
4. **Property-Based Testing**: Use proptest for edge cases

```rust
// Example: Property-based test
use proptest::prelude::*;

proptest! {
    #[test]
    fn test_vector_similarity_properties(
        vec_a in prop::collection::vec(-1.0..1.0, 10..100),
        vec_b in prop::collection::vec(-1.0..1.0, 10..100)
    ) {
        let similarity = cosine_similarity(&vec_a, &vec_b);
        prop_assert!((similarity >= -1.0) && (similarity <= 1.0));
    }
}
```

#### 2. Performance Regression
**Problem**: New features slow down the system

**Symptoms**:
- Search times gradually increase
- Memory usage grows over time
- Indexing becomes slower with each release

**Solutions**:
1. **Continuous Benchmarking**: Automated performance testing
2. **Performance Budgets**: Set and enforce performance targets
3. **Regression Detection**: Alert on performance degradation
4. **Regular Profiling**: Use tools to identify bottlenecks

```rust
// Example: Performance budget enforcement
#[cfg(test)]
mod performance_tests {
    #[tokio::test]
    async fn test_search_performance_budget() {
        let start = Instant::now();
        let results = search_engine.search(&test_query).await.unwrap();
        let duration = start.elapsed();

        // Enforce performance budget
        assert!(duration.as_millis() < 100,
               "Search exceeded 100ms budget: {:?}", duration);
    }
}
```

#### 3. Memory Leaks and Resource Management
**Problem**: Memory usage grows without bound

**Symptoms**:
- Process memory usage increases over time
- System becomes slower with extended use
- Out-of-memory crashes on long-running operations

**Solutions**:
1. **RAII Patterns**: Ensure proper resource cleanup
2. **Memory Profiling**: Regular use of memory profiling tools
3. **Resource Limits**: Implement memory usage limits
4. **Explicit Cleanup**: Manual resource management where needed

```rust
// Example: RAII for model management
pub struct ModelGuard {
    manager: ModelManager,
    model_name: String,
}

impl Drop for ModelGuard {
    fn drop(&mut self) {
        // Automatically unload model when guard goes out of scope
        if let Err(e) = self.manager.unload_model(&self.model_name) {
            eprintln!("Failed to unload model {}: {}", self.model_name, e);
        }
    }
}
```

### User Experience Pitfalls

#### 1. Poor Error Messages
**Problem**: Users don't understand what went wrong or how to fix it

**Symptoms**:
- Support requests for common issues
- Users give up after encountering errors
- Frustrating user experience

**Solutions**:
1. **Structured Error Types**: Use thiserror for clear error hierarchies
2. **Contextual Information**: Include relevant context in errors
3. **Actionable Suggestions**: Tell users how to fix problems
4. **Error Code Documentation**: Link errors to documentation

```rust
// Example: User-friendly error
use thiserror::Error;

#[derive(Error, Debug)]
pub enum IndexingError {
    #[error("Cannot index file '{path}': {reason}")]
    FileError {
        path: String,
        reason: String,
        #[error("suggestion")]
        suggestion: String,
    },

    #[error("Model '{model_name}' not found")]
    ModelNotFound {
        model_name: String,
        #[error("suggestion")]
        suggestion: String,
    },
}

impl IndexingError {
    pub fn with_suggestion(self, suggestion: impl Into<String>) -> Self {
        match self {
            IndexingError::FileError { path, reason, .. } => {
                IndexingError::FileError {
                    path,
                    reason,
                    suggestion: suggestion.into(),
                }
            }
            IndexingError::ModelNotFound { model_name, .. } => {
                IndexingError::ModelNotFound {
                    model_name,
                    suggestion: suggestion.into(),
                }
            }
        }
    }
}
```

#### 2. Inconsistent CLI Interface
**Problem**: Commands behave differently from user expectations

**Symptoms**:
- Users need to constantly check help documentation
- Commands don't follow consistent patterns
- Agent discovery is difficult

**Solutions**:
1. **Git-Style Consistency**: Follow established CLI patterns
2. **Comprehensive Help**: Detailed help for all commands
3. **Consistent Output**: Uniform output formats across commands
4. **Discovery Testing**: Test that agents can discover capabilities

```rust
// Example: Consistent command structure
#[derive(Parser)]
#[command(after_help = "
Examples:
  codesearch index .                    # Index current directory
  codesearch search 'auth' --type py   # Search Python files
  codesearch find 'main' --exact       # Find exact symbol matches

For more help on a specific command:
  codesearch <command> --help
")]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,

    /// Output results in JSON format (useful for scripts and agents)
    #[arg(short, long, global = true)]
    pub json: bool,

    /// Quiet mode - suppress non-error output
    #[arg(short, long, global = true)]
    pub quiet: bool,

    /// Verbose output (-v for normal verbose, -vv for very verbose)
    #[arg(short, long, global = true, action = clap::ArgAction::Count)]
    pub verbose: u8,
}
```

---

## Success Metrics and Validation

### Phase Completion Criteria

#### Phase 0 Completion
- [ ] Vector search performance validated against targets
- [ ] Cross-platform compatibility confirmed
- [ ] Architecture decisions documented
- [ ] Technical risks identified and mitigated
- [ ] Go/No-Go decision made for full implementation

#### Phase 1 Completion (MVP)
- [ ] CLI supports all core commands (index, search, find, status)
- [ ] Tool can index and search its own source code
- [ ] Semantic search returns relevant results for test queries
- [ ] Symbol search finds functions, classes, variables accurately
- [ ] Incremental indexing works for file changes
- [ ] Performance targets met (<100ms search, <30s indexing)
- [ ] Test coverage >80% with comprehensive integration tests
- [ ] Documentation covers all user-facing functionality

#### Phase 2 Completion
- [ ] JavaScript/TypeScript parsing and symbol extraction
- [ ] YAML/JSON configuration file support
- [ ] Shell script parsing capability
- [ ] Enhanced symbol relationships and cross-file resolution
- [ ] Language-specific query patterns implemented
- [ ] Multi-language search quality >85% relevance
- [ ] Performance maintained with additional languages

#### Phase 3 Completion
- [ ] Dependency graph traversal implemented
- [ ] Cross-file relationship mapping working
- [ ] Advanced query syntax supported
- [ ] Performance optimizations in place
- [ ] Enhanced CLI features (interactive mode, export, analysis)
- [ ] Production-ready with comprehensive error handling
- [ ] Full documentation and user guides

### Quality Gates

#### Performance Requirements
- **Small codebases** (<100 files): Search <50ms, Index <10s
- **Medium codebases** (100-1000 files): Search <100ms, Index <30s
- **Large codebases** (1000-10000 files): Search <200ms, Index <2min
- **Memory usage**: <1GB for typical workloads
- **Storage efficiency**: Index <10% of source code size

#### Quality Requirements
- **Test coverage**: >80% line coverage, >90% branch coverage for critical paths
- **Documentation**: All public APIs documented, user guide complete
- **Error handling**: Graceful degradation for all failure modes
- **Cross-platform**: Works on Windows, macOS, Linux (x86_64)

#### Usability Requirements
- **Agent discovery**: All capabilities discoverable via help commands
- **Setup time**: New user indexing first project <5 minutes
- **Learnability**: Common tasks accomplishable with <3 help lookups
- **Reliability**: <1% crash rate in normal usage

### Validation Methods

#### Automated Testing
```bash
# Continuous performance monitoring
cargo bench --all

# Cross-platform validation
cargo test --all-targets --all-features

# Memory leak detection
valgrind --tool=memcheck cargo test

# Coverage reporting
cargo tarpaulin --out Html --output-dir coverage/
```

#### Manual Validation Checklists
```markdown
## Dogfooding Validation

- [ ] Tool can search its own source code effectively
- [ ] Performance is acceptable on the tool's codebase (~200 files)
- [ ] Semantic search finds relevant code patterns
- [ ] Symbol search correctly identifies functions and classes
- [ ] Incremental indexing works when modifying source files
- [ ] Error handling is graceful for malformed files
- [ ] CLI commands are intuitive and well-documented
```

#### User Acceptance Testing
```markdown
## User Scenarios

**Scenario 1: New Project Discovery**
1. User clones new repository
2. Runs `codesearch index .`
3. Searches for "authentication logic"
4. Finds relevant code and understands architecture
5. Success: Time <5 minutes, relevant results in top 3

**Scenario 2: Symbol Finding**
1. User needs to find function definition
2. Runs `codesearch find "function_name"`
3. Gets exact match with file location and context
4. Success: Function found, correct location provided

**Scenario 3: Cross-Language Understanding**
1. User searches for "database configuration"
2. Gets results from YAML, Python, JavaScript files
3. Results show relationships between config files and code
4. Success: Comprehensive view of database setup
```

---

## Conclusion

This comprehensive implementation plan provides a complete roadmap for building the agent-first codebase search tool. The plan is designed to:

1. **Minimize Risk**: Validate technical assumptions early before major investment
2. **Ensure Quality**: Comprehensive testing and validation at every phase
3. **Maintain Velocity**: Clear priorities and frequent delivery of value
4. **Enable Success**: Dogfooding from day one ensures the tool solves real problems

The implementation follows best practices:
- **Test-Driven Development**: Write failing tests first, implement code to pass them
- **Incremental Delivery**: Each phase delivers working software
- **Continuous Integration**: Automated testing and validation
- **User-Centered Design**: Agent-first design with dogfooding validation

### Key Success Factors
- **Technical Validation**: Confirm vector search and cross-platform compatibility early
- **MVP Focus**: Deliver core functionality quickly, then expand
- **Quality First**: Comprehensive testing and documentation throughout
- **Performance Awareness**: Continuous monitoring and optimization
- **User Feedback**: Regular dogfooding and validation with real usage

### Next Steps
1. **Review and Approve**: Review this plan with stakeholders
2. **Environment Setup**: Set up development environment and tools
3. **Phase 0 Execution**: Begin technical validation (1-2 weeks)
4. **Architecture Review**: Review validation results and confirm architecture
5. **Phase 1 Implementation**: Begin MVP development (3-4 weeks)

With this plan, a skilled engineer can successfully implement the agent-first codebase search tool, delivering value to both AI agents and human developers while maintaining high quality standards and managing technical risks effectively.