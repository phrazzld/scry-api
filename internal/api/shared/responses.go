package shared

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/phrazzld/scry-api/internal/redact"
)

// ErrorResponse defines the standard error response structure.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"-"` // Not serialized to JSON, used for logging
	TraceID string `json:"trace_id,omitempty"`
}

// ResponseOption defines a function to customize response behavior.
type ResponseOption func(*responseOptions)

// responseOptions holds configurable options for error responses.
type responseOptions struct {
	elevateLogLevel bool
}

// WithElevatedLogLevel returns a ResponseOption that raises 4xx errors to WARN level
// instead of the default DEBUG level. Use for important operational issues like
// rate limiting or repeated auth failures.
func WithElevatedLogLevel() ResponseOption {
	return func(opts *responseOptions) {
		opts.elevateLogLevel = true
	}
}

// RespondWithJSON writes a JSON response with the given status code and data.
func RespondWithJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// RespondWithError writes a JSON error response with the given status code and message.
// It also sets the TraceID from the request context if available.
func RespondWithError(w http.ResponseWriter, r *http.Request, status int, message string) {
	// Get trace ID from context if available
	traceID := GetTraceID(r.Context())

	// Create the error response
	errorResponse := ErrorResponse{
		Error:   message,
		Code:    status,
		TraceID: traceID,
	}

	// Log the error with trace ID for correlation
	slog.Debug("sending error response",
		"status_code", status,
		"message", message,
		"trace_id", traceID,
		"path", r.URL.Path,
		"method", r.Method)

	RespondWithJSON(w, r, status, errorResponse)
}

// RespondWithErrorAndLog writes a JSON error response and also logs the detailed error.
// This is useful for handling errors where you want to log the full error but only
// expose a sanitized version to the client.
//
// Log level strategy:
// - 5xx errors: Always logged at ERROR level
// - 4xx errors: By default logged at DEBUG level
// - 429 Too Many Requests: Logged at WARN level (operational concern)
// - Other status codes: Logged at DEBUG level
//
// For special cases where 4xx errors need higher visibility (e.g., repeated auth failures),
// use the WithElevatedLogLevel() option to elevate to WARN level.
func RespondWithErrorAndLog(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	userMessage string,
	err error,
	opts ...ResponseOption,
) {
	// Get trace ID from context if available
	traceID := GetTraceID(r.Context())

	// Create the error response with only the safe message
	// Note: We never include the raw error string in the response
	errorResponse := ErrorResponse{
		Error:   userMessage,
		Code:    status,
		TraceID: traceID,
	}

	// Set up common log attributes
	logAttrs := []slog.Attr{
		slog.String("trace_id", traceID),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.Int("status_code", status),
		slog.String("user_message", userMessage),
	}

	// Include the redacted error details (but only in the logs)
	if err != nil {
		// Log the redacted error message
		redactedError := redact.Error(err)
		logAttrs = append(logAttrs, slog.String("error", redactedError))

		// Include the error type for debugging context (safe)
		logAttrs = append(logAttrs, slog.String("error_type", fmt.Sprintf("%T", err)))
	}

	// Initialize response options with defaults
	responseOpts := responseOptions{}

	// Apply any option overrides
	for _, opt := range opts {
		opt(&responseOpts)
	}

	// Set appropriate log level based on status code and options
	logLevel := slog.LevelDebug
	if status >= http.StatusInternalServerError {
		// Log server errors (5xx) at ERROR level
		logLevel = slog.LevelError
	} else if status == http.StatusTooManyRequests {
		// Rate limiting (429) is always an operational concern, log at WARN
		logLevel = slog.LevelWarn
	} else if responseOpts.elevateLogLevel && status >= http.StatusBadRequest && status < http.StatusInternalServerError {
		// Elevated 4xx errors (e.g., repeated auth failures) at WARN level when explicitly requested
		logLevel = slog.LevelWarn
	}

	// Log with the determined level
	slog.LogAttrs(r.Context(), logLevel, "API error response", logAttrs...)

	// Send sanitized response to client
	RespondWithJSON(w, r, status, errorResponse)
}
