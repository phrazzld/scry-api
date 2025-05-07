//go:build integration

package testutils

import (
	"database/sql"
	"testing"

	"github.com/phrazzld/scry-api/internal/testdb"
)

// WithTx executes a test function within a transaction, automatically rolling back
// after the test completes. This is a compatibility function that forwards to testdb.WithTx.
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	testdb.WithTx(t, db, fn)
}

// GetTestDB returns a database connection for testing.
// This is a compatibility function that forwards to testdb.GetTestDBWithT.
func GetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	return testdb.GetTestDBWithT(t)
}

// SetupTestDatabaseSchema runs database migrations to set up the test database.
// This is a compatibility function that forwards to testdb.SetupTestDatabaseSchema.
func SetupTestDatabaseSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	testdb.SetupTestDatabaseSchema(t, db)
}
