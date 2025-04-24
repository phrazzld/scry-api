package shared

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetAndGetTraceID(t *testing.T) {
	// Test setting and getting trace ID
	ctx := context.Background()

	// Verify no trace ID in original context
	traceID := GetTraceID(ctx)
	assert.Empty(t, traceID, "Expected empty trace ID in original context")

	// Set trace ID
	ctxWithTrace := SetTraceID(ctx)

	// Verify trace ID is now set
	traceID = GetTraceID(ctxWithTrace)
	assert.NotEmpty(t, traceID, "Expected non-empty trace ID after setting")
	assert.Len(t, traceID, 16, "Expected trace ID length to be 16 hex characters")

	// Original context should remain unchanged
	traceID = GetTraceID(ctx)
	assert.Empty(t, traceID, "Expected original context to remain unchanged")
}

func TestGetTraceIDWithInvalidContext(t *testing.T) {
	// Test getting trace ID with invalid context value
	ctx := context.WithValue(context.Background(), TraceIDKey, 123) // Not a string

	traceID := GetTraceID(ctx)
	assert.Empty(t, traceID, "Expected empty trace ID when context has invalid type")
}

func TestGenerateTraceID(t *testing.T) {
	// Test generating trace ID
	traceID := generateTraceID()
	assert.NotEmpty(t, traceID, "Expected non-empty trace ID")
	assert.Len(t, traceID, 16, "Expected trace ID length to be 16 hex characters")

	// Generate another ID to ensure uniqueness (probabilistic test)
	anotherID := generateTraceID()
	assert.NotEmpty(t, anotherID, "Expected non-empty trace ID")
	assert.NotEqual(t, traceID, anotherID, "Expected different trace IDs to be unique")
}
