package service

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
)

// NewCardRepositoryAdapter creates a new adapter that allows a store.CardStore
// to be used where a CardRepository is expected.
func NewCardRepositoryAdapter(cardStore store.CardStore, db *sql.DB) CardRepository {
	return &cardRepositoryAdapter{
		cardStore: cardStore,
		db:        db,
	}
}

// cardRepositoryAdapter adapts a store.CardStore to the CardRepository interface
type cardRepositoryAdapter struct {
	cardStore store.CardStore
	db        *sql.DB
}

// CreateMultiple implements CardRepository.CreateMultiple
func (a *cardRepositoryAdapter) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	return a.cardStore.CreateMultiple(ctx, cards)
}

// GetByID implements CardRepository.GetByID
func (a *cardRepositoryAdapter) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	return a.cardStore.GetByID(ctx, id)
}

// UpdateContent implements CardRepository.UpdateContent
func (a *cardRepositoryAdapter) UpdateContent(ctx context.Context, id uuid.UUID, content json.RawMessage) error {
	return a.cardStore.UpdateContent(ctx, id, content)
}

// Delete implements CardRepository.Delete
func (a *cardRepositoryAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.cardStore.Delete(ctx, id)
}

// WithTx implements CardRepository.WithTx
func (a *cardRepositoryAdapter) WithTx(tx *sql.Tx) CardRepository {
	return &cardRepositoryAdapter{
		cardStore: a.cardStore.WithTx(tx),
		db:        a.db,
	}
}

// DB implements CardRepository.DB
func (a *cardRepositoryAdapter) DB() *sql.DB {
	return a.db
}
