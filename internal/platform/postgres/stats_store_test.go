//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Test timeout to prevent long-running tests
const testStatsTimeout = 5 * time.Second

// TestUserCardStatsStoreIntegration runs a complete set of integration tests for the UserCardStatsStore implementation.
// This ensures all methods work as expected with a real database connection.
func TestUserCardStatsStoreIntegration(t *testing.T) {
	// Test methods will handle skipping if needed using testdb.GetTestDBWithT

	// Test all methods
	t.Run("TestPostgresUserCardStatsStore_Get", TestPostgresUserCardStatsStore_Get)
	t.Run(
		"TestPostgresUserCardStatsStore_GetForUpdate",
		TestPostgresUserCardStatsStore_GetForUpdate,
	)
	t.Run("TestPostgresUserCardStatsStore_Update", TestPostgresUserCardStatsStore_Update)
	t.Run("TestPostgresUserCardStatsStore_Delete", TestPostgresUserCardStatsStore_Delete)
}

// TestPostgresUserCardStatsStore_Get tests the Get method
func TestPostgresUserCardStatsStore_Get(t *testing.T) {
	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create necessary stores
		userStore := NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := NewPostgresMemoStore(tx, nil)
		cardStore := NewPostgresCardStore(tx, nil)
		statsStore := NewPostgresUserCardStatsStore(tx, nil)

		// Create a test user
		testUser, err := domain.NewUser("testgetstats@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), testUser),
			"Failed to create test user in DB",
		)

		// Create a test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Test memo for stats tests")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), testMemo),
			"Failed to create test memo in DB",
		)

		// Create a test card
		content := json.RawMessage(`{"front":"Stats test front","back":"Stats test back"}`)
		card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
		require.NoError(t, err, "Failed to create test card")
		require.NoError(
			t,
			cardStore.CreateMultiple(context.Background(), []*domain.Card{card}),
			"Failed to create test card in DB",
		)

		t.Run("existing_stats", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// CardStore.CreateMultiple already created stats, retrieve them
			stats, err := statsStore.Get(ctx, testUser.ID, card.ID)
			assert.NoError(t, err, "Get should find existing stats")
			assert.NotNil(t, stats, "Retrieved stats should not be nil")
			assert.Equal(
				t,
				testUser.ID,
				stats.UserID,
				"Retrieved stats should have correct user ID",
			)
			assert.Equal(t, card.ID, stats.CardID, "Retrieved stats should have correct card ID")
		})

		t.Run("non_existent_stats", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Generate random IDs that don't exist
			nonExistentUserID := uuid.New()
			nonExistentCardID := uuid.New()

			// Try to retrieve non-existent stats
			_, err := statsStore.Get(ctx, nonExistentUserID, card.ID)
			assert.Error(t, err, "Get should return error for non-existent user ID")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
			assert.ErrorContains(
				t,
				err,
				"user card stats",
				"Error should mention 'user card stats'",
			)

			_, err = statsStore.Get(ctx, testUser.ID, nonExistentCardID)
			assert.Error(t, err, "Get should return error for non-existent card ID")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
			assert.ErrorContains(
				t,
				err,
				"user card stats",
				"Error should mention 'user card stats'",
			)
		})
	})
}

