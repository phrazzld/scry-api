package task

import (
	"log/slog"

	"github.com/google/uuid"
)

// MemoGenerationTaskFactory creates MemoGenerationTask instances
type MemoGenerationTaskFactory struct {
	memoService MemoService
	generator   Generator
	cardService CardService
	logger      *slog.Logger
}

// NewMemoGenerationTaskFactory creates a new factory for MemoGenerationTasks
func NewMemoGenerationTaskFactory(
	memoService MemoService,
	generator Generator,
	cardService CardService,
	logger *slog.Logger,
) *MemoGenerationTaskFactory {
	return &MemoGenerationTaskFactory{
		memoService: memoService,
		generator:   generator,
		cardService: cardService,
		logger:      logger.With("component", "memo_generation_task_factory"),
	}
}

// CreateTask creates a new MemoGenerationTask for the specified memo
func (f *MemoGenerationTaskFactory) CreateTask(memoID uuid.UUID) (Task, error) {
	task, err := NewMemoGenerationTask(
		memoID,
		f.memoService,
		f.generator,
		f.cardService,
		f.logger,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}
