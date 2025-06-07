//go:build exported_core_functions

package testdb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
)

// TestTimeout defines a default timeout for test database operations.
const TestTimeout = 5 * time.Second

// GetTestDatabaseURL returns the database URL to use for tests, with standardized credentials.
// This version is for use in production code and exports the functionality with exported_core_functions tag.
func GetTestDatabaseURL() string {
	// Start with the most basic logger
	logger := slog.Default().With("component", "database")

	// Check if we're in a CI environment (GitHub Actions, GitLab CI, etc.)
	inCI := isCIEnvironmentInternal()

	// Log environment detection for debugging
	logger.Info("Database environment detection",
		"ci_environment", inCI)

	// Try to get database URL from environment variables in order of precedence
	candidateURLs := []string{
		os.Getenv("DATABASE_URL"),
		os.Getenv("SCRY_TEST_DB_URL"),
		os.Getenv("SCRY_DATABASE_URL"),
	}

	var dbURL string
	for _, candidate := range candidateURLs {
		if candidate != "" {
			dbURL = candidate
			break
		}
	}

	// If no URL found, return empty string - callers will need to handle this
	if dbURL == "" {
		logger.Warn("No database URL found in environment variables")
		return ""
	}

	// In CI environments, standardize database credentials to 'postgres'
	if inCI {
		logger.Info("Standardizing database URL for CI environment")

		// Parse the URL
		parsedURL, err := url.Parse(dbURL)
		if err != nil {
			logger.Error("Failed to parse database URL", "error", err)
			return dbURL // Return original URL on error
		}

		// Set standardized CI credentials
		parsedURL.User = url.UserPassword("postgres", "postgres")

		// Update the URL
		dbURL = parsedURL.String()

		// Log the standardized URL (with password masked)
		safeURL := strings.Replace(dbURL, "postgres:postgres", "postgres:****", 1)
		logger.Info("Using standardized database URL in CI", "url", safeURL)
	}

	return dbURL
}

// isCIEnvironmentInternal returns true if the code is running in a CI environment
// This is a separate function to avoid name conflicts with the one in project_root.go
func isCIEnvironmentInternal() bool {
	return os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("GITLAB_CI") != ""
}

// maskDatabaseURL masks the password in a database URL for safe logging
func maskDatabaseURL(dbURL string) string {
	// Parse the URL
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return "invalid-url"
	}

	// Mask the password if user info exists
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		parsedURL.User = url.UserPassword(username, "****")
		return parsedURL.String()
	}

	return dbURL
}

// IsIntegrationTestEnvironment returns true if any of the database URL environment
// variables are set, indicating that integration tests can be run.
func IsIntegrationTestEnvironment() bool {
	// Check if any of the database URL environment variables are set
	envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}

	for _, envVar := range envVars {
		if len(os.Getenv(envVar)) > 0 {
			return true
		}
	}

	return false
}

// ShouldSkipDatabaseTest returns true if the database connection environment variables
// are not set, indicating that database integration tests should be skipped.
// This provides a consistent way for tests to check for database availability.
func ShouldSkipDatabaseTest() bool {
	return !IsIntegrationTestEnvironment()
}

// GetTestDBWithT returns a database connection for testing, with t.Helper() support.
// It automatically skips the test if DATABASE_URL is not set, ensuring
// consistent behavior for integration tests.
func GetTestDBWithT(t *testing.T) *sql.DB {
	t.Helper()

	// Skip the test if the database URL is not available
	dbURL := GetTestDatabaseURL()
	if dbURL == "" {
		t.Skip("DATABASE_URL or SCRY_TEST_DB_URL not set - skipping integration test")
	}

	// Open database connection
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify the connection works
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}

	// Register cleanup to close the database connection
	t.Cleanup(func() {
		CleanupDB(t, db)
	})

	return db
}

// CleanupDB properly closes a database connection, logging any errors.
func CleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if db == nil {
		return
	}

	if err := db.Close(); err != nil {
		t.Logf("Warning: failed to close database connection: %v", err)
	}
}

// WithTx runs the provided function within a database transaction.
// The transaction is automatically rolled back after the function completes,
// ensuring test isolation. This allows tests to make database modifications
// without persisting them, enabling parallel test execution.
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()

	// Verify database connection is active
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Database connection failed before transaction: %v", err)
	}

	// Start a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Ensure rollback happens after test completes or fails
	defer func() {
		if r := recover(); r != nil {
			// If there was a panic, try to roll back the transaction before re-panicking
			rollbackErr := tx.Rollback()
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				t.Logf("Warning: failed to rollback transaction after panic: %v", rollbackErr)
			}
			// Re-panic with the original error
			// ALLOW-PANIC
			panic(r)
		}

		// Normal rollback path
		err := tx.Rollback()
		// sql.ErrTxDone is expected if tx is already committed or rolled back
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Logf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	// Execute the test function with the transaction
	fn(t, tx)
}
