package shared

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRespondWithJSON(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		data          interface{}
		expectedCheck func(t *testing.T, body []byte)
	}{
		{
			name:   "successful response",
			status: http.StatusOK,
			data: map[string]interface{}{
				"message": "success",
				"data":    123,
			},
			expectedCheck: func(t *testing.T, body []byte) {
				// Parse as map for flexible field order
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err, "Failed to unmarshal response JSON")

				// Assert each field individually
				assert.Equal(t, "success", response["message"], "Response message field mismatch")
				assert.Equal(t, float64(123), response["data"], "Response data field mismatch")
				assert.Len(t, response, 2, "Response should have exactly 2 fields")
			},
		},
		{
			name:   "empty response",
			status: http.StatusNoContent,
			data:   map[string]interface{}{},
			expectedCheck: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err, "Failed to unmarshal response JSON")
				assert.Empty(t, response, "Response should be an empty object")
			},
		},
		{
			name:   "nil response",
			status: http.StatusOK,
			data:   nil,
			expectedCheck: func(t *testing.T, body []byte) {
				// For nil data, we should get the string "null" in the response
				var nilValue interface{}
				err := json.Unmarshal(body, &nilValue)
				require.NoError(t, err, "Failed to unmarshal response JSON")
				assert.Nil(t, nilValue, "Response should unmarshal to nil")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request and response recorder
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			// Call function
			RespondWithJSON(w, req, tc.status, tc.data)

			// Check status code
			assert.Equal(t, tc.status, w.Code, "Status code mismatch")

			// Check Content-Type header
			assert.Equal(
				t,
				"application/json",
				w.Header().Get("Content-Type"),
				"Content-Type header should be application/json",
			)

			// Use the provided check function to validate response
			tc.expectedCheck(t, w.Body.Bytes())
		})
	}
}

// Test for json encoding errors - this requires a data type that can't be JSON encoded
type UnencodableType struct {
	Circular *UnencodableType
}

func TestRespondWithJSONEncodingError(t *testing.T) {
	// Create request and response recorder
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Create data that will cause encoding error
	data := &UnencodableType{}
	data.Circular = data // Circular reference that will fail to encode

	// Capture logs with a more resilient approach
	var logBuf strings.Builder
	handlerOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // Enable all log levels
	}
	logger := slog.New(slog.NewTextHandler(&logBuf, handlerOpts))
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)

	// Call function
	RespondWithJSON(w, req, http.StatusOK, data)

	// Status code should still be set
	assert.Equal(t, http.StatusOK, w.Code, "Status code should be preserved on encoding error")

	// Content-Type header should be set
	assert.Equal(
		t,
		"application/json",
		w.Header().Get("Content-Type"),
		"Content-Type header should be set on encoding error",
	)

	// Check logs for error - look for key attributes rather than exact message
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "failed to encode JSON response", "Log should indicate encoding failure")
	assert.Contains(t, logOutput, "error", "Log should include error details")

	// The response body should be empty or invalid JSON, but we can't check specific content
	// since it's behavior might vary by json encoder implementation
}

// ErrorResponseCheck is a helper for validating error responses
func ErrorResponseCheck(t *testing.T, body []byte, expectedError string, expectedTraceID string) {
	t.Helper()

	var response ErrorResponse
	err := json.Unmarshal(body, &response)
	require.NoError(t, err, "Failed to unmarshal error response")

	assert.Equal(t, expectedError, response.Error, "Error message mismatch")

	if expectedTraceID == "" {
		assert.Empty(t, response.TraceID, "TraceID should be empty")
	} else {
		assert.Equal(t, expectedTraceID, response.TraceID, "TraceID mismatch")
	}
}

func TestRespondWithError(t *testing.T) {
	// Set up context with trace ID
	ctx := context.WithValue(context.Background(), TraceIDKey, "test-trace-id")
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Call function
	RespondWithError(w, req, http.StatusBadRequest, "Invalid request")

	// Check status code
	assert.Equal(t, http.StatusBadRequest, w.Code, "Status code mismatch")

	// Check Content-Type header
	assert.Equal(
		t,
		"application/json",
		w.Header().Get("Content-Type"),
		"Content-Type header should be application/json",
	)

	// Use helper to check response structure and content
	ErrorResponseCheck(t, w.Body.Bytes(), "Invalid request", "test-trace-id")
}

func TestRespondWithErrorNoTraceID(t *testing.T) {
	// No trace ID in context
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Call function
	RespondWithError(w, req, http.StatusUnauthorized, "Unauthorized")

	// Check status code
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Status code mismatch")

	// Check Content-Type header
	assert.Equal(
		t,
		"application/json",
		w.Header().Get("Content-Type"),
		"Content-Type header should be application/json",
	)

	// Use helper to check response structure and content
	ErrorResponseCheck(t, w.Body.Bytes(), "Unauthorized", "")
}

