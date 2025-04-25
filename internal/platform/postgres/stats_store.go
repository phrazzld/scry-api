package postgres

import (
	"context"
	"database/sql"
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

// Compile-time check to ensure PostgresUserCardStatsStore implements store.UserCardStatsStore
var _ store.UserCardStatsStore = (*PostgresUserCardStatsStore)(nil)

// PostgresUserCardStatsStore implements the store.UserCardStatsStore interface
// using a PostgreSQL database as the storage backend.
type PostgresUserCardStatsStore struct {
	db     store.DBTX
	logger *slog.Logger
}

// NewPostgresUserCardStatsStore creates a new PostgreSQL implementation of the UserCardStatsStore interface.
// It accepts a database connection or transaction that should be initialized and managed by the caller.
// If logger is nil, a default logger will be used.
func NewPostgresUserCardStatsStore(db store.DBTX, logger *slog.Logger) *PostgresUserCardStatsStore {
	// Validate inputs
	if db == nil {
		// ALLOW-PANIC: Constructor enforcing required dependency
		panic("db cannot be nil")
	}

	// Use provided logger or create default
	if logger == nil {
		logger = slog.Default()
	}

	return &PostgresUserCardStatsStore{
		db:     db,
		logger: logger.With(slog.String("component", "user_card_stats_store")),
	}
}

// Compile-time check to ensure PostgresUserCardStatsStore implements store.UserCardStatsStore interface
var _ store.UserCardStatsStore = (*PostgresUserCardStatsStore)(nil)

// Create implements store.UserCardStatsStore.Create
// It saves a new user card statistics entry.
// Returns validation errors from the domain UserCardStats if data is invalid.
func (s *PostgresUserCardStatsStore) Create(
	ctx context.Context,
	stats *domain.UserCardStats,
) error {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("creating user card stats",
		slog.String("user_id", stats.UserID.String()),
		slog.String("card_id", stats.CardID.String()))

	// Validate stats before creating
	if err := stats.Validate(); err != nil {
		log.Warn("user card stats validation failed during creation",
			slog.String("error", err.Error()),
			slog.String("user_id", stats.UserID.String()),
			slog.String("card_id", stats.CardID.String()))
		return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
	}

	query := `
		INSERT INTO user_card_stats (user_id, card_id, interval, ease_factor, consecutive_correct,
								   last_reviewed_at, next_review_at, review_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	// Handling NULL value for LastReviewedAt
	var lastReviewedAt interface{}
	if stats.LastReviewedAt.IsZero() {
		lastReviewedAt = nil
	} else {
		lastReviewedAt = stats.LastReviewedAt
	}

	_, err := s.db.ExecContext(
		ctx,
		query,
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
		// Check for unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			log.Warn("duplicate user card stats entry",
				slog.String("error", err.Error()),
				slog.String("user_id", stats.UserID.String()),
				slog.String("card_id", stats.CardID.String()))
			return fmt.Errorf(
				"user card stats entry already exists for user_id=%s and card_id=%s: %w",
				stats.UserID,
				stats.CardID,
				store.ErrDuplicate,
			)
		}

		// Check for foreign key violation
		if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolationCode {
			if strings.Contains(pgErr.Message, "fk_user_card_stats_user") {
				log.Warn("foreign key violation - user does not exist",
					slog.String("error", err.Error()),
					slog.String("user_id", stats.UserID.String()))
				return fmt.Errorf("%w: user with ID %s not found",
					store.ErrInvalidEntity, stats.UserID)
			}
			if strings.Contains(pgErr.Message, "fk_user_card_stats_card") {
				log.Warn("foreign key violation - card does not exist",
					slog.String("error", err.Error()),
					slog.String("card_id", stats.CardID.String()))
				return fmt.Errorf("%w: card with ID %s not found",
					store.ErrInvalidEntity, stats.CardID)
			}
		}

		log.Error("failed to create user card stats",
			slog.String("error", err.Error()),
			slog.String("user_id", stats.UserID.String()),
			slog.String("card_id", stats.CardID.String()))
		return fmt.Errorf("failed to create user card stats: %w", MapError(err))
	}

	log.Debug("user card stats created successfully",
		slog.String("user_id", stats.UserID.String()),
		slog.String("card_id", stats.CardID.String()),
		slog.Time("next_review_at", stats.NextReviewAt))
	return nil
}

// Get implements store.UserCardStatsStore.Get
// It retrieves user card statistics by the combination of user ID and card ID.
// Returns store.ErrUserCardStatsNotFound if the statistics entry does not exist.
//
// IMPORTANT: This implementation does NOT use row locking. If you need to read
// and then update a row within a transaction with concurrency protection, use
// GetForUpdate instead.
func (s *PostgresUserCardStatsStore) Get(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("retrieving user card stats",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))

	query := `
		SELECT user_id, card_id, interval, ease_factor, consecutive_correct,
			   last_reviewed_at, next_review_at, review_count, created_at, updated_at
		FROM user_card_stats
		WHERE user_id = $1 AND card_id = $2
	`

	var stats domain.UserCardStats
	var lastReviewedAt sql.NullTime // Using sql.NullTime since last_reviewed_at can be NULL

	err := s.db.QueryRowContext(ctx, query, userID, cardID).Scan(
		&stats.UserID,
		&stats.CardID,
		&stats.Interval,
		&stats.EaseFactor,
		&stats.ConsecutiveCorrect,
		&lastReviewedAt,
		&stats.NextReviewAt,
		&stats.ReviewCount,
		&stats.CreatedAt,
		&stats.UpdatedAt,
	)

	if err != nil {
		if IsNotFoundError(err) {
			log.Debug("user card stats not found",
				slog.String("user_id", userID.String()),
				slog.String("card_id", cardID.String()))
			return nil, store.ErrUserCardStatsNotFound
		}
		log.Error("failed to get user card stats",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		return nil, fmt.Errorf("failed to query user card stats: %w", MapError(err))
	}

	// Handle the nullable LastReviewedAt field
	if lastReviewedAt.Valid {
		stats.LastReviewedAt = lastReviewedAt.Time
	} else {
		stats.LastReviewedAt = time.Time{} // Zero value for time.Time
	}

	log.Debug("user card stats retrieved successfully",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
		slog.Time("next_review_at", stats.NextReviewAt))
	return &stats, nil
}

// Update implements store.UserCardStatsStore.Update
// It modifies an existing statistics entry.
// Returns store.ErrUserCardStatsNotFound if the statistics entry does not exist.
// Returns validation errors from the domain UserCardStats if data is invalid.
func (s *PostgresUserCardStatsStore) Update(
	ctx context.Context,
	stats *domain.UserCardStats,
) error {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("updating user card stats",
		slog.String("user_id", stats.UserID.String()),
		slog.String("card_id", stats.CardID.String()))

	// Validate stats before updating
	if err := stats.Validate(); err != nil {
		log.Warn("user card stats validation failed during update",
			slog.String("error", err.Error()),
			slog.String("user_id", stats.UserID.String()),
			slog.String("card_id", stats.CardID.String()))
		return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
	}

	// Always update the UpdatedAt timestamp
	stats.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE user_card_stats
		SET interval = $1,
			ease_factor = $2,
			consecutive_correct = $3,
			last_reviewed_at = $4,
			next_review_at = $5,
			review_count = $6,
			updated_at = $7
		WHERE user_id = $8 AND card_id = $9
	`

	// Handling NULL value for LastReviewedAt
	var lastReviewedAt interface{}
	if stats.LastReviewedAt.IsZero() {
		lastReviewedAt = nil
	} else {
		lastReviewedAt = stats.LastReviewedAt
	}

	result, err := s.db.ExecContext(
		ctx,
		query,
		stats.Interval,
		stats.EaseFactor,
		stats.ConsecutiveCorrect,
		lastReviewedAt,
		stats.NextReviewAt,
		stats.ReviewCount,
		stats.UpdatedAt,
		stats.UserID,
		stats.CardID,
	)

	if err != nil {
		log.Error("failed to update user card stats",
			slog.String("error", err.Error()),
			slog.String("user_id", stats.UserID.String()),
			slog.String("card_id", stats.CardID.String()))
		return fmt.Errorf("failed to execute user card stats update: %w", MapError(err))
	}

	// Check if a row was actually updated using the helper
	err = CheckRowsAffected(result, "user card stats")
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.ErrUserCardStatsNotFound
		}
		return fmt.Errorf("failed to update user card stats: %w", err)
	}

	log.Debug("user card stats updated successfully",
		slog.String("user_id", stats.UserID.String()),
		slog.String("card_id", stats.CardID.String()),
		slog.Time("next_review_at", stats.NextReviewAt))
	return nil
}

