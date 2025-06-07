package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// CardStore defines the interface for card data persistence
type CardStore interface {
	// CreateMultiple saves multiple cards in a transaction for atomicity
	// Must be run within a transaction to prevent partial data insertion
	CreateMultiple(ctx context.Context, cards []*domain.Card) error

	// GetByID retrieves a card by ID or returns ErrCardNotFound
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error)

	// UpdateContent modifies a card's content field
	// Returns ErrCardNotFound or validation errors for invalid JSON
	UpdateContent(ctx context.Context, id uuid.UUID, content []byte) error

	// Delete removes a card and its associated records via CASCADE
	// Returns ErrCardNotFound if the card doesn't exist
	Delete(ctx context.Context, id uuid.UUID) error

	// GetNextReviewCard retrieves the earliest due card for review
	// Orders by NextReviewAt ascending; returns ErrCardNotFound when none due
	GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)

	// WithTx returns a store instance using the provided transaction
	WithTx(tx *sql.Tx) CardStore

	// DB returns the underlying database connection
	DB() *sql.DB
}
