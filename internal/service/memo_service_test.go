package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/events"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/stretchr/testify/mock"
)

// Note: We're skipping transaction-based tests in this package since they're better suited
// for integration tests. See cmd/server/*_test.go for transaction-based testing.

// MockMemoRepository is a mock implementation of the MemoRepository
type MockMemoRepository struct {
	mock.Mock
}

// GetByID implements task.MockMemoRepository and service.MemoRepository
func (m *MockMemoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	args := m.Called(ctx, id)
	memo, _ := args.Get(0).(*domain.Memo)
	return memo, args.Error(1)
}

// Update implements task.MockMemoRepository and service.MemoRepository
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

// Test NewMemoService constructor validation
func TestNewMemoService(t *testing.T) {
	tests := []struct {
		name         string
		memoRepo     MemoRepository
		taskRunner   TaskRunner
		eventEmitter events.EventEmitter
		logger       *slog.Logger
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "nil memoRepo",
			memoRepo:     nil,
			taskRunner:   &SimpleMockTaskRunner{},
			eventEmitter: &SimpleMockEventEmitter{},
			logger:       slog.Default(),
			expectError:  true,
			errorMsg:     "memoRepo",
		},
		{
			name:         "nil taskRunner",
			memoRepo:     &SimpleMockMemoRepository{},
			taskRunner:   nil,
			eventEmitter: &SimpleMockEventEmitter{},
			logger:       slog.Default(),
			expectError:  true,
			errorMsg:     "taskRunner",
		},
		{
			name:         "nil eventEmitter",
			memoRepo:     &SimpleMockMemoRepository{},
			taskRunner:   &SimpleMockTaskRunner{},
			eventEmitter: nil,
			logger:       slog.Default(),
			expectError:  true,
			errorMsg:     "eventEmitter",
		},
		{
			name:         "nil logger uses default",
			memoRepo:     &SimpleMockMemoRepository{},
			taskRunner:   &SimpleMockTaskRunner{},
			eventEmitter: &SimpleMockEventEmitter{},
			logger:       nil,
			expectError:  false,
		},
		{
			name:         "all dependencies provided",
			memoRepo:     &SimpleMockMemoRepository{},
			taskRunner:   &SimpleMockTaskRunner{},
			eventEmitter: &SimpleMockEventEmitter{},
			logger:       slog.Default(),
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewMemoService(tt.memoRepo, tt.taskRunner, tt.eventEmitter, tt.logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, service)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

// Test GetMemo method
func TestMemoService_GetMemo(t *testing.T) {
	ctx := context.Background()
	memoID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name          string
		memoID        uuid.UUID
		repoError     error
		repoMemo      *domain.Memo
		expectError   bool
		errorContains string
	}{
		{
			name:   "successful retrieval",
			memoID: memoID,
			repoMemo: &domain.Memo{
				ID:     memoID,
				UserID: userID,
				Text:   "Test memo",
				Status: domain.MemoStatusPending,
			},
			expectError: false,
		},
		{
			name:          "memo not found",
			memoID:        memoID,
			repoError:     store.ErrMemoNotFound,
			expectError:   true,
			errorContains: "",
		},
		{
			name:          "database error",
			memoID:        memoID,
			repoError:     errors.New("database connection failed"),
			expectError:   true,
			errorContains: "failed to retrieve memo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			memoRepo := &SimpleMockMemoRepository{
				getByIDError: tt.repoError,
				getByIDMemo:  tt.repoMemo,
			}
			taskRunner := &SimpleMockTaskRunner{}
			eventEmitter := &SimpleMockEventEmitter{}
			logger := slog.Default()

			service, err := NewMemoService(memoRepo, taskRunner, eventEmitter, logger)
			require.NoError(t, err)

			// Execute
			memo, err := service.GetMemo(ctx, tt.memoID)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, memo)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}

				// Check if it's a store.ErrMemoNotFound (returned directly) or MemoServiceError
				if tt.repoError == store.ErrMemoNotFound {
					assert.True(t, errors.Is(err, ErrMemoNotFound))
				} else {
					var memoSvcErr *MemoServiceError
					assert.True(t, errors.As(err, &memoSvcErr))
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, memo)
				assert.Equal(t, tt.repoMemo, memo)
			}

			// Verify method calls
			assert.True(t, memoRepo.getByIDCalled)
		})
	}
}

// Test UpdateMemoStatus method
func TestMemoService_UpdateMemoStatus(t *testing.T) {
	memoID := uuid.New()

	tests := []struct {
		name          string
		memoID        uuid.UUID
		status        domain.MemoStatus
		repoError     error
		expectError   bool
		errorContains string
	}{
		{
			name:        "successful status update",
			memoID:      memoID,
			status:      domain.MemoStatusCompleted,
			expectError: false,
		},
		{
			name:          "memo not found",
			memoID:        memoID,
			status:        domain.MemoStatusCompleted,
			repoError:     store.ErrMemoNotFound,
			expectError:   true,
			errorContains: "memo not found",
		},
		{
			name:          "database error",
			memoID:        memoID,
			status:        domain.MemoStatusCompleted,
			repoError:     errors.New("database error"),
			expectError:   true,
			errorContains: "failed to update memo status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// UpdateMemoStatus uses transactions, skip for unit tests
			t.Skip("Skipping test that requires transaction management")
		})
	}
}

