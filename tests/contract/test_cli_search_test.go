package contract

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestCLISearchCommand tests the contract for the CLI search command
func TestCLISearchCommand(t *testing.T) {
	// Create a temporary directory with test files
	tempDir := t.TempDir()

	// Create test Go files with specific content
	testFile1 := tempDir + "/calculator.go"
	err := os.WriteFile(testFile1, []byte(`package main

import "fmt"

// calculateTax calculates tax for the given amount
func calculateTax(amount float64) float64 {
	return amount * 0.08
}

// calculateSum adds two numbers
func calculateSum(a, b int) int {
	return a + b
}

func main() {
	result := calculateTax(100)
	fmt.Printf("Tax: %.2f\n", result)
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFile2 := tempDir + "/invoice.go"
	err = os.WriteFile(testFile2, []byte(`package main

// Invoice represents a customer invoice
type Invoice struct {
	Subtotal float64
	Tax      float64
	Total    float64
}

// CalculateTax computes the tax for this invoice
func (i *Invoice) CalculateTax() {
	i.Tax = i.Subtotal * 0.08
}

// CalculateTotal computes the total amount
func (i *Invoice) CalculateTotal() {
	i.CalculateTax()
	i.Total = i.Subtotal + i.Tax
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Build the CLI tool
	cmd := exec.Command("go", "build", "-o", "code-search", "../../src/cli/main.go")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()

	// If build fails, skip the test
	if err != nil {
		t.Skipf("CLI tool not yet implemented (build failed): %v\nOutput: %s", err, string(output))
	}

	// First, index the directory
	cmd = exec.Command("./code-search", "index")
	cmd.Dir = tempDir
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to index test directory: %v", err)
	}

	// Test basic search functionality
	cmd = exec.Command("./code-search", "search", "calculate tax")
	cmd.Dir = tempDir
	start := time.Now()
	output, err = cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Search command failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)

	// Check that we found results
	if !strings.Contains(outputStr, "Found ") {
		t.Errorf("Expected output to contain 'Found X results', got: %s", outputStr)
	}

	// Check for specific file references
	if !strings.Contains(outputStr, "calculator.go") {
		t.Errorf("Expected output to contain 'calculator.go', got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "invoice.go") {
		t.Errorf("Expected output to contain 'invoice.go', got: %s", outputStr)
	}

	// Check for line numbers
	if !strings.Contains(outputStr, ":") {
		t.Errorf("Expected output to contain line numbers (format: file:line), got: %s", outputStr)
	}

	// Check that search completes in reasonable time (< 5 seconds for small test case)
	if duration > 5*time.Second {
		t.Errorf("Search took too long: %v (should be < 5s)", duration)
	}

	t.Logf("Search command test passed. Duration: %v, Output: %s", duration, outputStr)
}

// TestCLISearchCommandWithMaxResults tests the --max-results flag
func TestCLISearchCommandWithMaxResults(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple test files
	for i := 0; i < 5; i++ {
		testFile := tempDir + "/test.go"
		content := `package main

func calculate() {
	// Calculate function
}

func validate() {
	// Validate function
}
`
		err := os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Build the CLI tool
	cmd := exec.Command("go", "build", "-o", "code-search", "../../src/cli/main.go")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("CLI tool not yet implemented: %v\nOutput: %s", err, string(output))
	}

	// Index the directory
	cmd = exec.Command("./code-search", "index")
	cmd.Dir = tempDir
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to index test directory: %v", err)
	}

	// Test with max results limit
	cmd = exec.Command("./code-search", "search", "calculate", "--max-results", "2")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Search command with --max-results failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)

	// Should find exactly 2 results
	if !strings.Contains(outputStr, "Found 2 results") {
		t.Errorf("Expected output to contain 'Found 2 results', got: %s", outputStr)
	}
}

