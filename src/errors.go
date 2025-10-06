package main

import "fmt"

// ExitCode represents the exit code for different error types
type ExitCode int

const (
	ExitCodeSuccess ExitCode = 0
	ExitCodeError   ExitCode = 1
	ExitCodeInvalid ExitCode = 2
	ExitCodeNotFound ExitCode = 3
)

// CLIError represents a CLI error with a specific exit code
type CLIError struct {
	Code    ExitCode
	Message string
	Err     error
}

func (e *CLIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *CLIError) Unwrap() error {
	return e.Err
}

// NewInvalidArgumentError creates a new invalid argument error (exit code 2)
func NewInvalidArgumentError(message string, err error) *CLIError {
	return &CLIError{
		Code:    ExitCodeInvalid,
		Message: message,
		Err:     err,
	}
}

// NewNotFoundError creates a new not found error (exit code 3)
func NewNotFoundError(message string, err error) *CLIError {
	return &CLIError{
		Code:    ExitCodeNotFound,
		Message: message,
		Err:     err,
	}
}

// NewGeneralError creates a new general error (exit code 1)
func NewGeneralError(message string, err error) *CLIError {
	return &CLIError{
		Code:    ExitCodeError,
		Message: message,
		Err:     err,
	}
}

// IsInvalidArgumentError checks if an error is an invalid argument error
func IsInvalidArgumentError(err error) bool {
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr.Code == ExitCodeInvalid
	}
	return false
}

// IsNotFoundError checks if an error is a not found error
func IsNotFoundError(err error) bool {
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr.Code == ExitCodeNotFound
	}
	return false
}