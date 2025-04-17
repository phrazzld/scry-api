// Package mocks provides mock implementations for testing task components.
package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// MemoRepository is a simple implementation of the MemoRepository interface.
type MemoRepository struct {
	GetByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Memo, error)
	UpdateFunc  func(ctx context.Context, memo *domain.Memo) error
}

// GetByID retrieves a memo by its unique ID.
func (m *MemoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

// Update saves changes to an existing memo.
func (m *MemoRepository) Update(ctx context.Context, memo *domain.Memo) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, memo)
	}
	return nil
}

// CardRepository is a simple implementation of the CardRepository interface.
type CardRepository struct {
	CreateMultipleFunc func(ctx context.Context, cards []*domain.Card) error
}

// CreateMultiple saves multiple new cards to the store.
func (m *CardRepository) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	if m.CreateMultipleFunc != nil {
		return m.CreateMultipleFunc(ctx, cards)
	}
	return nil
}

// Generator is a simple implementation of the Generator interface.
type Generator struct {
	GenerateCardsFunc func(ctx context.Context, memoText string, userID uuid.UUID) ([]*domain.Card, error)
}

// GenerateCards creates flashcards from memo text.
func (m *Generator) GenerateCards(ctx context.Context, memoText string, userID uuid.UUID) ([]*domain.Card, error) {
	if m.GenerateCardsFunc != nil {
		return m.GenerateCardsFunc(ctx, memoText, userID)
	}
	return nil, nil
}
