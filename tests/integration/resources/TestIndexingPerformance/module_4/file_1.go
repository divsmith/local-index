package module_4

import (
	"fmt"
)

// Function41 performs some operation
func Function41(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}

	// Process the data
	processed := fmt.Sprintf("processed: %s", data)
	fmt.Println(processed)

	return nil
}

// Validate41 validates input data
func Validate41(input string) bool {
	return len(input) > 0 && len(input) < 1000
}

// Process41 handles data processing
func Process41(items []string) ([]string, error) {
	var result []string
	for i, item := range items {
		if Validate41(item) {
			err := Function41(item)
			if err != nil {
				return nil, err
			}
			result = append(result, fmt.Sprintf("%d: %s", i, item))
		}
	}
	return result, nil
}
