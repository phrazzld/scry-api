package api_test

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/phrazzld/scry-api/internal/redact"
	"github.com/stretchr/testify/assert"
)

func TestErrorRedaction(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		contains []string
		omits    []string
	}{
		{
			name: "SQL error details",
			error: errors.New(
				"SQL error: syntax error at line 42 in query SELECT * FROM cards WHERE user_id = '...'",
			),
			contains: []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
			omits:    []string{"syntax error", "line 42", "WHERE user_id"},
		},
		{
			name: "Database connection details",
			error: errors.New(
				"connection error: could not connect to database at postgres://user:password@localhost:5432/db",
			),
			contains: []string{"[REDACTED_SQL_ERROR]", ":5432/db"},
			omits:    []string{"postgres://", "password@"},
		},
		{
			name: "Stack trace details",
			error: fmt.Errorf("runtime error: %w",
				errors.New(
					"panic: invalid memory address or nil pointer dereference [recovered]\n\tstack trace: goroutine 42...",
				)),
			contains: []string{"[REDACTED_SQL_ERROR]"},
			omits:    []string{"goroutine", "panic", "stack trace", "nil pointer"},
		},
		{
			name: "File path details",
			error: fmt.Errorf("file not found: %w",
				errors.New("/var/lib/postgresql/data/base/16384/2619: No such file or directory")),
			contains: []string{"[REDACTED_PATH]"},
			omits:    []string{"/var/lib/postgresql", "16384"},
		},
		{
			name: "AWS credentials",
			error: errors.New(
				"AccessDenied: User: arn:aws:iam::123456789012:user/admin is not authorized; AWSAccessKeyId: AKIAIOSFODNN7EXAMPLE",
			),
			contains: []string{"[REDACTED_KEY]"},
			omits:    []string{"AKIA", "AKIAIOSFODNN7EXAMPLE"},
		},
		{
			name:     "Email addresses",
			error:    errors.New("User with email user@example.com not found"),
			contains: []string{"[REDACTED_EMAIL]"},
			omits:    []string{"user@example.com"},
		},
		{
			name: "Multiple sensitive data types",
			error: errors.New(
				"Error processing request from user@example.com: db connection postgres://admin:secret@db.internal:5432/prod failed, check /var/log/app/errors.log",
			),
			contains: []string{"[REDACTED_EMAIL]", "[REDACTED_CREDENTIAL]", "[REDACTED_PATH]"},
			omits:    []string{"user@example.com", "postgres://", "secret@", "/var/log/app"},
		},
		{
			name: "Deeply wrapped error",
			error: fmt.Errorf(
				"controller error: %w",
				fmt.Errorf(
					"service error: %w",
					fmt.Errorf(
						"repo error: %w",
						errors.New("db error: postgres://user:dbpass@localhost/app"),
					),
				),
			),
			// Update expected output to match the new redaction pattern
			contains: []string{
				"controller error",
				"[REDACTED_SQL_ERROR]",
				"[REDACTED_CREDENTIAL]",
			},
			omits: []string{"postgres://", "dbpass@"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			redactedError := redact.Error(tc.error)

			// Check that redacted error contains expected markers
			for _, pattern := range tc.contains {
				assert.Contains(
					t,
					redactedError,
					pattern,
					"Redacted error should contain '%s'",
					pattern,
				)
			}

			// Check that sensitive patterns are removed
			for _, pattern := range tc.omits {
				assert.NotContains(
					t,
					redactedError,
					pattern,
					"Redacted error should not contain '%s'",
					pattern,
				)
			}

			// Check error type formatting
			errorType := fmt.Sprintf("%T", tc.error)
			logOutput := slog.String("error_type", errorType).String()
			assert.Contains(t, logOutput, errorType, "Logging error_type should work correctly")
		})
	}
}

func TestRedactInLogging(t *testing.T) {
	// Create a test error with sensitive data
	sensitiveError := errors.New("connection string: postgres://admin:password123@localhost/db")

	// Set up a buffer to capture log output
	var logBuf strings.Builder
	handlerOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(&logBuf, handlerOpts))
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)

	// Log the error in different ways
	// 1. Raw error (WRONG)
	slog.Error("Raw error", "error", sensitiveError)

	// 2. Redacted error string (CORRECT)
	redactedError := redact.Error(sensitiveError)
	slog.Error("Redacted error", "error", redactedError)

	// 3. Error type (SAFE)
	slog.Error("Error type", "error_type", fmt.Sprintf("%T", sensitiveError))

	// Check the log output
	logOutput := logBuf.String()

	// First log entry should contain sensitive data (shows what we're preventing)
	assert.Contains(t, logOutput, "postgres://")
	assert.Contains(t, logOutput, "password123")

	// Second log entry should contain redacted data
	assert.Contains(t, logOutput, "[REDACTED_CREDENTIAL]")

	// Third log entry should contain error type
	assert.Contains(t, logOutput, "*errors.errorString")
}
