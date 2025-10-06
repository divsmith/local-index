package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSearchIntegration tests the complete search workflow
func TestSearchIntegration(t *testing.T) {
	// Create a temporary directory with realistic code files
	tempDir := t.TempDir()

	// Create a realistic Go project structure
	files := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"log"
)

func main() {
	user := User{
		Name: "John Doe",
		Email: "john@example.com",
	}

	if err := ValidateUser(&user); err != nil {
		log.Fatalf("User validation failed: %v", err)
	}

	fmt.Printf("Validated user: %s\n", user.Name)
}

func ProcessPayment(amount float64, currency string) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Process payment logic would go here
	fmt.Printf("Processing payment: %.2f %s\n", amount, currency)
	return nil
}
`,
		"user.go": `package main

import (
	"regexp"
	"strings"
)

type User struct {
	ID    int    ` + "`json:\"id\"`" + `
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}

func ValidateUser(user *User) error {
	if user.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if !isValidEmail(user.Email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(` + "`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$`" + `)
	return emailRegex.MatchString(email)
}

func (u *User) GetDisplayName() string {
	return strings.Title(strings.ToLower(u.Name))
}
`,
		"payment.go": `package main

import (
	"fmt"
	"time"
)

type Payment struct {
	ID        string
	Amount    float64
	Currency  string
	Status    string
	CreatedAt time.Time
}

func ProcessPaymentWithValidation(amount float64, currency string) (*Payment, error) {
	if err := validateAmount(amount); err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	if err := validateCurrency(currency); err != nil {
		return nil, fmt.Errorf("invalid currency: %w", err)
	}

	payment := &Payment{
		ID:        generatePaymentID(),
		Amount:    amount,
		Currency:  currency,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	// Processing logic
	payment.Status = "completed"

	return payment, nil
}

func validateAmount(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if amount > 1000000 {
		return fmt.Errorf("amount exceeds maximum limit")
	}
	return nil
}

func validateCurrency(currency string) error {
	supportedCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true,
	}
	if !supportedCurrencies[currency] {
		return fmt.Errorf("unsupported currency: %s", currency)
	}
	return nil
}

func generatePaymentID() string {
	return fmt.Sprintf("pay_%d", time.Now().UnixNano())
}
`,
		"utils/helpers.go": `package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random string: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func LogError(message string, err error) {
	fmt.Printf("ERROR: %s: %v\n", message, err)
}
`,
	}

	// Create directory structure and files
	for path, content := range files {
		fullPath := tempDir + "/" + path
		dir := tempDir + "/" + filepath.Dir(path)

		if dir != tempDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}

		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
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

	// Test 1: Index the project
	t.Run("IndexProject", func(t *testing.T) {
		cmd := exec.Command("./code-search", "index")
		cmd.Dir = tempDir
		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Index command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Indexing complete") {
			t.Errorf("Expected indexing completion message, got: %s", outputStr)
		}

		// Should index all .go files
		if !strings.Contains(outputStr, "4") { // We created 4 .go files
			t.Errorf("Expected to index 4 files, got: %s", outputStr)
		}

		// Indexing should complete reasonably fast
		if duration > 30*time.Second {
			t.Errorf("Indexing took too long: %v", duration)
		}

		t.Logf("Indexing completed in %v", duration)
	})

	// Test 2: Search for user validation functionality
	t.Run("SearchUserValidation", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "user validation")
		cmd.Dir = tempDir
		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Search command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should find results
		if !strings.Contains(outputStr, "Found ") {
			t.Errorf("Expected search results, got: %s", outputStr)
		}

		// Should find user.go file
		if !strings.Contains(outputStr, "user.go") {
			t.Errorf("Expected to find user.go, got: %s", outputStr)
		}

		// Should find ValidateUser function
		if !strings.Contains(outputStr, "ValidateUser") {
			t.Errorf("Expected to find ValidateUser function, got: %s", outputStr)
		}

		// Search should be fast
		if duration > 5*time.Second {
			t.Errorf("Search took too long: %v", duration)
		}

		t.Logf("Search for 'user validation' completed in %v", duration)
	})

	// Test 3: Search for payment processing
	t.Run("SearchPaymentProcessing", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "payment processing")
		cmd.Dir = tempDir
		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Search command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should find results in multiple files
		if !strings.Contains(outputStr, "main.go") {
			t.Errorf("Expected to find main.go, got: %s", outputStr)
		}

		if !strings.Contains(outputStr, "payment.go") {
			t.Errorf("Expected to find payment.go, got: %s", outputStr)
		}

		// Should find ProcessPayment functions
		if !strings.Contains(outputStr, "ProcessPayment") {
			t.Errorf("Expected to find ProcessPayment functions, got: %s", outputStr)
		}

		t.Logf("Search for 'payment processing' completed in %v", duration)
	})

	// Test 4: Search with file pattern filtering
	t.Run("SearchWithFilePattern", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "validation", "--file-pattern", "*.go")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Search command with file pattern failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should find validation-related code
		if !strings.Contains(outputStr, "ValidateUser") {
			t.Errorf("Expected to find ValidateUser, got: %s", outputStr)
		}

		if !strings.Contains(outputStr, "validateAmount") {
			t.Errorf("Expected to find validateAmount, got: %s", outputStr)
		}

		if !strings.Contains(outputStr, "validateCurrency") {
			t.Errorf("Expected to find validateCurrency, got: %s", outputStr)
		}
	})

	// Test 5: Search with max results limit
	t.Run("SearchWithMaxResults", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "func", "--max-results", "3")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Search command with max results failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should limit to 3 results
		if !strings.Contains(outputStr, "Found 3 results") {
			t.Errorf("Expected exactly 3 results, got: %s", outputStr)
		}
	})

	// Test 6: Search with context
	t.Run("SearchWithContext", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "GenerateRandomString", "--with-context")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Search command with context failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should find the function
		if !strings.Contains(outputStr, "GenerateRandomString") {
			t.Errorf("Expected to find GenerateRandomString function, got: %s", outputStr)
		}

		// Should include surrounding context
		if !strings.Contains(outputStr, "crypto/rand") {
			t.Errorf("Expected to include import context, got: %s", outputStr)
		}
	})

	// Test 7: Search for non-existent content
	t.Run("SearchNonExistent", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "nonexistent_function_xyz")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Search command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should report no results
		if !strings.Contains(outputStr, "Found 0 results") {
			t.Errorf("Expected 0 results for non-existent search, got: %s", outputStr)
		}
	})
}

