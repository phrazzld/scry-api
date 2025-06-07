package store

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInternalError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic_error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "ErrInternal",
			err:      ErrInternal,
			expected: true,
		},
		{
			name:     "wrapped_ErrInternal",
			err:      fmt.Errorf("failed to process: %w", ErrInternal),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInternalError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStoreError_ErrorWithoutWrappedError(t *testing.T) {
	// Test the branch in StoreError.Error that doesn't have a wrapped error
	storeErr := &StoreError{
		Entity:    "user",
		Operation: "create",
		Message:   "validation failed",
		Err:       nil, // No wrapped error
	}

	expected := "create operation on user failed: validation failed"
	result := storeErr.Error()
	assert.Equal(t, expected, result)
}

func TestStoreError_ErrorWithWrappedError(t *testing.T) {
	// Test the branch in StoreError.Error that has a wrapped error
	originalErr := errors.New("database connection failed")
	storeErr := &StoreError{
		Entity:    "card",
		Operation: "update",
		Message:   "database error",
		Err:       originalErr,
	}

	expected := "update operation on card failed: database error: database connection failed"
	result := storeErr.Error()
	assert.Equal(t, expected, result)
}

func TestNewStoreError(t *testing.T) {
	originalErr := errors.New("connection timeout")
	entity := "memo"
	operation := "delete"
	message := "timeout occurred"

	storeErr := NewStoreError(entity, operation, message, originalErr)

	assert.NotNil(t, storeErr)
	assert.Equal(t, entity, storeErr.Entity)
	assert.Equal(t, operation, storeErr.Operation)
	assert.Equal(t, message, storeErr.Message)
	assert.Equal(t, originalErr, storeErr.Err)
}

func TestStoreError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	storeErr := NewStoreError("test", "test", "test", originalErr)

	unwrappedErr := storeErr.Unwrap()
	assert.Equal(t, originalErr, unwrappedErr)
}

func TestStoreError_ErrorsIs(t *testing.T) {
	originalErr := errors.New("database error")
	storeErr := NewStoreError("user", "create", "failed", originalErr)

	// Test that errors.Is works with the wrapped error
	assert.True(t, errors.Is(storeErr, originalErr))

	// Test that errors.Is works with StoreError itself
	assert.True(t, errors.Is(storeErr, storeErr))

	// Test that errors.Is fails with unrelated error
	otherErr := errors.New("other error")
	assert.False(t, errors.Is(storeErr, otherErr))
}

func TestStoreError_ErrorsAs(t *testing.T) {
	originalErr := errors.New("database error")
	storeErr := NewStoreError("user", "create", "failed", originalErr)

	// Test that errors.As works with StoreError
	var targetStoreErr *StoreError
	assert.True(t, errors.As(storeErr, &targetStoreErr))
	assert.Equal(t, storeErr, targetStoreErr)
}
