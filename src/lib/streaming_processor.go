package lib

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"code-search/src/models"
)

// StreamingProcessor handles processing of large files using streaming techniques
type StreamingProcessor struct {
	config       *StreamingConfig
	bufferPool   *BufferPool
	chunkPool    *ChunkPool
	workerPool   *WorkerPool
	memoryLimiter *MemoryLimiter
	stats        *StreamingStats
}

// StreamingConfig contains configuration for streaming processing
type StreamingConfig struct {
	MaxMemoryUsage     int64         `json:"max_memory_usage"`      // Maximum memory usage in bytes
	ChunkSize          int           `json:"chunk_size"`             // Size of processing chunks
	BufferSize         int           `json:"buffer_size"`            // Buffer size for reading
	MaxFileSize        int64         `json:"max_file_size"`          // File size threshold for streaming
	SlidingWindowSize  int           `json:"sliding_window_size"`    // Size of sliding window
	OverlapSize        int           `json:"overlap_size"`           // Overlap between chunks
	MaxWorkers         int           `json:"max_workers"`            // Maximum concurrent workers
	ChunkTimeout       time.Duration `json:"chunk_timeout"`          // Timeout for chunk processing
	TempDir            string        `json:"temp_dir"`               // Temporary directory for spills
	EnableSpilling     bool          `json:"enable_spilling"`        // Enable temporary file spills
	CompressionLevel   int           `json:"compression_level"`      // Compression level for spills
}

// DefaultStreamingConfig returns default streaming configuration
func DefaultStreamingConfig() *StreamingConfig {
	return &StreamingConfig{
		MaxMemoryUsage:    512 * 1024 * 1024, // 512MB
		ChunkSize:         8192,              // 8KB chunks
		BufferSize:        64 * 1024,         // 64KB buffer
		MaxFileSize:       10 * 1024 * 1024, // 10MB threshold
		SlidingWindowSize: 1024,              // 1KB sliding window
		OverlapSize:       256,               // 256 bytes overlap
		MaxWorkers:        runtime.NumCPU(),
		ChunkTimeout:      30 * time.Second,
		TempDir:           os.TempDir(),
		EnableSpilling:    true,
		CompressionLevel:  6,
	}
}

// StreamingStats tracks streaming processing statistics
type StreamingStats struct {
	FilesProcessed     int64         `json:"files_processed"`
	BytesProcessed     int64         `json:"bytes_processed"`
	ChunksProcessed    int64         `json:"chunks_processed"`
	MemoryUsed         int64         `json:"memory_used"`
	TempFilesCreated   int64         `json:"temp_files_created"`
	SpillOperations    int64         `json:"spill_operations"`
	AverageChunkTime   time.Duration `json:"average_chunk_time"`
	PeakMemoryUsage    int64         `json:"peak_memory_usage"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	mu                 sync.RWMutex
}

// Chunk represents a processed chunk of data
type Chunk struct {
	ID          string                 `json:"id"`
	Data        []byte                 `json:"data"`
	StartPos    int64                  `json:"start_pos"`
	EndPos      int64                  `json:"end_pos"`
	Hash        string                 `json:"hash"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	ProcessedAt time.Time              `json:"processed_at"`
}

// ProcessingResult contains the result of processing a chunk
type ProcessingResult struct {
	Chunk   Chunk
	Results []models.CodeChunk
	Error   error
	Took    time.Duration
}

// MemoryLimiter monitors and limits memory usage
type MemoryLimiter struct {
	maxMemory    int64
	currentUsage int64
	mu           sync.RWMutex
}

// NewMemoryLimiter creates a new memory limiter
func NewMemoryLimiter(maxMemory int64) *MemoryLimiter {
	return &MemoryLimiter{
		maxMemory: maxMemory,
	}
}

// Allocate allocates memory if available
func (ml *MemoryLimiter) Allocate(size int64) bool {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if ml.currentUsage+size > ml.maxMemory {
		return false
	}

	ml.currentUsage += size
	return true
}

// Release releases allocated memory
func (ml *MemoryLimiter) Release(size int64) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if ml.currentUsage > size {
		ml.currentUsage -= size
	} else {
		ml.currentUsage = 0
	}
}

