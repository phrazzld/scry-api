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
	//
	// This method only handles card entities and does not create any associated records
	// like UserCardStats. For coordinated creation of cards and stats, use a service-layer
	// orchestration method.
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
	//
	// IMPORTANT: This method relies on database-level CASCADE DELETE behavior to
	// automatically remove associated UserCardStats entries. This is configured
	// in the database schema through ON DELETE CASCADE foreign key constraints.
	//
	// The current PostgreSQL implementation depends on this database feature and does
	// not explicitly delete related records in application code. If using a different
	// database backend or if the schema is modified to remove cascade deletes, this
	// method must be updated to maintain referential integrity.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetNextReviewCard retrieves the next card due for review for a user.
	// It determines which card is due next based on the UserCardStats.NextReviewAt field,
	// returning the card with the earliest NextReviewAt that is <= current time.
	//
	// The method queries both the cards and user_card_stats tables, joining them to find
	// cards owned by the specified user that are due for review (based on NextReviewAt).
	// Results are ordered by NextReviewAt ascending (oldest due cards first).
	//
	// Parameters:
	//   - ctx: Context for the operation, which can be used for cancellation
	//   - userID: UUID of the user whose cards to check for review
	//
	// Returns:
	//   - (*domain.Card, nil): The next card due for review if one exists
	//   - (nil, store.ErrCardNotFound): If no cards are due for review
	//   - (nil, error): Any other error, typically from the database
	//
	// Error Handling:
	//   - Returns store.ErrCardNotFound (which wraps store.ErrNotFound) when no cards are due
	//   - Database errors are mapped to appropriate store errors via MapError or similar
	//
	// This method is central to the spaced repetition system (SRS) functionality and
	// should be optimized for performance, as it may be called frequently during review sessions.
	GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)

	// WithTx returns a new CardStore instance that uses the provided transaction.
	// This allows for multiple operations to be executed within a single transaction.
	// The transaction should be created and managed by the caller (typically a service).
	//
	// IMPORTANT: Methods like CreateMultiple REQUIRE transaction context for proper
	// atomicity and data consistency. Always use this method with store.RunInTransaction
	// when calling operations that modify multiple database records.
	//
	// Example usage:
	//   err := store.RunInTransaction(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
	//       txCardStore := cardStore.WithTx(tx)
	//       return txCardStore.CreateMultiple(ctx, cards)
	//   })
	WithTx(tx *sql.Tx) CardStore

	// DB returns the underlying database connection.
	// This is used for transaction management with store.RunInTransaction.
	DB() *sql.DB
}
