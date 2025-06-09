//go:build compat_layer

// Package db provides compatibility functions that forward to the testdb package.
// This package is being gradually deprecated in favor of using testdb directly.
// This file contains forwarding functions that maintain backwards compatibility
// during the transition.
package db

import (
	"database/sql"
	"testing"

	"github.com/phrazzld/scry-api/internal/testdb"
)

// IsIntegrationTestEnvironment returns true if any of the database URL environment
// variables are set, indicating that integration tests can be run.
//
// Deprecated: Use testdb.IsIntegrationTestEnvironment() instead.
func IsIntegrationTestEnvironment() bool {
	return testdb.IsIntegrationTestEnvironment()
}

// ShouldSkipDatabaseTest returns true if database-dependent tests should be skipped
// because no database connection is available.
//
// Deprecated: Use testdb.ShouldSkipDatabaseTest() instead.
func ShouldSkipDatabaseTest() bool {
	return testdb.ShouldSkipDatabaseTest()
}

// GetTestDatabaseURL returns the database URL for tests.
//
// Deprecated: Use testdb.GetTestDatabaseURL() instead.
func GetTestDatabaseURL() string {
	return testdb.GetTestDatabaseURL()
}

// SetupTestDatabaseSchema runs database migrations to set up the test database.
//
// Deprecated: Use testdb.SetupTestDatabaseSchema() instead.
func SetupTestDatabaseSchema(t *testing.T, db *sql.DB) {
	testdb.SetupTestDatabaseSchema(t, db)
}

// WithTx executes a test function within a transaction, automatically rolling back
// after the test completes.
//
// Deprecated: Use testdb.WithTx() instead.
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	testdb.WithTx(t, db, fn)
}

// GetTestDBWithT returns a database connection for testing.
//
// Deprecated: Use testdb.GetTestDBWithT() instead.
func GetTestDBWithT(t *testing.T) *sql.DB {
	return testdb.GetTestDBWithT(t)
}

// AssertRollbackNoError attempts to roll back a transaction and logs an error if it fails.
//
// Deprecated: Use testdb.AssertRollbackNoError() instead.
func AssertRollbackNoError(t *testing.T, tx *sql.Tx) {
	testdb.AssertRollbackNoError(t, tx)
}

// CleanupDB properly closes a database connection and logs any errors.
//
// Deprecated: Use testdb.CleanupDB() instead.
func CleanupDB(t *testing.T, db *sql.DB) {
	testdb.CleanupDB(t, db)
}
