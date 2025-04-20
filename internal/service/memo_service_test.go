package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/events"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/stretchr/testify/mock"
)

// Note: We're skipping transaction-based tests in this package since they're better suited
// for integration tests. See cmd/server/*_test.go for transaction-based testing.

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

// WithTx implements service.MemoRepository
func (m *MockMemoRepository) WithTx(tx *sql.Tx) MemoRepository {
	args := m.Called(tx)
	return args.Get(0).(MemoRepository)
}

// DB implements service.MemoRepository
func (m *MockMemoRepository) DB() *sql.DB {
	args := m.Called()
	if db, ok := args.Get(0).(*sql.DB); ok {
		return db
	}
	return nil
}

// MockTaskRunner is a mock implementation of the TaskRunner
type MockTaskRunner struct {
	mock.Mock
}

// Submit implements TaskRunner
func (m *MockTaskRunner) Submit(ctx context.Context, task task.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

// MockEventEmitter is a mock implementation of the events.EventEmitter interface
type MockEventEmitter struct {
	mock.Mock
}

// EmitEvent implements events.EventEmitter
func (m *MockEventEmitter) EmitEvent(ctx context.Context, event *events.TaskRequestEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockMemoGenerationTask is a mock implementation of a Task generated for memo processing
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
	// Test cases for transaction-based operations

	t.Run("success", func(t *testing.T) {
		// Skip test with transaction mocking - this would be tested in an integration test
		t.Skip("Skipping test that requires transaction management")
	})

	t.Run("memo creation fails", func(t *testing.T) {
		// Skip test with transaction mocking - this would be tested in an integration test
		t.Skip("Skipping test that requires transaction management")
	})

	t.Run("task creation fails", func(t *testing.T) {
		// Skip test with transaction mocking - this would be tested in an integration test
		t.Skip("Skipping test that requires transaction management")
	})

	t.Run("task enqueuing fails", func(t *testing.T) {
		// Skip test with transaction mocking - this would be tested in an integration test
		t.Skip("Skipping test that requires transaction management")
	})
}
