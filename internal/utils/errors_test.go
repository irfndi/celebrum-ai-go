package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Message: "test error message",
	}

	assert.Equal(t, "test error message", err.Error())
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("validation failed")

	assert.Error(t, err)
	assert.Equal(t, "validation failed", err.Error())

	// Check that it's the correct type
	validationErr, ok := err.(*ValidationError)
	assert.True(t, ok)
	assert.Equal(t, "validation failed", validationErr.Message)
}

func TestNewValidationErrorf(t *testing.T) {
	err := NewValidationErrorf("validation failed for field %s with value %d", "age", 150)

	assert.Error(t, err)
	assert.Equal(t, "validation failed for field age with value 150", err.Error())

	// Check that it's the correct type
	validationErr, ok := err.(*ValidationError)
	assert.True(t, ok)
	assert.Equal(t, "validation failed for field age with value 150", validationErr.Message)
}

func TestValidationError_Struct(t *testing.T) {
	err := ValidationError{
		Message: "struct test",
	}

	assert.Equal(t, "struct test", err.Message)
	assert.Equal(t, "struct test", err.Error())
}
