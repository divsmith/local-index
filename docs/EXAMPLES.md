# Code Search CLI - Examples and Usage Guide

## Quick Start

### Basic Usage

```bash
# Index current directory
code-search index

# Search current directory
code-search search "function.*error"

# Get help
code-search --help
code-search index --help
code-search search --help
```

## Directory Selection Examples

### Indexing Specific Directories

```bash
# Index a specific project
code-search index --dir /home/user/my-project

# Index using relative path
code-search index --dir ../sibling-project

# Index using home directory shortcut
code-search index --dir ~/projects/web-app

# Index current directory explicitly
code-search index --dir .
```

### Searching Specific Directories

```bash
# Search in a specific directory
code-search search "TODO" --dir /home/user/my-project

# Search with multiple options
code-search search "bug.*fix" --dir ~/project --max-results 10 --format json

# Search with context
code-search search "API.*endpoint" --dir ./src --with-context

# Search using different search types
code-search search "user.*auth" --dir ./auth --semantic
code-search search "class.*Controller" --dir ./controllers --exact
code-search search "functon.*erorr" --dir ./src --fuzzy
```

## Real-World Scenarios

### Scenario 1: Multi-Project Development

You're working on multiple related projects and want to search across them efficiently.

```bash
# Index all your projects
code-search index --dir ~/projects/frontend
code-search index --dir ~/projects/backend
code-search index --dir ~/projects/shared

# Search for API-related code across all projects
echo "=== Frontend ==="
code-search search "API.*endpoint" --dir ~/projects/frontend --max-results 5

echo "=== Backend ==="
code-search search "API.*endpoint" --dir ~/projects/backend --max-results 5

echo "=== Shared ==="
code-search search "API.*endpoint" --dir ~/projects/shared --max-results 5
```

### Scenario 2: Code Review and Auditing

You need to find all TODO comments and potential security issues across a codebase.

```bash
# Index the entire codebase
code-search index --dir ~/workspace/security-audit

# Find all TODO comments
code-search search "TODO|FIXME|HACK" --dir ~/workspace/security-audit --format json > todos.json

# Find potential security issues
code-search search "password.*=|secret.*=|token.*=" --dir ~/workspace/security-audit --with-context

# Find hardcoded URLs
code-search search "http://|https://.*localhost" --dir ~/workspace/security-audit --with-context

# Export results for analysis
code-search search "eval|exec|system" --dir ~/workspace/security-audit --format raw > risky_functions.txt
```

### Scenario 3: Learning a New Codebase

You're joining a new team and want to understand the codebase structure.

```bash
# Index the main project
code-search index --dir ~/newteam/project

# Find the main entry points
code-search search "func main|package main" --dir ~/newteam/project --with-context

# Find API endpoints
code-search search "@.*Mapping|@.*Controller|@.*Route" --dir ~/newteam/project --with-context

# Find database models
code-search search "type.*struct.*Model|class.*Model" --dir ~/newteam/project --with-context

# Find configuration files
code-search search "config|settings|properties" --dir ~/newteam/project --file-pattern "*.yml,*.yaml,*.json,*.properties"
```

### Scenario 4: Refactoring and Code Cleanup

You're planning a refactoring and need to understand dependencies.

```bash
# Index the project
code-search index --dir ~/project/refactor-target

# Find all imports of a specific module
code-search search "import.*oldmodule|from.*oldmodule" --dir ~/project/refactor-target

# Find usage of a deprecated function
code-search search "deprecated_function" --dir ~/project/refactor-target --with-context

# Find test files related to specific functionality
code-search search "Test.*function_name|test.*function_name" --dir ~/project/refactor-target --file-pattern "*test*.go,*test*.js"

# Generate a report
code-search search "legacy_|deprecated|old_" --dir ~/project/refactor-target --format json > legacy_code.json
```

### Scenario 5: Performance Analysis

You need to identify performance bottlenecks in a large codebase.

