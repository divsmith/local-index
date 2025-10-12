//go:build testdata

package main

import "fmt"

func Function8() {
	fmt.Println("Function 8")
}

func Validate8(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process8(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
