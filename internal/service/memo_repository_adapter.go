package service

import (
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
)

// MemoRepositoryAdapter adapts a store.MemoStore to task.MemoRepository
// to decouple the task package from directly depending on store implementations
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

// Ensure MemoRepositoryAdapter implements task.MemoRepository
var _ task.MemoRepository = (*MemoRepositoryAdapter)(nil)

// Verify that MemoRepositoryAdapter implements service.MemoRepository
var _ MemoRepository = (*MemoRepositoryAdapter)(nil)
