//go:build integration

package card_review

import (
	"context"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// MockCardReviewService is a mock implementation of the CardReviewService interface for testing.
type MockCardReviewService struct {
	GetNextCardFunc  func(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
	SubmitAnswerFunc func(ctx context.Context, userID, cardID uuid.UUID, answer ReviewAnswer) (*domain.UserCardStats, error)
	PostponeCardFunc func(ctx context.Context, userID, cardID uuid.UUID, hours int) error
}

// GetNextCard returns the next card for review for the given user.
func (m *MockCardReviewService) GetNextCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	if m.GetNextCardFunc != nil {
		return m.GetNextCardFunc(ctx, userID)
	}
	return nil, nil
}

// SubmitAnswer submits an answer for a card and returns the updated stats.
func (m *MockCardReviewService) SubmitAnswer(
	ctx context.Context,
	userID, cardID uuid.UUID,
	answer ReviewAnswer,
) (*domain.UserCardStats, error) {
	if m.SubmitAnswerFunc != nil {
		return m.SubmitAnswerFunc(ctx, userID, cardID, answer)
	}
	return nil, nil
}

// PostponeCard postpones a card for a specified number of hours.
func (m *MockCardReviewService) PostponeCard(ctx context.Context, userID, cardID uuid.UUID, hours int) error {
	if m.PostponeCardFunc != nil {
		return m.PostponeCardFunc(ctx, userID, cardID, hours)
	}
	return nil
}
