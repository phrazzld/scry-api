//go:build forwarding_functions

// Package testutils provides test utilities and helpers for the application.
// This file forwards database-related functions to the correct implementations,
// maintaining backward compatibility during the migration to the new testdb structure.
package testutils

import (
	"database/sql"
	"testing"

	"github.com/phrazzld/scry-api/internal/testdb"
)

// GetTestDB returns a database connection for testing.
// This forwards to the appropriate implementation and provides backwards compatibility.
func GetTestDB() (*sql.DB, error) {
	return testdb.GetTestDB()
}

// SetupTestDatabaseSchema initializes the database schema using project migrations.
// This forwards to the appropriate implementation and provides backwards compatibility.
func SetupTestDatabaseSchema(dbConn *sql.DB) error {
	// Re-implementing the version that doesn't require a testing.T
	// See similar implementation in compatibility.go
	return testdb.ApplyMigrations(dbConn, "./internal/platform/postgres/migrations")
}

// WithTx executes a test function within a transaction, automatically rolling back
// after the test completes. This forwards to the appropriate implementation
// and provides backwards compatibility.
func WithTx(t *testing.T, dbConn *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	testdb.WithTx(t, dbConn, fn)
}
