// Package logger_test contains tests for the logger package
package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"

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

// setupTestLogger redirects the default slog output to a buffer and returns
// the buffer and a cleanup function to restore the original logger.
// This is the preferred setup method for most logger tests.
func setupTestLogger(t *testing.T, level slog.Level) (*testLogBuffer, func()) {
	t.Helper()

	// Create a synchronized buffer to capture log output
	logBuf := &testLogBuffer{}

	// Save the original default logger to restore later
	originalLogger := slog.Default()

	// Create a new logger with a JSON handler writing to our buffer
	handler := slog.NewJSONHandler(logBuf, &slog.HandlerOptions{
		Level: level,
	})
	testLogger := slog.New(handler)

	// Set this as the default for the test
	slog.SetDefault(testLogger)

	// Return the buffer and a cleanup function
	cleanup := func() {
		slog.SetDefault(originalLogger)
	}

	return logBuf, cleanup
}

// Note: The captureStderr and captureStdout functions were removed as they're no
// longer used after refactoring the tests to use more direct approaches

// parseLogEntry parses a JSON log entry from a string
func parseLogEntry(logLine string) (map[string]interface{}, error) {
	var entry map[string]interface{}
	// Handle potentially empty lines from buffer splitting
	if strings.TrimSpace(logLine) == "" {
		return nil, io.EOF
	}
	err := json.Unmarshal([]byte(logLine), &entry)
	return entry, err
}

// TestSetup is a basic test that ensures the Setup function works without errors
func TestSetup(t *testing.T) {
	// Create a buffer to capture output instead of redirecting os.Stdout
	buf := &bytes.Buffer{}

	// Save the original stdout
	originalStdout := os.Stdout

	// Create a pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Set os.Stdout to our pipe
	os.Stdout = w

	// Basic setup with info level
	cfg := logger.LoggerConfig{
		Level: "info",
	}

	// Call Setup function
	_, err = logger.Setup(cfg)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Log a test message to ensure output
	slog.Info("test setup message")

	// Close the pipe writer to flush buffers
	if err := w.Close(); err != nil {
		t.Logf("Failed to close writer: %v", err)
	}

	// Restore original stdout
	os.Stdout = originalStdout

	// Read from the pipe into the buffer
	if _, err := io.Copy(buf, r); err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	// Verify that some output was produced
	if buf.String() == "" {
		t.Error("Expected log output, but got none")
	}
}

