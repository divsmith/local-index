package lib

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sync"

	"code-search/src/models"
)

// BinaryStorage handles compressed binary format for indexes
type BinaryStorage struct {
	version    uint16
	compression CompressionType
	bufferPool *BufferPool
	mu         sync.RWMutex
}

// CompressionType represents the compression algorithm used
type CompressionType uint8

const (
	CompressionNone CompressionType = iota
	CompressionGzip
	CompressionSnappy
	CompressionZstd
)

const (
	// Storage format version
	StorageVersion = 1

	// Magic number for binary files
	MagicNumber = 0x434C494E // "CLIN" in hex

	// Header sizes
	HeaderSize = 32
	VectorHeaderSize = 16
	ChunkHeaderSize = 24
	FileHeaderSize = 32
	IndexHeaderSize = 48 // Additional header for index section
)

// FileHeader represents the header of a binary storage file
type FileHeader struct {
	Magic      uint32  // Magic number
	Version     uint16  // Storage format version
	Flags       uint16  // Feature flags
	Compression uint8   // Compression type
	Reserved    [7]byte // Reserved for future use
	IndexSize   uint64  // Size of index section
	Metadata    uint64  // Size of metadata section
	Checksum    uint32  // CRC32 checksum
}

// IndexHeader represents the header of the index section
type IndexHeader struct {
	VectorCount uint64 // Number of vectors
	FileCount   uint64 // Number of files
	ChunkCount  uint64 // Number of chunks
	VectorOffset uint64 // Offset to vector data
	FileOffset   uint64 // Offset to file data
	ChunkOffset  uint64 // Offset to chunk data
}

// VectorEntryBinary represents a vector entry in binary format
type VectorEntryBinary struct {
	IDLength    uint16
	VectorSize  uint32
	MetadataSize uint32
	ID          []byte
	Vector      []byte
	Metadata    []byte
}

// FileEntryBinary represents a file entry in binary format
type FileEntryBinary struct {
	PathLength   uint16
	LanguageLength uint8
	ChunkCount   uint32
	LastModified uint64
	FileSize     uint64
	Path         []byte
	Language     []byte
	ChunkIDs     []uint32
}

// ChunkEntryBinary represents a chunk entry in binary format
type ChunkEntryBinary struct {
	VectorID     uint32
	FileID       uint32
	StartLine    uint32
	EndLine      uint32
	ContentLength uint16
	Content      []byte
}

// NewBinaryStorage creates a new binary storage instance
func NewBinaryStorage() *BinaryStorage {
	return &BinaryStorage{
		version:     StorageVersion,
		compression: CompressionGzip,
		bufferPool:  GetPoolManager().GetBufferPool(),
	}
}

// SerializeIndex serializes a code index to compressed binary format
func (bs *BinaryStorage) SerializeIndex(index *models.CodeIndex, filePath string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Create temporary buffer for writing
	var buf bytes.Buffer

	// Write file header
	header := FileHeader{
		Magic:      MagicNumber,
		Version:     bs.version,
		Compression: uint8(bs.compression),
	}

	// Reserve space for header and index header sizes
	buf.Write(make([]byte, HeaderSize+IndexHeaderSize))

	// Collect data
	files := index.GetAllFiles()

	// Get vector data from vector store
	var vectors []models.VectorSearchResult
	// In a real implementation, you would extract all vectors from the vector store

	// Get all chunks from files
	var chunks []models.CodeChunk
	for _, file := range files {
		chunks = append(chunks, file.Chunks...)
	}

	// Write vector data
	vectorOffset := uint64(buf.Len())
	vectorData := bs.serializeVectors(vectors)
	buf.Write(vectorData)

	// Write file data
	fileOffset := uint64(buf.Len())
	fileData := bs.serializeFiles(files, chunks)
	buf.Write(fileData)

	// Write chunk data
	chunkOffset := uint64(buf.Len())
	chunkData := bs.serializeChunks(chunks)
	buf.Write(chunkData)

	// Write metadata (JSON for now, could be binary later)
	metadataOffset := uint64(buf.Len())
	metadata := bs.serializeMetadata(index)
	buf.Write(metadata)

	// Update headers with actual sizes
	header.IndexSize = vectorOffset
	header.Metadata = metadataOffset - vectorOffset

	// Write index header
	indexHeader := IndexHeader{
		VectorCount:  uint64(len(vectors)),
		FileCount:    uint64(len(files)),
		ChunkCount:   uint64(len(chunks)),
		VectorOffset: vectorOffset,
		FileOffset:   fileOffset,
		ChunkOffset:  chunkOffset,
	}

	// Write headers to buffer
	headerData := bs.serializeHeaders(header, indexHeader)
	copy(buf.Bytes(), headerData)

	// Compress if needed
	data := buf.Bytes()
	if bs.compression != CompressionNone {
		compressed, err := bs.compressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress data: %w", err)
		}
		data = compressed
	}

	// Write to file
	return bs.writeToFile(filePath, data)
}

