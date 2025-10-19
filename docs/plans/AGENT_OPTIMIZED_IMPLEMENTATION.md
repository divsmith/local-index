# Agent-Optimized Code Search Implementation Plan

## Overview

This document provides a comprehensive implementation plan for transforming our code search tool into an agent-optimized CLI that minimizes token usage through semantic search instead of grep.

**Target Audience:** Skilled developers unfamiliar with our codebase, toolset, or problem domain.
**Principles:** DRY, YAGNI, TDD, frequent commits.
**Binary Size Target:** Keep CLI tool under 50MB for fast distribution.

## Current State Analysis

### What Already Exists
**Working Components:**
- Go-based CLI tool with basic search functionality (builds and runs)
- Search command with extensive options (currently human-focused, needs agent-first defaults)
- Basic indexing system with database persistence
- Mock embedding service that generates deterministic hash-based vectors
- Project structure: `src/`, `tests/`, `docs/`, `bin/`

**Current Status:**
- **Build:** ✅ Compiles successfully with `go build ./...`
- **Basic CLI:** ✅ `bin/code-search search <query>` works
- **Tests:** ⚠️ Some tests pass, some have integration issues
- **Performance:** 📊 Unknown - needs benchmarking

### Key Files to Understand First
1. `src/search_cmd.go` - Main CLI search interface (HEAVY MODIFICATION NEEDED)
2. `src/lib/embedding.go` - Mock embedding (COMPLETE REPLACEMENT NEEDED)
3. `src/services/search_service.go` - Core search logic (MODERATE CHANGES)
4. `src/services/indexing_service.go` - Indexing functionality (MODERATE CHANGES)
5. `src/models/` - Data structures (MINIMAL CHANGES)
6. `go.mod` - Dependencies (WILL NEED ONNX LIBRARIES)

### Immediate Action Items Before Starting
1. **Run current codebase to understand existing behavior:**
   ```bash
   go build -o bin/code-search ./...
   ./bin/code-search search --help
   ./bin/code-search search "test query"
   ```

2. **Benchmark current performance (Phase 0):**
   ```bash
   # Create a test repo with ~100k lines of code
   time ./bin/code-search index  # Time the indexing
   time ./bin/code-search search "function"  # Time a search
   ```

3. **Check existing test status:**
   ```bash
   go test ./...  # See what currently passes/fails
   ```

## Frequently Asked Questions

### 1. Current Status: What's the current state of the codebase?

**Working Components:**
- ✅ **Builds successfully:** `go build ./...` works
- ✅ **Basic CLI functional:** `bin/code-search search <query>` works
- ✅ **Mock embeddings:** Currently generates hash-based vectors (deterministic but not semantic)
- ✅ **Database persistence:** Indexes are stored and retrieved
- ⚠️ **Tests:** Some pass, some integration issues exist
- ❓ **Performance:** Unknown - needs benchmarking

**What's Working Right Now:**
```bash
# These commands should work immediately:
go build -o bin/code-search ./...
./bin/code-search search --help
./bin/code-search search "test query" --format json
./bin/code-search index  # Creates .code-search-index.db in current directory
```

**What Needs Complete Replacement:**
- `src/lib/embedding.go` - Mock service → Real ONNX implementation
- `src/search_cmd.go` - Human-focused defaults → Agent-optimized defaults
- Index storage - Repository-local → Centralized in `~/.code-search/`

### 2. Priority: Which phase should we start with?

**RECOMMENDED STARTING ORDER:**

**✅ Phase 0: Baseline Understanding** (30 minutes) - **COMPLETED**
```bash
# 1. Get familiar with current functionality ✅
go build -o bin/code-search ./...
./bin/code-search search --help
./bin/code-search search "function" --max-results 10

# 2. Check test status ✅
go test ./...  # All tests pass - excellent baseline

# 3. Benchmark current performance ✅
time ./bin/code-search index  # ~45 files/min (slow - needs optimization)
time ./bin/code-search search "function"
```