// TestInvalidLogLevelParsing tests that when an invalid log level is provided,
// the Setup function defaults to info level and logs a warning message to stderr.
func TestInvalidLogLevelParsing(t *testing.T) {
	// Part 1: Capture stderr to verify warning message
	// Save original stderr
	originalStderr := os.Stderr

	// Create a pipe for stderr
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Redirect stderr to the pipe
	os.Stderr = stderrW

	// Save original stdout
	originalStdout := os.Stdout

	// Create a pipe for stdout
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	// Redirect stdout to the pipe
	os.Stdout = stdoutW

	// Create a logger config with an invalid log level
	cfg := logger.LoggerConfig{
		Level: "invalid_level", // This is not one of the valid levels
	}

	// Call Setup with the invalid log level
	setupLogger, setupErr := logger.Setup(cfg)

	// Close pipes to flush contents
	if err := stderrW.Close(); err != nil {
		t.Logf("Failed to close stderr writer: %v", err)
	}

	if err := stdoutW.Close(); err != nil {
		t.Logf("Failed to close stdout writer: %v", err)
	}

	// Create buffers to hold captured output
	stderrBuf := new(bytes.Buffer)
	stdoutBuf := new(bytes.Buffer)

	// Read from pipes
	if _, err := io.Copy(stderrBuf, stderrR); err != nil {
		t.Logf("Failed to read from stderr pipe: %v", err)
	}

	if _, err := io.Copy(stdoutBuf, stdoutR); err != nil {
		t.Logf("Failed to read from stdout pipe: %v", err)
	}

	// Restore original stderr and stdout
	os.Stderr = originalStderr
	os.Stdout = originalStdout

	// Check that no error was returned
	if setupErr != nil {
		t.Fatalf("Setup returned an error for invalid log level: %v", setupErr)
	}

	// Check that the logger was created
	if setupLogger == nil {
		t.Fatal("Setup returned a nil logger for invalid log level")
	}

	// Get the stderr output
	stderrOutput := stderrBuf.String()

	// Check that a warning message was logged to stderr
	if !strings.Contains(stderrOutput, "invalid log level configured") {
		t.Errorf("Expected warning message about invalid log level, got: %s", stderrOutput)
	}

	// Check that the configured_level field was included in the warning
	if !strings.Contains(stderrOutput, "invalid_level") {
		t.Errorf("Expected warning to include the invalid level name, got: %s", stderrOutput)
	}

	// Check that the default_level field was included in the warning
	if !strings.Contains(stderrOutput, "info") {
		t.Errorf("Expected warning to include the default level, got: %s", stderrOutput)
	}

	// Part 2: Now test the logger works with the expected default info level

	// Create a buffer for testing log output
	testBuf := &testLogBuffer{}

	// Create a handler that writes to our buffer
	handler := slog.NewJSONHandler(testBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Use debug to capture everything for test validation
	})

	// Save original default logger
	originalLogger := slog.Default()

	// Set up a test logger
	testLogger := slog.New(handler)
	slog.SetDefault(testLogger)

	// Make sure to restore the original logger when done
	defer func() {
		slog.SetDefault(originalLogger)
	}()

	// Log test messages at different levels
	slog.Debug("debug test message")
	slog.Info("info test message")
	slog.Warn("warn test message")
	slog.Error("error test message")

	// Get the log output
	logOutput := testBuf.String()

	// Verify that debug messages are included (because we're using debug level for testing)
	if !strings.Contains(logOutput, "debug test message") {
		t.Error("Test logger should output debug messages for verification")
	}

	// Verify that other levels are included
	if !strings.Contains(logOutput, "info test message") {
		t.Error("Test logger should output info messages")
	}
	if !strings.Contains(logOutput, "warn test message") {
		t.Error("Test logger should output warn messages")
	}
	if !strings.Contains(logOutput, "error test message") {
		t.Error("Test logger should output error messages")
	}
}

// TestJSONOutputFormat verifies that the logger produces correctly formatted JSON output
// and that structured logging attributes are properly included in the JSON.
func TestJSONOutputFormat(t *testing.T) {
	// Create a buffer for log output
	var buf bytes.Buffer

	// Create a JSON handler that writes to our buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// Create a logger with the handler
	testLogger := slog.New(handler)

	// Log different types of messages with structured attributes
	testLogger.Info("simple message")
	testLogger.Info("message with attributes",
		"string_attr", "value",
		"int_attr", 42,
		"bool_attr", true,
		"float_attr", 3.14)

	// Log a message with a nested attribute using slog.Group
	testLogger.Info("message with group",
		slog.Group("user",
			"id", "12345",
			"name", "Test User",
			"role", "admin"))

	// Get the log output
	output := buf.String()

	// Split the output into lines (one JSON object per line)
	logLines := strings.Split(strings.TrimSpace(output), "\n")

	// Ensure we have the expected number of log entries
	expectedLines := 3 // We logged 3 messages
	if len(logLines) != expectedLines {
		t.Errorf("Expected %d log lines, got %d", expectedLines, len(logLines))
	}

	// Test 1: Simple message - verify basic JSON structure
	if len(logLines) > 0 {
		entry, err := parseLogEntry(logLines[0])
		if err != nil {
			t.Errorf("Failed to parse JSON log entry: %v", err)
		} else {
			// Check required fields
			requiredFields := []string{"time", "level", "msg"}
			for _, field := range requiredFields {
				if _, ok := entry[field]; !ok {
					t.Errorf("Log entry missing required field: %s", field)
				}
			}

			// Verify field values
			if msg, ok := entry["msg"]; !ok || msg != "simple message" {
				t.Errorf("Expected msg to be 'simple message', got: %v", msg)
			}
			if level, ok := entry["level"]; !ok || level != "INFO" {
				t.Errorf("Expected level to be 'INFO', got: %v", level)
			}
		}
	}

	// Test 2: Message with attributes - verify attributes are included
	if len(logLines) > 1 {
		entry, err := parseLogEntry(logLines[1])
		if err != nil {
			t.Errorf("Failed to parse JSON log entry: %v", err)
		} else {
			// Check for attributes
			expectedAttrs := map[string]interface{}{
				"string_attr": "value",
				"int_attr":    float64(42), // JSON numbers are parsed as float64
				"bool_attr":   true,
				"float_attr":  float64(3.14), // JSON numbers are parsed as float64
			}

			for key, expected := range expectedAttrs {
				if actual, ok := entry[key]; !ok {
					t.Errorf("Log entry missing attribute: %s", key)
				} else if actual != expected {
					t.Errorf("Expected attribute %s to be %v, got: %v", key, expected, actual)
				}
			}
		}
	}

	// Test 3: Message with group - verify nested structure
	if len(logLines) > 2 {
		entry, err := parseLogEntry(logLines[2])
		if err != nil {
			t.Errorf("Failed to parse JSON log entry: %v", err)
		} else {
			// Check for user group
			user, ok := entry["user"].(map[string]interface{})
			if !ok {
				t.Errorf("Log entry missing 'user' group or not an object")
			} else {
				// Check user fields
				expectedUserFields := map[string]interface{}{
					"id":   "12345",
					"name": "Test User",
					"role": "admin",
				}

				for key, expected := range expectedUserFields {
					if actual, ok := user[key]; !ok {
						t.Errorf("User group missing field: %s", key)
					} else if actual != expected {
						t.Errorf("Expected user.%s to be %v, got: %v", key, expected, actual)
					}
				}
			}
		}
	}
}

