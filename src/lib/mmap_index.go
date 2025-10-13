package lib

import (
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// MemoryMappedIndex provides memory-mapped access to index files
type MemoryMappedIndex struct {
	file       *os.File
	data       []byte
	size       int64
	mapped     bool
	readOnly   bool
	mu         sync.RWMutex
	segments   map[string]*MemorySegment
	header     *IndexHeader
}

// MemorySegment represents a mapped segment of the index
type MemorySegment struct {
	data       []byte
	offset     int64
	size       int64
	mu         sync.RWMutex
	loaded     bool
	prefetch   bool
}

// MMapOptions contains options for memory mapping
type MMapOptions struct {
	ReadOnly    bool          `json:"read_only"`
	Advise      int           `json:"advise"`      // MADV_* flags
	Prefetch    bool          `json:"prefetch"`
	SegmentSize int64         `json:"segment_size"`
	MaxSegments int           `json:"max_segments"`
}

// DefaultMMapOptions returns default options for memory mapping
func DefaultMMapOptions() MMapOptions {
	return MMapOptions{
		ReadOnly:    true,
		Advise:      unix.MADV_RANDOM,
		Prefetch:    true,
		SegmentSize: 1024 * 1024, // 1MB segments
		MaxSegments: 100,
	}
}

// NewMemoryMappedIndex creates a new memory-mapped index
func NewMemoryMappedIndex(filePath string, options MMapOptions) (*MemoryMappedIndex, error) {
	// Open file
	flags := os.O_RDONLY
	if !options.ReadOnly {
		flags = os.O_RDWR
	}

	file, err := os.OpenFile(filePath, flags, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Get file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	size := info.Size()
	if size == 0 {
		file.Close()
		return nil, fmt.Errorf("empty file")
	}

	// Memory map the file
	data, err := unix.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to mmap file: %w", err)
	}

	// Apply memory advice if specified
	if options.Advise != 0 {
		unix.Madvise(data, unix.MADV_RANDOM)
	}

	// Parse header
	header, err := parseMappedHeader(data)
	if err != nil {
		unix.Munmap(data)
		file.Close()
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Create index
	mmapIndex := &MemoryMappedIndex{
		file:     file,
		data:     data,
		size:     size,
		mapped:   true,
		readOnly: options.ReadOnly,
		segments: make(map[string]*MemorySegment),
		header:   header,
	}

	// Create segments if prefetching is enabled
	if options.Prefetch {
		mmapIndex.createSegments(options)
	}

	return mmapIndex, nil
}

// parseMappedHeader parses the header from mapped data
func parseMappedHeader(data []byte) (*IndexHeader, error) {
	if len(data) < HeaderSize+IndexHeaderSize {
		return nil, fmt.Errorf("data too short for headers")
	}

	// Parse file header to get to index header
	offset := HeaderSize
	var header IndexHeader

	header.VectorCount = binary.LittleEndian.Uint64(data[offset : offset+8])
	header.FileCount = binary.LittleEndian.Uint64(data[offset+8 : offset+16])
	header.ChunkCount = binary.LittleEndian.Uint64(data[offset+16 : offset+24])
	header.VectorOffset = binary.LittleEndian.Uint64(data[offset+24 : offset+32])
	header.FileOffset = binary.LittleEndian.Uint64(data[offset+32 : offset+40])
	header.ChunkOffset = binary.LittleEndian.Uint64(data[offset+40 : offset+48])

	return &header, nil
}

// createSegments creates memory segments for different parts of the index
func (mmi *MemoryMappedIndex) createSegments(options MMapOptions) {
	mmi.mu.Lock()
	defer mmi.mu.Unlock()

	// Vector segment
	if mmi.header.VectorCount > 0 {
		vectorSize := int64(mmi.header.FileOffset - mmi.header.VectorOffset)
		mmi.segments["vectors"] = &MemorySegment{
			data:     mmi.data[mmi.header.VectorOffset:mmi.header.FileOffset],
			offset:   int64(mmi.header.VectorOffset),
			size:     vectorSize,
			loaded:   true,
			prefetch: true,
		}
	}

	// File segment
	if mmi.header.FileCount > 0 {
		fileSize := int64(mmi.header.ChunkOffset - mmi.header.FileOffset)
		mmi.segments["files"] = &MemorySegment{
			data:     mmi.data[mmi.header.FileOffset:mmi.header.ChunkOffset],
			offset:   int64(mmi.header.FileOffset),
			size:     fileSize,
			loaded:   true,
			prefetch: true,
		}
	}

	// Chunk segment
	if mmi.header.ChunkCount > 0 {
		chunkSize := mmi.size - int64(mmi.header.ChunkOffset)
		mmi.segments["chunks"] = &MemorySegment{
			data:     mmi.data[mmi.header.ChunkOffset:],
			offset:   int64(mmi.header.ChunkOffset),
			size:     chunkSize,
			loaded:   true,
			prefetch: true,
		}
	}
}

// GetSegment returns a memory segment by name
func (mmi *MemoryMappedIndex) GetSegment(name string) *MemorySegment {
	mmi.mu.RLock()
	defer mmi.mu.RUnlock()
	return mmi.segments[name]
}

// GetVectors returns the vector segment data
func (mmi *MemoryMappedIndex) GetVectors() []byte {
	segment := mmi.GetSegment("vectors")
	if segment != nil {
		segment.mu.RLock()
		defer segment.mu.RUnlock()
		return segment.data
	}

	// Return slice from main data if no segment
	if mmi.header.VectorOffset < mmi.header.FileOffset {
		return mmi.data[mmi.header.VectorOffset:mmi.header.FileOffset]
	}
	return nil
}

// GetFiles returns the file segment data
func (mmi *MemoryMappedIndex) GetFiles() []byte {
	segment := mmi.GetSegment("files")
	if segment != nil {
		segment.mu.RLock()
		defer segment.mu.RUnlock()
		return segment.data
	}

	// Return slice from main data if no segment
	if mmi.header.FileOffset < mmi.header.ChunkOffset {
		return mmi.data[mmi.header.FileOffset:mmi.header.ChunkOffset]
	}
	return nil
}

// GetChunks returns the chunk segment data
func (mmi *MemoryMappedIndex) GetChunks() []byte {
	segment := mmi.GetSegment("chunks")
	if segment != nil {
		segment.mu.RLock()
		defer segment.mu.RUnlock()
		return segment.data
	}

	// Return slice from main data if no segment
	if int64(mmi.header.ChunkOffset) < mmi.size {
		return mmi.data[mmi.header.ChunkOffset:]
	}
	return nil
}

// PrefetchSegment prefetches a segment into memory
func (mmi *MemoryMappedIndex) PrefetchSegment(name string) error {
	mmi.mu.Lock()
	defer mmi.mu.Unlock()

	segment := mmi.segments[name]
	if segment == nil || segment.prefetch {
		return nil
	}

	// Use madvise to prefetch
	if len(segment.data) > 0 {
		err := unix.Madvise(segment.data, unix.MADV_WILLNEED)
		if err != nil {
			return fmt.Errorf("failed to prefetch segment %s: %w", name, err)
		}
		segment.prefetch = true
	}

	return nil
}

// ReadVector reads a vector at a specific offset
func (mmi *MemoryMappedIndex) ReadVector(offset int64, size int) ([]byte, error) {
	if offset < 0 || size <= 0 {
		return nil, fmt.Errorf("invalid offset or size")
	}

	vectorData := mmi.GetVectors()
	if vectorData == nil {
		return nil, fmt.Errorf("no vector data available")
	}

	if offset+int64(size) > int64(len(vectorData)) {
		return nil, fmt.Errorf("read beyond vector data bounds")
	}

	return vectorData[offset : offset+int64(size)], nil
}

// ReadFile reads file data at a specific offset
func (mmi *MemoryMappedIndex) ReadFile(offset int64, size int) ([]byte, error) {
	if offset < 0 || size <= 0 {
		return nil, fmt.Errorf("invalid offset or size")
	}

	fileData := mmi.GetFiles()
	if fileData == nil {
		return nil, fmt.Errorf("no file data available")
	}

	if offset+int64(size) > int64(len(fileData)) {
		return nil, fmt.Errorf("read beyond file data bounds")
	}

	return fileData[offset : offset+int64(size)], nil
}

// ReadChunk reads chunk data at a specific offset
func (mmi *MemoryMappedIndex) ReadChunk(offset int64, size int) ([]byte, error) {
	if offset < 0 || size <= 0 {
		return nil, fmt.Errorf("invalid offset or size")
	}

	chunkData := mmi.GetChunks()
	if chunkData == nil {
		return nil, fmt.Errorf("no chunk data available")
	}

	if offset+int64(size) > int64(len(chunkData)) {
		return nil, fmt.Errorf("read beyond chunk data bounds")
	}

	return chunkData[offset : offset+int64(size)], nil
}

// GetHeader returns the parsed index header
func (mmi *MemoryMappedIndex) GetHeader() *IndexHeader {
	mmi.mu.RLock()
	defer mmi.mu.RUnlock()
	return mmi.header
}

// Size returns the size of the mapped file
func (mmi *MemoryMappedIndex) Size() int64 {
	mmi.mu.RLock()
	defer mmi.mu.RUnlock()
	return mmi.size
}

// IsReadOnly returns whether the mapping is read-only
func (mmi *MemoryMappedIndex) IsReadOnly() bool {
	mmi.mu.RLock()
	defer mmi.mu.RUnlock()
	return mmi.readOnly
}

// Sync synchronizes the mapping with the file (for writable mappings)
func (mmi *MemoryMappedIndex) Sync() error {
	if mmi.readOnly {
		return fmt.Errorf("cannot sync read-only mapping")
	}

	return unix.Msync(mmi.data, unix.MS_SYNC)
}

// Advise provides memory advice for the mapping
func (mmi *MemoryMappedIndex) Advise(advice int) error {
	return unix.Madvise(mmi.data, advice)
}

// Close unmaps the memory and closes the file
func (mmi *MemoryMappedIndex) Close() error {
	mmi.mu.Lock()
	defer mmi.mu.Unlock()

	if !mmi.mapped {
		return nil
	}

	var err error
	if mmi.data != nil {
		err = unix.Munmap(mmi.data)
		mmi.data = nil
	}

	if mmi.file != nil {
		fileErr := mmi.file.Close()
		if err == nil {
			err = fileErr
		}
		mmi.file = nil
	}

	mmi.mapped = false
	mmi.segments = nil

	return err
}

// Remap remaps the file with a new size (for writable mappings)
func (mmi *MemoryMappedIndex) Remap(newSize int64) error {
	if mmi.readOnly {
		return fmt.Errorf("cannot remap read-only mapping")
	}

	mmi.mu.Lock()
	defer mmi.mu.Unlock()

	// Unmap current mapping
	if mmi.data != nil {
		if err := unix.Munmap(mmi.data); err != nil {
			return fmt.Errorf("failed to unmap: %w", err)
		}
	}

	// Remap with new size
	data, err := unix.Mmap(int(mmi.file.Fd()), 0, int(newSize), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("failed to mmap: %w", err)
	}

	mmi.data = data
	mmi.size = newSize

	// Re-parse header
	header, err := parseMappedHeader(data)
	if err != nil {
		unix.Munmap(data)
		return fmt.Errorf("failed to parse header: %w", err)
	}
	mmi.header = header

	return nil
}

// GetMemoryUsage returns memory usage statistics
func (mmi *MemoryMappedIndex) GetMemoryUsage() MemoryUsage {
	mmi.mu.RLock()
	defer mmi.mu.RUnlock()

	usage := MemoryUsage{
		MappedSize: mmi.size,
		Segments:   make(map[string]int64),
	}

	for name, segment := range mmi.segments {
		usage.Segments[name] = segment.size
	}

	return usage
}

// MemoryUsage contains memory usage statistics
type MemoryUsage struct {
	MappedSize int64            `json:"mapped_size"`
	Segments   map[string]int64 `json:"segments"`
}

// NewLazyMappedIndex creates a lazy-loaded memory-mapped index
func NewLazyMappedIndex(filePath string) (*LazyMappedIndex, error) {
	return &LazyMappedIndex{
		filePath: filePath,
		options:  DefaultMMapOptions(),
		loaded:   false,
	}, nil
}

// LazyMappedIndex provides lazy loading of memory-mapped indexes
type LazyMappedIndex struct {
	filePath string
	options  MMapOptions
	index    *MemoryMappedIndex
	loaded   bool
	mu       sync.RWMutex
}

// Load loads the index if not already loaded
func (lmi *LazyMappedIndex) Load() error {
	lmi.mu.Lock()
	defer lmi.mu.Unlock()

	if lmi.loaded {
		return nil
	}

	index, err := NewMemoryMappedIndex(lmi.filePath, lmi.options)
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	lmi.index = index
	lmi.loaded = true

	return nil
}

// GetIndex returns the underlying memory-mapped index
func (lmi *LazyMappedIndex) GetIndex() (*MemoryMappedIndex, error) {
	if !lmi.loaded {
		if err := lmi.Load(); err != nil {
			return nil, err
		}
	}

	return lmi.index, nil
}

// Close closes the lazy index
func (lmi *LazyMappedIndex) Close() error {
	lmi.mu.Lock()
	defer lmi.mu.Unlock()

	if lmi.index != nil {
		err := lmi.index.Close()
		lmi.index = nil
		lmi.loaded = false
		return err
	}

	return nil
}

// Helper function to get slice from unsafe pointer (for performance)
func unsafeSlice(data []byte, offset, length int64) []byte {
	if offset < 0 || length < 0 {
		return nil
	}

	var slice []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	sh.Data = uintptr(unsafe.Pointer(&data[0])) + uintptr(offset)
	sh.Len = int(length)
	sh.Cap = int(length)

	return slice
}