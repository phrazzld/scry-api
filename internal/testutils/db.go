//go:build test_without_external_deps

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
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/ciutil"
	"github.com/pressly/goose/v3"
)

var (
	// migrationsRunOnce ensures migrations are only run once across all tests
	migrationsRunOnce sync.Once
)

// DatabaseHealthCheckConfig configures database connection health check behavior.
type DatabaseHealthCheckConfig struct {
	// MaxRetries is the maximum number of connection retry attempts
	MaxRetries int
	// InitialRetryDelay is the initial delay between retry attempts
	InitialRetryDelay time.Duration
	// MaxRetryDelay is the maximum delay between retry attempts
	MaxRetryDelay time.Duration
	// ConnectionTimeout is the timeout for individual connection attempts
	ConnectionTimeout time.Duration
	// ValidationTimeout is the timeout for running validation queries
	ValidationTimeout time.Duration
	// EnableAutoHealthCheck enables automatic health checks in GetTestDBWithT
	EnableAutoHealthCheck bool
}

// DefaultHealthCheckConfig returns a sensible default configuration for database health checks.
func DefaultHealthCheckConfig() DatabaseHealthCheckConfig {
	return DatabaseHealthCheckConfig{
		MaxRetries:            3,
		InitialRetryDelay:     500 * time.Millisecond,
		MaxRetryDelay:         5 * time.Second,
		ConnectionTimeout:     10 * time.Second,
		ValidationTimeout:     5 * time.Second,
		EnableAutoHealthCheck: true,
	}
}

// CIHealthCheckConfig returns a configuration optimized for CI environments.
func CIHealthCheckConfig() DatabaseHealthCheckConfig {
	return DatabaseHealthCheckConfig{
		MaxRetries:            5,
		InitialRetryDelay:     1 * time.Second,
		MaxRetryDelay:         10 * time.Second,
		ConnectionTimeout:     15 * time.Second,
		ValidationTimeout:     10 * time.Second,
		EnableAutoHealthCheck: true,
	}
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

// GetTestDBWithT returns a database connection for testing with automatic health checks.
// It automatically sets up the database schema using migrations, making it ready for tests.
// The function uses standardized database URL handling consistent with the rest of the system,
// including proper CI environment detection and URL standardization.
//
// The function automatically performs database health checks with retry logic before
// returning the connection, ensuring robust database connectivity in CI environments.
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
	return GetTestDBWithTAndConfig(t, getHealthCheckConfigForEnvironment())
}

// GetTestDBWithTAndConfig returns a database connection for testing with custom health check configuration.
func GetTestDBWithTAndConfig(t *testing.T, config DatabaseHealthCheckConfig) *sql.DB {
	t.Helper()

	// Use standardized database URL handling from ciutil package
	// This ensures consistency with other parts of the system and proper CI standardization
	logger := slog.Default().With(
		slog.String("component", "testutils"),
		slog.String("function", "GetTestDBWithTAndConfig"),
	)

	dbURL := getDatabaseURLForTests(logger)
	if dbURL == "" {
		// Use default local database URL as fallback
		dbURL = "postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable"
		logger.Info("Using default database URL for tests",
			slog.String("url", "postgres://postgres:****@localhost:5432/scry_test?sslmode=disable"))
	}

	// Perform health check with retry logic if enabled
	if config.EnableAutoHealthCheck {
		logger.Info("Performing automatic database health check")
		performTestHealthCheck(t, dbURL, config, logger)
	}

	// Open database connection (this should succeed after health check)
	logger.Info("Opening database connection for tests",
		slog.String("masked_url", maskDBURL(dbURL)))

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

	// Verify the connection works with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
	defer cancel()

	logger.Info("Testing database connection")
	if err := db.PingContext(ctx); err != nil {
		logger.Error("Database connection failed",
			slog.String("error", err.Error()),
			slog.String("masked_url", maskDBURL(dbURL)))
		t.Fatalf("Database ping failed: %v", err)
	}
	logger.Info("Database connection successful")

	// Setup database schema
	if err := SetupTestDatabaseSchema(db); err != nil {
		t.Fatalf("Failed to setup database schema: %v", err)
	}

	// Configure connection pool settings for tests
	db.SetMaxOpenConns(25) // Reasonable number of concurrent connections for tests
	db.SetMaxIdleConns(25) // Keep connections ready for test parallelism
	db.SetConnMaxLifetime(5 * time.Minute)

	logger.Info("Database setup completed successfully")
	return db
}

// getHealthCheckConfigForEnvironment returns the appropriate health check configuration
// based on the current environment (CI vs local development).
func getHealthCheckConfigForEnvironment() DatabaseHealthCheckConfig {
	if ciutil.IsCI() {
		return CIHealthCheckConfig()
	}
	return DefaultHealthCheckConfig()
}

