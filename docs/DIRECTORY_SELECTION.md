# Directory Selection Feature Documentation

## Overview

The directory selection feature extends the code-search CLI tool to allow users to specify target directories for both indexing and searching operations using the `--dir <directory>` flag. This feature provides enhanced flexibility while maintaining backward compatibility.

## Features

### Core Functionality
- **Directory-specific indexing**: Index any directory on the filesystem
- **Directory-specific searching**: Search within indexed directories
- **Automatic index location management**: Indexes stored in `.clindex/` subdirectories
- **Backward compatibility**: Existing workflows continue to work unchanged
- **Comprehensive validation**: Security checks, permissions, and size limits

### Advanced Features
- **Performance optimization**: Optimized scanning for large directories
- **File locking**: Prevents concurrent access conflicts
- **Index migration**: Seamless upgrade from legacy index formats
- **Cross-platform support**: Works on Windows, macOS, and Linux

## Usage

### Basic Commands

#### Index a specific directory
```bash
code-search index --dir /path/to/my-project
```

#### Search a specific directory
```bash
code-search search "function.*error" --dir /path/to/my-project
```

#### Search with additional options
```bash
code-search search "TODO" --dir ~/project --max-results 20 --format json
```

### Directory Path Formats

The `--dir` flag supports various path formats:

```bash
# Absolute paths
code-search index --dir /home/user/my-project
code-search index --dir C:\Users\user\my-project

# Relative paths
code-search index --dir ../sibling-project
code-search index --dir ./subdirectory

# Home directory expansion
code-search index --dir ~/my-project
code-search index --dir ~user/project

# Current directory (default behavior)
code-search index --dir .
code-search index  # Equivalent to --dir .
```

## Index Storage

### Index Location Structure

When using the `--dir` flag, indexes are stored in a `.clindex/` subdirectory within the target directory:

```
/path/to/my-project/
├── .clindex/
│   ├── metadata.json       # Index metadata and configuration
│   ├── data.index          # Main index data
│   ├── lock                # File lock for concurrent access
│   └── [additional files]  # Other index-related files
├── src/
├── README.md
└── ...
```

### Legacy Index Compatibility

The tool automatically detects and migrates legacy index files:

- **Legacy format**: `.code-search-index` (in directory root)
- **New format**: `.clindex/` subdirectory with structured storage

Migration is automatic and includes:
- Backup of legacy files
- Migration metadata tracking
- Rollback capability

## Performance Considerations

### Large Directory Optimization

The tool automatically optimizes performance for large directories:

- **Parallel scanning**: Uses multiple CPU cores for directory traversal
- **Batch processing**: Processes files in configurable batches
- **Memory management**: Monitors and controls memory usage
- **Smart filtering**: Excludes common non-source directories

### Automatic Optimizations

| Directory Size | Optimization Strategy |
|----------------|----------------------|
| < 1,000 files  | Sequential scan |
| 1,000-10,000 files | Small batch processing |
| 10,000-100,000 files | Parallel processing |
| > 100,000 files | High-performance parallel with large batches |

### Performance Tuning

Performance can be influenced by:

- **CPU cores**: More cores enable better parallelization
- **Available memory**: Larger batch sizes for memory-rich systems
- **Disk I/O**: SSD vs HDD affects scan performance
- **Directory structure**: Deep nesting vs flat structures

## Security and Validation

### Directory Validation

All directories undergo comprehensive validation:

1. **Path Resolution**: Converts to absolute paths, expands `~`
2. **Existence Check**: Verifies directory exists and is accessible
3. **Permission Verification**: Confirms read, write, and execute permissions
4. **Security Scanning**: Detects path traversal attempts
5. **Size Assessment**: Checks against configured limits

### Security Features

- **Path traversal protection**: Prevents `../` attacks
- **Permission validation**: Ensures proper access rights
- **Size limits**: Prevents processing of overly large directories
- **Safe file operations**: Atomic writes and proper error handling

### Configurable Limits

Default limits can be customized:

```go
// Default limits
MaxDirectorySize: 1GB
MaxFileCount: 10,000 files
MaxFileSize: 100MB per file
```

## Error Handling

### Error Types

The tool provides specific error types for different scenarios:

| Error Type | Exit Code | Description |
|------------|-----------|-------------|
| Success | 0 | Operation completed successfully |
| General Error | 1 | Generic error during operation |
| Invalid Arguments | 2 | Invalid command-line arguments |
| Index Not Found | 3 | No index exists for specified directory |

### Common Error Scenarios

#### Directory does not exist
```bash
$ code-search index --dir /non/existent/path
Error: directory validation failed: Directory '/non/existent/path' does not exist
```

#### Permission denied
```bash
$ code-search index --dir /root
Error: directory validation failed: Permission denied accessing '/root' for read
```

#### Index not found
```bash
$ code-search search "test" --dir /unindexed/project
Error: search failed: No index found in directory '/unindexed/project'
```

#### Path traversal attempt
```bash
$ code-search index --dir ../../etc
Error: directory validation failed: Path traversal detected: '../../etc'
```

## File Locking

### Lock Types

- **Exclusive Lock**: Used during indexing (write operations)
- **Shared Lock**: Used during searching (read operations)

### Lock Behavior

| Operation | Lock Type | Concurrent Behavior |
|-----------|-----------|---------------------|
| Indexing | Exclusive | Blocks other indexing and searching |
| Searching | Shared | Allows multiple concurrent searches |
| Index + Search | Conflict | Search waits for indexing to complete |

### Lock Files

Lock files are stored in the `.clindex/` directory:
```
.clindex/lock  # Lock file for the directory
```

## Migration Guide

### From Legacy Index Format

If you have existing `.code-search-index` files:

1. **Automatic Migration**: The tool detects and migrates automatically
2. **Manual Migration**: Use migration utilities if needed
3. **Rollback**: Migration can be rolled back if issues occur

### Migration Process

```bash
# Automatic migration (occurs during first use)
code-search index --dir /path/to/project

# The tool will:
# 1. Detect legacy .code-search-index file
# 2. Create .clindex/ directory structure
# 3. Migrate index data
# 4. Update metadata
# 5. Clean up legacy files (if successful)
```

## Advanced Usage

### Custom Configuration

While the tool works out-of-the-box, advanced users can customize behavior:

#### Environment Variables
```bash
# Set custom memory limit (MB)
export CODE_SEARCH_MEMORY_LIMIT=1024

# Set custom worker count
export CODE_SEARCH_WORKERS=8

# Enable verbose logging
export CODE_SEARCH_DEBUG=1
```

#### Configuration Files
```json
{
  "performance": {
    "max_workers": 8,
    "batch_size": 2000,
    "memory_limit_mb": 1024
  },
  "limits": {
    "max_directory_size_mb": 2048,
    "max_file_count": 50000,
    "max_file_size_mb": 200
  },
  "exclude_dirs": [
    ".git",
    "node_modules",
    ".clindex",
    "vendor",
    "build"
  ]
}
```

### Batch Operations

For multiple directories:

```bash
# Index multiple directories
for dir in project1 project2 project3; do
    code-search index --dir "$dir"
done

# Search multiple directories
for dir in project1 project2 project3; do
    echo "Searching in $dir:"
    code-search search "TODO" --dir "$dir"
    echo "---"
done
```

## Troubleshooting

### Common Issues

#### Slow indexing
- **Cause**: Large directory with many files
- **Solution**: Use `--verbose` flag to monitor progress, consider excluding large directories

#### Memory usage
- **Cause**: Processing very large directories
- **Solution**: System automatically manages memory, consider limiting directory size

#### Permission errors
- **Cause**: Insufficient permissions on target directory
- **Solution**: Check directory permissions, ensure read/write access

#### Index corruption
- **Cause**: System crash or interruption during indexing
- **Solution**: Delete `.clindex/` directory and re-index

### Debug Information

Enable verbose output for debugging:

```bash
# Verbose indexing
code-search index --dir /path/to/project --verbose

# Verbose searching
code-search search "query" --dir /path/to/project --verbose
```

## API Reference

### Core Models

#### DirectoryConfig
```go
type DirectoryConfig struct {
    Path         string            `json:"path"`
    OriginalPath string            `json:"original_path"`
    IsDefault    bool              `json:"is_default"`
    Permissions  DirectoryPerms    `json:"permissions"`
    Limits       DirectoryLimits   `json:"limits"`
    Metadata     DirectoryMetadata `json:"metadata"`
}
```

#### IndexLocation
```go
type IndexLocation struct {
    BaseDirectory string `json:"base_directory"`
    IndexDir      string `json:"index_dir"`
    MetadataFile  string `json:"metadata_file"`
    DataFile      string `json:"data_file"`
    LockFile      string `json:"lock_file"`
}
```

### Validation Services

#### DirectoryValidator
```go
validator := lib.NewDirectoryValidator()
config, err := validator.ValidateDirectory("/path/to/directory")
```

#### FileUtilities
```go
fileUtils := lib.NewFileUtilities()
indexPath, err := fileUtils.GetIndexLocation("/path/to/directory")
```

### Performance Optimization

#### PerformanceOptimizer
```go
optimizer := lib.NewPerformanceOptimizer()
options := lib.DefaultScanOptions()
result, err := optimizer.FastDirectoryScan("/path/to/directory", options)
```

## Best Practices

### Performance
- Index directories when they're relatively stable
- Use appropriate file type filters to reduce index size
- Consider excluding large non-source directories
- Monitor memory usage for very large directories

### Security
- Avoid indexing directories with sensitive information
- Use absolute paths for predictable behavior
- Ensure proper file permissions on target directories
- Regular cleanup of unused indexes

### Organization
- Use descriptive directory names
- Separate indexes for different project types
- Consider project-specific vs. shared indexes
- Document index location for team collaboration

## Integration Examples

### CI/CD Pipeline

```yaml
# GitHub Actions example
- name: Index source code
  run: |
    code-search index --dir ${{ github.workspace }} --verbose

- name: Search for TODOs
  run: |
    code-search search "TODO" --dir ${{ github.workspace }} --format json > todos.json
```

### Development Workflow

```bash
# 1. Index current project
code-search index --dir .

# 2. Search within project
code-search search "bug" --dir . --with-context

# 3. Index related project
code-search index --dir ../shared-library

# 4. Search across projects
code-search search "API.*endpoint" --dir . --format json
code-search search "API.*endpoint" --dir ../shared-library --format json
```

## Migration from Previous Versions

### Breaking Changes
- None. The feature is fully backward compatible.

### New Features
- `--dir` flag for both `index` and `search` commands
- `.clindex/` directory structure for new indexes
- Performance optimizations for large directories
- Enhanced error reporting and validation

### Upgrade Path
1. Existing workflows continue to work unchanged
2. New indexes automatically use `.clindex/` structure
3. Legacy indexes are migrated automatically on first use
4. No configuration changes required

## Contributing

### Development Setup
```bash
# Clone repository
git clone <repository-url>
cd local-index

# Run tests
go test ./tests/unit/...
go test ./tests/integration/...
go test ./tests/contract/...

# Build
go build -o bin/code-search ./src/main.go
```

### Adding New Features
1. Follow TDD approach: write tests first
2. Update documentation
3. Ensure backward compatibility
4. Add performance benchmarks for large directories
5. Test cross-platform compatibility

### Performance Testing
```bash
# Create large test directory
./scripts/create-test-directory.sh --files 100000 --size 1GB

# Benchmark indexing
time code-search index --dir ./test-large-dir --verbose

# Benchmark searching
time code-search search "test" --dir ./test-large-dir --max-results 100
```

## Support and Feedback

For issues, questions, or contributions:
- GitHub Issues: Report bugs and request features
- Documentation: Check this guide and inline help
- Examples: See test cases and integration tests