```bash
# Index with verbose output to see performance metrics
code-search index --dir ~/large-project --verbose

# Find database queries
code-search search "SELECT|INSERT|UPDATE|DELETE|sql\.|query\." --dir ~/large-project --with-context

# Find loops and recursive functions
code-search search "for.*range|while.*true|func.*recurs" --dir ~/large-project --with-context

# Find file I/O operations
code-search search "os\.Open|file\.Read|file\.Write|io\.Copy" --dir ~/large-project --with-context

# Search in specific performance-critical directories
code-search search "cache|buffer|pool" --dir ~/large-project/src/performance --semantic
```

## Advanced Usage Patterns

### Batch Operations

```bash
#!/bin/bash
# Index multiple projects in parallel
PROJECTS=("frontend" "backend" "shared" "docs")
BASE_DIR="/home/user/projects"

for project in "${PROJECTS[@]}"; do
    echo "Indexing $project..."
    code-search index --dir "$BASE_DIR/$project" --verbose &
done

wait
echo "All projects indexed!"
```

### Search Across Multiple Directories

```bash
#!/bin/bash
# Search across multiple related directories
QUERY="authentication.*token"
DIRS=("$HOME/projects/api" "$HOME/projects/auth" "$HOME/projects/shared")

echo "Searching for: $QUERY"
for dir in "${DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo "=== Results from $dir ==="
        code-search search "$QUERY" --dir "$dir" --max-results 3
        echo ""
    fi
done
```

### Conditional Indexing

```bash
#!/bin/bash
# Only index if changes detected
PROJECT_DIR="/home/user/project"
INDEX_MARKER="$PROJECT_DIR/.clindex/last_index"

# Get latest modification time
LATEST_MOD=$(find "$PROJECT_DIR" -type f -not -path "*/.clindex/*" -printf "%T@\n" | sort -n | tail -1)

if [ -f "$INDEX_MARKER" ]; then
    LAST_INDEX=$(cat "$INDEX_MARKER")
    if [ "$(echo "$LATEST_MOD > $LAST_INDEX" | bc -l)" -eq 1 ]; then
        echo "Changes detected, reindexing..."
        code-search index --dir "$PROJECT_DIR"
        echo "$LATEST_MOD" > "$INDEX_MARKER"
    else
        echo "No changes detected, using existing index."
    fi
else
    echo "No index found, creating initial index..."
    code-search index --dir "$PROJECT_DIR"
    echo "$LATEST_MOD" > "$INDEX_MARKER"
fi
```

## Output Formats and Processing

### JSON Output for Scripting

```bash
# Search and get JSON results
code-search search "TODO" --dir ./src --format json > todos.json

# Process with jq
cat todos.json | jq '.results[] | {file: .file_path, line: .start_line, content: .content}'

# Count results by file type
code-search search "class.*Test" --dir ./test --format json | \
  jq -r '.results[].file_path' | \
  sort | uniq -c | sort -nr
```

### Raw Output for Simple Processing

```bash
# Get simple file:line:content format
code-search search "FIXME" --dir ./src --format raw > fixme_list.txt

# Count occurrences
code-search search "console\.log|debug\.log" --dir ./src --format raw | wc -l

# Extract unique files
code-search search "import.*React" --dir ./src --format raw | cut -d: -f1 | sort -u
```

### Table Output for Human Reading

```bash
# Pretty table output with context
code-search search "bug.*fix" --dir ./src --with-context --max-results 20

# Search with higher relevance threshold
code-search search "performance" --dir ./src --semantic --threshold 0.8
```

## Integration with Development Tools

### Git Integration

```bash
# Search only in modified files
MODIFIED_FILES=$(git diff --name-only HEAD~1)
for file in $MODIFIED_FILES; do
    if [ -f "$file" ]; then
        echo "=== $file ==="
        code-search search "TODO|FIXME" --dir . --file-pattern "$file"
    fi
done

# Search in current branch only
git diff --name-only HEAD~1 | xargs -I {} code-search search "test" --dir . --file-pattern "{}"
```

### IDE Integration (VS Code)

