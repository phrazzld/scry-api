package gemini_tests

import (
	"errors"
	"fmt"
	"testing"

	"github.com/phrazzld/scry-api/internal/generation"
	"github.com/stretchr/testify/assert"
)

// Test error wrapping in the generation package
func TestErrorWrapping(t *testing.T) {
	// Test wrapping ErrGenerationFailed
	origErr := errors.New("some underlying error")
	wrappedErr := fmt.Errorf("%w: %v", generation.ErrGenerationFailed, origErr)

	assert.True(
		t,
		errors.Is(wrappedErr, generation.ErrGenerationFailed),
		"Wrapped error should be ErrGenerationFailed",
	)
	assert.Contains(
		t,
		wrappedErr.Error(),
		origErr.Error(),
		"Wrapped error should contain the original error",
	)
}

// Test that error types are distinct
func TestErrorTypes(t *testing.T) {
	errTypes := []error{
		generation.ErrGenerationFailed,
		generation.ErrInvalidResponse,
		generation.ErrContentBlocked,
		generation.ErrTransientFailure,
		generation.ErrInvalidConfig,
	}

	// Verify each error is distinct
	for i, err1 := range errTypes {
		for j, err2 := range errTypes {
			if i != j {
				assert.NotEqual(t, err1, err2, "Errors should be distinct: %v and %v", err1, err2)
			}
		}
	}
}

// Test error message formatting
func TestErrorMessageFormat(t *testing.T) {
	testCases := []struct {
		name     string
		baseErr  error
		details  string
		expected string
	}{
		{
			name:     "Generation failed",
			baseErr:  generation.ErrGenerationFailed,
			details:  "API timeout",
			expected: "failed to generate cards from text: API timeout",
		},
		{
			name:     "Invalid response",
			baseErr:  generation.ErrInvalidResponse,
			details:  "missing cards array",
			expected: "invalid response from language model: missing cards array",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrappedErr := fmt.Errorf("%w: %s", tc.baseErr, tc.details)
			assert.Equal(
				t,
				tc.expected,
				wrappedErr.Error(),
				"Error message formatting should match expected",
			)
		})
	}
}
