package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorHandlingConsistency verifies that all handlers handle errors consistently
// by using the centralized error handling functions.
func TestErrorHandlingConsistency(t *testing.T) {
	// Table-driven test for different error scenarios
	tests := []struct {
		name             string
		err              error
		defaultMsg       string
		expectedStatus   int
		expectedMessage  string
		expectDefaultMsg bool
	}{
		// Authentication errors
		{
			name:            "invalid token",
			err:             auth.ErrInvalidToken,
			defaultMsg:      "Custom default message",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid token",
		},
		{
			name:            "expired token",
			err:             auth.ErrExpiredToken,
			defaultMsg:      "Custom default message",
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid token",
		},
		// Not found errors
		{
			name:            "user not found",
			err:             store.ErrUserNotFound,
			defaultMsg:      "Custom default message",
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "User not found",
		},
		{
			name:            "card not found",
			err:             card_review.ErrCardNotFound,
			defaultMsg:      "Custom default message",
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "Card not found",
		},
		// Validation errors
		{
			name:            "invalid ID",
			err:             domain.ErrInvalidID,
			defaultMsg:      "Custom default message",
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Invalid ID",
		},
		{
			name:            "validation error",
			err:             domain.ErrValidation,
			defaultMsg:      "Custom default message",
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Validation failed",
		},
		{
			name: "field validation error",
			err: domain.NewValidationError(
				"email",
				"must be a valid format",
				nil,
			),
			defaultMsg:      "Custom default message",
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Invalid email: must be a valid format",
		},
		// Server errors
		{
			name:             "unexpected error",
			err:              errors.New("database connection error"),
			defaultMsg:       "Friendly server error message",
			expectedStatus:   http.StatusInternalServerError,
			expectedMessage:  "Friendly server error message",
			expectDefaultMsg: true,
		},
		// Special cases
		{
			name:            "no cards due",
			err:             card_review.ErrNoCardsDue,
			defaultMsg:      "Custom default message",
			expectedStatus:  http.StatusNoContent,
			expectedMessage: "No cards due for review",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a response recorder
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Test HandleAPIError
			HandleAPIError(rr, req, tc.err, tc.defaultMsg)

			// Verify status code
			assert.Equal(t, tc.expectedStatus, rr.Code, "Wrong status code for HandleAPIError")

			// Parse response
			var response map[string]interface{}
			err := json.NewDecoder(rr.Body).Decode(&response)
			require.NoError(t, err, "Failed to decode response")

			// Verify expected message
			errorMsg, ok := response["error"].(string)
			require.True(t, ok, "Error field missing in response")

			if tc.expectDefaultMsg {
				assert.Equal(t, tc.defaultMsg, errorMsg, "Wrong error message for HandleAPIError")
			} else {
				assert.Equal(t, tc.expectedMessage, errorMsg, "Wrong error message for HandleAPIError")
			}
		})
	}
}

// TestValidationErrorConsistency verifies that validation errors are handled
// consistently across handlers.
func TestValidationErrorConsistency(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "domain validation error",
			err: domain.NewValidationError(
				"username",
				"must be at least 3 characters",
				nil,
			),
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Invalid username: must be at least 3 characters",
		},
		{
			name: "generic validation error",
			err: errors.New(
				"Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag",
			),
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Invalid Email: required field",
		},
		{
			name:            "generic validation without field",
			err:             errors.New("validation error"),
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Validation error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a response recorder
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Test HandleValidationError
			HandleValidationError(rr, req, tc.err)

			// Verify status code
			assert.Equal(t, tc.expectedStatus, rr.Code, "Wrong status code for HandleValidationError")

			// Parse response
			var response map[string]interface{}
			err := json.NewDecoder(rr.Body).Decode(&response)
			require.NoError(t, err, "Failed to decode response")

			// Verify expected message
			errorMsg, ok := response["error"].(string)
			require.True(t, ok, "Error field missing in response")
			assert.Equal(t, tc.expectedMessage, errorMsg, "Wrong error message for HandleValidationError")
		})
	}
}

