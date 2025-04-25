package task

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// Common errors for MemoServiceAdapter
var (
	ErrNilRepository        = errors.New("repository cannot be nil")
	ErrMissingGetByIDMethod = errors.New(
		"repository must implement GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error)",
	)
	ErrMissingUpdateMethod = errors.New(
		"repository must implement Update(ctx context.Context, memo *domain.Memo) error",
	)
	ErrRepositoryMethodsUnavailable = errors.New(
		"repository must implement both GetByID and Update methods",
	)
)

// MemoServiceAdapter adapts a repository to the MemoService interface
// This helps break circular dependencies between the task and service packages
type MemoServiceAdapter struct {
	// Store these as explicit fields since we no longer have MemoRepository in this package
	getByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Memo, error)
	updateFn  func(ctx context.Context, memo *domain.Memo) error
}

// NewMemoServiceAdapter creates a new adapter that implements MemoService
// by using a repository that has the following methods:
//
// Required methods:
// - GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error)
// - Update(ctx context.Context, memo *domain.Memo) error
//
// The adapter will use these methods to implement the MemoService interface.
// If any required method is missing, an error will be returned.
func NewMemoServiceAdapter(repo interface{}) (*MemoServiceAdapter, error) {
	// Validate repository is not nil
	if repo == nil {
		return nil, ErrNilRepository
	}

	// Extract the methods we need using type assertions
	var getByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Memo, error)
	var updateFn func(ctx context.Context, memo *domain.Memo) error

	// If repo has a GetByID method with the right signature, use it
	if repoWithGetByID, ok := repo.(interface {
		GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error)
	}); ok {
		getByIDFn = repoWithGetByID.GetByID
	} else {
		return nil, ErrMissingGetByIDMethod
	}

	// If repo has an Update method with the right signature, use it
	if repoWithUpdate, ok := repo.(interface {
		Update(ctx context.Context, memo *domain.Memo) error
	}); ok {
		updateFn = repoWithUpdate.Update
	} else {
		return nil, ErrMissingUpdateMethod
	}

	// Both methods are available, create and return the adapter
	return &MemoServiceAdapter{
		getByIDFn: getByIDFn,
		updateFn:  updateFn,
	}, nil
}

// GetMemo retrieves a memo by its ID (simple pass-through to repository)
func (a *MemoServiceAdapter) GetMemo(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error) {
	return a.getByIDFn(ctx, memoID)
}

// UpdateMemoStatus updates a memo's status
// This implements the business logic required by the task execution
func (a *MemoServiceAdapter) UpdateMemoStatus(
	ctx context.Context,
	memoID uuid.UUID,
	status domain.MemoStatus,
) error {
	// Retrieve the memo first
	memo, err := a.getByIDFn(ctx, memoID)
	if err != nil {
		return err
	}

	// Update the memo's status
	err = memo.UpdateStatus(status)
	if err != nil {
		return err
	}

	// Save the updated memo
	return a.updateFn(ctx, memo)
}

// Ensure MemoServiceAdapter implements MemoService
var _ MemoService = (*MemoServiceAdapter)(nil)
