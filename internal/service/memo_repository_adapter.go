package service

import (
	"context"

	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/task"
)

// MemoRepositoryAdapter adapts a task.MemoRepository to service.MemoRepository
// by adding the Create method
type MemoRepositoryAdapter struct {
	task.MemoRepository
	createFn func(ctx context.Context, memo *domain.Memo) error
}

// NewMemoRepositoryAdapter creates a new adapter that implements service.MemoRepository
// by combining a task.MemoRepository with a createFn function
func NewMemoRepositoryAdapter(
	repo task.MemoRepository,
	createFn func(ctx context.Context, memo *domain.Memo) error,
) *MemoRepositoryAdapter {
	return &MemoRepositoryAdapter{
		MemoRepository: repo,
		createFn:       createFn,
	}
}

// Create implements the service.MemoRepository.Create method by delegating to the createFn
func (a *MemoRepositoryAdapter) Create(ctx context.Context, memo *domain.Memo) error {
	return a.createFn(ctx, memo)
}

// Verify that MemoRepositoryAdapter implements service.MemoRepository
var _ MemoRepository = (*MemoRepositoryAdapter)(nil)
