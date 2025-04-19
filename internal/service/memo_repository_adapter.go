package service

import (
	"github.com/phrazzld/scry-api/internal/store"
)

// MemoRepositoryAdapter adapts a store.MemoStore to the service.MemoRepository interface
// This enables proper dependency injection and separation of concerns
type MemoRepositoryAdapter struct {
	store.MemoStore
}

// NewMemoRepositoryAdapter creates a new adapter that implements task.MemoRepository
// by delegating to a store.MemoStore implementation
func NewMemoRepositoryAdapter(
	memoStore store.MemoStore,
) *MemoRepositoryAdapter {
	return &MemoRepositoryAdapter{
		MemoStore: memoStore,
	}
}

// Ensure MemoRepositoryAdapter implements interfaces properly

// Verify that MemoRepositoryAdapter implements service.MemoRepository
var _ MemoRepository = (*MemoRepositoryAdapter)(nil)
