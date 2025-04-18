package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// CardStore defines the interface for card data persistence.
type CardStore interface {
	// CreateMultiple saves multiple cards to the store in a single transaction.
	// All cards must be valid according to domain validation rules.
	// The transaction should be atomic - either all cards are created or none.
	// Returns validation errors if any card data is invalid.
	// May also create corresponding UserCardStats entries for each card
	// based on the implementation.
	CreateMultiple(ctx context.Context, cards []*domain.Card) error

	// GetByID retrieves a card by its unique ID.
	// Returns ErrNotFound if the card does not exist.
	// The returned card will have its Content field properly populated from JSONB.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error)

	// UpdateContent modifies an existing card's content field.
	// Returns ErrNotFound if the card does not exist.
	// Returns validation errors if the content is invalid JSON.
	// Implementations should validate the content before updating.
	UpdateContent(ctx context.Context, id uuid.UUID, content []byte) error

	// Delete removes a card from the store by its ID.
	// Returns ErrNotFound if the card does not exist.
	// Depending on the implementation, this may also delete associated
	// UserCardStats entries via cascade delete in the database.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetNextReviewCard retrieves the next card due for review for a user.
	// This is based on the UserCardStats.NextReviewAt field.
	// Returns ErrNotImplemented for stub implementations.
	// Returns ErrNotFound if there are no cards due for review.
	// This method may involve complex sorting/filtering logic based on SRS.
	GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
}
