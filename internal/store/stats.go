package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// UserCardStatsStore manages SRS scheduling data persistence
type UserCardStatsStore interface {
	// Create saves a new statistics entry
	// Returns validation errors or duplicate entry errors
	Create(ctx context.Context, stats *domain.UserCardStats) error

	// Get retrieves statistics by user+card IDs without row locking
	// Returns ErrUserCardStatsNotFound if not found
	Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)

	// GetForUpdate retrieves statistics with row-level lock (SELECT FOR UPDATE)
	// Must be used in a transaction for concurrent update protection
	GetForUpdate(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)

	// Update modifies existing statistics and updates timestamp
	// Returns ErrUserCardStatsNotFound or validation errors
	Update(ctx context.Context, stats *domain.UserCardStats) error

	// Delete removes statistics by user+card IDs
	// Returns ErrUserCardStatsNotFound if not found
	Delete(ctx context.Context, userID, cardID uuid.UUID) error

	// WithTx returns a store instance using the provided transaction
	WithTx(tx *sql.Tx) UserCardStatsStore
}
