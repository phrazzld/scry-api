package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// Common user card stats store errors
var (
	// ErrUserCardStatsNotFound indicates that the requested user card stats do not exist in the store.
	ErrUserCardStatsNotFound = errors.New("user card stats not found")
)

// UserCardStatsStore defines the interface for user card statistics data persistence.
type UserCardStatsStore interface {
	// Get retrieves user card statistics by the combination of user ID and card ID.
	// Returns ErrUserCardStatsNotFound if the statistics entry does not exist.
	// This method retrieves a single entry that matches both IDs exactly.
	Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)

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
}
