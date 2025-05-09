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
				fmt.Printf("Using database URL from %s environment variable\n", envVar)
			}
			return dbURL
		}
	}

	// No valid URL found
	return ""
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

	// Enhanced error message with diagnostics
	dirContents := ""
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		dirContents = fmt.Sprintf("\nCurrent directory contents: %v", names)
	}

	return "", fmt.Errorf("could not find go.mod in any parent directory.\n"+
		"Checked environment variables: %v\n"+
		"Checked paths: %v\n"+
		"Current directory: %s%s\n"+
		"CI environment: %v\n"+
		"CI environment vars: GITHUB_WORKSPACE=%s, CI_PROJECT_DIR=%s\n"+
		"To fix this, set SCRY_PROJECT_ROOT environment variable to the project root directory",
		checkedEnvVars, checkedPaths, dir, dirContents,
		isCIEnvironment(), os.Getenv("GITHUB_WORKSPACE"), os.Getenv("CI_PROJECT_DIR"))
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
		// Provide more detailed error information about environment variables
		envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}
		missingVars := []string{}

		for _, envVar := range envVars {
			if os.Getenv(envVar) == "" {
				missingVars = append(missingVars, envVar)
			}
		}

		// Enhanced error message with detailed diagnostics
		errMsg := fmt.Sprintf("database connection failed: no database URL available\n"+
			"Required environment variables missing: %v\n"+
			"CI environment: %v\n"+
			"Current working directory: %s\n"+
			"Please ensure one of DATABASE_URL, SCRY_TEST_DB_URL, or SCRY_DATABASE_URL is set.",
			missingVars, os.Getenv("CI") != "", getCurrentDir())

		return nil, errors.New(errMsg)
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
		closeErr := db.Close()

		// Combine both errors in the message with enhanced diagnostics
		errMsg := fmt.Sprintf("database ping failed: %v\n"+
			"Database URL used: %s (masked)\n"+
			"CI environment: %v\n"+
			"Please check:\n"+
			"1. PostgreSQL service is running\n"+
			"2. Credentials are correct\n"+
			"3. Database exists\n"+
			"4. Network connectivity/firewall settings",
			err, maskDatabaseURL(dbURL), os.Getenv("CI") != "")

		if closeErr != nil {
			errMsg += fmt.Sprintf("\nAdditional error when closing connection: %v", closeErr)
		}

		return nil, errors.New(errMsg)
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
