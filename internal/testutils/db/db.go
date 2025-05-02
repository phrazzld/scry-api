// Package db provides testing utilities for database-related testing.
// It focuses on transaction isolation patterns, database connection management,
// and utilities for setting up test data in the database.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
)

var (
	// migrationsRunOnce ensures migrations are only run once across all tests
	migrationsRunOnce sync.Once
)

// TestTimeout defines a default timeout for test database operations.
const TestTimeout = 5 * time.Second

// testGooseLogger is a simple implementation of the goose.Logger interface
// that doesn't output anything during tests to keep output clean.
type testGooseLogger struct {
	t *testing.T
}

// Printf implements the required logging method for goose's SetLogger
func (l *testGooseLogger) Printf(format string, v ...interface{}) {
	if l.t != nil {
		msg := fmt.Sprintf(format, v...)
		l.t.Log("Goose: " + strings.TrimSpace(msg))
	}
}

// Fatal implements the required logging method for goose's SetLogger
func (l *testGooseLogger) Fatal(v ...interface{}) {
	fmt.Println(v...)
	os.Exit(1)
}

// Fatalf implements the required logging method for goose's SetLogger
func (l *testGooseLogger) Fatalf(format string, v ...interface{}) {
	if l.t != nil {
		msg := fmt.Sprintf(format, v...)
		l.t.Fatal("Goose fatal error: " + strings.TrimSpace(msg))
	} else {
		fmt.Printf(format, v...)
		os.Exit(1)
	}
}

// Print is required for goose.Logger interface
func (l *testGooseLogger) Print(v ...interface{}) {
	// Silence regular prints during tests
}

// Println is required for goose.Logger interface
func (l *testGooseLogger) Println(v ...interface{}) {
	// Silence regular prints during tests
}

// IsIntegrationTestEnvironment returns true if the DATABASE_URL environment
// variable is set, indicating that integration tests can be run.
func IsIntegrationTestEnvironment() bool {
	return len(os.Getenv("DATABASE_URL")) > 0
}

// SetupTestDatabaseSchemaWithT runs database migrations to set up the test database.
// This is the version that takes a testing.T parameter which allows it to use the testing framework.
func SetupTestDatabaseSchemaWithT(t *testing.T, db *sql.DB) {
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

// SetupTestDatabaseSchema initializes the database schema using project migrations.
// It resets the schema to baseline (by running migrations down to version 0),
// then applies all migrations. This ensures tests run against the canonical schema.
//
// This function should typically be called once in TestMain, rather than for each test.
// It uses sync.Once to ensure migrations are only run once even if called multiple times.
func SetupTestDatabaseSchema(db *sql.DB) error {
	var setupErr error
	migrationsRunOnce.Do(func() {
		// Set the goose dialect
		if err := goose.SetDialect("postgres"); err != nil {
			setupErr = fmt.Errorf("failed to set goose dialect: %w", err)
			return
		}

		// Get the project root directory
		projectRoot, err := findProjectRoot()
		if err != nil {
			setupErr = fmt.Errorf("failed to find project root: %w", err)
			return
		}

		// Path to migrations directory
		migrationsDir := filepath.Join(
			projectRoot,
			"internal",
			"platform",
			"postgres",
			"migrations",
		)

		// Verify migrations directory exists
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			setupErr = fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
			return
		}

		// Set custom logger for goose to avoid unwanted output during tests
		goose.SetLogger(&testGooseLogger{})

		// Reset database schema to baseline
		if err := goose.DownTo(db, migrationsDir, 0); err != nil {
			setupErr = fmt.Errorf("failed to reset database schema: %w", err)
			return
		}

		// Apply all migrations
		if err := goose.Up(db, migrationsDir); err != nil {
			setupErr = fmt.Errorf("failed to apply migrations: %w", err)
			return
		}
	})

	return setupErr
}

