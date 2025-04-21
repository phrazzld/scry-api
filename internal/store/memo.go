package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// MemoStore defines the interface for memo data persistence.
// Version: 1.0
type MemoStore interface {
	// Create saves a new memo to the store.
	// It handles domain validation internally.
	// Returns validation errors from the domain Memo if data is invalid.
	Create(ctx context.Context, memo *domain.Memo) error

	// GetByID retrieves a memo by its unique ID.
	// Returns ErrMemoNotFound if the memo does not exist.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error)

	// Update saves changes to an existing memo.
	// Returns ErrMemoNotFound if the memo does not exist.
	// Returns validation errors if the memo data is invalid.
	Update(ctx context.Context, memo *domain.Memo) error

	// UpdateStatus updates the status of an existing memo.
	// Returns ErrMemoNotFound if the memo does not exist.
	// Returns validation errors if the status is invalid.
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error

	// FindMemosByStatus retrieves all memos with the specified status.
	// Returns an empty slice if no memos match the criteria.
	// Can limit the number of results and paginate through offset.
	FindMemosByStatus(ctx context.Context, status domain.MemoStatus, limit, offset int) ([]*domain.Memo, error)

	// WithTx returns a new MemoStore instance that uses the provided transaction.
	// This allows for multiple operations to be executed within a single transaction.
	// The transaction should be created and managed by the caller (typically a service).
	WithTx(tx *sql.Tx) MemoStore
}
