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
//	    // Get a DB connection
//	    db, err := testutils.GetTestDB()
//	    require.NoError(t, err)
//	    defer testutils.AssertCloseNoError(t, db)
//
//	    // Run your test in a transaction
//	    testutils.WithTx(t, db, func(tx store.DBTX) {
//	        // Create test store instances with the transaction
//	        stores := testutils.CreateTestStores(tx)
//
//	        // Use the stores to test your functionality
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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/store"
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
		migrationsDir := filepath.Join(projectRoot, "internal", "platform", "postgres", "migrations")

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
//	    db, err := testutils.GetTestDB()
//	    require.NoError(t, err)
//	    defer testutils.AssertCloseNoError(t, db)
//
//	    testutils.WithTx(t, db, func(tx store.DBTX) {
//	        // Create store instances with the transaction
//	        userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)
//	        memoStore := postgres.NewPostgresMemoStore(tx, nil)
//
//	        // Test your store methods - changes are automatically rolled back
//	        user, err := userStore.Create(ctx, testUser)
//	        require.NoError(t, err)
//	        // ... more test code
//	    })
//	}
func WithTx(t *testing.T, db *sql.DB, fn func(tx store.DBTX)) {
	t.Helper()

	// Begin a transaction
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Make sure the transaction is rolled back when the test is done
	defer AssertRollbackNoError(t, tx)

	// Run the test function with the transaction
	fn(tx)
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

// GetTestDB returns a database connection for testing.
// It automatically sets up the database schema using migrations, making it ready for tests.
// The function uses the following order of precedence for the database URL:
//
// 1. DATABASE_URL environment variable (used by CI/CD and integration tests)
// 2. SCRY_TEST_DB_URL environment variable (for local developer configuration)
// 3. Default local database URL (for developer convenience)
//
// This function handles proper connection validation and initialization, ensuring
// that tests can immediately use the returned database connection without additional setup.
//
// Usage:
//
//	db, err := testutils.GetTestDB()
//	require.NoError(t, err)
//	defer testutils.AssertCloseNoError(t, db)
//
//	// db is now ready for use in tests
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
			return nil, fmt.Errorf("database ping failed: %w (and failed to close connection: %v)", err, closeErr)
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
