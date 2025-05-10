//go:build integration

// This file is part of the compatibility layer for supporting older test code
// while transitioning to the new structure. The build tag "integration" ensures
// it's only included during integration tests, avoiding function redeclarations.

package testutils

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/stretchr/testify/require"
)

// IsIntegrationTestEnvironment forwards to testdb.IsIntegrationTestEnvironment
func IsIntegrationTestEnvironment() bool {
	return testdb.IsIntegrationTestEnvironment()
}

// ShouldSkipDatabaseTest forwards to testdb.ShouldSkipDatabaseTest
func ShouldSkipDatabaseTest() bool {
	return testdb.ShouldSkipDatabaseTest()
}

// WithTx executes a test function within a transaction, automatically rolling back
// after the test completes. This is a compatibility function that forwards to testdb.WithTx.
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	testdb.WithTx(t, db, fn)
}

// GetTestDBWithT returns a database connection for testing (new version with testing.T).
// This is a compatibility function that forwards to testdb.GetTestDBWithT.
func GetTestDBWithT(t *testing.T) *sql.DB {
	t.Helper()
	return testdb.GetTestDBWithT(t)
}

// GetTestDB returns a database connection for testing (original version returning error).
// This maintains the original function signature for backward compatibility.
func GetTestDB() (*sql.DB, error) {
	return testdb.GetTestDB()
}

// MustGetTestDatabaseURL returns the test database URL or panics if it's not available.
// This is a compatibility helper function for test code only.
func MustGetTestDatabaseURL() string {
	dbURL := testdb.GetTestDatabaseURL()
	if dbURL == "" {
		// ALLOW-PANIC
		panic("No test database URL available. Set DATABASE_URL, SCRY_TEST_DB_URL, or SCRY_DATABASE_URL")
	}
	return dbURL
}

// SetupTestDatabaseSchemaWithT runs database migrations to set up the test database.
// This is the new version that takes a testing.T parameter.
func SetupTestDatabaseSchemaWithT(t *testing.T, db *sql.DB) {
	t.Helper()
	testdb.SetupTestDatabaseSchema(t, db)
}

// SetupTestDatabaseSchema runs database migrations to set up the test database.
// This maintains the original function signature for backward compatibility.
func SetupTestDatabaseSchema(db *sql.DB) error {
	// We need an implementation that doesn't use testing.T features
	// Get project root - similar to the implementation in testdb
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find go.mod by traversing up directories
	projectRoot := ""
	for {
		if _, err := os.Stat(fmt.Sprintf("%s/go.mod", dir)); err == nil {
			projectRoot = dir
			break
		}

		parent := fmt.Sprintf("%s/..", dir)
		if parent == dir {
			return fmt.Errorf("could not find project root (go.mod file)")
		}
		dir = parent
	}

	// Set up migrations directory
	migrationsDir := fmt.Sprintf("%s/internal/platform/postgres/migrations", projectRoot)
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
	}

	// Run migrations
	if err := testdb.ApplyMigrations(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// AssertCloseNoError asserts that the given closer can be closed without error
func AssertCloseNoError(t *testing.T, closer interface{ Close() error }) {
	t.Helper()
	if closer == nil {
		return
	}
	require.NoError(t, closer.Close(), "Failed to close resource")
}

// RunInTx is an alias for WithTx for backward compatibility
func RunInTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	WithTx(t, db, fn)
}
