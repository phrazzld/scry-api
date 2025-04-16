package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/task"
)

// MemoRepository extends task.MemoRepository with additional operations
type MemoRepository interface {
	task.MemoRepository

	// Create saves a new memo to the store
	Create(ctx context.Context, memo *domain.Memo) error
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
}

// memoServiceImpl implements the MemoService interface
type memoServiceImpl struct {
	memoRepo    MemoRepository
	taskRunner  TaskRunner
	taskFactory MemoGenerationTaskFactory
	logger      *slog.Logger
}

// NewMemoService creates a new MemoService
func NewMemoService(
	memoRepo MemoRepository,
	taskRunner TaskRunner,
	taskFactory MemoGenerationTaskFactory,
	logger *slog.Logger,
) MemoService {
	return &memoServiceImpl{
		memoRepo:    memoRepo,
		taskRunner:  taskRunner,
		taskFactory: taskFactory,
		logger:      logger.With("component", "memo_service"),
	}
}

// CreateMemoAndEnqueueTask creates a new memo with pending status and enqueues a generation task
func (s *memoServiceImpl) CreateMemoAndEnqueueTask(
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

	// 2. Save the memo to the database
	err = s.memoRepo.Create(ctx, memo)
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
