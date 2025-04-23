package shared

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

// Key type for context values
type ContextKey string

// Context keys for various values
const (
	// UserIDContextKey is the context key for the user ID
	UserIDContextKey ContextKey = "userID"

	// TraceIDKey is the key for the trace ID in the request context
	TraceIDKey ContextKey = "traceID"
)

// SetTraceID adds a trace ID to the context.
// This is useful for correlating logs and error responses.
func SetTraceID(ctx context.Context) context.Context {
	traceID := generateTraceID()
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from the context.
// If no trace ID exists, it returns an empty string.
func GetTraceID(ctx context.Context) string {
	traceID, ok := ctx.Value(TraceIDKey).(string)
	if !ok {
		return ""
	}
	return traceID
}

// generateTraceID creates a random trace ID for request tracking.
func generateTraceID() string {
	b := make([]byte, 8) // 16 hex characters
	_, err := rand.Read(b)
	if err != nil {
		slog.Error("failed to generate trace ID", "error", err)
		return "00000000"
	}
	return hex.EncodeToString(b)
}