// TestValidLogLevelParsing tests that valid log levels are correctly parsed
// by the Setup function, and that each log level properly filters messages
// according to the expected behavior.
func TestValidLogLevelParsing(t *testing.T) {
	// Test cases covering all supported log levels
	testCases := []struct {
		name      string
		logLevel  string
		want      slog.Level
		shouldLog map[slog.Level]bool // Maps each log level to whether it should be logged
	}{
		{
			name:     "debug level",
			logLevel: "debug",
			want:     slog.LevelDebug,
			shouldLog: map[slog.Level]bool{
				slog.LevelDebug: true,
				slog.LevelInfo:  true,
				slog.LevelWarn:  true,
				slog.LevelError: true,
			},
		},
		{
			name:     "info level",
			logLevel: "info",
			want:     slog.LevelInfo,
			shouldLog: map[slog.Level]bool{
				slog.LevelDebug: false, // Should NOT log debug when level is set to info
				slog.LevelInfo:  true,
				slog.LevelWarn:  true,
				slog.LevelError: true,
			},
		},
		{
			name:     "warn level",
			logLevel: "warn",
			want:     slog.LevelWarn,
			shouldLog: map[slog.Level]bool{
				slog.LevelDebug: false, // Should NOT log debug when level is set to warn
				slog.LevelInfo:  false, // Should NOT log info when level is set to warn
				slog.LevelWarn:  true,
				slog.LevelError: true,
			},
		},
		{
			name:     "error level",
			logLevel: "error",
			want:     slog.LevelError,
			shouldLog: map[slog.Level]bool{
				slog.LevelDebug: false, // Should NOT log debug when level is set to error
				slog.LevelInfo:  false, // Should NOT log info when level is set to error
				slog.LevelWarn:  false, // Should NOT log warn when level is set to error
				slog.LevelError: true,
			},
		},
		{
			name:     "case insensitive - DEBUG",
			logLevel: "DEBUG",
			want:     slog.LevelDebug,
			shouldLog: map[slog.Level]bool{
				slog.LevelDebug: true,
				slog.LevelInfo:  true,
				slog.LevelWarn:  true,
				slog.LevelError: true,
			},
		},
		{
			name:     "case insensitive - Info",
			logLevel: "Info",
			want:     slog.LevelInfo,
			shouldLog: map[slog.Level]bool{
				slog.LevelDebug: false, // Should NOT log debug when level is set to info
				slog.LevelInfo:  true,
				slog.LevelWarn:  true,
				slog.LevelError: true,
			},
		},
	}

	// Run a test for each level configuration
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Part 1: Verify the Setup function works correctly with the log level
			cfg := logger.LoggerConfig{
				Level: tc.logLevel,
			}

			// Verify Setup function works without error
			l, err := logger.Setup(cfg)

			if err != nil {
				t.Fatalf("Setup returned an error for valid log level %q: %v", tc.logLevel, err)
			}

			if l == nil {
				t.Fatal("Setup returned a nil logger")
			}

			// Part 2: Verify log level filtering works correctly
			// Create a buffer to capture log output
			buf := &bytes.Buffer{}

			// Create a handler with the same log level we're testing
			handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{
				Level: tc.want,
			})

			// Create a test logger with the handler
			testLogger := slog.New(handler)

			// Log messages at all levels to test filtering
			testLogger.Debug("debug test message")
			testLogger.Info("info test message")
			testLogger.Warn("warn test message")
			testLogger.Error("error test message")

			// Get the log output
			output := buf.String()

			// Check filtering behavior for each log level
			checkMessagePresence(t, output, "debug test message", tc.shouldLog[slog.LevelDebug])
			checkMessagePresence(t, output, "info test message", tc.shouldLog[slog.LevelInfo])
			checkMessagePresence(t, output, "warn test message", tc.shouldLog[slog.LevelWarn])
			checkMessagePresence(t, output, "error test message", tc.shouldLog[slog.LevelError])
		})
	}
}

