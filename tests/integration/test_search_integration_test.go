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
	// Set up test environment
	resourceDir := setupTestEnvironment(t, "TestSearchIntegration")

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
	"fmt"
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
		fullPath := resourceDir + "/" + path
		dir := resourceDir + "/" + filepath.Dir(path)

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

	// Test 1: Index the project
	t.Run("IndexProject", func(t *testing.T) {
		cmd := exec.Command("./code-search", "index")
		cmd.Dir = resourceDir
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
		cmd := exec.Command("./code-search", "search", "ValidateUser")
		cmd.Dir = resourceDir
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

		// Should find user.go file (since ValidateUser is defined there)
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

		t.Logf("Search for 'ValidateUser' completed in %v", duration)
	})

	// Test 3: Search for payment processing
	t.Run("SearchPaymentProcessing", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "ProcessPayment")
		cmd.Dir = resourceDir
		start := time.Now()
		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Search command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should find results in payment.go where ProcessPayment is defined
		if !strings.Contains(outputStr, "payment.go") {
			t.Errorf("Expected to find payment.go, got: %s", outputStr)
		}

		// Should find ProcessPayment functions
		if !strings.Contains(outputStr, "ProcessPayment") {
			t.Errorf("Expected to find ProcessPayment functions, got: %s", outputStr)
		}

		t.Logf("Search for 'ProcessPayment' completed in %v", duration)
	})

	// Test 4: Search with file pattern filtering
	t.Run("SearchWithFilePattern", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "validateAmount", "--file-pattern", "*.go")
		cmd.Dir = resourceDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Search command with file pattern failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should find validation-related code
		if !strings.Contains(outputStr, "validateAmount") {
			t.Errorf("Expected to find validateAmount, got: %s", outputStr)
		}

		if !strings.Contains(outputStr, "payment.go") {
			t.Errorf("Expected to find payment.go where validateAmount is defined, got: %s", outputStr)
		}
	})

	// Test 5: Search with max results limit
	t.Run("SearchWithMaxResults", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "func", "--max-results", "3")
		cmd.Dir = resourceDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Search command with max results failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should limit to showing 3 results
		if !strings.Contains(outputStr, "Showing 3 of") {
			t.Errorf("Expected to show 3 of many results, got: %s", outputStr)
		}
	})

	// Test 6: Search with context
	t.Run("SearchWithContext", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "GenerateRandomString", "--with-context")
		cmd.Dir = resourceDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Search command with context failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should find the function
		if !strings.Contains(outputStr, "GenerateRandomString") {
			t.Errorf("Expected to find GenerateRandomString function, got: %s", outputStr)
		}

		// Should include surrounding context (check if context lines are shown)
		if !strings.Contains(outputStr, "   ") {
			t.Errorf("Expected to see context lines (indented), got: %s", outputStr)
		}
	})

	// Test 7: Search for non-existent content
	t.Run("SearchNonExistent", func(t *testing.T) {
		cmd := exec.Command("./code-search", "search", "nonexistent_function_xyz")
		cmd.Dir = resourceDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Errorf("Search command failed: %v, output: %s", err, string(output))
		}

		outputStr := string(output)

		// Should report no results
		if !strings.Contains(outputStr, "No results found") {
			t.Errorf("Expected no results for non-existent search, got: %s", outputStr)
		}
	})
}

// TestSearchPerformance tests search performance requirements
func TestSearchPerformance(t *testing.T) {
	// Set up test environment
	resourceDir := setupTestEnvironment(t, "TestSearchPerformance")

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

		err := os.WriteFile(resourceDir+"/"+fileName, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
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

	// Index the codebase
	cmd := exec.Command("./code-search", "index")
	cmd.Dir = resourceDir
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
			cmd.Dir = resourceDir
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