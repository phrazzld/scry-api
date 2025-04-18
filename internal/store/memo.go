package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// Common memo store errors
var (
	// ErrMemoNotFound indicates that the requested memo does not exist in the store.
	ErrMemoNotFound = errors.New("memo not found")
)

// MemoStore defines the interface for memo data persistence.
type MemoStore interface {
	// Create saves a new memo to the store.
	// It handles domain validation internally.
	// Returns validation errors from the domain Memo if data is invalid.
	Create(ctx context.Context, memo *domain.Memo) error

	// GetByID retrieves a memo by its unique ID.
	// Returns ErrMemoNotFound if the memo does not exist.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error)

	// UpdateStatus updates the status of an existing memo.
	// Returns ErrMemoNotFound if the memo does not exist.
	// Returns validation errors if the status is invalid.
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error

	// FindMemosByStatus retrieves all memos with the specified status.
	// Returns an empty slice if no memos match the criteria.
	// Can limit the number of results and paginate through offset.
	FindMemosByStatus(ctx context.Context, status domain.MemoStatus, limit, offset int) ([]*domain.Memo, error)
}
