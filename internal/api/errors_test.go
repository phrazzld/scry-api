package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestMapErrorToStatusCode(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "nil error",
			err:            nil,
			expectedStatus: http.StatusInternalServerError, // Default to 500 for nil error
		},
		{
			name:           "authentication error",
			err:            auth.ErrInvalidToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "wrapped authentication error",
			err:            fmt.Errorf("failed to authenticate: %w", auth.ErrInvalidToken),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "authorization error",
			err:            card_review.ErrCardNotOwned,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "not found error",
			err:            store.ErrCardNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "conflict error",
			err:            store.ErrEmailExists,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "bad request error",
			err:            store.ErrInvalidEntity,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no cards due error",
			err:            card_review.ErrNoCardsDue,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "unknown error",
			err:            errors.New("unknown error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := MapErrorToStatusCode(tt.err)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestGetSafeErrorMessage(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{
			name:            "nil error",
			err:             nil,
			expectedMessage: "An unexpected error occurred",
		},
		{
			name:            "authentication error",
			err:             auth.ErrInvalidToken,
			expectedMessage: "Invalid token",
		},
		{
			name:            "wrapped authentication error",
			err:             fmt.Errorf("failed due to: %w", auth.ErrInvalidToken),
			expectedMessage: "Invalid token",
		},
		{
			name:            "card not owned error",
			err:             card_review.ErrCardNotOwned,
			expectedMessage: "You do not own this card",
		},
		{
			name:            "unknown error",
			err:             errors.New("database error: connection refused"),
			expectedMessage: "An unexpected error occurred", // Database error details are hidden
		},
		{
			name:            "wrapped database error with SQL details",
			err:             fmt.Errorf("SQL error: %w", errors.New("syntax error at line 42 in SELECT * FROM users")),
			expectedMessage: "An unexpected error occurred", // SQL details are hidden
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := GetSafeErrorMessage(tt.err)
			assert.Equal(t, tt.expectedMessage, message)

			// Verify no sensitive details are leaked
			if tt.err != nil {
				if tt.expectedMessage == "An unexpected error occurred" {
					assert.NotContains(t, message, tt.err.Error(), "Error message should not contain the actual error")
				}
			}
		})
	}
}

func TestSanitizeValidationError(t *testing.T) {
	testError := errors.New("Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag")
	safeMessage := SanitizeValidationError(testError)

	// The sanitized message should not contain the full error details
	assert.NotEqual(t, testError.Error(), safeMessage)

	// It should contain a user-friendly reference to the field
	assert.Contains(t, safeMessage, "Email")

	// Verify that the specific field and tag are present in a user-friendly format
	assert.Equal(t, "Invalid Email: required field", safeMessage)

	// Test with a different format error
	otherError := errors.New("Some other kind of error")
	genericMessage := SanitizeValidationError(otherError)
	assert.Equal(t, "Validation error", genericMessage)
}
