# API Reference Documentation

## Overview

This document provides detailed API reference for the directory selection feature in the code-search CLI tool. It covers core models, services, and utilities that enable directory-specific indexing and searching.

## Core Models

### DirectoryConfig

Represents a validated directory configuration with metadata and permissions.

```go
type DirectoryConfig struct {
    Path         string            `json:"path"`          // Absolute path to directory
    OriginalPath string            `json:"original_path"`  // Path as provided by user
    IsDefault    bool              `json:"is_default"`     // True for current directory
    Permissions  DirectoryPerms    `json:"permissions"`   // Directory permissions
    Limits       DirectoryLimits   `json:"limits"`        // Operational limits
    Metadata     DirectoryMetadata `json:"metadata"`      // Directory information
}
```

**Methods:**
- `ToJSON() ([]byte, error)` - Convert to JSON
- `FromJSON(data []byte) error` - Load from JSON

### DirectoryPerms

Contains directory permission information.

```go
type DirectoryPerms struct {
    CanRead  bool `json:"can_read"`   // Read permission
    CanWrite bool `json:"can_write"`  // Write permission
    CanExec  bool `json:"can_exec"`   // Execute permission
}
```

### DirectoryLimits

Configurable limits for directory operations.

```go
type DirectoryLimits struct {
    MaxDirectorySize int64 `json:"max_directory_size"` // Maximum directory size in bytes
    MaxFileCount     int   `json:"max_file_count"`     // Maximum number of files
    MaxFileSize      int64 `json:"max_file_size"`      // Maximum individual file size
}
```

**Factory Function:**
- `NewDefaultDirectoryLimits() *DirectoryLimits` - Returns default limits

### DirectoryMetadata

Contains metadata about a directory including performance metrics.

```go
type DirectoryMetadata struct {
    FileCount    int64     `json:"file_count"`     // Number of files in directory
    TotalSize    int64     `json:"total_size"`     // Total size in bytes
    LastIndexed  time.Time `json:"last_indexed"`   // Last indexing timestamp
    IndexVersion string    `json:"index_version"`  // Index format version
    CreatedAt    time.Time `json:"created_at"`     // Directory creation time
    ModifiedAt   time.Time `json:"modified_at"`    // Last modification time
    ScanDuration string    `json:"scan_duration,omitempty"` // Time taken for scan
    MemoryUsed   float64   `json:"memory_used_mb,omitempty"` // Memory used during scan
}
```

**Methods:**
- `ToJSON() ([]byte, error)` - Convert to JSON
- `FromJSON(data []byte) error` - Load from JSON
- `IsIndexValid(dirModified time.Time) bool` - Check if index is still valid
- `MarkIndexed()` - Update metadata to reflect indexing

### IndexLocation

Represents the location and structure of index files.

```go
type IndexLocation struct {
    BaseDirectory string `json:"base_directory"` // Base directory being indexed
    IndexDir      string `json:"index_dir"`      // .clindex directory path
    MetadataFile  string `json:"metadata_file"`  // metadata.json path
    DataFile      string `json:"data_file"`      // Main index data file path
    LockFile      string `json:"lock_file"`      // Lock file path
}
```

**Factory Function:**
- `NewIndexLocation(baseDirectory string) *IndexLocation` - Creates new index location

### IndexMetadata

Metadata about the index itself (for migration tracking).

```go
type IndexMetadata struct {
    Version       string    `json:"version"`        // Index format version
    CreatedAt     time.Time `json:"created_at"`     // Index creation time
    UpdatedAt     time.Time `json:"updated_at"`     // Last update time
    FileCount     int       `json:"file_count"`     // Number of indexed files
    ChunkCount    int       `json:"chunk_count"`    // Number of code chunks
    Directory     string    `json:"directory"`      // Indexed directory path
    IndexerType   string    `json:"indexer_type"`   // Type of indexer used
    Migrated      bool      `json:"migrated"`       // True if migrated from legacy format
    MigrationDate time.Time `json:"migration_date,omitempty"` // Migration timestamp
    LegacyFiles   []string  `json:"legacy_files,omitempty"` // Legacy files that were migrated
    LegacyVersion string    `json:"legacy_version,omitempty"` // Legacy format version
}
```

### IndexInfo

Information about an index for querying status.