**✅ Phase 1: Agent-Optimized CLI Defaults** (1-2 days) - **COMPLETED**
- ✅ Task 1.1: Ultra-minimal defaults (2 results, file:line only)
- ✅ Task 1.2: Smart file filtering (exclude tests, vendor, etc.)
- ✅ Task 1.3: Agent-first help system

**Why Start Here:**
- Immediate value for agents
- No new dependencies required
- Builds confidence with quick wins
- Foundation for all later phases

**Phase 2-3: Smart Query Detection & Centralized Storage** (2-3 days)
- Can be done in parallel with Phase 4
- Uses existing mock embedding initially
- Gets architecture in place before ONNX complexity

**Phase 4: ONNX Integration** (3-4 days)
- Should be done after other features are working
- Can develop with mock embeddings, then swap in ONNX
- Model download is critical path - implement this first in Phase 4

### 3. Dependencies: Do you have the ONNX model file available?

**MODEL STATUS: NOT AVAILABLE**

**What You Need to Implement:**
1. **Download functionality** in `src/lib/model_manager.go`
2. **HTTP client** to fetch from Hugging Face
3. **Local storage** in `~/.code-search/models/`
4. **Progress indication** for 438MB download

**Model Details:**
- **Name:** `all-mpnet-base-v2.onnx`
- **Size:** ~438MB
- **Source:** Hugging Face sentence-transformers
- **Storage:** `~/.code-search/models/all-mpnet-base-v2.onnx`

**Download Strategy:**
```go
// URL pattern for download:
"https://huggingface.co/sentence-transformers/resolve/main/all-mpnet-base-v2.onnx"
```

**Implementation Priority:**
1. Create mock download functionality first (returns existing model if exists)
2. Implement real HTTP download
3. Add progress bars and error handling
4. Test with actual 438MB download

### 4. Performance Requirements: Have you benchmarked current performance?

**PERFORMANCE STATUS: UNBENCHMARKED**

**Targets We Need to Hit:**
- **Indexing 100k lines:** <10 seconds (current: unknown)
- **Search response:** <2 seconds (current: unknown)
- **Memory usage:** <100MB during indexing (current: unknown)
- **Binary size:** <50MB final CLI (current: smaller)

**Immediate Benchmarking Tasks:**
```bash
# 1. Create test repository with ~100k lines
# 2. Time the indexing process
time ./bin/code-search index

# 3. Time various search types
time ./bin/code-search search "function"              # Text search
time ./bin/code-search search "function" --semantic   # Mock semantic search

# 4. Check memory usage
/usr/bin/time -v ./bin/code-search search "function"  # Look at "Maximum resident set size"
```

**Performance Optimization Plan:**
- Phase 1-2: Measure current baseline
- Phase 3: Optimize indexing (parallel processing, smart filtering)
- Phase 4: Optimize ONNX inference (batching, caching)
- Phase 5: Optimize hybrid search (result merging, caching)

**If Current Performance is Poor:**
- Focus on low-hanging fruit first (file filtering, indexing)
- Consider performance optimizations before adding ONNX complexity
- May need to revisit target numbers based on baseline

## Phase 1: Agent-Optimized CLI Defaults

### ✅ Task 1.1: Implement Ultra-Minimal Default Behavior - **COMPLETED**
**Goal:** Change default search to return 2 results, file:line only, core source files only.

**✅ Files Modified:**
- `src/search_cmd.go` - Updated default options in `parseSearchOptions()` (maxResults: 2)
- `src/search_cmd.go` - Modified `displayTableResults()` for minimal output format

**Implementation Details:**
```go
// In parseSearchOptions(), change defaults:
options := SearchOptions{
    maxResults:    2,        // Changed from 10
    filePattern:   "",       // Will implement smart filtering
    withContext:   false,    // Default: no context
    format:        "table",  // Keep table but make it minimal
    threshold:     0.7,
    semantic:      false,    // Will implement smart detection
    exact:         false,    // Will implement smart detection
    fuzzy:         false,    // Will implement smart detection
}
```

**New Minimal Output Format:**
```
src/db/connection.go:15
src/models/user.go:89
```

