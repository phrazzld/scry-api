package testutils_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParallelIsolation demonstrates why the transaction-based approach to test
// isolation is better than the previous approach with table truncation.
// This test shows both approaches and verifies the transaction-based approach works.
func TestParallelIsolation(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Connect to database
	dbURL := testutils.MustGetTestDatabaseURL()
	db, err := sql.Open("pgx", dbURL)
	require.NoError(t, err, "Failed to open database connection")
	defer testutils.AssertCloseNoError(t, db)

	// Set up the database schema
	err = testutils.SetupTestDatabaseSchema(db)
	require.NoError(t, err, "Failed to set up test database schema")

	t.Run("WithoutTransactionIsolation", func(t *testing.T) {
		t.Skip(
			"Skipped by default to prevent table truncation. Remove t.Skip() to demonstrate the issue with non-transaction tests.",
		)

		// This test demonstrates the issue with non-transaction based tests
		// Reset test data for this test
		err := testutils.ResetTestData(db)
		require.NoError(t, err, "Failed to reset test data")

		// Create a test record
		ctx := context.Background()
		id1 := uuid.New()
		_, err = db.ExecContext(ctx, `
			INSERT INTO users (id, email, hashed_password, created_at, updated_at)
			VALUES ($1, 'test1@example.com', 'hash1', NOW(), NOW())
		`, id1)
		require.NoError(t, err, "Failed to insert test record")

		// Without transaction isolation, reset test data would truncate tables,
		// causing data loss for concurrent tests
		if testutils.ResetTestData(db) == nil {
			// Count users after reset - should be 0 if reset worked properly
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
			require.NoError(t, err, "Failed to count users")
			assert.Equal(t, 0, count, "ResetTestData should have truncated the users table")
		}
	})

	t.Run("WithTransactionIsolation", func(t *testing.T) {
		t.Parallel() // This is safe with transaction isolation

		// Use a WaitGroup to track concurrent subtests
		var wg sync.WaitGroup
		wg.Add(2)

		// Run two subtests that would conflict without transaction isolation
		t.Run("Transaction1", func(t *testing.T) {
			t.Parallel()
			defer wg.Done()

			testutils.WithTx(t, db, func(tx store.DBTX) {
				ctx := context.Background()
				// Insert a test record with a specific email
				id1 := uuid.New()
				_, err := tx.ExecContext(ctx, `
					INSERT INTO users (id, email, hashed_password, created_at, updated_at)
					VALUES ($1, 'same-email@example.com', 'hash1', NOW(), NOW())
				`, id1)
				require.NoError(t, err, "Failed to insert test record in Transaction1")

				// Verify the record exists in this transaction
				var count int
				err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = 'same-email@example.com'").
					Scan(&count)
				require.NoError(t, err, "Failed to count users")
				assert.Equal(t, 1, count, "Record should exist in Transaction1")

				// Sleep to ensure Transaction2 runs concurrently
				time.Sleep(100 * time.Millisecond)
			})
		})

		t.Run("Transaction2", func(t *testing.T) {
			t.Parallel()
			defer wg.Done()

			testutils.WithTx(t, db, func(tx store.DBTX) {
				ctx := context.Background()
				// Insert a test record with the same email in a different transaction
				id2 := uuid.New()
				_, err := tx.ExecContext(ctx, `
					INSERT INTO users (id, email, hashed_password, created_at, updated_at)
					VALUES ($1, 'same-email@example.com', 'hash2', NOW(), NOW())
				`, id2)
				require.NoError(
					t,
					err,
					"Should be able to insert record with same email in Transaction2",
				)

				// Verify the record exists in this transaction
				var count int
				err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = 'same-email@example.com'").
					Scan(&count)
				require.NoError(t, err, "Failed to count users")
				assert.Equal(t, 1, count, "Record should exist in Transaction2")
			})
		})

		// Wait for both subtests to complete
		wg.Wait()

		// Verify that after both transactions have been rolled back,
		// there are no records in the database with the test email
		var count int
		err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM users WHERE email = 'same-email@example.com'").
			Scan(&count)
		require.NoError(t, err, "Failed to count users")
		assert.Equal(t, 0, count, "No records should exist after transactions are rolled back")
	})
}
