//go:build integration || test_without_external_deps

package card_review_test

import (
	"errors"
	"testing"

	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/stretchr/testify/assert"
)

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *card_review.ServiceError
		expected string
	}{
		{
			name: "error_with_underlying_error",
			err: &card_review.ServiceError{
				Operation: "test_operation",
				Message:   "test message",
				Err:       errors.New("underlying error"),
			},
			expected: "test_operation operation failed: test message: underlying error",
		},
		{
			name: "error_without_underlying_error",
			err: &card_review.ServiceError{
				Operation: "test_operation",
				Message:   "test message",
				Err:       nil,
			},
			expected: "test_operation operation failed: test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")

	tests := []struct {
		name     string
		err      *card_review.ServiceError
		expected error
	}{
		{
			name: "with_underlying_error",
			err: &card_review.ServiceError{
				Operation: "test_operation",
				Message:   "test message",
				Err:       underlyingErr,
			},
			expected: underlyingErr,
		},
		{
			name: "without_underlying_error",
			err: &card_review.ServiceError{
				Operation: "test_operation",
				Message:   "test message",
				Err:       nil,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Unwrap()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewSubmitAnswerError(t *testing.T) {
	underlyingErr := errors.New("database error")
	message := "failed to process answer"

	serviceErr := card_review.NewSubmitAnswerError(message, underlyingErr)

	assert.NotNil(t, serviceErr)
	assert.Equal(t, "submit_answer", serviceErr.Operation)
	assert.Equal(t, message, serviceErr.Message)
	assert.Equal(t, underlyingErr, serviceErr.Err)

	// Test that it implements error interface correctly
	expectedError := "submit_answer operation failed: failed to process answer: database error"
	assert.Equal(t, expectedError, serviceErr.Error())

	// Test unwrapping
	assert.Equal(t, underlyingErr, serviceErr.Unwrap())
}

func TestNewGetNextCardError(t *testing.T) {
	underlyingErr := errors.New("database error")
	message := "failed to retrieve card"

	serviceErr := card_review.NewGetNextCardError(message, underlyingErr)

	assert.NotNil(t, serviceErr)
	assert.Equal(t, "get_next_card", serviceErr.Operation)
	assert.Equal(t, message, serviceErr.Message)
	assert.Equal(t, underlyingErr, serviceErr.Err)

	// Test that it implements error interface correctly
	expectedError := "get_next_card operation failed: failed to retrieve card: database error"
	assert.Equal(t, expectedError, serviceErr.Error())

	// Test unwrapping
	assert.Equal(t, underlyingErr, serviceErr.Unwrap())
}

func TestServiceError_ErrorsIs(t *testing.T) {
	underlyingErr := errors.New("database connection failed")
	serviceErr := card_review.NewSubmitAnswerError("test message", underlyingErr)

	// Test that errors.Is works with the underlying error
	assert.True(t, errors.Is(serviceErr, underlyingErr))

	// Test that errors.Is works with ServiceError itself
	assert.True(t, errors.Is(serviceErr, serviceErr))

	// Test that errors.Is fails with unrelated error
	otherErr := errors.New("other error")
	assert.False(t, errors.Is(serviceErr, otherErr))
}

func TestServiceError_ErrorsAs(t *testing.T) {
	underlyingErr := errors.New("database error")
	serviceErr := card_review.NewSubmitAnswerError("test message", underlyingErr)

	// Test that errors.As works with ServiceError
	var targetServiceErr *card_review.ServiceError
	assert.True(t, errors.As(serviceErr, &targetServiceErr))
	assert.Equal(t, serviceErr, targetServiceErr)

	// Test that errors.As works with the operation type
	assert.Equal(t, "submit_answer", targetServiceErr.Operation)
}
