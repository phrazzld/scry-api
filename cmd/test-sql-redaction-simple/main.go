package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/redact"
)

func main() {
	// Set up logger with debug level
	loggerConfig := logger.LoggerConfig{
		Level: "debug",
	}
	l, err := logger.Setup(loggerConfig)
	if err != nil {
		fmt.Printf("Failed to set up logger: %v\n", err)
		os.Exit(1)
	}
	slog.SetDefault(l)

	l.Info("Starting SQL redaction test...")

	// Test SQL statements with sensitive data
	testSQLRedaction(l)

	l.Info("SQL redaction test completed.")
}

func testSQLRedaction(l *slog.Logger) {
	// Sample SQL queries with sensitive data
	queries := []string{
		// SELECT with sensitive data in WHERE clause
		"SELECT * FROM users WHERE id = '123e4567-e89b-12d3-a456-426614174000' AND email = 'admin@example.com' AND password = 'secret123'",

		// INSERT with sensitive data in VALUES clause
		"INSERT INTO users (id, username, email, password) VALUES ('550e8400-e29b-41d4-a716-446655440000', 'johndoe', 'john@example.com', 'hashed_password_value')",

		// UPDATE with sensitive data in SET clause
		"UPDATE users SET email = 'new@example.com', password = 'new_password', last_login = NOW() WHERE id = '123e4567-e89b-12d3-a456-426614174000'",

		// DELETE with sensitive data in WHERE clause
		"DELETE FROM sessions WHERE user_id = '123e4567-e89b-12d3-a456-426614174000' AND token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaWF0IjoxNTE2MjM5MDIyfQ'",

		// Complex query with multiple sensitive data types
		`SELECT u.id, u.username FROM users u
		JOIN cards c ON u.id = c.user_id
		WHERE u.email = 'admin@example.com'
		AND c.created_at > '2023-01-01'
		AND c.id IN (SELECT card_id FROM user_card_stats WHERE level > 3 AND user_id = '123e4567-e89b-12d3-a456-426614174000')`,
	}

	// Log each query both directly and in an error
	for i, query := range queries {
		// Direct logging of original query - this should be redacted by the logger
		l.Info(fmt.Sprintf("SQL Test %d - Original query", i+1), "query", query)

		// Log pre-redacted query - shows what redaction is doing explicitly
		redactedQuery := redact.String(query)
		l.Info(fmt.Sprintf("SQL Test %d - Pre-redacted query", i+1), "redacted_query", redactedQuery)

		// Test query in error message - should be redacted when logged
		err := fmt.Errorf("database error: %s", query)
		l.Error(fmt.Sprintf("SQL Test %d - Error with query", i+1), "error", err)

		// Test wrapped error with query - should be redacted in nested errors
		wrappedErr := fmt.Errorf(
			"operation failed: %w",
			fmt.Errorf("database error with query: %s", query),
		)
		l.Error(fmt.Sprintf("SQL Test %d - Wrapped error with query", i+1), "error", wrappedErr)
	}

	// Additional tests for different redaction patterns
	l.Info("Testing query with UUID", "uuid", "123e4567-e89b-12d3-a456-426614174000")
	l.Info("Testing query with email", "email", "user@example.com")
	l.Info(
		"Testing query with JWT token",
		"token",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaWF0IjoxNTE2MjM5MDIyfQ",
	)

	// Check combined redaction patterns
	combinedSensitiveData := "User 123e4567-e89b-12d3-a456-426614174000 (admin@example.com) with API key 'secret123' logged in from 192.168.1.1"
	l.Info("Testing combined sensitive data", "data", combinedSensitiveData)

	// Check different SQL types are properly redacted
	// Log a realistic PostgreSQL error message that might appear in logs
	pgError := "ERROR: duplicate key value violates unique constraint \"users_email_key\" (SQLSTATE 23505), Key (email)=(user@example.com) already exists"
	l.Error("Database operation failed", "pg_error", pgError)
}