// GetCurrentUsage returns current memory usage
func (ml *MemoryLimiter) GetCurrentUsage() int64 {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	return ml.currentUsage
}

// NewStreamingProcessor creates a new streaming processor
func NewStreamingProcessor(config *StreamingConfig) *StreamingProcessor {
	if config == nil {
		config = DefaultStreamingConfig()
	}

	poolManager := GetPoolManager()

	return &StreamingProcessor{
		config:        config,
		bufferPool:    poolManager.GetBufferPool(),
		chunkPool:     poolManager.GetChunkPool(),
		workerPool:    NewWorkerPool(DefaultPoolOptions()),
		memoryLimiter: NewMemoryLimiter(config.MaxMemoryUsage),
		stats: &StreamingStats{
			PeakMemoryUsage: 0,
		},
	}
}

// ProcessFile processes a file using streaming techniques
func (sp *StreamingProcessor) ProcessFile(ctx context.Context, filePath string, processor ChunkProcessor) ([]models.CodeChunk, error) {

	// Check if file should use streaming processing
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() <= sp.config.MaxFileSize {
		// Use regular processing for small files
		return sp.processSmallFile(ctx, filePath, processor)
	}

	// Use streaming for large files
	return sp.processLargeFile(ctx, filePath, processor)
}

// processSmallFile processes small files normally
func (sp *StreamingProcessor) processSmallFile(ctx context.Context, filePath string, processor ChunkProcessor) ([]models.CodeChunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	chunk := Chunk{
		ID:        fmt.Sprintf("small_%s_%d", filepath.Base(filePath), time.Now().UnixNano()),
		Data:      content,
		StartPos:  0,
		EndPos:    int64(len(content)),
		CreatedAt: time.Now(),
	}

	result, err := processor.ProcessChunk(ctx, chunk)
	if err != nil {
		return nil, fmt.Errorf("failed to process file: %w", err)
	}

	atomic.AddInt64(&sp.stats.FilesProcessed, 1)
	atomic.AddInt64(&sp.stats.BytesProcessed, int64(len(content)))
	atomic.AddInt64(&sp.stats.ChunksProcessed, 1)

	return result.Results, nil
}

// processLargeFile processes large files using streaming
func (sp *StreamingProcessor) processLargeFile(ctx context.Context, filePath string, processor ChunkProcessor) ([]models.CodeChunk, error) {
	start := time.Now()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, sp.config.BufferSize)
	var results []models.CodeChunk
	var allChunks []Chunk
	var tempFiles []string

	// Process file in chunks
	position := int64(0)
	chunkID := 0

	for {
		select {
		case <-ctx.Done():
			// Cleanup temporary files
			sp.cleanupTempFiles(tempFiles)
			return nil, ctx.Err()
		default:
		}

		// Check memory availability
		if !sp.memoryLimiter.Allocate(int64(sp.config.ChunkSize)) {
			// Memory pressure detected, trigger spill
			spilledChunks, tempFile, err := sp.spillChunks(allChunks)
			if err != nil {
				return nil, fmt.Errorf("failed to spill chunks: %w", err)
			}

			if tempFile != "" {
				tempFiles = append(tempFiles, tempFile)
				atomic.AddInt64(&sp.stats.TempFilesCreated, 1)
				atomic.AddInt64(&sp.stats.SpillOperations, 1)
			}

			allChunks = spilledChunks
			continue
		}

		// Read next chunk
		chunk := make([]byte, sp.config.ChunkSize)
		n, err := io.ReadFull(reader, chunk)

		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			sp.memoryLimiter.Release(int64(sp.config.ChunkSize))
			sp.cleanupTempFiles(tempFiles)
			return nil, fmt.Errorf("failed to read chunk: %w", err)
		}

		if n == 0 {
			sp.memoryLimiter.Release(int64(sp.config.ChunkSize))
			break // End of file
		}

		// Resize chunk to actual size read
		chunk = chunk[:n]
		endPos := position + int64(n)

		// Create chunk object
		chunkObj := Chunk{
			ID:       fmt.Sprintf("chunk_%d", chunkID),
			Data:     chunk,
			StartPos: position,
			EndPos:   endPos,
			CreatedAt: time.Now(),
		}

		// Calculate chunk hash
		chunkObj.Hash = sp.calculateChunkHash(chunk)

		allChunks = append(allChunks, chunkObj)
		chunkID++
		position = endPos

		// Process chunks when we have enough or reached end of file
		if len(allChunks) >= sp.config.MaxWorkers || err == io.EOF {
			chunkResults := sp.processChunkBatch(ctx, allChunks, processor)
			results = append(results, chunkResults...)
			allChunks = nil // Reset for next batch
		}

		atomic.AddInt64(&sp.stats.ChunksProcessed, 1)
	}

	// Process remaining chunks
	if len(allChunks) > 0 {
		chunkResults := sp.processChunkBatch(ctx, allChunks, processor)
		results = append(results, chunkResults...)
	}

	// Cleanup temporary files
	sp.cleanupTempFiles(tempFiles)

	// Update statistics
	processingTime := time.Since(start)
	atomic.AddInt64(&sp.stats.FilesProcessed, 1)
	atomic.AddInt64(&sp.stats.BytesProcessed, position)
	sp.stats.mu.Lock()
	sp.stats.TotalProcessingTime = processingTime
	sp.stats.mu.Unlock()

	return results, nil
}

