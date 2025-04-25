package shared

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Len(t, traceID, 32, "Expected trace ID length to be 32 hex characters (16 bytes)")

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
	assert.Len(t, traceID, 32, "Expected trace ID length to be 32 hex characters (16 bytes)")

	// Verify trace ID is valid hex
	_, err := hex.DecodeString(traceID)
	assert.NoError(t, err, "Expected valid hex string")

	// Generate multiple IDs to ensure uniqueness (probabilistic test)
	const iterations = 1000
	seen := make(map[string]bool, iterations)

	for i := 0; i < iterations; i++ {
		id := generateTraceID()
		assert.Len(t, id, 32, "Expected all trace IDs to be 32 hex characters")
		assert.False(t, seen[id], "Expected all trace IDs to be unique")
		seen[id] = true
	}

	// Verify we have exactly the right number of unique IDs
	assert.Len(t, seen, iterations, "Expected all generated trace IDs to be unique")
}

// mockErrorReader is a mock reader that always fails
type mockErrorReader struct{}

func (m *mockErrorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("simulated rand failure")
}

// Instead of mocking rand.Reader directly (which is not allowed in Go 1.20+),
// we'll create a more testable version of generateTraceID for testing

// testableGenerateTraceID is a version of generateTraceID that allows for
// injecting a custom reader for testing error cases
func testableGenerateTraceID(reader io.Reader) string {
	b := make([]byte, TraceIDLength)
	n, err := reader.Read(b)

	if err != nil || n != TraceIDLength {
		// Use fallback
		return generateFallbackTraceID()
	}

	return hex.EncodeToString(b)
}

// TestGenerateTraceIDWithRandFailure tests the fallback mechanism when rand.Read fails
func TestGenerateTraceIDWithRandFailure(t *testing.T) {
	// Use our custom reader that always fails
	traceID := testableGenerateTraceID(&mockErrorReader{})

	// Verify fallback ID's properties
	assert.NotEmpty(t, traceID, "Expected non-empty trace ID even after rand failure")
	assert.Len(t, traceID, 32, "Expected trace ID length to be 32 hex characters")

	// Verify it's valid hex
	_, err := hex.DecodeString(traceID)
	assert.NoError(t, err, "Expected valid hex string")
}

// TestGenerateTraceIDWithPartialRead tests the fallback mechanism when rand.Read
// doesn't read enough bytes
func TestGenerateTraceIDWithPartialRead(t *testing.T) {
	// Create a reader that limits to half the bytes needed
	limitReader := io.LimitReader(rand.Reader, TraceIDLength/2)

	// Generate trace ID using the testable version - should use fallback mechanism
	traceID := testableGenerateTraceID(limitReader)

	// Verify fallback ID's properties
	assert.NotEmpty(t, traceID, "Expected non-empty trace ID even after partial read")
	assert.Len(t, traceID, 32, "Expected trace ID length to be 32 hex characters")

	// Verify it's valid hex
	_, err := hex.DecodeString(traceID)
	assert.NoError(t, err, "Expected valid hex string")
}

// TestFallbackTraceIDUniqueness tests that fallback trace IDs are reasonably unique
func TestFallbackTraceIDUniqueness(t *testing.T) {
	const iterations = 100
	seen := make(map[string]bool, iterations)

	for i := 0; i < iterations; i++ {
		id := generateFallbackTraceID()
		assert.Len(t, id, 32, "Expected all fallback trace IDs to be 32 hex characters")
		// Ensure each ID can be decoded as hex
		_, err := hex.DecodeString(id)
		require.NoError(t, err, "Fallback ID must be valid hex")

		// Small sleep to ensure time-based components change
		time.Sleep(time.Millisecond)

		// Check uniqueness
		assert.False(t, seen[id], "Expected all fallback trace IDs to be unique")
		seen[id] = true
	}

	// Verify we have exactly the right number of unique IDs
	assert.Len(t, seen, iterations, "Expected all generated fallback trace IDs to be unique")
}
