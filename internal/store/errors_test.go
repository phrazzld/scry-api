package store

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "wrapped generic error",
			err:      fmt.Errorf("failed to do something: %w", errors.New("some error")),
			expected: false,
		},
		{
			name:     "ErrNotFound",
			err:      ErrNotFound,
			expected: true,
		},
		{
			name:     "wrapped ErrNotFound",
			err:      fmt.Errorf("failed to do something: %w", ErrNotFound),
			expected: true,
		},
		{
			name:     "ErrUserNotFound",
			err:      ErrUserNotFound,
			expected: true,
		},
		{
			name:     "wrapped ErrUserNotFound",
			err:      fmt.Errorf("failed to find user: %w", ErrUserNotFound),
			expected: true,
		},
		{
			name:     "ErrCardNotFound",
			err:      ErrCardNotFound,
			expected: true,
		},
		{
			name:     "ErrMemoNotFound",
			err:      ErrMemoNotFound,
			expected: true,
		},
		{
			name:     "ErrUserCardStatsNotFound",
			err:      ErrUserCardStatsNotFound,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsDuplicateError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "ErrDuplicate",
			err:      ErrDuplicate,
			expected: true,
		},
		{
			name:     "wrapped ErrDuplicate",
			err:      fmt.Errorf("failed to create: %w", ErrDuplicate),
			expected: true,
		},
		{
			name:     "ErrEmailExists",
			err:      ErrEmailExists,
			expected: true,
		},
		{
			name:     "wrapped ErrEmailExists",
			err:      fmt.Errorf("failed to create user: %w", ErrEmailExists),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDuplicateError(tt.err); got != tt.expected {
				t.Errorf("IsDuplicateError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStoreError(t *testing.T) {
	// Create a store error
	originalErr := errors.New("database connection failed")
	storeErr := NewStoreError("user", "create", "database error", originalErr)

	// Test Error method
	expectedErrorString := "create operation on user failed: database error: database connection failed"
	if got := storeErr.Error(); got != expectedErrorString {
		t.Errorf("StoreError.Error() = %v, want %v", got, expectedErrorString)
	}

	// Test Unwrap method
	if got := storeErr.Unwrap(); !errors.Is(got, originalErr) {
		t.Errorf("StoreError.Unwrap() not returning original error")
	}

	// Test errors.Is with the wrapped error
	if !errors.Is(storeErr, originalErr) {
		t.Errorf("errors.Is() not recognizing the wrapped error")
	}
}
