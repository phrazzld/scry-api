package shared

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"log/slog"
	"time"
)

// Key type for context values
type ContextKey string

// Context keys for various values
const (
	// UserIDContextKey is the context key for the user ID
	UserIDContextKey ContextKey = "userID"

	// TraceIDKey is the key for the trace ID in the request context
	TraceIDKey ContextKey = "traceID"

	// TraceIDLength is the number of bytes used to generate the trace ID
	TraceIDLength = 16 // 32 hex characters
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
// Returns a 32-character hex string (16 bytes) for optimal uniqueness.
// If crypto/rand fails, falls back to a secure alternative based on timestamp
// and process information, but never returns a static value.
func generateTraceID() string {
	b := make([]byte, TraceIDLength)
	n, err := rand.Read(b)

	if err != nil || n != TraceIDLength {
		// Log detailed error with context
		slog.Error("failed to generate secure random trace ID",
			"error", err,
			"bytes_read", n,
			"bytes_requested", TraceIDLength,
			"fallback", "time-based generation")

		// Generate a fallback ID based on timestamp and some entropy
		// This is less secure but better than a static value
		return generateFallbackTraceID()
	}

	return hex.EncodeToString(b)
}

// generateFallbackTraceID creates a trace ID using time and additional
// sources of entropy when the crypto/rand source fails.
// This is less secure than a purely random ID but better than a static value.
func generateFallbackTraceID() string {
	// Create a 16-byte buffer for our fallback ID
	fallbackID := make([]byte, TraceIDLength)

	// Use current timestamp for first 8 bytes (provides chronological uniqueness)
	now := time.Now().UnixNano()
	binary.BigEndian.PutUint64(fallbackID[:8], uint64(now))

	// Use process information for additional 4 bytes
	// This helps distinguish between concurrent requests in the same millisecond
	pid := time.Now().Nanosecond() // Just using nanosecond precision here as an example
	binary.BigEndian.PutUint32(fallbackID[8:12], uint32(pid))

	// Add some additional uniqueness to the last 4 bytes
	// By using a different timestamp measurement
	binary.BigEndian.PutUint32(fallbackID[12:16], uint32(time.Now().Unix()))

	return hex.EncodeToString(fallbackID)
}
