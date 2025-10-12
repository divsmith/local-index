//go:build testdata

package main

import "fmt"

func Function36() {
	fmt.Println("Function 36")
}

func Validate36(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process36(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