// Delete implements store.UserCardStatsStore.Delete
// It removes user card statistics by the combination of user ID and card ID.
// Returns store.ErrUserCardStatsNotFound if the statistics entry does not exist.
func (s *PostgresUserCardStatsStore) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("deleting user card stats",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))

	query := `
		DELETE FROM user_card_stats
		WHERE user_id = $1 AND card_id = $2
	`

	result, err := s.db.ExecContext(ctx, query, userID, cardID)
	if err != nil {
		log.Error("failed to delete user card stats",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		return fmt.Errorf("failed to execute user card stats deletion: %w", MapError(err))
	}

	// Check if a row was actually deleted using the helper
	err = CheckRowsAffected(result, "user card stats")
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.ErrUserCardStatsNotFound
		}
		return fmt.Errorf("failed to delete user card stats: %w", err)
	}

	log.Debug("user card stats deleted successfully",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))
	return nil
}

// GetForUpdate is similar to Get but uses a SELECT FOR UPDATE to acquire a row-level lock.
// This should be used within a transaction when you plan to update the row
// to prevent concurrent modifications.
// Returns store.ErrUserCardStatsNotFound if the statistics entry does not exist.
func (s *PostgresUserCardStatsStore) GetForUpdate(
	ctx context.Context,
	userID, cardID uuid.UUID,
) (*domain.UserCardStats, error) {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("retrieving user card stats with lock",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))

	query := `
		SELECT user_id, card_id, interval, ease_factor, consecutive_correct,
			   last_reviewed_at, next_review_at, review_count, created_at, updated_at
		FROM user_card_stats
		WHERE user_id = $1 AND card_id = $2
		FOR UPDATE
	`

	var stats domain.UserCardStats
	var lastReviewedAt sql.NullTime // Using sql.NullTime since last_reviewed_at can be NULL

	err := s.db.QueryRowContext(ctx, query, userID, cardID).Scan(
		&stats.UserID,
		&stats.CardID,
		&stats.Interval,
		&stats.EaseFactor,
		&stats.ConsecutiveCorrect,
		&lastReviewedAt,
		&stats.NextReviewAt,
		&stats.ReviewCount,
		&stats.CreatedAt,
		&stats.UpdatedAt,
	)

	if err != nil {
		if IsNotFoundError(err) {
			log.Debug("user card stats not found for update",
				slog.String("user_id", userID.String()),
				slog.String("card_id", cardID.String()))
			return nil, store.ErrUserCardStatsNotFound
		}
		log.Error("failed to get user card stats with lock",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		return nil, fmt.Errorf("failed to query user card stats with lock: %w", MapError(err))
	}

	// Handle the nullable LastReviewedAt field
	if lastReviewedAt.Valid {
		stats.LastReviewedAt = lastReviewedAt.Time
	} else {
		stats.LastReviewedAt = time.Time{} // Zero value for time.Time
	}

	log.Debug("user card stats retrieved with lock successfully",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
		slog.Time("next_review_at", stats.NextReviewAt))
	return &stats, nil
}

// WithTx implements store.UserCardStatsStore.WithTx
// It returns a new UserCardStatsStore instance that uses the provided transaction.
// This allows for multiple operations to be executed within a single transaction.
func (s *PostgresUserCardStatsStore) WithTx(tx *sql.Tx) store.UserCardStatsStore {
	return &PostgresUserCardStatsStore{
		db:     tx,
		logger: s.logger,
	}
}
