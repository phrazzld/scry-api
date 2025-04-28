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

// TestUserCardStatsStoreErrorLeakage tests that UserCardStatsStore operations
// do not leak internal database error details in their returned errors.
func TestUserCardStatsStoreErrorLeakage(t *testing.T) {
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
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)

	// Setup: create a user, memo, and card to associate with stats
	user, err := domain.NewUser("stats-test@example.com", "Password123!")
	require.NoError(t, err)
	userStore := postgres.NewPostgresUserStore(tx, 10)
	err = userStore.Create(ctx, user)
	require.NoError(t, err)

	memo, err := domain.NewMemo(user.ID, "Test memo for stats")
	require.NoError(t, err)
	memoStore := postgres.NewPostgresMemoStore(tx, nil)
	err = memoStore.Create(ctx, memo)
	require.NoError(t, err)

	validCardContent := []byte(`{"front": "Question", "back": "Answer"}`)
	card := &domain.Card{
		ID:        uuid.New(),
		UserID:    user.ID,
		MemoID:    memo.ID,
		Content:   validCardContent,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	cardStore := postgres.NewPostgresCardStore(tx, nil)
	err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
	require.NoError(t, err)

	// Tests for Get operation
	t.Run("Get errors do not leak details", func(t *testing.T) {
		// Test non-existent user ID
		nonExistentUserID := uuid.New()
		_, err := statsStore.Get(ctx, nonExistentUserID, card.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrUserCardStatsNotFound)
		AssertNoErrorLeakage(t, err)

		// Test non-existent card ID
		nonExistentCardID := uuid.New()
		_, err = statsStore.Get(ctx, user.ID, nonExistentCardID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrUserCardStatsNotFound)
		AssertNoErrorLeakage(t, err)
	})

	// Get the stats entry that was automatically created when the card was created
	stats, err := statsStore.Get(ctx, user.ID, card.ID)
	require.NoError(t, err)

	// Tests for Update operation
	t.Run("Update errors do not leak details", func(t *testing.T) {
		// Test non-existent stats entry
		nonExistentStats, err := domain.NewUserCardStats(uuid.New(), uuid.New())
		require.NoError(t, err)
		err = statsStore.Update(ctx, nonExistentStats)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrUserCardStatsNotFound)
		AssertNoErrorLeakage(t, err)

		// Test invalid stats
		invalidStats := &domain.UserCardStats{
			UserID:             stats.UserID,
			CardID:             stats.CardID,
			Interval:           -1, // Invalid interval (negative)
			EaseFactor:         stats.EaseFactor,
			ConsecutiveCorrect: stats.ConsecutiveCorrect,
			LastReviewedAt:     stats.LastReviewedAt,
			NextReviewAt:       stats.NextReviewAt,
			ReviewCount:        stats.ReviewCount,
			CreatedAt:          stats.CreatedAt,
			UpdatedAt:          stats.UpdatedAt,
		}
		err = statsStore.Update(ctx, invalidStats)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrInvalidEntity)
		AssertNoErrorLeakage(t, err)
	})

	// Tests for Delete operation
	t.Run("Delete errors do not leak details", func(t *testing.T) {
		// Test non-existent stats entry
		nonExistentUserID := uuid.New()
		nonExistentCardID := uuid.New()
		err := statsStore.Delete(ctx, nonExistentUserID, nonExistentCardID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrUserCardStatsNotFound)
		AssertNoErrorLeakage(t, err)
	})
}
