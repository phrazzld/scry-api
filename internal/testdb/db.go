//go:build integration || test_without_external_deps

// Package testdb provides utilities specifically for database testing.
// It maintains a clean dependency structure by only depending on store interfaces
// and standard database packages, not on specific implementations.
package testdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// TestTimeout defines a default timeout for test database operations.
const TestTimeout = 5 * time.Second

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

// GetTestDatabaseURL returns the database URL for tests.
// It checks DATABASE_URL, SCRY_TEST_DB_URL, and SCRY_DATABASE_URL environment variables
// in that order, returning the first non-empty value.
func GetTestDatabaseURL() string {
	// Check environment variables in priority order
	envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}

	for _, envVar := range envVars {
		dbURL := os.Getenv(envVar)
		if dbURL != "" {
			// Log which environment variable was used if we're in CI
			if os.Getenv("CI") != "" {
				fmt.Printf("Using database URL from %s environment variable: %s\n",
					envVar, maskDatabaseURL(dbURL))

				// Check for potential user issues in CI environment to prevent "role root does not exist" errors
				if isCIEnvironment() {
					// Parse URL to check for problematic usernames
					parsedURL, err := url.Parse(dbURL)
					if err == nil && parsedURL.User != nil {
						username := parsedURL.User.Username()
						if username == "root" {
							fmt.Println(
								"WARNING: Database URL contains 'root' user which may not exist in CI PostgreSQL. " +
									"Consider using 'postgres' user instead.",
							)

							// Check if we can automatically fix this in CI
							if os.Getenv("GITHUB_ACTIONS") != "" &&
								(username == "root" || username == "") {
								// Fix the URL to use postgres user
								password, _ := parsedURL.User.Password()
								parsedURL.User = url.UserPassword("postgres", password)
								fmt.Printf("Auto-fixing database URL to use 'postgres' user: %s\n",
									maskDatabaseURL(parsedURL.String()))
								return parsedURL.String()
							}
						}
					}
				}
			}
			return dbURL
		}
	}

	// No valid URL found
	return ""
}

// SetupTestDatabaseSchema runs database migrations to set up the test database.
// It provides enhanced error messages with diagnostics information for common failures.
func SetupTestDatabaseSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	// Check database connection first
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		dbURL := GetTestDatabaseURL()
		errDetail := formatDBConnectionError(err, dbURL)
		t.Fatalf("Database connection failed before migrations: %v", errDetail)
	}

	// Find project root to locate migration files
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Set up goose for migrations
	migrationsDir := filepath.Join(projectRoot, "internal", "platform", "postgres", "migrations")

	// Check that migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		errDetail := fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
		t.Fatalf("Migrations directory error: %v", errDetail)
	}

	// Configure goose with custom logger
	goose.SetLogger(&testGooseLogger{t: t})
	goose.SetTableName(MigrationTableName)
	goose.SetBaseFS(os.DirFS(migrationsDir))

	// Try to run migrations with enhanced error reporting
	if err := goose.Up(db, "."); err != nil {
		errDetail := formatMigrationError(err, migrationsDir)
		t.Fatalf("Migration failed: %v", errDetail)
	}

	t.Logf("Database migrations applied successfully to schema")
}