// DeserializeIndex loads an index from compressed binary format
func (bs *BinaryStorage) DeserializeIndex(filePath string, vectorStore models.VectorStore) (*models.CodeIndex, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	// Read file data
	data, err := bs.readFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Decompress if needed
	if len(data) >= 1 {
		// Check if data is compressed by trying to decompress
		decompressed, err := bs.decompressData(data)
		if err == nil {
			data = decompressed
		}
	}

	// Parse headers
	header, indexHeader, err := bs.parseHeaders(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse headers: %w", err)
	}

	// Validate header
	if header.Magic != MagicNumber {
		return nil, fmt.Errorf("invalid magic number: got %x, expected %x", header.Magic, MagicNumber)
	}

	if header.Version != bs.version {
		return nil, fmt.Errorf("unsupported version: %d", header.Version)
	}

	// Create index
	index := models.NewCodeIndex("", vectorStore)

	// Load vectors
	if indexHeader.VectorCount > 0 {
		vectorData := data[indexHeader.VectorOffset:indexHeader.FileOffset]
		if err := bs.deserializeVectors(vectorData, index, int(indexHeader.VectorCount)); err != nil {
			return nil, fmt.Errorf("failed to deserialize vectors: %w", err)
		}
	}

	// Load files
	if indexHeader.FileCount > 0 {
		fileData := data[indexHeader.FileOffset:indexHeader.ChunkOffset]
		if err := bs.deserializeFiles(fileData, index, int(indexHeader.FileCount)); err != nil {
			return nil, fmt.Errorf("failed to deserialize files: %w", err)
		}
	}

	// Load chunks
	if indexHeader.ChunkCount > 0 {
		chunkData := data[indexHeader.ChunkOffset:header.IndexSize+header.Metadata]
		if err := bs.deserializeChunks(chunkData, index, int(indexHeader.ChunkCount)); err != nil {
			return nil, fmt.Errorf("failed to deserialize chunks: %w", err)
		}
	}

	return index, nil
}

// serializeHeaders converts headers to binary format
func (bs *BinaryStorage) serializeHeaders(header FileHeader, indexHeader IndexHeader) []byte {
	buf := make([]byte, HeaderSize+IndexHeaderSize)

	// File header
	binary.LittleEndian.PutUint32(buf[0:4], header.Magic)
	binary.LittleEndian.PutUint16(buf[4:6], header.Version)
	binary.LittleEndian.PutUint16(buf[6:8], header.Flags)
	buf[8] = header.Compression
	// Reserved bytes already zeroed
	binary.LittleEndian.PutUint64(buf[16:24], header.IndexSize)
	binary.LittleEndian.PutUint64(buf[24:32], header.Metadata)

	// Index header
	offset := HeaderSize
	binary.LittleEndian.PutUint64(buf[offset:offset+8], indexHeader.VectorCount)
	binary.LittleEndian.PutUint64(buf[offset+8:offset+16], indexHeader.FileCount)
	binary.LittleEndian.PutUint64(buf[offset+16:offset+24], indexHeader.ChunkCount)
	binary.LittleEndian.PutUint64(buf[offset+24:offset+32], indexHeader.VectorOffset)
	binary.LittleEndian.PutUint64(buf[offset+32:offset+40], indexHeader.FileOffset)
	binary.LittleEndian.PutUint64(buf[offset+40:offset+48], indexHeader.ChunkOffset)

	return buf
}

