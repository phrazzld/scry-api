package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// UserCardStatsStore defines the interface for user card statistics data persistence.
// Version: 1.0
type UserCardStatsStore interface {
	// Create saves a new user card statistics entry.
	// It handles domain validation internally.
	// Returns validation errors from the domain UserCardStats if data is invalid.
	// Returns an error if the entry already exists.
	Create(ctx context.Context, stats *domain.UserCardStats) error

	// Get retrieves user card statistics by the combination of user ID and card ID.
	// Returns ErrUserCardStatsNotFound if the statistics entry does not exist.
	// This method retrieves a single entry that matches both IDs exactly.
	// NOTE: This method does NOT provide any row locking, so it should not be used
	// when you plan to update the row and need concurrency protection.
	Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)

	// GetForUpdate retrieves user card statistics with a row-level lock using SELECT FOR UPDATE.
	// This should be used within a transaction when you plan to update the row
	// and need protection from concurrent modifications.
	// Returns ErrUserCardStatsNotFound if the statistics entry does not exist.
	GetForUpdate(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)

	// Update modifies an existing statistics entry.
	// It handles domain validation internally.
	// The userID and cardID fields in the stats object are used to identify the record to update.
	// Returns ErrUserCardStatsNotFound if the statistics entry does not exist.
	// Returns validation errors from the domain UserCardStats if data is invalid.
	Update(ctx context.Context, stats *domain.UserCardStats) error

	// Delete removes user card statistics by the combination of user ID and card ID.
	// Returns ErrUserCardStatsNotFound if the statistics entry does not exist.
	// This operation is permanent and cannot be undone.
	Delete(ctx context.Context, userID, cardID uuid.UUID) error

	// WithTx returns a new UserCardStatsStore instance that uses the provided transaction.
	// This allows for multiple operations to be executed within a single transaction.
	// The transaction should be created and managed by the caller (typically a service).
	WithTx(tx *sql.Tx) UserCardStatsStore
}
