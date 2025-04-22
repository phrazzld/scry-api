package service_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/events"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// MockFailingMemoRepository is a specialized mock that can be configured to fail at specific points
type MockFailingMemoRepository struct {
	mock.Mock
	MemoStore       store.MemoStore
	FailOnCreate    bool
	FailOnUpdate    bool
	FailOnGetByID   bool
	FailAfterCreate bool    // Special flag to fail after a successful create operation
	dbConn          *sql.DB // Renamed to avoid naming conflict with DB method
}

func (m *MockFailingMemoRepository) Create(ctx context.Context, memo *domain.Memo) error {
	if m.FailOnCreate {
		return errors.New("simulated create failure")
	}
	err := m.MemoStore.Create(ctx, memo)
	if err != nil {
		return err
	}
	if m.FailAfterCreate {
		return errors.New("simulated failure after successful create")
	}
	return nil
}

func (m *MockFailingMemoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	if m.FailOnGetByID {
		return nil, errors.New("simulated GetByID failure")
	}
	return m.MemoStore.GetByID(ctx, id)
}

func (m *MockFailingMemoRepository) Update(ctx context.Context, memo *domain.Memo) error {
	if m.FailOnUpdate {
		return errors.New("simulated update failure")
	}
	return m.MemoStore.Update(ctx, memo)
}

func (m *MockFailingMemoRepository) WithTx(tx *sql.Tx) service.MemoRepository {
	// Return a new instance with the transaction set
	return &MockFailingMemoRepository{
		MemoStore:       m.MemoStore.WithTx(tx),
		FailOnCreate:    m.FailOnCreate,
		FailOnUpdate:    m.FailOnUpdate,
		FailOnGetByID:   m.FailOnGetByID,
		FailAfterCreate: m.FailAfterCreate,
		dbConn:          m.dbConn,
	}
}

func (m *MockFailingMemoRepository) DB() *sql.DB {
	return m.dbConn
}

// MockTaskRunner for task submission
type MockTaskRunner struct {
	mock.Mock
}