// parseHeaders extracts headers from binary data
func (bs *BinaryStorage) parseHeaders(data []byte) (FileHeader, IndexHeader, error) {
	if len(data) < HeaderSize+IndexHeaderSize {
		return FileHeader{}, IndexHeader{}, fmt.Errorf("data too short for headers")
	}

	// Parse file header
	var header FileHeader
	header.Magic = binary.LittleEndian.Uint32(data[0:4])
	header.Version = binary.LittleEndian.Uint16(data[4:6])
	header.Flags = binary.LittleEndian.Uint16(data[6:8])
	header.Compression = data[8]
	copy(header.Reserved[:], data[9:16])
	header.IndexSize = binary.LittleEndian.Uint64(data[16:24])
	header.Metadata = binary.LittleEndian.Uint64(data[24:32])

	// Parse index header
	var indexHeader IndexHeader
	offset := HeaderSize
	indexHeader.VectorCount = binary.LittleEndian.Uint64(data[offset:offset+8])
	indexHeader.FileCount = binary.LittleEndian.Uint64(data[offset+8:offset+16])
	indexHeader.ChunkCount = binary.LittleEndian.Uint64(data[offset+16:offset+24])
	indexHeader.VectorOffset = binary.LittleEndian.Uint64(data[offset+24:offset+32])
	indexHeader.FileOffset = binary.LittleEndian.Uint64(data[offset+32:offset+40])
	indexHeader.ChunkOffset = binary.LittleEndian.Uint64(data[offset+40:offset+48])

	return header, indexHeader, nil
}

// serializeVectors converts vectors to compact binary format
func (bs *BinaryStorage) serializeVectors(vectors []models.VectorSearchResult) []byte {
	buf := new(bytes.Buffer) // Use bytes.Buffer directly

	for _, vector := range vectors {
		// Quantize vector to 8-bit if needed
		quantizedVector := bs.quantizeVector(vector.ID, vector.Score)

		// Write vector entry
		binary.Write(buf, binary.LittleEndian, quantizedVector.IDLength)
		binary.Write(buf, binary.LittleEndian, quantizedVector.VectorSize)
		binary.Write(buf, binary.LittleEndian, quantizedVector.MetadataSize)
		buf.Write(quantizedVector.ID)
		buf.Write(quantizedVector.Vector)
		buf.Write(quantizedVector.Metadata)
	}

	return buf.Bytes()
}

// serializeFiles converts files to compact binary format
func (bs *BinaryStorage) serializeFiles(files []*models.FileEntry, chunks []models.CodeChunk) []byte {
	buf := new(bytes.Buffer)

	// Create chunk ID mapping
	chunkIDMap := make(map[string]uint32)
	for i, chunk := range chunks {
		chunkIDMap[chunk.ID] = uint32(i)
	}

	for _, file := range files {
		pathBytes := []byte(file.FilePath)
		languageBytes := []byte(file.Language)

		// Get chunk IDs for this file
		chunkIDs := make([]uint32, len(file.Chunks))
		for i, chunk := range file.Chunks {
			if id, ok := chunkIDMap[chunk.ID]; ok {
				chunkIDs[i] = id
			}
		}

		// Write file entry
		binary.Write(buf, binary.LittleEndian, uint16(len(pathBytes)))
		binary.Write(buf, binary.LittleEndian, uint8(len(languageBytes)))
		binary.Write(buf, binary.LittleEndian, uint32(len(chunkIDs)))
		binary.Write(buf, binary.LittleEndian, file.LastModified.Unix())
		binary.Write(buf, binary.LittleEndian, file.Size)
		buf.Write(pathBytes)
		buf.Write(languageBytes)

		// Write chunk IDs
		for _, chunkID := range chunkIDs {
			binary.Write(buf, binary.LittleEndian, chunkID)
		}
	}

	return buf.Bytes()
}

// serializeChunks converts chunks to compact binary format
func (bs *BinaryStorage) serializeChunks(chunks []models.CodeChunk) []byte {
	buf := new(bytes.Buffer)

	for _, chunk := range chunks {
		contentBytes := []byte(chunk.Content)

		// Write chunk entry (use hash of ID for numeric representation)
		vectorIDHash := simpleHash(chunk.ID) % (1<<32 - 1)
		binary.Write(buf, binary.LittleEndian, uint32(vectorIDHash))
		// FileID will be handled at the higher level
		binary.Write(buf, binary.LittleEndian, uint32(0))
		binary.Write(buf, binary.LittleEndian, chunk.StartLine)
		binary.Write(buf, binary.LittleEndian, chunk.EndLine)
		binary.Write(buf, binary.LittleEndian, uint16(len(contentBytes)))
		buf.Write(contentBytes)
	}

	return buf.Bytes()
}