// performTestHealthCheck runs a lightweight health check suitable for test setup.
// This is separate from the full ValidateDatabaseConnection to avoid test framework dependencies.
func performTestHealthCheck(t *testing.T, dbURL string, config DatabaseHealthCheckConfig, logger *slog.Logger) {
	t.Helper()

	// Attempt connection with retry logic
	db, err := connectWithRetry(dbURL, config, logger)
	if err != nil {
		t.Fatalf("Database health check failed after %d retries: %v", config.MaxRetries, err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Warn("Failed to close health check database connection",
				slog.String("error", closeErr.Error()))
		}
	}()

	// Perform basic validation
	if err := performDatabaseValidation(db, config, logger); err != nil {
		t.Fatalf("Database health check validation failed: %v", err)
	}

	logger.Info("Database health check completed successfully")
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
	return GetTestDBWithHealthCheck(getHealthCheckConfigForEnvironment())
}

// GetTestDBWithHealthCheck returns a database connection with custom health check configuration.
// This version returns an error rather than using t.Helper for non-test contexts.
func GetTestDBWithHealthCheck(config DatabaseHealthCheckConfig) (*sql.DB, error) {
	// Use standardized database URL handling for consistency
	logger := slog.Default().With(
		slog.String("component", "testutils"),
		slog.String("function", "GetTestDBWithHealthCheck"),
	)

	dbURL := getDatabaseURLForTests(logger)
	if dbURL == "" {
		// Use default local database URL as fallback
		dbURL = "postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable"
		logger.Info("Using default database URL for tests")
	}

	// Perform health check with retry logic if enabled
	if config.EnableAutoHealthCheck {
		logger.Info("Performing automatic database health check")
		db, err := connectWithRetry(dbURL, config, logger)
		if err != nil {
			return nil, fmt.Errorf("database health check failed after %d retries: %w", config.MaxRetries, err)
		}

		// Perform validation
		if err := performDatabaseValidation(db, config, logger); err != nil {
			if closeErr := db.Close(); closeErr != nil {
				logger.Warn("Failed to close database during cleanup", slog.String("close_error", closeErr.Error()))
			}
			return nil, fmt.Errorf("database health check validation failed: %w", err)
		}

		// Close the health check connection
		if err := db.Close(); err != nil {
			logger.Warn("Failed to close health check connection", slog.String("error", err.Error()))
		}

		logger.Info("Database health check completed successfully")
	}

	// Open database connection (this should succeed after health check)
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Verify the connection works with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
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

// getDatabaseURLForTests returns a database URL using standardized logic.
// This ensures consistency with other parts of the system and proper CI environment handling.
func getDatabaseURLForTests(logger *slog.Logger) string {
	// Use the standardized database URL handling from ciutil
	return ciutil.GetTestDatabaseURL(logger)
}

// maskDBURL masks sensitive information in database URLs for safe logging.
func maskDBURL(dbURL string) string {
	if dbURL == "" {
		return ""
	}
	return ciutil.MaskSensitiveValue(dbURL)
}

// ValidateDatabaseConnection performs comprehensive database connection health checks
// using the default configuration.
func ValidateDatabaseConnection(t *testing.T) {
	t.Helper()
	ValidateDatabaseConnectionWithConfig(t, DefaultHealthCheckConfig())
}

// ValidateDatabaseConnectionWithConfig performs comprehensive database connection health checks
// with retry logic and configurable timeouts.
func ValidateDatabaseConnectionWithConfig(t *testing.T, config DatabaseHealthCheckConfig) {
	t.Helper()

	logger := slog.Default().With(
		slog.String("component", "testutils"),
		slog.String("function", "ValidateDatabaseConnectionWithConfig"),
	)

	// Get database URL using standardized logic
	dbURL := getDatabaseURLForTests(logger)
	if dbURL == "" {
		t.Fatal("No database URL available for connection validation")
	}

	logger.Info("Starting database connection validation with retry logic",
		slog.String("masked_url", maskDBURL(dbURL)),
		slog.Int("max_retries", config.MaxRetries),
		slog.Duration("connection_timeout", config.ConnectionTimeout))

	// Attempt connection with retry logic
	db, err := connectWithRetry(dbURL, config, logger)
	if err != nil {
		t.Fatalf("Database connection validation failed after %d retries: %v", config.MaxRetries, err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Warn("Failed to close validation database connection",
				slog.String("error", closeErr.Error()))
		}
	}()

	// Perform comprehensive validation
	if err := performDatabaseValidation(db, config, logger); err != nil {
		t.Fatalf("Database validation failed: %v", err)
	}

	logger.Info("Database connection validation successful")
}

// connectWithRetry attempts to establish a database connection with exponential backoff retry logic.
func connectWithRetry(dbURL string, config DatabaseHealthCheckConfig, logger *slog.Logger) (*sql.DB, error) {
	var db *sql.DB
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			delay := time.Duration(float64(config.InitialRetryDelay) * math.Pow(2, float64(attempt-1)))
			if delay > config.MaxRetryDelay {
				delay = config.MaxRetryDelay
			}

			logger.Info("Retrying database connection",
				slog.Int("attempt", attempt),
				slog.Int("max_attempts", config.MaxRetries+1),
				slog.Duration("delay", delay))

			time.Sleep(delay)
		}

		// Attempt to open connection
		var err error
		db, err = sql.Open("pgx", dbURL)
		if err != nil {
			lastErr = fmt.Errorf("failed to open database connection (attempt %d): %w", attempt+1, err)
			logger.Warn("Database connection attempt failed",
				slog.Int("attempt", attempt+1),
				slog.String("error", err.Error()))
			continue
		}

		// Test the connection with timeout
		ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
		err = db.PingContext(ctx)
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("database ping failed (attempt %d): %w", attempt+1, err)
			logger.Warn("Database ping attempt failed",
				slog.Int("attempt", attempt+1),
				slog.String("error", err.Error()))

			// Close the connection before retrying
			if closeErr := db.Close(); closeErr != nil {
				logger.Warn("Failed to close failed connection",
					slog.String("error", closeErr.Error()))
			}
			continue
		}

		// Connection successful
		logger.Info("Database connection established",
			slog.Int("attempt", attempt+1))
		return db, nil
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// performDatabaseValidation runs comprehensive validation tests on an established database connection.
func performDatabaseValidation(db *sql.DB, config DatabaseHealthCheckConfig, logger *slog.Logger) error {
	// Test basic query execution
	ctx, cancel := context.WithTimeout(context.Background(), config.ValidationTimeout)
	defer cancel()

	logger.Info("Testing basic query execution")
	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		logger.Error("Database query test failed during validation",
			slog.String("error", err.Error()))
		return fmt.Errorf("query test failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("query test returned unexpected result: got %d, expected 1", result)
	}

	// Test transaction capabilities
	logger.Info("Testing transaction capabilities")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("Transaction begin failed during validation",
			slog.String("error", err.Error()))
		return fmt.Errorf("transaction begin failed: %w", err)
	}

	// Test query within transaction
	var txResult int
	if err := tx.QueryRowContext(ctx, "SELECT 2").Scan(&txResult); err != nil {
		logger.Error("Transaction query failed during validation",
			slog.String("error", err.Error()))
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			logger.Warn(
				"Failed to rollback transaction during cleanup",
				slog.String("rollback_error", rollbackErr.Error()),
			)
		}
		return fmt.Errorf("transaction query failed: %w", err)
	}

	// Rollback the test transaction
	if err := tx.Rollback(); err != nil {
		logger.Error("Transaction rollback failed during validation",
			slog.String("error", err.Error()))
		return fmt.Errorf("transaction rollback failed: %w", err)
	}

	logger.Info("Database validation tests completed successfully")
	return nil
}

