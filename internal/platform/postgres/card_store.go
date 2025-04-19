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
// Also creates corresponding UserCardStats entries for each card.
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

	// Determine if we need to start a transaction or use existing one
	var tx = s.db
	var txCreated bool

	// Use type assertion to check if s.db is already a transaction
	_, isTransaction := s.db.(interface {
		Commit() error
		Rollback() error
	})

	// Only create a new transaction if not already in one
	if !isTransaction {
		// Type assertion to get the underlying database connection
		dbConn, ok := s.db.(interface {
			BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
		})
		if !ok {
			log.Error("failed to create transaction - db doesn't support BeginTx")
			return fmt.Errorf("%w: database connection doesn't support transactions", store.ErrTransactionFailed)
		}

		var err error
		txObj, err := dbConn.BeginTx(ctx, nil)
		if err != nil {
			log.Error("failed to begin transaction", slog.String("error", err.Error()))
			return fmt.Errorf("%w: failed to begin transaction: %v", store.ErrTransactionFailed, err)
		}
		tx = txObj
		txCreated = true
		log.Debug("transaction created for batch card insertion")
	}

	var txObj interface {
		Commit() error
		Rollback() error
	}

	// If we created a transaction, ensure it gets committed or rolled back
	if txCreated {
		// Use type assertion to get the actual transaction methods
		var ok bool
		txObj, ok = tx.(interface {
			Commit() error
			Rollback() error
		})
		if !ok {
			log.Error("invalid transaction type")
			return fmt.Errorf("%w: invalid transaction type", store.ErrTransactionFailed)
		}

		defer func() {
			// If panic occurs, try to roll back
			if r := recover(); r != nil {
				_ = txObj.Rollback()
				log.Error("panic during card creation", slog.Any("panic", r))
				panic(r) // Re-throw the panic after cleanup
			}
		}()
	}

	// Insert cards
	cardQuery := `
		INSERT INTO cards (id, user_id, memo_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, card := range cards {
		_, err := tx.ExecContext(
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
					if txCreated {
						_ = txObj.Rollback()
					}
					return fmt.Errorf("%w: user with ID %s not found",
						store.ErrInvalidEntity, card.UserID)
				}
				if strings.Contains(pgErr.Message, "fk_cards_memo") {
					log.Warn("foreign key violation - memo does not exist",
						slog.String("error", err.Error()),
						slog.String("card_id", card.ID.String()),
						slog.String("memo_id", card.MemoID.String()))
					if txCreated {
						_ = txObj.Rollback()
					}
					return fmt.Errorf("%w: memo with ID %s not found",
						store.ErrInvalidEntity, card.MemoID)
				}
			}

			log.Error("failed to insert card",
				slog.String("error", err.Error()),
				slog.String("card_id", card.ID.String()))
			if txCreated {
				_ = txObj.Rollback()
			}
			return fmt.Errorf("failed to insert card: %w", MapError(err))
		}

		log.Debug("card inserted successfully",
			slog.String("card_id", card.ID.String()),
			slog.String("user_id", card.UserID.String()),
			slog.String("memo_id", card.MemoID.String()))
	}

	// Now create the corresponding UserCardStats entries
	statsQuery := `
		INSERT INTO user_card_stats
		(user_id, card_id, interval, ease_factor, consecutive_correct,
		 last_reviewed_at, next_review_at, review_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	for _, card := range cards {
		// Create default stats object
		stats, err := domain.NewUserCardStats(card.UserID, card.ID)
		if err != nil {
			log.Error("failed to create default stats",
				slog.String("error", err.Error()),
				slog.String("card_id", card.ID.String()),
				slog.String("user_id", card.UserID.String()))
			if txCreated {
				_ = txObj.Rollback()
			}
			return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
		}

		// Handle zero time for last_reviewed_at
		var lastReviewedAt interface{}
		if stats.LastReviewedAt.IsZero() {
			lastReviewedAt = nil
		} else {
			lastReviewedAt = stats.LastReviewedAt
		}

		_, err = tx.ExecContext(
			ctx,
			statsQuery,
			stats.UserID,
			stats.CardID,
			stats.Interval,
			stats.EaseFactor,
			stats.ConsecutiveCorrect,
			lastReviewedAt,
			stats.NextReviewAt,
			stats.ReviewCount,
			stats.CreatedAt,
			stats.UpdatedAt,
		)

		if err != nil {
			log.Error("failed to insert card stats",
				slog.String("error", err.Error()),
				slog.String("card_id", card.ID.String()),
				slog.String("user_id", card.UserID.String()))
			if txCreated {
				_ = txObj.Rollback()
			}
			return fmt.Errorf("failed to insert card stats: %w", MapError(err))
		}

		log.Debug("card stats inserted successfully",
			slog.String("card_id", card.ID.String()),
			slog.String("user_id", card.UserID.String()))
	}

	// Commit the transaction if we created it
	if txCreated {
		if err := txObj.Commit(); err != nil {
			log.Error("failed to commit transaction", slog.String("error", err.Error()))
			_ = txObj.Rollback()
			return fmt.Errorf("%w: failed to commit transaction: %v", store.ErrTransactionFailed, err)
		}
		log.Debug("transaction committed successfully")
	}

	log.Info("batch card creation completed successfully",
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

	log.Info("card content updated successfully",
		slog.String("card_id", id.String()))
	return nil
}

// Delete implements store.CardStore.Delete
// It removes a card from the store by its ID.
// Returns store.ErrCardNotFound if the card does not exist.
// Associated user_card_stats entries are also deleted via cascade delete.
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

	log.Info("card deleted successfully",
		slog.String("card_id", id.String()))
	return nil
}

// GetNextReviewCard implements store.CardStore.GetNextReviewCard
// It retrieves the next card due for review for a user.
// This is based on the UserCardStats.NextReviewAt field.
func (s *PostgresCardStore) GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	// This is a stub implementation to satisfy the interface.
	// The actual implementation will be done in a separate ticket.
	return nil, store.ErrNotImplemented
}
