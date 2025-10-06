package main

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
