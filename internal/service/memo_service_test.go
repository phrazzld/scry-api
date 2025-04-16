package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMemoRepository is a mock implementation of the MemoRepository
type MockMemoRepository struct {
	mock.Mock
}

// GetByID implements task.MemoRepository and service.MemoRepository
func (m *MockMemoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	args := m.Called(ctx, id)
	memo, _ := args.Get(0).(*domain.Memo)
	return memo, args.Error(1)
}

// Update implements task.MemoRepository and service.MemoRepository
func (m *MockMemoRepository) Update(ctx context.Context, memo *domain.Memo) error {
	args := m.Called(ctx, memo)
	return args.Error(0)
}

// Create implements service.MemoRepository
func (m *MockMemoRepository) Create(ctx context.Context, memo *domain.Memo) error {
	args := m.Called(ctx, memo)
	return args.Error(0)
}

// MockTaskRunner is a mock implementation of the TaskRunner
type MockTaskRunner struct {
	mock.Mock
}

func (m *MockTaskRunner) Submit(ctx context.Context, task task.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

// MockMemoGenerationTaskFactory is a mock factory for creating MemoGenerationTask instances
type MockMemoGenerationTaskFactory struct {
	mock.Mock
}

func (m *MockMemoGenerationTaskFactory) CreateTask(memoID uuid.UUID) (task.Task, error) {
	args := m.Called(memoID)
	task, _ := args.Get(0).(task.Task)
	return task, args.Error(1)
}

// MockMemoGenerationTask is a mock implementation of the Task interface
type MockMemoGenerationTask struct {
	mock.Mock
	id     uuid.UUID
	status task.TaskStatus
}

func NewMockMemoGenerationTask() *MockMemoGenerationTask {
	return &MockMemoGenerationTask{
		id:     uuid.New(),
		status: task.TaskStatusPending,
	}
}

func (m *MockMemoGenerationTask) ID() uuid.UUID {
	return m.id
}

func (m *MockMemoGenerationTask) Type() string {
	return task.TaskTypeMemoGeneration
}

func (m *MockMemoGenerationTask) Status() task.TaskStatus {
	return m.status
}

func (m *MockMemoGenerationTask) Payload() []byte {
	return []byte{}
}

func (m *MockMemoGenerationTask) Execute(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestMemoService_CreateMemoAndEnqueueTask(t *testing.T) {
	// Common test setup
	userID := uuid.New()
	memoText := "This is a test memo"
	logger := slog.Default()

	t.Run("success", func(t *testing.T) {
		// Setup mocks
		memoRepo := &MockMemoRepository{}
		taskRunner := &MockTaskRunner{}
		taskFactory := &MockMemoGenerationTaskFactory{}
		mockTask := NewMockMemoGenerationTask()

		// Configure mock behavior
		memoRepo.On("Create", mock.Anything, mock.MatchedBy(func(memo *domain.Memo) bool {
			return memo.UserID == userID && memo.Text == memoText && memo.Status == domain.MemoStatusPending
		})).Return(nil)

		taskFactory.On("CreateTask", mock.MatchedBy(func(memoID uuid.UUID) bool {
			return memoID != uuid.Nil
		})).Return(mockTask, nil)

		taskRunner.On("Submit", mock.Anything, mockTask).Return(nil)

		// Create service
		service := NewMemoService(memoRepo, taskRunner, taskFactory, logger)

		// Call service method
		memo, err := service.CreateMemoAndEnqueueTask(context.Background(), userID, memoText)

		// Assertions
		require.NoError(t, err)
		assert.NotNil(t, memo)
		assert.Equal(t, userID, memo.UserID)
		assert.Equal(t, memoText, memo.Text)
		assert.Equal(t, domain.MemoStatusPending, memo.Status)

		// Verify mocks
		memoRepo.AssertExpectations(t)
		taskFactory.AssertExpectations(t)
		taskRunner.AssertExpectations(t)
	})

	t.Run("memo creation fails", func(t *testing.T) {
		// Setup mocks
		memoRepo := &MockMemoRepository{}
		taskRunner := &MockTaskRunner{}
		taskFactory := &MockMemoGenerationTaskFactory{}

		// Configure mock behavior - simulate DB error during memo creation
		expectedError := errors.New("database error")
		memoRepo.On("Create", mock.Anything, mock.MatchedBy(func(memo *domain.Memo) bool {
			return memo.UserID == userID && memo.Text == memoText
		})).Return(expectedError)

		// Create service
		service := NewMemoService(memoRepo, taskRunner, taskFactory, logger)

		// Call service method
		memo, err := service.CreateMemoAndEnqueueTask(context.Background(), userID, memoText)

		// Assertions
		require.Error(t, err)
		assert.Nil(t, memo)
		assert.ErrorContains(t, err, "failed to create memo")

		// Verify that no task was created or submitted
		taskFactory.AssertNotCalled(t, "CreateTask", mock.Anything)
		taskRunner.AssertNotCalled(t, "Submit", mock.Anything, mock.Anything)
	})

	t.Run("task creation fails", func(t *testing.T) {
		// Setup mocks
		memoRepo := &MockMemoRepository{}
		taskRunner := &MockTaskRunner{}
		taskFactory := &MockMemoGenerationTaskFactory{}

		// Configure mock behavior
		memoRepo.On("Create", mock.Anything, mock.MatchedBy(func(memo *domain.Memo) bool {
			return memo.UserID == userID && memo.Text == memoText
		})).Return(nil)

		// Simulate error during task creation
		expectedError := errors.New("task creation error")
		taskFactory.On("CreateTask", mock.MatchedBy(func(memoID uuid.UUID) bool {
			return memoID != uuid.Nil
		})).Return(nil, expectedError)

		// Create service
		service := NewMemoService(memoRepo, taskRunner, taskFactory, logger)

		// Call service method
		memo, err := service.CreateMemoAndEnqueueTask(context.Background(), userID, memoText)

		// Assertions
		require.Error(t, err)
		assert.Nil(t, memo)
		assert.ErrorContains(t, err, "failed to create task")

		// Verify that no task was submitted
		taskRunner.AssertNotCalled(t, "Submit", mock.Anything, mock.Anything)
	})

	t.Run("task enqueuing fails", func(t *testing.T) {
		// Setup mocks
		memoRepo := &MockMemoRepository{}
		taskRunner := &MockTaskRunner{}
		taskFactory := &MockMemoGenerationTaskFactory{}
		mockTask := NewMockMemoGenerationTask()

		// Configure mock behavior
		memoRepo.On("Create", mock.Anything, mock.MatchedBy(func(memo *domain.Memo) bool {
			return memo.UserID == userID && memo.Text == memoText
		})).Return(nil)

		taskFactory.On("CreateTask", mock.MatchedBy(func(memoID uuid.UUID) bool {
			return memoID != uuid.Nil
		})).Return(mockTask, nil)

		// Simulate error during task submission
		expectedError := errors.New("queue full")
		taskRunner.On("Submit", mock.Anything, mockTask).Return(expectedError)

		// Create service
		service := NewMemoService(memoRepo, taskRunner, taskFactory, logger)

		// Call service method
		memo, err := service.CreateMemoAndEnqueueTask(context.Background(), userID, memoText)

		// Assertions
		require.Error(t, err)
		assert.Nil(t, memo)
		assert.ErrorContains(t, err, "failed to enqueue task")
	})
}
