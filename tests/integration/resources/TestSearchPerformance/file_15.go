//go:build testdata

package main

import "fmt"

func Function15() {
	fmt.Println("Function 15")
}

func Validate15(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process15(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