**✅ Testing Strategy - COMPLETED:**
- ✅ Tested: `code-search "database"` returns exactly 2 results
- ✅ Tested: Default output format matches expected minimal pattern (file:line only)
- ✅ Tested: Backward compatibility with existing options

**✅ How to Test - COMPLETED:**
```bash
go build -o bin/code-search ./...
./bin/code-search "database"  # Returns 2 results max ✅
./bin/code-search "database" --max-results 5  # Still works ✅
```

### ✅ Task 1.2: Implement Smart File Filtering - **COMPLETED**
**Goal:** Filter out tests/, docs/, vendor/, build artifacts by default.

**✅ Files Created/Modified:**
- `src/models/util/file_filter.go` - New file for intelligent file filtering
- `src/models/search_query.go` - Added SmartFilter field and ShouldIncludeFile logic
- `src/search_cmd.go` - Integrated filtering into search options

**Implementation Details:**
```go
// In new file_filter.go:
type FileFilter struct {
    excludePatterns []string
    includeOnly     []string
}

func NewAgentFileFilter() *FileFilter {
    return &FileFilter{
        excludePatterns: []string{
            "*/test*", "*/tests/*", "*_test.go",
            "*/vendor/*", "*/node_modules/*",
            "*/docs/*", "*/doc/*",
            "*/build/*", "*/dist/*", "*/target/*",
            "*/.git/*",
        },
        includeOnly: []string{".go", ".js", ".ts", ".py", ".java", ".cpp", ".c", ".h"},
    }
}
```

**Testing Strategy:**
- Test that test files are excluded by default
- Test that vendor directories are excluded
- Test that core source files are included
- Test that `--file-pattern` can override defaults

**How to Test:**
```bash
# Create test structure with various file types
./bin/code-search "function" --format json | jq '.results[].file_path' | should not include "_test.go"
```

### ✅ Task 1.3: Create Agent-First Help System - **COMPLETED**
**Goal:** Reorganize help to prioritize agent-frequently used options.

**✅ Files Modified:**
- `src/search_cmd.go` - Updated `printSearchHelp()` method

**✅ Implementation Details - COMPLETED:**
```
Usage: code-search search <query> [options]

Top Options for Agents:
  --format json            Machine-readable output for parsing
  -m, --max-results <n>    Number of results (default: 2)
  -f, --file-pattern <p>   Include specific file types
  -c, --with-context       Add code snippets when needed

All Options:
  [less common options...]
```

**✅ Testing Strategy - COMPLETED:**
- ✅ Tested: Help output contains agent options first
- ✅ Tested: Help is accessible and readable

**✅ How to Test - COMPLETED:**
```bash
./bin/code-search --help | head -20  # Shows agent options first ✅
```

## Phase 2: Smart Query Detection System

### ✅ Task 2.1: Implement Query Type Analysis - **COMPLETED**
**Goal:** Automatically detect query patterns and select optimal search strategy.

**✅ Files Created/Modified:**
- `src/lib/query_analyzer.go` - ✅ New file for query analysis
- `src/lib/query_analyzer_test.go` - ✅ Comprehensive test suite
- `src/search_cmd.go` - ✅ Integrate analyzer into command execution

**Implementation Details:**
```go
// In new query_analyzer.go:
type QueryType int
const (
    QueryTypeUnknown QueryType = iota
    QueryTypeExact      // "TODO", "FIXME" - exact strings
    QueryTypeRegex      // "func.*error" - patterns
    QueryTypeSemantic   // "user authentication" - concepts
    QueryTypeHybrid     // "calculate tax" - function + concept
)

type QueryAnalyzer struct{}

func (qa *QueryAnalyzer) AnalyzeQuery(query string) QueryType {
    // Implement detection logic:
    if strings.Contains(query, "TODO") || strings.Contains(query, "FIXME") {
        return QueryTypeExact
    }
    if isRegexPattern(query) {
        return QueryTypeRegex
    }
    if isConceptQuery(query) {
        return QueryTypeSemantic
    }
    return QueryTypeHybrid
}
```