// WithTx executes a test function within a transaction, automatically rolling back
// after the test completes. This ensures test isolation and prevents side effects.
// It provides enhanced error handling and diagnostics for transaction failures.
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()

	// Verify database connection is active
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		dbURL := GetTestDatabaseURL()

		// Log more diagnostic information in CI environments
		if isCIEnvironment() {
			fmt.Printf("CI Debug: Database ping failed with error: %v\n", err)
			fmt.Printf("CI Debug: Database URL (masked): %s\n", maskDatabaseURL(dbURL))

			// Try to diagnose connection issues
			parsedURL, parseErr := url.Parse(dbURL)
			if parseErr == nil && parsedURL.User != nil {
				username := parsedURL.User.Username()
				fmt.Printf("CI Debug: Attempting connection with user: %s\n", username)
			}

			// Try a simpler query to check connectivity
			queryCtx, queryCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer queryCancel()
			var one int
			if queryErr := db.QueryRowContext(queryCtx, "SELECT 1").Scan(&one); queryErr != nil {
				fmt.Printf("CI Debug: Simple 'SELECT 1' query also failed: %v\n", queryErr)
			} else {
				fmt.Printf("CI Debug: Simple 'SELECT 1' query succeeded with result: %d\n", one)
			}
		}

		errDetail := formatDBConnectionError(err, dbURL)
		t.Fatalf("Database connection failed before transaction: %v", errDetail)
	}

	// Start a transaction with timeout context
	txCtx, txCancel := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout
	defer txCancel()

	tx, err := db.BeginTx(txCtx, nil)
	if err != nil {
		// Additional diagnostics in CI
		if isCIEnvironment() {
			fmt.Printf("CI Debug: Transaction start failed: %v\n", err)
			stats := db.Stats()
			fmt.Printf("CI Debug: Current connection stats: MaxOpen=%d, Open=%d, InUse=%d, Idle=%d\n",
				stats.MaxOpenConnections, stats.OpenConnections, stats.InUse, stats.Idle)
		}

		t.Fatalf(
			"Failed to begin transaction: %v\nThis may indicate database connectivity issues or resource constraints",
			err,
		)
	}

	// Add transaction metadata for debugging if available
	if tx != nil {
		// Some drivers support querying transaction state
		t.Logf("Transaction started successfully")

		// In CI, verify transaction is working with a simple query
		if isCIEnvironment() {
			var one int
			if err := tx.QueryRow("SELECT 1").Scan(&one); err != nil {
				t.Logf("Warning: Test transaction may be unstable - simple query failed: %v", err)
			}
		}
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

// ApplyMigrations runs migrations without using testing.T
// This exists for backward compatibility with code that was written
// before the testdb package was created.
// The function includes enhanced error handling and diagnostics.
func ApplyMigrations(db *sql.DB, migrationsDir string) error {
	// Verify database connection is active
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		// We can't use formatDBConnectionError here since we don't have the URL
		// Instead, create a descriptive error message
		return fmt.Errorf("database connection failed before migrations: %w", err)
	}

	// Verify migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
	}

	// Configure goose
	goose.SetTableName("schema_migrations")
	goose.SetBaseFS(os.DirFS(migrationsDir))

	// Run migrations with comprehensive error handling
	if err := goose.Up(db, "."); err != nil {
		// Create detailed error with migration information
		migrationFiles := ""
		if entries, err := os.ReadDir(migrationsDir); err == nil && len(entries) > 0 {
			names := make([]string, 0, len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					names = append(names, entry.Name())
				}
			}
			migrationFiles = fmt.Sprintf(" (available migrations: %v)", names)
		}

		return fmt.Errorf("failed to run migrations in %s%s: %w", migrationsDir, migrationFiles, err)
	}

	return nil
}

