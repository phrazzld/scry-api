package postgres

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
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
// Returns store.ErrNotFound if the statistics entry does not exist.
func (s *PostgresUserCardStatsStore) Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error) {
	// This is a stub implementation to satisfy the interface.
	// The actual implementation will be done in a separate ticket.
	return nil, store.ErrNotImplemented
}

// Update implements store.UserCardStatsStore.Update
// It modifies an existing statistics entry.
// Returns store.ErrNotFound if the statistics entry does not exist.
// Returns validation errors from the domain UserCardStats if data is invalid.
func (s *PostgresUserCardStatsStore) Update(ctx context.Context, stats *domain.UserCardStats) error {
	// This is a stub implementation to satisfy the interface.
	// The actual implementation will be done in a separate ticket.
	return store.ErrNotImplemented
}

// Delete implements store.UserCardStatsStore.Delete
// It removes user card statistics by the combination of user ID and card ID.
// Returns store.ErrNotFound if the statistics entry does not exist.
func (s *PostgresUserCardStatsStore) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	// This is a stub implementation to satisfy the interface.
	// The actual implementation will be done in a separate ticket.
	return store.ErrNotImplemented
}
