//go:build integration

package postgres_test

import (
	"context"
	"encoding/json"
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

// TestGetNextReviewCardExtended provides additional test cases for the GetNextReviewCard method
// beyond what's already covered in card_store_test.go
func TestGetNextReviewCardExtended(t *testing.T) {
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test")
	}

	// We need to run real database tests for proper testing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	// Create a transaction for isolation
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback() // Intentionally ignoring error as it's cleanup code
	}()

	// Set up the stores
	userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for test speed
	cardStore := postgres.NewPostgresCardStore(tx, nil)
	memoStore := postgres.NewPostgresMemoStore(tx, nil)
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)

	// Create a test user
	testUser, err := domain.NewUser("getnextreview-extended@example.com", "password123")
	require.NoError(t, err, "Failed to create test user")
	require.NoError(t, userStore.Create(ctx, testUser), "Failed to save test user")

	// Create a test memo
	testMemo, err := domain.NewMemo(testUser.ID, "Extended GetNextReviewCard test memo")
	require.NoError(t, err, "Failed to create test memo")
	require.NoError(t, memoStore.Create(ctx, testMemo), "Failed to save test memo")

	// Helper function to create a card with stats
	createCardWithStats := func(userID, memoID uuid.UUID, nextReviewAt time.Time) (*domain.Card, *domain.UserCardStats, error) {
		content := json.RawMessage(`{"front":"Test front","back":"Test back"}`)
		card, err := domain.NewCard(userID, memoID, content)
		if err != nil {
			return nil, nil, err
		}

		err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
		if err != nil {
			return nil, nil, err
		}

		// Create stats for the card with specified next review time
		stats, err := domain.NewUserCardStats(userID, card.ID)
		if err != nil {
			return nil, nil, err
		}
		stats.NextReviewAt = nextReviewAt

		err = statsStore.Create(ctx, stats)
		if err != nil {
			return nil, nil, err
		}

		return card, stats, nil
	}

	t.Run("exact_same_review_time", func(t *testing.T) {
		// Create multiple cards with exactly the same review time
		// According to the SQL query, ordering should be by next_review_at ASC, then card ID ASC
		// This test verifies the deterministic ordering by card ID when timestamps match
		sameReviewTime := time.Now().UTC().Add(-1 * time.Hour)

		var createdCards []*domain.Card
		for i := 0; i < 3; i++ {
			card, _, err := createCardWithStats(testUser.ID, testMemo.ID, sameReviewTime)
			require.NoError(t, err, "Failed to create card with stats")
			createdCards = append(createdCards, card)
		}

		// Sort the created cards by ID to determine which one should be returned first
		// This simulates the ORDER BY c.id ASC in the query
		var expectedCard *domain.Card
		lowestID := uuid.Nil
		for _, card := range createdCards {
			if lowestID == uuid.Nil || card.ID.String() < lowestID.String() {
				lowestID = card.ID
				expectedCard = card
			}
		}
		require.NotNil(t, expectedCard, "Failed to identify card with lowest ID")

		// Call GetNextReviewCard
		card, err := cardStore.GetNextReviewCard(ctx, testUser.ID)
		assert.NoError(t, err, "GetNextReviewCard should succeed")
		assert.Equal(
			t,
			expectedCard.ID,
			card.ID,
			"Should return the card with lowest ID when review times are identical",
		)
	})

	t.Run("stats_without_matching_card", func(t *testing.T) {
		// Create a non-existent card ID
		nonExistentCardID := uuid.New()

		// Create stats for non-existent card (simulating orphaned stats)
		stats, err := domain.NewUserCardStats(testUser.ID, nonExistentCardID)
		require.NoError(t, err, "Failed to create stats")

		// Set review time to past so it would be due
		stats.NextReviewAt = time.Now().UTC().Add(-2 * time.Hour)

		err = statsStore.Create(ctx, stats)
		require.NoError(t, err, "Failed to create orphaned stats")

		// Call GetNextReviewCard - should not return the orphaned stats
		// due to inner join with cards table
		card, err := cardStore.GetNextReviewCard(ctx, testUser.ID)
		assert.NoError(
			t,
			err,
			"GetNextReviewCard should succeed with valid cards, ignoring orphaned stats",
		)

		// Should return one of the previously created cards, not the orphaned one
		assert.NotEqual(t, nonExistentCardID, card.ID, "Should not return the orphaned stats")
	})

	t.Run("limit_works_correctly", func(t *testing.T) {
		// Create many cards with different review times to test the LIMIT 1 clause
		now := time.Now().UTC()

		// Create 10 cards with different review times
		for i := 0; i < 10; i++ {
			reviewTime := now.Add(
				time.Duration(-i-10) * time.Hour,
			) // All in the past but with different times
			_, _, err := createCardWithStats(testUser.ID, testMemo.ID, reviewTime)
			require.NoError(t, err, "Failed to create card with stats")
		}

		// Oldest review time should be returned (i=9 would be the oldest)
		oldestReviewTime := now.Add(-19 * time.Hour)

		// Create card with the oldest review time but don't save for assertion
		oldestCard, _, err := createCardWithStats(testUser.ID, testMemo.ID, oldestReviewTime)
		require.NoError(t, err, "Failed to create oldest card")

		// Get next card, which should be the one with the oldest review time
		card, err := cardStore.GetNextReviewCard(ctx, testUser.ID)
		assert.NoError(t, err, "GetNextReviewCard should succeed")
		assert.Equal(
			t,
			oldestCard.ID,
			card.ID,
			"Should return the card with the oldest review time",
		)
	})

	t.Run("handles_nil_user_id", func(t *testing.T) {
		// Call with zero UUID
		_, err := cardStore.GetNextReviewCard(ctx, uuid.Nil)

		// Should not panic and should return ErrCardNotFound
		// since no cards would match a zero UUID
		assert.Error(t, err, "GetNextReviewCard should return an error for nil UUID")
		assert.ErrorIs(t, err, store.ErrCardNotFound, "Error should be ErrCardNotFound")
	})

	t.Run("no_stats_for_card", func(t *testing.T) {
		// Create a card without associated stats
		content := json.RawMessage(`{"front":"Card without stats","back":"Test back"}`)
		card, err := domain.NewCard(testUser.ID, testMemo.ID, content)
		require.NoError(t, err, "Failed to create test card")

		err = cardStore.CreateMultiple(ctx, []*domain.Card{card})
		require.NoError(t, err, "Failed to insert card")

		// Don't create stats for this card

		// This card should not be returned by GetNextReviewCard
		// because the JOIN with user_card_stats will filter it out
		gotCard, err := cardStore.GetNextReviewCard(ctx, testUser.ID)
		assert.NoError(t, err, "GetNextReviewCard should succeed with other due cards")
		assert.NotEqual(t, card.ID, gotCard.ID, "Should not return card without stats")
	})

	t.Run("error_mapping", func(t *testing.T) {
		// This is already covered by error_leakage_test.go but adding for completeness
		nonExistentUserID := uuid.New()
		_, err := cardStore.GetNextReviewCard(ctx, nonExistentUserID)
		assert.Error(t, err, "GetNextReviewCard should return error for nonexistent user")
		assert.ErrorIs(t, err, store.ErrCardNotFound, "Error should be mapped to ErrCardNotFound")

		// Ensure no internal error details are leaked
		errString := err.Error()
		sensitiveTerms := []string{
			"psql: ", "sql: ", "pq: ", "SQLSTATE",
		}
		for _, term := range sensitiveTerms {
			assert.NotContains(t, errString, term,
				"Error message should not contain sensitive details")
		}
	})
}