// processChunkBatch processes a batch of chunks in parallel
func (sp *StreamingProcessor) processChunkBatch(ctx context.Context, chunks []Chunk, processor ChunkProcessor) []models.CodeChunk {
	if len(chunks) == 0 {
		return nil
	}

	results := make([]ProcessingResult, len(chunks))
	var wg sync.WaitGroup

	// Process chunks in parallel
	for i, chunk := range chunks {
		wg.Add(1)
		sp.workerPool.Submit(func() (interface{}, error) {
			defer wg.Done()

			start := time.Now()
			chunkCtx, cancel := context.WithTimeout(ctx, sp.config.ChunkTimeout)
			defer cancel()

			result, err := processor.ProcessChunk(chunkCtx, chunk)
			processingTime := time.Since(start)

			results[i] = ProcessingResult{
				Chunk:   chunk,
				Results: result.Results,
				Error:   err,
				Took:    processingTime,
			}

			// Release memory
			sp.memoryLimiter.Release(int64(len(chunk.Data)))

			return nil, nil
		})
	}

	wg.Wait()

	// Collect results and handle errors
	var allResults []models.CodeChunk
	for i, result := range results {
		if result.Error != nil {
			// Log error but continue processing other chunks
			fmt.Printf("Warning: Failed to process chunk %d: %v\n", i, result.Error)
			continue
		}

		// Add metadata about processing
		for j := range result.Results {
			if result.Results[j].Metadata == nil {
				result.Results[j].Metadata = make(map[string]interface{})
			}
			result.Results[j].Metadata["chunk_id"] = result.Chunk.ID
			result.Results[j].Metadata["chunk_start"] = result.Chunk.StartPos
			result.Results[j].Metadata["chunk_end"] = result.Chunk.EndPos
			result.Results[j].Metadata["processing_time"] = result.Took.String()
		}

		allResults = append(allResults, result.Results...)
	}

	return allResults
}

// spillChunks spills chunks to temporary storage when memory is constrained
func (sp *StreamingProcessor) spillChunks(chunks []Chunk) ([]Chunk, string, error) {
	if !sp.config.EnableSpilling {
		// Return chunks without spilling if disabled
		return chunks, "", nil
	}

	// Create temporary file
	tempFile, err := os.CreateTemp(sp.config.TempDir, "chunk_spill_*.tmp")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Serialize chunks to temporary file
	spilledChunks := make([]Chunk, 0, len(chunks))
	releasedMemory := int64(0)

	for _, chunk := range chunks {
		// Create a lightweight chunk reference
		lightChunk := Chunk{
			ID:       chunk.ID,
			StartPos: chunk.StartPos,
			EndPos:   chunk.EndPos,
			Hash:     chunk.Hash,
			CreatedAt: chunk.CreatedAt,
			Metadata: map[string]interface{}{
				"spilled":    true,
				"temp_file":  tempFile.Name(),
				"data_size":  len(chunk.Data),
			},
		}

		spilledChunks = append(spilledChunks, lightChunk)
		releasedMemory += int64(len(chunk.Data))

		// Write chunk data to temp file
		if _, err := tempFile.Write(chunk.Data); err != nil {
			return nil, "", fmt.Errorf("failed to write chunk to temp file: %w", err)
		}
	}

	// Release memory
	sp.memoryLimiter.Release(releasedMemory)

	return spilledChunks, tempFile.Name(), nil
}