```go
type IndexInfo struct {
    Directory string    `json:"directory"`  // Directory path
    IndexPath string    `json:"index_path"` // Index file path
    Exists    bool      `json:"exists"`     // Whether index exists
    Locked    bool      `json:"locked"`     // Whether index is locked
    Size      int64     `json:"size"`       // Index file size
    Modified  time.Time `json:"modified"`   // Last modification time
}
```

## Services

### DirectoryValidator

Validates directories for indexing and searching operations.

```go
type DirectoryValidator struct {
    fileUtils *FileUtilities
}
```

**Constructor:**
- `NewDirectoryValidator() *DirectoryValidator`

**Methods:**
- `ValidateDirectory(path string) (*models.DirectoryConfig, error)` - Validates directory and returns config

**Example:**
```go
validator := lib.NewDirectoryValidator()
config, err := validator.ValidateDirectory("/path/to/project")
if err != nil {
    log.Printf("Validation failed: %v", err)
    return
}
fmt.Printf("Validated directory: %s (%d files)", config.Path, config.Metadata.FileCount)
```

### FileUtilities

Provides file system utility functions for path resolution, index management, and file operations.

```go
type FileUtilities struct {}
```

**Constructor:**
- `NewFileUtilities() *FileUtilities`

**Core Methods:**
- `ResolvePath(path string) (string, error)` - Resolves path to absolute with tilde expansion
- `GetIndexLocation(directory string) (string, error)` - Gets index file path for directory
- `CreateIndexLocation(baseDirectory string) *models.IndexLocation` - Creates index location object
- `AcquireLock(baseDirectory string) (*os.File, error)` - Acquires exclusive file lock
- `AcquireSharedLock(baseDirectory string) (*os.File, error)` - Acquires shared file lock
- `ReleaseLock(lockFile *os.File) error` - Releases file lock
- `IsLocked(baseDirectory string) bool` - Checks if directory is locked

**Directory Operations:**
- `EnsureDirectory(path string) error` - Creates directory if it doesn't exist
- `DirectoryExists(path string) bool` - Checks if directory exists
- `FileExists(path string) bool` - Checks if file exists
- `IsAccessible(path string) bool` - Checks if directory is readable
- `CanWrite(path string) bool` - Checks if directory is writable

**Path Operations:**
- `ValidatePathSecurity(path string, allowedBasePaths []string) error` - Validates path security
- `GetSafeRelativePath(basePath, targetPath string) (string, error)` - Gets relative path safely
- `GetDirectorySize(path string) (int64, int, error)` - Calculates directory size and file count

**Index Management:**
- `GetIndexInfo(directory string) (*models.IndexInfo, error)` - Gets index information
- `ListIndexedDirectories() ([]string, error)` - Lists directories with indexes
- `CleanupIndexFiles(indexLocation *models.IndexLocation) error` - Removes index files

**Utility Methods:**
- `GetFilePermissions(path string) (models.DirectoryPerms, error)` - Gets file permissions
- `FormatBytes(bytes int64) string` - Formats bytes into human-readable string

**Example:**
```go
fileUtils := lib.NewFileUtilities()

// Get index location
indexPath, err := fileUtils.GetIndexLocation("/path/to/project")
if err != nil {
    log.Printf("Failed to get index location: %v", err)
    return
}

// Acquire lock for indexing
lockFile, err := fileUtils.AcquireLock("/path/to/project")
if err != nil {
    log.Printf("Failed to acquire lock: %v", err)
    return
}
defer fileUtils.ReleaseLock(lockFile)

// Perform indexing operation...
```

### IndexMigrator

Handles migration of legacy index files to new format.

```go
type IndexMigrator struct {
    fileUtils *FileUtilities
}
```

**Constructor:**
- `NewIndexMigrator() *IndexMigrator`

**Core Methods:**
- `DetectLegacyIndexes(directory string) ([]string, error)` - Detects legacy index files
- `NeedsMigration(directory string) (bool, error)` - Checks if migration is needed
- `MigrateIndex(directory string, force bool) (*MigrationResult, error)` - Migrates legacy indexes
- `GetMigrationStatus(directory string) (string, error)` - Gets migration status
- `RollbackMigration(directory string) error` - Rolls back migration

