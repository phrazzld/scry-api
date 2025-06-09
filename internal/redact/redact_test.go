package redact_test

import (
	"errors"
	"fmt"
	"strings"
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

// TestSQLRedaction provides comprehensive tests specifically focused on SQL redaction
func TestSQLRedaction(t *testing.T) {
	// These test cases focus specifically on SQL redaction scenarios, covering various
	// statement types, clauses, data types, and edge cases
	tests := []struct {
		name      string
		input     string
		notExpect []string // Strings that should NOT appear in the output
		contains  []string // Strings that SHOULD appear in the output
	}{
		// SELECT statements with various clauses
		{
			name:      "SELECT with simple WHERE clause",
			input:     "SELECT * FROM users WHERE id = 42",
			notExpect: []string{"id = 42"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SELECT with quoted string in WHERE",
			input:     "SELECT * FROM users WHERE username = 'admin'",
			notExpect: []string{"username = 'admin'", "admin"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SELECT with JOIN clause",
			input:     "SELECT u.name, c.title FROM users u JOIN cards c ON u.id = c.user_id WHERE u.email = 'user@example.com'",
			notExpect: []string{"user@example.com"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SELECT with multiple JOIN clauses",
			input:     "SELECT u.name, c.title, m.content FROM users u JOIN cards c ON u.id = c.user_id JOIN memos m ON c.id = m.card_id WHERE u.email = 'user@example.com'",
			notExpect: []string{"user@example.com"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SELECT with ORDER BY, GROUP BY, and LIMIT",
			input:     "SELECT category, COUNT(*) FROM products WHERE price > 100 GROUP BY category ORDER BY COUNT(*) DESC LIMIT 10",
			notExpect: []string{"price > 100"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SELECT with subquery",
			input:     "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders WHERE total > 1000)",
			notExpect: []string{"total > 1000"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SELECT with multiple conditions in WHERE",
			input:     "SELECT * FROM users WHERE email = 'admin@example.com' AND password = 'hashed_secret' OR api_key = 'abc123xyz789'",
			notExpect: []string{"admin@example.com", "hashed_secret", "abc123xyz789"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SELECT with UNION",
			input:     "SELECT id, email FROM users WHERE role = 'admin' UNION SELECT id, email FROM deleted_users WHERE delete_date > '2023-01-01'",
			notExpect: []string{"role = 'admin'", "delete_date > '2023-01-01'"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},

		// INSERT statements with different patterns
		{
			name:      "INSERT with single row of values",
			input:     "INSERT INTO users (id, username, email) VALUES (42, 'johndoe', 'john@example.com')",
			notExpect: []string{"42", "johndoe", "john@example.com"},
			contains:  []string{"INSERT INTO users", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "INSERT with multiple rows",
			input:     "INSERT INTO logs (user_id, action, timestamp) VALUES (1, 'login', '2023-04-01'), (2, 'logout', '2023-04-02'), (3, 'update', '2023-04-03')",
			notExpect: []string{"login", "logout", "update", "2023-04-01"},
			contains:  []string{"INSERT INTO logs", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "INSERT with SELECT subquery",
			input:     "INSERT INTO user_backup (id, name, email) SELECT id, name, email FROM users WHERE created_at < '2023-01-01'",
			notExpect: []string{"created_at < '2023-01-01'"},
			contains:  []string{"INSERT INTO user_backup", "SELECT FROM..."},
		},
		{
			name:      "INSERT with function calls",
			input:     "INSERT INTO sessions (id, user_id, token, expiry) VALUES (gen_random_uuid(), 42, encode(sha256('secret'), 'hex'), now() + interval '1 day')",
			notExpect: []string{"gen_random_uuid()", "42", "secret", "now()"},
			contains:  []string{"INSERT INTO sessions", "[SQL_VALUES_REDACTED]"},
		},

		// UPDATE statements with various clauses
		{
			name:      "UPDATE with simple SET and WHERE",
			input:     "UPDATE users SET last_login = '2023-04-05' WHERE id = 42",
			notExpect: []string{"last_login = '2023-04-05'", "id = 42"},
			contains:  []string{"UPDATE users SET", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "UPDATE with multiple SET clauses",
			input:     "UPDATE users SET email = 'new@example.com', password = 'hashed_new_password', updated_at = NOW() WHERE username = 'johndoe'",
			notExpect: []string{"new@example.com", "hashed_new_password", "johndoe"},
			contains:  []string{"UPDATE users SET", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "UPDATE with complex WHERE condition",
			input:     "UPDATE products SET price = price * 1.1 WHERE category = 'electronics' AND created_at < '2023-01-01'",
			notExpect: []string{"price * 1.1", "electronics", "2023-01-01"},
			contains:  []string{"UPDATE products SET", "[SQL_VALUES_REDACTED]"},
		},

		// DELETE statements with different patterns
		{
			name:      "DELETE with simple WHERE",
			input:     "DELETE FROM sessions WHERE expires_at < NOW()",
			notExpect: []string{"expires_at < NOW()"},
			contains:  []string{"DELETE FROM sessions", "[SQL_WHERE_REDACTED]"},
		},
		{
			name:      "DELETE with complex WHERE",
			input:     "DELETE FROM users WHERE id IN (SELECT user_id FROM inactive_accounts WHERE last_login < '2022-01-01')",
			notExpect: []string{"last_login < '2022-01-01'"},
			contains:  []string{"DELETE FROM users", "[SQL_WHERE_REDACTED]"},
		},
		{
			name:      "DELETE with JOIN-like syntax (PostgreSQL)",
			input:     "DELETE FROM orders USING users WHERE orders.user_id = users.id AND users.email = 'removed@example.com'",
			notExpect: []string{"removed@example.com"},
			contains:  []string{"DELETE FROM orders", "[SQL_WHERE_REDACTED]"},
		},
		{
			name:     "DELETE all rows",
			input:    "DELETE FROM temp_logs",
			contains: []string{"DELETE FROM temp_logs", "[SQL_WHERE_REDACTED]"},
		},

		// Different sensitive data types in SQL
		{
			name:      "SQL with UUID values",
			input:     "SELECT * FROM orders WHERE id = '550e8400-e29b-41d4-a716-446655440000'",
			notExpect: []string{"550e8400-e29b-41d4-a716-446655440000"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with email address",
			input:     "SELECT * FROM users WHERE email = 'admin@example.com'",
			notExpect: []string{"admin@example.com"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with password",
			input:     "SELECT * FROM users WHERE password = 'hashed_password_value'",
			notExpect: []string{"hashed_password_value"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with numeric IDs",
			input:     "SELECT * FROM products WHERE id IN (1, 2, 3, 4, 5)",
			notExpect: []string{"(1, 2, 3, 4, 5)"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with dates and timestamps",
			input:     "SELECT * FROM logs WHERE created_at BETWEEN '2023-01-01 00:00:00' AND '2023-01-31 23:59:59'",
			notExpect: []string{"2023-01-01", "2023-01-31"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with JSON data",
			input:     "SELECT * FROM configs WHERE settings @> '{\"debug\": true, \"api_key\": \"secret123\"}'",
			notExpect: []string{"debug", "api_key", "secret123"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with binary data",
			input:     "INSERT INTO files (name, content) VALUES ('document.pdf', E'\\x89504E470D0A1A0A')",
			notExpect: []string{"document.pdf", "\\x89504E470D0A1A0A"},
			contains:  []string{"INSERT INTO files", "[SQL_VALUES_REDACTED]"},
		},

		// SQL syntax variations
		{
			name:      "SQL with different quotes",
			input:     "SELECT * FROM users WHERE name = 'John' AND department = \"Sales\"",
			notExpect: []string{"John", "Sales"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:     "SQL with different JOIN types",
			input:    "SELECT u.name, o.total FROM users u LEFT JOIN orders o ON u.id = o.user_id RIGHT JOIN payments p ON o.id = p.order_id",
			contains: []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with parenthesized expressions",
			input:     "SELECT * FROM orders WHERE (status = 'pending' OR status = 'processing') AND created_at > '2023-01-01'",
			notExpect: []string{"pending", "processing", "2023-01-01"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with functions",
			input:     "SELECT * FROM users WHERE LOWER(email) = 'admin@example.com' AND password = SHA256('secret')",
			notExpect: []string{"admin@example.com", "secret"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with aggregates and GROUP BY",
			input:     "SELECT category, COUNT(*), AVG(price) FROM products WHERE price > 100 GROUP BY category HAVING COUNT(*) > 5",
			notExpect: []string{"price > 100", "COUNT(*) > 5"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with CASE expression",
			input:     "SELECT id, name, CASE WHEN status = 'active' THEN 'Current' WHEN status = 'inactive' THEN 'Former' ELSE 'Unknown' END AS user_status FROM users",
			notExpect: []string{"status = 'active'", "status = 'inactive'"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with comments",
			input:     "SELECT id, username -- Get user info\nFROM users -- Users table\nWHERE email = 'admin@example.com' -- Admin user",
			notExpect: []string{"admin@example.com"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},

		// Edge cases
		{
			name:      "Very long SQL query",
			input:     "SELECT u.id, u.name, u.email, u.phone, u.address, u.city, u.state, u.zip, u.country, o.id, o.date, o.total, o.status, p.id, p.method, p.amount, p.date FROM users u JOIN orders o ON u.id = o.user_id JOIN payments p ON o.id = p.order_id WHERE u.email = 'customer@example.com' AND o.date > '2023-01-01' AND o.total > 100 ORDER BY o.date DESC LIMIT 10",
			notExpect: []string{"customer@example.com", "2023-01-01", "total > 100"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with mixed capitalization",
			input:     "sElEcT * FrOm UsErS wHeRe EmAiL = 'admin@example.com'",
			notExpect: []string{"admin@example.com"},
			contains:  []string{"FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "SQL with unusual whitespace",
			input:     "SELECT    *    FROM    users    WHERE    email    =    'admin@example.com'",
			notExpect: []string{"admin@example.com"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		// Partial SQL fragments and syntax errors should not be redacted
		{
			name:     "Partial SQL fragment",
			input:    "WHERE user_id = 42 AND role = 'admin'",
			contains: []string{"WHERE user_id = 42 AND role = 'admin'"},
		},
		{
			name:     "SQL with syntax error",
			input:    "SELCT * FORM users WEHRE id = 42", // Intentional typos
			contains: []string{"SELCT * FORM users WEHRE id = 42"},
		},

		// PostgreSQL-specific SQL features
		{
			name:      "PostgreSQL dollar-quoted strings",
			input:     "SELECT * FROM users WHERE data = $$sensitive information with 'quotes' inside$$",
			notExpect: []string{"sensitive information"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "PostgreSQL array operations",
			input:     "SELECT * FROM users WHERE roles && ARRAY['admin', 'moderator']",
			notExpect: []string{"admin", "moderator"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "PostgreSQL JSONB operations",
			input:     "SELECT * FROM profiles WHERE data @> '{\"email\":\"admin@example.com\"}'",
			notExpect: []string{"admin@example.com"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "PostgreSQL RETURNING clause",
			input:     "INSERT INTO users (name, email) VALUES ('John Doe', 'john@example.com') RETURNING id, created_at",
			notExpect: []string{"John Doe", "john@example.com"},
			contains:  []string{"INSERT INTO users", "[SQL_VALUES_REDACTED]"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Redact the input
			result := redact.String(tc.input)

			// Verify that sensitive information is not present
			for _, sensitive := range tc.notExpect {
				if sensitive != "" {
					assert.NotContains(t, result, sensitive, "Sensitive data was not properly redacted")
				}
			}

			// Verify that expected patterns appear in the result
			for _, expected := range tc.contains {
				if expected != "" {
					assert.Contains(t, result, expected, "Expected content missing from redacted output")
				}
			}
		})
	}
}

// TestCombinedRedactionPatterns tests how different redaction patterns interact with each other
func TestCombinedRedactionPatterns(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		notExpect []string
		contains  []string
	}{
		{
			name:      "SQL query with multiple sensitive data types",
			input:     "SELECT * FROM users WHERE email = 'admin@example.com' AND api_key = 'secret123' AND created_at > '2023-01-01'",
			notExpect: []string{"admin@example.com", "secret123", "2023-01-01"},
			contains:  []string{"SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "Error log with SQL query and file path",
			input:     "Error in /var/app/server.go: Failed to execute SQL: SELECT * FROM users WHERE password = 'hashed_password'",
			notExpect: []string{"/var/app/server.go", "hashed_password"},
			contains:  []string{"Error in [REDACTED_PATH]", "SELECT FROM...", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:      "Error with stack trace and SQL query",
			input:     "panic: SQL error\ngoroutine 1 [running]:\nmain.executeQuery(\"SELECT * FROM users WHERE id = 42\")\n\t/app/main.go:42",
			notExpect: []string{"/app/main.go", "id = 42"},
			contains:  []string{"[STACK_TRACE_REDACTED]"},
		},
		{
			name:  "Query with UUID, email, and JWT",
			input: "INSERT INTO sessions (user_id, token) VALUES ('550e8400-e29b-41d4-a716-446655440000', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c') WHERE email = 'user@example.com'",
			notExpect: []string{
				"550e8400-e29b-41d4-a716-446655440000",
				"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
				"user@example.com",
			},
			contains: []string{"INSERT INTO sessions", "[SQL_VALUES_REDACTED]"},
		},
		{
			name:  "Query with all sensitive data types",
			input: "INSERT INTO logs (msg) VALUES ('User with UUID 550e8400-e29b-41d4-a716-446655440000 and email user@example.com used API key abcdef123456 from host.example.com to access /var/data/secure.db with password secret123')",
			notExpect: []string{
				"550e8400-e29b-41d4-a716-446655440000",
				"user@example.com",
				"abcdef123456",
				"host.example.com",
				"/var/data/secure.db",
				"secret123",
			},
			contains: []string{"INSERT INTO logs", "[SQL_VALUES_REDACTED]"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := redact.String(tc.input)

			// Check that no sensitive data remains
			for _, sensitive := range tc.notExpect {
				if sensitive != "" {
					assert.NotContains(t, result, sensitive, "Sensitive data was not properly redacted")
				}
			}

			// Check that expected patterns appear in the result
			for _, expected := range tc.contains {
				if expected != "" {
					assert.Contains(t, result, expected, "Expected content missing from redacted output")
				}
			}
		})
	}
}

// TestRedactionPerformance verifies the redaction logic works efficiently with large inputs
func TestRedactionPerformance(t *testing.T) {
	// Create a large SQL statement with sensitive data
	var builder strings.Builder
	builder.WriteString("SELECT * FROM users WHERE ")

	// Add 100 conditions with sensitive data
	for i := 0; i < 100; i++ {
		if i > 0 {
			builder.WriteString(" OR ")
		}
		builder.WriteString(fmt.Sprintf("(email = 'user%d@example.com' AND api_key = 'secret%d')", i, i))
	}

	largeSQLQuery := builder.String()

	// Verify redaction works correctly with large input
	result := redact.String(largeSQLQuery)

	// Check for expected SQL redaction pattern
	assert.Contains(t, result, "SELECT FROM...")
	assert.Contains(t, result, "[SQL_VALUES_REDACTED]")

	// Ensure no sensitive data remains in the result
	assert.NotContains(t, result, "user")
	assert.NotContains(t, result, "example.com")
	assert.NotContains(t, result, "secret")
}
