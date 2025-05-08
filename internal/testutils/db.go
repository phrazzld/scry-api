//go:build (!compatibility && ignore_redeclarations) || test_without_external_deps || !exclude_compat

// Package testutils provides testing utilities with a focus on database testing
// with transaction isolation. This package enables writing isolated, parallel
// integration tests that don't interfere with each other, even when they
// manipulate the same database tables and entities.
//
// Transaction Isolation Pattern:
//
// The primary pattern implemented in this package is transaction-based isolation.
// Each test runs in its own transaction, which is automatically rolled back
// when the test completes. This provides several benefits:
//
// 1. Tests can run in parallel without interfering with each other (t.Parallel())
// 2. No manual cleanup is needed - changes are rolled back automatically
// 3. Tests see a consistent database state (the transaction's snapshot)
// 4. Tests can operate on the same tables/data without conflicts
// 5. Tests run faster since there's no need to truncate tables between tests
//
// Usage:
//
//	func TestMyFeature(t *testing.T) {
//	    // Enable parallel testing safely
//	    t.Parallel()
//
//	    // Get a DB connection with automatic cleanup
//	    db := testutils.GetTestDBWithT(t)
//	    // No need to manually close - t.Cleanup is registered in GetTestDBWithT
//
//	    // Run your test in a transaction
//	    testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
//	        // Create test store instances with the transaction
//	        stores := testutils.CreateTestStores(tx, bcrypt.MinCost)
//
//	        // Use the stores to test your functionality
//	        ctx := context.Background()
//	        result, err := stores.UserStore.Create(ctx, testUser)
//	        require.NoError(t, err)
//
//	        // No cleanup needed - transaction will be rolled back automatically
//	    })
//	}
//
// See transaction_example_test.go for complete examples.
package testutils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
)

var (
	// migrationsRunOnce ensures migrations are only run once across all tests
	migrationsRunOnce sync.Once
)

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
//	    db := testutils.GetTestDBWithT(t)
//	    // No need for defer AssertCloseNoError - cleanup is registered by GetTestDBWithT
//
//	    testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
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
	defer AssertRollbackNoError(t, tx) // Uses the implementation from helpers.go

	// Run the test function with the transaction, passing both t and tx directly
	fn(t, tx)
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

// testGooseLogger is a simple implementation of the goose.Logger interface
// that doesn't output anything during tests to keep output clean.
type testGooseLogger struct{}

func (*testGooseLogger) Fatal(v ...interface{}) {
	fmt.Println(v...)
	os.Exit(1)
}

func (*testGooseLogger) Fatalf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	os.Exit(1)
}

func (*testGooseLogger) Print(v ...interface{}) {
	// Silence regular prints during tests
}

func (*testGooseLogger) Println(v ...interface{}) {
	// Silence regular prints during tests
}

func (*testGooseLogger) Printf(format string, v ...interface{}) {
	// Silence regular prints during tests
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
//	    db := testutils.GetTestDBWithT(t)
//	    // No need for defer or cleanup - db will be closed automatically
//
//	    testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
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
//	db, err := testutils.GetTestDB()
//	require.NoError(t, err)
//	defer testutils.AssertCloseNoError(t, db)
//
//	// Modern pattern:
//	// db := testutils.GetTestDBWithT(t)
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

// AssertRollbackNoError ensures that the Rollback() method on the provided tx
// executes without error, unless the error is sql.ErrTxDone which indicates
// the transaction was already committed or rolled back.
//
// This is specifically designed for use with SQL transactions, as it includes
// special handling for the common case where a transaction might already be
// committed or rolled back.
//
// Usage:
//
//	tx, err := db.BeginTx(ctx, nil)
//	require.NoError(t, err)
//	defer testutils.AssertRollbackNoError(t, tx)
func AssertRollbackNoError(t *testing.T, tx *sql.Tx) {
	t.Helper()
	if tx == nil {
		return
	}
	err := tx.Rollback()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		t.Logf("Failed to rollback transaction: %v", err)
	}
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
//	db, err := testutils.GetTestDB()
//	require.NoError(t, err)
//	t.Cleanup(func() { testutils.CleanupDB(t, db) })
//
//	// Better pattern
//	db := testutils.GetTestDBWithT(t)
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

// AssertCloseNoError is implemented in helpers.go
