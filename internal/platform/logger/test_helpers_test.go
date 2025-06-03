//go:build test_without_external_deps

package logger_test

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestLogBuffer(t *testing.T) {
	buffer := &logger.TestLogBuffer{}

	// Test Write
	data := []byte("test log message")
	n, err := buffer.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	// Test String
	assert.Equal(t, "test log message", buffer.String())

	// Test Bytes
	assert.Equal(t, data, buffer.Bytes())

	// Test Reset
	buffer.Reset()
	assert.Equal(t, "", buffer.String())
	assert.Equal(t, 0, len(buffer.Bytes()))
}

func TestTestLogBuffer_GetLogEntries(t *testing.T) {
	buffer := &logger.TestLogBuffer{}

	// Write multiple JSON log entries
	entry1 := map[string]interface{}{
		"time":  "2025-01-01T12:00:00Z",
		"level": "INFO",
		"msg":   "first message",
	}
	entry2 := map[string]interface{}{
		"time":  "2025-01-01T12:01:00Z",
		"level": "ERROR",
		"msg":   "second message",
	}

	jsonEntry1, _ := json.Marshal(entry1)
	jsonEntry2, _ := json.Marshal(entry2)

	_, _ = buffer.Write(jsonEntry1)
	_, _ = buffer.Write([]byte("\n"))
	_, _ = buffer.Write(jsonEntry2)
	_, _ = buffer.Write([]byte("\n"))

	// Test GetLogEntries
	entries, err := buffer.GetLogEntries()
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Verify first entry
	assert.Equal(t, "INFO", entries[0]["level"])
	assert.Equal(t, "first message", entries[0]["msg"])

	// Verify second entry
	assert.Equal(t, "ERROR", entries[1]["level"])
	assert.Equal(t, "second message", entries[1]["msg"])
}

func TestSetupTestLogger(t *testing.T) {
	buffer, log, cleanup := logger.SetupTestLogger(t, nil)
	defer cleanup()
	assert.NotNil(t, log)
	assert.NotNil(t, buffer)

	// Test logging
	log.Info("test message", "key", "value")

	// Verify the message was captured
	output := buffer.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
}

func TestSetupTestFailureLogger(t *testing.T) {
	buffer, failureLogger, cleanup := logger.SetupTestFailureLogger(t)
	defer cleanup()
	assert.NotNil(t, failureLogger)
	assert.NotNil(t, buffer)

	// Test that we have a failure logger (we can't easily test its actual logging without complex setup)
	assert.NotNil(t, failureLogger)
}

func TestAssertLogContains(t *testing.T) {
	buffer := &logger.TestLogBuffer{}
	_, _ = buffer.Write([]byte("test log message with important info"))

	// Should not panic when the text is found
	assert.NotPanics(t, func() {
		logger.AssertLogContains(t, buffer, "important info")
	})
}

func TestAssertLogField(t *testing.T) {
	buffer := &logger.TestLogBuffer{}

	// Write a JSON log entry with specific fields
	entry := map[string]interface{}{
		"time":   "2025-01-01T12:00:00Z",
		"level":  "INFO",
		"msg":    "test message",
		"userID": "user123",
		"count":  float64(42), // JSON unmarshaling converts numbers to float64
	}

	jsonEntry, _ := json.Marshal(entry)
	_, _ = buffer.Write(jsonEntry)
	_, _ = buffer.Write([]byte("\n"))

	// Test field assertions
	assert.NotPanics(t, func() {
		logger.AssertLogField(t, buffer, "userID", "user123")
	})

	assert.NotPanics(t, func() {
		logger.AssertLogField(t, buffer, "count", float64(42))
	})
}

func TestNewLogCaptureContext(t *testing.T) {
	captureCtx, cleanup := logger.NewLogCaptureContext(t)
	defer cleanup()
	assert.NotNil(t, captureCtx)

	// Test that we can access the buffer
	buffer := captureCtx.Buffer
	assert.NotNil(t, buffer)

	// Test that we can get the context
	ctx := captureCtx.Context
	assert.NotNil(t, ctx)
}

func TestLogTestContext(t *testing.T) {
	ctx := logger.LogTestContext(t)
	assert.NotNil(t, ctx)

	// Test that we can get a logger from this context
	log := logger.FromContext(ctx)
	assert.NotNil(t, log)

	// Test logging (output goes to test output)
	log.Info("test context message")
}

func TestGetTestLogger(t *testing.T) {
	log, buffer := logger.GetTestLogger(t)
	assert.NotNil(t, log)
	assert.NotNil(t, buffer)

	// Test logging
	log.Info("test logger message")

	// Verify message was captured
	output := buffer.String()
	assert.Contains(t, output, "test logger message")
}

func TestCaptureLogs(t *testing.T) {
	output := logger.CaptureLogs(t, func(log *slog.Logger) {
		log.Info("captured message", "key", "value")
		log.Error("captured error", "error_type", "test")
	})

	assert.NotEmpty(t, output)

	// Verify both messages were captured
	assert.Contains(t, output, "captured message")
	assert.Contains(t, output, "captured error")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
	assert.Contains(t, output, "error_type")
	assert.Contains(t, output, "test")
}

func TestNewTestLogPipe(t *testing.T) {
	pipe := logger.NewTestLogPipe(t)
	assert.NotNil(t, pipe)

	// Test Write
	data := []byte("test pipe message")
	n, err := pipe.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
}

func TestSetupLogToTestOutput(t *testing.T) {
	log, cleanup := logger.SetupLogToTestOutput(t)
	defer cleanup()
	assert.NotNil(t, log)

	// Test logging (output goes to test output)
	log.Info("test output message")
}

func TestParseLogEntry(t *testing.T) {
	// Test valid JSON log entry
	validJSON := `{"time":"2025-01-01T12:00:00Z","level":"INFO","msg":"test message","userID":"user123"}`
	entry, err := logger.ParseLogEntry(validJSON)
	assert.NoError(t, err)
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "test message", entry["msg"])
	assert.Equal(t, "user123", entry["userID"])

	// Test invalid JSON
	invalidJSON := `{"invalid": json}`
	_, err = logger.ParseLogEntry(invalidJSON)
	assert.Error(t, err)
}

func TestWriteTestSummary(t *testing.T) {
	// This function writes to stderr, so we can't easily capture it
	// but we can verify it doesn't panic
	log := slog.Default()
	summary := map[string]interface{}{
		"suite":  "TestSuite",
		"total":  10,
		"passed": 8,
		"failed": 2,
	}
	assert.NotPanics(t, func() {
		logger.WriteTestSummary(t, log, summary)
	})
}
