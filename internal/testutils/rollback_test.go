package testutils

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAssertRollbackNoErrorReportsCorrectLine is intentionally designed to fail.
//
// IMPORTANT: This test is commented out by default. To use it:
//  1. Temporarily uncomment the WithTx block below
//  2. Run the test with: go test ./internal/testutils -run TestAssertRollbackNoErrorReportsCorrectLine
//  3. Manually verify in the output that the failure points to line 30 (the t.Fatal line),
//     not to the AssertRollbackNoError function or defer line
//  4. Recomment the code before committing
//
// The purpose of this test is to verify that the AssertRollbackNoError function
// correctly reports errors at the location they occur. Since it uses t.Helper(),
// failures should point to the line where the test fails, not where the rollback happens.
func TestAssertRollbackNoErrorReportsCorrectLine(t *testing.T) {
	t.Log("This test is designed to fail and is for manual verification only")

	// This test block is intentionally commented out to prevent test failures in normal runs.
	// When uncommenting for manual verification, the failure should point to the t.Fatal line.
	if false {
		// The SQL package needs to be used within an if false block to satisfy the import
		var tx *sql.Tx
		_ = tx // Avoid unused variable warning

		// Uncommenting below would cause test failure, intentionally:
		/*
			db := GetTestDBWithT(t)

			WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
				// This intentional failure should be reported at THIS line number in test output
				t.Fatal("intentional failure to verify line number reporting")
			})
		*/
	}
}

// TestAssertRollbackNoErrorHandlesTxDone verifies that AssertRollbackNoError
// properly handles the case where the transaction is already committed or rolled back.
//
// This test requires a database connection. When run without integration tags or
// database configuration, it will be skipped.
func TestAssertRollbackNoErrorHandlesTxDone(t *testing.T) {
	// Skip if DATABASE_URL is not set - this makes the test compatible with both
	// regular test runs and integration test runs
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set - skipping integration test")
	}

	// Get a test DB connection
	db := GetTestDBWithT(t)

	// Begin a transaction
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err, "Failed to begin transaction")

	// Rollback the transaction
	err = tx.Rollback()
	require.NoError(t, err, "Failed to rollback transaction")

	// Try to rollback again using AssertRollbackNoError - this should not panic or fail
	// even though the transaction is already rolled back
	AssertRollbackNoError(t, tx)
}
