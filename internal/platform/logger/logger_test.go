// Package logger_test contains tests for the logger package
package logger_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/logger"
)

// testLogBuffer is a synchronized buffer for capturing log output in tests
type testLogBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

// Write implements io.Writer interface for the testLogBuffer
func (b *testLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns the buffer contents as a string
func (b *testLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Reset clears the buffer
func (b *testLogBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Reset()
}

// testSetup contains all test fixtures for logger tests
type testSetup struct {
	// The buffer that captures log output
	buffer *testLogBuffer
	// The original stderr and stdout for restore
	originalStderr *os.File
	originalStdout *os.File
	// Pipes for capturing stderr and stdout (to be used in future tests)
	stderrReader io.Reader //nolint:unused // Will be used in future tests
	stdoutReader io.Reader //nolint:unused // Will be used in future tests
	// The handler used in tests
	handler slog.Handler
}

// global test setup instance for all tests in this package
var testFixture *testSetup

// setupLogCapture redirects stdout/stderr to capturing buffers and
// returns a cleanup function that restores the original stdout/stderr
func setupLogCapture(t *testing.T) (ts *testSetup, cleanup func()) {
	t.Helper()

	if testFixture != nil {
		testFixture.buffer.Reset()
		return testFixture, func() {}
	}

	testFixture = &testSetup{
		buffer: &testLogBuffer{},
	}

	// Save original stdout and stderr
	testFixture.originalStdout = os.Stdout
	testFixture.originalStderr = os.Stderr

	// Create a JSON handler that writes to our test buffer
	testFixture.handler = slog.NewJSONHandler(testFixture.buffer, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Capture all levels in tests
	})

	// Create cleanup function
	cleanup = func() {
		// Reset default logger
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}

	return testFixture, cleanup
}

// parseLogEntry parses a JSON log entry from a string
// Will be used in future tests for parsing and validating log output
//
//nolint:unused // Will be used in future tests
func parseLogEntry(logLine string) (map[string]interface{}, error) {
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(logLine), &entry)
	return entry, err
}

// TestSetup is a placeholder for future tests of the Setup function.
// This will be expanded in subsequent tasks to test log level parsing,
// default log level handling, and JSON output format.
func TestSetup(t *testing.T) {
	// Set up log capture
	_, cleanup := setupLogCapture(t)
	defer cleanup()

	// Example test setup - we'll expand on this in future tasks
	cfg := config.ServerConfig{
		LogLevel: "info",
	}

	// We'll expand this test in the next task to actually test the output
	_, err := logger.Setup(cfg)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// We'll also add output verification in the next task
}