// TestPostgresUserCardStatsStore_Update tests the Update method
func TestPostgresUserCardStatsStore_Update(t *testing.T) {
	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create necessary stores
		userStore := NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := NewPostgresMemoStore(tx, nil)
		cardStore := NewPostgresCardStore(tx, nil)
		statsStore := NewPostgresUserCardStatsStore(tx, nil)

		// Create a test user
		testUser, err := domain.NewUser("testupdatestats@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), testUser),
			"Failed to create test user in DB",
		)

		// Create a test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Test memo for stats update tests")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), testMemo),
			"Failed to create test memo in DB",
		)

		// Create a test card
		content := json.RawMessage(
			`{"front":"Stats update test front","back":"Stats update test back"}`,
		)
		card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
		require.NoError(t, err, "Failed to create test card")
		require.NoError(
			t,
			cardStore.CreateMultiple(context.Background(), []*domain.Card{card}),
			"Failed to create test card in DB",
		)

		t.Run("successful_update", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Get current stats
			statsOriginal, err := statsStore.Get(ctx, testUser.ID, card.ID)
			require.NoError(t, err, "Get should find existing stats")

			// Modify stats
			statsUpdate := *statsOriginal
			statsUpdate.Interval = 2
			statsUpdate.EaseFactor = 2.2
			statsUpdate.ConsecutiveCorrect = 3
			statsUpdate.ReviewCount = 5
			statsUpdate.LastReviewedAt = time.Now().
				UTC().
				Truncate(time.Second)
				// Truncate to avoid microsecond differences
			statsUpdate.NextReviewAt = time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)

			// Update stats
			err = statsStore.Update(ctx, &statsUpdate)
			assert.NoError(t, err, "Update should succeed with valid stats")

			// Retrieve updated stats
			statsRetrieved, err := statsStore.Get(ctx, testUser.ID, card.ID)
			assert.NoError(t, err, "Get should find updated stats")
			assert.Equal(
				t,
				statsUpdate.Interval,
				statsRetrieved.Interval,
				"Retrieved stats should have updated interval",
			)
			assert.Equal(
				t,
				statsUpdate.EaseFactor,
				statsRetrieved.EaseFactor,
				"Retrieved stats should have updated ease factor",
			)
			assert.Equal(
				t,
				statsUpdate.ConsecutiveCorrect,
				statsRetrieved.ConsecutiveCorrect,
				"Retrieved stats should have updated consecutive correct",
			)
			assert.Equal(
				t,
				statsUpdate.ReviewCount,
				statsRetrieved.ReviewCount,
				"Retrieved stats should have updated review count",
			)
			assert.WithinDuration(
				t,
				statsUpdate.LastReviewedAt,
				statsRetrieved.LastReviewedAt,
				time.Second,
				"Retrieved stats should have updated last reviewed at",
			)
			assert.WithinDuration(
				t,
				statsUpdate.NextReviewAt,
				statsRetrieved.NextReviewAt,
				time.Second,
				"Retrieved stats should have updated next review at",
			)
		})

		t.Run("non_existent_stats", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Create stats object with non-existent IDs
			nonExistentUserID := uuid.New()
			nonExistentCardID := uuid.New()
			now := time.Now().UTC()

			statsNonexistentUser := &domain.UserCardStats{
				UserID:             nonExistentUserID,
				CardID:             card.ID,
				Interval:           1,
				EaseFactor:         2.5,
				ConsecutiveCorrect: 0,
				LastReviewedAt:     time.Time{},
				NextReviewAt:       now,
				ReviewCount:        0,
				CreatedAt:          now,
				UpdatedAt:          now,
			}

			statsNonexistentCard := &domain.UserCardStats{
				UserID:             testUser.ID,
				CardID:             nonExistentCardID,
				Interval:           1,
				EaseFactor:         2.5,
				ConsecutiveCorrect: 0,
				LastReviewedAt:     time.Time{},
				NextReviewAt:       now,
				ReviewCount:        0,
				CreatedAt:          now,
				UpdatedAt:          now,
			}

			// Try to update non-existent stats
			err := statsStore.Update(ctx, statsNonexistentUser)
			assert.Error(t, err, "Update should return error for non-existent user ID")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
			assert.ErrorContains(
				t,
				err,
				"user card stats",
				"Error should mention 'user card stats'",
			)

			err = statsStore.Update(ctx, statsNonexistentCard)
			assert.Error(t, err, "Update should return error for non-existent card ID")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
			assert.ErrorContains(
				t,
				err,
				"user card stats",
				"Error should mention 'user card stats'",
			)
		})

		t.Run("invalid_stats", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Get current stats
			statsOriginal, err := statsStore.Get(ctx, testUser.ID, card.ID)
			require.NoError(t, err, "Get should find existing stats")

			// Create invalid stats with negative interval
			statsInvalid := *statsOriginal
			statsInvalid.Interval = -1

			// Try to update with invalid stats
			err = statsStore.Update(ctx, &statsInvalid)
			assert.Error(t, err, "Update should return error for invalid stats")
			assert.ErrorIs(t, err, store.ErrInvalidEntity, "Error should be ErrInvalidEntity")
			assert.ErrorContains(
				t,
				err,
				"invalid interval",
				"Error should contain domain validation error",
			)

			// Create invalid stats with invalid ease factor
			statsInvalid = *statsOriginal
			statsInvalid.EaseFactor = 0.5

			// Try to update with invalid stats
			err = statsStore.Update(ctx, &statsInvalid)
			assert.Error(t, err, "Update should return error for invalid stats")
			assert.ErrorIs(t, err, store.ErrInvalidEntity, "Error should be ErrInvalidEntity")
			assert.ErrorContains(
				t,
				err,
				"invalid ease factor",
				"Error should contain domain validation error",
			)
		})
	})
}