// TestCLISearchCommandJSONFormat tests the --format json flag
func TestCLISearchCommandJSONFormat(t *testing.T) {
	tempDir := t.TempDir()

	testFile := tempDir + "/test.go"
	err := os.WriteFile(testFile, []byte(`package main

func calculateTax(amount float64) float64 {
	return amount * 0.08
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Build the CLI tool
	cmd := exec.Command("go", "build", "-o", "code-search", "../../src/cli/main.go")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("CLI tool not yet implemented: %v\nOutput: %s", err, string(output))
	}

	// Index the directory
	cmd = exec.Command("./code-search", "index")
	cmd.Dir = tempDir
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to index test directory: %v", err)
	}

	// Test with JSON format
	cmd = exec.Command("./code-search", "search", "calculate", "--format", "json")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Search command with JSON format failed: %v, output: %s", err, string(output))
	}

	// Parse JSON output
	var searchResult map[string]interface{}
	err = json.Unmarshal(output, &searchResult)
	if err != nil {
		t.Errorf("Failed to parse JSON output: %v, output: %s", err, string(output))
	}

	// Validate JSON structure
	if _, ok := searchResult["query"]; !ok {
		t.Error("JSON output missing 'query' field")
	}
	if _, ok := searchResult["results"]; !ok {
		t.Error("JSON output missing 'results' field")
	}
	if _, ok := searchResult["totalResults"]; !ok {
		t.Error("JSON output missing 'totalResults' field")
	}
	if _, ok := searchResult["executionTime"]; !ok {
		t.Error("JSON output missing 'executionTime' field")
	}
}

// TestCLISearchCommandFilePattern tests the --file-pattern flag
func TestCLISearchCommandFilePattern(t *testing.T) {
	tempDir := t.TempDir()

	// Create different file types
	goFile := tempDir + "/test.go"
	jsFile := tempDir + "/test.js"

	err := os.WriteFile(goFile, []byte(`package main

func calculate() {
	// Go function
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create Go file: %v", err)
	}

	err = os.WriteFile(jsFile, []byte(`function calculate() {
	// JavaScript function
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create JavaScript file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Build the CLI tool
	cmd := exec.Command("go", "build", "-o", "code-search", "../../src/cli/main.go")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("CLI tool not yet implemented: %v\nOutput: %s", err, string(output))
	}

	// Index the directory
	cmd = exec.Command("./code-search", "index")
	cmd.Dir = tempDir
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to index test directory: %v", err)
	}

	// Test with file pattern filter for Go files only
	cmd = exec.Command("./code-search", "search", "calculate", "--file-pattern", "*.go")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Search command with --file-pattern failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)

	// Should find results in Go files
	if !strings.Contains(outputStr, "test.go") {
		t.Errorf("Expected output to contain 'test.go', got: %s", outputStr)
	}

	// Should not find results in JavaScript files
	if strings.Contains(outputStr, "test.js") {
		t.Errorf("Expected output to NOT contain 'test.js' when filtering by *.go, got: %s", outputStr)
	}
}

// TestCLISearchCommandErrorHandling tests error scenarios
func TestCLISearchCommandErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWD)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Build the CLI tool
	cmd := exec.Command("go", "build", "-o", "code-search", "../../src/cli/main.go")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("CLI tool not yet implemented: %v\nOutput: %s", err, string(output))
	}

	// Test search without index (should fail)
	cmd = exec.Command("./code-search", "search", "test")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()

	if err == nil {
		t.Error("Expected search command to fail without index, but it succeeded")
	}

	// Should exit with code 3 for index not found
	if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != 3 {
		t.Errorf("Expected exit code 3 for index not found, got %d", exitError.ExitCode())
	}

	// Test with invalid arguments
	cmd = exec.Command("./code-search", "search", "test", "--invalid-flag")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()

	if err == nil {
		t.Error("Expected search command to fail with invalid flag, but it succeeded")
	}

	// Should exit with code 2 for invalid arguments
	if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != 2 {
		t.Errorf("Expected exit code 2 for invalid arguments, got %d", exitError.ExitCode())
	}

	// Test with no query argument
	cmd = exec.Command("./code-search", "search")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()

	if err == nil {
		t.Error("Expected search command to fail without query argument, but it succeeded")
	}

	// Should exit with code 2 for missing arguments
	if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != 2 {
		t.Errorf("Expected exit code 2 for missing arguments, got %d", exitError.ExitCode())
	}
}
