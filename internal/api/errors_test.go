package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/phrazzld/scry-api/internal/domain"
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
			name: "wrapped database error with SQL details",
			err: fmt.Errorf(
				"SQL error: %w",
				errors.New("syntax error at line 42 in SELECT * FROM users"),
			),
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
					assert.NotContains(
						t,
						message,
						tt.err.Error(),
						"Error message should not contain the actual error",
					)
				}
			}
		})
	}
}

func TestSanitizeValidationError(t *testing.T) {
	testError := errors.New(
		"Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag",
	)
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

// TestMapErrorToStatusCodeWithCustomErrorTypes tests how error mapping handles custom error types
func TestMapErrorToStatusCodeWithCustomErrorTypes(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "domain validation error",
			err:            domain.NewValidationError("email", "must be valid format", nil),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "domain validation error wrapped",
			err: fmt.Errorf(
				"validation failed: %w",
				domain.NewValidationError("password", "too short", nil),
			),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "card review service error - submit answer",
			err:            card_review.NewSubmitAnswerError("failed to process", nil),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "card review service error - get next card",
			err:            card_review.NewGetNextCardError("database error", nil),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "card review service error wrapping not found",
			err:            card_review.NewGetNextCardError("not found", store.ErrCardNotFound),
			expectedStatus: http.StatusNotFound, // Should check the wrapped error
		},
		{
			name: "store error wrapping validation",
			err: store.NewStoreError(
				"user",
				"create",
				"validation failed",
				domain.ErrValidation,
			),
			expectedStatus: http.StatusBadRequest, // Should check the wrapped domain.ErrValidation
		},
		{
			name:           "store error wrapping not found",
			err:            store.NewStoreError("card", "get", "not found", store.ErrCardNotFound),
			expectedStatus: http.StatusNotFound, // Should check the wrapped store.ErrCardNotFound
		},
		{
			name: "store error wrapping duplicate",
			err: store.NewStoreError(
				"user",
				"create",
				"already exists",
				store.ErrEmailExists,
			),
			expectedStatus: http.StatusConflict, // Should check the wrapped store.ErrEmailExists
		},
		{
			name: "store error with no specific wrapped error",
			err: store.NewStoreError(
				"memo",
				"update",
				"database error",
				errors.New("connection refused"),
			),
			expectedStatus: http.StatusInternalServerError, // Generic error
		},
		{
			name: "deeply nested error",
			err: fmt.Errorf(
				"outer: %w",
				fmt.Errorf(
					"middle: %w",
					store.NewStoreError("user", "get", "lookup failed", store.ErrUserNotFound),
				),
			),
			expectedStatus: http.StatusNotFound, // Should unwrap to the store.ErrUserNotFound
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := MapErrorToStatusCode(tt.err)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

// TestGetSafeErrorMessageWithCustomErrorTypes tests error messages for custom error types
func TestGetSafeErrorMessageWithCustomErrorTypes(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{
			name:            "domain validation error with field",
			err:             domain.NewValidationError("email", "must be valid format", nil),
			expectedMessage: "Invalid email: must be valid format",
		},
		{
			name: "domain validation error without field",
			err: domain.NewValidationError(
				"",
				"validation failed",
				domain.ErrValidation,
			),
			expectedMessage: "validation failed", // Now matches the ValidationError.Message directly
		},
		{
			name: "domain validation error wrapped",
			err: fmt.Errorf(
				"validation failed: %w",
				domain.NewValidationError("password", "too short", nil),
			),
			expectedMessage: "Invalid password: too short",
		},
		{
			name:            "card review service error - submit answer",
			err:             card_review.NewSubmitAnswerError("failed to process", nil),
			expectedMessage: "Card review operation failed",
		},
		{
			name:            "card review service error - get next card",
			err:             card_review.NewGetNextCardError("database error", nil),
			expectedMessage: "Card review operation failed",
		},
		{
			name:            "card review service error wrapping not found",
			err:             card_review.NewGetNextCardError("not found", store.ErrCardNotFound),
			expectedMessage: "Card not found", // Should check the wrapped error
		},
		{
			name: "store error wrapping validation",
			err: store.NewStoreError(
				"user",
				"create",
				"validation failed",
				domain.ErrValidation,
			),
			expectedMessage: "Validation failed", // Should check the wrapped domain.ErrValidation
		},
		{
			name:            "store error wrapping not found",
			err:             store.NewStoreError("card", "get", "not found", store.ErrCardNotFound),
			expectedMessage: "Card not found", // Should check the wrapped store.ErrCardNotFound
		},
		{
			name: "store error wrapping email exists",
			err: store.NewStoreError(
				"user",
				"create",
				"already exists",
				store.ErrEmailExists,
			),
			expectedMessage: "Email already exists", // Should check the wrapped store.ErrEmailExists
		},
		{
			name: "store error with generic error",
			err: store.NewStoreError(
				"memo",
				"update",
				"database error",
				errors.New("connection refused"),
			),
			expectedMessage: "Operation failed: database error", // Now matches the StoreError message format
		},
		{
			name: "deeply nested error",
			err: fmt.Errorf(
				"outer: %w",
				fmt.Errorf(
					"middle: %w",
					store.NewStoreError("user", "get", "lookup failed", store.ErrUserNotFound),
				),
			),
			expectedMessage: "User not found", // Should unwrap to the store.ErrUserNotFound
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := GetSafeErrorMessage(tt.err)
			assert.Equal(t, tt.expectedMessage, message)

			// For errors that should return a generic message, ensure no sensitive details are leaked
			if tt.expectedMessage == "An unexpected error occurred" {
				assert.NotContains(
					t,
					message,
					tt.err.Error(),
					"Error message should not contain the actual error",
				)
			}
		})
	}
}

