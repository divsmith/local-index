package models

import (
	"encoding/json"
	"path/filepath"
	"time"
)

// DirectoryConfig represents validated directory configuration
type DirectoryConfig struct {
	Path         string            `json:"path"`
	OriginalPath string            `json:"original_path"`
	IsDefault    bool              `json:"is_default"`
	Permissions  DirectoryPerms    `json:"permissions"`
	Limits       DirectoryLimits   `json:"limits"`
	Metadata     DirectoryMetadata `json:"metadata"`
}

// DirectoryPerms represents directory permissions
type DirectoryPerms struct {
	CanRead  bool `json:"can_read"`
	CanWrite bool `json:"can_write"`
	CanExec  bool `json:"can_exec"`
}

// DirectoryLimits represents directory operation limits
type DirectoryLimits struct {
	MaxDirectorySize int64 `json:"max_directory_size"`
	MaxFileCount     int   `json:"max_file_count"`
	MaxFileSize      int64 `json:"max_file_size"`
}

// DirectoryMetadata represents directory information
type DirectoryMetadata struct {
	FileCount    int64     `json:"file_count"`
	TotalSize    int64     `json:"total_size"`
	LastIndexed  time.Time `json:"last_indexed"`
	IndexVersion string    `json:"index_version"`
	CreatedAt    time.Time `json:"created_at"`
	ModifiedAt   time.Time `json:"modified_at"`
	ScanDuration string    `json:"scan_duration,omitempty"`
	MemoryUsed   float64   `json:"memory_used_mb,omitempty"`
}

// IndexLocation represents information about where index files are stored
type IndexLocation struct {
	BaseDirectory string `json:"base_directory"`
	IndexDir      string `json:"index_dir"`
	MetadataFile  string `json:"metadata_file"`
	DataFile      string `json:"data_file"`
	IndexFile     string `json:"index_file"`
	LockFile      string `json:"lock_file"`
}

// NewIndexLocation creates a new IndexLocation for a given directory
func NewIndexLocation(baseDirectory string) *IndexLocation {
	indexDir := filepath.Join(baseDirectory, ".clindex")
	return &IndexLocation{
		BaseDirectory: baseDirectory,
		IndexDir:      indexDir,
		MetadataFile:  filepath.Join(indexDir, "metadata.json"),
		DataFile:      filepath.Join(indexDir, "data.index"),
		IndexFile:     filepath.Join(indexDir, "index.db"),
		LockFile:      filepath.Join(indexDir, "lock"),
	}
}

// NewDefaultDirectoryLimits returns default directory limits
func NewDefaultDirectoryLimits() *DirectoryLimits {
	return &DirectoryLimits{
		MaxDirectorySize: 1024 * 1024 * 1024, // 1GB
		MaxFileCount:     10000,              // 10,000 files
		MaxFileSize:      100 * 1024 * 1024,  // 100MB
	}
}

// NewDirectoryConfig creates a new DirectoryConfig with defaults
func NewDirectoryConfig(path, originalPath string, isDefault bool) *DirectoryConfig {
	return &DirectoryConfig{
		Path:         path,
		OriginalPath: originalPath,
		IsDefault:    isDefault,
		Permissions:  DirectoryPerms{},
		Limits:       *NewDefaultDirectoryLimits(),
		Metadata:     DirectoryMetadata{},
	}
}

// ToJSON converts DirectoryConfig to JSON
func (dc *DirectoryConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(dc, "", "  ")
}

// FromJSON loads DirectoryConfig from JSON
func (dc *DirectoryConfig) FromJSON(data []byte) error {
	return json.Unmarshal(data, dc)
}

// ToJSON converts DirectoryMetadata to JSON
func (dm *DirectoryMetadata) ToJSON() ([]byte, error) {
	return json.MarshalIndent(dm, "", "  ")
}

// FromJSON loads DirectoryMetadata from JSON
func (dm *DirectoryMetadata) FromJSON(data []byte) error {
	return json.Unmarshal(data, dm)
}

// IsIndexValid checks if the index is still valid based on modification times
func (dm *DirectoryMetadata) IsIndexValid(dirModified time.Time) bool {
	return dm.LastIndexed.After(dirModified)
}

// MarkIndexed updates the metadata to reflect that the directory has been indexed
func (dm *DirectoryMetadata) MarkIndexed() {
	dm.LastIndexed = time.Now()
	dm.IndexVersion = "1.0.0"
}

// IndexingOptions contains options for the indexing process
type IndexingOptions struct {
	IncludeHidden     bool          `json:"include_hidden"`
	FileTypes         []string      `json:"file_types"`
	ExcludePatterns   []string      `json:"exclude_patterns"`
	MaxFileSize       int64         `json:"max_file_size"`
	ChunkSize         int           `json:"chunk_size"`
	ChunkOverlap      int           `json:"chunk_overlap"`
	MaxConcurrency    int           `json:"max_concurrency"`
	Timeout           time.Duration `json:"timeout"`
	EnableIncremental bool          `json:"enable_incremental"`
}

// FileStats contains file statistics
type FileStats struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time"`
	IsDir        bool      `json:"is_dir"`
	Language     string    `json:"language"`
}

// IndexInfo contains information about an index
type IndexInfo struct {
	Directory string    `json:"directory"`
	IndexPath string    `json:"index_path"`
	Exists    bool      `json:"exists"`
	Locked    bool      `json:"locked"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
}

// IndexMetadata contains metadata about an index
type IndexMetadata struct {
	Version        string    `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	ModifiedAt     time.Time `json:"modified_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Migrated       bool      `json:"migrated"`
	MigrationDate  time.Time `json:"migration_date"`
	LegacyFiles    []string  `json:"legacy_files"`
	FileCount      int       `json:"file_count"`
	ChunkCount     int       `json:"chunk_count"`
	IndexedSize    int64     `json:"indexed_size"`
	IndexPath      string    `json:"index_path"`
	Directory      string    `json:"directory"`
	IndexerType    string    `json:"indexer_type"`
	LegacyVersion  string    `json:"legacy_version"`
}