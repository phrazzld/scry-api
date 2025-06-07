//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"database/sql"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestPostgresMemoStore_Create tests the Create method
func TestPostgresMemoStore_Create(t *testing.T) {
	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create a new memo store
		memoStore := postgres.NewPostgresMemoStore(tx, nil)

		// Test Case 1: Successful memo creation
		t.Run("Successful memo creation", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Insert a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"memo-test-user@example.com",
				bcrypt.MinCost,
			)

			// Create a test memo
			memo := testutils.CreateTestMemo(t, userID)

			// Call the Create method
			err := memoStore.Create(ctx, memo)

			// Verify the result
			require.NoError(t, err, "Memo creation should succeed")

			// Verify the memo was inserted into the database
			var dbMemo domain.Memo
			var statusStr string

			err = tx.QueryRowContext(ctx, `
				SELECT id, user_id, text, status, created_at, updated_at
				FROM memos
				WHERE id = $1
			`, memo.ID).Scan(
				&dbMemo.ID,
				&dbMemo.UserID,
				&dbMemo.Text,
				&statusStr,
				&dbMemo.CreatedAt,
				&dbMemo.UpdatedAt,
			)

			require.NoError(t, err, "Should be able to retrieve memo")
			dbMemo.Status = domain.MemoStatus(statusStr)

			assert.Equal(t, memo.ID, dbMemo.ID, "Memo ID should match")
			assert.Equal(t, memo.UserID, dbMemo.UserID, "User ID should match")
			assert.Equal(t, memo.Text, dbMemo.Text, "Text should match")
			assert.Equal(t, memo.Status, dbMemo.Status, "Status should match")
			assert.False(t, dbMemo.CreatedAt.IsZero(), "CreatedAt should not be zero")
			assert.False(t, dbMemo.UpdatedAt.IsZero(), "UpdatedAt should not be zero")
		})

		// Test Case 2: Create memo with invalid data
		t.Run("Invalid memo data", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"invalid-memo-test@example.com",
				bcrypt.MinCost,
			)

			// Create an invalid memo (empty text)
			memo := &domain.Memo{
				ID:        uuid.New(),
				UserID:    userID,
				Text:      "", // Invalid: empty text
				Status:    domain.MemoStatusPending,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			// Call the Create method
			err := memoStore.Create(ctx, memo)

			// Verify the result
			assert.Error(t, err, "Creating memo with empty text should fail")
			assert.Equal(t, domain.ErrMemoTextEmpty, err, "Error should be ErrMemoTextEmpty")

			// Verify no memo was created
			count := testutils.CountMemos(ctx, t, tx, "id = $1", memo.ID)
			assert.Equal(t, 0, count, "No memo should be created with invalid data")
		})

		// Test Case 3: Create memo with non-existent user ID
		t.Run("Non-existent user ID", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create a memo with a non-existent user ID
			nonExistentUserID := uuid.New() // Random UUID that doesn't exist
			memo := testutils.CreateTestMemo(t, nonExistentUserID)

			// Call the Create method
			err := memoStore.Create(ctx, memo)

			// Verify the result
			assert.Error(t, err, "Creating memo with non-existent user ID should fail")
			assert.True(t, errors.Is(err, store.ErrInvalidEntity),
				"Error should wrap ErrInvalidEntity")

			// Verify no memo was created
			count := testutils.CountMemos(ctx, t, tx, "id = $1", memo.ID)
			assert.Equal(t, 0, count, "No memo should be created with non-existent user ID")
		})
	})
}

