package mocks

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// MockCardService implements service.CardService for testing
type MockCardService struct {
	// Custom behavior functions
	UpdateCardContentFn func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error
	DeleteCardFn        func(ctx context.Context, userID, cardID uuid.UUID) error
	PostponeCardFn      func(ctx context.Context, userID, cardID uuid.UUID, days int) (*domain.UserCardStats, error)
	CreateCardsFn       func(ctx context.Context, cards []*domain.Card) error
	GetCardFn           func(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)

	// Default return values
	Card         *domain.Card
	Stats        *domain.UserCardStats
	DefaultError error
}

// UpdateCardContent implements the CardService.UpdateCardContent method
func (m *MockCardService) UpdateCardContent(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error {
	if m.UpdateCardContentFn != nil {
		return m.UpdateCardContentFn(ctx, userID, cardID, content)
	}
	return m.DefaultError
}

// DeleteCard implements the CardService.DeleteCard method
func (m *MockCardService) DeleteCard(ctx context.Context, userID, cardID uuid.UUID) error {
	if m.DeleteCardFn != nil {
		return m.DeleteCardFn(ctx, userID, cardID)
	}
	return m.DefaultError
}

// PostponeCard implements the CardService.PostponeCard method
func (m *MockCardService) PostponeCard(ctx context.Context, userID, cardID uuid.UUID, days int) (*domain.UserCardStats, error) {
	if m.PostponeCardFn != nil {
		return m.PostponeCardFn(ctx, userID, cardID, days)
	}
	return m.Stats, m.DefaultError
}

// CreateCards implements the CardService.CreateCards method
func (m *MockCardService) CreateCards(ctx context.Context, cards []*domain.Card) error {
	if m.CreateCardsFn != nil {
		return m.CreateCardsFn(ctx, cards)
	}
	return m.DefaultError
}

// GetCard implements the CardService.GetCard method
func (m *MockCardService) GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
	if m.GetCardFn != nil {
		return m.GetCardFn(ctx, cardID)
	}
	return m.Card, m.DefaultError
}
