package shared

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorResponse defines the standard error response structure.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"-"` // Not serialized to JSON, used for logging
	TraceID string `json:"trace_id,omitempty"`
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
func RespondWithErrorAndLog(w http.ResponseWriter, r *http.Request, status int, userMessage string, err error) {
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

	// Include the full error details (but only in the logs)
	if err != nil {
		logAttrs = append(logAttrs, slog.Any("error", err))
	}

	// Set appropriate log level based on status code
	logLevel := slog.LevelDebug
	if status >= http.StatusInternalServerError {
		// Log server errors (5xx) at ERROR level
		logLevel = slog.LevelError
	} else if status >= http.StatusBadRequest {
		// Log client errors (4xx) at WARN level
		logLevel = slog.LevelWarn
	}

	// Log with the determined level
	slog.LogAttrs(r.Context(), logLevel, "API error response", logAttrs...)

	// Send sanitized response to client
	RespondWithJSON(w, r, status, errorResponse)
}