**✅ Implementation Details - COMPLETED:**
- ✅ QueryType enum: Exact, Regex, Semantic, Hybrid, Unknown
- ✅ Pattern detection with regex-based analysis
- ✅ Semantic keyword recognition (auth, database, api, etc.)
- ✅ Helper methods: ShouldUseSemanticSearch(), ShouldUseExactMatch(), ShouldUseRegex()

**✅ Testing Strategy - COMPLETED:**
- ✅ Test query type detection for various patterns
- ✅ Test that `"TODO"` triggers exact search
- ✅ Test that `"user authentication"` triggers semantic search
- ✅ Test that `"func.*error"` triggers regex search
- ✅ All unit tests pass (100% success rate)

**✅ How to Test - COMPLETED:**
```bash
# These should be detected automatically:
./bin/code-search "TODO"           # ✅ Should use exact search
./bin/code-search "user auth"      # ✅ Should use semantic search
./bin/code-search "func.*Error"    # ✅ Should use regex search
```

### ✅ Task 2.2: Integrate Smart Query Routing - **COMPLETED**
**Goal:** Automatically set search parameters based on query analysis.

**✅ Files Modified:**
- `src/search_cmd.go` - ✅ Update `Execute()` method to use query analyzer

**Implementation Details:**
```go
// In Execute() method, add:
analyzer := lib.NewQueryAnalyzer()
queryType := analyzer.AnalyzeQuery(queryText)

switch queryType {
case lib.QueryTypeExact:
    options.exact = true
case lib.QueryTypeRegex:
    options.semantic = false  // Use text search for regex
case lib.QueryTypeSemantic:
    options.semantic = true
case lib.QueryTypeHybrid:
    options.semantic = true
    query.SearchType = models.SearchTypeHybrid
}
```

**✅ Implementation Details - COMPLETED:**
- ✅ Query analyzer initialization in Execute() method
- ✅ Smart query routing that respects explicit options first
- ✅ Automatic search type selection based on query analysis
- ✅ Fallback to hybrid for unknown queries

**✅ Testing Strategy - COMPLETED:**
- ✅ Test that query analyzer correctly influences search parameters
- ✅ Test integration with existing search functionality
- ✅ Verified smart routing works with different query types

**✅ How to Test - COMPLETED:**
```bash
# Test that different query types produce appropriate results
./bin/code-search "TODO" --format json | jq '.results[0].match_type'  # ✅ Should be "exact"
./bin/code-search "user auth" --format json                         # ✅ Should use semantic
./bin/code-search "func.*Error"                                     # ✅ Should use regex
```

## Phase 3: Centralized Index Storage System

### Task 3.1: Implement Project-Scoped Storage
**Goal:** Move from repository-local to centralized project-scoped index storage.

**Files to Create/Modify:**
- `src/lib/storage_manager.go` - New file for centralized storage management
- `src/lib/project_detector.go` - New file for project boundary detection
- `src/services/indexing_service.go` - Update to use storage manager
- `src/search_cmd.go` - Update to use centralized storage

**Implementation Details:**
```go
// In new storage_manager.go:
type StorageManager struct {
    baseDir string
}

func NewStorageManager() *StorageManager {
    homeDir, _ := os.UserHomeDir()
    return &StorageManager{
        baseDir: filepath.Join(homeDir, ".code-search"),
    }
}

func (sm *StorageManager) GetProjectIndexPath(projectPath string) string {
    // Create project identifier (hash of full path)
    projectID := sm.hashProjectPath(projectPath)
    return filepath.Join(sm.baseDir, "indexes", projectID+".db")
}

func (sm *StorageManager) GetProjectEmbeddingPath(projectPath string) string {
    projectID := sm.hashProjectPath(projectPath)
    return filepath.Join(sm.baseDir, "embeddings", projectID+".onnx")
}
```

**Directory Structure:**
```
~/.code-search/
├── indexes/
│   {project-hash-1}.db
│   {project-hash-2}.db
├── embeddings/
│   {project-hash-1}.cache
│   {project-hash-2}.cache
└── models/
    └── all-mpnet-base-v2.onnx
```

**Testing Strategy:**
- Test that different projects get separate indexes
- Test that indexes are stored in correct location
- Test project path hashing is consistent

