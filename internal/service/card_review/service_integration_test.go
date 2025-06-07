//go:build integration

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
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// mockCardRepository wraps a real CardStore but implements the repository interface
// needed for transaction management in service tests
type mockCardRepository struct {
	store.CardStore
	dbConn *sql.DB
}

func (m *mockCardRepository) WithTx(tx *sql.Tx) store.CardStore {
	return &mockCardRepository{
		CardStore: m.CardStore.WithTx(tx),
		dbConn:    m.dbConn,
	}
}

func (m *mockCardRepository) DB() *sql.DB {
	return m.dbConn
}

// mockStatsRepository wraps a real UserCardStatsStore but implements the repository interface
// needed for transaction management in service tests
type mockStatsRepository struct {
	store.UserCardStatsStore
	dbConn *sql.DB
}

func (m *mockStatsRepository) WithTx(tx *sql.Tx) store.UserCardStatsStore {
	return &mockStatsRepository{
		UserCardStatsStore: m.UserCardStatsStore.WithTx(tx),
		dbConn:             m.dbConn,
	}
}

func (m *mockStatsRepository) DB() *sql.DB {
	return m.dbConn
}

// TestSubmitAnswer_IntegrationFlow tests the complete SubmitAnswer workflow with real transactions
func TestSubmitAnswer_IntegrationFlow(t *testing.T) {
	// Get test database
	db := testdb.GetTestDBWithT(t)

	// Run tests in transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Set up stores with transaction for test isolation
		cardStore := postgres.NewPostgresCardStore(tx, nil)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)
		userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := postgres.NewPostgresMemoStore(tx, nil)

		// Create mock repositories that provide DB() access for transaction management
		mockCardRepo := &mockCardRepository{
			CardStore: cardStore,
			dbConn:    db,
		}
		mockStatsRepo := &mockStatsRepository{
			UserCardStatsStore: statsStore,
			dbConn:             db,
		}

		srsService, err := srs.NewDefaultService()
		require.NoError(t, err)

		// Create service with mock repositories that support transaction management
		service, err := card_review.NewCardReviewService(mockCardRepo, mockStatsRepo, srsService, nil)
		require.NoError(t, err)

		// Create test user
		user, err := domain.NewUser("test@example.com", "password123456")
		require.NoError(t, err)
		require.NoError(t, userStore.Create(context.Background(), user))

		// Create test memo
		memo, err := domain.NewMemo(user.ID, "Test memo for integration testing")
		require.NoError(t, err)
		require.NoError(t, memoStore.Create(context.Background(), memo))

		// Create test card
		cardContent := []byte(`{"front": "What is 2+2?", "back": "4"}`)
		card, err := domain.NewCard(user.ID, memo.ID, cardContent)
		require.NoError(t, err)
		require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{card}))

		t.Run("successful_first_review", func(t *testing.T) {
			// Test first review of a new card
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}

			stats, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer)
			assert.NoError(t, err)
			assert.NotNil(t, stats)

			// Verify stats were created correctly
			assert.Equal(t, user.ID, stats.UserID)
			assert.Equal(t, card.ID, stats.CardID)
			assert.Equal(t, 1, stats.ReviewCount)
			assert.True(t, stats.LastReviewedAt.After(time.Time{}))
			assert.True(t, stats.NextReviewAt.After(time.Now()))
		})

		t.Run("successful_subsequent_review", func(t *testing.T) {
			// Test second review of the same card
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeEasy}

			stats, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer)
			assert.NoError(t, err)
			assert.NotNil(t, stats)

			// Verify stats were updated correctly
			assert.Equal(t, 2, stats.ReviewCount) // Should be incremented
			assert.True(t, stats.LastReviewedAt.After(time.Time{}))
		})

		t.Run("card_not_found", func(t *testing.T) {
			// Test with non-existent card
			nonExistentCardID := uuid.New()
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}

			_, err := service.SubmitAnswer(context.Background(), user.ID, nonExistentCardID, answer)
			assert.ErrorIs(t, err, card_review.ErrCardNotFound)
		})

		t.Run("card_not_owned", func(t *testing.T) {
			// Create another user and card
			otherUser, err := domain.NewUser("other@example.com", "password123456")
			require.NoError(t, err)
			require.NoError(t, userStore.Create(context.Background(), otherUser))

			otherMemo, err := domain.NewMemo(otherUser.ID, "Other user's memo")
			require.NoError(t, err)
			require.NoError(t, memoStore.Create(context.Background(), otherMemo))

			otherCard, err := domain.NewCard(otherUser.ID, otherMemo.ID, cardContent)
			require.NoError(t, err)
			require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{otherCard}))

			// Try to review other user's card
			answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
			_, err = service.SubmitAnswer(context.Background(), user.ID, otherCard.ID, answer)
			assert.ErrorIs(t, err, card_review.ErrCardNotOwned)
		})

		t.Run("all_valid_outcomes", func(t *testing.T) {
			// Test all valid review outcomes
			outcomes := []domain.ReviewOutcome{
				domain.ReviewOutcomeAgain,
				domain.ReviewOutcomeHard,
				domain.ReviewOutcomeGood,
				domain.ReviewOutcomeEasy,
			}

			for _, outcome := range outcomes {
				// Create a new card for each outcome test
				testCard, err := domain.NewCard(
					user.ID,
					memo.ID,
					[]byte(`{"front": "Test question", "back": "Test answer"}`),
				)
				require.NoError(t, err)
				require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{testCard}))

				answer := card_review.ReviewAnswer{Outcome: outcome}
				stats, err := service.SubmitAnswer(context.Background(), user.ID, testCard.ID, answer)
				assert.NoError(t, err, "Failed for outcome: %s", outcome)
				assert.NotNil(t, stats, "Stats should not be nil for outcome: %s", outcome)
				assert.Equal(t, 1, stats.ReviewCount, "Review count should be 1 for outcome: %s", outcome)

				// Test different interval calculations based on outcome
				switch outcome {
				case domain.ReviewOutcomeAgain:
					assert.Equal(t, 1, stats.Interval, "Again outcome should have interval 1")
				case domain.ReviewOutcomeHard:
					assert.LessOrEqual(t, stats.Interval, 1, "Hard outcome should have small interval")
				case domain.ReviewOutcomeGood:
					assert.GreaterOrEqual(t, stats.Interval, 1, "Good outcome should have reasonable interval")
				case domain.ReviewOutcomeEasy:
					assert.GreaterOrEqual(t, stats.Interval, 1, "Easy outcome should have larger interval")
				}
			}
		})

		t.Run("invalid_outcome", func(t *testing.T) {
			// Test invalid outcome (should be caught before transaction)
			answer := card_review.ReviewAnswer{Outcome: "invalid"}

			_, err := service.SubmitAnswer(context.Background(), user.ID, card.ID, answer)
			assert.ErrorIs(t, err, card_review.ErrInvalidAnswer)
		})
	})
}

