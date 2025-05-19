package testutils_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/phrazzld/scry-api/internal/testutils/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransactionIsolation_StoresExample demonstrates how to use the transaction
// isolation pattern for database tests.
func TestTransactionIsolation_StoresExample(t *testing.T) {
	// Skip if not in integration test environment
	if !db.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection using the helper
	dbConn, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, dbConn)

	// This test shows the pattern of using transaction isolation with multiple stores
	t.Run("TransactionIsolationWithMultipleStores", func(t *testing.T) {
		t.Parallel() // Safe to run in parallel because of transaction isolation

		testutils.WithTx(t, dbConn, func(t *testing.T, tx *sql.Tx) {
			ctx := context.Background()

			// 1. Create a user directly in the database
			now := time.Now().UTC()
			userID := uuid.New()
			email := "transaction-test@example.com"

			// Create and execute the insert query
			_, err := tx.ExecContext(ctx,
				"INSERT INTO users (id, email, hashed_password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)",
				userID, email, "hashedpassword", now, now,
			)
			require.NoError(t, err, "Failed to insert test user")

			// 2. Create a memo for this user
			memoID := uuid.New()
			memo := &domain.Memo{
				ID:        memoID,
				UserID:    userID,
				Text:      "This is a test memo to demonstrate transaction isolation.",
				Status:    domain.MemoStatusPending,
				CreatedAt: now,
				UpdatedAt: now,
			}

			// Create and use memo store directly
			testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			memoStore := postgres.NewPostgresMemoStore(tx, testLogger)
			err = memoStore.Create(ctx, memo)
			require.NoError(t, err, "Failed to create memo")

			// 3. Try to retrieve the memo
			retrievedMemo, err := memoStore.GetByID(ctx, memo.ID)
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

		testutils.WithTx(t, dbConn, func(t *testing.T, tx *sql.Tx) {
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
	if !db.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection using the helper
	dbConn, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, dbConn)

	// Run multiple subtests that would conflict without transaction isolation
	for i := 0; i < 5; i++ {
		// Use i value directly in Go 1.22+
		t.Run("ConcurrentTest-"+string(rune('A'+i)), func(t *testing.T) {
			t.Parallel() // All these tests run in parallel

			testutils.WithTx(t, dbConn, func(t *testing.T, tx *sql.Tx) {
				ctx := context.Background()

				// Create a user with the same email in each test
				// This would fail without transaction isolation due to unique constraint
				email := "same-email-for-all@example.com"

				// Insert user directly
				userID := uuid.New()
				now := time.Now().UTC()
				_, err := tx.ExecContext(
					ctx,
					"INSERT INTO users (id, email, hashed_password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)",
					userID,
					email,
					"hashedpassword",
					now,
					now,
				)
				require.NoError(t, err, "Failed to insert test user")

				// Verify we can retrieve the user in this transaction
				var retrievedEmail string
				err = tx.QueryRowContext(ctx,
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