// TestMapErrorToStatusCode verifies the consistent status code mapping
func TestMapErrorToStatusCode_Consistency(t *testing.T) {
	// Map of error types to expected status codes
	errorMap := map[error]int{
		// Authentication errors
		auth.ErrInvalidToken:        http.StatusUnauthorized,
		auth.ErrExpiredToken:        http.StatusUnauthorized,
		auth.ErrInvalidRefreshToken: http.StatusUnauthorized,
		auth.ErrExpiredRefreshToken: http.StatusUnauthorized,
		auth.ErrWrongTokenType:      http.StatusUnauthorized, // Added
		domain.ErrUnauthorized:      http.StatusUnauthorized,

		// Authorization errors
		card_review.ErrCardNotOwned: http.StatusForbidden,

		// Not found errors
		store.ErrUserNotFound:            http.StatusNotFound,
		store.ErrCardNotFound:            http.StatusNotFound,
		store.ErrMemoNotFound:            http.StatusNotFound,
		card_review.ErrCardNotFound:      http.StatusNotFound,
		card_review.ErrCardStatsNotFound: http.StatusNotFound,
		store.ErrNotFound:                http.StatusNotFound,

		// Conflict errors
		store.ErrEmailExists: http.StatusConflict,
		store.ErrDuplicate:   http.StatusConflict,

		// Validation errors
		domain.ErrValidation:           http.StatusBadRequest,
		domain.ErrInvalidID:            http.StatusBadRequest,
		domain.ErrInvalidEmail:         http.StatusBadRequest,
		domain.ErrInvalidPassword:      http.StatusBadRequest,
		domain.ErrEmptyContent:         http.StatusBadRequest, // Added
		domain.ErrInvalidFormat:        http.StatusBadRequest, // Added
		domain.ErrInvalidReviewOutcome: http.StatusBadRequest,
		domain.ErrInvalidCardContent:   http.StatusBadRequest, // Added
		domain.ErrInvalidMemoStatus:    http.StatusBadRequest, // Added
		store.ErrInvalidEntity:         http.StatusBadRequest,
		card_review.ErrInvalidAnswer:   http.StatusBadRequest,

		// Special cases
		card_review.ErrNoCardsDue: http.StatusNoContent,

		// Default case
		errors.New("unknown error"): http.StatusInternalServerError,
	}

	// Verify each error maps to the expected status code
	for err, expectedStatus := range errorMap {
		t.Run(err.Error(), func(t *testing.T) {
			actualStatus := MapErrorToStatusCode(err)
			assert.Equal(t, expectedStatus, actualStatus, "Error %v should map to status %d", err, expectedStatus)
		})
	}

	// Test wrapped errors
	wrappedAuth := errors.New("wrapped: auth error")
	wrappedAuth = errors.New(wrappedAuth.Error() + ": " + auth.ErrInvalidToken.Error())
	assert.Equal(
		t,
		http.StatusInternalServerError,
		MapErrorToStatusCode(wrappedAuth),
		"Wrapped errors should map to 500 unless using errors.Wrap",
	)

	properlyWrapped := errors.New("properly wrapped")
	properlyWrapped = errors.New(properlyWrapped.Error() + ": " + auth.ErrInvalidToken.Error())
	assert.Equal(
		t,
		http.StatusInternalServerError,
		MapErrorToStatusCode(properlyWrapped),
		"String concatenated errors aren't properly wrapped",
	)

	// Test a properly wrapped error using fmt.Errorf with %w
	properWrapped := fmt.Errorf("wrapper: %w", auth.ErrInvalidToken)
	assert.Equal(
		t,
		http.StatusUnauthorized,
		MapErrorToStatusCode(properWrapped),
		"Properly wrapped error should keep original status code",
	)

	// Test nested properly wrapped errors
	nestedWrapped := fmt.Errorf("outer wrapper: %w", fmt.Errorf("inner wrapper: %w", auth.ErrInvalidToken))
	assert.Equal(
		t,
		http.StatusUnauthorized,
		MapErrorToStatusCode(nestedWrapped),
		"Nested wrapped errors should keep original status code",
	)

	// Test domain.ValidationError
	validationErr := domain.NewValidationError("email", "must be valid", nil)
	assert.Equal(
		t,
		http.StatusBadRequest,
		MapErrorToStatusCode(validationErr),
		"ValidationError should map to 400 Bad Request",
	)

	// Test wrapped domain.ValidationError
	wrappedValidationErr := fmt.Errorf("validation failed: %w", validationErr)
	assert.Equal(
		t,
		http.StatusBadRequest,
		MapErrorToStatusCode(wrappedValidationErr),
		"Wrapped ValidationError should map to 400 Bad Request",
	)

	// Test store.StoreError wrapping a known error
	storeErr := store.NewStoreError("user", "create", "failed to create user", store.ErrEmailExists)
	assert.Equal(
		t,
		http.StatusConflict,
		MapErrorToStatusCode(storeErr),
		"StoreError wrapping a known error should use the wrapped error's status code",
	)

	// Test card_review.ServiceError wrapping a known error
	serviceErr := card_review.NewSubmitAnswerError("failed to submit answer", card_review.ErrCardNotFound)
	assert.Equal(
		t,
		http.StatusNotFound,
		MapErrorToStatusCode(serviceErr),
		"ServiceError wrapping a known error should use the wrapped error's status code",
	)
}