**MigrationResult:**
```go
type MigrationResult struct {
    Success        bool     `json:"success"`         // Migration success status
    MigratedFiles  []string `json:"migrated_files"`  // List of migrated files
    Errors         []string `json:"errors"`          // Migration errors
    SourcePath     string   `json:"source_path"`     // Source directory
    TargetPath     string   `json:"target_path"`     // Target directory
    FilesMigrated  int      `json:"files_migrated"`  // Number of files migrated
    BytesMigrated  int64    `json:"bytes_migrated"`  // Number of bytes migrated
    Duration       string   `json:"duration"`        // Migration duration
}
```

**Example:**
```go
migrator := lib.NewIndexMigrator()

// Check if migration is needed
needsMigration, err := migrator.NeedsMigration("/path/to/project")
if err != nil {
    log.Printf("Failed to check migration need: %v", err)
    return
}

if needsMigration {
    result, err := migrator.MigrateIndex("/path/to/project", false)
    if err != nil {
        log.Printf("Migration failed: %v", err)
        return
    }
    log.Printf("Migration completed: %d files in %s", result.FilesMigrated, result.Duration)
}
```

### PerformanceOptimizer

Optimizes directory scanning and processing for large directories.

```go
type PerformanceOptimizer struct {
    workerPool    int
    batchSize     int
    maxMemoryMB   int64
    parallelWalk  bool
}
```

**Constructor:**
- `NewPerformanceOptimizer() *PerformanceOptimizer`

**Configuration Methods:**
- `OptimizeForLargeDirectory(estimatedFileCount int64)` - Adjusts parameters for large directories
- `GetMemoryUsage() (float64, float64)` - Returns current memory usage (alloc, system)

**Core Methods:**
- `FastDirectoryScan(directory string, options ScanOptions) (*ScanResult, error)` - Performs optimized directory scan
- `EstimateDirectorySize(directory string) (int64, int64, error)` - Quick size estimation

**ScanOptions:**
```go
type ScanOptions struct {
    BaseDir        string        // Base directory for relative paths
    MaxConcurrency int           // Maximum concurrent goroutines
    BatchSize      int           // Files per batch
    MaxDepth       int           // Maximum scan depth (0 = unlimited)
    FollowSymlinks bool          // Follow symbolic links
    FileTypes      []string      // File types to include
    ExcludeDirs    []string      // Directories to exclude
    MaxFileSize    int64         // Maximum file size
    Timeout        time.Duration // Operation timeout
}
```

**ScanResult:**
```go
type ScanResult struct {
    Files         []FileInfo `json:"files"`          // File information
    TotalFiles    int64       `json:"total_files"`    // Total file count
    TotalSize     int64       `json:"total_size"`     // Total size in bytes
    TotalDirs     int64       `json:"total_dirs"`     // Total directory count
    SkippedFiles  int64       `json:"skipped_files"`  // Number of skipped files
    SkippedDirs   int64       `json:"skipped_dirs"`   // Number of skipped directories
    Errors        []string    `json:"errors"`         // Scan errors
    Duration      string      `json:"duration"`       // Scan duration
    MemoryUsedMB  float64     `json:"memory_used_mb"` // Memory used in MB
}
```

**Factory Function:**
- `DefaultScanOptions() ScanOptions` - Returns default scan options

**Example:**
```go
optimizer := lib.NewPerformanceOptimizer()

// Optimize for large directory
optimizer.OptimizeForLargeDirectory(50000)

// Configure scan options
options := lib.DefaultScanOptions()
options.BaseDir = "/path/to/project"
options.ExcludeDirs = []string{".git", "node_modules", "build"}
options.MaxFileSize = 100 * 1024 * 1024 // 100MB

// Perform optimized scan
result, err := optimizer.FastDirectoryScan("/path/to/project", options)
if err != nil {
    log.Printf("Scan failed: %v", err)
    return
}

log.Printf("Scanned %d files (%d bytes) in %s", result.TotalFiles, result.TotalSize, result.Duration)
```

### BatchProcessor

Processes files in batches for memory efficiency.

```go
type BatchProcessor struct {
    batchSize   int
    processor   func([]FileInfo) error
    memoryLimit int64
}
```

**Constructor:**
- `NewBatchProcessor(batchSize int, processor func([]FileInfo) error) *BatchProcessor`

**Methods:**
- `ProcessFiles(files []FileInfo) error` - Processes files in batches

**Example:**
```go
processor := lib.NewBatchProcessor(1000, func(batch []lib.FileInfo) error {
    for _, file := range batch {
        // Process file
        fmt.Printf("Processing: %s (%d bytes)\n", file.Path, file.Size)
    }
    return nil
})

err := processor.ProcessFiles(files)
if err != nil {
    log.Printf("Batch processing failed: %v", err)
}
```

