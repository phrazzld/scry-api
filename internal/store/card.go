package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// CardStore defines the interface for card data persistence.
// Version: 1.0
type CardStore interface {
	// CreateMultiple saves multiple cards to the store.
	// IMPORTANT: This method MUST be run within a transaction for atomicity and data consistency.
	// Use the WithTx method with store.RunInTransaction to ensure proper transaction handling.
	// Calling this method outside a transaction will not guarantee atomic behavior and may
	// result in partial data insertion if failures occur.
	//
	// All cards must be valid according to domain validation rules.
	// Returns validation errors if any card data is invalid.
	// May also create corresponding UserCardStats entries for each card
	// based on the implementation.
	//
	// Usage example:
	//   err := store.RunInTransaction(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
	//       txCardStore := cardStore.WithTx(tx)
	//       return txCardStore.CreateMultiple(ctx, cards)
	//   })
	CreateMultiple(ctx context.Context, cards []*domain.Card) error

	// GetByID retrieves a card by its unique ID.
	// Returns ErrCardNotFound if the card does not exist.
	// The returned card will have its Content field properly populated from JSONB.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error)

	// UpdateContent modifies an existing card's content field.
	// Returns ErrCardNotFound if the card does not exist.
	// Returns validation errors if the content is invalid JSON.
	// Implementations should validate the content before updating.
	UpdateContent(ctx context.Context, id uuid.UUID, content []byte) error

	// Delete removes a card from the store by its ID.
	// Returns ErrCardNotFound if the card does not exist.
	// Depending on the implementation, this may also delete associated
	// UserCardStats entries via cascade delete in the database.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetNextReviewCard retrieves the next card due for review for a user.
	// This is based on the UserCardStats.NextReviewAt field.
	// Returns ErrNotImplemented for stub implementations.
	// Returns ErrNotFound if there are no cards due for review.
	// This method may involve complex sorting/filtering logic based on SRS.
	GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)

	// WithTxCardStore returns a new CardStore instance that uses the provided transaction.
	// This allows for multiple operations to be executed within a single transaction.
	// The transaction should be created and managed by the caller (typically a service).
	//
	// IMPORTANT: Methods like CreateMultiple REQUIRE transaction context for proper
	// atomicity and data consistency. Always use this method with store.RunInTransaction
	// when calling operations that modify multiple database records.
	//
	// Example usage:
	//   err := store.RunInTransaction(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
	//       txCardStore := cardStore.WithTxCardStore(tx)
	//       return txCardStore.CreateMultiple(ctx, cards)
	//   })
	WithTxCardStore(tx *sql.Tx) CardStore
}