// findProjectRoot locates the project root directory by traversing upwards
// until it finds a directory with go.mod file. Provides enhanced diagnostics
// particularly for CI environments.
//
// Project root detection order:
// 1. SCRY_PROJECT_ROOT environment variable (if set)
// 2. GITHUB_WORKSPACE environment variable (GitHub Actions CI)
// 3. CI_PROJECT_DIR environment variable (GitLab CI)
// 4. Smart traversal from current directory looking for go.mod
//
// If you're experiencing issues with project root detection in CI:
// - Set SCRY_PROJECT_ROOT environment variable to explicitly specify the path
// - Ensure the repository is properly cloned in the CI environment
// - Check if the repository is part of a monorepo structure
func findProjectRoot() (string, error) {
	// Keep track of paths we've checked for debugging
	checkedPaths := []string{}
	checkedEnvVars := []string{}

	// 1. Check for explicit project root environment variable
	if projectRoot := os.Getenv("SCRY_PROJECT_ROOT"); projectRoot != "" {
		checkedEnvVars = append(checkedEnvVars, "SCRY_PROJECT_ROOT="+projectRoot)
		goModPath := filepath.Join(projectRoot, "go.mod")
		checkedPaths = append(checkedPaths, goModPath)

		if _, err := os.Stat(goModPath); err == nil {
			return projectRoot, nil
		}

		// If SCRY_PROJECT_ROOT is set but invalid, log a warning
		fmt.Printf("Warning: SCRY_PROJECT_ROOT is set to %q but go.mod not found at %s\n",
			projectRoot, goModPath)
	}

	// 2. Special handling for GitHub Actions CI environment
	if githubWorkspace := os.Getenv("GITHUB_WORKSPACE"); githubWorkspace != "" {
		checkedEnvVars = append(checkedEnvVars, "GITHUB_WORKSPACE="+githubWorkspace)

		// Try direct path
		goModPath := filepath.Join(githubWorkspace, "go.mod")
		checkedPaths = append(checkedPaths, goModPath)

		if _, err := os.Stat(goModPath); err == nil {
			return githubWorkspace, nil
		}

		// Try with 'scry-api' subdirectory for monorepo setups
		repoPath := filepath.Join(githubWorkspace, "scry-api")
		goModPath = filepath.Join(repoPath, "go.mod")
		checkedPaths = append(checkedPaths, goModPath)

		if _, err := os.Stat(goModPath); err == nil {
			return repoPath, nil
		}
	}

	// 3. Check for GitLab CI environment
	if gitlabProjectDir := os.Getenv("CI_PROJECT_DIR"); gitlabProjectDir != "" {
		checkedEnvVars = append(checkedEnvVars, "CI_PROJECT_DIR="+gitlabProjectDir)
		goModPath := filepath.Join(gitlabProjectDir, "go.mod")
		checkedPaths = append(checkedPaths, goModPath)

		if _, err := os.Stat(goModPath); err == nil {
			return gitlabProjectDir, nil
		}
	}

	// 3a. More explicit GitHub Actions detection with better debugging
	if isCIEnvironment() && os.Getenv("GITHUB_ACTIONS") != "" {
		// Try to be even more explicit by looking at GITHUB_WORKSPACE
		githubWorkspace := os.Getenv("GITHUB_WORKSPACE")
		if githubWorkspace != "" {
			// Log for debugging in CI
			fmt.Printf("CI debug: Checking GITHUB_WORKSPACE=%s for go.mod\n", githubWorkspace)

			// Try direct path again with explicit logging
			goModPath := filepath.Join(githubWorkspace, "go.mod")
			if _, err := os.Stat(goModPath); err == nil {
				fmt.Printf("CI debug: Found go.mod at %s\n", goModPath)
				return githubWorkspace, nil
			} else {
				fmt.Printf("CI debug: go.mod not found at %s: %v\n", goModPath, err)
			}

			// List directory contents for debugging
			if entries, err := os.ReadDir(githubWorkspace); err == nil {
				names := make([]string, 0, len(entries))
				for _, entry := range entries {
					names = append(names, entry.Name())
				}
				fmt.Printf("CI debug: GITHUB_WORKSPACE contents: %v\n", names)
			}
		}
	}

	// 4. Start with current working directory and traverse upwards
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Try common project subdirectories first if we might be in a nested location
	if isCIEnvironment() {
		// Check if we're in a subdirectory of the project
		commonDirs := []string{"internal", "cmd", "pkg", "test"}
		for _, subdir := range commonDirs {
			if strings.HasSuffix(dir, subdir) || strings.Contains(dir, "/"+subdir+"/") {
				// Try to find project root by traversing up
				potentialRoot := dir
				for i := 0; i < 5; i++ { // Go up max 5 levels
					potentialRoot = filepath.Dir(potentialRoot)
					goModPath := filepath.Join(potentialRoot, "go.mod")
					checkedPaths = append(checkedPaths, goModPath)

					if _, err := os.Stat(goModPath); err == nil {
						return potentialRoot, nil
					}
				}
				break
			}
		}
	}

	// Standard traversal - go up until we find go.mod
	maxAttempts := 10 // Prevent infinite loops
	attempts := 0

	for attempts < maxAttempts {
		attempts++

		// Check if go.mod exists in the current directory
		goModPath := filepath.Join(dir, "go.mod")
		checkedPaths = append(checkedPaths, goModPath)

		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(dir)
		// If we're at the root and haven't found go.mod, we've gone too far
		if parentDir == dir {
			break
		}
		dir = parentDir
	}

	// Last resort: Check for specific project structure patterns
	currentDir, _ := os.Getwd()
	for _, segment := range []string{"scry-api", "scry", "scry-api-go"} {
		if strings.Contains(currentDir, segment) {
			// Extract the project root by finding the segment in the path
			idx := strings.Index(currentDir, segment)
			if idx != -1 {
				possibleRoot := currentDir[:idx+len(segment)]
				goModPath := filepath.Join(possibleRoot, "go.mod")
				checkedPaths = append(checkedPaths, goModPath)

				if _, err := os.Stat(goModPath); err == nil {
					return possibleRoot, nil
				}
			}
		}
	}

	// Use our standardized error formatting function
	return "", formatProjectRootError(checkedPaths, checkedEnvVars)
}

