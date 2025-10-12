//go:build testdata

package main

import "fmt"

func Function42() {
	fmt.Println("Function 42")
}

func Validate42(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process42(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
