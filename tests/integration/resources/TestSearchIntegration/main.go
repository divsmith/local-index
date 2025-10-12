//go:build testdata

package main

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
