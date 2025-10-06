package main

import (
	"fmt"
	"os"
)

func main() {
	// Create and run CLI application
	app := NewCLI()

	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}