// TestGetSafeErrorMessage_Consistency verifies the consistent error message generation
func TestGetSafeErrorMessage_Consistency(t *testing.T) {
	// Map of error types to expected messages
	errorMap := map[error]string{
		// Authentication errors
		auth.ErrInvalidToken:        "Invalid token",
		auth.ErrExpiredToken:        "Invalid token",
		auth.ErrInvalidRefreshToken: "Invalid refresh token",
		auth.ErrExpiredRefreshToken: "Invalid refresh token",
		auth.ErrWrongTokenType:      "Invalid refresh token",
		domain.ErrUnauthorized:      "Unauthorized operation",

		// Authorization errors
		card_review.ErrCardNotOwned: "You do not own this card",

		// Not found errors
		store.ErrUserNotFound:            "User not found",
		store.ErrCardNotFound:            "Card not found",
		store.ErrMemoNotFound:            "Memo not found",
		card_review.ErrCardNotFound:      "Card not found",
		card_review.ErrCardStatsNotFound: "Card statistics not found",
		store.ErrNotFound:                "Resource not found",

		// Conflict errors
		store.ErrEmailExists: "Email already exists",
		store.ErrDuplicate:   "Resource already exists",

		// Validation errors
		domain.ErrValidation:           "Validation failed",
		domain.ErrInvalidID:            "Invalid ID",
		domain.ErrInvalidEmail:         "Invalid email format",
		domain.ErrInvalidPassword:      "Invalid password",
		domain.ErrEmptyContent:         "Content cannot be empty",
		domain.ErrInvalidFormat:        "Invalid format",
		domain.ErrInvalidReviewOutcome: "Invalid review outcome",
		domain.ErrInvalidCardContent:   "Invalid card content",
		domain.ErrInvalidMemoStatus:    "Invalid memo status",
		store.ErrInvalidEntity:         "Invalid entity data",
		card_review.ErrInvalidAnswer:   "Invalid answer",

		// Special cases
		card_review.ErrNoCardsDue: "No cards due for review",

		// Default case
		errors.New("unknown error"): "An unexpected error occurred",
	}

	// Verify each error maps to the expected message
	for err, expectedMessage := range errorMap {
		t.Run(err.Error(), func(t *testing.T) {
			actualMessage := GetSafeErrorMessage(err)
			assert.Equal(t, expectedMessage, actualMessage, "Error %v should map to message '%s'", err, expectedMessage)
		})
	}

	// Test domain.ValidationError with field
	validationErr := domain.NewValidationError("email", "must be valid", nil)
	assert.Equal(t, "Invalid email: must be valid", GetSafeErrorMessage(validationErr))

	// Test wrapped validationErr
	wrappedValidationErr := fmt.Errorf("validation failed: %w", validationErr)
	assert.Equal(t, "Invalid email: must be valid", GetSafeErrorMessage(wrappedValidationErr))

	// Test store.StoreError with wrapped error
	storeErr := store.NewStoreError("user", "get", "failed to get user", store.ErrUserNotFound)
	assert.Equal(t, "User not found", GetSafeErrorMessage(storeErr))

	// Test store.StoreError without known wrapped error
	storeErrUnknown := store.NewStoreError("memo", "update", "database error", errors.New("SQL error"))
	assert.Equal(t, "Operation failed: database error", GetSafeErrorMessage(storeErrUnknown))

	// Test card_review.ServiceError with wrapped error
	serviceErr := card_review.NewSubmitAnswerError("failed to submit answer", card_review.ErrCardNotFound)
	assert.Equal(t, "Card not found", GetSafeErrorMessage(serviceErr))

	// Test card_review.ServiceError without known wrapped error
	serviceErrUnknown := card_review.NewSubmitAnswerError("failed to submit answer", errors.New("database error"))
	assert.Equal(t, "Card review operation failed", GetSafeErrorMessage(serviceErrUnknown))
}

