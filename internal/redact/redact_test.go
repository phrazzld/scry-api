package redact_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/phrazzld/scry-api/internal/redact"
	"github.com/stretchr/testify/assert"
)

func TestRedactString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no sensitive data",
			input:    "This is a normal log message",
			expected: "This is a normal log message",
		},
		{
			name:     "database connection string",
			input:    "Error connecting to postgres://user:password123@localhost:5432/db",
			expected: "Error connecting to [REDACTED_CREDENTIAL]localhost:5432/db",
		},
		{
			name:     "password parameter",
			input:    "Request failed with password=secret123 in payload",
			expected: "Request failed with [REDACTED_CREDENTIAL] in payload",
		},
		{
			name:     "API key",
			input:    "Using api_key=abcdef1234567890ghijklmnop for authentication",
			expected: "Using [REDACTED_KEY] for authentication",
		},
		{
			name:     "AWS access key",
			input:    "AWS credentials: AKIAIOSFODNN7EXAMPLE",
			expected: "AWS credentials: [REDACTED_KEY]",
		},
		{
			name:     "JWT token",
			input:    "Invalid token format: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expected: "Invalid token format: Bearer [REDACTED_JWT]",
		},
		{
			name:     "file path",
			input:    "File not found at /var/lib/postgresql/data/pg_hba.conf",
			expected: "[REDACTED_FILE_ERROR] at [REDACTED_PATH]",
		},
		{
			name:     "Windows path",
			input:    "Access denied to C:\\Program Files\\App\\config.json",
			expected: "Access denied to [REDACTED_PATH]",
		},
		{
			name:     "stack trace",
			input:    "panic: runtime error\ngoroutine 1 [running]:\nmain.main()\n\t/app/main.go:42",
			expected: "[STACK_TRACE_REDACTED]",
		},
		{
			name:     "email address",
			input:    "User admin@example.com not found",
			expected: "User [REDACTED_EMAIL] not found",
		},
		// Enhanced SQL redaction test cases (contingency approach)
		{
			name:     "SQL SELECT with WHERE clause",
			input:    "Error executing: SELECT * FROM users WHERE email = 'user@example.com'",
			expected: "Error executing: SELECT FROM... [SQL_VALUES_REDACTED]",
		},
		{
			name:     "SQL INSERT statement",
			input:    "Error executing: INSERT INTO users (id, email, password) VALUES ('123e4567-e89b-12d3-a456-426614174000', 'user@example.com', 'hashed_password')",
			expected: "Error executing: INSERT INTO users (id, email, password) VALUES [SQL_VALUES_REDACTED]",
		},
		{
			name:     "SQL UPDATE with SET clause",
			input:    "Error executing: UPDATE users SET email = 'new_user@example.com', updated_at = '2023-04-05' WHERE id = '123e4567-e89b-12d3-a456-426614174000'",
			expected: "Error executing: UPDATE users SET [SQL_VALUES_REDACTED]",
		},
		{
			name:     "SQL DELETE with WHERE clause",
			input:    "Error executing: DELETE FROM users WHERE id = '123e4567-e89b-12d3-a456-426614174000'",
			expected: "Error executing: DELETE FROM users [SQL_WHERE_REDACTED]",
		},
		{
			name:     "SQL query with UUID",
			input:    "Query failed: SELECT * FROM cards WHERE user_id = '123e4567-e89b-12d3-a456-426614174000'",
			expected: "Query failed: SELECT FROM... [SQL_VALUES_REDACTED]",
		},
		{
			name:     "SQL query with JOIN and multiple conditions",
			input:    "Error: SELECT c.* FROM cards c JOIN users u ON c.user_id = u.id WHERE u.email = 'user@example.com' AND c.id = '123e4567-e89b-12d3-a456-426614174000'",
			expected: "Error: SELECT FROM... [SQL_VALUES_REDACTED]",
		},
		{
			name:     "multiple sensitive data types",
			input:    "Error processing request from user@company.com: db connection postgres://admin:secret@db.internal:5432/prod failed, check /var/log/app/errors.log",
			expected: "Error processing request from [REDACTED_EMAIL]: db connection [REDACTED_CREDENTIAL][REDACTED_HOST]/prod failed, check [REDACTED_PATH]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := redact.String(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRedactError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.Equal(t, "", redact.Error(nil))
	})

	t.Run("simple error", func(t *testing.T) {
		err := errors.New("Connection failed with password=secret123")
		assert.Equal(t, "Connection failed with [REDACTED_CREDENTIAL]", redact.Error(err))
	})

	t.Run("wrapped error", func(t *testing.T) {
		innerErr := errors.New("db error: postgres://user:dbpass@localhost:5432/app")
		wrappedErr := fmt.Errorf("service layer: %w", innerErr)
		assert.Equal(
			t,
			"service layer: db error: [REDACTED_CREDENTIAL]localhost:5432/app",
			redact.Error(wrappedErr),
		)
	})

	t.Run("JWT token in error", func(t *testing.T) {
		err := errors.New(
			"Invalid token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		)
		// Because of pattern matching priority, the token: part matches the apiKeyRegex first
		// The word "token" is recognized by the API key regex, but the actual token should still get redacted
		assert.Equal(t, "Invalid [REDACTED_KEY]", redact.Error(err))

		// Verify that the JWT token is still properly redacted
		assert.NotContains(t, redact.Error(err), "eyJhbGci")
	})

	t.Run("UUID in error message", func(t *testing.T) {
		err := errors.New("Card with ID 123e4567-e89b-12d3-a456-426614174000 not found")
		assert.Equal(t, "Card with ID [REDACTED_UUID] not found", redact.Error(err))
	})

	t.Run("SQL query with UUID in error", func(t *testing.T) {
		err := errors.New("Failed to execute: SELECT * FROM cards WHERE id = '123e4567-e89b-12d3-a456-426614174000'")
		redacted := redact.Error(err)
		// Check that UUID is redacted correctly
		assert.NotContains(t, redacted, "123e4567-e89b-12d3-a456-426614174000")
		// Check that SQL structure is preserved with contingency approach
		assert.Contains(t, redacted, "SELECT FROM...")
		assert.Contains(t, redacted, "[SQL_VALUES_REDACTED]")
	})

	t.Run("SQL insert with multiple sensitive data", func(t *testing.T) {
		err := errors.New(
			"Failed to execute: INSERT INTO users (id, email, password) VALUES ('123e4567-e89b-12d3-a456-426614174000', 'user@example.com', 'secret123')",
		)
		redacted := redact.Error(err)
		// Check that sensitive values are redacted
		assert.NotContains(t, redacted, "123e4567-e89b-12d3-a456-426614174000")
		assert.NotContains(t, redacted, "user@example.com")
		assert.NotContains(t, redacted, "secret123")
		// Check that SQL structure is preserved with contingency approach
		assert.Contains(t, redacted, "INSERT INTO users")
		assert.Contains(t, redacted, "[SQL_VALUES_REDACTED]")
	})
}
