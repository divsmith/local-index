package integration

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)


// TestIndexingIntegration tests the complete indexing workflow
func TestIndexingIntegration(t *testing.T) {
	// Set up test environment
	resourceDir := setupTestEnvironment(t, "TestIndexingIntegration")

	// Create a realistic codebase structure
	projectStructure := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	config := LoadConfig()

	if err := StartServer(config); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Println("Server started successfully")
}

func LoadConfig() *Config {
	return &Config{
		Port:    8080,
		Timeout: 30,
	}
}
`,
		"config/config.go": `package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port    int    ` + "`json:\"port\"`" + `
	Timeout int    ` + "`json:\"timeout\"`" + `
	Host    string ` + "`json:\"host\"`" + `
	Debug   bool   ` + "`json:\"debug\"`" + `
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}
`,
		"server/server.go": `package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Server struct {
	config *config.Config
	server *http.Server
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			ReadTimeout:  time.Duration(cfg.Timeout) * time.Second,
			WriteTimeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) SetupRoutes() {
	http.HandleFunc("/", s.handleHome)
	http.HandleFunc("/health", s.handleHealth)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}
`,
		"handlers/api.go": `package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

type APIHandler struct {
	server *server.Server
}

func NewAPIHandler(srv *server.Server) *APIHandler {
	return &APIHandler{
		server: srv,
	}
}

func (h *APIHandler) HandleAPIRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message": "GET request handled",
		"time":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *APIHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"received": data,
		"status":   "processed",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
`,
		"utils/helpers.go": `package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func LogInfo(message string) {
	log.Printf("[INFO] %s", message)
}

func LogError(message string, err error) {
	log.Printf("[ERROR] %s: %v", message, err)
}

func SanitizeInput(input string) string {
	// Remove potentially dangerous characters
	sanitized := strings.ReplaceAll(input, "<", "&lt;")
	sanitized = strings.ReplaceAll(sanitized, ">", "&gt;")
	return strings.TrimSpace(sanitized)
}

func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(` + "`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$`" + `)
	return emailRegex.MatchString(email)
}

func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1e6)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
`,
		"internal/database/db.go": `package database

import (
	"database/sql"
	"fmt"
	"time"
	_ "github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dsn string) (*Database, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) CreateUser(name, email string) error {
	query := ` + "`INSERT INTO users (name, email) VALUES ($1, $2)`" + `
	_, err := d.db.Exec(query, name, email)
	return err
}

func (d *Database) GetUser(id int) (*User, error) {
	query := ` + "`SELECT id, name, email, created_at FROM users WHERE id = $1`" + `
	var user User
	err := d.db.QueryRow(query, id).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

type User struct {
	ID        int       ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\"`" + `
	Email     string    ` + "`json:\"email\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`,
		"README.md": `# Test Project

This is a test project for the code search indexing functionality.

## Features

- HTTP server with configuration
- Database integration
- API handlers
- Utility functions

## Usage

1. Run ` + "`go run main.go`" + `
2. Visit http://localhost:8080

## Configuration

See config/config.go for configuration options.
`,
	}

	// Create the directory structure and files
	for path, content := range projectStructure {
		fullPath := filepath.Join(resourceDir, path)
		dir := filepath.Dir(fullPath)

		if dir != resourceDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}

		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Change to resource directory
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(resourceDir)
	if err != nil {
		t.Fatalf("Failed to change to resource directory: %v", err)
	}

	// Test 1: Basic indexing
	t.Run("BasicIndexing", func(t *testing.T) {
		cmd := exec.Command("./code-search", "index")
		cmd.Dir = resourceDir
		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Index command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should report success
		if !strings.Contains(outputStr, "Indexing complete") {
			t.Errorf("Expected success message, got: %s", outputStr)
		}

		// Should have indexed Go files
		if !strings.Contains(outputStr, "6") { // We have 6 .go files
			t.Errorf("Expected to index 6 Go files, got: %s", outputStr)
		}

		// Should complete in reasonable time
		if duration > 30*time.Second {
			t.Errorf("Indexing took too long: %v", duration)
		}

		// Check that index file was created
		if _, err := os.Stat(resourceDir + "/.code-search-index"); os.IsNotExist(err) {
			t.Error("Expected index file to be created")
		}

		t.Logf("Basic indexing completed in %v", duration)
	})

	// Test 2: Force re-indexing
	t.Run("ForceReindexing", func(t *testing.T) {
		// Modify a file to ensure reindexing creates different content
		readmePath := resourceDir + "/README.md"
		readmeContent, err := os.ReadFile(readmePath)
		if err != nil {
			t.Fatalf("Failed to read README.md: %v", err)
		}

		// Add a new line to change the content
		modifiedReadmeContent := string(readmeContent) + "\n\n# Force Reindex Test\nThis line was added for force reindex testing.\n"
		err = os.WriteFile(readmePath, []byte(modifiedReadmeContent), 0644)
		if err != nil {
			t.Fatalf("Failed to modify README.md: %v", err)
		}

		// Wait a bit to ensure different timestamp
		time.Sleep(100 * time.Millisecond)

		// Force re-index (note: --force uses .test suffix)
		indexFile := resourceDir + "/.code-search-index.test"

		// Remove existing test index if it exists
		os.Remove(indexFile)

		cmd := exec.Command("./code-search", "index", "--force")
		cmd.Dir = resourceDir
		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Force re-index command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Indexing complete") {
			t.Errorf("Expected success message, got: %s", outputStr)
		}

		// Check that test index file was created
		stat2, err := os.Stat(indexFile)
		if err != nil {
			t.Fatalf("Failed to stat test index file after force re-index: %v", err)
		}

		// Verify the test index file exists and has reasonable size
		if stat2.Size() == 0 {
			t.Error("Expected test index file to have content after force re-index")
		}

		t.Logf("Force re-indexing completed in %v, test index size: %d bytes", duration, stat2.Size())
	})

	// Test 3: Incremental indexing (modify existing file)
	t.Run("IncrementalIndexing", func(t *testing.T) {
		// Modify an existing file
		mainGoPath := resourceDir + "/main.go"
		originalContent, err := os.ReadFile(mainGoPath)
		if err != nil {
			t.Fatalf("Failed to read main.go: %v", err)
		}

		// Add a new function
		modifiedContent := string(originalContent) + `

// New function added for incremental indexing test
func NewFunction() {
	fmt.Println("This is a new function")
}
`

		err = os.WriteFile(mainGoPath, []byte(modifiedContent), 0644)
		if err != nil {
			t.Fatalf("Failed to modify main.go: %v", err)
		}

		// Run indexing again
		cmd := exec.Command("./code-search", "index")
		cmd.Dir = resourceDir
		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Incremental index command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Indexing complete") {
			t.Errorf("Expected success message, got: %s", outputStr)
		}

		// Incremental indexing should be faster
		if duration > 10*time.Second {
			t.Errorf("Incremental indexing took too long: %v", duration)
		}

		t.Logf("Incremental indexing completed in %v", duration)
	})

	// Test 4: Index file types
	t.Run("IndexFileTypes", func(t *testing.T) {
		// Create necessary directories
		frontendDir := resourceDir + "/frontend"
		scriptsDir := resourceDir + "/scripts"

		if err := os.MkdirAll(frontendDir, 0755); err != nil {
			t.Fatalf("Failed to create frontend directory: %v", err)
		}

		if err := os.MkdirAll(scriptsDir, 0755); err != nil {
			t.Fatalf("Failed to create scripts directory: %v", err)
		}

		// Create additional file types
		jsFile := resourceDir + "/frontend/app.js"
		err := os.WriteFile(jsFile, []byte(`// JavaScript file
function processData(data) {
    return data.map(item => ({
        ...item,
        processed: true
    }));
}

class APIManager {
    constructor(baseURL) {
        this.baseURL = baseURL;
    }

    async fetch(endpoint) {
        const response = await fetch(this.baseURL + endpoint);
        return response.json();
    }
}
`), 0644)
		if err != nil {
			t.Fatalf("Failed to create JavaScript file: %v", err)
		}

		pyFile := resourceDir + "/scripts/setup.py"
		err = os.WriteFile(pyFile, []byte(`#!/usr/bin/env python3

import os
import sys

def setup_environment():
    """Setup the development environment"""
    print("Setting up environment...")

    # Create necessary directories
    os.makedirs("logs", exist_ok=True)
    os.makedirs("data", exist_ok=True)

    print("Environment setup complete!")

def validate_config():
    """Validate configuration files"""
    config_file = "config.json"
    if not os.path.exists(config_file):
        print(f"Configuration file {config_file} not found")
        return False

    print("Configuration is valid")
    return True

if __name__ == "__main__":
    setup_environment()
    validate_config()
`), 0644)
		if err != nil {
			t.Fatalf("Failed to create Python file: %v", err)
		}

		// Force re-index to include new files
		cmd := exec.Command("./code-search", "index", "--force")
		cmd.Dir = resourceDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Index with new file types failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Indexing complete") {
			t.Errorf("Expected success message, got: %s", outputStr)
		}

		// Should now index more files (including non-Go files)
		if !strings.Contains(outputStr, "9") { // Should now have 9 files
			t.Errorf("Expected to index 9 files including JS and Python, got: %s", outputStr)
		}
	})

	// Test 5: Index with hidden files
	t.Run("IndexWithHiddenFiles", func(t *testing.T) {
		// Create hidden files and directories
		hiddenFile := resourceDir + "/.env"
		err := os.WriteFile(hiddenFile, []byte(`# Environment variables
DATABASE_URL=postgres://localhost:5432/mydb
API_KEY=secret_key_here
DEBUG=true
`), 0644)
		if err != nil {
			t.Fatalf("Failed to create hidden file: %v", err)
		}

		hiddenDir := resourceDir + "/.config"
		err = os.MkdirAll(hiddenDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create hidden directory: %v", err)
		}

		hiddenConfigFile := hiddenDir + "/app.json"
		err = os.WriteFile(hiddenConfigFile, []byte(`{
    "app_name": "test_app",
    "version": "1.0.0",
    "features": ["search", "indexing"]
}`), 0644)
		if err != nil {
			t.Fatalf("Failed to create hidden config file: %v", err)
		}

		// Index without including hidden files (default behavior)
		cmd := exec.Command("./code-search", "index", "--force")
		cmd.Dir = resourceDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Index without hidden files failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Indexing complete") {
			t.Errorf("Expected success message, got: %s", outputStr)
		}

		// Index with hidden files included
		cmd = exec.Command("./code-search", "index", "--force", "--include-hidden")
		cmd.Dir = resourceDir
		output, err = cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Index with hidden files failed: %v, output: %s", err, string(output))
		}

		outputStr = string(output)
		if !strings.Contains(outputStr, "Indexing complete") {
			t.Errorf("Expected success message, got: %s", outputStr)
		}

		// Should now include the hidden files
		if !strings.Contains(outputStr, "10") { // Should now have 10 files total
			t.Errorf("Expected to index 10 files including hidden files, got: %s", outputStr)
		}
	})

	// Test 6: Error handling
	t.Run("IndexingErrorHandling", func(t *testing.T) {
		// Create a file that can't be read (permission denied simulation)
		unreadableFile := resourceDir + "/unreadable.go"
		err := os.WriteFile(unreadableFile, []byte(`package main

func thisShouldNotBeIndexed() {
    fmt.Println("This file should cause an error")
}`), 0000) // No permissions
		if err != nil {
			t.Fatalf("Failed to create unreadable file: %v", err)
		}

		// Index should handle the error gracefully
		cmd := exec.Command("./code-search", "index", "--force")
		cmd.Dir = resourceDir
		output, err := cmd.CombinedOutput()

		// Should still succeed but maybe log a warning
		if err != nil {
			t.Errorf("Index command failed with unreadable file: %v, output: %s", err, string(output))
		}

		// Clean up
		os.Chmod(unreadableFile, 0644)
		os.Remove(unreadableFile)
	})
}

