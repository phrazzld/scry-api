package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
)

// MemoRepository defines the repository interface for the service layer
// This is now aligned with store.MemoStore to ensure proper separation of concerns
type MemoRepository interface {
	// Create saves a new memo to the store
	Create(ctx context.Context, memo *domain.Memo) error

	// GetByID retrieves a memo by its unique ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error)

	// Update saves changes to an existing memo
	Update(ctx context.Context, memo *domain.Memo) error

	// WithTx returns a new repository instance that uses the provided transaction
	// This is used for transactional operations
	WithTx(tx *sql.Tx) MemoRepository

	// DB returns the underlying database connection
	DB() *sql.DB
}

// TaskRunner defines the interface for submitting background tasks
type TaskRunner interface {
	// Submit adds a task to the processing queue
	Submit(ctx context.Context, task task.Task) error
}

// MemoGenerationTaskFactory creates MemoGenerationTask instances
type MemoGenerationTaskFactory interface {
	// CreateTask creates a new MemoGenerationTask for the specified memo
	CreateTask(memoID uuid.UUID) (task.Task, error)
}

// MemoService provides memo-related operations
type MemoService interface {
	// CreateMemoAndEnqueueTask creates a new memo and enqueues it for processing
	CreateMemoAndEnqueueTask(ctx context.Context, userID uuid.UUID, text string) (*domain.Memo, error)

	// UpdateMemoStatus updates a memo's status and handles related business logic
	UpdateMemoStatus(ctx context.Context, memoID uuid.UUID, status domain.MemoStatus) error

	// GetMemo retrieves a memo by its ID
	GetMemo(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error)
}

// MemoServiceImpl implements the MemoService interface
// We're making this public so that main.go can set the task factory
type MemoServiceImpl struct {
	memoRepo    MemoRepository
	taskRunner  TaskRunner
	taskFactory MemoGenerationTaskFactory
	logger      *slog.Logger
}

// SetTaskFactory allows setting the task factory after construction
// This helps break the circular dependency between service and task packages
func (s *MemoServiceImpl) SetTaskFactory(factory MemoGenerationTaskFactory) {
	s.taskFactory = factory
}

// NewMemoService creates a new MemoService
func NewMemoService(
	memoRepo MemoRepository,
	taskRunner TaskRunner,
	taskFactory MemoGenerationTaskFactory,
	logger *slog.Logger,
) MemoService {
	return &MemoServiceImpl{
		memoRepo:    memoRepo,
		taskRunner:  taskRunner,
		taskFactory: taskFactory,
		logger:      logger.With("component", "memo_service"),
	}
}

// CreateMemoAndEnqueueTask creates a new memo with pending status and enqueues a generation task
// Uses a transaction for the memo creation part to ensure atomicity
func (s *MemoServiceImpl) CreateMemoAndEnqueueTask(
	ctx context.Context,
	userID uuid.UUID,
	text string,
) (*domain.Memo, error) {
	// 1. Create a new memo with pending status
	memo, err := domain.NewMemo(userID, text)
	if err != nil {
		s.logger.Error("failed to create memo object",
			"error", err,
			"user_id", userID)
		return nil, fmt.Errorf("failed to create memo: %w", err)
	}

	// 2. Save the memo to the database using a transaction
	err = store.RunInTransaction(ctx, s.memoRepo.DB(), func(ctx context.Context, tx *sql.Tx) error {
		// Get a transactional repo
		txRepo := s.memoRepo.WithTx(tx)

		// Create the memo within the transaction
		return txRepo.Create(ctx, memo)
	})

	if err != nil {
		s.logger.Error("failed to save memo to database",
			"error", err,
			"user_id", userID,
			"memo_id", memo.ID)
		return nil, fmt.Errorf("failed to create memo: %w", err)
	}

	s.logger.Info("memo created successfully with pending status",
		"memo_id", memo.ID,
		"user_id", userID)

	// 3. Create a MemoGenerationTask
	genTask, err := s.taskFactory.CreateTask(memo.ID)
	if err != nil {
		s.logger.Error("failed to create memo generation task",
			"error", err,
			"memo_id", memo.ID,
			"user_id", userID)
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 4. Submit the task to the task runner
	err = s.taskRunner.Submit(ctx, genTask)
	if err != nil {
		s.logger.Error("failed to enqueue memo generation task",
			"error", err,
			"memo_id", memo.ID,
			"user_id", userID,
			"task_id", genTask.ID())
		return nil, fmt.Errorf("failed to enqueue task: %w", err)
	}

	s.logger.Info("memo generation task enqueued successfully",
		"memo_id", memo.ID,
		"user_id", userID,
		"task_id", genTask.ID())

	return memo, nil
}

// GetMemo retrieves a memo by its ID
func (s *MemoServiceImpl) GetMemo(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error) {
	memo, err := s.memoRepo.GetByID(ctx, memoID)
	if err != nil {
		s.logger.Error("failed to retrieve memo",
			"error", err,
			"memo_id", memoID)
		return nil, fmt.Errorf("failed to retrieve memo: %w", err)
	}

	s.logger.Debug("retrieved memo successfully",
		"memo_id", memoID,
		"user_id", memo.UserID,
		"status", memo.Status)

	return memo, nil
}

// UpdateMemoStatus updates a memo's status and handles related business logic
// This centralizes all status transition logic in the service layer and uses transactions
// to ensure atomicity of the operation.
func (s *MemoServiceImpl) UpdateMemoStatus(ctx context.Context, memoID uuid.UUID, status domain.MemoStatus) error {
	// Use a transaction to ensure atomicity
	return store.RunInTransaction(ctx, s.memoRepo.DB(), func(ctx context.Context, tx *sql.Tx) error {
		// Get a transactional repo
		txRepo := s.memoRepo.WithTx(tx)

		// Retrieve the memo first
		memo, err := txRepo.GetByID(ctx, memoID)
		if err != nil {
			s.logger.Error("failed to retrieve memo for status update",
				"error", err,
				"memo_id", memoID,
				"target_status", status)
			return fmt.Errorf("failed to retrieve memo for status update: %w", err)
		}

		// Update the memo's status
		err = memo.UpdateStatus(status)
		if err != nil {
			s.logger.Error("failed to update memo status",
				"error", err,
				"memo_id", memoID,
				"current_status", memo.Status,
				"target_status", status)
			return fmt.Errorf("failed to update memo status to %s: %w", status, err)
		}

		// Save the updated memo using the transactional repo
		err = txRepo.Update(ctx, memo)
		if err != nil {
			s.logger.Error("failed to save updated memo status",
				"error", err,
				"memo_id", memoID,
				"status", status)
			return fmt.Errorf("failed to save memo status %s: %w", status, err)
		}

		s.logger.Info("memo status updated successfully in transaction",
			"memo_id", memoID,
			"status", status)
		return nil
	})
}