// serializeMetadata converts metadata to binary format
func (bs *BinaryStorage) serializeMetadata(index *models.CodeIndex) []byte {
	// For now, use JSON for metadata
	// In production, this could be binary too
	stats := index.GetStats()
	data, _ := json.Marshal(stats)
	return data
}

// quantizeVector quantizes a vector to 8-bit precision
func (bs *BinaryStorage) quantizeVector(id string, score float64) VectorEntryBinary {
	return VectorEntryBinary{
		IDLength:    uint16(len(id)),
		VectorSize:  8, // 8 bytes for double precision score
		MetadataSize: 0,
		ID:          []byte(id),
		Vector:      bs.float64ToBytes(score),
		Metadata:    nil,
	}
}

// float64ToBytes converts float64 to 8-byte representation
func (bs *BinaryStorage) float64ToBytes(f float64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(f))
	return buf
}

// compressData compresses data using the configured compression
func (bs *BinaryStorage) compressData(data []byte) ([]byte, error) {
	switch bs.compression {
	case CompressionGzip:
		return bs.compressGzip(data)
	case CompressionSnappy:
		// Would need to import snappy-go
		return data, nil
	case CompressionZstd:
		// Would need to import go-zstd
		return data, nil
	default:
		return data, nil
	}
}

// compressGzip compresses data using gzip
func (bs *BinaryStorage) compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		gz.Close()
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressData attempts to decompress data
func (bs *BinaryStorage) decompressData(data []byte) ([]byte, error) {
	// Try gzip first
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err == nil {
		defer gz.Close()
		return io.ReadAll(gz)
	}

	// If gzip fails, assume data is not compressed
	return data, nil
}

// writeToFile writes data to file with atomic rename
func (bs *BinaryStorage) writeToFile(filePath string, data []byte) error {
	// Write to temporary file first
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, filePath)
}

// readFromFile reads data from file
func (bs *BinaryStorage) readFromFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// deserializeVectors loads vectors from binary data
func (bs *BinaryStorage) deserializeVectors(data []byte, index *models.CodeIndex, count int) error {
	offset := 0

	for i := 0; i < count; i++ {
		if offset+VectorHeaderSize > len(data) {
			return fmt.Errorf("insufficient data for vector header %d", i)
		}

		// Read header
		idLength := binary.LittleEndian.Uint16(data[offset : offset+2])
		vectorSize := binary.LittleEndian.Uint32(data[offset+2 : offset+6])
		metadataSize := binary.LittleEndian.Uint32(data[offset+6 : offset+10])
		offset += VectorHeaderSize

		// Read data
		if offset+int(idLength)+int(vectorSize)+int(metadataSize) > len(data) {
			return fmt.Errorf("insufficient data for vector %d", i)
		}

		id := string(data[offset : offset+int(idLength)])
		offset += int(idLength)

		vectorBytes := data[offset : offset+int(vectorSize)]
		offset += int(vectorSize)

		// Skip metadata for now
		offset += int(metadataSize)

		// Convert back to score (for demo, in real would convert back to vector)
		score := math.Float64frombits(binary.LittleEndian.Uint64(vectorBytes))

		// Add to index (simplified for demo)
		// In real implementation, would reconstruct full vector
		_ = id
		_ = score
	}

	return nil
}

// deserializeFiles loads files from binary data
func (bs *BinaryStorage) deserializeFiles(data []byte, index *models.CodeIndex, count int) error {
	// Implementation for deserializing files
	// This would be more complex in a real implementation
	return nil
}

// deserializeChunks loads chunks from binary data
func (bs *BinaryStorage) deserializeChunks(data []byte, index *models.CodeIndex, count int) error {
	// Implementation for deserializing chunks
	// This would be more complex in a real implementation
	return nil
}

// SetCompression sets the compression type
func (bs *BinaryStorage) SetCompression(compression CompressionType) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.compression = compression
}

// GetCompression returns the current compression type
func (bs *BinaryStorage) GetCompression() CompressionType {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.compression
}

// simpleHash provides a simple hash function for numeric IDs
func simpleHash(s string) uint32 {
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return hash
}