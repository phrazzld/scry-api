package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// MockCardService is a mock implementation of the card service for testing.
type MockCardService struct {
	CreateCardsFunc func(ctx context.Context, cards []*domain.Card) error
	GetCardFunc     func(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)
}

// CreateCards implements the CardService interface for testing.
func (m *MockCardService) CreateCards(ctx context.Context, cards []*domain.Card) error {
	if m.CreateCardsFunc != nil {
		return m.CreateCardsFunc(ctx, cards)
	}
	return nil
}

// GetCard implements the CardService interface for testing.
func (m *MockCardService) GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
	if m.GetCardFunc != nil {
		return m.GetCardFunc(ctx, cardID)
	}
	return nil, nil
}
