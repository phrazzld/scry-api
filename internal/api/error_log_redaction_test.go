package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/redact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupLogCapture sets up a string builder to capture logs and returns:
// 1. A function to get the captured logs
// 2. A cleanup function to restore the original logger
func setupLogCapture() (func() string, func()) {
	var logBuf strings.Builder
	handlerOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // Enable all log levels
	}
	logger := slog.New(slog.NewTextHandler(&logBuf, handlerOpts))
	oldLogger := slog.Default()
	slog.SetDefault(logger)

	// Return a function to get the log content and a cleanup function
	return func() string {
			return logBuf.String()
		}, func() {
			slog.SetDefault(oldLogger)
		}
}

// sensitiveErrorCreator holds functions that generate errors with sensitive information
var sensitiveErrorCreators = map[string]func() error{
	"database connection": func() error {
		return errors.New("failed to connect to postgres://user:s3cr3tP@ssw0rd@db.example.com:5432/mydb")
	},
	"SQL query": func() error {
		return errors.New(
			"error executing SQL: SELECT * FROM users WHERE email='admin@example.com' AND password='hunter2'",
		)
	},
	"path information": func() error {
		return errors.New("file not found: /home/user/config/.secrets/credentials.json")
	},
	"API key": func() error {
		return errors.New("failed to authenticate with API key: api_key=AbCdEf123456789XyZ")
	},
	"email address": func() error {
		return errors.New("user not found: john.doe@example.com")
	},
	"stack trace": func() error {
		return errors.New("panic: runtime error\ngoroutine 1 [running]:\nmain.main()\n\t/app/main.go:42")
	},
	"wrapped error": func() error {
		inner := errors.New("database error: mysql://root:password123@localhost:3306/app")
		return &domain.ValidationError{
			Message: "validation failed accessing database",
			Field:   "database_url",
			Err:     inner,
		}
	},
	"AWS key": func() error {
		return errors.New("authentication failed with AWS key AKIAIOSFODNN7EXAMPLE")
	},
}

// TestErrorRedactionWithHandleAPIError tests that HandleAPIError properly redacts sensitive information
func TestErrorRedactionWithHandleAPIError(t *testing.T) {
	for name, createError := range sensitiveErrorCreators {
		t.Run(name, func(t *testing.T) {
			// Setup log capture
			getLogs, cleanup := setupLogCapture()
			defer cleanup()

			// Create the error with sensitive information
			sensitiveErr := createError()

			// Create test request and response recorder
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			// Use the centralzied handler
			api.HandleAPIError(w, req, sensitiveErr, "Default error message")

			// Get the captured logs
			logs := getLogs()

			// 1. Verify sensitive information is NOT in the logs
			testRedaction(t, logs, sensitiveErr.Error())

			// 2. Verify the error was properly logged
			assert.Contains(t, logs, "API error response")
		})
	}
}

// TestErrorRedactionWithHandleValidationError tests that HandleValidationError properly redacts
// sensitive information in validation errors
func TestErrorRedactionWithHandleValidationError(t *testing.T) {
	// Create validation errors with sensitive information
	validationErrors := []struct {
		name        string
		createError func() error
	}{
		{
			name: "simple validation error",
			createError: func() error {
				return errors.New("validation failed for email: admin@example.com")
			},
		},
		{
			name: "validator structured error",
			createError: func() error {
				return errors.New(
					"Key: 'User.Password' Error:Field validation for 'Password' failed on the 'min' tag with value 'password123'",
				)
			},
		},
		{
			name: "domain validation error",
			createError: func() error {
				inner := errors.New("invalid format: example.com/users/password=secret123")
				return &domain.ValidationError{
					Message: "invalid URL",
					Field:   "website",
					Err:     inner,
				}
			},
		},
	}

	for _, tc := range validationErrors {
		t.Run(tc.name, func(t *testing.T) {
			// Setup log capture
			getLogs, cleanup := setupLogCapture()
			defer cleanup()

			// Create the validation error with sensitive information
			validationErr := tc.createError()

			// Create test request and response recorder
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			// Use the centralized handler
			api.HandleValidationError(w, req, validationErr)

			// Get the captured logs
			logs := getLogs()

			// Verify sensitive information is NOT in the logs
			testRedaction(t, logs, validationErr.Error())

			// Verify the validation error was properly logged
			assert.Contains(t, logs, "API error response")
			assert.Contains(t, logs, "status_code=400")
		})
	}
}

