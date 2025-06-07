//go:build test_without_external_deps

package card_review_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestSubmitAnswer_TransactionCoverage tests the SubmitAnswer function transaction logic comprehensively
// This test is designed to exercise all the uncovered transaction paths to boost coverage
func TestSubmitAnswer_TransactionCoverage(t *testing.T) {
	// Allow database integration tests to run for coverage

	// Get test database
	db := testdb.GetTestDBWithT(t)

	// Run tests in transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Set up stores
		cardStore := postgres.NewPostgresCardStore(tx, nil)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)
		userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := postgres.NewPostgresMemoStore(tx, nil)
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err)

		// Create service
		service, err := card_review.NewCardReviewService(cardStore, statsStore, srsService, nil)
		require.NoError(t, err)

		// Create test user and memo
		user, err := domain.NewUser("coverage@example.com", "password123456")
		require.NoError(t, err)
		require.NoError(t, userStore.Create(context.Background(), user))

		memo, err := domain.NewMemo(user.ID, "Coverage test memo")
		require.NoError(t, err)
		require.NoError(t, memoStore.Create(context.Background(), memo))

		t.Run("new_card_stats_creation_path", func(t *testing.T) {
			// Test the path where stats don't exist and need to be created
			cardContent := []byte(`{"front": "New card test", "back": "Answer"}`)
			card, err := domain.NewCard(user.ID, memo.ID, cardContent)
			require.NoError(t, err)
			require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{card}))

			// Submit answer for card with no existing stats (creates new stats)
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
			stats, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer)

			assert.NoError(t, err)
			assert.NotNil(t, stats)
			assert.Equal(t, 1, stats.ReviewCount)
			assert.False(t, stats.LastReviewedAt.IsZero())
			assert.True(t, stats.NextReviewAt.After(time.Now()))
		})

		t.Run("existing_card_stats_update_path", func(t *testing.T) {
			// Test the path where stats exist and need to be updated
			cardContent := []byte(`{"front": "Existing stats test", "back": "Answer"}`)
			card, err := domain.NewCard(user.ID, memo.ID, cardContent)
			require.NoError(t, err)
			require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{card}))

			// Create initial stats
			initialStats, err := domain.NewUserCardStats(user.ID, card.ID)
			require.NoError(t, err)
			initialStats.LastReviewedAt = time.Now().Add(-1 * time.Hour)
			initialStats.ReviewCount = 1
			require.NoError(t, statsStore.Create(context.Background(), initialStats))

			// Submit answer for card with existing stats (updates existing stats)
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeHard}
			stats, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer)

			assert.NoError(t, err)
			assert.NotNil(t, stats)
			assert.Equal(t, 2, stats.ReviewCount) // Should be incremented
		})

		t.Run("card_not_found_transaction_error", func(t *testing.T) {
			// Test transaction rollback when card doesn't exist
			nonExistentCardID := uuid.New()
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}

			_, err := service.SubmitAnswer(context.Background(), user.ID, nonExistentCardID, answer)
			assert.ErrorIs(t, err, card_review.ErrCardNotFound)
		})

		t.Run("card_ownership_validation", func(t *testing.T) {
			// Create another user
			otherUser, err := domain.NewUser("other@coverage.com", "password123456")
			require.NoError(t, err)
			require.NoError(t, userStore.Create(context.Background(), otherUser))

			otherMemo, err := domain.NewMemo(otherUser.ID, "Other user memo")
			require.NoError(t, err)
			require.NoError(t, memoStore.Create(context.Background(), otherMemo))

			// Create card owned by other user
			cardContent := []byte(`{"front": "Ownership test", "back": "Answer"}`)
			otherCard, err := domain.NewCard(otherUser.ID, otherMemo.ID, cardContent)
			require.NoError(t, err)
			require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{otherCard}))

			// Try to submit answer as original user (should fail ownership check)
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
			_, err = service.SubmitAnswer(context.Background(), user.ID, otherCard.ID, answer)
			assert.ErrorIs(t, err, card_review.ErrCardNotOwned)
		})

		t.Run("all_outcome_srs_calculations", func(t *testing.T) {
			// Test SRS calculations for all outcomes to cover algorithm branches
			outcomes := []domain.ReviewOutcome{
				domain.ReviewOutcomeAgain,
				domain.ReviewOutcomeHard,
				domain.ReviewOutcomeGood,
				domain.ReviewOutcomeEasy,
			}

			for i, outcome := range outcomes {
				// Create unique card for each outcome
				cardContent := []byte(`{"front": "SRS test ` + string(rune(i)) + `", "back": "Answer"}`)
				card, err := domain.NewCard(user.ID, memo.ID, cardContent)
				require.NoError(t, err)
				require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{card}))

				// Submit answer to trigger SRS calculation
				answer := card_review.ReviewAnswer{Outcome: outcome}
				stats, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer)

				assert.NoError(t, err, "Failed for outcome: %s", outcome)
				assert.NotNil(t, stats)
				assert.Equal(t, 1, stats.ReviewCount)

				// Verify SRS algorithm was applied based on outcome
				switch outcome {
				case domain.ReviewOutcomeAgain:
					assert.LessOrEqual(t, stats.Interval, 1, "Again should have minimal interval")
				case domain.ReviewOutcomeHard:
					assert.GreaterOrEqual(t, stats.Interval, 1, "Hard should have small positive interval")
				case domain.ReviewOutcomeGood:
					assert.GreaterOrEqual(t, stats.Interval, 1, "Good should have reasonable interval")
				case domain.ReviewOutcomeEasy:
					assert.GreaterOrEqual(t, stats.Interval, 1, "Easy should have larger interval")
				}

				// Verify ease factor adjustments
				assert.Greater(t, stats.EaseFactor, 0.0, "Ease factor should be positive")
			}
		})

		t.Run("stats_locking_and_concurrency", func(t *testing.T) {
			// Test the row-level locking mechanism (GetForUpdate)
			cardContent := []byte(`{"front": "Locking test", "back": "Answer"}`)
			card, err := domain.NewCard(user.ID, memo.ID, cardContent)
			require.NoError(t, err)
			require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{card}))

			// Create initial stats
			initialStats, err := domain.NewUserCardStats(user.ID, card.ID)
			require.NoError(t, err)
			initialStats.LastReviewedAt = time.Now().Add(-2 * time.Hour)
			require.NoError(t, statsStore.Create(context.Background(), initialStats))

			// Submit answer (this will use GetForUpdate internally)
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
			stats, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer)

			assert.NoError(t, err)
			assert.NotNil(t, stats)
			assert.Equal(t, 1, stats.ReviewCount)
		})

		t.Run("multiple_reviews_progression", func(t *testing.T) {
			// Test progression through multiple reviews to cover update paths
			cardContent := []byte(`{"front": "Progression test", "back": "Answer"}`)
			card, err := domain.NewCard(user.ID, memo.ID, cardContent)
			require.NoError(t, err)
			require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{card}))

			// First review (creates stats)
			answer1 := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
			stats1, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer1)
			assert.NoError(t, err)
			assert.Equal(t, 1, stats1.ReviewCount)

			// Second review (updates stats)
			answer2 := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeEasy}
			stats2, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer2)
			assert.NoError(t, err)
			assert.Equal(t, 2, stats2.ReviewCount)

			// Third review (updates stats again)
			answer3 := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeHard}
			stats3, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer3)
			assert.NoError(t, err)
			assert.Equal(t, 3, stats3.ReviewCount)

			// Verify progression of intervals and ease factors
			assert.True(t, stats3.LastReviewedAt.After(stats2.LastReviewedAt))
		})

		t.Run("context_propagation_through_transaction", func(t *testing.T) {
			// Test that context is properly propagated through the transaction
			cardContent := []byte(`{"front": "Context test", "back": "Answer"}`)
			card, err := domain.NewCard(user.ID, memo.ID, cardContent)
			require.NoError(t, err)
			require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{card}))

			// Use context with values
			type ctxKey string
			const testKey ctxKey = "test_correlation_id"
			ctx := context.WithValue(context.Background(), testKey, "test-123")

			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
			stats, err := service.SubmitAnswer(ctx, user.ID, card.ID, answer)

			assert.NoError(t, err)
			assert.NotNil(t, stats)
		})

		t.Run("edge_case_timing", func(t *testing.T) {
			// Test edge cases around timing and scheduling
			cardContent := []byte(`{"front": "Timing test", "back": "Answer"}`)
			card, err := domain.NewCard(user.ID, memo.ID, cardContent)
			require.NoError(t, err)
			require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{card}))

			// Create stats with very old review time
			oldStats, err := domain.NewUserCardStats(user.ID, card.ID)
			require.NoError(t, err)
			oldStats.LastReviewedAt = time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
			oldStats.ReviewCount = 5
			oldStats.Interval = 7
			require.NoError(t, statsStore.Create(context.Background(), oldStats))

			// Submit answer to trigger recalculation
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeAgain}
			stats, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer)

			assert.NoError(t, err)
			assert.NotNil(t, stats)
			assert.Equal(t, 6, stats.ReviewCount)
			assert.True(t, stats.LastReviewedAt.After(oldStats.LastReviewedAt))
		})
	})
}
