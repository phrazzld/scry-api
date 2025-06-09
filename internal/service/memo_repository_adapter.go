package service

import (
	"database/sql"

	"github.com/phrazzld/scry-api/internal/store"
)

// MemoRepositoryAdapter adapts a store.MemoStore to the service.MemoRepository interface
// This enables proper dependency injection and separation of concerns
type MemoRepositoryAdapter struct {
	store.MemoStore
	db *sql.DB
}

// NewMemoRepositoryAdapter creates a new adapter that implements task.MockMemoRepository
// by delegating to a store.MemoStore implementation
func NewMemoRepositoryAdapter(
	memoStore store.MemoStore,
	db *sql.DB,
) *MemoRepositoryAdapter {
	return &MemoRepositoryAdapter{
		MemoStore: memoStore,
		db:        db,
	}
}

// WithTx returns a new repository instance that uses the provided transaction
func (a *MemoRepositoryAdapter) WithTx(tx *sql.Tx) MemoRepository {
	return &MemoRepositoryAdapter{
		MemoStore: a.MemoStore.WithTx(tx),
		db:        a.db,
	}
}

// DB returns the underlying database connection
func (a *MemoRepositoryAdapter) DB() *sql.DB {
	return a.db
}

// Ensure MemoRepositoryAdapter implements interfaces properly

// Verify that MemoRepositoryAdapter implements service.MemoRepository
var _ MemoRepository = (*MemoRepositoryAdapter)(nil)
