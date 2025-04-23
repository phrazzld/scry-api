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
	errorResponse := ErrorResponse{
		Error:   userMessage,
		Code:    status,
		TraceID: traceID,
	}

	// Log the full error details with trace ID for correlation
	log := slog.With(
		slog.String("trace_id", traceID),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.Int("status_code", status),
		slog.String("user_message", userMessage),
	)

	if err != nil {
		log = log.With(slog.String("error", err.Error()))
	}

	log.Debug("error response")

	RespondWithJSON(w, r, status, errorResponse)
}