// TestPostgresUserCardStatsStore_GetForUpdate tests the GetForUpdate method
func TestPostgresUserCardStatsStore_GetForUpdate(t *testing.T) {
	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create necessary stores
		userStore := NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := NewPostgresMemoStore(tx, nil)
		cardStore := NewPostgresCardStore(tx, nil)
		statsStore := NewPostgresUserCardStatsStore(tx, nil)

		// Create a test user
		testUser, err := domain.NewUser("testgetforupdatestats@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), testUser),
			"Failed to create test user in DB",
		)

		// Create a test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Test memo for stats GetForUpdate tests")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), testMemo),
			"Failed to create test memo in DB",
		)

		// Create a test card
		content := json.RawMessage(
			`{"front":"Stats GetForUpdate test front","back":"Stats GetForUpdate test back"}`,
		)
		card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
		require.NoError(t, err, "Failed to create test card")
		require.NoError(
			t,
			cardStore.CreateMultiple(context.Background(), []*domain.Card{card}),
			"Failed to create test card in DB",
		)

		t.Run("successful_get_for_update", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Get stats with lock
			stats, err := statsStore.GetForUpdate(ctx, testUser.ID, card.ID)
			assert.NoError(t, err, "GetForUpdate should find existing stats with lock")
			assert.NotNil(t, stats, "Retrieved stats should not be nil")
			assert.Equal(
				t,
				testUser.ID,
				stats.UserID,
				"Retrieved stats should have correct user ID",
			)
			assert.Equal(t, card.ID, stats.CardID, "Retrieved stats should have correct card ID")
		})

		t.Run("non_existent_stats", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Generate random IDs that don't exist
			nonExistentUserID := uuid.New()
			nonExistentCardID := uuid.New()

			// Try to retrieve non-existent stats with lock
			_, err := statsStore.GetForUpdate(ctx, nonExistentUserID, card.ID)
			assert.Error(t, err, "GetForUpdate should return error for non-existent user ID")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
			assert.ErrorContains(
				t,
				err,
				"user card stats",
				"Error should mention 'user card stats'",
			)

			_, err = statsStore.GetForUpdate(ctx, testUser.ID, nonExistentCardID)
			assert.Error(t, err, "GetForUpdate should return error for non-existent card ID")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
			assert.ErrorContains(
				t,
				err,
				"user card stats",
				"Error should mention 'user card stats'",
			)
		})

		t.Run("update_after_get_for_update", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Get stats with lock
			stats, err := statsStore.GetForUpdate(ctx, testUser.ID, card.ID)
			assert.NoError(t, err, "GetForUpdate should find existing stats with lock")

			// Modify and update stats
			stats.Interval = 3
			stats.EaseFactor = 2.3
			stats.ConsecutiveCorrect = 4
			stats.ReviewCount = 6
			stats.LastReviewedAt = time.Now().UTC().Truncate(time.Second)
			stats.NextReviewAt = time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second)

			// Update stats
			err = statsStore.Update(ctx, stats)
			assert.NoError(t, err, "Update should succeed after GetForUpdate")

			// Verify update was successful
			updatedStats, err := statsStore.Get(ctx, testUser.ID, card.ID)
			assert.NoError(t, err, "Get should find updated stats")
			assert.Equal(
				t,
				stats.Interval,
				updatedStats.Interval,
				"Stats should have updated interval",
			)
			assert.Equal(
				t,
				stats.EaseFactor,
				updatedStats.EaseFactor,
				"Stats should have updated ease factor",
			)
			assert.Equal(
				t,
				stats.ConsecutiveCorrect,
				updatedStats.ConsecutiveCorrect,
				"Stats should have updated consecutive correct",
			)
			assert.Equal(
				t,
				stats.ReviewCount,
				updatedStats.ReviewCount,
				"Stats should have updated review count",
			)
			assert.WithinDuration(
				t,
				stats.LastReviewedAt,
				updatedStats.LastReviewedAt,
				time.Second,
				"Stats should have updated last reviewed at",
			)
			assert.WithinDuration(
				t,
				stats.NextReviewAt,
				updatedStats.NextReviewAt,
				time.Second,
				"Stats should have updated next review at",
			)
		})
	})
}

