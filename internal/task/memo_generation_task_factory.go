package task

import (
	"log/slog"

	"github.com/google/uuid"
)

// MemoGenerationTaskFactory creates MemoGenerationTask instances
type MemoGenerationTaskFactory struct {
	memoRepo  MemoRepository
	generator Generator
	cardRepo  CardRepository
	logger    *slog.Logger
}

// NewMemoGenerationTaskFactory creates a new factory for MemoGenerationTasks
func NewMemoGenerationTaskFactory(
	memoRepo MemoRepository,
	generator Generator,
	cardRepo CardRepository,
	logger *slog.Logger,
) *MemoGenerationTaskFactory {
	return &MemoGenerationTaskFactory{
		memoRepo:  memoRepo,
		generator: generator,
		cardRepo:  cardRepo,
		logger:    logger.With("component", "memo_generation_task_factory"),
	}
}

// CreateTask creates a new MemoGenerationTask for the specified memo
func (f *MemoGenerationTaskFactory) CreateTask(memoID uuid.UUID) (Task, error) {
	task, err := NewMemoGenerationTask(
		memoID,
		f.memoRepo,
		f.generator,
		f.cardRepo,
		f.logger,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}
