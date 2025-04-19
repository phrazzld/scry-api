package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoStoreErrorLeakage tests that MemoStore operations do not leak internal
// database error details in their returned errors.
func TestMemoStoreErrorLeakage(t *testing.T) {
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test")
	}

	// We need to run real database tests to trigger actual PostgreSQL errors
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create a transaction for isolation
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback() // Intentionally ignoring error as it's cleanup code
	}()

	// Create the store
	memoStore := postgres.NewPostgresMemoStore(tx, nil)

	// Setup: create a user to associate with memos
	user, err := domain.NewUser("memo-test@example.com", "Password123!")
	require.NoError(t, err)
	userStore := postgres.NewPostgresUserStore(tx, 10)
	err = userStore.Create(ctx, user)
	require.NoError(t, err)

	// Tests for Create operation
	t.Run("Create errors do not leak details", func(t *testing.T) {
		// Test foreign key violation (non-existent user)
		memo, err := domain.NewMemo(uuid.New(), "Test memo") // Using a random user ID
		require.NoError(t, err)
		err = memoStore.Create(ctx, memo)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)

		// Test invalid memo
		invalidMemo := &domain.Memo{
			ID:     uuid.New(),
			UserID: user.ID,
			// Missing required text field
			Status:    domain.MemoStatusPending,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		err = memoStore.Create(ctx, invalidMemo)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)
	})

	// Create a valid memo for use in subsequent tests
	validMemo, err := domain.NewMemo(user.ID, "Valid test memo")
	require.NoError(t, err)
	err = memoStore.Create(ctx, validMemo)
	require.NoError(t, err)

	// Tests for GetByID operation
	t.Run("GetByID errors do not leak details", func(t *testing.T) {
		// Test not found error
		nonExistentID := uuid.New()
		_, err := memoStore.GetByID(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrMemoNotFound)
		AssertNoErrorLeakage(t, err)
	})

	// Tests for UpdateStatus operation
	t.Run("UpdateStatus errors do not leak details", func(t *testing.T) {
		// Test non-existent memo
		nonExistentID := uuid.New()
		err := memoStore.UpdateStatus(ctx, nonExistentID, domain.MemoStatusProcessing)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrMemoNotFound)
		AssertNoErrorLeakage(t, err)

		// Test invalid status
		err = memoStore.UpdateStatus(ctx, validMemo.ID, "invalid_status")
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)
	})

	// Tests for Update operation
	t.Run("Update errors do not leak details", func(t *testing.T) {
		// Test non-existent memo ID
		nonExistentMemo, err := domain.NewMemo(user.ID, "Non-existent memo")
		require.NoError(t, err)
		err = memoStore.Update(ctx, nonExistentMemo)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrMemoNotFound)
		AssertNoErrorLeakage(t, err)

		// Test invalid memo
		invalidMemo := &domain.Memo{
			ID:     validMemo.ID,
			UserID: user.ID,
			// Missing required text field
			Status:    domain.MemoStatusPending,
			CreatedAt: validMemo.CreatedAt,
			UpdatedAt: time.Now().UTC(),
		}
		err = memoStore.Update(ctx, invalidMemo)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)
	})
}
