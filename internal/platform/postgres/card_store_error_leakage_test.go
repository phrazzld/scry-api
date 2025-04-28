//go:build integration

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

// TestCardStoreErrorLeakage tests that CardStore operations do not leak internal
// database error details in their returned errors.
func TestCardStoreErrorLeakage(t *testing.T) {
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
	cardStore := postgres.NewPostgresCardStore(tx, nil)

	// Setup: create a user and memo to associate with cards
	user, err := domain.NewUser("card-test@example.com", "Password123!")
	require.NoError(t, err)
	userStore := postgres.NewPostgresUserStore(tx, 10)
	err = userStore.Create(ctx, user)
	require.NoError(t, err)

	memo, err := domain.NewMemo(user.ID, "Test memo for cards")
	require.NoError(t, err)
	memoStore := postgres.NewPostgresMemoStore(tx, nil)
	err = memoStore.Create(ctx, memo)
	require.NoError(t, err)

	// Sample valid card content
	validCardContent := []byte(`{
		"front": "What is the capital of France?",
		"back": "Paris"
	}`)

	// Tests for CreateMultiple operation
	t.Run("CreateMultiple errors do not leak details", func(t *testing.T) {
		// Test foreign key violation (non-existent user)
		invalidUserCard := &domain.Card{
			ID:        uuid.New(),
			UserID:    uuid.New(), // Random non-existent user ID
			MemoID:    memo.ID,
			Content:   validCardContent,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		err := cardStore.CreateMultiple(ctx, []*domain.Card{invalidUserCard})
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)

		// Test foreign key violation (non-existent memo)
		invalidMemoCard := &domain.Card{
			ID:        uuid.New(),
			UserID:    user.ID,
			MemoID:    uuid.New(), // Random non-existent memo ID
			Content:   validCardContent,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		err = cardStore.CreateMultiple(ctx, []*domain.Card{invalidMemoCard})
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)

		// Test invalid card
		invalidCard := &domain.Card{
			ID:        uuid.New(),
			UserID:    user.ID,
			MemoID:    memo.ID,
			Content:   []byte(`invalid json`), // Invalid JSON content
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		err = cardStore.CreateMultiple(ctx, []*domain.Card{invalidCard})
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)
	})

	// Create a valid card for subsequent tests
	validCard := &domain.Card{
		ID:        uuid.New(),
		UserID:    user.ID,
		MemoID:    memo.ID,
		Content:   validCardContent,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err = cardStore.CreateMultiple(ctx, []*domain.Card{validCard})
	require.NoError(t, err)

	// Tests for GetByID operation
	t.Run("GetByID errors do not leak details", func(t *testing.T) {
		// Test not found error
		nonExistentID := uuid.New()
		_, err := cardStore.GetByID(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrCardNotFound)
		AssertNoErrorLeakage(t, err)
	})

	// Tests for UpdateContent operation
	t.Run("UpdateContent errors do not leak details", func(t *testing.T) {
		// Test non-existent card
		nonExistentID := uuid.New()
		err := cardStore.UpdateContent(ctx, nonExistentID, validCardContent)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrCardNotFound)
		AssertNoErrorLeakage(t, err)

		// Test invalid content
		invalidContent := []byte(`invalid json`)
		err = cardStore.UpdateContent(ctx, validCard.ID, invalidContent)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)
	})

	// Tests for Delete operation
	t.Run("Delete errors do not leak details", func(t *testing.T) {
		// Test non-existent card
		nonExistentID := uuid.New()
		err := cardStore.Delete(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrCardNotFound)
		AssertNoErrorLeakage(t, err)
	})
}