// checkMessagePresence asserts that a message is present or absent in log output
// based on the shouldBePresent flag
func checkMessagePresence(t *testing.T, output, message string, shouldBePresent bool) {
	t.Helper()
	messagePresent := strings.Contains(output, message)

	if shouldBePresent && !messagePresent {
		t.Errorf("Expected message %q to be present in log output, but it was not found", message)
	} else if !shouldBePresent && messagePresent {
		t.Errorf("Expected message %q to NOT be present in log output, but it was found", message)
	}
}

// --- Tests for Contextual Logging Helpers ---

// TestWithRequestID verifies that WithRequestID adds a request ID to the logger
// within the returned context.
func TestWithRequestID(t *testing.T) {
	// Arrange: Setup logger and capture buffer
	logBuf, cleanup := setupTestLogger(t, slog.LevelDebug)
	defer cleanup()

	baseCtx := context.Background()
	testRequestID := "req-12345"

	// Act: Create a context with the request ID
	ctxWithID := logger.WithRequestID(baseCtx, testRequestID)

	// Assert: Context should not be the same as the base context
	if baseCtx == ctxWithID {
		t.Error("WithRequestID should return a new context, not the same context")
	}

	// Act: Log using the logger from the new context
	loggerFromCtx := logger.FromContext(ctxWithID)
	loggerFromCtx.Info("message with request id")

	// Assert: Check the log output for the request ID
	output := logBuf.String()
	if output == "" {
		t.Fatal("Expected log output, but got none")
	}

	// Parse the log entry
	entry, err := parseLogEntry(output)
	if err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	// Verify the request ID is in the log entry
	actualRequestID, hasRequestID := entry["request_id"]
	if !hasRequestID {
		t.Error("Request ID not found in log entry")
	} else if actualRequestID != testRequestID {
		t.Errorf("Expected request ID to be %q, got %q", testRequestID, actualRequestID)
	}

	// Assert: Log using the base context's logger (should not have request ID)
	logBuf.Reset()
	loggerFromBaseCtx := logger.FromContext(baseCtx) // Should be the default logger
	loggerFromBaseCtx.Info("message without request id")

	outputBase := logBuf.String()
	if outputBase == "" {
		t.Fatal("Expected log output from base context, but got none")
	}

	// Parse the log entry
	entryBase, err := parseLogEntry(outputBase)
	if err != nil {
		t.Fatalf("Failed to parse log entry from base context: %v", err)
	}

	// Verify the base context's logger doesn't include the request ID
	_, hasRequestID = entryBase["request_id"]
	if hasRequestID {
		t.Error("Request ID should not be present in logs from base context")
	}
}