// WithTx runs a test function with transaction-based isolation.
// It creates a new transaction, runs the test function with that transaction,
// and then rolls back the transaction to ensure test isolation.
//
// This enables parallel testing since each test runs in its own transaction
// and changes are automatically rolled back, preventing interference between tests.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    t.Parallel() // Safe with transaction isolation
//
//	    // Get a DB connection with automatic cleanup
//	    db := testutils.db.GetTestDBWithT(t)
//	    // No need for defer AssertCloseNoError - cleanup is registered by GetTestDBWithT
//
//	    testutils.db.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
//	        // Function receives testing.T and sql.Tx parameters
//	        ctx := context.Background()
//
//	        // Option 1: Create individual stores with the transaction
//	        userStore := postgres.NewPostgresUserStore(tx, bcrypt.MinCost)
//	        memoStore := postgres.NewPostgresMemoStore(tx, nil)
//
//	        // Option 2: Create all stores at once
//	        // stores := testutils.CreateTestStores(tx, bcrypt.MinCost)
//
//	        // Test your store methods - changes are automatically rolled back
//	        user, err := userStore.Create(ctx, testUser)
//	        require.NoError(t, err)
//	        // ... more test code
//	    })
//	}
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()

	// Begin a transaction
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Make sure the transaction is rolled back when the test is done
	defer AssertRollbackNoError(t, tx)

	// Run the test function with the transaction, passing both t and tx directly
	fn(t, tx)
}

// AssertRollbackNoError attempts to roll back a transaction and logs an error if it fails.
// This is a utility function for testing to ensure clean rollback of transactions.
func AssertRollbackNoError(t *testing.T, tx *sql.Tx) {
	t.Helper()

	if tx == nil {
		return
	}

	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		t.Logf("Warning: failed to roll back transaction: %v", err)
	}
}

// ResetTestData truncates all test tables to ensure test isolation.
//
// NOTE: This function is now deprecated and only provided for backward compatibility.
// Tests should use WithTx instead to achieve isolation via transactions.
func ResetTestData(db *sql.DB) error {
	// With transaction-based isolation, this is no longer needed for new tests.
	// It's kept for backward compatibility with existing tests.
	//
	// Use CASCADE to handle foreign key constraints
	_, err := db.Exec("TRUNCATE TABLE users CASCADE")
	if err != nil {
		return fmt.Errorf("failed to truncate users table: %w", err)
	}
	return nil
}

// findProjectRoot attempts to locate the project root directory.
// It works by searching for the go.mod file starting from the current file's directory
// and going up the directory tree.
func findProjectRoot() (string, error) {
	// Get the current file's directory
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}
	currentDir := filepath.Dir(currentFile)

	// Traverse up to find go.mod
	dir := currentDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod
			return dir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Reached root without finding go.mod
			return "", fmt.Errorf("could not find project root (go.mod file)")
		}
		dir = parentDir
	}
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

// GetTestDBWithT returns a database connection for testing.
// It automatically sets up the database schema using migrations, making it ready for tests.
// The function uses the following order of precedence for the database URL:
//
// 1. DATABASE_URL environment variable (used by CI/CD and integration tests)
// 2. SCRY_TEST_DB_URL environment variable (for local developer configuration)
// 3. Default local database URL (for developer convenience)
//
// This function handles proper connection validation and initialization, ensuring
// that tests can immediately use the returned database connection without additional setup.
// It also registers automatic cleanup with t.Cleanup() so you don't need to manually close.
//
// Usage:
//
//	// Simple pattern with minimal boilerplate
//	func TestSomething(t *testing.T) {
//	    t.Parallel()
//
//	    // Get a DB connection - no error handling needed
//	    db := testutils.db.GetTestDBWithT(t)
//	    // No need for defer or cleanup - db will be closed automatically
//
//	    testutils.db.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
//	        // Test code using transaction
//	    })
//	}
//
// For the original version that returns an error, use GetTestDB.
func GetTestDBWithT(t *testing.T) *sql.DB {
	t.Helper()

	// First check for DATABASE_URL from integration tests
	dbURL := os.Getenv("DATABASE_URL")

	// Fall back to SCRY_TEST_DB_URL if DATABASE_URL is not set
	if dbURL == "" {
		dbURL = os.Getenv("SCRY_TEST_DB_URL")
	}

	// If neither environment variable is set, use default
	if dbURL == "" {
		// Use default local database URL
		dbURL = "postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable"
	}

	// Open database connection
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	// Register cleanup to close the database connection
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("Warning: failed to close database connection: %v", closeErr)
		}
	})

	// Verify the connection works
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}

	// Setup database schema
	if err := SetupTestDatabaseSchema(db); err != nil {
		t.Fatalf("Failed to setup database schema: %v", err)
	}

	// Configure connection pool settings for tests
	db.SetMaxOpenConns(25) // Reasonable number of concurrent connections for tests
	db.SetMaxIdleConns(25) // Keep connections ready for test parallelism
	db.SetConnMaxLifetime(5 * time.Minute)

	return db
}