// TestResponseFormat verifies that error responses follow a consistent format
func TestResponseFormat(t *testing.T) {
	// Test cases for different errors but with the same expected format
	testCases := []struct {
		name           string
		err            error
		defaultMsg     string
		expectedStatus int
	}{
		{
			name:           "validation error",
			err:            domain.ErrValidation,
			defaultMsg:     "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found error",
			err:            store.ErrCardNotFound,
			defaultMsg:     "",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "server error with default message",
			err:            errors.New("database error"),
			defaultMsg:     "An error occurred while processing your request",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test request and response recorder
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Add a context with a trace ID
			ctx := r.Context()
			traceID := "test-trace-id"
			ctx = context.WithValue(ctx, shared.TraceIDKey, traceID)
			r = r.WithContext(ctx)

			// Call HandleAPIError
			HandleAPIError(w, r, tc.err, tc.defaultMsg)

			// Check Content-Type header
			assert.Equal(
				t,
				"application/json",
				w.Header().Get("Content-Type"),
				"Content-Type should be application/json",
			)

			// Decode response
			var response map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err, "Failed to decode response")

			// Check response format has expected fields
			assert.Contains(t, response, "error", "Response should contain 'error' field")
			assert.Contains(t, response, "trace_id", "Response should contain 'trace_id' field")
			assert.Equal(t, traceID, response["trace_id"], "trace_id should match expected value")
		})
	}
}

// TestConsistentErrorHandling tests that different error types produce consistent responses
func TestConsistentErrorHandling(t *testing.T) {
	// Create a common request and different errors
	commonErrors := []struct {
		name           string
		err            error
		defaultMsg     string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "validation error",
			err:            domain.NewValidationError("email", "invalid format", nil),
			defaultMsg:     "",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid email: invalid format",
		},
		{
			name:           "not found error",
			err:            store.ErrUserNotFound,
			defaultMsg:     "",
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
		{
			name:           "unauthorized error",
			err:            auth.ErrInvalidToken,
			defaultMsg:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "Invalid token",
		},
		{
			name:           "server error with default message",
			err:            errors.New("database error"),
			defaultMsg:     "Something went wrong",
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Something went wrong",
		},
	}

	for _, ce := range commonErrors {
		t.Run(ce.name, func(t *testing.T) {
			// Create a test trace ID
			traceID := "test-trace-id-" + ce.name

			// First test with HandleAPIError
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Add trace ID to context
			ctx1 := r1.Context()
			ctx1 = context.WithValue(ctx1, shared.TraceIDKey, traceID)
			r1 = r1.WithContext(ctx1)

			HandleAPIError(w1, r1, ce.err, ce.defaultMsg)

			assert.Equal(t, ce.expectedStatus, w1.Code, "Status code mismatch for HandleAPIError")

			var resp1 map[string]interface{}
			err1 := json.NewDecoder(w1.Body).Decode(&resp1)
			require.NoError(t, err1, "Failed to decode response")

			assert.Equal(t, ce.expectedMsg, resp1["error"], "Error message mismatch for HandleAPIError")
			assert.Equal(t, traceID, resp1["trace_id"], "trace_id mismatch in HandleAPIError response")

			// For validation errors, also test HandleValidationError
			if ce.expectedStatus == http.StatusBadRequest && errors.Is(ce.err, domain.ErrValidation) {
				w2 := httptest.NewRecorder()
				r2 := httptest.NewRequest(http.MethodGet, "/test", nil)

				// Add trace ID to context
				ctx2 := r2.Context()
				ctx2 = context.WithValue(ctx2, shared.TraceIDKey, traceID)
				r2 = r2.WithContext(ctx2)

				HandleValidationError(w2, r2, ce.err)

				assert.Equal(t, http.StatusBadRequest, w2.Code, "Status code mismatch for HandleValidationError")

				var resp2 map[string]interface{}
				err2 := json.NewDecoder(w2.Body).Decode(&resp2)
				require.NoError(t, err2, "Failed to decode response")

				// The message may be different for validation errors
				assert.NotEmpty(t, resp2["error"], "Error message missing in HandleValidationError response")
				assert.Equal(t, traceID, resp2["trace_id"], "trace_id mismatch in HandleValidationError response")
			}
		})
	}
}