```bash
# Search from VS Code integrated terminal
code-search search "function.*handle" --dir . --with-context

# Quick file search
code-search search "className" --dir . --exact --format raw | head -5

# Find related test files
code-search search "import.*from.*src" --dir ./test --format raw | cut -d: -f1
```

### CI/CD Pipeline Integration

```yaml
# GitHub Actions example
- name: Index source code
  run: |
    code-search index --dir ${{ github.workspace }} --verbose

- name: Search for TODOs
  run: |
    code-search search "TODO|FIXME" --dir ${{ github.workspace }} --format json > todos.json

- name: Search for security issues
  run: |
    code-search search "password|secret|token.*=" --dir ${{ github.workspace }} --with-context > security_issues.txt

- name: Upload artifacts
  uses: actions/upload-artifact@v3
  with:
    name: search-results
    path: |
      todos.json
      security_issues.txt
```

## Performance Examples

### Large Directory Optimization

```bash
# Monitor indexing performance
time code-search index --dir ~/large-codebase --verbose

# Search with performance options
code-search search "important.*function" --dir ~/large-codebase \
  --max-results 50 \
  --semantic \
  --threshold 0.9

# Use file patterns to reduce search scope
code-search search "API.*endpoint" --dir ~/large-codebase \
  --file-pattern "*.go,*.js,*.py" \
  --max-results 20
```

### Memory-Constrained Operations

```bash
# Search with small result sets for limited memory
code-search search "config" --dir ~/huge-project \
  --max-results 10 \
  --format raw

# Batch search for memory efficiency
for term in "error" "warning" "info"; do
    echo "=== Searching for $term ==="
    code-search search "$term" --dir ~/huge-project --max-results 5
    sleep 1  # Allow memory cleanup
done
```

## Troubleshooting Examples

### Common Issues and Solutions

```bash
# Issue: Directory not found
# Solution: Check path and permissions
ls -la /path/to/directory
code-search index --dir /path/to/directory

# Issue: Permission denied
# Solution: Check directory permissions
ls -ld /path/to/directory
code-search index --dir /path/to/directory --verbose

# Issue: Index not found
# Solution: Create index first
code-search index --dir /path/to/project
code-search search "query" --dir /path/to/project

# Issue: Slow performance
# Solution: Use verbose mode to diagnose
code-search index --dir /path/to/large-project --verbose
code-search search "query" --dir /path/to/large-project --max-results 10

# Issue: Memory usage
# Solution: Limit results and use smaller batches
code-search search "broad.*term" --dir /path/to/project \
  --max-results 50 \
  --format raw
```

### Debug Information

```bash
# Enable verbose output for debugging
code-search index --dir ./project --verbose 2>&1 | tee index.log
code-search search "query" --dir ./project --verbose 2>&1 | tee search.log

# Check index status
ls -la ./project/.clindex/
cat ./project/.clindex/metadata.json

# Test directory validation
code-search index --dir ./project --verbose 2>&1 | grep -E "(validation|error|permission)"
```

## Migration Examples

### Upgrading from Legacy Index Format

```bash
# Check current status
echo "=== Checking migration status ==="
code-search index --dir ./project --verbose

# Force migration if needed
code-search index --dir ./project --force --verbose

# Verify migration worked
ls -la ./project/.clindex/
code-search search "test" --dir ./project --max-results 1
```

### Batch Migration of Multiple Projects

```bash
#!/bin/bash
# Migrate multiple legacy projects
PROJECTS=(
    "/home/user/project1"
    "/home/user/project2"
    "/home/user/project3"
)

for project in "${PROJECTS[@]}"; do
    echo "=== Migrating $project ==="
    if [ -f "$project/.code-search-index" ]; then
        code-search index --dir "$project" --force --verbose
        echo "Migration completed for $project"
    else
        echo "No legacy index found in $project"
    fi
    echo ""
done
```

## Best Practices

### Directory Organization

```bash
# Good: Use clear, descriptive directory names
code-search index --dir ~/projects/ecommerce-frontend
code-search index --dir ~/projects/ecommerce-backend

# Good: Use relative paths for project-local searches
code-search search "API.*endpoint" --dir ./src/api
code-search search "test.*user" --dir ./tests/user

# Good: Use home directory expansion for personal projects
code-search search "personal.*config" --dir ~/.config/myapp
```

