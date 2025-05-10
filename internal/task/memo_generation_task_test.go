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

// createCardServiceMock creates a card service mock for testing
func createCardServiceMock(
	createCardsFunc func(ctx context.Context, cards []*domain.Card) error,
) *mocks.MockCardService {
	return &mocks.MockCardService{
		CreateCardsFunc: createCardsFunc,
		GetCardFunc: func(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
			return nil, nil // Default implementation
		},
	}
}

func TestNewMemoGenerationTask(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	validMemoID := uuid.New()

	t.Run("creates task with valid parameters", func(t *testing.T) {
		memoService := &mocks.MockMemoService{}
		generator := &mocks.MockGenerator{}
		cardService := createCardServiceMock(func(ctx context.Context, cards []*domain.Card) error {
			return nil
		})

		task, err := NewMemoGenerationTask(validMemoID, memoService, generator, cardService, logger)

		require.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, validMemoID, task.memoID)
		assert.Equal(t, TaskStatus(statusPending), task.Status())
		assert.Equal(t, TaskTypeMemoGeneration, task.Type())
		assert.NotEqual(t, uuid.Nil, task.ID())
	})

	t.Run("fails with nil memo service", func(t *testing.T) {
		generator := &mocks.MockGenerator{}
		cardService := createCardServiceMock(func(ctx context.Context, cards []*domain.Card) error {
			return nil
		})

		task, err := NewMemoGenerationTask(validMemoID, nil, generator, cardService, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrNilMemoService, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil generator", func(t *testing.T) {
		memoService := &mocks.MockMemoService{}
		cardService := createCardServiceMock(func(ctx context.Context, cards []*domain.Card) error {
			return nil
		})

		task, err := NewMemoGenerationTask(validMemoID, memoService, nil, cardService, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrNilGenerator, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil card service", func(t *testing.T) {
		memoService := &mocks.MockMemoService{}
		generator := &mocks.MockGenerator{}

		task, err := NewMemoGenerationTask(validMemoID, memoService, generator, nil, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrNilCardService, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil logger", func(t *testing.T) {
		memoService := &mocks.MockMemoService{}
		generator := &mocks.MockGenerator{}
		cardService := createCardServiceMock(func(ctx context.Context, cards []*domain.Card) error {
			return nil
		})

		task, err := NewMemoGenerationTask(validMemoID, memoService, generator, cardService, nil)

		assert.Error(t, err)
		assert.Equal(t, ErrNilLogger, err)
		assert.Nil(t, task)
	})

	t.Run("fails with nil memo ID", func(t *testing.T) {
		memoService := &mocks.MockMemoService{}
		generator := &mocks.MockGenerator{}
		cardService := createCardServiceMock(func(ctx context.Context, cards []*domain.Card) error {
			return nil
		})

		task, err := NewMemoGenerationTask(uuid.Nil, memoService, generator, cardService, logger)

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
	generator := &mocks.MockGenerator{}
	cardService := createCardServiceMock(func(ctx context.Context, cards []*domain.Card) error {
		return nil
	})

	task, err := NewMemoGenerationTask(validMemoID, memoService, generator, cardService, logger)
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
	generator := &mocks.MockGenerator{}
	cardService := createCardServiceMock(func(ctx context.Context, cards []*domain.Card) error {
		return nil
	})

	task, err := NewMemoGenerationTask(validMemoID, memoService, generator, cardService, logger)
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

		generator := &mocks.MockGenerator{
			GenerateCardsFunc: func(ctx context.Context, text string, userID uuid.UUID) ([]*domain.Card, error) {
				return cards, nil
			},
		}

		cardService := createCardServiceMock(func(ctx context.Context, cards []*domain.Card) error {
			return nil
		})

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardService, logger)
		require.NoError(t, err)

		// Execute the task directly now that we have proper mocks
		err = task.Execute(context.Background())

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, TaskStatus(statusCompleted), task.Status())
		assert.Equal(t, domain.MemoStatusCompleted, memo.Status)
	})

	t.Run("handles memo not found error", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()
		notFoundErr := errors.New("memo not found")

		// Setup mocks
		memoService := &mocks.MockMemoService{
			GetMemoFn: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return nil, notFoundErr
			},
		}

		generator := &mocks.MockGenerator{}
		cardService := createCardServiceMock(nil)
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardService, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.Error(t, err)
		assert.ErrorContains(t, err, "memo not found")
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles update memo status error", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()
		userID := uuid.New()
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}
		updateErr := errors.New("update status error")

		// Setup mocks
		memoService := &mocks.MockMemoService{
			GetMemoFn: func(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
				return memo, nil
			},
			UpdateMemoStatusFn: func(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error {
				return updateErr
			},
		}

		generator := &mocks.MockGenerator{}
		cardService := createCardServiceMock(nil)
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardService, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.Error(t, err)
		assert.ErrorContains(t, err, "update status error")
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
	})

	t.Run("handles generate cards error", func(t *testing.T) {
		// Setup mocks and data
		memoID := uuid.New()
		userID := uuid.New()
		memo := &domain.Memo{
			ID:     memoID,
			UserID: userID,
			Text:   "Test memo text",
			Status: domain.MemoStatusPending,
		}
		genErr := errors.New("generation error")

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

		generator := &mocks.MockGenerator{
			GenerateCardsFunc: func(ctx context.Context, text string, userID uuid.UUID) ([]*domain.Card, error) {
				return nil, genErr
			},
		}

		cardService := createCardServiceMock(nil)
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardService, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.Error(t, err)
		assert.ErrorContains(t, err, "generation error")
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
		saveErr := errors.New("save error")

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

		generator := &mocks.MockGenerator{
			GenerateCardsFunc: func(ctx context.Context, text string, userID uuid.UUID) ([]*domain.Card, error) {
				return cards, nil
			},
		}

		cardService := createCardServiceMock(func(ctx context.Context, cards []*domain.Card) error {
			return saveErr
		})

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		// Create and execute task
		task, err := NewMemoGenerationTask(memoID, memoService, generator, cardService, logger)
		require.NoError(t, err)

		err = task.Execute(context.Background())

		// Assertions
		assert.Error(t, err)
		assert.ErrorContains(t, err, "save error")
		assert.Equal(t, TaskStatus(statusFailed), task.Status())
		assert.Equal(t, domain.MemoStatusFailed, memo.Status)
	})
}
