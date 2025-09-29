package utils

import "fmt"

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message string) error {
	return &ValidationError{
		Message: message,
	}
}

// NewValidationErrorf creates a new validation error with formatting
func NewValidationErrorf(format string, args ...interface{}) error {
	return &ValidationError{
		Message: fmt.Sprintf(format, args...),
	}
}
