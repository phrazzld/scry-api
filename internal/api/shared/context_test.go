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
)

// TraceIDValidation is a helper to validate trace ID properties
func TraceIDValidation(t *testing.T, traceID string, shouldBeEmpty bool) {
	t.Helper()

	if shouldBeEmpty {
		assert.Empty(t, traceID, "Expected empty trace ID")
		return
	}

	// Validate non-empty trace ID properties
	assert.NotEmpty(t, traceID, "Expected non-empty trace ID")
	assert.Len(t, traceID, 32, "Trace ID should be 32 hex characters (16 bytes)")

	// Verify it's valid hex
	decoded, err := hex.DecodeString(traceID)
	assert.NoError(t, err, "Trace ID should be valid hex")
	assert.Len(t, decoded, 16, "Decoded bytes should be 16 bytes long")
}

func TestSetAndGetTraceID(t *testing.T) {
	// Test setting and getting trace ID
	ctx := context.Background()

	// Verify no trace ID in original context
	traceID := GetTraceID(ctx)
	TraceIDValidation(t, traceID, true)

	// Set trace ID
	ctxWithTrace := SetTraceID(ctx)

	// Verify trace ID is now set
	traceID = GetTraceID(ctxWithTrace)
	TraceIDValidation(t, traceID, false)

	// Original context should remain unchanged
	traceID = GetTraceID(ctx)
	TraceIDValidation(t, traceID, true)
}

func TestGetTraceIDWithInvalidContext(t *testing.T) {
	// Test different invalid context values
	testCases := []struct {
		name         string
		contextValue interface{}
	}{
		{
			name:         "integer value",
			contextValue: 123,
		},
		{
			name:         "boolean value",
			contextValue: true,
		},
		{
			name:         "slice value",
			contextValue: []byte{1, 2, 3},
		},
		{
			name:         "nil value",
			contextValue: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), TraceIDKey, tc.contextValue)
			traceID := GetTraceID(ctx)
			TraceIDValidation(t, traceID, true)
		})
	}
}

func TestGenerateTraceID(t *testing.T) {
	// Test generating trace ID
	traceID := generateTraceID()
	TraceIDValidation(t, traceID, false)

	// Generate multiple IDs to ensure uniqueness (probabilistic test)
	// Use a smaller number of iterations to speed up tests but still catch obvious issues
	const iterations = 100
	seen := make(map[string]bool, iterations)

	for i := 0; i < iterations; i++ {
		id := generateTraceID()
		TraceIDValidation(t, id, false)
		assert.False(t, seen[id], "Generated trace IDs should be unique")
		seen[id] = true
	}

	// Verify we have exactly the right number of unique IDs
	assert.Len(t, seen, iterations, "All generated trace IDs should be unique")
}

// mockErrorReader is a mock reader that always fails
type mockErrorReader struct{}

func (m *mockErrorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("simulated rand failure")
}

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

// TestGenerateTraceIDWithFailures tests the fallback mechanism with different failure scenarios
func TestGenerateTraceIDWithFailures(t *testing.T) {
	testCases := []struct {
		name   string
		reader io.Reader
	}{
		{
			name:   "complete read failure",
			reader: &mockErrorReader{},
		},
		{
			name:   "partial read failure",
			reader: io.LimitReader(rand.Reader, TraceIDLength/2),
		},
		{
			name:   "nearly complete read failure",
			reader: io.LimitReader(rand.Reader, TraceIDLength-1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate trace ID using the testable version with the specific reader
			traceID := testableGenerateTraceID(tc.reader)

			// All failure cases should produce valid fallback IDs
			TraceIDValidation(t, traceID, false)
		})
	}
}

// TestFallbackTraceIDUniqueness tests that fallback trace IDs are reasonably unique
func TestFallbackTraceIDUniqueness(t *testing.T) {
	// Use a reasonable number of iterations that's still fast enough for testing
	const iterations = 20
	seen := make(map[string]bool, iterations)

	for i := 0; i < iterations; i++ {
		id := generateFallbackTraceID()
		TraceIDValidation(t, id, false)

		// Small sleep to ensure time-based components change
		time.Sleep(time.Millisecond)

		// Check uniqueness
		assert.False(t, seen[id], "Fallback trace IDs should be unique")
		seen[id] = true
	}

	// Verify all IDs were unique
	assert.Len(t, seen, iterations, "All fallback trace IDs should be unique")
}

// TestTraceIDHexFormat ensures trace IDs are valid hex strings without special formats
func TestTraceIDHexFormat(t *testing.T) {
	// Test both standard and fallback generation methods
	traceIDs := []string{
		generateTraceID(),
		generateFallbackTraceID(),
	}

	for i, id := range traceIDs {
		t.Run(fmt.Sprintf("traceID_%d", i), func(t *testing.T) {
			// Verify correct length
			assert.Len(t, id, 32, "Trace ID should be 32 characters long")

			// Verify it's hexadecimal only
			for _, c := range id {
				assert.Contains(t, "0123456789abcdef", string(c),
					"Trace ID should only contain hex characters")
			}

			// Verify it doesn't contain formatting characters like hyphens
			assert.NotContains(t, id, "-", "Trace ID should not contain hyphens")
			assert.NotContains(t, id, ":", "Trace ID should not contain colons")
			assert.NotContains(t, id, " ", "Trace ID should not contain spaces")
		})
	}
}