## CLI Commands

### IndexCommand

Handles directory indexing operations.

```go
type IndexCommand struct {
    indexingService *services.IndexingService
    logger          *services.DefaultLogger
    validator       *lib.DirectoryValidator
}
```

**Constructor:**
- `NewIndexCommand() *IndexCommand`

**Methods:**
- `Execute(args []string) error` - Executes index command
- `GetHelp() string` - Returns help text

**IndexOptions:**
```go
type IndexOptions struct {
    force           bool
    includeHidden   bool
    fileTypes       []string
    excludePatterns []string
    maxFileSize     int64
    verbose         bool
    quiet           bool
    directory       string  // --dir flag value
}
```

### SearchCommand

Handles directory search operations.

```go
type SearchCommand struct {
    searchService *services.SearchService
    logger        services.Logger
    fileUtils     *lib.FileUtilities
}
```

**Constructor:**
- `NewSearchCommand() *SearchCommand`

**Methods:**
- `Execute(args []string) error` - Executes search command
- `GetHelp() string` - Returns help text

**SearchOptions:**
```go
type SearchOptions struct {
    maxResults  int
    filePattern string
    withContext bool
    force       bool
    format      string
    threshold   float64
    semantic    bool
    exact       bool
    fuzzy       bool
    directory   string  // --dir flag value
}
```

## Error Handling

### CLIError

Represents CLI errors with specific exit codes.

```go
type CLIError struct {
    Code    ExitCode `json:"code"`
    Message string   `json:"message"`
    Err     error    `json:"error,omitempty"`
}
```

**Exit Codes:**
```go
const (
    ExitCodeSuccess ExitCode = 0
    ExitCodeError   ExitCode = 1
    ExitCodeInvalid ExitCode = 2
    ExitCodeNotFound ExitCode = 3
)
```

**Error Constructors:**
- `NewInvalidArgumentError(message string, err error) *CLIError`
- `NewNotFoundError(message string, err error) *CLIError`
- `NewGeneralError(message string, err error) *CLIError`
- `NewDirectoryNotFoundError(path string) *CLIError`
- `NewPermissionDeniedError(path string, operation string) *CLIError`
- `NewDirectoryTooLargeError(path string, size string, limit string) *CLIError`
- `NewTooManyFilesError(path string, count int, limit int) *CLIError`
- `NewIndexNotFoundError(path string) *CLIError`
- `NewIndexCorruptedError(path string) *CLIError`
- `NewIndexLockedError(path string) *CLIError`
- `NewPathTraversalError(path string) *CLIError`

**Error Checkers:**
- `IsInvalidArgumentError(err error) bool`
- `IsNotFoundError(err error) bool`

## Constants and Defaults

### Default Limits
```go
const (
    DefaultMaxDirectorySize = 1024 * 1024 * 1024 // 1GB
    DefaultMaxFileCount     = 10000              // 10,000 files
    DefaultMaxFileSize      = 100 * 1024 * 1024  // 100MB
)
```

### Default Exclusions
```go
var DefaultExcludeDirs = []string{
    ".git",
    "node_modules",
    ".clindex",
    ".svn",
    "vendor",
    "build",
    "dist",
    "target",
}
```

### Index Structure
```go
const (
    IndexDirName    = ".clindex"
    IndexFileName   = "data.index"
    MetadataFileName = "metadata.json"
    LockFileName    = "lock"
)
```

## Usage Examples

### Basic Directory Validation
```go
validator := lib.NewDirectoryValidator()
config, err := validator.ValidateDirectory("/path/to/project")
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}

if config.Metadata.FileCount > config.Limits.MaxFileCount {
    return fmt.Errorf("too many files: %d > %d",
        config.Metadata.FileCount, config.Limits.MaxFileCount)
}
```

### Index Management
```go
fileUtils := lib.NewFileUtilities()

// Create index location
indexLoc := fileUtils.CreateIndexLocation("/path/to/project")

// Ensure index directory exists
err := fileUtils.EnsureDirectory(indexLoc.IndexDir)
if err != nil {
    return fmt.Errorf("failed to create index directory: %w", err)
}

// Get index info
info, err := fileUtils.GetIndexInfo("/path/to/project")
if err != nil {
    return fmt.Errorf("failed to get index info: %w", err)
}

if !info.Exists {
    return fmt.Errorf("no index exists for directory")
}
```