**How to Test:**
```bash
# Create two test projects
cd /tmp/project1
./bin/code-search index
cd ../project2
./bin/code-search index
# Verify separate index files exist in ~/.code-search/indexes/
```

### Task 3.2: Implement Project Boundary Detection
**Goal:** Automatically detect project root directories for proper scoping.

**Files to Create/Modify:**
- `src/lib/project_detector.go` - Implement git-aware project detection

**Implementation Details:**
```go
// In project_detector.go:
type ProjectDetector struct{}

func (pd *ProjectDetector) DetectProjectRoot(startPath string) (string, error) {
    // First, try to find .git directory
    if gitRoot := pd.findGitRoot(startPath); gitRoot != "" {
        return gitRoot, nil
    }

    // Fallback: use current directory
    return startPath, nil
}

func (pd *ProjectDetector) findGitRoot(path string) string {
    for {
        if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
            return path
        }

        parent := filepath.Dir(path)
        if parent == path {
            return ""  // Reached root
        }
        path = parent
    }
}
```

**Testing Strategy:**
- Test detection of git repository roots
- Test behavior in non-git directories
- Test behavior in nested subdirectories

**How to Test:**
```bash
# Test in git repo
cd /my-git-project/subdir
./bin/code-search "function"  # Should search whole git project
```

## Phase 4: ONNX Embedding Model Integration

### Task 4.1: Add ONNX Runtime Dependencies
**Goal:** Integrate real ONNX-based embedding model instead of mock implementation.

**DEPENDENCY STATUS:** ONNX model file is NOT currently available. Must implement download functionality first.

**Files to Modify:**
- `go.mod` - Add ONNX runtime dependency
- `src/lib/embedding.go` - Replace mock with ONNX implementation

**Dependencies to Add:**
```go
// In go.mod, add:
require (
    github.com/owulveryck/onnx-go v0.7.0
    gorgonia.org/gorgonia v0.9.0  // For tensor operations
)
```

**Implementation Details:**
```go
// In embedding.go, replace MockEmbeddingService:
type ONNXEmbeddingService struct {
    model    onnx.Model
    session  onnx.Session
    config   EmbeddingConfig
    cache    *EmbeddingCache
}

func NewONNXEmbeddingService(config EmbeddingConfig) (*ONNXEmbeddingService, error) {
    // Load ONNX model from bundled file or download path
    modelPath := config.ModelPath
    if modelPath == "" {
        modelPath = filepath.Join(config.ModelDir, "all-mpnet-base-v2.onnx")
    }

    model, err := onnx.Load(modelPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load ONNX model: %w", err)
    }

    return &ONNXEmbeddingService{
        model:  model,
        config: config,
        cache:  NewEmbeddingCache(config.CacheSize, config.MemoryLimit, 24*time.Hour),
    }, nil
}
```

### Task 4.2: Download and Bundle Model (CRITICAL PATH)
**Goal:** Handle model acquisition (bundle with binary or download on first run).

**Files to Create/Modify:**
- `src/lib/model_manager.go` - New file for model management
- `src/search_cmd.go` - Add model download command

**MODEL AVAILABILITY:** The all-mpnet-base-v2 model needs to be:
1. Downloaded from Hugging Face (438MB ONNX file)
2. Stored in `~/.code-search/models/`
3. Available for first-time users

**Implementation Details:**
```go
// In model_manager.go:
type ModelManager struct {
    modelDir string
    client   *http.Client
}

func (mm *ModelManager) EnsureModel(modelName string) (string, error) {
    modelPath := filepath.Join(mm.modelDir, modelName+".onnx")

    if _, err := os.Stat(modelPath); err == nil {
        return modelPath, nil  // Model already exists
    }

    // Download model
    return mm.downloadModel(modelName, modelPath)
}

func (mm *ModelManager) downloadModel(modelName, savePath string) (string, error) {
    // Download from Hugging Face or other source
    url := fmt.Sprintf("https://huggingface.co/sentence-transformers/resolve/main/%s.onnx", modelName)

    resp, err := mm.client.Get(url)
    if err != nil {
        return "", fmt.Errorf("failed to download model: %w", err)
    }
    defer resp.Body.Close()

    // Save to disk
    file, err := os.Create(savePath)
    if err != nil {
        return "", fmt.Errorf("failed to create model file: %w", err)
    }
    defer file.Close()

    _, err = io.Copy(file, resp.Body)
    return savePath, err
}
```

