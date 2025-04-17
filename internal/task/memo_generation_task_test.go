package task

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/task/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoGenerationTask(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	validMemoID := uuid.New()

	t.Run("creates task with valid parameters", func(t *testing.T) {
		memoRepo := &mocks.MemoRepository{}
		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(validMemoID, memoRepo, generator, cardRepo, logger)

		require.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, validMemoID, task.memoID)
		assert.Equal(t, TaskStatus(statusPending), task.Status())
		assert.Equal(t, TaskTypeMemoGeneration, task.Type())
		assert.NotEqual(t, uuid.Nil, task.ID())
	})

	t.Run("fails with nil memo repository", func(t *testing.T) {
		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(validMemoID, nil, generator, cardRepo, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrNilMemoRepository, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil generator", func(t *testing.T) {
		memoRepo := &mocks.MemoRepository{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(validMemoID, memoRepo, nil, cardRepo, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrNilGenerator, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil card repository", func(t *testing.T) {
		memoRepo := &mocks.MemoRepository{}
		generator := &mocks.Generator{}

		task, err := NewMemoGenerationTask(validMemoID, memoRepo, generator, nil, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrNilCardRepository, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil logger", func(t *testing.T) {
		memoRepo := &mocks.MemoRepository{}
		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(validMemoID, memoRepo, generator, cardRepo, nil)

		assert.Error(t, err)
		assert.Equal(t, ErrNilLogger, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil memo ID", func(t *testing.T) {
		memoRepo := &mocks.MemoRepository{}
		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(uuid.Nil, memoRepo, generator, cardRepo, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrEmptyMemoID, err)
		assert.Nil(t, task)
	})
}

func TestMemoGenerationTaskPayload(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	memoID := uuid.New()
	memoRepo := &mocks.MemoRepository{}
	generator := &mocks.Generator{}
	cardRepo := &mocks.CardRepository{}

	task, err := NewMemoGenerationTask(memoID, memoRepo, generator, cardRepo, logger)
	require.NoError(t, err)

	payload := task.Payload()
	assert.NotEmpty(t, payload)

	var decodedPayload memoGenerationPayload
	err = json.Unmarshal(payload, &decodedPayload)
	require.NoError(t, err)

	assert.Equal(t, memoID, decodedPayload.MemoID)
}

func TestMemoGenerationTaskInterface(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	memoID := uuid.New()
	memoRepo := &mocks.MemoRepository{}
	generator := &mocks.Generator{}
	cardRepo := &mocks.CardRepository{}

	task, err := NewMemoGenerationTask(memoID, memoRepo, generator, cardRepo, logger)
	require.NoError(t, err)

	// Validate the struct fields
	assert.NotEqual(t, uuid.Nil, task.id)
	assert.Equal(t, memoID, task.memoID)
	assert.NotNil(t, task.memoRepo)
	assert.NotNil(t, task.generator)
	assert.NotNil(t, task.logger)
	assert.Equal(t, statusPending, task.status)

	// Check ID method
	assert.NotEqual(t, uuid.Nil, task.ID())

	// Check Type method
	assert.Equal(t, TaskTypeMemoGeneration, task.Type())

	// Check Status method
	assert.Equal(t, TaskStatus(statusPending), task.Status())
}

func TestMemoGenerationTask_Execute(t *testing.T) {
	memoID := uuid.New()
	userID := uuid.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("successfully generates cards", func(t *testing.T) {
		// Setup mocks
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		// Card that will be "generated"
		cardContent := json.RawMessage(`{"front":"Test question","back":"Test answer"}`)
		generatedCards := []*domain.Card{
			{
				ID:      uuid.New(),
				UserID:  userID,
				MemoID:  memoID,
				Content: cardContent,
			},
		}

		// Setup repositories with expected behavior
		memoRepo := &mocks.MemoRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				assert.Equal(t, memoID, id)
				return memo, nil
			},
			UpdateFunc: func(ctx context.Context, m *domain.Memo) error {
				// First call should update to processing
				if m.Status == domain.MemoStatusProcessing {
					return nil
				}
				// Second call should update to completed
				assert.Equal(t, domain.MemoStatusCompleted, m.Status)
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, memoText string, id uuid.UUID) ([]*domain.Card, error) {
				assert.Equal(t, memo.Text, memoText)
				assert.Equal(t, userID, id)
				return generatedCards, nil
			},
		}

		cardRepo := &mocks.CardRepository{
			CreateMultipleFunc: func(ctx context.Context, cards []*domain.Card) error {
				assert.Equal(t, generatedCards, cards)
				return nil
			},
		}

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoRepo, generator, cardRepo, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, TaskStatus(statusCompleted), task.Status())
	})

	t.Run("handles memo not found error", func(t *testing.T) {
		expectedErr := errors.New("memo not found")
		memoRepo := &mocks.MemoRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return nil, expectedErr
			},
		}

		task, err := NewMemoGenerationTask(
			memoID,
			memoRepo,
			&mocks.Generator{},
			&mocks.CardRepository{},
			logger,
		)
		require.NoError(t, err)

		err = task.Execute(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles memo update to processing error", func(t *testing.T) {
		expectedErr := errors.New("update error")
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		memoRepo := &mocks.MemoRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateFunc: func(ctx context.Context, m *domain.Memo) error {
				if m.Status == domain.MemoStatusProcessing {
					return expectedErr
				}
				return nil
			},
		}

		task, err := NewMemoGenerationTask(
			memoID,
			memoRepo,
			&mocks.Generator{},
			&mocks.CardRepository{},
			logger,
		)
		require.NoError(t, err)

		err = task.Execute(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles generation error", func(t *testing.T) {
		expectedErr := errors.New("generation error")
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		memoRepo := &mocks.MemoRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateFunc: func(ctx context.Context, m *domain.Memo) error {
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, memoText string, id uuid.UUID) ([]*domain.Card, error) {
				return nil, expectedErr
			},
		}

		task, err := NewMemoGenerationTask(
			memoID,
			memoRepo,
			generator,
			&mocks.CardRepository{},
			logger,
		)
		require.NoError(t, err)

		err = task.Execute(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles save cards error", func(t *testing.T) {
		expectedErr := errors.New("save error")
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		// Card that will be "generated"
		cardContent := json.RawMessage(`{"front":"Test question","back":"Test answer"}`)
		generatedCards := []*domain.Card{
			{
				ID:      uuid.New(),
				UserID:  userID,
				MemoID:  memoID,
				Content: cardContent,
			},
		}

		memoRepo := &mocks.MemoRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateFunc: func(ctx context.Context, m *domain.Memo) error {
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, memoText string, id uuid.UUID) ([]*domain.Card, error) {
				return generatedCards, nil
			},
		}

		cardRepo := &mocks.CardRepository{
			CreateMultipleFunc: func(ctx context.Context, cards []*domain.Card) error {
				return expectedErr
			},
		}

		task, err := NewMemoGenerationTask(
			memoID,
			memoRepo,
			generator,
			cardRepo,
			logger,
		)
		require.NoError(t, err)

		err = task.Execute(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles final update error but returns completed", func(t *testing.T) {
		updateErr := errors.New("final update error")
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		// Card that will be "generated"
		cardContent := json.RawMessage(`{"front":"Test question","back":"Test answer"}`)
		generatedCards := []*domain.Card{
			{
				ID:      uuid.New(),
				UserID:  userID,
				MemoID:  memoID,
				Content: cardContent,
			},
		}

		updateCalled := 0
		memoRepo := &mocks.MemoRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateFunc: func(ctx context.Context, m *domain.Memo) error {
				updateCalled++
				// First call (to processing) succeeds
				if updateCalled == 1 {
					return nil
				}
				// Second call (to completed) fails
				return updateErr
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, memoText string, id uuid.UUID) ([]*domain.Card, error) {
				return generatedCards, nil
			},
		}

		cardRepo := &mocks.CardRepository{
			CreateMultipleFunc: func(ctx context.Context, cards []*domain.Card) error {
				return nil
			},
		}

		task, err := NewMemoGenerationTask(
			memoID,
			memoRepo,
			generator,
			cardRepo,
			logger,
		)
		require.NoError(t, err)

		// Should log the error but still return success because the cards were created
		err = task.Execute(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, TaskStatus(statusCompleted), task.Status())
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		memoRepo := &mocks.MemoRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateFunc: func(ctx context.Context, m *domain.Memo) error {
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, memoText string, id uuid.UUID) ([]*domain.Card, error) {
				// Simulate long-running task that checks context
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				return nil, nil // Should never reach this
			},
		}

		task, err := NewMemoGenerationTask(
			memoID,
			memoRepo,
			generator,
			&mocks.CardRepository{},
			logger,
		)
		require.NoError(t, err)

		// Create a context that's already canceled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = task.Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, ctx.Err())
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles no cards generated", func(t *testing.T) {
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		memoRepo := &mocks.MemoRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateFunc: func(ctx context.Context, m *domain.Memo) error {
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, memoText string, id uuid.UUID) ([]*domain.Card, error) {
				// Return empty slice
				return []*domain.Card{}, nil
			},
		}

		task, err := NewMemoGenerationTask(
			memoID,
			memoRepo,
			generator,
			&mocks.CardRepository{},
			logger,
		)
		require.NoError(t, err)

		err = task.Execute(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, TaskStatus(statusCompleted), task.Status())
	})
}