// TestPostgresMemoStore_GetByID tests the GetByID method
func TestPostgresMemoStore_GetByID(t *testing.T) {
	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create a new memo store
		memoStore := postgres.NewPostgresMemoStore(tx, nil)

		// Test Case 1: Successfully get a memo by ID
		t.Run("Successfully get memo by ID", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Insert a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"get-memo-test@example.com",
				bcrypt.MinCost,
			)

			// Insert a test memo
			memo := testutils.MustInsertMemo(ctx, t, tx, userID)

			// Call the GetByID method
			retrievedMemo, err := memoStore.GetByID(ctx, memo.ID)

			// Verify the result
			require.NoError(t, err, "Getting memo by ID should succeed")
			require.NotNil(t, retrievedMemo, "Retrieved memo should not be nil")

			// Verify memo fields
			assert.Equal(t, memo.ID, retrievedMemo.ID, "Memo ID should match")
			assert.Equal(t, memo.UserID, retrievedMemo.UserID, "User ID should match")
			assert.Equal(t, memo.Text, retrievedMemo.Text, "Text should match")
			assert.Equal(t, memo.Status, retrievedMemo.Status, "Status should match")
		})

		// Test Case 2: Try to get a non-existent memo
		t.Run("Non-existent memo", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create a random UUID that doesn't exist
			nonExistentID := uuid.New()

			// Call the GetByID method
			retrievedMemo, err := memoStore.GetByID(ctx, nonExistentID)

			// Verify the result
			assert.Error(t, err, "Getting non-existent memo should fail")
			assert.Equal(t, store.ErrMemoNotFound, err, "Error should be ErrMemoNotFound")
			assert.Nil(t, retrievedMemo, "Retrieved memo should be nil")
		})
	})
}

// TestPostgresMemoStore_UpdateStatus tests the UpdateStatus method
func TestPostgresMemoStore_UpdateStatus(t *testing.T) {
	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create a new memo store
		memoStore := postgres.NewPostgresMemoStore(tx, nil)

		// Test Case 1: Successfully update memo status
		t.Run("Successfully update memo status", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Insert a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"status-update-test@example.com",
				bcrypt.MinCost,
			)

			// Insert a test memo
			memo := testutils.MustInsertMemo(ctx, t, tx, userID)

			// Verify initial status
			assert.Equal(
				t,
				domain.MemoStatusPending,
				memo.Status,
				"Initial status should be 'pending'",
			)

			// Update status to 'processing'
			err := memoStore.UpdateStatus(ctx, memo.ID, domain.MemoStatusProcessing)
			require.NoError(t, err, "Updating memo status should succeed")

			// Verify the status was updated in the database
			var statusStr string
			err = tx.QueryRowContext(ctx, "SELECT status FROM memos WHERE id = $1", memo.ID).
				Scan(&statusStr)
			require.NoError(t, err, "Should be able to query memo status")

			assert.Equal(t, string(domain.MemoStatusProcessing), statusStr,
				"Status should be updated to 'processing'")
		})

		// Test Case 2: Update with invalid status
		t.Run("Invalid status", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Insert a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"invalid-status-test@example.com",
				bcrypt.MinCost,
			)

			// Insert a test memo
			memo := testutils.MustInsertMemo(ctx, t, tx, userID)

			// Try to update with invalid status
			invalidStatus := domain.MemoStatus("invalid_status")
			err := memoStore.UpdateStatus(ctx, memo.ID, invalidStatus)

			// Verify the result
			assert.Error(t, err, "Updating memo with invalid status should fail")
			assert.Equal(
				t,
				domain.ErrInvalidMemoStatus,
				err,
				"Error should be ErrInvalidMemoStatus",
			)

			// Verify the status was not updated
			var statusStr string
			err = tx.QueryRowContext(ctx, "SELECT status FROM memos WHERE id = $1", memo.ID).
				Scan(&statusStr)
			require.NoError(t, err, "Should be able to query memo status")

			assert.Equal(t, string(domain.MemoStatusPending), statusStr,
				"Status should remain 'pending'")
		})

		// Test Case 3: Update non-existent memo
		t.Run("Non-existent memo", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create a random UUID that doesn't exist
			nonExistentID := uuid.New()

			// Try to update non-existent memo
			err := memoStore.UpdateStatus(ctx, nonExistentID, domain.MemoStatusProcessing)

			// Verify the result
			assert.Error(t, err, "Updating non-existent memo should fail")
			assert.Equal(t, store.ErrMemoNotFound, err, "Error should be ErrMemoNotFound")
		})
	})
}