// LogCheck is a helper for validating log entries
type LogCheck struct {
	Level       string   // Expected log level
	Contains    []string // Strings that should be present in the log
	NotContains []string // Strings that should not be present in the log
}

// ValidateLog checks if a log contains expected content
func ValidateLog(t *testing.T, logOutput string, check LogCheck) {
	t.Helper()

	// Check log level if specified
	if check.Level != "" {
		assert.Contains(t, logOutput, check.Level, "Log should contain expected level: %s", check.Level)
	}

	// Check for strings that should be present
	for _, s := range check.Contains {
		assert.Contains(t, logOutput, s, "Log should contain: %s", s)
	}

	// Check for strings that should not be present
	for _, s := range check.NotContains {
		assert.NotContains(t, logOutput, s, "Log should not contain: %s", s)
	}
}

func TestRespondWithErrorAndLog(t *testing.T) {
	tests := []struct {
		name             string
		statusCode       int
		message          string
		err              error
		expectedLogLevel string
		elevateLogLevel  bool
		logCheck         LogCheck
	}{
		{
			name:             "server error",
			statusCode:       http.StatusInternalServerError,
			message:          "Internal server error",
			err:              errors.New("database connection failed"),
			expectedLogLevel: "ERROR",
			elevateLogLevel:  false,
			logCheck: LogCheck{
				Level:    "ERROR",
				Contains: []string{"Internal server error", "error_type="},
			},
		},
		{
			name:             "client error (4xx) with default log level",
			statusCode:       http.StatusBadRequest,
			message:          "Bad request",
			err:              errors.New("invalid input"),
			expectedLogLevel: "DEBUG", // Changed from WARN to DEBUG per T021
			elevateLogLevel:  false,
			logCheck: LogCheck{
				Level:    "DEBUG",
				Contains: []string{"Bad request", "error_type="},
			},
		},
		{
			name:             "client error (4xx) with elevated log level",
			statusCode:       http.StatusBadRequest,
			message:          "Bad request (elevated)",
			err:              errors.New("invalid input requiring attention"),
			expectedLogLevel: "WARN",
			elevateLogLevel:  true,
			logCheck: LogCheck{
				Level:    "WARN",
				Contains: []string{"Bad request (elevated)", "error_type="},
			},
		},
		{
			name:             "rate limiting error",
			statusCode:       http.StatusTooManyRequests,
			message:          "Too many requests",
			err:              errors.New("rate limit exceeded"),
			expectedLogLevel: "WARN", // 429 is always logged at WARN level
			elevateLogLevel:  false,
			logCheck: LogCheck{
				Level:    "WARN",
				Contains: []string{"Too many requests", "error_type="},
			},
		},
		{
			name:             "redirect",
			statusCode:       http.StatusMovedPermanently,
			message:          "Moved permanently",
			err:              errors.New("redirect error"),
			expectedLogLevel: "DEBUG",
			elevateLogLevel:  false,
			logCheck: LogCheck{
				Level:    "DEBUG",
				Contains: []string{"Moved permanently", "error_type="},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up context with trace ID
			ctx := context.WithValue(context.Background(), TraceIDKey, "test-trace-id")
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			// Capture logs with debug level enabled
			var logBuf strings.Builder
			handlerOpts := &slog.HandlerOptions{
				Level: slog.LevelDebug, // Enable all log levels
			}
			logger := slog.New(slog.NewTextHandler(&logBuf, handlerOpts))
			oldLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(oldLogger)

			// Call function with or without the elevated log level option
			if tc.elevateLogLevel {
				RespondWithErrorAndLog(
					w,
					req,
					tc.statusCode,
					tc.message,
					tc.err,
					WithElevatedLogLevel(),
				)
			} else {
				RespondWithErrorAndLog(w, req, tc.statusCode, tc.message, tc.err)
			}

			// Check response status code
			assert.Equal(t, tc.statusCode, w.Code, "Status code mismatch")

			// Check response structure with helper
			ErrorResponseCheck(t, w.Body.Bytes(), tc.message, "test-trace-id")

			// Check logs using the log validator helper
			logOutput := logBuf.String()

			// Add trace ID check to all test cases
			tc.logCheck.Contains = append(tc.logCheck.Contains, "trace_id=")

			// Validate log with our reusable helper
			ValidateLog(t, logOutput, tc.logCheck)

			// Note: Redaction is tested elsewhere in the codebase (error_redaction_test.go)
			// This test focuses on log levels and the presence of key fields
		})
	}
}

func TestWithElevatedLogLevel(t *testing.T) {
	// Test the option function itself
	opts := responseOptions{}
	WithElevatedLogLevel()(&opts)
	assert.True(t, opts.elevateLogLevel, "WithElevatedLogLevel should set elevateLogLevel to true")
}