**Testing Strategy:**
- Test model download functionality
- Test ONNX model loading and inference
- Test embedding dimensionality (should be 768 for mpnet-base-v2)

**How to Test:**
```bash
# First run should trigger model download
./bin/code-search "test query"  # Should download model if missing
# Verify model file exists in ~/.code-search/models/
```

**IMPORTANT NOTE:** This phase should be implemented AFTER Phase 1-3 are complete and tested. The current mock implementation can be used for initial development and testing of other features.

## Phase 5: Hybrid Search Strategy

### Task 5.1: Implement Fast Text Search Fallback
**Goal:** Provide immediate text search results while semantic indexing completes.

**Files to Modify:**
- `src/services/search_service.go` - Add hybrid search logic
- `src/services/indexing_service.go` - Add background indexing

**Implementation Details:**
```go
// In search_service.go:
type HybridSearchService struct {
    textSearchService   *SearchService
    semanticSearchService *EnhancedSearchService
    indexingService     *IndexingService
}

func (hss *HybridSearchService) Search(query *models.SearchQuery, indexPath string) (*models.SearchResults, error) {
    // Always start with fast text search
    textResults, err := hss.textSearchService.Search(query, indexPath)
    if err != nil {
        return nil, err
    }

    // If semantic search is ready and requested
    if hss.isSemanticSearchReady(indexPath) && query.NeedsSemantic() {
        semanticResults, err := hss.semanticSearchService.Search(query, indexPath)
        if err == nil && len(semanticResults.Results) > 0 {
            return hss.mergeResults(textResults, semanticResults, query)
        }
    }

    return textResults, nil
}
```

### Task 5.2: Background Indexing with Progress
**Goal:** Index in background and show progress to agents.

**Files to Create/Modify:**
- `src/services/background_indexer.go` - New file for background indexing
- `src/search_cmd.go` - Add progress indication

**Implementation Details:**
```go
// In background_indexer.go:
type BackgroundIndexer struct {
    queue chan IndexingJob
    status map[string]*IndexingStatus
}

func (bi *BackgroundIndexer) QueueIndexing(projectPath string) {
    job := IndexingJob{
        ProjectPath: projectPath,
        StartTime:   time.Now(),
    }

    bi.queue <- job
    bi.status[projectPath] = &IndexingStatus{
        Status: "queued",
        Progress: 0.0,
    }
}
```

**Testing Strategy:**
- Test hybrid search returns immediate results
- Test semantic results appear when ready
- Test background indexing doesn't block searches

**How to Test:**
```bash
# First search in new project should be fast (text-only)
time ./bin/code-search "function"  # Should return <1s
# Subsequent searches should include semantic results
./bin/code-search "function" --format json | jq '.results[0].score'  # Should have semantic scores
```

## Phase 6: Auto-Update System

### Task 6.1: File Watcher Implementation
**Goal:** Automatically update indexes when files change.

**Files to Create/Modify:**
- `src/services/file_watcher.go` - New file for file system monitoring
- `src/services/indexing_service.go` - Integrate with watcher

**Implementation Details:**
```go
// In file_watcher.go:
type FileWatcher struct {
    watcher   *fsnotify.Watcher
    indexer   *IndexingService
    project   string
    debounce  time.Duration
}

func NewFileWatcher(projectPath string, indexer *IndexingService) (*FileWatcher, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }

    fw := &FileWatcher{
        watcher:  watcher,
        indexer:  indexer,
        project:  projectPath,
        debounce: 1 * time.Second,
    }

    // Watch for file changes
    err = fw.watchDirectory(projectPath)
    return fw, err
}

func (fw *FileWatcher) Start() {
    go fw.processEvents()
}

func (fw *FileWatcher) processEvents() {
    var timer *time.Timer

    for {
        select {
        case event := <-fw.watcher.Events:
            if fw.shouldProcessEvent(event) {
                // Debounce rapid file changes
                if timer != nil {
                    timer.Stop()
                }
                timer = time.AfterFunc(fw.debounce, func() {
                    fw.indexer.UpdateIndex(fw.project, []string{event.Name})
                })
            }
        }
    }
}
```

