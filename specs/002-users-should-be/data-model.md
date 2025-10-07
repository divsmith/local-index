# Data Model: Optional Directory Selection for Indexing and Searching

**Feature**: Optional Directory Selection for Indexing and Searching
**Date**: 2025-10-06

## Core Entities

### DirectoryConfig
Configuration for directory selection and validation.

```go
type DirectoryConfig struct {
    Path         string            `json:"path"`          // Resolved absolute path
    OriginalPath string            `json:"original_path"` // User-provided path (may be relative)
    IsDefault    bool              `json:"is_default"`    // Whether this is the default directory
    Permissions  DirectoryPerms    `json:"permissions"`   // Directory access permissions
    Limits       DirectoryLimits   `json:"limits"`        // Size and count limits
    Metadata     DirectoryMetadata `json:"metadata"`      // Directory information
}
```

### DirectoryPerms
Directory permission and access validation.

```go
type DirectoryPerms struct {
    CanRead  bool `json:"can_read"`   // Read permission for indexing
    CanWrite bool `json:"can_write"`  // Write permission for index files
    CanExec  bool `json:"can_exec"`   // Execute permission for directory traversal
}
```

### DirectoryLimits
Configurable limits for directory operations.

```go
type DirectoryLimits struct {
    MaxDirectorySize int64 `json:"max_directory_size"` // Maximum total size in bytes
    MaxFileCount     int   `json:"max_file_count"`     // Maximum number of files
    MaxFileSize      int64 `json:"max_file_size"`      // Maximum individual file size
}
```

### DirectoryMetadata
Information about the target directory.

```go
type DirectoryMetadata struct {
    FileCount       int64     `json:"file_count"`        // Total files in directory
    TotalSize       int64     `json:"total_size"`        // Total size in bytes
    LastIndexed     time.Time `json:"last_indexed"`      // When directory was last indexed
    IndexVersion    string    `json:"index_version"`     // Index format version
    CreatedAt       time.Time `json:"created_at"`        // Directory creation time
    ModifiedAt      time.Time `json:"modified_at"`       // Last modification time
}
```

### IndexLocation
Information about where index files are stored.

```go
type IndexLocation struct {
    BaseDirectory string    `json:"base_directory"` // Directory containing source content
    IndexDir      string    `json:"index_dir"`      // .clindex subdirectory path
    MetadataFile  string    `json:"metadata_file"`  // metadata.json path
    DataFile      string    `json:"data_file"`      // data.index path
    LockFile      string    `json:"lock_file"`      // lock file path
}
```

## Validation Rules

### Directory Validation
1. **Path Resolution**: Convert relative paths to absolute paths using `filepath.Abs()`
2. **Existence Check**: Directory must exist and be accessible
3. **Permission Check**: Verify read/write permissions as required by operation
4. **Symlink Safety**: Only follow symlinks within the same directory tree
5. **Path Traversal**: Ensure resolved path stays within expected bounds

### Size Limit Validation
1. **Directory Size**: Check against `MaxDirectorySize` limit
2. **File Count**: Verify file count doesn't exceed `MaxFileCount`
3. **File Size**: Individual files must not exceed `MaxFileSize`

### Index File Validation
1. **Index Existence**: Verify index files exist for search operations
2. **Index Integrity**: Validate index file format and checksums
3. **Version Compatibility**: Ensure index version matches current implementation

## State Transitions

### Directory Selection Flow
```
User Input → Path Resolution → Validation → Config Creation → Operation
    ↓              ↓                ↓            ↓           ↓
  (raw path) → (abs path) → (perms/limits) → (struct) → (index/search)
```

### Index Operation Flow
```
DirectoryConfig → IndexLocation → Lock → Process → Unlock → Result
       ↓               ↓           ↓        ↓        ↓        ↓
   (validated)     (computed)   (file)   (work)   (file)  (output)
```

## Error States

### Validation Errors
- `ErrDirectoryNotFound`: Directory does not exist
- `ErrPermissionDenied`: Insufficient permissions
- `ErrDirectoryTooLarge`: Exceeds size limits
- `ErrTooManyFiles`: Exceeds file count limits
- `ErrPathTraversal`: Path attempts to escape allowed bounds

### Index Errors
- `ErrIndexNotFound`: No index files found in directory
- `ErrIndexCorrupted`: Index files are invalid or corrupted
- `ErrIndexLocked`: Index is currently being used by another process
- `ErrIndexVersionMismatch`: Index version incompatible with current implementation

## Configuration Management

### Default Configuration
When no directory is specified, use current working directory:
```go
defaultConfig := &DirectoryConfig{
    Path:         getCurrentWorkingDirectory(),
    OriginalPath: ".",
    IsDefault:    true,
    Limits:       getDefaultLimits(),
}
```

### User-Specified Configuration
When user provides directory path:
```go
userConfig := &DirectoryConfig{
    Path:         resolveAbsolutePath(userPath),
    OriginalPath: userPath,
    IsDefault:    false,
    Limits:       getUserLimits(),
}
```

---

**Data model complete. Ready for contract generation.**