//go:build test_without_external_deps

package card_review_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/stretchr/testify/assert"
)

func TestMockCardReviewService_GetNextCard(t *testing.T) {
	t.Run("with_function_returns_result", func(t *testing.T) {
		expectedCard := createTestCard(uuid.New())
		expectedError := errors.New("test error")

		mock := &card_review.MockCardReviewService{
			GetNextCardFunc: func(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
				return expectedCard, expectedError
			},
		}

		card, err := mock.GetNextCard(context.Background(), uuid.New())
		assert.Equal(t, expectedCard, card)
		assert.Equal(t, expectedError, err)
	})

	t.Run("without_function_returns_nil", func(t *testing.T) {
		mock := &card_review.MockCardReviewService{}

		card, err := mock.GetNextCard(context.Background(), uuid.New())
		assert.Nil(t, card)
		assert.Nil(t, err)
	})
}

func TestMockCardReviewService_SubmitAnswer(t *testing.T) {
	t.Run("with_function_returns_result", func(t *testing.T) {
		userID := uuid.New()
		cardID := uuid.New()
		expectedStats := &domain.UserCardStats{
			UserID: userID,
			CardID: cardID,
		}
		expectedError := errors.New("test error")

		mock := &card_review.MockCardReviewService{
			SubmitAnswerFunc: func(ctx context.Context, userID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error) {
				return expectedStats, expectedError
			},
		}

		answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
		stats, err := mock.SubmitAnswer(context.Background(), userID, cardID, answer)
		assert.Equal(t, expectedStats, stats)
		assert.Equal(t, expectedError, err)
	})

	t.Run("without_function_returns_nil", func(t *testing.T) {
		mock := &card_review.MockCardReviewService{}

		answer := card_review.ReviewAnswer{Outcome: domain.ReviewOutcomeGood}
		stats, err := mock.SubmitAnswer(context.Background(), uuid.New(), uuid.New(), answer)
		assert.Nil(t, stats)
		assert.Nil(t, err)
	})
}

func TestMockCardReviewService_PostponeCard(t *testing.T) {
	t.Run("with_function_returns_result", func(t *testing.T) {
		expectedError := errors.New("test error")

		mock := &card_review.MockCardReviewService{
			PostponeCardFunc: func(ctx context.Context, userID, cardID uuid.UUID, hours int) error {
				return expectedError
			},
		}

		err := mock.PostponeCard(context.Background(), uuid.New(), uuid.New(), 24)
		assert.Equal(t, expectedError, err)
	})

	t.Run("without_function_returns_nil", func(t *testing.T) {
		mock := &card_review.MockCardReviewService{}

		err := mock.PostponeCard(context.Background(), uuid.New(), uuid.New(), 24)
		assert.Nil(t, err)
	})
}
