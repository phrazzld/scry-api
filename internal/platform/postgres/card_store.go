package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/store"
)

// PostgresCardStore implements the store.CardStore interface
// using a PostgreSQL database as the storage backend.
type PostgresCardStore struct {
	db     store.DBTX
	logger *slog.Logger
}

// NewPostgresCardStore creates a new PostgreSQL implementation of the CardStore interface.
// It accepts a database connection or transaction that should be initialized and managed by the caller.
// If logger is nil, a default logger will be used.
func NewPostgresCardStore(db store.DBTX, logger *slog.Logger) *PostgresCardStore {
	// Validate inputs
	if db == nil {
		panic("db cannot be nil")
	}

	// Use provided logger or create default
	if logger == nil {
		logger = slog.Default()
	}

	return &PostgresCardStore{
		db:     db,
		logger: logger.With(slog.String("component", "card_store")),
	}
}

// Ensure PostgresCardStore implements store.CardStore interface
var _ store.CardStore = (*PostgresCardStore)(nil)

// CreateMultiple implements store.CardStore.CreateMultiple
// It saves multiple cards to the database in a single transaction.
// The operation is atomic - either all cards are created or none.
// Returns validation errors if any card data is invalid.
func (s *PostgresCardStore) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	// This is a stub implementation to satisfy the interface.
	// The actual implementation will be done in a separate ticket.
	return store.ErrNotImplemented
}

// GetByID implements store.CardStore.GetByID
// It retrieves a card by its unique ID.
// Returns store.ErrCardNotFound if the card does not exist.
func (s *PostgresCardStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("retrieving card by ID", slog.String("card_id", id.String()))

	query := `
		SELECT id, user_id, memo_id, content, created_at, updated_at
		FROM cards
		WHERE id = $1
	`

	var card domain.Card

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&card.ID,
		&card.UserID,
		&card.MemoID,
		&card.Content,
		&card.CreatedAt,
		&card.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("card not found", slog.String("card_id", id.String()))
			return nil, store.ErrCardNotFound
		}
		log.Error("failed to get card by ID",
			slog.String("error", err.Error()),
			slog.String("card_id", id.String()))
		return nil, err
	}

	log.Debug("card retrieved successfully",
		slog.String("card_id", id.String()),
		slog.String("user_id", card.UserID.String()),
		slog.String("memo_id", card.MemoID.String()))
	return &card, nil
}

// UpdateContent implements store.CardStore.UpdateContent
// It modifies an existing card's content field.
// Returns store.ErrNotFound if the card does not exist.
// Returns validation errors if the content is invalid JSON.
func (s *PostgresCardStore) UpdateContent(ctx context.Context, id uuid.UUID, content []byte) error {
	// This is a stub implementation to satisfy the interface.
	// The actual implementation will be done in a separate ticket.
	return store.ErrNotImplemented
}

// Delete implements store.CardStore.Delete
// It removes a card from the store by its ID.
// Returns store.ErrNotFound if the card does not exist.
func (s *PostgresCardStore) Delete(ctx context.Context, id uuid.UUID) error {
	// This is a stub implementation to satisfy the interface.
	// The actual implementation will be done in a separate ticket.
	return store.ErrNotImplemented
}

// GetNextReviewCard implements store.CardStore.GetNextReviewCard
// It retrieves the next card due for review for a user.
// This is based on the UserCardStats.NextReviewAt field.
func (s *PostgresCardStore) GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	// This is a stub implementation to satisfy the interface.
	// The actual implementation will be done in a separate ticket.
	return nil, store.ErrNotImplemented
}
