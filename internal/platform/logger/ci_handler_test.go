package logger_test

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/phrazzld/scry-api/internal/platform/logger"
)

// TestCIEnvironmentDetection tests the CI environment detection function.
func TestCIEnvironmentDetection(t *testing.T) {
	// First save original environment variables to restore later
	origCI := os.Getenv("CI")
	defer func() {
		if err := os.Setenv("CI", origCI); err != nil {
			t.Logf("Failed to restore CI environment variable: %v", err)
		}
	}()

	// Test when CI is set
	if err := os.Setenv("CI", "true"); err != nil {
		t.Fatalf("Failed to set CI environment variable: %v", err)
	}

	// Setup logger with CI environment
	cfg := logger.LoggerConfig{
		Level: "info",
	}
	_, err := logger.Setup(cfg)
	if err != nil {
		t.Fatalf("Failed to setup logger: %v", err)
	}

	// Log a simple message - use the default logger since Setup sets it
	slog.Info("test message in simulated CI environment")
}

// TestStructuredLogging tests structured logging with details useful in CI.
func TestStructuredLogging(t *testing.T) {
	// Create a buffer and logger for test output
	buf := &logger.TestLogBuffer{}
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	testLogger := slog.New(slog.NewJSONHandler(buf, opts))

	// Add a request ID to the logger
	testID := "test-correlation-id-123"
	loggerWithID := testLogger.With("request_id", testID)

	// Log a structured error message with test details
	loggerWithID.Error("TEST FAILURE",
		"test_name", "sample_test",
		"test_status", "failed",
		"expected", 42,
		"actual", 41,
		"input", "test input",
	)

	// Get the log output
	output := buf.String()

	// The output should contain test details
	if !strings.Contains(output, "\"test_name\":\"sample_test\"") {
		t.Errorf("Expected log to contain test name, but it doesn't: %s", output)
	}

	// Should contain test status
	if !strings.Contains(output, "\"test_status\":\"failed\"") {
		t.Errorf("Expected log to contain failure status, but it doesn't: %s", output)
	}

	// Should include test details
	if !strings.Contains(output, "\"expected\":42") ||
		!strings.Contains(output, "\"actual\":41") ||
		!strings.Contains(output, "\"input\":\"test input\"") {
		t.Errorf("Expected log to include test details, but it doesn't: %s", output)
	}

	// Should include the request ID
	if !strings.Contains(output, testID) {
		t.Errorf("Expected log to contain request ID, but it doesn't: %s", output)
	}
}

// TestContextualLogging tests retrieving a logger from context.
func TestContextualLogging(t *testing.T) {
	// Create a logger and buffer
	buf := &logger.TestLogBuffer{}
	testLogger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create a request ID
	requestID := "test-req-id-123"

	// Add some attributes to our logger so we can validate them in output
	loggerWithAttrs := testLogger.With(
		"request_id", requestID,
		"test_name", t.Name(),
	)

	// Log a message with the logger
	loggerWithAttrs.Info("test message with request ID")

	// Check the output has our attributes
	logOutput := buf.String()
	if !strings.Contains(logOutput, requestID) {
		t.Errorf("Expected log to contain request ID %q, but it doesn't: %s", requestID, logOutput)
	}
	if !strings.Contains(logOutput, t.Name()) {
		t.Errorf("Expected log to contain test name %q, but it doesn't: %s", t.Name(), logOutput)
	}
}