// Test MemoServiceError methods
func TestMemoServiceError(t *testing.T) {
	t.Run("Error method", func(t *testing.T) {
		tests := []struct {
			name      string
			operation string
			message   string
			err       error
			expected  string
		}{
			{
				name:      "with underlying error",
				operation: "create",
				message:   "validation failed",
				err:       errors.New("invalid input"),
				expected:  "memo service create failed: validation failed: invalid input",
			},
			{
				name:      "without underlying error",
				operation: "delete",
				message:   "not found",
				err:       nil,
				expected:  "memo service delete failed: not found",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := &MemoServiceError{
					Operation: tt.operation,
					Message:   tt.message,
					Err:       tt.err,
				}

				assert.Equal(t, tt.expected, err.Error())
			})
		}
	})

	t.Run("Unwrap method", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		err := &MemoServiceError{
			Operation: "test",
			Message:   "test message",
			Err:       underlyingErr,
		}

		assert.Equal(t, underlyingErr, err.Unwrap())

		// Test with nil error
		err.Err = nil
		assert.Nil(t, err.Unwrap())
	})
}

// Test NewMemoServiceError constructor
func TestNewMemoServiceError(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		message     string
		err         error
		expectNil   bool
		expectError bool
	}{
		{
			name:        "with underlying error",
			operation:   "test_operation",
			message:     "test message",
			err:         errors.New("underlying error"),
			expectError: true,
		},
		{
			name:      "with nil error returns nil",
			operation: "test_operation",
			message:   "test message",
			err:       nil,
			expectNil: true,
		},
		{
			name:        "with store error returns sentinel",
			operation:   "get",
			message:     "retrieval failed",
			err:         store.ErrMemoNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewMemoServiceError(tt.operation, tt.message, tt.err)

			if tt.expectNil {
				assert.Nil(t, err)
			} else if tt.expectError {
				assert.Error(t, err)
				if tt.err == store.ErrMemoNotFound {
					// Should return the sentinel error directly
					assert.True(t, errors.Is(err, ErrMemoNotFound))
				} else if tt.err != nil {
					assert.True(t, errors.Is(err, tt.err))
				}
			}
		})
	}
}

// Simple mock implementations for testing (non-testify based)

type SimpleMockMemoRepository struct {
	// Method call tracking
	createCalled       bool
	getByIDCalled      bool
	updateCalled       bool
	updateStatusCalled bool
	withTxCalled       bool
	dbCalled           bool

	// Return values
	createError       error
	getByIDError      error
	getByIDMemo       *domain.Memo
	updateError       error
	updateStatusError error
	withTxReturn      MemoRepository
	dbReturn          *sql.DB
}

func (m *SimpleMockMemoRepository) Create(ctx context.Context, memo *domain.Memo) error {
	m.createCalled = true
	return m.createError
}

func (m *SimpleMockMemoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	m.getByIDCalled = true
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	return m.getByIDMemo, nil
}

func (m *SimpleMockMemoRepository) Update(ctx context.Context, memo *domain.Memo) error {
	m.updateCalled = true
	return m.updateError
}

func (m *SimpleMockMemoRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error {
	m.updateStatusCalled = true
	return m.updateStatusError
}

func (m *SimpleMockMemoRepository) WithTx(tx *sql.Tx) MemoRepository {
	m.withTxCalled = true
	if m.withTxReturn != nil {
		return m.withTxReturn
	}
	return &SimpleMockMemoRepository{}
}

func (m *SimpleMockMemoRepository) DB() *sql.DB {
	m.dbCalled = true
	return m.dbReturn
}

type SimpleMockTaskRunner struct {
	// Method call tracking
	submitCalled bool

	// Return values
	submitError error
}

func (m *SimpleMockTaskRunner) Submit(ctx context.Context, task task.Task) error {
	m.submitCalled = true
	return m.submitError
}

type SimpleMockTaskFactory struct {
	// Method call tracking
	createTaskCalled bool

	// Return values
	createTaskError error
	createTaskTask  task.Task
}

func (m *SimpleMockTaskFactory) CreateTask(memoID uuid.UUID) (task.Task, error) {
	m.createTaskCalled = true
	if m.createTaskError != nil {
		return nil, m.createTaskError
	}
	if m.createTaskTask != nil {
		return m.createTaskTask, nil
	}
	return &SimpleMockTask{id: memoID}, nil
}

type SimpleMockEventEmitter struct {
	// Method call tracking
	emitEventCalled bool

	// Return values
	emitEventError error
}

func (m *SimpleMockEventEmitter) EmitEvent(ctx context.Context, event *events.TaskRequestEvent) error {
	m.emitEventCalled = true
	return m.emitEventError
}

type SimpleMockTask struct {
	id uuid.UUID
}

func (m *SimpleMockTask) ID() uuid.UUID {
	return m.id
}

func (m *SimpleMockTask) Type() string {
	return "memo_generation"
}

func (m *SimpleMockTask) Payload() []byte {
	return []byte{}
}

func (m *SimpleMockTask) Status() task.TaskStatus {
	return task.TaskStatusPending
}

func (m *SimpleMockTask) Execute(ctx context.Context) error {
	return nil
}
