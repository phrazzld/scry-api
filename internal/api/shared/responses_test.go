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
		name         string
		status       int
		data         interface{}
		expectedBody string
	}{
		{
			name:   "successful response",
			status: http.StatusOK,
			data: map[string]interface{}{
				"message": "success",
				"data":    123,
			},
			expectedBody: `{"message":"success","data":123}`,
		},
		{
			name:         "empty response",
			status:       http.StatusNoContent,
			data:         map[string]interface{}{},
			expectedBody: `{}`,
		},
		{
			name:         "nil response",
			status:       http.StatusOK,
			data:         nil,
			expectedBody: `null`,
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
			assert.Equal(t, tc.status, w.Code)

			// Check Content-Type header
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Check response body - unmarshal and verify the structure instead of string matching
			if tc.name == "successful response" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, "success", response["message"])
				assert.Equal(t, float64(123), response["data"])
			} else {
				// For empty or nil responses, just check the content length
				assert.Equal(t, tc.expectedBody+"\n", w.Body.String())
			}
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

	// Capture logs
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
	assert.Equal(t, http.StatusOK, w.Code)

	// Content-Type header should be set
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Check logs for error
	assert.Contains(t, logBuf.String(), "failed to encode JSON response")
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
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Check Content-Type header
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check response content
	assert.Equal(t, "Invalid request", response.Error)
	assert.Equal(t, "test-trace-id", response.TraceID)
}

func TestRespondWithErrorNoTraceID(t *testing.T) {
	// No trace ID in context
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Call function
	RespondWithError(w, req, http.StatusUnauthorized, "Unauthorized")

	// Check response
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// TraceID should be empty
	assert.Equal(t, "Unauthorized", response.Error)
	assert.Empty(t, response.TraceID)
}

func TestRespondWithErrorAndLog(t *testing.T) {
	tests := []struct {
		name             string
		statusCode       int
		message          string
		err              error
		expectedLogLevel string
		elevateLogLevel  bool
	}{
		{
			name:             "server error",
			statusCode:       http.StatusInternalServerError,
			message:          "Internal server error",
			err:              errors.New("database connection failed"),
			expectedLogLevel: "ERROR",
			elevateLogLevel:  false,
		},
		{
			name:             "client error (4xx) with default log level",
			statusCode:       http.StatusBadRequest,
			message:          "Bad request",
			err:              errors.New("invalid input"),
			expectedLogLevel: "DEBUG", // Changed from WARN to DEBUG per T021
			elevateLogLevel:  false,
		},
		{
			name:             "client error (4xx) with elevated log level",
			statusCode:       http.StatusBadRequest,
			message:          "Bad request (elevated)",
			err:              errors.New("invalid input requiring attention"),
			expectedLogLevel: "WARN",
			elevateLogLevel:  true,
		},
		{
			name:             "rate limiting error",
			statusCode:       http.StatusTooManyRequests,
			message:          "Too many requests",
			err:              errors.New("rate limit exceeded"),
			expectedLogLevel: "WARN", // 429 is always logged at WARN level
			elevateLogLevel:  false,
		},
		{
			name:             "redirect",
			statusCode:       http.StatusMovedPermanently,
			message:          "Moved permanently",
			err:              errors.New("redirect error"),
			expectedLogLevel: "DEBUG",
			elevateLogLevel:  false,
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

			// Check response
			assert.Equal(t, tc.statusCode, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tc.message, response.Error)
			assert.Equal(t, "test-trace-id", response.TraceID)

			// Check logs for expected log level
			logOutput := logBuf.String()
			assert.Contains(t, logOutput, tc.expectedLogLevel)
			assert.Contains(t, logOutput, tc.message)
			assert.Contains(t, logOutput, "trace_id=test-trace-id")

			// For the cases with errors, we should find the error_type field
			if tc.err != nil {
				// We now redact the raw error details, but should still find the error_type
				assert.Contains(t, logOutput, "error_type=")
			}
		})
	}
}

func TestWithElevatedLogLevel(t *testing.T) {
	// Test the option function itself
	opts := responseOptions{}
	WithElevatedLogLevel()(&opts)
	assert.True(t, opts.elevateLogLevel)
}
