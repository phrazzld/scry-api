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
		memoService := &mocks.MockMemoService{}
		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(validMemoID, memoService, generator, cardRepo, logger)

		require.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, validMemoID, task.memoID)
		assert.Equal(t, TaskStatus(statusPending), task.Status())
		assert.Equal(t, TaskTypeMemoGeneration, task.Type())
		assert.NotEqual(t, uuid.Nil, task.ID())
	})

	t.Run("fails with nil memo service", func(t *testing.T) {
		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(validMemoID, nil, generator, cardRepo, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrNilMemoService, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil generator", func(t *testing.T) {
		memoService := &mocks.MockMemoService{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(validMemoID, memoService, nil, cardRepo, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrNilGenerator, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil card repository", func(t *testing.T) {
		memoService := &mocks.MockMemoService{}
		generator := &mocks.Generator{}

		task, err := NewMemoGenerationTask(validMemoID, memoService, generator, nil, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrNilCardRepository, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil logger", func(t *testing.T) {
		memoService := &mocks.MockMemoService{}
		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(validMemoID, memoService, generator, cardRepo, nil)

		assert.Error(t, err)
		assert.Equal(t, ErrNilLogger, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil memo ID", func(t *testing.T) {
		memoService := &mocks.MockMemoService{}
		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}

		task, err := NewMemoGenerationTask(uuid.Nil, memoService, generator, cardRepo, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrEmptyMemoID, err)
		assert.Nil(t, task)
	})
}

func TestMemoGenerationTaskInterface(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	validMemoID := uuid.New()
	memoService := &mocks.MockMemoService{}
	generator := &mocks.Generator{}
	cardRepo := &mocks.CardRepository{}

	task, err := NewMemoGenerationTask(validMemoID, memoService, generator, cardRepo, logger)
	require.NoError(t, err)

	// Test Task interface methods
	assert.Equal(t, validMemoID, task.memoID)
	assert.Equal(t, TaskStatus(statusPending), task.Status())
	assert.Equal(t, TaskTypeMemoGeneration, task.Type())
	assert.NotEqual(t, uuid.Nil, task.ID())
}

func TestMemoGenerationTaskPayload(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	validMemoID := uuid.New()
	memoService := &mocks.MockMemoService{}
	generator := &mocks.Generator{}
	cardRepo := &mocks.CardRepository{}

	task, err := NewMemoGenerationTask(validMemoID, memoService, generator, cardRepo, logger)
	require.NoError(t, err)

	// Test payload serialization
	payload := task.Payload()
	assert.NotEmpty(t, payload)

	// Verify payload contents
	var data memoGenerationPayload
	err = json.Unmarshal(payload, &data)
	require.NoError(t, err)
	assert.Equal(t, validMemoID, data.MemoID)
}

func TestMemoGenerationTask_Execute(t *testing.T) {
	t.Run("successfully generates cards", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()
		userID := uuid.New()
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}
		cards := []*domain.Card{
			{
				ID:      uuid.New(),
				MemoID:  memoID,
				UserID:  userID,
				Content: json.RawMessage(`{"front":"Test front","back":"Test back"}`),
			},
		}

		// Setup mocks
		memoService := &mocks.MockMemoService{
			GetMemoFn: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateMemoStatusFn: func(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error {
				memo.Status = status
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, text string, userID uuid.UUID) ([]*domain.Card, error) {
				return cards, nil
			},
		}

		cardRepo := &mocks.CardRepository{
			CreateMultipleFunc: func(ctx context.Context, cards []*domain.Card) error {
				return nil
			},
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardRepo, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, TaskStatus(statusCompleted), task.Status())
		assert.Equal(t, domain.MemoStatusCompleted, memo.Status)
	})

	t.Run("handles memo not found error", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()

		// Setup mocks
		memoService := &mocks.MockMemoService{
			GetMemoFn: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return nil, errors.New("memo not found")
			},
		}

		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardRepo, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles memo update to processing error", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()
		userID := uuid.New()
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		// Setup mocks
		memoService := &mocks.MockMemoService{
			GetMemoFn: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateMemoStatusFn: func(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error {
				return errors.New("update error")
			},
		}

		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardRepo, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles generation error", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()
		userID := uuid.New()
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		// Setup mocks
		memoService := &mocks.MockMemoService{
			GetMemoFn: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateMemoStatusFn: func(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error {
				memo.Status = status
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, text string, userID uuid.UUID) ([]*domain.Card, error) {
				return nil, errors.New("generation error")
			},
		}

		cardRepo := &mocks.CardRepository{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardRepo, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
		assert.Equal(t, domain.MemoStatusFailed, memo.Status)
	})

	t.Run("handles save cards error", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()
		userID := uuid.New()
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}
		cards := []*domain.Card{
			{
				ID:      uuid.New(),
				MemoID:  memoID,
				UserID:  userID,
				Content: json.RawMessage(`{"front":"Test front","back":"Test back"}`),
			},
		}

		// Setup mocks
		memoService := &mocks.MockMemoService{
			GetMemoFn: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateMemoStatusFn: func(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error {
				memo.Status = status
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, text string, userID uuid.UUID) ([]*domain.Card, error) {
				return cards, nil
			},
		}

		cardRepo := &mocks.CardRepository{
			CreateMultipleFunc: func(ctx context.Context, cards []*domain.Card) error {
				return errors.New("save error")
			},
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardRepo, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
		assert.Equal(t, domain.MemoStatusFailed, memo.Status)
	})

	t.Run("handles final update error but returns completed", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()
		userID := uuid.New()
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}
		cards := []*domain.Card{
			{
				ID:      uuid.New(),
				MemoID:  memoID,
				UserID:  userID,
				Content: json.RawMessage(`{"front":"Test front","back":"Test back"}`),
			},
		}

		// Setup mocks
		firstUpdateCalled := false
		memoService := &mocks.MockMemoService{
			GetMemoFn: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateMemoStatusFn: func(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error {
				if status == domain.MemoStatusCompleted && !firstUpdateCalled {
					firstUpdateCalled = true
					return errors.New("final update error")
				}
				memo.Status = status
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, text string, userID uuid.UUID) ([]*domain.Card, error) {
				return cards, nil
			},
		}

		cardRepo := &mocks.CardRepository{
			CreateMultipleFunc: func(ctx context.Context, cards []*domain.Card) error {
				return nil
			},
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardRepo, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, TaskStatus(statusCompleted), task.Status())
		assert.Equal(t, domain.MemoStatusProcessing, memo.Status) // Not updated to completed due to error
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()

		// Setup mocks
		memoService := &mocks.MockMemoService{}
		generator := &mocks.Generator{}
		cardRepo := &mocks.CardRepository{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardRepo, logger)
		require.NoError(t, err)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = task.Execute(ctx)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles no cards generated", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()
		userID := uuid.New()
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}

		// Setup mocks
		memoService := &mocks.MockMemoService{
			GetMemoFn: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateMemoStatusFn: func(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error {
				memo.Status = status
				return nil
			},
		}

		generator := &mocks.Generator{
			GenerateCardsFunc: func(ctx context.Context, text string, userID uuid.UUID) ([]*domain.Card, error) {
				return []*domain.Card{}, nil // Empty slice
			},
		}

		cardRepo := &mocks.CardRepository{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardRepo, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, TaskStatus(statusCompleted), task.Status())
		assert.Equal(t, domain.MemoStatusCompleted, memo.Status)
	})
}
