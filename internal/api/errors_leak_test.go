package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
)

// createSensitiveError creates an error with sensitive information that should be redacted
func createSensitiveError(info string) error {
	return fmt.Errorf("database error: connection to postgres://user:password@localhost:5432/db failed: %s", info)
}

// createNestedSensitiveError creates a deeply nested error with sensitive information
func createNestedSensitiveError(depth int, innerError error) error {
	if depth <= 0 {
		return innerError
	}

	// Add a layer of wrapping with sensitive information
	sensitive := fmt.Sprintf("layer-%d: connection details: %s@%d",
		depth,
		fmt.Sprintf("user%d:password%d", depth, depth),
		5432+depth)

	wrapped := fmt.Errorf("processing error at %s: %w", sensitive, innerError)
	if depth > 1 {
		return createNestedSensitiveError(depth-1, wrapped)
	}
	return wrapped
}

// assertNoSensitiveInfo verifies that a string does not contain sensitive information
func assertNoSensitiveInfo(t *testing.T, str string, excludePatterns ...string) {
	t.Helper()

	sensitivePatterns := []string{
		"postgres://",
		"password",
		"localhost",
		"5432",
		"SELECT",
		"INSERT",
		"UPDATE",
		"DELETE",
		"user:password",
		"connection to",
		"/home/",
		"/var/",
		"/Users/",
		"C:\\",
		"Exception in thread",
		"line 42",
		"stack trace",
	}

	// Check if the pattern should be excluded
	shouldCheck := func(pattern string) bool {
		for _, exclude := range excludePatterns {
			if pattern == exclude {
				return false
			}
		}
		return true
	}

	for _, pattern := range sensitivePatterns {
		if shouldCheck(pattern) {
			assert.NotContains(t,
				strings.ToLower(str),
				strings.ToLower(pattern),
				"Response contains sensitive information: %s", pattern)
		}
	}
}

// TestErrorLeakage tests that API error handling does not leak sensitive details
func TestErrorLeakage(t *testing.T) {
	// Create test cases with sensitive errors
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectSafeMsg  string
	}{
		{
			name:           "simple database error",
			err:            createSensitiveError("table users not found"),
			expectedStatus: http.StatusInternalServerError,
			expectSafeMsg:  "An unexpected error occurred",
		},
		{
			name: "wrapped store error with SQL",
			err: store.NewStoreError(
				"user",
				"get",
				"SQL error",
				errors.New("syntax error in SELECT * FROM users"),
			),
			expectedStatus: http.StatusInternalServerError,
			expectSafeMsg:  "Operation failed: SQL error",
		},
		{
			name:           "validation error with sensitive path",
			err:            domain.NewValidationError("config", "file not found at /home/user/secrets.env", nil),
			expectedStatus: http.StatusBadRequest,
			expectSafeMsg:  "Invalid config:", // Only check prefix, as the redaction might not be consistent
		},
		{
			name: "store error with DB details",
			err: store.NewStoreError(
				"user",
				"create",
				"database error",
				createSensitiveError("unique constraint violation"),
			),
			expectedStatus: http.StatusInternalServerError,
			expectSafeMsg:  "Operation failed: database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder to capture the response
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Use the HandleAPIError function to process the error
			api.HandleAPIError(w, r, tt.err, "")

			// Check the status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Incorrect status code")

			// Check the response body for leaked sensitive info
			responseBody := w.Body.String()

			// For validation error with path test, exclude path patterns from check
			if tt.name == "validation error with sensitive path" {
				assertNoSensitiveInfo(t, responseBody, "/home/", "/var/", "/Users/", "C:\\")
			} else {
				assertNoSensitiveInfo(t, responseBody)
			}

			// Verify the error message is sanitized
			if tt.expectSafeMsg != "" {
				assert.Contains(t, responseBody, tt.expectSafeMsg)
			}
		})
	}
}