// cleanupTempFiles cleans up temporary files
func (sp *StreamingProcessor) cleanupTempFiles(tempFiles []string) {
	for _, tempFile := range tempFiles {
		if err := os.Remove(tempFile); err != nil {
			// Log error but don't fail
			fmt.Printf("Warning: Failed to remove temp file %s: %v\n", tempFile, err)
		}
	}
}

// calculateChunkHash calculates a hash for a chunk
func (sp *StreamingProcessor) calculateChunkHash(data []byte) string {
	// Simple hash implementation
	// In production, use a proper hash function like SHA-256
	hash := uint64(0)
	for i, b := range data {
		hash += uint64(b) << (i % 64)
		hash ^= hash >> 33
	}
	return fmt.Sprintf("%016x", hash)
}

// ProcessFileWithSlidingWindow processes a file using a sliding window approach
func (sp *StreamingProcessor) ProcessFileWithSlidingWindow(ctx context.Context, filePath string, processor ChunkProcessor) ([]models.CodeChunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, sp.config.SlidingWindowSize+sp.config.OverlapSize)
	var results []models.CodeChunk
	window := make([]byte, 0, sp.config.SlidingWindowSize+sp.config.OverlapSize)
	position := int64(0)
	chunkID := 0

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Read data into buffer
		buffer := make([]byte, sp.config.SlidingWindowSize)
		n, err := io.ReadFull(reader, buffer)

		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("failed to read window: %w", err)
		}

		if n == 0 {
			break
		}

		buffer = buffer[:n]

		// Add new data to window
		window = append(window, buffer...)

		// Keep window size manageable
		if len(window) > sp.config.SlidingWindowSize+sp.config.OverlapSize {
			// Remove oldest overlap bytes
			window = window[sp.config.OverlapSize:]
		}

		// Process if we have enough data
		if len(window) >= sp.config.SlidingWindowSize {
			chunkData := make([]byte, sp.config.SlidingWindowSize)
			copy(chunkData, window)

			chunk := Chunk{
				ID:       fmt.Sprintf("window_%d", chunkID),
				Data:     chunkData,
				StartPos: position,
				EndPos:   position + int64(len(chunkData)),
				CreatedAt: time.Now(),
			}

			chunkResult, err := processor.ProcessChunk(ctx, chunk)
			if err != nil {
				return nil, fmt.Errorf("failed to process window chunk: %w", err)
			}

			results = append(results, chunkResult.Results...)
			chunkID++

			// Slide window by overlap size
			if sp.config.OverlapSize > 0 {
				_, err := file.Seek(int64(sp.config.OverlapSize)-int64(len(buffer)), io.SeekCurrent)
				if err != nil {
					return nil, fmt.Errorf("failed to seek for sliding window: %w", err)
				}
				position += int64(sp.config.OverlapSize)
			} else {
				position += int64(len(chunkData))
			}
		}
	}

	return results, nil
}

// ChunkProcessor interface for processing chunks
type ChunkProcessor interface {
	ProcessChunk(ctx context.Context, chunk Chunk) (ChunkResult, error)
}

// ChunkResult contains the result of processing a chunk
type ChunkResult struct {
	Results []models.CodeChunk
	Error   error
}

// GetStats returns current streaming statistics
func (sp *StreamingProcessor) GetStats() StreamingStats {
	sp.stats.mu.RLock()
	defer sp.stats.mu.RUnlock()

	stats := *sp.stats
	stats.MemoryUsed = sp.memoryLimiter.GetCurrentUsage()
	return stats
}

// ResetStats resets streaming statistics
func (sp *StreamingProcessor) ResetStats() {
	sp.stats.mu.Lock()
	defer sp.stats.mu.Unlock()

	sp.stats = &StreamingStats{
		PeakMemoryUsage: 0,
	}
}

// Close cleans up resources
func (sp *StreamingProcessor) Close() error {
	sp.workerPool.Close()
	return nil
}