// GetTestDB is the original version that returns an error rather than using t.Helper
// This is maintained for backward compatibility with existing tests.
//
// NOTE: Prefer using GetTestDBWithT instead, which handles errors and cleanup automatically.
//
// Usage:
//
//	// Legacy pattern (not recommended for new tests)
//	db, err := testutils.db.GetTestDB()
//	require.NoError(t, err)
//	defer testutils.AssertCloseNoError(t, db)
//
//	// Modern pattern:
//	// db := testutils.db.GetTestDBWithT(t)
func GetTestDB() (*sql.DB, error) {
	// First check for DATABASE_URL from integration tests
	dbURL := os.Getenv("DATABASE_URL")

	// Fall back to SCRY_TEST_DB_URL if DATABASE_URL is not set
	if dbURL == "" {
		dbURL = os.Getenv("SCRY_TEST_DB_URL")
	}

	// If neither environment variable is set, use default
	if dbURL == "" {
		// Use default local database URL
		dbURL = "postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable"
	}

	// Open database connection
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Verify the connection works
	if err := db.Ping(); err != nil {
		// Close the connection to avoid leaking resources
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf(
				"database ping failed: %w (and failed to close connection: %v)",
				err,
				closeErr,
			)
		}
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	// Setup database schema
	if err := SetupTestDatabaseSchema(db); err != nil {
		// Close the connection to avoid leaking resources
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf(
				"failed to setup database schema: %w (additionally, failed to close db: %v)",
				err,
				closeErr,
			)
		}
		return nil, fmt.Errorf("failed to setup database schema: %w", err)
	}

	// Configure connection pool settings for tests
	db.SetMaxOpenConns(25) // Reasonable number of concurrent connections for tests
	db.SetMaxIdleConns(25) // Keep connections ready for test parallelism
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// CleanupDB properly closes a database connection and logs any errors.
// This function should be used with t.Cleanup() to ensure proper resource cleanup
// in tests that use database connections.
//
// NOTE: You don't need to call this directly when using GetTestDBWithT(t),
// as that function automatically registers cleanup with t.Cleanup().
//
// Usage:
//
//	// Older pattern (prefer GetTestDBWithT instead)
//	db, err := testutils.db.GetTestDB()
//	require.NoError(t, err)
//	t.Cleanup(func() { testutils.db.CleanupDB(t, db) })
//
//	// Better pattern
//	db := testutils.db.GetTestDBWithT(t)
//	// No manual cleanup needed - handled by GetTestDBWithT
func CleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		t.Logf("Warning: failed to close database connection: %v", err)
	}
}

// RunInTransaction executes the given function within a transaction.
// The transaction is automatically rolled back after the function completes.
// This is an alias for WithTx to maintain compatibility with existing code.
func RunInTransaction(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	WithTx(t, db, fn)
}

// AssertCloseNoError safely closes a resource that implements io.Closer,
// ensuring any errors are properly logged but not failing the test.
// This is useful in defer statements for cleanup.
func AssertCloseNoError(t *testing.T, closer io.Closer) {
	t.Helper()
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		t.Logf("Warning: failed to close resource: %v", err)
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