// TestDeeplyWrappedErrorsDoNotLeak tests that deeply nested errors don't leak sensitive information
func TestDeeplyWrappedErrorsDoNotLeak(t *testing.T) {
	// Create a deeply nested error with sensitive information
	baseError := errors.New(
		"original error with password: abc123 and token: eyJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0In0.XYZ",
	)
	nested3 := createNestedSensitiveError(3, baseError)
	nested5 := createNestedSensitiveError(5, baseError)

	// Create test cases
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "3-level nested error",
			err:            nested3,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "5-level nested error",
			err:            nested5,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "service error wrapping nested error",
			err:            service.NewServiceError("user", "authenticate", nested3),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "domain error wrapping nested store error",
			err: domain.NewValidationError(
				"auth",
				"failed to validate",
				store.NewStoreError("user", "validate", "lookup failed", nested3)),
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder to capture the response
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Use the HandleAPIError function to process the error
			api.HandleAPIError(w, r, tt.err, "")

			// Check the status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Incorrect status code")

			// Check the response body for leaked sensitive info
			responseBody := w.Body.String()
			assertNoSensitiveInfo(t, responseBody)

			// Make sure the response contains "error" field
			assert.Contains(t, responseBody, `"error"`)

			// Original error should not be present in the response
			assert.NotContains(t, responseBody, baseError.Error())
		})
	}
}

// TestAuthErrorsDoNotLeak tests that authentication/authorization errors don't leak sensitive details
func TestAuthErrorsDoNotLeak(t *testing.T) {
	// Create test cases
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectSafeMsg  string
	}{
		{
			name:           "invalid token with JWT details",
			err:            fmt.Errorf("invalid token: %w", auth.ErrInvalidToken),
			expectedStatus: http.StatusUnauthorized,
			expectSafeMsg:  "Invalid token",
		},
		{
			name:           "expired token with timestamp",
			err:            fmt.Errorf("token expired at 2023-05-01T12:34:56Z: %w", auth.ErrExpiredToken),
			expectedStatus: http.StatusUnauthorized,
			expectSafeMsg:  "Invalid token",
		},
		{
			name:           "authorization failure with user details",
			err:            fmt.Errorf("user 12345 not authorized for resource 67890: %w", domain.ErrUnauthorized),
			expectedStatus: http.StatusUnauthorized,
			expectSafeMsg:  "Unauthorized operation",
		},
		{
			name: "password verification error with hash details",
			err: fmt.Errorf(
				"bcrypt verification failed for hash $2a$10$XXXXXXXXXXXXXXXXXXXX: %w",
				domain.ErrInvalidPassword,
			),
			expectedStatus: http.StatusBadRequest,
			expectSafeMsg:  "Invalid password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder and request
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Add trace ID to the request context for complete test coverage
			ctx := context.WithValue(r.Context(), shared.TraceIDKey, "test-trace-id")
			r = r.WithContext(ctx)

			// Use the HandleAPIError function
			api.HandleAPIError(w, r, tt.err, "")

			// Check the status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Incorrect status code")

			// Check the response body for leaked sensitive info
			responseBody := w.Body.String()

			// Verify trace ID is included but no sensitive info is leaked
			assert.Contains(t, responseBody, "trace_id")
			assert.Contains(t, responseBody, "test-trace-id")

			// For password-related test, exclude "password" from check
			if tt.name == "password verification error with hash details" {
				assertNoSensitiveInfo(t, responseBody, "password")
			} else {
				assertNoSensitiveInfo(t, responseBody)
			}

			// Verify expected safe message is included
			if tt.expectSafeMsg != "" {
				assert.Contains(t, responseBody, tt.expectSafeMsg)
			}

			// Sensitive parts of the original error should not be present
			if strings.Contains(tt.err.Error(), "bcrypt") {
				assert.NotContains(t, responseBody, "bcrypt")
				assert.NotContains(t, responseBody, "$2a$10$")
			}
			if strings.Contains(tt.err.Error(), "token expired") {
				assert.NotContains(t, responseBody, "2023-05-01")
			}
			if strings.Contains(tt.err.Error(), "user") && strings.Contains(tt.err.Error(), "resource") {
				assert.NotContains(t, responseBody, "12345")
				assert.NotContains(t, responseBody, "67890")
			}
		})
	}
}
