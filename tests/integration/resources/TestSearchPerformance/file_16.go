//go:build testdata

package main

import "fmt"

func Function16() {
	fmt.Println("Function 16")
}

func Validate16(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process16(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