// TestSanitizeValidationErrorWithCustomTypes tests validation error sanitization with custom types
func TestSanitizeValidationErrorWithCustomTypes(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{
			name:            "domain validation error with field",
			err:             domain.NewValidationError("email", "must be valid format", nil),
			expectedMessage: "Invalid email: must be valid format",
		},
		{
			name:            "domain validation error without field",
			err:             domain.NewValidationError("", "validation failed", nil),
			expectedMessage: "validation failed",
		},
		{
			name: "domain validation error with nil wrapped error",
			err: domain.NewValidationError(
				"password",
				"must be at least 8 characters",
				nil,
			),
			expectedMessage: "Invalid password: must be at least 8 characters",
		},
		{
			name: "domain validation error with specific wrapped error",
			err: domain.NewValidationError(
				"username",
				"must be unique",
				store.ErrDuplicate,
			),
			expectedMessage: "Invalid username: must be unique",
		},
		{
			name: "wrapped domain validation error",
			err: fmt.Errorf(
				"failed to create user: %w",
				domain.NewValidationError("email", "already exists", store.ErrEmailExists),
			),
			expectedMessage: "Invalid email: already exists",
		},
		{
			name:            "non-validation error",
			err:             errors.New("some other error"),
			expectedMessage: "Validation error", // Generic message for non-validation errors
		},
		{
			name: "validator library error format",
			err: errors.New(
				"Key: 'UserRequest.Password' Error:Field validation for 'Password' failed on the 'min' tag",
			),
			expectedMessage: "Invalid Password: too short",
		},
		{
			name:            "malformed validator error",
			err:             errors.New("Field validation for Email failed"),
			expectedMessage: "Validation error", // Fallback for malformed validator error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := SanitizeValidationError(tt.err)
			assert.Equal(t, tt.expectedMessage, message)

			// Verify no sensitive error details are leaked
			if !errors.As(tt.err, new(*domain.ValidationError)) {
				assert.NotContains(
					t,
					message,
					tt.err.Error(),
					"Sanitized message should not contain raw error details",
				)
			}
		})
	}
}
