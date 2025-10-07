# Research Document: Optional Directory Selection for Indexing and Searching

**Feature**: Optional Directory Selection for Indexing and Searching
**Date**: 2025-10-06
**Scope**: Technical research and best practices for CLI directory handling

## Directory Path Handling Best Practices

### Decision: Use Go's `filepath` package for cross-platform compatibility
**Rationale**: The `filepath` package provides cross-platform path handling, automatically handling path separators (`/` vs `\`) and resolving relative paths correctly on different operating systems.

**Alternatives considered**:
- Manual string manipulation: Rejected due to platform-specific issues
- `path` package: Rejected as it's URI-focused, not filesystem-focused

### Decision: Implement absolute path resolution with user-friendly error messages
**Rationale**: Users may provide relative paths (e.g., `.` or `../src`) which should be resolved to absolute paths for consistency in index file storage and validation.

**Implementation approach**:
- Use `filepath.Abs()` to resolve relative paths
- Store resolved absolute paths in configuration
- Display resolved paths in user feedback

## Index File Storage Strategy

### Decision: Store index files in `.clindex` subdirectory within target directory
**Rationale**: Placing index files directly in the target directory could conflict with user files. A hidden subdirectory keeps index files organized and separate from user content.

**Alternatives considered**:
- Direct storage in target directory: Rejected due to potential file conflicts
- Global index directory: Rejected as it separates index from content, violating the requirement
- User home directory: Rejected as it makes directory management complex

### File naming convention:
- Index metadata: `.clindex/metadata.json`
- Index data: `.clindex/data.index`
- Lock file: `.clindex/lock` (for concurrent access protection)

## Permission and Validation Strategy

### Decision: Implement comprehensive permission checking before operations
**Rationale**: Early validation prevents operations from failing partway through, providing better user experience.

**Validation checks**:
1. Directory existence verification
2. Read permission for indexing operations
3. Write permission for index file creation
4. Sufficient disk space estimation
5. Directory traversal protection (symlink handling)

### Decision: Follow symlinks only within the same directory tree
**Rationale**: Prevents infinite loops and unauthorized access while still allowing legitimate symlink usage within the project scope.

## Performance Considerations

### Decision: Implement configurable size limits with sensible defaults
**Rationale**: Prevents accidental indexing of massive directories while allowing power users to adjust limits as needed.

**Default limits**:
- Maximum directory size: 1GB
- Maximum file count: 10,000 files
- Maximum individual file size: 100MB

### Decision: Use streaming file processing for large directories
**Rationale**: Reduces memory footprint during indexing operations.

## CLI Interface Design

### Decision: Use optional flag-based directory selection
**Rationale**: Maintains backward compatibility while providing the new functionality.

**Interface design**:
- Index command: `code-search index [--dir PATH]`
- Search command: `code-search search [--dir PATH] QUERY`
- Default behavior: Use current working directory when `--dir` not specified

### Decision: Provide clear, actionable error messages
**Rationale**: CLI tools need to be helpful and guide users toward correct usage.

## Backward Compatibility Strategy

### Decision: Maintain current directory as default when no directory specified
**Rationale**: Ensures existing workflows continue to work without modification.

**Compatibility approach**:
- Current working directory remains the default target
- All existing CLI flags and behaviors preserved
- New functionality is opt-in via new flags

## Security Considerations

### Decision: Implement path sanitization and validation
**Rationale**: Prevents path traversal attacks and ensures only intended directories are accessed.

**Security measures**:
1. Path canonicalization before processing
2. Validation against allowed base directories
3. Protection against symbolic link attacks
4. Permission verification before operations

---

**All research complete. No NEEDS CLARIFICATION items remain.**