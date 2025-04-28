package testutils_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestTransactionIsolation_StoresExample demonstrates how to use the transaction
// isolation with the new CreateTestStores helper.
// This test serves as executable documentation for the proper testing pattern.
func TestTransactionIsolation_StoresExample(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection using the helper
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	// This test shows the pattern of using transaction isolation with multiple stores
	t.Run("TransactionIsolationWithMultipleStores", func(t *testing.T) {
		t.Parallel() // Safe to run in parallel because of transaction isolation

		testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
			ctx := context.Background()

			// Create all stores with the same transaction
			stores := testutils.CreateTestStores(tx, bcrypt.MinCost)

			// 1. Create a user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"transaction-test@example.com",
				bcrypt.MinCost,
			)

			// 2. Create a memo for this user
			memo := &domain.Memo{
				ID:        uuid.New(),
				UserID:    userID,
				Text:      "This is a test memo to demonstrate transaction isolation.",
				Status:    domain.MemoStatusPending,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			// Use the memo store to create the memo
			err := stores.MemoStore.Create(ctx, memo)
			require.NoError(t, err, "Failed to create memo")

			// 3. Try to retrieve the memo
			retrievedMemo, err := stores.MemoStore.GetByID(ctx, memo.ID)
			require.NoError(t, err, "Failed to retrieve memo")
			assert.Equal(t, memo.ID, retrievedMemo.ID, "Retrieved memo should have the same ID")
			assert.Equal(
				t,
				memo.Text,
				retrievedMemo.Text,
				"Retrieved memo should have the same text",
			)

			// The transaction will be automatically rolled back at the end of this function
			// No need to worry about cleanup!
		})
	})

	// This demonstrates that the transaction rollback actually worked
	t.Run("VerifyRollback", func(t *testing.T) {
		t.Parallel()

		testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
			ctx := context.Background()

			// Count memos with text containing the specific phrase from the previous test
			var count int
			err := tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM memos WHERE text LIKE '%transaction isolation%'",
			).Scan(&count)
			require.NoError(t, err, "Failed to count memos")

			// Confirm that the transaction from the previous test was indeed rolled back
			assert.Equal(
				t,
				0,
				count,
				"The memo from previous test should not exist - transaction should have been rolled back",
			)
		})
	})
}

// TestTransactionIsolation_Concurrency demonstrates that multiple tests can run
// concurrently without interfering with each other, even when they operate on the
// same tables and data.
func TestTransactionIsolation_Concurrency(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection using the helper
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	// Run multiple subtests that would conflict without transaction isolation
	for i := 0; i < 5; i++ {
		// Capture i for use in subtest
		t.Run("ConcurrentTest-"+string(rune('A'+i)), func(t *testing.T) {
			t.Parallel() // All these tests run in parallel

			testutils.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
				ctx := context.Background()

				// Create a user with the same email in each test
				// This would fail without transaction isolation due to unique constraint
				email := "same-email-for-all@example.com"
				userID := testutils.MustInsertUser(ctx, t, tx, email, bcrypt.MinCost)

				// Verify we can retrieve the user in this transaction
				var retrievedEmail string
				err := tx.QueryRowContext(ctx,
					"SELECT email FROM users WHERE id = $1",
					userID,
				).Scan(&retrievedEmail)
				require.NoError(t, err, "Failed to retrieve email")
				assert.Equal(t, email, retrievedEmail, "Email should match")

				// Each transaction sees its own world of data
				// Other transactions' users don't exist here
			})
		})
	}
}
