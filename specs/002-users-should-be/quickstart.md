# Quickstart Guide: Optional Directory Selection

## Overview
This feature allows users to optionally select directories for indexing and searching, with index files stored alongside the content being indexed.

## Prerequisites
- `code-search` CLI tool installed
- Read/write permissions for target directories

## Basic Usage

### 1. Index a Directory
Index the current directory (default behavior):
```bash
code-search index
```

Index a specific directory:
```bash
code-search index --dir /path/to/my-project
```

Index a relative directory:
```bash
code-search index --dir ../sibling-project
```

### 2. Search an Indexed Directory
Search in the current directory:
```bash
code-search search "function main"
```

Search in a specific indexed directory:
```bash
code-search search --dir /path/to/my-project "TODO"
```

Search with JSON output:
```bash
code-search search --format json --dir /path/to/my-project "import.*react"
```

## Workflow Examples

### Project Setup Workflow
1. Navigate to your project directory
2. Index the project:
   ```bash
   cd /path/to/my-project
   code-search index
   ```
3. Search within your project:
   ```bash
   code-search search "class.*Controller"
   ```

### Multi-Project Workflow
1. Index multiple projects:
   ```bash
   code-search index --dir ~/project-a
   code-search index --dir ~/project-b
   code-search index --dir ~/shared/project-c
   ```
2. Search across specific projects:
   ```bash
   code-search search --dir ~/project-a "authentication"
   code-search search --dir ~/project-b "database"
   ```

### Code Review Workflow
1. Index the codebase to review:
   ```bash
   code-search index --dir ~/code-to-review
   ```
2. Search for patterns of interest:
   ```bash
   code-search search --dir ~/code-to-review "TODO|FIXME|HACK"
   ```

## Advanced Usage

### Customizing Index Limits
Set custom limits via environment variables:
```bash
export CODESEARCH_MAX_SIZE=2G
export CODESEARCH_MAX_FILES=50000
code-search index --dir /large/project
```

### Force Reindexing
Force reindexing even if index exists:
```bash
code-search index --dir /path/to/project --force
```

### Search Options
Limit search results:
```bash
code-search search --max-results 10 "function.*test"
```

## Directory Management

### Index File Location
Index files are stored in `.clindex/` subdirectory within the indexed directory:
```
/path/to/my-project/
├── src/
├── .clindex/
│   ├── metadata.json
│   ├── data.index
│   └── lock
└── README.md
```

### Checking Index Status
To check if a directory is indexed, look for the `.clindex` directory:
```bash
ls -la /path/to/project/.clindex
```

### Removing Index
Delete the index directory to remove the index:
```bash
rm -rf /path/to/project/.clindex
```

## Troubleshooting

### Common Issues

**Permission Denied Error**
```bash
Error: Permission denied accessing '/path/to/directory'
```
Solution: Ensure you have read permissions for the directory.

**Directory Too Large Error**
```bash
Error: Directory '/path/to/directory' (2GB) exceeds limit (1GB)
```
Solution: Increase the size limit or exclude large files/directories.

**No Index Found Error**
```bash
Error: No index found in directory '/path/to/directory'
```
Solution: Run `code-search index --dir /path/to/directory` first.

### Getting Help
Show command help:
```bash
code-search index --help
code-search search --help
```

## Migration from Previous Version

### Backward Compatibility
Existing workflows continue to work unchanged:
```bash
# Old command (still works)
code-search index
code-search search "query"

# New equivalent commands
code-search index --dir .
code-search search --dir . "query"
```

### Updating Scripts
Update automated scripts to use explicit directory selection:
```bash
# Before
code-search index

# After (recommended)
code-search index --dir /path/to/project
```

## Best Practices

1. **Index at Project Root**: Index the entire project directory for comprehensive search
2. **Use Absolute Paths**: Use absolute paths in scripts for reliability
3. **Regular Reindexing**: Reindex after significant code changes
4. **Monitor Index Size**: Use `--verbose` flag to monitor indexing progress
5. **Respect Limits**: Configure appropriate limits for your project sizes

## Next Steps
- Read the [CLI Interfaces Contract](contracts/cli-interfaces.yaml) for detailed API information
- Review the [Data Model](data-model.md) for technical implementation details
- Run tests to verify functionality: `make test`