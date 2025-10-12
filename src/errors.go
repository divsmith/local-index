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

// Directory-specific error constructors

// NewDirectoryNotFoundError creates a new directory not found error
func NewDirectoryNotFoundError(path string) *CLIError {
	return &CLIError{
		Code:    ExitCodeNotFound,
		Message: fmt.Sprintf("Directory '%s' does not exist", path),
		Err:     nil,
	}
}

// NewPermissionDeniedError creates a new permission denied error
func NewPermissionDeniedError(path string, operation string) *CLIError {
	return &CLIError{
		Code:    ExitCodeError,
		Message: fmt.Sprintf("Permission denied accessing '%s' for %s", path, operation),
		Err:     nil,
	}
}

// NewDirectoryTooLargeError creates a new directory size limit error
func NewDirectoryTooLargeError(path string, size string, limit string) *CLIError {
	return &CLIError{
		Code:    ExitCodeError,
		Message: fmt.Sprintf("Directory '%s' (%s) exceeds size limit (%s)", path, size, limit),
		Err:     nil,
	}
}

// NewTooManyFilesError creates a new file count limit error
func NewTooManyFilesError(path string, count int, limit int) *CLIError {
	return &CLIError{
		Code:    ExitCodeError,
		Message: fmt.Sprintf("Directory '%s' contains %d files, limit is %d", path, count, limit),
		Err:     nil,
	}
}

// NewIndexNotFoundError creates a new index not found error
func NewIndexNotFoundError(path string) *CLIError {
	return &CLIError{
		Code:    ExitCodeNotFound,
		Message: fmt.Sprintf("No index found in directory '%s'", path),
		Err:     nil,
	}
}

// NewIndexCorruptedError creates a new index corrupted error
func NewIndexCorruptedError(path string) *CLIError {
	return &CLIError{
		Code:    ExitCodeError,
		Message: fmt.Sprintf("Index files are corrupted or invalid in directory '%s'", path),
		Err:     nil,
	}
}

// NewIndexLockedError creates a new index locked error
func NewIndexLockedError(path string) *CLIError {
	return &CLIError{
		Code:    ExitCodeError,
		Message: fmt.Sprintf("Directory '%s' is currently being indexed by another process", path),
		Err:     nil,
	}
}

// NewPathTraversalError creates a new path traversal error
func NewPathTraversalError(path string) *CLIError {
	return &CLIError{
		Code:    ExitCodeInvalid,
		Message: fmt.Sprintf("Path traversal detected: '%s'", path),
		Err:     nil,
	}
}