### File Locking
```go
fileUtils := lib.NewFileUtilities()

// Acquire exclusive lock (for indexing)
lockFile, err := fileUtils.AcquireLock("/path/to/project")
if err != nil {
    return fmt.Errorf("failed to acquire lock: %w", err)
}
defer fileUtils.ReleaseLock(lockFile)

// Perform indexed operation...

// Acquire shared lock (for searching)
searchLock, err := fileUtils.AcquireSharedLock("/path/to/project")
if err != nil {
    return fmt.Errorf("failed to acquire search lock: %w", err)
}
defer fileUtils.ReleaseLock(searchLock)

// Perform search operation...
```

### Performance Optimization
```go
optimizer := lib.NewPerformanceOptimizer()

// Estimate directory size
size, count, err := optimizer.EstimateDirectorySize("/path/to/large/project")
if err != nil {
    return fmt.Errorf("estimation failed: %w", err)
}

fmt.Printf("Estimated: %d files, %d bytes\n", count, size)

// Configure for large directory
optimizer.OptimizeForLargeDirectory(count)

// Perform optimized scan
options := lib.DefaultScanOptions()
options.BaseDir = "/path/to/large/project"
options.ExcludeDirs = append(options.ExcludeDirs, "build", "dist")
options.MaxConcurrency = runtime.NumCPU() * 2

result, err := optimizer.FastDirectoryScan("/path/to/large/project", options)
if err != nil {
    return fmt.Errorf("scan failed: %w", err)
}

fmt.Printf("Scanned %d files in %s using %.2f MB memory\n",
    result.TotalFiles, result.Duration, result.MemoryUsedMB)
```

### Index Migration
```go
migrator := lib.NewIndexMigrator()

// Check migration status
status, err := migrator.GetMigrationStatus("/path/to/project")
if err != nil {
    return fmt.Errorf("failed to get migration status: %w", err)
}

switch status {
case "legacy":
    fmt.Println("Legacy index detected, migrating...")
    result, err := migrator.MigrateIndex("/path/to/project", false)
    if err != nil {
        return fmt.Errorf("migration failed: %w", err)
    }
    fmt.Printf("Migration completed: %s\n", result.Duration)
case "migrated":
    fmt.Println("Index already migrated")
case "new":
    fmt.Println("Using new index format")
default:
    fmt.Println("No index found")
}
```

## Testing Utilities

### Test Directory Creation
```go
func createTestDirectory(t *testing.T, files map[string]string) string {
    tempDir := t.TempDir()

    for path, content := range files {
        fullPath := filepath.Join(tempDir, path)
        dir := filepath.Dir(fullPath)
        if err := os.MkdirAll(dir, 0755); err != nil {
            t.Fatalf("Failed to create directory: %v", err)
        }
        if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
            t.Fatalf("Failed to create file: %v", err)
        }
    }

    return tempDir
}
```

### Mock Services
```go
type MockFileUtils struct {
    // Mock implementation fields
}

func (m *MockFileUtils) ResolvePath(path string) (string, error) {
    // Mock implementation
    return filepath.Abs(path)
}

func (m *MockFileUtils) GetIndexLocation(directory string) (string, error) {
    // Mock implementation
    return filepath.Join(directory, ".clindex", "index.db"), nil
}

// Implement other methods as needed for testing
```

## Performance Considerations

### Memory Management
- Large directories are processed in batches to control memory usage
- Garbage collection is triggered when memory limits are exceeded
- Parallel processing is balanced against memory consumption

### Concurrency
- File locking prevents concurrent access conflicts
- Worker pools limit concurrent goroutines
- Context-based cancellation for timeout handling

### I/O Optimization
- Sequential walks for small directories
- Parallel walks for large directories
- Batched file operations to reduce system calls

## Integration Points

### Services Integration
The directory selection feature integrates with existing services:

- **IndexingService**: Uses directory validation and index location management
- **SearchService**: Uses index location resolution for directory-specific searches
- **FileSystemService**: Enhanced with directory-specific operations

### CLI Integration
Commands are extended with `--dir` flag support:

- **index command**: `code-search index --dir <directory>`
- **search command**: `code-search search <query> --dir <directory>`

### Configuration Integration
Directory-specific settings can be configured through:

- Command-line flags
- Environment variables
- Configuration files
- Programmatic API calls