### Task 6.2: Git-Aware Incremental Updates
**Goal:** Optimize updates by only processing changed files since last index.

**Files to Create/Modify:**
- `src/lib/git_utils.go` - New file for git operations
- `src/services/indexing_service.go` - Add git-aware updating

**Implementation Details:**
```go
// In git_utils.go:
type GitUtils struct {
    repoPath string
}

func (gu *GitUtils) GetChangedFiles(since time.Time) ([]string, error) {
    // Use git commands to get changed files
    cmd := exec.Command("git", "diff", "--name-only", "--since", since.Format(time.RFC3339))
    cmd.Dir = gu.repoPath

    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    files := strings.Split(string(output), "\n")
    result := make([]string, 0, len(files))

    for _, file := range files {
        if strings.TrimSpace(file) != "" {
            result = append(result, filepath.Join(gu.repoPath, file))
        }
    }

    return result, nil
}
```

**Testing Strategy:**
- Test file watcher detects file changes
- Test git integration correctly identifies changed files
- Test incremental updates are faster than full reindexing

**How to Test:**
```bash
# Start watching in background
./bin/code-search --watch &
# Modify a file
echo "// new comment" >> src/main.go
# Search should reflect changes quickly
./bin/code-search "new comment"  # Should find the addition
```

## Testing Strategy Throughout

### Unit Testing Requirements
For each component, create corresponding test files:

```
tests/
├── lib/
│   ├── query_analyzer_test.go
│   ├── file_filter_test.go
│   ├── storage_manager_test.go
│   └── model_manager_test.go
├── services/
│   ├── search_service_test.go
│   ├── indexing_service_test.go
│   └── file_watcher_test.go
└── search_cmd_test.go
```

### Integration Testing
Create end-to-end tests in `tests/integration/`:

```
tests/integration/
├── basic_search_test.go      # Test basic search functionality
├── semantic_search_test.go   # Test semantic search accuracy
├── project_isolation_test.go # Test project separation
└── performance_test.go       # Test performance targets
```

### Performance Benchmarks
Create benchmarks in `tests/benchmarks/`:

```go
// benchmarks/search_test.go
func BenchmarkBasicSearch(b *testing.B) {
    // Benchmark 2-result default search
}

func BenchmarkSemanticSearch(b *testing.B) {
    // Benchmark semantic search with mpnet-base-v2
}

func BenchmarkIndexingSpeed(b *testing.B) {
    // Benchmark indexing 100k lines <10s
}
```

### Continuous Integration Requirements
- All tests must pass on each commit
- Performance benchmarks must not regress
- Binary size must stay under 50MB
- Go 1.24.6 compatibility must be maintained

## Documentation Requirements

### Code Documentation
- All public functions must have Go doc comments
- Complex algorithms need inline comments
- Integration points need clear documentation

### User Documentation
Create/update in `docs/`:
- `docs/AGENT_GUIDE.md` - How agents should use the tool
- `docs/PERFORMANCE.md` - Performance characteristics and tuning
- `docs/TROUBLESHOOTING.md` - Common issues and solutions

### API Documentation
- Update help text and command documentation
- Document all configuration options
- Provide example usage patterns

## Success Criteria

Each phase should be considered complete when:
1. All new tests pass
2. No existing tests break
3. Performance targets are met
4. Code review feedback is addressed
5. Documentation is updated

## Rollback Plan

If any phase causes issues:
1. Use git to rollback to previous working state
2. Keep backup of existing CLI binary
3. Document what went wrong for learning
4. Consider alternative approaches

---

This implementation plan provides a clear roadmap for transforming the code search tool into an agent-optimized CLI. Each task builds upon previous work while maintaining system stability and performance.