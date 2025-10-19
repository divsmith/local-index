// ABOUTME: Manages downloading and storage of ONNX embedding models

package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ModelManager handles downloading and managing ONNX models
type ModelManager struct {
	modelDir string
	client   *http.Client
}

// NewModelManager creates a new model manager
func NewModelManager() *ModelManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir() // Fallback to temp directory
	}

	modelDir := filepath.Join(homeDir, ".code-search", "models")

	return &ModelManager{
		modelDir: modelDir,
		client: &http.Client{
			Timeout: 10 * time.Minute, // Long timeout for large model downloads
		},
	}
}

// EnsureModel ensures the specified model is available for use
func (mm *ModelManager) EnsureModel(modelName string) (string, error) {
	modelPath := filepath.Join(mm.modelDir, modelName+".onnx")

	// Check if model already exists
	if _, err := os.Stat(modelPath); err == nil {
		// Model exists, verify it's valid
		if mm.verifyModel(modelPath) {
			return modelPath, nil
		}
		// Model exists but is corrupted, remove it
		os.Remove(modelPath)
	}

	// Model doesn't exist or is corrupted, download it
	return mm.downloadModel(modelName, modelPath)
}

// downloadModel downloads a model from Hugging Face
func (mm *ModelManager) downloadModel(modelName, savePath string) (string, error) {
	// Ensure model directory exists
	if err := os.MkdirAll(mm.modelDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create model directory: %w", err)
	}

	// Construct download URL
	// Use the correct path structure from Hugging Face repository
	url := fmt.Sprintf("https://huggingface.co/sentence-transformers/%s/resolve/main/onnx/model.onnx", modelName)

	fmt.Printf("Downloading model %s from Hugging Face...\n", modelName)
	fmt.Printf("URL: %s\n", url)

	// Start download request
	resp, err := mm.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download model: HTTP %d - %s", resp.StatusCode, resp.Status)
	}

	// Create temporary file for download
	tempPath := savePath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("failed to create model file: %w", err)
	}
	defer file.Close()

	// Copy data with progress tracking
	var downloaded int64
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			written, err := file.Write(buf[:n])
			if err != nil {
				os.Remove(tempPath)
				return "", fmt.Errorf("failed to write model file: %w", err)
			}
			downloaded += int64(written)

			// Show progress (only for large downloads)
			if downloaded > 1024*1024 { // 1MB
				mb := float64(downloaded) / (1024 * 1024)
				fmt.Printf("\rDownloaded: %.1f MB", mb)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tempPath)
			return "", fmt.Errorf("failed to download model: %w", err)
		}
	}

	fmt.Printf("\nDownload completed: %.1f MB\n", float64(downloaded)/(1024*1024))

	// Verify the downloaded file
	if !mm.verifyModel(tempPath) {
		os.Remove(tempPath)
		return "", fmt.Errorf("downloaded model file is corrupted")
	}

	// Move temporary file to final location
	if err := os.Rename(tempPath, savePath); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("failed to save model file: %w", err)
	}

	return savePath, nil
}

// verifyModel verifies that a model file is valid
func (mm *ModelManager) verifyModel(modelPath string) bool {
	file, err := os.Open(modelPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Check file size (should be reasonable for an ONNX model)
	stat, err := file.Stat()
	if err != nil {
		return false
	}

	// all-mpnet-base-v2 should be around 438MB
	minSize := int64(100 * 1024 * 1024)  // 100MB minimum
	maxSize := int64(2 * 1024 * 1024 * 1024) // 2GB maximum

	if stat.Size() < minSize || stat.Size() > maxSize {
		fmt.Printf("Model file size verification failed: %d bytes\n", stat.Size())
		return false
	}

	// Try to read the first few bytes to check if it's a valid ONNX file
	header := make([]byte, 8)
	if _, err := file.Read(header); err != nil {
		return false
	}

	// ONNX files typically start with "ONNX" or similar magic bytes
	// This is a basic check - in production you'd want more thorough validation
	validHeaders := [][]byte{
		[]byte("ONNX"),
		[]byte{0x08, 0x03}, // Protocol buffers often start with these bytes
	}

	for _, validHeader := range validHeaders {
		if len(header) >= len(validHeader) &&
		   strings.HasPrefix(string(header), string(validHeader)) {
			return true
		}
	}

	// If we can't verify via headers, at least check that we can read the file
	// This will be further validated when we try to load it with ONNX runtime
	return stat.Size() > 0
}

// GetModelInfo returns information about a downloaded model
func (mm *ModelManager) GetModelInfo(modelName string) (*ModelInfo, error) {
	modelPath := filepath.Join(mm.modelDir, modelName+".onnx")

	stat, err := os.Stat(modelPath)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}

	// Calculate file hash
	hash, err := mm.calculateFileHash(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate model hash: %w", err)
	}

	return &ModelInfo{
		Name:         modelName,
		Path:         modelPath,
		Size:         stat.Size(),
		LastModified: stat.ModTime(),
		SHA256:       hash,
		Format:       "ONNX",
	}, nil
}

// ListModels returns all available models
func (mm *ModelManager) ListModels() ([]*ModelInfo, error) {
	if _, err := os.Stat(mm.modelDir); os.IsNotExist(err) {
		return []*ModelInfo{}, nil
	}

	files, err := os.ReadDir(mm.modelDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read model directory: %w", err)
	}

	var models []*ModelInfo
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".onnx") {
			modelName := strings.TrimSuffix(file.Name(), ".onnx")
			info, err := mm.GetModelInfo(modelName)
			if err == nil {
				models = append(models, info)
			}
		}
	}

	return models, nil
}

// RemoveModel removes a model from storage
func (mm *ModelManager) RemoveModel(modelName string) error {
	modelPath := filepath.Join(mm.modelDir, modelName+".onnx")

	if err := os.Remove(modelPath); err != nil {
		return fmt.Errorf("failed to remove model: %w", err)
	}

	return nil
}

// calculateFileHash calculates SHA256 hash of a file
func (mm *ModelManager) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ModelInfo contains information about a model
type ModelInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	SHA256       string    `json:"sha256"`
	Format       string    `json:"format"`
}

// GetHumanSize returns human-readable size
func (mi *ModelInfo) GetHumanSize() string {
	const unit = 1024
	if mi.Size < unit {
		return fmt.Sprintf("%d B", mi.Size)
	}
	div, exp := int64(unit), 0
	for n := mi.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(mi.Size)/float64(div), "KMGTPE"[exp])
}

// IsDefaultModel checks if this is the default model
func (mi *ModelInfo) IsDefaultModel() bool {
	return mi.Name == "all-mpnet-base-v2"
}

// GetRecommendedModel returns the recommended model for general use
func (mm *ModelManager) GetRecommendedModel() string {
	return "all-mpnet-base-v2"
}

// ValidateModelName checks if a model name is supported
func (mm *ModelManager) ValidateModelName(modelName string) error {
	supportedModels := []string{
		"all-mpnet-base-v2",
		"all-MiniLM-L6-v2",
		"paraphrase-mpnet-base-v2",
	}

	for _, supported := range supportedModels {
		if modelName == supported {
			return nil
		}
	}

	return fmt.Errorf("unsupported model: %s. Supported models: %v", modelName, supportedModels)
}