// TestPostgresMemoStore_FindMemosByStatus tests the FindMemosByStatus method
func TestPostgresMemoStore_FindMemosByStatus(t *testing.T) {
	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create a new memo store
		memoStore := postgres.NewPostgresMemoStore(tx, nil)

		// Test Case 1: Find memos with specific status
		t.Run("Find memos with specific status", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Insert a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"find-status-test@example.com",
				bcrypt.MinCost,
			)

			// Insert multiple memos with different statuses
			memo1 := testutils.MustInsertMemo(ctx, t, tx, userID)
			require.Equal(t, domain.MemoStatusPending, memo1.Status)

			memo2 := testutils.MustInsertMemo(ctx, t, tx, userID)
			err := memoStore.UpdateStatus(ctx, memo2.ID, domain.MemoStatusProcessing)
			require.NoError(t, err)

			memo3 := testutils.MustInsertMemo(ctx, t, tx, userID)
			require.Equal(t, domain.MemoStatusPending, memo3.Status)

			// Call FindMemosByStatus for 'pending' status
			pendingMemos, err := memoStore.FindMemosByStatus(ctx, domain.MemoStatusPending, 10, 0)

			// Verify the result
			require.NoError(t, err, "Finding memos by status should succeed")
			assert.Equal(t, 2, len(pendingMemos), "Should find 2 pending memos")

			// Call FindMemosByStatus for 'processing' status
			processingMemos, err := memoStore.FindMemosByStatus(
				ctx,
				domain.MemoStatusProcessing,
				10,
				0,
			)

			// Verify the result
			require.NoError(t, err, "Finding memos by status should succeed")
			assert.Equal(t, 1, len(processingMemos), "Should find 1 processing memo")
			assert.Equal(
				t,
				memo2.ID,
				processingMemos[0].ID,
				"Should find the correct processing memo",
			)
		})

		// Test Case 2: Pagination with limit and offset
		t.Run("Pagination with limit and offset", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Insert a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"pagination-test@example.com",
				bcrypt.MinCost,
			)

			// Insert multiple memos with the same status
			for i := 0; i < 5; i++ {
				testutils.MustInsertMemo(ctx, t, tx, userID)
			}

			// Test with limit 2, offset 0
			memos1, err := memoStore.FindMemosByStatus(ctx, domain.MemoStatusPending, 2, 0)
			require.NoError(t, err, "Finding memos with limit and offset should succeed")
			assert.Equal(t, 2, len(memos1), "Should find 2 memos with limit 2")

			// Test with limit 2, offset 2
			memos2, err := memoStore.FindMemosByStatus(ctx, domain.MemoStatusPending, 2, 2)
			require.NoError(t, err, "Finding memos with limit and offset should succeed")
			assert.Equal(t, 2, len(memos2), "Should find 2 memos with limit 2, offset 2")

			// Verify different memos are returned
			assert.NotEqual(
				t,
				memos1[0].ID,
				memos2[0].ID,
				"Should return different memos for different offsets",
			)
			assert.NotEqual(
				t,
				memos1[1].ID,
				memos2[1].ID,
				"Should return different memos for different offsets",
			)
		})

		// Test Case 3: Empty result
		t.Run("Empty result", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Find memos with a status that no memos have
			memos, err := memoStore.FindMemosByStatus(ctx, domain.MemoStatusFailed, 10, 0)

			// Verify the result
			require.NoError(t, err, "Finding memos with no matches should succeed")
			assert.NotNil(t, memos, "Result should not be nil")
			assert.Equal(t, 0, len(memos), "Should find 0 memos")
			assert.IsType(t, []*domain.Memo{}, memos, "Should return empty slice, not nil")
		})

		// Test Case 4: Invalid pagination parameters
		t.Run("Invalid pagination parameters", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Insert a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"invalid-pagination-test@example.com",
				bcrypt.MinCost,
			)

			// Insert a test memo
			testutils.MustInsertMemo(ctx, t, tx, userID)

			// Test with negative limit
			memos1, err := memoStore.FindMemosByStatus(ctx, domain.MemoStatusPending, -5, 0)
			require.NoError(t, err, "Finding memos with negative limit should use default")
			assert.GreaterOrEqual(t, len(memos1), 1, "Should use default limit instead of negative")

			// Test with negative offset
			memos2, err := memoStore.FindMemosByStatus(ctx, domain.MemoStatusPending, 10, -5)
			require.NoError(t, err, "Finding memos with negative offset should use default")
			assert.GreaterOrEqual(
				t,
				len(memos2),
				1,
				"Should use default offset of 0 instead of negative",
			)
		})
	})
}

