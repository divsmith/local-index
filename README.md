# CodeSearch - Agent-First Semantic Code Search Tool

**Vision**: A locally-installed CLI tool that provides semantic and symbol-aware code search capabilities for AI coding agents, solving the token inefficiency and noise problems of grep-only retrieval while avoiding MCP complexity.

## Status

🚧 **Currently in Phase 0: Foundation & Validation**

This is the initial project setup phase. The tool is not yet functional but the basic structure is in place.

## Project Goals

- **Agent Discovery**: Tool capabilities understood within 2-3 help commands
- **Performance**: Search results < 100ms for indexed codebases
- **Relevance**: Top 3 results contain relevant information > 80% of the time
- **Token Efficiency**: Reduce context needed vs grep by 40%+
- **Developer Experience**: Setup within 5 minutes, incremental indexing < 10 seconds

## Architecture Overview

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

## Key Technologies

- **Language**: Rust (performance, single-binary deployment)
- **AST Parsing**: Tree-sitter with WASM runtime (multi-language support)
- **Vector Embeddings**: ONNX runtime with CodeBERT models (semantic search)
- **Metadata Storage**: SQLite (file metadata, symbols, configuration)
- **Vector Storage**: Custom binary format with memory mapping (performance)

## Installation (Future)

```bash
cargo install codesearch
```

## Usage (Planned)

```bash
# Index a directory
codesearch index .

# Search for code semantically
codesearch search "user authentication logic"

# Find specific symbols
codesearch find "authenticate_user" --exact

# Check indexing status
codesearch status
```

## Development

### Prerequisites

- Rust 1.70+ (stable toolchain)
- Git

### Setup

```bash
# Clone repository
git clone <repository-url>
cd local-index

# Install development tools
cargo install cargo-watch cargo-nextest

# Run tests
cargo test

# Run with auto-reloading
cargo watch -x 'run'
```

### Project Structure

```
src/
├── main.rs               # CLI entry point
├── lib.rs                # Library interface
├── cli/                  # CLI command modules
├── index/                # Index engine
├── search/               # Search engine
├── models/               # Model management
├── storage/              # Data storage
├── parsers/              # Language parsers
├── filesystem/           # File system operations
├── config/               # Configuration management
├── validation/           # Technical validation
└── utils/                # Utility functions
```

## Implementation Plan

This project follows a comprehensive implementation plan:

- **Phase 0 (1-2 weeks)**: Foundation & Technical Validation
- **Phase 1 (3-4 weeks)**: MVP - Basic semantic search for 2-3 languages
- **Phase 2 (2-3 weeks)**: Multi-language support and improved parsing
- **Phase 3 (2-3 weeks)**: Advanced features and optimization

See [docs/plans/comprehensive_implementation_plan.md](docs/plans/comprehensive_implementation_plan.md) for details.

## Testing Strategy

- **Test-Driven Development**: Write failing tests first, then implement code
- **Comprehensive Coverage**: Target >80% line coverage
- **Performance Testing**: Continuous benchmarking
- **Cross-Platform Testing**: Windows, macOS, Linux validation

## Contributing

This project is currently in early development. Please see the implementation plan for details on how to contribute.

## License

MIT License - see LICENSE file for details.

---

**Note**: This is an agent-first tool designed specifically for AI coding agents. The CLI interface and help system are designed to be easily discoverable and navigable by AI assistants.