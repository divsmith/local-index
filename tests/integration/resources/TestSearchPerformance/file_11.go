//go:build testdata

package main

import "fmt"

func Function11() {
	fmt.Println("Function 11")
}

func Validate11(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process11(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
