package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/store"
)

// Compile-time check to ensure PostgresCardStore implements store.CardStore
var _ store.CardStore = (*PostgresCardStore)(nil)

// PostgresCardStore implements the store.CardStore interface
// using a PostgreSQL database as the storage backend.
type PostgresCardStore struct {
	db     store.DBTX
	logger *slog.Logger
	// Cached reference to the original *sql.DB for transaction management
	sqlDB *sql.DB
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

	// Store the database connection if it's a *sql.DB
	var sqlDB *sql.DB
	if dbConn, ok := db.(*sql.DB); ok {
		sqlDB = dbConn
	}

	return &PostgresCardStore{
		db:     db,
		logger: logger.With(slog.String("component", "card_store")),
		sqlDB:  sqlDB,
	}
}

// CreateMultiple implements store.CardStore.CreateMultiple
// It saves multiple cards to the database.
//
// TRANSACTION REQUIREMENT:
// This method MUST be called within a transaction for atomicity and data consistency.
// It assumes it's already running in a transaction context and makes no attempt to
// create, commit, or rollback transactions itself.
//
// If called outside a transaction:
// - Atomicity is not guaranteed (partial data may be inserted)
// - Error handling will be incomplete (failed operations won't be rolled back)
//
// Correct usage example:
//
//	store.RunInTransaction(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
//	    txCardStore := cardStore.WithTxCardStore(tx)
//	    return txCardStore.CreateMultiple(ctx, cards)
//	})
//
// Returns validation errors if any card data is invalid.
func (s *PostgresCardStore) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	// Empty list case
	if len(cards) == 0 {
		log.Debug("no cards to create")
		return nil
	}

	// Validate all cards before proceeding
	for i, card := range cards {
		if err := card.Validate(); err != nil {
			log.Warn("card validation failed",
				slog.String("error", err.Error()),
				slog.String("card_id", card.ID.String()),
				slog.Int("card_index", i))
			return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
		}
	}

	// Insert cards
	cardQuery := `
		INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, card := range cards {
		_, err := s.db.ExecContext(
			ctx,
			cardQuery,
			card.ID,
			card.UserID,
			card.MemoID,
			card.Content,
			card.CreatedAt,
			card.UpdatedAt,
		)

		if err != nil {
			// Check for foreign key violation
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolationCode {
				if strings.Contains(pgErr.Message, "fk_cards_user") {
					log.Warn("foreign key violation - user does not exist",
						slog.String("error", err.Error()),
						slog.String("card_id", card.ID.String()),
						slog.String("user_id", card.UserID.String()))
					return fmt.Errorf("%w: user with ID %s not found",
						store.ErrInvalidEntity, card.UserID)
				}
				if strings.Contains(pgErr.Message, "fk_cards_memo") {
					log.Warn("foreign key violation - memo does not exist",
						slog.String("error", err.Error()),
						slog.String("card_id", card.ID.String()),
						slog.String("memo_id", card.MemoID.String()))
					return fmt.Errorf("%w: memo with ID %s not found",
						store.ErrInvalidEntity, card.MemoID)
				}
			}

			log.Error("failed to insert card",
				slog.String("error", err.Error()),
				slog.String("card_id", card.ID.String()))
			return fmt.Errorf("failed to insert card: %w", MapError(err))
		}

		log.Debug("card inserted successfully",
			slog.String("card_id", card.ID.String()),
			slog.String("user_id", card.UserID.String()),
			slog.String("memo_id", card.MemoID.String()))
	}

	log.Debug("batch card creation completed successfully",
		slog.Int("card_count", len(cards)))
	return nil
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
		if IsNotFoundError(err) {
			log.Debug("card not found", slog.String("card_id", id.String()))
			return nil, store.ErrCardNotFound
		}
		log.Error("failed to get card by ID",
			slog.String("error", err.Error()),
			slog.String("card_id", id.String()))
		return nil, fmt.Errorf("failed to get card by ID: %w", MapError(err))
	}

	log.Debug("card retrieved successfully",
		slog.String("card_id", id.String()),
		slog.String("user_id", card.UserID.String()),
		slog.String("memo_id", card.MemoID.String()))
	return &card, nil
}

// UpdateContent implements store.CardStore.UpdateContent
// It modifies an existing card's content field.
// Returns store.ErrCardNotFound if the card does not exist.
// Returns validation errors if the content is invalid JSON.
func (s *PostgresCardStore) UpdateContent(ctx context.Context, id uuid.UUID, content []byte) error {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("updating card content",
		slog.String("card_id", id.String()))

	// Validate JSON content before updating
	if !json.Valid(content) {
		log.Warn("invalid JSON content for card update",
			slog.String("card_id", id.String()))
		return fmt.Errorf("%w: %v", store.ErrInvalidEntity, domain.ErrInvalidCardContent)
	}

	// Set update timestamp
	updatedAt := time.Now().UTC()

	query := `
		UPDATE cards
		SET content = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := s.db.ExecContext(
		ctx,
		query,
		content,
		updatedAt,
		id,
	)

	if err != nil {
		log.Error("failed to update card content",
			slog.String("error", err.Error()),
			slog.String("card_id", id.String()))
		return fmt.Errorf("failed to update card content: %w", MapError(err))
	}

	// Check if a row was actually updated
	err = CheckRowsAffected(result, "card")
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.ErrCardNotFound
		}
		return fmt.Errorf("failed to update card content: %w", err)
	}

	log.Debug("card content updated successfully",
		slog.String("card_id", id.String()))
	return nil
}

