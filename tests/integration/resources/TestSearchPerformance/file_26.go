//go:build testdata

package main

import "fmt"

func Function26() {
	fmt.Println("Function 26")
}

func Validate26(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process26(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
