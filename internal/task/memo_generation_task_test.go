package task

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
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

	// Execute method is not fully implemented yet but should not panic
	ctx := context.Background()
	err = task.Execute(ctx)
	assert.Error(t, err) // Should return "not implemented" error
	assert.Equal(t, "not implemented", err.Error())
}

// TestExecuteReturnsNotImplemented verifies that the Execute method returns the expected error
func TestExecuteReturnsNotImplemented(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	memoID := uuid.New()

	task, err := NewMemoGenerationTask(
		memoID,
		&mocks.MemoRepository{},
		&mocks.Generator{},
		&mocks.CardRepository{},
		logger,
	)
	require.NoError(t, err)

	err = task.Execute(context.Background())
	assert.Error(t, err)
	assert.Equal(t, errors.New("not implemented"), err)
}
