package mocks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/stretchr/testify/assert"
)

func TestMockCardReviewService(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	cardID := uuid.New()
	memoID := uuid.New()
	now := time.Now().UTC()

	// Create sample card and stats for testing
	cardContent := map[string]interface{}{
		"front": "Test Question",
		"back":  "Test Answer",
	}
	contentBytes, _ := json.Marshal(cardContent)

	sampleCard := &domain.Card{
		ID:        cardID,
		UserID:    userID,
		MemoID:    memoID,
		Content:   contentBytes,
		CreatedAt: now,
		UpdatedAt: now,
	}

	sampleStats := &domain.UserCardStats{
		UserID:             userID,
		CardID:             cardID,
		Interval:           1,
		EaseFactor:         2.5,
		ConsecutiveCorrect: 1,
		LastReviewedAt:     now,
		NextReviewAt:       now.Add(24 * time.Hour),
		ReviewCount:        1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	customErr := errors.New("custom error")

	t.Run("Default Values", func(t *testing.T) {
		// Create mock with default values
		mock := NewMockCardReviewService(
			WithNextCard(sampleCard),
			WithUpdatedStats(sampleStats),
			WithError(nil),
		)

		// GetNextCard should return the default card
		card, err := mock.GetNextCard(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, sampleCard, card)
		assert.Equal(t, 1, mock.GetNextCardCalls.Count)
		assert.Equal(t, userID, mock.GetNextCardCalls.UserIDs[0])

		// SubmitAnswer should return the default stats
		answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
		stats, err := mock.SubmitAnswer(ctx, userID, cardID, answer)
		assert.NoError(t, err)
		assert.Equal(t, sampleStats, stats)
		assert.Equal(t, 1, mock.SubmitAnswerCalls.Count)
		assert.Equal(t, userID, mock.SubmitAnswerCalls.UserIDs[0])
		assert.Equal(t, cardID, mock.SubmitAnswerCalls.CardIDs[0])
		assert.Equal(t, answer, mock.SubmitAnswerCalls.Answers[0])
	})

	t.Run("Custom Functions", func(t *testing.T) {
		// Create a mock with custom function implementations
		mock := NewMockCardReviewService(
			WithGetNextCardFn(func(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
				return sampleCard, nil
			}),
			WithSubmitAnswerFn(
				func(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error) {
					if answer.Outcome == domain.ReviewOutcomeGood {
						return sampleStats, nil
					}
					return nil, customErr
				},
			),
		)

		// Test GetNextCard with custom function
		card, err := mock.GetNextCard(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, sampleCard, card)

		// Test SubmitAnswer with custom function - success case
		goodAnswer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
		stats, err := mock.SubmitAnswer(ctx, userID, cardID, goodAnswer)
		assert.NoError(t, err)
		assert.Equal(t, sampleStats, stats)

		// Test SubmitAnswer with custom function - error case
		badAnswer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeAgain}
		stats, err = mock.SubmitAnswer(ctx, userID, cardID, badAnswer)
		assert.Error(t, err)
		assert.Equal(t, customErr, err)
		assert.Nil(t, stats)

		// Verify call tracking
		assert.Equal(t, 1, mock.GetNextCardCalls.Count)
		assert.Equal(t, 2, mock.SubmitAnswerCalls.Count)
	})

	t.Run("Reset", func(t *testing.T) {
		mock := NewMockCardReviewService()

		// Make some calls
		_, _ = mock.GetNextCard(ctx, userID)
		_, _ = mock.SubmitAnswer(
			ctx,
			userID,
			cardID,
			card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood},
		)

		// Verify counts
		assert.Equal(t, 1, mock.GetNextCardCalls.Count)
		assert.Equal(t, 1, mock.SubmitAnswerCalls.Count)

		// Reset the mock
		mock.Reset()

		// Verify counts are reset
		assert.Equal(t, 0, mock.GetNextCardCalls.Count)
		assert.Equal(t, 0, mock.SubmitAnswerCalls.Count)
		assert.Empty(t, mock.GetNextCardCalls.UserIDs)
		assert.Empty(t, mock.SubmitAnswerCalls.UserIDs)
	})

	t.Run("Convenience Constructors", func(t *testing.T) {
		// Test the convenience constructors

		noCardsMock := NewMockCardReviewServiceWithNoCardsDue()
		_, err := noCardsMock.GetNextCard(ctx, userID)
		assert.Equal(t, card_review.ErrNoCardsDue, err)

		notFoundMock := NewMockCardReviewServiceWithCardNotFound()
		_, err = notFoundMock.GetNextCard(ctx, userID)
		assert.Equal(t, card_review.ErrCardNotFound, err)

		notOwnedMock := NewMockCardReviewServiceWithCardNotOwned()
		_, err = notOwnedMock.GetNextCard(ctx, userID)
		assert.Equal(t, card_review.ErrCardNotOwned, err)

		invalidAnswerMock := NewMockCardReviewServiceWithInvalidAnswer()
		_, err = invalidAnswerMock.GetNextCard(ctx, userID)
		assert.Equal(t, card_review.ErrInvalidAnswer, err)
	})
}