// TestIndexingPerformance tests indexing performance requirements
func TestIndexingPerformance(t *testing.T) {
	// Set up test environment
	resourceDir := setupTestEnvironment(t, "TestIndexingPerformance")

	// Create a larger codebase for performance testing
	for i := 0; i < 100; i++ {
		dirName := fmt.Sprintf("module_%d", i)
		moduleDir := filepath.Join(resourceDir, dirName)
		err := os.MkdirAll(moduleDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create module directory: %v", err)
		}

		for j := 0; j < 5; j++ {
			fileName := fmt.Sprintf("file_%d.go", j)
			content := fmt.Sprintf(`package %s

import (
	"fmt"
	"time"
)

// Function%d%d performs some operation
func Function%d%d(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}

	// Process the data
	processed := fmt.Sprintf("processed: %%s", data)
	fmt.Println(processed)

	return nil
}

// Validate%d%d validates input data
func Validate%d%d(input string) bool {
	return len(input) > 0 && len(input) < 1000
}

// Process%d%d handles data processing
func Process%d%d(items []string) ([]string, error) {
	var result []string
	for i, item := range items {
		if Validate%d%d(item) {
			processed, err := Function%d%d(item)
			if err != nil {
				return nil, err
			}
			result = append(result, fmt.Sprintf("%%d: %%s", i, processed))
		}
	}
	return result, nil
}
`, dirName, i, j, i, j, i, j, i, j, i, j, i, j, i, j, i, j)

			filePath := filepath.Join(moduleDir, fileName)
			err = os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(resourceDir)
	if err != nil {
		t.Fatalf("Failed to change to resource directory: %v", err)
	}

	// Test indexing performance
	t.Run("LargeCodebaseIndexing", func(t *testing.T) {
		cmd := exec.Command("./code-search", "index")
		cmd.Dir = resourceDir
		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Index command failed on large codebase: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Indexing complete") {
			t.Errorf("Expected success message, got: %s", outputStr)
		}

		// Should have indexed 500 files (100 modules * 5 files each)
		if !strings.Contains(outputStr, "500") {
			t.Errorf("Expected to index 500 files, got: %s", outputStr)
		}

		// Performance requirement: should complete in under 60 seconds for this test size
		if duration > 60*time.Second {
			t.Errorf("Large codebase indexing took too long: %v (should be < 60s)", duration)
		}

		t.Logf("Large codebase indexing (500 files) completed in %v", duration)
	})
}

// calculateContentHash calculates SHA256 hash of file content
func calculateContentHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}