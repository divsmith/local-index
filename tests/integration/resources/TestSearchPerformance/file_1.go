//go:build testdata

package main

import "fmt"

func Function1() {
	fmt.Println("Function 1")
}

func Validate1(data string) error {
	if data == "" {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}

func Process1(items []string) error {
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)
	}
	return nil
}
