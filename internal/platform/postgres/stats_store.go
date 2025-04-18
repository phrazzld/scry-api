package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/store"
)

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

// Ensure PostgresUserCardStatsStore implements store.UserCardStatsStore interface
var _ store.UserCardStatsStore = (*PostgresUserCardStatsStore)(nil)

// Get implements store.UserCardStatsStore.Get
// It retrieves user card statistics by the combination of user ID and card ID.
// Returns store.ErrUserCardStatsNotFound if the statistics entry does not exist.
func (s *PostgresUserCardStatsStore) Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error) {
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
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("user card stats not found",
				slog.String("user_id", userID.String()),
				slog.String("card_id", cardID.String()))
			return nil, store.ErrUserCardStatsNotFound
		}
		log.Error("failed to get user card stats",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		return nil, err
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
func (s *PostgresUserCardStatsStore) Update(ctx context.Context, stats *domain.UserCardStats) error {
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
		return err
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
		return err
	}

	// Check if a row was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected",
			slog.String("error", err.Error()),
			slog.String("user_id", stats.UserID.String()),
			slog.String("card_id", stats.CardID.String()))
		return err
	}

	// If no rows were affected, the stats entry didn't exist
	if rowsAffected == 0 {
		log.Debug("user card stats not found for update",
			slog.String("user_id", stats.UserID.String()),
			slog.String("card_id", stats.CardID.String()))
		return store.ErrUserCardStatsNotFound
	}

	log.Info("user card stats updated successfully",
		slog.String("user_id", stats.UserID.String()),
		slog.String("card_id", stats.CardID.String()),
		slog.Time("next_review_at", stats.NextReviewAt))
	return nil
}

// Delete implements store.UserCardStatsStore.Delete
// It removes user card statistics by the combination of user ID and card ID.
// Returns store.ErrUserCardStatsNotFound if the statistics entry does not exist.
func (s *PostgresUserCardStatsStore) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	// This is a stub implementation to satisfy the interface.
	// The actual implementation will be done in a separate ticket.
	return store.ErrNotImplemented
}
