# CLI Contract: High Performance Local CLI Vectorized Codebase Search

## Command: `code-search index`

### Description
Indexes the current directory and subdirectories to create a vectorized representation of the codebase for fast searching.

### Arguments
- None required

### Options
- `--force` : Force re-indexing even if index exists
- `--include-hidden` : Include hidden files and directories in indexing
- `--file-types` : Specify file types to include (default: common programming languages)

### Expected Exit Codes
- `0` : Success
- `1` : Error during indexing
- `2` : Invalid arguments

### Standard Output
```
Indexing complete. Indexed {number} files in {time} seconds.
```

### Standard Error
```
Error: {error description}
```

---

## Command: `code-search search <query>`

### Description
Searches the indexed codebase for patterns matching the query using vectorized search.

### Arguments
- `query` : The search query string

### Options
- `--max-results <n>` : Maximum number of results to return (default: 10)
- `--with-context` : Include code context in results
- `--file-pattern <pattern>` : Filter results by file pattern (e.g., "*.go")
- `--format <format>` : Output format (table, json, raw; default: table)

### Expected Exit Codes
- `0` : Search completed successfully (results may or may not be found)
- `1` : Error during search
- `2` : Invalid arguments
- `3` : Index not found (run index command first)

### Standard Output (table format)
```
Found {n} results:

1. {file_path}:{start_line}-{end_line}
   {code_content_snippet}

2. {file_path}:{start_line}-{end_line}
   {code_content_snippet}
```

### Standard Output (json format)
```json
{
  "query": "{search_query}",
  "results": [
    {
      "filePath": "{file_path}",
      "startLine": {line_number},
      "endLine": {line_number},
      "content": "{code_content}",
      "context": "{surrounding_code}",
      "relevanceScore": {score}
    }
  ],
  "totalResults": {number},
  "executionTime": "{duration}"
}
```

### Standard Error
```
Error: {error description}
```