//go:build testdata

package main

import "fmt"

func Function12() {
	fmt.Println("Function 12")
}

func Validate12(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process12(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