### Search Optimization

```bash
# Good: Use specific search terms
code-search search "UserController.*authenticate" --dir ./src --exact

# Good: Use appropriate search types
code-search search "user.*experience" --dir ./docs --semantic
code-search search "function.*error" --dir ./src --fuzzy

# Good: Limit results to relevant files
code-search search "test.*unit" --dir ./src --file-pattern "*test*.go"
code-search search "style.*css" --dir ./assets --file-pattern "*.css"
```

### Performance Considerations

```bash
# Good: Index when code is stable
code-search index --dir ./stable-branch

# Good: Use file patterns to reduce scope
code-search search "import.*React" --dir ./src --file-pattern "*.jsx,*.tsx"

# Good: Limit results for broad searches
code-search search "function" --dir ./src --max-results 20

# Good: Use appropriate output format
code-search search "API.*endpoint" --dir ./api --format json  # For scripts
code-search search "bug.*fix" --dir ./src --with-context  # For human reading
```

## Script Templates

### Template 1: Project Setup Script

```bash
#!/bin/bash
# setup-project-search.sh - Set up code search for a project

PROJECT_DIR="${1:-.}"
PROJECT_NAME=$(basename "$PROJECT_DIR")

echo "Setting up code search for: $PROJECT_NAME ($PROJECT_DIR)"

# Create the index
echo "Creating index..."
code-search index --dir "$PROJECT_DIR" --verbose

# Test the index
echo "Testing index..."
code-search search "main|function" --dir "$PROJECT_DIR" --max-results 3

# Show project stats
echo "Project stats:"
code-search index --dir "$PROJECT_DIR" --verbose | grep -E "(files|indexed|size)"

echo "Setup complete! Use: code-search search '<query>' --dir '$PROJECT_DIR'"
```

### Template 2: Multi-Project Search Script

```bash
#!/bin/bash
# search-projects.sh - Search across multiple projects

QUERY="${1:-TODO}"
PROJECTS=(
    "$HOME/projects/frontend"
    "$HOME/projects/backend"
    "$HOME/projects/shared"
)

echo "Searching for: $QUERY"
echo "========================"

for project in "${PROJECTS[@]}"; do
    if [ -d "$project" ]; then
        project_name=$(basename "$project")
        echo ""
        echo "=== $project_name ==="
        code-search search "$QUERY" --dir "$project" --max-results 5
    fi
done
```

### Template 3: Code Review Helper

```bash
#!/bin/bash
# code-review-helper.sh - Help with code review

PROJECT_DIR="${1:-.}"
REVIEW_FILE="review_results.txt"

echo "Code Review Report for: $(basename "$PROJECT_DIR")" > "$REVIEW_FILE"
echo "Generated: $(date)" >> "$REVIEW_FILE"
echo "======================================" >> "$REVIEW_FILE"

# Find TODOs and FIXMEs
echo "" >> "$REVIEW_FILE"
echo "TODOs and FIXMEs:" >> "$REVIEW_FILE"
echo "------------------" >> "$REVIEW_FILE"
code-search search "TODO|FIXME|HACK|XXX" --dir "$PROJECT_DIR" --with-context >> "$REVIEW_FILE"

# Find potential security issues
echo "" >> "$REVIEW_FILE"
echo "Potential Security Issues:" >> "$REVIEW_FILE"
echo "-------------------------" >> "$REVIEW_FILE"
code-search search "password.*=|secret.*=|token.*=|key.*=" --dir "$PROJECT_DIR" --with-context >> "$REVIEW_FILE"

# Find complex functions (long, many parameters)
echo "" >> "$REVIEW_FILE"
echo "Complex Functions:" >> "$REVIEW_FILE"
echo "------------------" >> "$REVIEW_FILE"
# This would need more sophisticated search patterns

echo "Review complete! Results saved to: $REVIEW_FILE"
```

These examples demonstrate practical usage patterns and can be adapted for specific workflows and requirements.