package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx driver for database/sql
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/redact"
)

func main() {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up logger with debug level to see SQL queries
	loggerConfig := logger.LoggerConfig{
		Level: "debug",
	}
	l, err := logger.Setup(loggerConfig)
	if err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}
	slog.SetDefault(l)

	// Connect to database
	l.Info("Connecting to database...")
	db, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		l.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			l.Error("Error closing database connection", "error", err)
		}
	}()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		l.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	l.Info("Successfully connected to database")

	// Generate and log SQL queries with sensitive data
	generateAndLogQueries(l, db)
}

func generateAndLogQueries(l *slog.Logger, db *sql.DB) {
	// Log some queries with sensitive data

	// SELECT with sensitive data in WHERE clause
	selectQuery := "SELECT * FROM users WHERE id = '123e4567-e89b-12d3-a456-426614174000' AND email = 'admin@example.com' AND password = 'secret123'"
	l.Info("Executing query with sensitive data", "query", selectQuery)

	// INSERT with sensitive data in VALUES clause
	insertQuery := "INSERT INTO users (id, username, email, password) VALUES ('550e8400-e29b-41d4-a716-446655440000', 'johndoe', 'john@example.com', 'hashed_password_value')"
	l.Info("Executing query with sensitive data", "query", insertQuery)

	// UPDATE with sensitive data in SET clause
	updateQuery := "UPDATE users SET email = 'new@example.com', password = 'new_password', last_login = NOW() WHERE id = '123e4567-e89b-12d3-a456-426614174000'"
	l.Info("Executing query with sensitive data", "query", updateQuery)

	// DELETE with sensitive data in WHERE clause
	deleteQuery := "DELETE FROM sessions WHERE user_id = '123e4567-e89b-12d3-a456-426614174000' AND token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaWF0IjoxNTE2MjM5MDIyfQ'"
	l.Info("Executing query with sensitive data", "query", deleteQuery)

	// Test with complex SQL that includes JOINs, subqueries, and multiple sensitive values
	complexQuery := `
		SELECT u.id, u.username, c.title
		FROM users u
		JOIN cards c ON u.id = c.user_id
		WHERE u.email = 'admin@example.com'
		AND c.created_at > '2023-01-01'
		AND c.id IN (
			SELECT card_id FROM user_card_stats
			WHERE level > 3 AND user_id = '123e4567-e89b-12d3-a456-426614174000'
		)
	`
	l.Info("Executing complex query with sensitive data", "query", complexQuery)

	// Test with actual DB query (doesn't need to succeed, just log the query)
	_, err := db.Exec("SELECT 1 WHERE 'sensitive_value' = 'should_be_redacted'")
	if err != nil {
		l.Error("Query execution failed", "error", err)
	}

	// Test redaction through errors (not directly in logs)
	l.Info("Testing SQL query in error message")
	err = fmt.Errorf("failed to execute query: %s", selectQuery)
	l.Error("Error with SQL", "error", err)

	// Test pre-redacted query to check redaction is working
	l.Info("Testing pre-redacted SQL query")
	redactedQuery := redact.String(selectQuery)
	l.Info("Pre-redacted query", "query", redactedQuery)

	// Log redaction wrapper
	l.Info("Testing wrapped SQL query in error")
	wrappedErr := fmt.Errorf("operation failed: %w", fmt.Errorf("database error with query: %s", selectQuery))
	l.Error("Wrapped error with SQL", "error", wrappedErr)
}

// loadConfig loads the application configuration from environment variables or config file.
// Returns the loaded config and any loading error.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, nil
}