func (m *MockTaskRunner) Submit(ctx context.Context, task task.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

// MockEventEmitter implements the events.EventEmitter interface for testing
type MockEventEmitter struct {
	mock.Mock
}

func (m *MockEventEmitter) EmitEvent(ctx context.Context, event *events.TaskRequestEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockMemoTask is a simple mock task implementation
type MockMemoTask struct {
	mock.Mock
	id uuid.UUID
}

func NewMockMemoTask() *MockMemoTask {
	return &MockMemoTask{id: uuid.New()}
}

func (m *MockMemoTask) ID() uuid.UUID {
	return m.id
}

func (m *MockMemoTask) Type() string {
	return task.TaskTypeMemoGeneration
}

func (m *MockMemoTask) Status() task.TaskStatus {
	return task.TaskStatusPending
}

func (m *MockMemoTask) Payload() []byte {
	return []byte(`{"memo_id":"` + m.id.String() + `"}`)
}

func (m *MockMemoTask) Execute(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestMemoService_CreateMemoAndEnqueueTask_Atomicity tests that memo creation is atomic
func TestMemoService_CreateMemoAndEnqueueTask_Atomicity(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	testutils.WithTx(t, db, func(tx store.DBTX) {
		ctx := context.Background()
		logger := slog.Default()

		// Create a user for testing
		userEmail := "memo-tx-test@example.com"
		userID := testutils.MustInsertUser(ctx, t, tx, userEmail, bcrypt.MinCost)

		// Setup base memo store with transaction
		memoStore := postgres.NewPostgresMemoStore(tx, logger)

		t.Run("Transaction_Rollback_On_Failure", func(t *testing.T) {
			// Create a failing repository that first creates the memo but then fails
			failingRepo := &MockFailingMemoRepository{
				MemoStore:       memoStore,
				FailAfterCreate: true, // Fail after successfully creating the memo
				dbConn:          db,   // Need the real DB for transaction management
			}

			// Create mocks for tasks
			mockRunner := new(MockTaskRunner)
			mockEventEmitter := new(MockEventEmitter)

			// Setup expectations
			mockEventEmitter.On("EmitEvent", mock.Anything, mock.Anything).Return(nil) // Should never be called

			// Create service with the failing repository
			memoService, err := service.NewMemoService(failingRepo, mockRunner, mockEventEmitter, logger)
			require.NoError(t, err, "Failed to create memo service")

			// Attempt to create a memo - this should fail after committing the memo to DB but before committing the transaction
			memoText := "Test memo for rollback verification"
			memo, err := memoService.CreateMemoAndEnqueueTask(ctx, userID, memoText)

			// Verify the operation failed
			assert.Error(t, err, "Operation should fail")
			assert.Nil(t, memo, "No memo should be returned")

			// Verify the memo was NOT actually persisted due to transaction rollback
			var count int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM memos WHERE user_id = $1 AND text = $2",
				userID, memoText,
			).Scan(&count)
			require.NoError(t, err, "Failed to count memos")
			assert.Equal(t, 0, count, "No memo should exist in the database due to transaction rollback")

			// Verify event emission was never called
			mockEventEmitter.AssertNotCalled(t, "EmitEvent", mock.Anything, mock.Anything)
		})

		t.Run("Transaction_Commit_On_Success", func(t *testing.T) {
			// Create a succeeding repository
			successRepo := &MockFailingMemoRepository{
				MemoStore: memoStore,
				dbConn:    db,
			}

			// Create mocks for tasks
			mockRunner := new(MockTaskRunner)
			mockEventEmitter := new(MockEventEmitter)

			// Setup expectations
			mockEventEmitter.On("EmitEvent", mock.Anything, mock.Anything).Return(nil)

			// Create service with the succeeding repository
			memoService, err := service.NewMemoService(successRepo, mockRunner, mockEventEmitter, logger)
			require.NoError(t, err, "Failed to create memo service")

			// Create a memo - this should succeed
			memoText := "Test memo for commit verification"
			memo, err := memoService.CreateMemoAndEnqueueTask(ctx, userID, memoText)

			// Verify the operation succeeded
			assert.NoError(t, err, "Operation should succeed")
			assert.NotNil(t, memo, "Memo should be returned")

			// Verify the memo was actually persisted
			var count int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM memos WHERE id = $1 AND user_id = $2 AND text = $3",
				memo.ID, userID, memoText,
			).Scan(&count)
			require.NoError(t, err, "Failed to count memos")
			assert.Equal(t, 1, count, "Memo should exist in the database")

			// Verify event emission was called
			mockEventEmitter.AssertCalled(t, "EmitEvent", mock.Anything, mock.Anything)
		})
	})
}

// TestMemoService_UpdateMemoStatus_Atomicity tests the atomicity of memo status updates
func TestMemoService_UpdateMemoStatus_Atomicity(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	testutils.WithTx(t, db, func(tx store.DBTX) {
		ctx := context.Background()
		logger := slog.Default()

		// Create a user for testing
		userEmail := "memo-status-tx-test@example.com"
		userID := testutils.MustInsertUser(ctx, t, tx, userEmail, bcrypt.MinCost)

		// Setup base memo store with transaction
		memoStore := postgres.NewPostgresMemoStore(tx, logger)

		// Create mocks for task components (not used in these tests)
		mockRunner := new(MockTaskRunner)
		mockEventEmitter := new(MockEventEmitter)

		// Create a test memo directly
		memoText := "Memo for status update transaction test"
		memo := &domain.Memo{
			ID:        uuid.New(),
			UserID:    userID,
			Text:      memoText,
			Status:    domain.MemoStatusPending,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		// Save the memo
		err := memoStore.Create(ctx, memo)
		require.NoError(t, err, "Failed to create test memo")

		t.Run("Transaction_Rollback_On_Update_Failure", func(t *testing.T) {
			// Create a failing repository that fails on update
			failingRepo := &MockFailingMemoRepository{
				MemoStore:    memoStore,
				FailOnUpdate: true, // Fail during update
				dbConn:       db,   // Need the real DB for transaction management
			}

			// Create service with the failing repository
			memoService, err := service.NewMemoService(failingRepo, mockRunner, mockEventEmitter, logger)
			require.NoError(t, err, "Failed to create memo service")

			// Attempt to update the memo status - this should fail
			updateErr := memoService.UpdateMemoStatus(ctx, memo.ID, domain.MemoStatusProcessing)

			// Verify the operation failed
			assert.Error(t, updateErr, "Operation should fail")
			assert.Contains(t, updateErr.Error(), "simulated update failure", "Error should be from our mock")

			// Verify the memo status was NOT changed due to transaction rollback
			var status string
			err = tx.QueryRowContext(ctx,
				"SELECT status FROM memos WHERE id = $1",
				memo.ID,
			).Scan(&status)
			require.NoError(t, err, "Failed to get memo status")
			assert.Equal(t, string(domain.MemoStatusPending), status,
				"Memo status should remain unchanged due to transaction rollback")
		})

		t.Run("Transaction_Rollback_On_GetByID_Failure", func(t *testing.T) {
			// Create a failing repository that fails on GetByID
			failingRepo := &MockFailingMemoRepository{
				MemoStore:     memoStore,
				FailOnGetByID: true, // Fail during retrieval
				dbConn:        db,   // Need the real DB for transaction management
			}

			// Create service with the failing repository
			memoService, err := service.NewMemoService(failingRepo, mockRunner, mockEventEmitter, logger)
			require.NoError(t, err, "Failed to create memo service")

			// Attempt to update the memo status - this should fail during GetByID
			updateErr := memoService.UpdateMemoStatus(ctx, memo.ID, domain.MemoStatusProcessing)

			// Verify the operation failed
			assert.Error(t, updateErr, "Operation should fail")
			assert.Contains(t, updateErr.Error(), "simulated GetByID failure", "Error should be from our mock")

			// Verify the memo status was NOT changed due to transaction rollback
			var status string
			err = tx.QueryRowContext(ctx,
				"SELECT status FROM memos WHERE id = $1",
				memo.ID,
			).Scan(&status)
			require.NoError(t, err, "Failed to get memo status")
			assert.Equal(t, string(domain.MemoStatusPending), status,
				"Memo status should remain unchanged due to transaction rollback")
		})

		t.Run("Transaction_Commit_On_Success", func(t *testing.T) {
			// Create a succeeding repository
			successRepo := &MockFailingMemoRepository{
				MemoStore: memoStore,
				dbConn:    db,
			}

			// Create service with the succeeding repository
			memoService, err := service.NewMemoService(successRepo, mockRunner, mockEventEmitter, logger)
			require.NoError(t, err, "Failed to create memo service")

			// Update the memo status - this should succeed
			updateErr := memoService.UpdateMemoStatus(ctx, memo.ID, domain.MemoStatusProcessing)

			// Verify the operation succeeded
			assert.NoError(t, updateErr, "Operation should succeed")

			// Verify the memo status was actually updated
			var status string
			err = tx.QueryRowContext(ctx,
				"SELECT status FROM memos WHERE id = $1",
				memo.ID,
			).Scan(&status)
			require.NoError(t, err, "Failed to get memo status")
			assert.Equal(t, string(domain.MemoStatusProcessing), status, "Memo status should be updated")
		})
	})
}

// TestComplexTransactionWithMultipleStores verifies that transactions can span multiple stores
func TestComplexTransactionWithMultipleStores(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	// Custom transaction function for testing multiple operations in a transaction
	runInTransaction := func(ctx context.Context, t *testing.T, memoID uuid.UUID) error {
		return store.RunInTransaction(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
			// Create both stores with the same transaction
			userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for testing
			memoStore := postgres.NewPostgresMemoStore(tx, slog.Default())

			// 1. Create a user
			user, err := domain.NewUser(fmt.Sprintf("complex-tx-%s@test.com", uuid.New().String()), "password123")
			if err != nil {
				return err
			}
			if err := userStore.Create(ctx, user); err != nil {
				return err
			}

			// 2. Create a memo for this user
			memo := &domain.Memo{
				ID:        memoID,
				UserID:    user.ID,
				Text:      "This is a memo created in a complex transaction",
				Status:    domain.MemoStatusPending,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := memoStore.Create(ctx, memo); err != nil {
				return err
			}

			// 3. Simulate a failure after both operations to test atomicity
			// This error should cause both the user and memo creations to be rolled back
			return errors.New("simulated failure after complex transaction")
		})
	}

	testutils.WithTx(t, db, func(tx store.DBTX) {
		ctx := context.Background()
		memoID := uuid.New()

		// Run transaction that will create both a user and memo, but then fail
		err := runInTransaction(ctx, t, memoID)
		assert.Error(t, err, "Transaction should fail with our simulated error")
		assert.Contains(t, err.Error(), "simulated failure", "Error should be our simulated error")

		// Verify that no memo was persisted due to transaction rollback
		var memoCount int
		err = tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM memos WHERE id = $1",
			memoID,
		).Scan(&memoCount)
		require.NoError(t, err, "Failed to count memos")
		assert.Equal(t, 0, memoCount, "No memo should exist due to transaction rollback")

		// Verify that no user was persisted due to transaction rollback
		// Here we're checking for the existence of a user with email matching our pattern
		var userCount int
		err = tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM users WHERE email LIKE 'complex-tx-%'",
		).Scan(&userCount)
		require.NoError(t, err, "Failed to count users")
		assert.Equal(t, 0, userCount, "No user should exist due to transaction rollback")
	})
}
