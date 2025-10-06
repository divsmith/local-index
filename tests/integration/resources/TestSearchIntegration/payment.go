package main

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