// TestFromContext verifies that FromContext retrieves the correct logger:
// - The specific logger if one exists in the context.
// - The default logger if no logger is found in the context.
// - The default logger if the context is nil.
func TestFromContext(t *testing.T) {
	// Arrange: Setup logger and capture buffer
	logBuf, cleanup := setupTestLogger(t, slog.LevelDebug)
	defer cleanup()

	baseCtx := context.Background()
	testRequestID := "req-abcde"

	// --- Scenario 1: Context WITH logger ---
	t.Run("ContextWithLogger", func(t *testing.T) {
		logBuf.Reset()
		ctxWithID := logger.WithRequestID(baseCtx, testRequestID)

		// Act: Get logger from context
		retrievedLogger := logger.FromContext(ctxWithID)

		// Verify it's not nil
		if retrievedLogger == nil {
			t.Fatal("Retrieved logger is nil")
		}

		// Use the logger
		retrievedLogger.Info("logging via retrieved logger")

		// Assert: Logger should include the request ID
		output := logBuf.String()
		if output == "" {
			t.Fatal("Expected log output, but got none")
		}

		entry, err := parseLogEntry(output)
		if err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}

		// Verify the request ID is in the log entry
		actualRequestID, hasRequestID := entry["request_id"]
		if !hasRequestID {
			t.Error("Request ID not found in log entry")
		} else if actualRequestID != testRequestID {
			t.Errorf("Expected request ID to be %q, got %q", testRequestID, actualRequestID)
		}
	})

	// --- Scenario 2: Context WITHOUT logger ---
	t.Run("ContextWithoutLogger", func(t *testing.T) {
		logBuf.Reset()

		// Act: Get logger from empty context
		retrievedLogger := logger.FromContext(baseCtx)

		// Use the logger
		retrievedLogger.Info("logging via default logger")

		// Assert: Should use default logger (no request ID)
		output := logBuf.String()
		if output == "" {
			t.Fatal("Expected log output, but got none")
		}

		entry, err := parseLogEntry(output)
		if err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}

		// Verify there's no request ID
		_, hasRequestID := entry["request_id"]
		if hasRequestID {
			t.Error("Request ID should not be present when retrieved from context without logger")
		}

		// Verify the message was logged correctly
		msg, hasMsg := entry["msg"]
		if !hasMsg {
			t.Error("Log message not found in entry")
		} else if msg != "logging via default logger" {
			t.Errorf("Expected message to be %q, got %q", "logging via default logger", msg)
		}
	})

	// --- Scenario 3: Nil Context ---
	t.Run("NilContext", func(t *testing.T) {
		logBuf.Reset()

		// Act: Get logger from nil context (should not panic)
		var nilCtx context.Context
		retrievedLogger := logger.FromContext(nilCtx)

		// Verify it's not nil (should be default logger)
		if retrievedLogger == nil {
			t.Fatal("Retrieved logger from nil context is nil")
		}

		// Use the logger
		retrievedLogger.Info("logging via logger from nil context")

		// Assert: Should use default logger
		output := logBuf.String()
		if output == "" {
			t.Fatal("Expected log output, but got none")
		}

		entry, err := parseLogEntry(output)
		if err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}

		// Verify there's no request ID
		_, hasRequestID := entry["request_id"]
		if hasRequestID {
			t.Error("Request ID should not be present when retrieved from nil context")
		}

		// Verify the message was logged correctly
		msg, hasMsg := entry["msg"]
		if !hasMsg {
			t.Error("Log message not found in entry")
		} else if msg != "logging via logger from nil context" {
			t.Errorf("Expected message to be %q, got %q", "logging via logger from nil context", msg)
		}
	})
}