// Delete implements store.CardStore.Delete
// It removes a card from the store by its ID.
// Returns store.ErrCardNotFound if the card does not exist.
//
// IMPORTANT DATABASE DEPENDENCY:
// This implementation relies entirely on the database's CASCADE DELETE functionality
// to maintain referential integrity. When a card is deleted:
//  1. The database automatically deletes related user_card_stats entries via the
//     ON DELETE CASCADE constraint defined in the user_card_stats table schema
//  2. No explicit deletion of related records occurs in this application code
//
// This approach provides better performance and atomicity, but creates a critical
// dependency on correct database schema configuration. If the database schema changes
// or if using a storage backend without CASCADE DELETE support, this method must be
// modified to explicitly handle related record deletion.
//
// See: internal/platform/postgres/migrations/20250415000004_create_user_card_stats_table.sql
// for the constraint definition.
func (s *PostgresCardStore) Delete(ctx context.Context, id uuid.UUID) error {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("deleting card", slog.String("card_id", id.String()))

	query := `
		DELETE FROM cards
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		log.Error("failed to delete card",
			slog.String("error", err.Error()),
			slog.String("card_id", id.String()))
		return fmt.Errorf("failed to delete card: %w", MapError(err))
	}

	// Check if a row was actually deleted
	err = CheckRowsAffected(result, "card")
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.ErrCardNotFound
		}
		return fmt.Errorf("failed to delete card: %w", err)
	}

	log.Debug("card deleted successfully",
		slog.String("card_id", id.String()))
	return nil
}

// GetNextReviewCard implements store.CardStore.GetNextReviewCard
// It retrieves the next card due for review for a user.
// This is based on the UserCardStats.NextReviewAt field.
//
// TODO(card-review): Implement GetNextReviewCard with proper SRS algorithm integration:
//  1. Add SQL query to fetch the next card due for review, using the UserCardStats.NextReviewAt
//     field to determine which cards are ready for review
//  2. Sort by NextReviewAt (ascending) to prioritize cards that are most overdue
//  3. Apply additional filtering for card status if needed
//  4. Join with cards table to return complete card data
//  5. Add unit tests verifying correct card selection based on review times
//
// See docs/design/srs_algorithm.md for full implementation details and parameters.
// Reference the domain/srs package for algorithm calculation functions.
func (s *PostgresCardStore) GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Warn("GetNextReviewCard not implemented",
		slog.String("user_id", userID.String()))

	// Return standard error instead of panicking
	// This follows proper error handling patterns for not-yet-implemented methods
	return nil, store.ErrNotImplemented
}

// WithTxCardStore implements store.CardStore.WithTxCardStore
// It returns a new CardStore instance that uses the provided transaction.
// This allows for multiple operations to be executed within a single transaction.
func (s *PostgresCardStore) WithTxCardStore(tx *sql.Tx) store.CardStore {
	return &PostgresCardStore{
		db:     tx,
		logger: s.logger,
		sqlDB:  s.sqlDB, // Preserve the original DB connection
	}
}

// The following methods allow PostgresCardStore to be used with the task.CardRepository interface

// WithTx returns a new CardStore instance that uses the provided transaction.
// This is part of the task.CardRepository interface.
func (s *PostgresCardStore) WithTx(tx *sql.Tx) store.CardStore {
	return &PostgresCardStore{
		db:     tx,
		logger: s.logger,
		sqlDB:  s.sqlDB, // Preserve the original DB connection
	}
}

// DB returns the underlying database connection.
// This is part of the task.CardRepository interface to support transaction management.
func (s *PostgresCardStore) DB() *sql.DB {
	return s.sqlDB
}
