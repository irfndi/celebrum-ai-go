package utils

import "fmt"

// ValidationError represents an error occurring during data validation.
type ValidationError struct {
	Message string
}

// Error returns the error message string.
func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new ValidationError with a specific message.
//
// Parameters:
//   - message: The validation error message.
//
// Returns:
//   - An error interface wrapping the ValidationError.
func NewValidationError(message string) error {
	return &ValidationError{
		Message: message,
	}
}

// NewValidationErrorf creates a new ValidationError with a formatted message.
//
// Parameters:
//   - format: The format string.
//   - args: Arguments for the format string.
//
// Returns:
//   - An error interface wrapping the ValidationError.
func NewValidationErrorf(format string, args ...interface{}) error {
	return &ValidationError{
		Message: fmt.Sprintf(format, args...),
	}
}
