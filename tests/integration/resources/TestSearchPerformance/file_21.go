//go:build testdata

package main

import "fmt"

func Function21() {
	fmt.Println("Function 21")
}

func Validate21(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process21(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
