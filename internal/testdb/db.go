// Package testdb provides utilities specifically for database testing.
// It maintains a clean dependency structure by only depending on store interfaces
// and standard database packages, not on specific implementations.
package testdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// TestTimeout defines a default timeout for test database operations.
const TestTimeout = 5 * time.Second

// IsIntegrationTestEnvironment returns true if the DATABASE_URL environment
// variable is set, indicating that integration tests can be run.
func IsIntegrationTestEnvironment() bool {
	return len(os.Getenv("DATABASE_URL")) > 0
}

// GetTestDatabaseURL returns the database URL for tests.
// It checks DATABASE_URL and SCRY_TEST_DB_URL environment variables
// in that order, returning the first non-empty value.
func GetTestDatabaseURL() string {
	// First check for DATABASE_URL from integration tests
	dbURL := os.Getenv("DATABASE_URL")

	// Fall back to SCRY_TEST_DB_URL if DATABASE_URL is not set
	if dbURL == "" {
		dbURL = os.Getenv("SCRY_TEST_DB_URL")
	}

	return dbURL
}

// SetupTestDatabaseSchema runs database migrations to set up the test database.
func SetupTestDatabaseSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	// Find project root to locate migration files
	projectRoot, err := findProjectRoot()
	require.NoError(t, err, "Failed to find project root")

	// Set up goose for migrations
	migrationsDir := filepath.Join(projectRoot, "internal", "platform", "postgres", "migrations")
	require.DirExists(t, migrationsDir, "Migrations directory does not exist: %s", migrationsDir)

	// Configure goose
	goose.SetLogger(&testGooseLogger{t: t})
	goose.SetTableName("schema_migrations")
	goose.SetBaseFS(os.DirFS(migrationsDir))

	// Run migrations
	err = goose.Up(db, ".")
	require.NoError(t, err, "Failed to run migrations")
}

// WithTx executes a test function within a transaction, automatically rolling back
// after the test completes. This ensures test isolation and prevents side effects.
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()

	// Start a transaction
	tx, err := db.Begin()
	require.NoError(t, err, "Failed to begin transaction")

	// Ensure rollback happens after test completes or fails
	defer func() {
		err := tx.Rollback()
		// sql.ErrTxDone is expected if tx is already committed or rolled back
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Logf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	// Execute the test function with the transaction
	fn(t, tx)
}

// ApplyMigrations runs migrations without using testing.T
// This exists for backward compatibility with code that was written
// before the testdb package was created
func ApplyMigrations(db *sql.DB, migrationsDir string) error {
	// Configure goose
	goose.SetTableName("schema_migrations")
	goose.SetBaseFS(os.DirFS(migrationsDir))

	// Run migrations
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// findProjectRoot locates the project root directory by traversing upwards
// until it finds a directory with go.mod file.
func findProjectRoot() (string, error) {
	// Start with current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Traverse up until we find go.mod
	for {
		// Check if go.mod exists in the current directory
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(dir)
		// If we're at the root and haven't found go.mod, we've gone too far
		if parentDir == dir {
			return "", fmt.Errorf("could not find go.mod in any parent directory")
		}
		dir = parentDir
	}
}

// testGooseLogger implements a minimal logger interface for goose
type testGooseLogger struct {
	t *testing.T
}

// Printf implements the required logging method for goose's SetLogger
func (l *testGooseLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.t.Log("Goose: " + strings.TrimSpace(msg))
}

// Fatalf implements the required logging method for goose's SetLogger
func (l *testGooseLogger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.t.Fatal("Goose fatal error: " + strings.TrimSpace(msg))
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
	require.NoError(t, err, "Failed to open database connection")

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify the connection works
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	err = db.PingContext(ctx)
	require.NoError(t, err, "Database ping failed")

	// Register cleanup to close the database connection
	t.Cleanup(func() {
		CleanupDB(t, db)
	})

	return db
}

// GetTestDB returns a database connection for testing without t.Helper() support.
// This is useful for non-test code that needs database access.
// Returns nil if DATABASE_URL is not set.
func GetTestDB() (*sql.DB, error) {
	// Check if the database URL is available
	dbURL := GetTestDatabaseURL()
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL or SCRY_TEST_DB_URL not set")
	}

	// Open database connection
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
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
		closeErr := db.Close()
		if closeErr != nil {
			// Combine both errors in the message
			return nil, fmt.Errorf("database ping failed: %w (close error: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return db, nil
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

// RunInTx executes the given function within a transaction.
// The transaction is automatically rolled back after the function completes.
func RunInTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()

	// Start a transaction
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err, "Failed to begin transaction")

	// Ensure rollback happens after test completes or fails
	defer func() {
		err := tx.Rollback()
		// sql.ErrTxDone is expected if tx is already committed or rolled back
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Logf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	// Execute the test function with the transaction
	fn(t, tx)
}