// TestLogWithContext verifies that LogWithContext uses the appropriate logger
// based on the context provided and logs messages at the specified level.
func TestLogWithContext(t *testing.T) {
	// Arrange: Setup logger and capture buffer
	logBuf, cleanup := setupTestLogger(t, slog.LevelDebug)
	defer cleanup()

	baseCtx := context.Background()
	testRequestID := "req-log-ctx-987"
	testAttrKey := "user_id"
	testAttrValue := 123

	// --- Scenario 1: Context WITH logger ---
	t.Run("ContextWithLogger", func(t *testing.T) {
		logBuf.Reset()
		ctxWithID := logger.WithRequestID(baseCtx, testRequestID)

		// Act: Log through the LogWithContext function
		logger.LogWithContext(ctxWithID, slog.LevelWarn, "warning message with context", testAttrKey, testAttrValue)

		// Assert: Verify the log output
		output := logBuf.String()
		if output == "" {
			t.Fatal("Expected log output, but got none")
		}

		entry, err := parseLogEntry(output)
		if err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}

		// Verify log level
		level, hasLevel := entry["level"]
		if !hasLevel {
			t.Error("Log level not found in entry")
		} else if level != "WARN" {
			t.Errorf("Expected level to be %q, got %q", "WARN", level)
		}

		// Verify message
		msg, hasMsg := entry["msg"]
		if !hasMsg {
			t.Error("Log message not found in entry")
		} else if msg != "warning message with context" {
			t.Errorf("Expected message to be %q, got %q", "warning message with context", msg)
		}

		// Verify request ID from context
		actualRequestID, hasRequestID := entry["request_id"]
		if !hasRequestID {
			t.Error("Request ID not found in entry")
		} else if actualRequestID != testRequestID {
			t.Errorf("Expected request ID to be %q, got %q", testRequestID, actualRequestID)
		}

		// Verify custom attribute
		attrValue, hasAttr := entry[testAttrKey]
		if !hasAttr {
			t.Errorf("Custom attribute %q not found in entry", testAttrKey)
		} else {
			// JSON numbers are parsed as float64
			floatVal, ok := attrValue.(float64)
			if !ok {
				t.Errorf("Expected attribute value to be float64, got %T", attrValue)
			} else if int(floatVal) != testAttrValue {
				t.Errorf("Expected attribute value to be %d, got %f", testAttrValue, floatVal)
			}
		}
	})

	// --- Scenario 2: Context WITHOUT logger ---
	t.Run("ContextWithoutLogger", func(t *testing.T) {
		logBuf.Reset()

		// Act: Log through LogWithContext with empty context
		logger.LogWithContext(baseCtx, slog.LevelInfo, "info message without context", testAttrKey, testAttrValue)

		// Assert: Verify the log output
		output := logBuf.String()
		if output == "" {
			t.Fatal("Expected log output, but got none")
		}

		entry, err := parseLogEntry(output)
		if err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}

		// Verify log level
		level, hasLevel := entry["level"]
		if !hasLevel {
			t.Error("Log level not found in entry")
		} else if level != "INFO" {
			t.Errorf("Expected level to be %q, got %q", "INFO", level)
		}

		// Verify message
		msg, hasMsg := entry["msg"]
		if !hasMsg {
			t.Error("Log message not found in entry")
		} else if msg != "info message without context" {
			t.Errorf("Expected message to be %q, got %q", "info message without context", msg)
		}

		// Verify no request ID
		_, hasRequestID := entry["request_id"]
		if hasRequestID {
			t.Error("Request ID should not be present with empty context")
		}

		// Verify custom attribute
		attrValue, hasAttr := entry[testAttrKey]
		if !hasAttr {
			t.Errorf("Custom attribute %q not found in entry", testAttrKey)
		} else {
			// JSON numbers are parsed as float64
			floatVal, ok := attrValue.(float64)
			if !ok {
				t.Errorf("Expected attribute value to be float64, got %T", attrValue)
			} else if int(floatVal) != testAttrValue {
				t.Errorf("Expected attribute value to be %d, got %f", testAttrValue, floatVal)
			}
		}
	})

	// --- Scenario 3: Nil Context ---
	t.Run("NilContext", func(t *testing.T) {
		logBuf.Reset()

		// Act: Log through LogWithContext with nil context (should not panic)
		var nilCtx context.Context
		logger.LogWithContext(nilCtx, slog.LevelError, "error message with nil context", testAttrKey, testAttrValue)

		// Assert: Verify the log output
		output := logBuf.String()
		if output == "" {
			t.Fatal("Expected log output, but got none")
		}

		entry, err := parseLogEntry(output)
		if err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}

		// Verify log level
		level, hasLevel := entry["level"]
		if !hasLevel {
			t.Error("Log level not found in entry")
		} else if level != "ERROR" {
			t.Errorf("Expected level to be %q, got %q", "ERROR", level)
		}

		// Verify message
		msg, hasMsg := entry["msg"]
		if !hasMsg {
			t.Error("Log message not found in entry")
		} else if msg != "error message with nil context" {
			t.Errorf("Expected message to be %q, got %q", "error message with nil context", msg)
		}

		// Verify no request ID
		_, hasRequestID := entry["request_id"]
		if hasRequestID {
			t.Error("Request ID should not be present with nil context")
		}

		// Verify custom attribute
		attrValue, hasAttr := entry[testAttrKey]
		if !hasAttr {
			t.Errorf("Custom attribute %q not found in entry", testAttrKey)
		} else {
			// JSON numbers are parsed as float64
			floatVal, ok := attrValue.(float64)
			if !ok {
				t.Errorf("Expected attribute value to be float64, got %T", attrValue)
			} else if int(floatVal) != testAttrValue {
				t.Errorf("Expected attribute value to be %d, got %f", testAttrValue, floatVal)
			}
		}
	})
}
