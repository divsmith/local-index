//go:build testdata

package main

import "fmt"

func Function25() {
	fmt.Println("Function 25")
}

func Validate25(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process25(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
