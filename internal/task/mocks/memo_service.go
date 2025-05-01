package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// MockMemoService is a mock implementation of task.MemoService
type MockMemoService struct {
	GetMemoFn          func(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error)
	UpdateMemoStatusFn func(ctx context.Context, memoID uuid.UUID, status domain.MemoStatus) error
}

// GetMemo implements task.MemoService
func (m *MockMemoService) GetMemo(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error) {
	if m.GetMemoFn != nil {
		return m.GetMemoFn(ctx, memoID)
	}
	return nil, nil
}

// UpdateMemoStatus implements task.MemoService
func (m *MockMemoService) UpdateMemoStatus(
	ctx context.Context,
	memoID uuid.UUID,
	status domain.MemoStatus,
) error {
	if m.UpdateMemoStatusFn != nil {
		return m.UpdateMemoStatusFn(ctx, memoID, status)
	}
	return nil
}