// isCIEnvironment returns true if running in a CI environment
func isCIEnvironment() bool {
	// Check common CI environment variables
	ciVars := []string{
		"CI",             // Generic
		"GITHUB_ACTIONS", // GitHub Actions
		"GITLAB_CI",      // GitLab CI
		"JENKINS_URL",    // Jenkins
		"TRAVIS",         // Travis CI
		"CIRCLECI",       // Circle CI
	}

	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
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

// getCurrentDir returns the current working directory or an error message if it fails
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("error getting current directory: %v", err)
	}
	return dir
}

// Error Helper Functions

// formatDBConnectionError creates a detailed error message for database connection failures.
// It includes environment variable status, connection details, and troubleshooting guidance.
func formatDBConnectionError(baseErr error, dbURL string) error {
	// Basic environment info
	envInfo := fmt.Sprintf("CI environment: %v\nCurrent working directory: %s",
		isCIEnvironment(), getCurrentDir())

	// Database connection info (safely masked)
	dbInfo := fmt.Sprintf("Database URL used: %s (masked)", maskDatabaseURL(dbURL))

	// Format the comprehensive error message
	errMsg := fmt.Sprintf("Database connection failed: %v\n%s\n%s\n"+
		"Please check:\n"+
		"1. PostgreSQL service is running\n"+
		"2. Credentials and connection string are correct\n"+
		"3. Database exists and is accessible\n"+
		"4. Network connectivity and firewall settings",
		baseErr, dbInfo, envInfo)

	return fmt.Errorf("%s", errMsg)
}

// formatEnvVarError creates a detailed error message when required environment variables are missing.
// It provides guidance on which variables should be set and current environment status.
func formatEnvVarError() error {
	// Check which environment variables are missing
	envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}
	missingVars := []string{}

	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}

	// Environment status information
	envInfo := fmt.Sprintf("CI environment: %v\nCurrent working directory: %s",
		isCIEnvironment(), getCurrentDir())

	// Create the error message
	errMsg := fmt.Sprintf("Database connection failed: no database URL available\n"+
		"Required environment variables missing: %v\n%s\n"+
		"Please ensure one of DATABASE_URL, SCRY_TEST_DB_URL, or SCRY_DATABASE_URL is set.",
		missingVars, envInfo)

	return fmt.Errorf("%s", errMsg)
}

