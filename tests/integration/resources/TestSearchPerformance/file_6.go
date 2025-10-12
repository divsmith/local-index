//go:build testdata

package main

import "fmt"

func Function6() {
	fmt.Println("Function 6")
}

func Validate6(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process6(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
