package main

import (
	"fmt"
	"os"
)

func main() {
	// Create and run CLI application
	app := NewCLI()

	if err := app.Run(os.Args[1:]); err != nil {
		var exitCode int = 1 // default error code

		// Check for specific error types
		if cliErr, ok := err.(*CLIError); ok {
			exitCode = int(cliErr.Code)
		} else if IsInvalidArgumentError(err) {
			exitCode = 2
		} else if IsNotFoundError(err) {
			exitCode = 3
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitCode)
	}
}
