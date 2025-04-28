package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// MockCardService is a mock implementation of the CardService interface
type CardService struct {
	CreateCardsFunc func(ctx context.Context, cards []*domain.Card) error
	GetCardFunc     func(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)
}

// CreateCards creates multiple cards and their associated stats in a single transaction
func (m *CardService) CreateCards(ctx context.Context, cards []*domain.Card) error {
	if m.CreateCardsFunc != nil {
		return m.CreateCardsFunc(ctx, cards)
	}
	return nil
}

// GetCard retrieves a card by its ID
func (m *CardService) GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
	if m.GetCardFunc != nil {
		return m.GetCardFunc(ctx, cardID)
	}
	return nil, nil
}
