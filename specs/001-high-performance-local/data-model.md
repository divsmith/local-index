# Data Model: High Performance Local CLI Vectorized Codebase Search

## Key Entities

### CodeIndex
- **Description**: Represents the indexed representation of the codebase
- **Fields**:
  - `id` (string): Unique identifier for the index
  - `version` (string): Version of the indexing schema
  - `repositoryPath` (string): Path to the root of the indexed repository
  - `lastModified` (time.Time): Timestamp of last index modification
  - `fileEntries` (map[string]FileEntry): Map of file paths to their indexed data
  - `vectorStore` (interface{}): Embedded vector database interface

### FileEntry
- **Description**: Represents a single file in the indexed codebase
- **Fields**:
  - `filePath` (string): Path to the file relative to repository root
  - `lastModified` (time.Time): Last modification time of the file
  - `contentHash` (string): Hash of the file content for change detection
  - `chunks` ([]CodeChunk): List of code chunks from the file

### CodeChunk
- **Description**: A segment of code that has been vectorized for search
- **Fields**:
  - `id` (string): Unique identifier for this chunk
  - `content` (string): The actual code content
  - `startLine` (int): Starting line number in the original file
  - `endLine` (int): Ending line number in the original file
  - `vector` ([]float64): Vector representation of the code content
  - `context` (string): Surrounding context of the code

### SearchQuery
- **Description**: Represents a search request from the user
- **Fields**:
  - `queryText` (string): The text to search for
  - `maxResults` (int): Maximum number of results to return (default: 10)
  - `includeContext` (bool): Whether to include code context in results
  - `fileFilter` (string): Optional filter for specific file patterns

### SearchResult
- **Description**: Represents a single match found during search
- **Fields**:
  - `filePath` (string): Path to the file containing the match
  - `startLine` (int): Line number where the match starts
  - `endLine` (int): Line number where the match ends
  - `content` (string): The matching code content
  - `context` (string): Surrounding code context
  - `relevanceScore` (float64): Score indicating relevance of the match
  - `vectorDistance` (float64): Distance in vector space from the query

### SearchResults
- **Description**: Collection of search results
- **Fields**:
  - `query` (SearchQuery): The original query that generated these results
  - `results` ([]SearchResult): List of individual search results
  - `totalResults` (int): Total number of matches found
  - `executionTime` (time.Duration): Time taken to execute the search

## Relationships
- A `CodeIndex` contains many `FileEntry` objects
- A `FileEntry` contains many `CodeChunk` objects
- A `SearchQuery` generates one `SearchResults`
- A `SearchResults` contains many `SearchResult` objects

## Validation Rules
- Each `FileEntry` must have a valid file path
- Each `CodeChunk` must have a valid content and vector representation
- Each `SearchResult` must have a relevance score between 0 and 1
- The `contentHash` in `FileEntry` must match the actual file content hash
- Each `CodeChunk` must have a non-empty vector representation