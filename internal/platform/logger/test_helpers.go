// Package logger provides structured logging functionality for the application
// using Go's standard library log/slog package.
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// TestLogBuffer is a thread-safe buffer for capturing log output in tests.
type TestLogBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

// Write implements io.Writer for TestLogBuffer.
func (b *TestLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns the buffer contents as a string.
func (b *TestLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Reset clears the buffer contents.
func (b *TestLogBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Reset()
}

// Bytes returns a copy of the buffer contents as a byte slice.
func (b *TestLogBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Bytes()
}

// GetLogEntries parses the buffer contents as JSON log entries.
// Each line is assumed to be a separate JSON log entry.
func (b *TestLogBuffer) GetLogEntries() ([]map[string]interface{}, error) {
	b.mu.Lock()
	logs := b.buf.String()
	b.mu.Unlock()

	lines := strings.Split(logs, "\n")
	entries := make([]map[string]interface{}, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// SetupTestLogger creates a test logger that outputs to a buffer.
// It returns the buffer and a cleanup function.
func SetupTestLogger(t *testing.T, opts *slog.HandlerOptions) (*TestLogBuffer, *slog.Logger, func()) {
	t.Helper()

	// Create a buffer for capturing log output
	logBuf := &TestLogBuffer{}

	// Save the original default logger to restore later
	originalLogger := slog.Default()

	// Create a handler for the test
	// If no options are provided, use reasonable defaults
	if opts == nil {
		opts = &slog.HandlerOptions{
			Level: slog.LevelDebug, // Use debug level to capture all logs
		}
	}

	// Create a handler based on the test environment
	var handler slog.Handler
	if isInCIEnvironment() {
		// Use the CI handler for comprehensive CI env testing
		handler = NewCIHandler(logBuf, opts)
	} else {
		// Use a simple JSON handler for basic tests
		handler = slog.NewJSONHandler(logBuf, opts)
	}

	// Create a logger with the handler
	logger := slog.New(handler)

	// Set this as the default for the test
	slog.SetDefault(logger)

	// Create a cleanup function
	cleanup := func() {
		slog.SetDefault(originalLogger)
	}

	return logBuf, logger, cleanup
}

// SetupTestFailureLogger creates a test failure logger for capturing
// and formatting test failures in CI environments.
func SetupTestFailureLogger(t *testing.T) (*TestLogBuffer, *TestFailureLogger, func()) {
	t.Helper()

	// Setup a test logger
	logBuf, logger, cleanup := SetupTestLogger(t, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true, // Always add source for test failures
	})

	// Create a test failure logger
	failLogger := NewTestFailureLogger(logger)

	return logBuf, failLogger, cleanup
}

// AssertLogContains checks if the log buffer contains specific content.
// If the content is not found, it fails the test with a useful message.
func AssertLogContains(t *testing.T, logBuf *TestLogBuffer, content string) {
	t.Helper()

	logs := logBuf.String()
	if !strings.Contains(logs, content) {
		t.Errorf("Expected log to contain %q, but it doesn't.\nLogs:\n%s", content, logs)
	}
}

// AssertLogField checks if the log entries contain a specific field with a specific value.
// It fails the test if the field is not found or doesn't match the expected value.
func AssertLogField(t *testing.T, logBuf *TestLogBuffer, field string, expected interface{}) {
	t.Helper()

	entries, err := logBuf.GetLogEntries()
	if err != nil {
		t.Fatalf("Failed to parse log entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("No log entries found")
	}

	// Check each entry
	found := false
	for _, entry := range entries {
		if value, ok := entry[field]; ok {
			if value == expected {
				found = true
				break
			}
		}
	}

	if !found {
		t.Errorf("Expected log entries to contain field %q with value %v, but it wasn't found", field, expected)
	}
}

// LogCaptureContext provides a context and logger for capturing logs in tests.
// This is particularly useful for testing structured logging with context.
type LogCaptureContext struct {
	Context context.Context
	Logger  *slog.Logger
	Buffer  *TestLogBuffer
}

// NewLogCaptureContext creates a new context with a logger for capturing logs.
func NewLogCaptureContext(t *testing.T) (*LogCaptureContext, func()) {
	t.Helper()

	// Create a buffer and logger
	logBuf := &TestLogBuffer{}
	handler := slog.NewJSONHandler(logBuf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	logger := slog.New(handler)

	// Create a context with the logger
	ctx := context.Background()
	ctx = WithLogger(ctx, logger)

	// Create a cleanup function
	cleanup := func() {
		// Nothing to clean up for this specific use case
	}

	return &LogCaptureContext{
		Context: ctx,
		Logger:  logger,
		Buffer:  logBuf,
	}, cleanup
}

// LogTestContext creates a testing-specific context with a correlation ID.
// This helps with tracing test logs in CI environments.
func LogTestContext(t *testing.T) context.Context {
	// Create a context with a request ID that includes the test name
	testID := "test-" + t.Name()
	ctx := context.Background()
	return WithRequestID(ctx, testID)
}

// GetTestLogger creates a logger for use in tests.
// The logger is configured to capture all logs at debug level.
func GetTestLogger(t *testing.T) (*slog.Logger, *TestLogBuffer) {
	t.Helper()

	// Create a buffer for capturing log output
	logBuf := &TestLogBuffer{}

	// Configure handler options
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true, // Add source for better test diagnostics
	}

	// Create a handler and logger
	handler := slog.NewJSONHandler(logBuf, opts)
	logger := slog.New(handler)

	return logger, logBuf
}

// CaptureLogs runs the provided function and captures all logs.
// It returns the captured logs as a string.
func CaptureLogs(t *testing.T, fn func(*slog.Logger)) string {
	t.Helper()

	// Create a logger and buffer
	logger, logBuf := GetTestLogger(t)

	// Run the function
	fn(logger)

	// Return the captured logs
	return logBuf.String()
}

// TestLogPipe is a pipe that forwards all logs to a test's log output.
// This is useful for integrating logs with the test output in CI.
type TestLogPipe struct {
	t *testing.T
}

// NewTestLogPipe creates a new TestLogPipe.
func NewTestLogPipe(t *testing.T) *TestLogPipe {
	return &TestLogPipe{t: t}
}

// Write implements io.Writer for TestLogPipe.
func (p *TestLogPipe) Write(data []byte) (int, error) {
	p.t.Log(string(data))
	return len(data), nil
}

// SetupLogToTestOutput configures logging to write directly to the test output.
// This is useful for capturing logs in CI environments.
func SetupLogToTestOutput(t *testing.T) (*slog.Logger, func()) {
	t.Helper()

	// Save the original default logger
	originalLogger := slog.Default()

	// Create a pipe to the test output
	pipe := NewTestLogPipe(t)

	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}

	// Create a handler and logger
	var handler slog.Handler
	if isInCIEnvironment() {
		handler = NewCIHandler(pipe, opts)
	} else {
		handler = slog.NewJSONHandler(pipe, opts)
	}
	logger := slog.New(handler)

	// Set as default
	slog.SetDefault(logger)

	// Create cleanup function
	cleanup := func() {
		slog.SetDefault(originalLogger)
	}

	return logger, cleanup
}

// ParseLogEntry parses a JSON log entry string.
func ParseLogEntry(logLine string) (map[string]interface{}, error) {
	if strings.TrimSpace(logLine) == "" {
		return nil, io.EOF
	}

	var entry map[string]interface{}
	err := json.Unmarshal([]byte(logLine), &entry)
	return entry, err
}

// WriteTestSummary logs a structured test summary at the end of a test.
// This helps with visibility in CI environments.
func WriteTestSummary(t *testing.T, logger *slog.Logger, summary map[string]interface{}) {
	t.Helper()

	// Create a base set of attributes
	attrs := []any{
		"test_name", t.Name(),
		"test_result", "summary",
	}

	// Add all summary fields
	for k, v := range summary {
		attrs = append(attrs, k, v)
	}

	// Log the summary
	logger.Info("TEST SUMMARY", attrs...)
}