// TestPostgresMemoStore_Update tests the Update method
func TestPostgresMemoStore_Update(t *testing.T) {
	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create a new memo store
		memoStore := postgres.NewPostgresMemoStore(tx, nil)

		// Test Case 1: Successfully update memo
		t.Run("Successfully update memo", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Insert a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"update-memo-test@example.com",
				bcrypt.MinCost,
			)

			// Insert a test memo
			memo := testutils.MustInsertMemo(ctx, t, tx, userID)

			// Update memo
			newText := "Updated memo text " + uuid.New().String()
			updatedTime := time.Now().UTC().Add(time.Hour) // simulate some time passing

			memo.Text = newText
			memo.Status = domain.MemoStatusProcessing
			memo.UpdatedAt = updatedTime

			// Call the Update method
			err := memoStore.Update(ctx, memo)

			// Verify the result
			require.NoError(t, err, "Updating memo should succeed")

			// Verify the memo was updated in the database
			var dbMemo domain.Memo
			var statusStr string

			err = tx.QueryRowContext(ctx, `
				SELECT id, user_id, text, status, created_at, updated_at
				FROM memos
				WHERE id = $1
			`, memo.ID).Scan(
				&dbMemo.ID,
				&dbMemo.UserID,
				&dbMemo.Text,
				&statusStr,
				&dbMemo.CreatedAt,
				&dbMemo.UpdatedAt,
			)

			require.NoError(t, err, "Should be able to retrieve updated memo")
			dbMemo.Status = domain.MemoStatus(statusStr)

			assert.Equal(t, memo.ID, dbMemo.ID, "Memo ID should match")
			assert.Equal(t, memo.UserID, dbMemo.UserID, "User ID should match")
			assert.Equal(t, newText, dbMemo.Text, "Text should be updated")
			assert.Equal(t, domain.MemoStatusProcessing, dbMemo.Status, "Status should be updated")
			assert.WithinDuration(t, updatedTime, dbMemo.UpdatedAt, time.Second,
				"UpdatedAt should be updated")
		})

		// Test Case 2: Update with invalid data
		t.Run("Invalid memo data", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Insert a test user
			userID := testutils.MustInsertUser(
				ctx,
				t,
				tx,
				"invalid-update-test@example.com",
				bcrypt.MinCost,
			)

			// Insert a test memo
			memo := testutils.MustInsertMemo(ctx, t, tx, userID)

			// Try to update with invalid data
			originalText := memo.Text
			memo.Text = "" // Invalid: empty text

			// Call the Update method
			err := memoStore.Update(ctx, memo)

			// Verify the result
			assert.Error(t, err, "Updating memo with empty text should fail")
			assert.Equal(t, domain.ErrMemoTextEmpty, err, "Error should be ErrMemoTextEmpty")

			// Verify the memo was not updated
			var dbText string
			err = tx.QueryRowContext(ctx, "SELECT text FROM memos WHERE id = $1", memo.ID).
				Scan(&dbText)
			require.NoError(t, err, "Should be able to query memo text")

			assert.Equal(t, originalText, dbText, "Text should not be updated")
		})

		// Test Case 3: Update non-existent memo
		t.Run("Non-existent memo", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create a valid memo with non-existent ID
			nonExistentID := uuid.New()
			memo := &domain.Memo{
				ID:        nonExistentID,
				UserID:    uuid.New(), // Any user ID will do for this test
				Text:      "This memo doesn't exist",
				Status:    domain.MemoStatusPending,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			// Try to update non-existent memo
			err := memoStore.Update(ctx, memo)

			// Verify the result
			assert.Error(t, err, "Updating non-existent memo should fail")
			assert.Equal(t, store.ErrMemoNotFound, err, "Error should be ErrMemoNotFound")
		})
	})
}