// formatProjectRootError creates a detailed error message when the project root cannot be found.
// It includes paths checked, environment variables, and suggested actions.
func formatProjectRootError(checkedPaths []string, checkedEnvVars []string) error {
	dir := getCurrentDir()

	// Get current directory contents for debugging
	dirContents := ""
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		dirContents = fmt.Sprintf("\nCurrent directory contents: %v", names)
	}

	// Create comprehensive error message
	errMsg := fmt.Sprintf("Could not find go.mod in any parent directory.\n"+
		"Checked environment variables: %v\n"+
		"Checked paths: %v\n"+
		"Current directory: %s%s\n"+
		"CI environment: %v\n"+
		"CI environment vars: GITHUB_WORKSPACE=%s, CI_PROJECT_DIR=%s\n"+
		"To fix this, set SCRY_PROJECT_ROOT environment variable to the project root directory",
		checkedEnvVars, checkedPaths, dir, dirContents,
		isCIEnvironment(), os.Getenv("GITHUB_WORKSPACE"), os.Getenv("CI_PROJECT_DIR"))

	return fmt.Errorf("%s", errMsg)
}

// formatMigrationError creates a detailed error message when database migrations fail.
// It includes information about the migrations directory, error details, and suggestions.
func formatMigrationError(baseErr error, migrationsDir string) error {
	// Verify if the migrations directory exists
	dirExists := "exists"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		dirExists = "does not exist"
	}

	// Get list of migration files for debugging if directory exists
	migrationFiles := ""
	if dirExists == "exists" {
		if entries, err := os.ReadDir(migrationsDir); err == nil && len(entries) > 0 {
			names := make([]string, 0, len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					names = append(names, entry.Name())
				}
			}
			migrationFiles = fmt.Sprintf("\nMigration files: %v", names)
		}
	}

	// Environment information
	envInfo := fmt.Sprintf("CI environment: %v\nCurrent working directory: %s",
		isCIEnvironment(), getCurrentDir())

	// Create comprehensive error message
	errMsg := fmt.Sprintf("Failed to run database migrations: %v\n"+
		"Migrations directory: %s (%s)%s\n%s\n"+
		"Please check:\n"+
		"1. Migrations directory path is correct\n"+
		"2. Migration files exist and are valid\n"+
		"3. Database connection is working\n"+
		"4. Database user has permissions to create tables and modify schema",
		baseErr, migrationsDir, dirExists, migrationFiles, envInfo)

	return fmt.Errorf("%s", errMsg)
}

// maskDatabaseURL masks sensitive information in a database URL for safe logging
// Format: postgres://username:password@hostname:port/database?parameters
func maskDatabaseURL(dbURL string) string {
	// If empty or invalid format, return safely
	if dbURL == "" {
		return ""
	}

	// Try regex matching first for consistent output format
	re := regexp.MustCompile(`://([^:]+):([^@]+)@`)
	if re.MatchString(dbURL) {
		return re.ReplaceAllString(dbURL, "://$1:****@")
	}

	// Fall back to URL parsing if regex doesn't match
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		// If both approaches fail, return a generic masked version
		return "database-url-with-masked-credentials"
	}

	// For properly parsed URLs, mask the password
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		parsedURL.User = url.UserPassword(username, "****")
		return parsedURL.String()
	}

	// If no user info is found, return the original URL
	return dbURL
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
// Returns an error with detailed diagnostics if the database connection cannot be established.
func GetTestDB() (*sql.DB, error) {
	// Check if the database URL is available
	dbURL := GetTestDatabaseURL()
	if dbURL == "" {
		return nil, formatEnvVarError()
	}

	// Open database connection
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w\nDatabase URL format may be incorrect", err)
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
		// Close the connection if ping fails
		closeErr := db.Close()

		// Create the base error with detailed diagnostics
		baseErr := formatDBConnectionError(err, dbURL)

		// Add any additional connection close errors if they occurred
		if closeErr != nil {
			return nil, fmt.Errorf("%v\nAdditional error when closing connection: %w", baseErr, closeErr)
		}

		return nil, baseErr
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
// This function is an alias for WithTx maintained for backward compatibility.
func RunInTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()

	// Simply call WithTx to avoid code duplication
	WithTx(t, db, fn)
}