// TestPostgresUserCardStatsStore_Delete tests the Delete method
func TestPostgresUserCardStatsStore_Delete(t *testing.T) {
	t.Parallel() // Enable parallel testing

	// Get a database connection
	db := testdb.GetTestDBWithT(t)

	// Run the test within a transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create necessary stores
		userStore := NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := NewPostgresMemoStore(tx, nil)
		cardStore := NewPostgresCardStore(tx, nil)
		statsStore := NewPostgresUserCardStatsStore(tx, nil)

		// Create a test user
		testUser, err := domain.NewUser("testdeletestats@example.com", "password123")
		require.NoError(t, err, "Failed to create test user")
		require.NoError(
			t,
			userStore.Create(context.Background(), testUser),
			"Failed to create test user in DB",
		)

		// Create a test memo
		testMemo, err := domain.NewMemo(testUser.ID, "Test memo for stats delete tests")
		require.NoError(t, err, "Failed to create test memo")
		require.NoError(
			t,
			memoStore.Create(context.Background(), testMemo),
			"Failed to create test memo in DB",
		)

		t.Run("successful_delete", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Create a test card - this will also create stats
			content := json.RawMessage(
				`{"front":"Stats delete test front","back":"Stats delete test back"}`,
			)
			card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
			require.NoError(t, err, "Failed to create test card")
			require.NoError(
				t,
				cardStore.CreateMultiple(ctx, []*domain.Card{card}),
				"Failed to create test card in DB",
			)

			// Verify stats exist
			_, err = statsStore.Get(ctx, testUser.ID, card.ID)
			assert.NoError(t, err, "Stats should exist before deletion")

			// Delete stats
			err = statsStore.Delete(ctx, testUser.ID, card.ID)
			assert.NoError(t, err, "Delete should succeed for existing stats")

			// Verify stats no longer exist
			_, err = statsStore.Get(ctx, testUser.ID, card.ID)
			assert.Error(t, err, "Stats should not exist after deletion")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
			assert.ErrorContains(
				t,
				err,
				"user card stats",
				"Error should mention 'user card stats'",
			)
		})

		t.Run("non_existent_stats", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Generate random IDs that don't exist
			nonExistentUserID := uuid.New()
			nonExistentCardID := uuid.New()

			// Try to delete non-existent stats
			err := statsStore.Delete(ctx, nonExistentUserID, nonExistentCardID)
			assert.Error(t, err, "Delete should return error for non-existent stats")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
			assert.ErrorContains(
				t,
				err,
				"user card stats",
				"Error should mention 'user card stats'",
			)
		})

		t.Run("cascade_delete_through_card", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testStatsTimeout)
			defer cancel()

			// Create a test card - this will also create stats
			content := json.RawMessage(
				`{"front":"Stats cascade test front","back":"Stats cascade test back"}`,
			)
			card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
			require.NoError(t, err, "Failed to create test card")
			require.NoError(
				t,
				cardStore.CreateMultiple(ctx, []*domain.Card{card}),
				"Failed to create test card in DB",
			)

			// Verify stats exist
			_, err = statsStore.Get(ctx, testUser.ID, card.ID)
			assert.NoError(t, err, "Stats should exist before card deletion")

			// Delete the card - this should cascade delete the stats
			err = cardStore.Delete(ctx, card.ID)
			assert.NoError(t, err, "Card delete should succeed")

			// Verify stats no longer exist
			_, err = statsStore.Get(ctx, testUser.ID, card.ID)
			assert.Error(t, err, "Stats should not exist after card deletion")
			assert.ErrorIs(
				t,
				err,
				store.ErrUserCardStatsNotFound,
				"Error should be ErrUserCardStatsNotFound",
			)
			assert.ErrorContains(
				t,
				err,
				"user card stats",
				"Error should mention 'user card stats'",
			)
		})
	})
}