// TestErrorRedactionWithLiveHandlerScenarios tests realistic handler scenarios
func TestErrorRedactionWithLiveHandlerScenarios(t *testing.T) {
	scenarios := []struct {
		name             string
		setupHandler     func(getLogs func() string) (http.Handler, *httptest.ResponseRecorder)
		request          func() *http.Request
		sensitiveStrings []string
	}{
		{
			name: "auth handler with password error",
			setupHandler: func(getLogs func() string) (http.Handler, *httptest.ResponseRecorder) {
				recorder := httptest.NewRecorder()

				// Mock handler function that simulates the login scenario with password error
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					err := errors.New("password validation failed: password=secretPassword123!")
					api.HandleValidationError(w, r, err)
				})

				return handler, recorder
			},
			request: func() *http.Request {
				loginJSON := `{"email":"user@example.com","password":"bad-password"}`
				req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(loginJSON))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			sensitiveStrings: []string{
				"password=secretPassword123!",
				"bad-password",
				"user@example.com",
			},
		},
		{
			name: "memo handler with SQL error",
			setupHandler: func(getLogs func() string) (http.Handler, *httptest.ResponseRecorder) {
				recorder := httptest.NewRecorder()

				// Mock handler function that simulates SQL error
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Create a context with user ID
					userID := uuid.New()
					ctx := context.WithValue(r.Context(), shared.UserIDContextKey, userID)
					r = r.WithContext(ctx)

					err := errors.New(
						"error creating memo: SQL error: INSERT INTO memos (user_id, text) VALUES ('1234', 'sensitive data with password: secret123')",
					)
					api.HandleAPIError(w, r, err, "Failed to create memo")
				})

				return handler, recorder
			},
			request: func() *http.Request {
				memoJSON := `{"text":"This is a memo with sensitive data"}`
				req, _ := http.NewRequest(http.MethodPost, "/memos", bytes.NewBufferString(memoJSON))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			sensitiveStrings: []string{
				"INSERT INTO",
				"password: secret123",
				"This is a memo with sensitive data",
			},
		},
		{
			name: "card handler with path error",
			setupHandler: func(getLogs func() string) (http.Handler, *httptest.ResponseRecorder) {
				recorder := httptest.NewRecorder()

				// Mock handler function that simulates file path error
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					err := errors.New("error reading card template from /var/lib/app/templates/cards/personal.json")
					api.HandleAPIError(w, r, err, "Failed to process card")
				})

				return handler, recorder
			},
			request: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/cards/next", nil)
				return req
			},
			sensitiveStrings: []string{
				"/var/lib/app/templates",
			},
		},
	}

	// Run each scenario
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Setup log capture
			getLogs, cleanup := setupLogCapture()
			defer cleanup()

			// Set up handler and response recorder
			handler, recorder := scenario.setupHandler(getLogs)

			// Execute the request
			handler.ServeHTTP(recorder, scenario.request())

			// Get the captured logs
			logs := getLogs()

			// Verify response status is set correctly (should be an error status)
			assert.GreaterOrEqual(t, recorder.Code, 400, "Should be an error status code")

			// Verify response is valid JSON
			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err, "Response should be valid JSON")

			// Check that sensitive info is not in the logs
			for _, sensitive := range scenario.sensitiveStrings {
				assert.NotContains(t, logs, sensitive,
					"Logs should not contain sensitive string: %s", sensitive)
			}

			// There should be redaction markers in the logs
			assert.Contains(t, logs, "[REDACTED", "Logs should contain redaction markers")
		})
	}
}

// Helper function to verify redaction is happening properly
func testRedaction(t *testing.T, logs, rawErrorMessage string) {
	// First verify the original message has sensitive information
	assert.NotEmpty(t, rawErrorMessage, "Raw error message should not be empty")

	// Check for common sensitive patterns that should never appear in logs
	sensitivePatterns := []string{
		"password",
		"secret",
		"postgres://",
		"mysql://",
		"@example.com",
		"api_key=",
		"AKIA",
		"SELECT",
		"INSERT",
		"/home/",
		"/var/",
		"stack trace:",
		"goroutine",
	}

	// For each pattern that exists in the original error
	for _, pattern := range sensitivePatterns {
		if strings.Contains(rawErrorMessage, pattern) {
			// Verify it's not present in the logs
			assert.NotContains(t, logs, pattern,
				"Logs should not contain sensitive pattern: %s", pattern)
		}
	}

	// Verify redaction placeholders exist in the logs when sensitive data is present
	redactionMarkers := []string{
		"[REDACTED_CREDENTIAL]",
		"[REDACTED_PATH]",
		"[REDACTED_KEY]",
		"[REDACTED_EMAIL]",
		"[REDACTED_SQL]",
		"[STACK_TRACE_REDACTED]",
	}

	// At least one redaction marker should be present if sensitive data existed
	foundMarker := false
	for _, marker := range redactionMarkers {
		if strings.Contains(logs, marker) {
			foundMarker = true
			break
		}
	}

	// If the raw error contained sensitive information (check against common patterns)
	containsSensitiveData := false
	for _, pattern := range sensitivePatterns {
		if strings.Contains(rawErrorMessage, pattern) {
			containsSensitiveData = true
			break
		}
	}

	// If there was sensitive data, at least one redaction marker should exist
	if containsSensitiveData {
		assert.True(t, foundMarker,
			"Logs should contain at least one redaction marker for sensitive data")
	}
}

// TestDirectErrorLogging tests the behavior of direct error logging without redaction
// This demonstrates what we're trying to prevent and serves as a verification that
// our test setup can detect unredacted errors
func TestDirectErrorLogging(t *testing.T) {
	// Setup log capture
	getLogs, cleanup := setupLogCapture()
	defer cleanup()

	// Create error with sensitive data
	sensitiveErr := errors.New(
		"database connection failed: postgres://admin:secretpassword@db.example.com:5432/production",
	)

	// Log directly WITHOUT redaction - WRONG WAY
	slog.Error("Database error", "error", sensitiveErr)

	// Get logs
	logs := getLogs()

	// Verify sensitive data IS present in this case (showing what we're preventing)
	assert.Contains(t, logs, "postgres://", "Direct logging should expose sensitive data")
	assert.Contains(t, logs, "secretpassword", "Direct logging should expose sensitive data")

	// Now correct way
	// Create a new log entry
	slog.Error("Database error (redacted)", "error", redact.Error(sensitiveErr))

	// Get logs - should contain both entries now
	logs = getLogs()

	// Verify the second log entry DOESN'T contain sensitive data
	assert.Contains(t, logs, "[REDACTED_CREDENTIAL]", "Redacted logging should hide sensitive data")
}