// ValidateDatabaseEnvironment checks that database environment variables are properly configured.
// This is useful for CI diagnostics when tests fail due to environment issues.
func ValidateDatabaseEnvironment() {
	logger := slog.Default().With(
		slog.String("component", "testutils"),
		slog.String("function", "ValidateDatabaseEnvironment"),
	)

	// Check if we're in CI
	inCI := ciutil.IsCI()
	logger.Info("Environment validation",
		slog.Bool("ci_environment", inCI))

	// Check for required environment variables
	envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}
	foundVars := make([]string, 0)

	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			foundVars = append(foundVars, envVar)
			logger.Info("Environment variable found",
				slog.String("var", envVar),
				slog.String("masked_value", ciutil.MaskSensitiveValue(value)))
		}
	}

	if len(foundVars) == 0 {
		logger.Error("No database environment variables found",
			slog.String("checked_vars", strings.Join(envVars, ", ")))
	} else {
		logger.Info("Database environment variables available",
			slog.String("found_vars", strings.Join(foundVars, ", ")))
	}

	// Additional CI-specific checks
	if inCI {
		logger.Info("CI environment detected - checking additional variables")
		ciVars := []string{"CI", "GITHUB_ACTIONS", "GITHUB_WORKSPACE"}
		for _, envVar := range ciVars {
			if value := os.Getenv(envVar); value != "" {
				logger.Info("CI environment variable found",
					slog.String("var", envVar),
					slog.String("value", value))
			}
		}
	}
}

// GetTestDBWithoutHealthCheck returns a database connection for testing without health checks.
// This can be useful in scenarios where health checks might interfere with test setup
// or when testing database failure scenarios.
func GetTestDBWithoutHealthCheck(t *testing.T) *sql.DB {
	t.Helper()
	config := getHealthCheckConfigForEnvironment()
	config.EnableAutoHealthCheck = false
	return GetTestDBWithTAndConfig(t, config)
}

// DisabledHealthCheckConfig returns a configuration with health checks disabled.
// This is useful for testing scenarios where you want to bypass health checks.
func DisabledHealthCheckConfig() DatabaseHealthCheckConfig {
	config := DefaultHealthCheckConfig()
	config.EnableAutoHealthCheck = false
	return config
}

// AssertCloseNoError is implemented in helpers.go
