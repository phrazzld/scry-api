//go:build integration

package testdb

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
)

// TestBasicDatabaseConnection verifies that a connection to the database can be established
// and a simple query can be executed. This serves as an early "canary" test to catch
// database configuration issues quickly, especially in CI environments.
func TestBasicDatabaseConnection(t *testing.T) {
	// Only include the integration build tag, not test_without_external_deps
	// This ensures the test only runs when actually connecting to a database

	// Get the database URL using our standardized function
	// This will respect environment variables and handle CI-specific standardization
	dbURL := GetTestDatabaseURL()
	if dbURL == "" {
		t.Skip("No database URL available - skipping basic database connection test")
	}

	// Create a logger with appropriate context
	logger := slog.Default().With(
		slog.String("test", "TestBasicDatabaseConnection"),
		slog.Bool("ci_environment", isCIEnvironment()),
	)

	// Log the test execution for visibility in CI
	if isCIEnvironment() {
		logger.Info("executing basic database connection test",
			slog.String("database_url", maskDatabaseURL(dbURL)),
		)
	}

	// Open a database connection
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer func() {
		// Ensure connection is properly closed after test
		if err := db.Close(); err != nil {
			t.Logf("Warning: failed to close database connection: %v", err)
		}
	}()

	// Configure connection pool
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(2 * time.Minute)

	// Create a context with timeout to prevent hanging tests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to ping the database to verify connection
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Database ping failed: %v\nDatabase URL: %s",
			err, maskDatabaseURL(dbURL))
	}

	// Log successful ping in CI
	if isCIEnvironment() {
		logger.Info("database ping successful")
	}

	// Execute a simple query to verify connectivity
	var one int
	row := db.QueryRowContext(ctx, "SELECT 1 AS ping")
	if err := row.Scan(&one); err != nil {
		t.Fatalf("Failed to execute simple query: %v", err)
	}

	// Verify the query result
	if one != 1 {
		t.Errorf("Unexpected result from test query: expected 1, got %d", one)
	}

	// For CI environments, provide additional diagnostics
	if isCIEnvironment() {
		// Get database version
		var version string
		if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
			logger.Warn("failed to get database version",
				slog.String("error", err.Error()))
		} else {
			logger.Info("connected to PostgreSQL server",
				slog.String("version", version))
		}

		// Get database user
		var user string
		if err := db.QueryRowContext(ctx, "SELECT current_user").Scan(&user); err != nil {
			logger.Warn("failed to get current database user",
				slog.String("error", err.Error()))
		} else {
			logger.Info("connected as database user",
				slog.String("user", user))
		}

		// Get connection stats
		stats := db.Stats()
		logger.Info("database connection pool statistics",
			slog.Int("open_connections", stats.OpenConnections),
			slog.Int("in_use", stats.InUse),
			slog.Int("idle", stats.Idle),
			slog.Int64("wait_count", stats.WaitCount),
			slog.Float64("wait_duration_seconds", stats.WaitDuration.Seconds()),
		)
	}

	// Log test completion
	logger.Info("basic database connection test completed successfully")

	// Additional verification: Try to execute a transaction to ensure permissions
	txCtx, txCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer txCancel()

	tx, err := db.BeginTx(txCtx, nil)
	if err != nil {
		t.Errorf("Failed to begin transaction, may indicate permission issues: %v", err)
		return
	}

	// Execute a simple query within the transaction
	if _, err := tx.ExecContext(txCtx, "SELECT 1"); err != nil {
		t.Errorf("Failed to execute query within transaction: %v", err)
	}

	// Rollback the transaction (we don't actually want to commit anything)
	if err := tx.Rollback(); err != nil {
		t.Logf("Warning: failed to rollback transaction: %v", err)
	}

	// Success - we were able to connect, query, and use transactions
	t.Logf("Successfully verified database connection, query execution, and transaction support")
}

// TestDatabaseURLConsistency verifies that the database URL is consistently
// returned from GetTestDatabaseURL() and properly standardized in CI environments.
func TestDatabaseURLConsistency(t *testing.T) {
	// Only run if we have a database URL configured
	if !IsIntegrationTestEnvironment() {
		t.Skip("No database URL available - skipping URL consistency test")
	}

	// Get the URL twice to ensure consistency
	url1 := GetTestDatabaseURL()
	url2 := GetTestDatabaseURL()

	// URLs should be identical across multiple calls
	if url1 != url2 {
		t.Errorf("GetTestDatabaseURL() returned inconsistent results:\nFirst call:  %s\nSecond call: %s",
			maskDatabaseURL(url1), maskDatabaseURL(url2))
	}

	// In CI, verify the URL has the expected username
	if isCIEnvironment() {
		// Standard database URL format: postgres://username:password@host:port/dbname
		// We'll use a simple string check rather than full URL parsing for this test
		// since parsing is already tested in the unit tests
		if url1 == "" {
			t.Errorf("Empty database URL returned in CI environment")
		} else if !containsSubstring(url1, "postgres://postgres:") {
			t.Errorf("Database URL in CI should use 'postgres' user, got: %s", maskDatabaseURL(url1))
		}

		// For GitHub Actions specifically, verify both username and password
		if isGitHubActionsCI() && !containsSubstring(url1, "postgres://postgres:postgres@") {
			t.Errorf("Database URL in GitHub Actions should use 'postgres:postgres' credentials, got: %s",
				maskDatabaseURL(url1))
		}
	}
}

// containsSubstring is a helper function to check if a string contains a substring.
// This helps make the tests more readable.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