// TestGetNextCard_Integration tests GetNextCard with real database
func TestGetNextCard_Integration(t *testing.T) {
	// Get test database
	db := testdb.GetTestDBWithT(t)

	// Run tests in transaction for isolation
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Set up stores with transaction for test isolation
		cardStore := postgres.NewPostgresCardStore(tx, nil)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, nil)
		userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)
		memoStore := postgres.NewPostgresMemoStore(tx, nil)

		// Create mock repositories that provide DB() access for transaction management
		mockCardRepo := &mockCardRepository{
			CardStore: cardStore,
			dbConn:    db,
		}
		mockStatsRepo := &mockStatsRepository{
			UserCardStatsStore: statsStore,
			dbConn:             db,
		}

		srsService, err := srs.NewDefaultService()
		require.NoError(t, err)

		// Create service with mock repositories that support transaction management
		service, err := card_review.NewCardReviewService(mockCardRepo, mockStatsRepo, srsService, nil)
		require.NoError(t, err)

		// Create test user
		user, err := domain.NewUser("nextcard@example.com", "password123456")
		require.NoError(t, err)
		require.NoError(t, userStore.Create(context.Background(), user))

		t.Run("no_cards_available", func(t *testing.T) {
			// Test when user has no cards
			_, err := service.GetNextCard(context.Background(), user.ID)
			assert.ErrorIs(t, err, card_review.ErrNoCardsDue)
		})

		t.Run("card_available_for_review", func(t *testing.T) {
			// Create memo and card
			memo, err := domain.NewMemo(user.ID, "Test memo for next card")
			require.NoError(t, err)
			require.NoError(t, memoStore.Create(context.Background(), memo))

			cardContent := []byte(`{"front": "Next card test", "back": "Answer"}`)
			card, err := domain.NewCard(user.ID, memo.ID, cardContent)
			require.NoError(t, err)
			require.NoError(t, cardStore.CreateMultiple(context.Background(), []*domain.Card{card}))

			// Create stats with past due date
			stats, err := domain.NewUserCardStats(user.ID, card.ID)
			require.NoError(t, err)
			stats.NextReviewAt = time.Now().Add(-1 * time.Hour) // Past due
			require.NoError(t, statsStore.Create(context.Background(), stats))

			// Should find the card
			foundCard, err := service.GetNextCard(context.Background(), user.ID)
			assert.NoError(t, err)
			assert.NotNil(t, foundCard)
			assert.Equal(t, card.ID, foundCard.ID)
		})

		t.Run("nonexistent_user", func(t *testing.T) {
			// Test with non-existent user
			nonExistentUserID := uuid.New()
			_, err := service.GetNextCard(context.Background(), nonExistentUserID)
			assert.ErrorIs(t, err, card_review.ErrNoCardsDue)
		})
	})
}