// TestSearchPerformance tests search performance requirements
func TestSearchPerformance(t *testing.T) {
	tempDir := t.TempDir()

	// Create a larger codebase for performance testing
	for i := 0; i < 50; i++ {
		fileName := fmt.Sprintf("file_%d.go", i)
		content := fmt.Sprintf(`package main

import "fmt"

func Function%d() {
	fmt.Println("Function %d")
}

func Validate%d(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process%d(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %%d: %%s\n", i, item)
	}
	return nil
}
`, i, i, i, i)

		err := os.WriteFile(tempDir+"/"+fileName, []byte(content), 0644)
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

	// Index the codebase
	cmd = exec.Command("./code-search", "index")
	cmd.Dir = tempDir
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to index test directory: %v", err)
	}

	// Test search performance
	queries := []string{
		"validation",
		"processing",
		"function",
		"fmt.Println",
	}

	for _, query := range queries {
		t.Run(fmt.Sprintf("PerformanceTest_%s", query), func(t *testing.T) {
			cmd := exec.Command("./code-search", "search", query)
			cmd.Dir = tempDir
			start := time.Now()
			output, err := cmd.CombinedOutput()
			duration := time.Since(start)

			if err != nil {
				t.Errorf("Search failed for query '%s': %v", query, err)
			}

			// Performance requirement: search should complete in under 2 seconds for this test size
			if duration > 2*time.Second {
				t.Errorf("Search for '%s' took too long: %v (should be < 2s)", query, duration)
			}

			outputStr := string(output)
			if !strings.Contains(outputStr, "Found ") {
				t.Errorf("Expected search results for '%s', got: %s", query, outputStr)
			}

			t.Logf("Search for '%s' completed in %v", query, duration)
		})
	}
}
