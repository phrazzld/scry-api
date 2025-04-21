package task

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMemoGenerationType = "memo_generation"

// MockMemoGenerationTaskFactory mock implementation of MemoGenerationTaskFactory
type MockMemoGenerationTaskFactory struct {
	CreateTaskFn     func(memoID uuid.UUID) (interface{}, error)
	CreateTaskCalled bool
	LastMemoID       uuid.UUID
}

func (m *MockMemoGenerationTaskFactory) CreateTask(memoID uuid.UUID) (interface{}, error) {
	m.CreateTaskCalled = true
	m.LastMemoID = memoID
	return m.CreateTaskFn(memoID)
}

// MockTaskRunner mock implementation of TaskRunner
type MockTaskRunner struct {
	SubmitFn       func(ctx context.Context, task interface{}) error
	SubmitCalled   bool
	StartFn        func() error
	StopFn         func()
	LastSubmitTask interface{}
}

func (m *MockTaskRunner) Submit(ctx context.Context, task interface{}) error {
	m.SubmitCalled = true
	m.LastSubmitTask = task
	return m.SubmitFn(ctx, task)
}

func (m *MockTaskRunner) Start() error {
	return m.StartFn()
}

func (m *MockTaskRunner) Stop() {
	m.StopFn()
}

// MockTaskWithID is a mock task implementation for testing
type MockTaskWithID struct {
	TaskID uuid.UUID
}

func (t *MockTaskWithID) ID() uuid.UUID {
	return t.TaskID
}

func TestTaskFactoryEventHandler_HandleEvent(t *testing.T) {
	// Create a minimal logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully handle memo generation event", func(t *testing.T) {
		// Create mock dependencies
		taskID := uuid.New()
		mockTask := &MockTaskWithID{TaskID: taskID}

		mockFactory := &MockMemoGenerationTaskFactory{
			CreateTaskFn: func(memoID uuid.UUID) (interface{}, error) {
				return mockTask, nil
			},
		}

		mockRunner := &MockTaskRunner{
			SubmitFn: func(ctx context.Context, task interface{}) error {
				return nil
			},
		}

		// Create the handler
		handler := NewTaskFactoryEventHandler(mockFactory, mockRunner, logger)

		// Create test data
		ctx := context.Background()
		memoID := uuid.New()

		// Create an event
		payload := map[string]string{"memo_id": memoID.String()}
		event, err := events.NewTaskRequestEvent(testMemoGenerationType, payload)
		require.NoError(t, err)

		// Test the handler
		err = handler.HandleEvent(ctx, event)
		assert.NoError(t, err)

		// Verify expectations
		assert.True(t, mockFactory.CreateTaskCalled)
		assert.Equal(t, memoID, mockFactory.LastMemoID)
		assert.True(t, mockRunner.SubmitCalled)
		assert.Equal(t, mockTask, mockRunner.LastSubmitTask)
	})

	t.Run("ignore unsupported event type", func(t *testing.T) {
		// Create mock dependencies
		mockFactory := &MockMemoGenerationTaskFactory{
			CreateTaskFn: func(memoID uuid.UUID) (interface{}, error) {
				t.Fail() // Should not be called
				return nil, nil
			},
		}

		mockRunner := &MockTaskRunner{
			SubmitFn: func(ctx context.Context, task interface{}) error {
				t.Fail() // Should not be called
				return nil
			},
		}

		// Create the handler
		handler := NewTaskFactoryEventHandler(mockFactory, mockRunner, logger)

		// Create an event with an unsupported type
		event, err := events.NewTaskRequestEvent("unsupported_type", map[string]string{"key": "value"})
		require.NoError(t, err)

		// Test the handler
		err = handler.HandleEvent(context.Background(), event)
		assert.NoError(t, err)

		// Verify factory and runner were not called
		assert.False(t, mockFactory.CreateTaskCalled)
		assert.False(t, mockRunner.SubmitCalled)
	})

	t.Run("handle invalid memo ID", func(t *testing.T) {
		// Create mock dependencies
		mockFactory := &MockMemoGenerationTaskFactory{
			CreateTaskFn: func(memoID uuid.UUID) (interface{}, error) {
				t.Fail() // Should not be called
				return nil, nil
			},
		}

		mockRunner := &MockTaskRunner{
			SubmitFn: func(ctx context.Context, task interface{}) error {
				t.Fail() // Should not be called
				return nil
			},
		}

		// Create the handler
		handler := NewTaskFactoryEventHandler(mockFactory, mockRunner, logger)

		// Create an event with an invalid memo ID
		payload := map[string]string{"memo_id": "invalid-uuid"}
		event, err := events.NewTaskRequestEvent(testMemoGenerationType, payload)
		require.NoError(t, err)

		// Test the handler
		err = handler.HandleEvent(context.Background(), event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid memo ID")

		// Verify factory and runner were not called
		assert.False(t, mockFactory.CreateTaskCalled)
		assert.False(t, mockRunner.SubmitCalled)
	})

	t.Run("handle task creation failure", func(t *testing.T) {
		// Create mock dependencies
		expectedErr := errors.New("task creation failed")

		mockFactory := &MockMemoGenerationTaskFactory{
			CreateTaskFn: func(memoID uuid.UUID) (interface{}, error) {
				return nil, expectedErr
			},
		}

		mockRunner := &MockTaskRunner{
			SubmitFn: func(ctx context.Context, task interface{}) error {
				t.Fail() // Should not be called
				return nil
			},
		}

		// Create the handler
		handler := NewTaskFactoryEventHandler(mockFactory, mockRunner, logger)

		// Create test data
		ctx := context.Background()
		memoID := uuid.New()

		// Create an event
		payload := map[string]string{"memo_id": memoID.String()}
		event, err := events.NewTaskRequestEvent(testMemoGenerationType, payload)
		require.NoError(t, err)

		// Test the handler
		err = handler.HandleEvent(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create task")

		// Verify expectations
		assert.True(t, mockFactory.CreateTaskCalled)
		assert.Equal(t, memoID, mockFactory.LastMemoID)
		assert.False(t, mockRunner.SubmitCalled)
	})

	t.Run("handle task submission failure", func(t *testing.T) {
		// Create mock dependencies
		expectedErr := errors.New("task submission failed")
		taskID := uuid.New()
		mockTask := &MockTaskWithID{TaskID: taskID}

		mockFactory := &MockMemoGenerationTaskFactory{
			CreateTaskFn: func(memoID uuid.UUID) (interface{}, error) {
				return mockTask, nil
			},
		}

		mockRunner := &MockTaskRunner{
			SubmitFn: func(ctx context.Context, task interface{}) error {
				return expectedErr
			},
		}

		// Create the handler
		handler := NewTaskFactoryEventHandler(mockFactory, mockRunner, logger)

		// Create test data
		ctx := context.Background()
		memoID := uuid.New()

		// Create an event
		payload := map[string]string{"memo_id": memoID.String()}
		event, err := events.NewTaskRequestEvent(testMemoGenerationType, payload)
		require.NoError(t, err)

		// Test the handler
		err = handler.HandleEvent(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to submit task")

		// Verify expectations
		assert.True(t, mockFactory.CreateTaskCalled)
		assert.Equal(t, memoID, mockFactory.LastMemoID)
		assert.True(t, mockRunner.SubmitCalled)
		assert.Equal(t, mockTask, mockRunner.LastSubmitTask)
	})
}
