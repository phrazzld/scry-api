package task

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validRepository is a test implementation that provides all required methods
type validRepository struct {
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Memo, error)
	updateFunc  func(ctx context.Context, memo *domain.Memo) error
}

func (r *validRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	return r.getByIDFunc(ctx, id)
}

func (r *validRepository) Update(ctx context.Context, memo *domain.Memo) error {
	return r.updateFunc(ctx, memo)
}

// missingGetByIDRepository is a test implementation that is missing the GetByID method
type missingGetByIDRepository struct {
	updateFunc func(ctx context.Context, memo *domain.Memo) error
}

func (r *missingGetByIDRepository) Update(ctx context.Context, memo *domain.Memo) error {
	return r.updateFunc(ctx, memo)
}

// missingUpdateRepository is a test implementation that is missing the Update method
type missingUpdateRepository struct {
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Memo, error)
}

func (r *missingUpdateRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	return r.getByIDFunc(ctx, id)
}

func TestNewMemoServiceAdapter(t *testing.T) {
	t.Run("valid repository", func(t *testing.T) {
		// Arrange
		repo := &validRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return &domain.Memo{ID: id}, nil
			},
			updateFunc: func(ctx context.Context, memo *domain.Memo) error {
				return nil
			},
		}

		// Act
		adapter, err := NewMemoServiceAdapter(repo)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, adapter)
		require.NotNil(t, adapter.getByIDFn)
		require.NotNil(t, adapter.updateFn)

		// Verify the adapter works
		memo, err := adapter.GetMemo(context.Background(), uuid.New())
		assert.NoError(t, err)
		assert.NotNil(t, memo)
	})

	t.Run("nil repository", func(t *testing.T) {
		// Act
		adapter, err := NewMemoServiceAdapter(nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, adapter)
		assert.Equal(t, ErrNilRepository, err)
	})

	t.Run("missing GetByID method", func(t *testing.T) {
		// Arrange
		repo := &missingGetByIDRepository{
			updateFunc: func(ctx context.Context, memo *domain.Memo) error {
				return nil
			},
		}

		// Act
		adapter, err := NewMemoServiceAdapter(repo)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, adapter)
		assert.Equal(t, ErrMissingGetByIDMethod, err)
	})

	t.Run("missing Update method", func(t *testing.T) {
		// Arrange
		repo := &missingUpdateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return &domain.Memo{ID: id}, nil
			},
		}

		// Act
		adapter, err := NewMemoServiceAdapter(repo)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, adapter)
		assert.Equal(t, ErrMissingUpdateMethod, err)
	})

	t.Run("adapter behavior - success", func(t *testing.T) {
		// Arrange
		testMemoID := uuid.New()
		testMemo := &domain.Memo{
			ID:     testMemoID,
			Status: domain.MemoStatusPending,
		}

		repo := &validRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				if id == testMemoID {
					return testMemo, nil
				}
				return nil, errors.New("not found")
			},
			updateFunc: func(ctx context.Context, memo *domain.Memo) error {
				testMemo = memo // Save the updated memo
				return nil
			},
		}

		adapter, err := NewMemoServiceAdapter(repo)
		require.NoError(t, err)
		require.NotNil(t, adapter)

		// Act
		err = adapter.UpdateMemoStatus(context.Background(), testMemoID, domain.MemoStatusProcessing)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, domain.MemoStatusProcessing, testMemo.Status)
	})

	t.Run("adapter behavior - memo not found", func(t *testing.T) {
		// Arrange
		testMemoID := uuid.New()
		notFoundErr := errors.New("memo not found")

		repo := &validRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return nil, notFoundErr
			},
			updateFunc: func(ctx context.Context, memo *domain.Memo) error {
				return nil
			},
		}

		adapter, err := NewMemoServiceAdapter(repo)
		require.NoError(t, err)
		require.NotNil(t, adapter)

		// Act
		err = adapter.UpdateMemoStatus(context.Background(), testMemoID, domain.MemoStatusProcessing)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, notFoundErr, err)
	})

	t.Run("adapter behavior - invalid status from repo", func(t *testing.T) {
		// Arrange
		testMemoID := uuid.New()
		updateErr := errors.New("invalid status update error")
		testMemo := &domain.Memo{
			ID:     testMemoID,
			Status: domain.MemoStatusPending,
		}

		repo := &validRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return testMemo, nil
			},
			updateFunc: func(ctx context.Context, memo *domain.Memo) error {
				return updateErr // Simulate error from repo's Update method
			},
		}

		adapter, err := NewMemoServiceAdapter(repo)
		require.NoError(t, err)
		require.NotNil(t, adapter)

		// Act
		err = adapter.UpdateMemoStatus(context.Background(), testMemoID, domain.MemoStatusProcessing)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, updateErr, err) // Should return the error from repo's Update method
	})
}
