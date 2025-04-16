package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// Task type and status constants
const (
	TaskTypeMemoGeneration = "memo_generation"

	// Status constants for MemoGenerationTask
	// These match the TaskStatus values defined in task.go
	statusPending    = "pending"
	statusProcessing = "processing"
	statusCompleted  = "completed"
	statusFailed     = "failed"
)

// Common errors
var (
	ErrNilMemoRepository = errors.New("memo repository cannot be nil")
	ErrNilGenerator      = errors.New("generator cannot be nil")
	ErrNilCardRepository = errors.New("card repository cannot be nil")
	ErrNilLogger         = errors.New("logger cannot be nil")
	ErrEmptyMemoID       = errors.New("memo ID cannot be empty")
)

// MemoRepository defines the interface for memo data operations
type MemoRepository interface {
	// GetByID retrieves a memo by its unique ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error)

	// Update saves changes to an existing memo
	Update(ctx context.Context, memo *domain.Memo) error
}

// Generator defines the interface for flashcard generation services
type Generator interface {
	// GenerateCards creates flashcards from memo text
	GenerateCards(ctx context.Context, memoText string, userID uuid.UUID) ([]*domain.Card, error)
}

// CardRepository defines the interface for card data operations
type CardRepository interface {
	// CreateMultiple saves multiple new cards to the store
	CreateMultiple(ctx context.Context, cards []*domain.Card) error
}

// memoGenerationPayload represents the serialized data stored in the task
type memoGenerationPayload struct {
	MemoID uuid.UUID `json:"memo_id"`
}

// MemoGenerationTask implements the Task interface for generating
// flashcards from a memo
type MemoGenerationTask struct {
	id        uuid.UUID
	memoID    uuid.UUID
	memoRepo  MemoRepository
	generator Generator
	cardRepo  CardRepository
	logger    *slog.Logger
	status    string // Using string instead of TaskStatus to avoid circular imports
}

// NewMemoGenerationTask creates a new memo generation task
func NewMemoGenerationTask(
	memoID uuid.UUID,
	memoRepo MemoRepository,
	generator Generator,
	cardRepo CardRepository,
	logger *slog.Logger,
) (*MemoGenerationTask, error) {
	// Validate dependencies
	if memoRepo == nil {
		return nil, ErrNilMemoRepository
	}
	if generator == nil {
		return nil, ErrNilGenerator
	}
	if cardRepo == nil {
		return nil, ErrNilCardRepository
	}
	if logger == nil {
		return nil, ErrNilLogger
	}

	// Validate memo ID
	if memoID == uuid.Nil {
		return nil, ErrEmptyMemoID
	}

	return &MemoGenerationTask{
		id:        uuid.New(),
		memoID:    memoID,
		memoRepo:  memoRepo,
		generator: generator,
		cardRepo:  cardRepo,
		logger:    logger.With("task_type", TaskTypeMemoGeneration, "memo_id", memoID),
		status:    statusPending,
	}, nil
}

// ID returns the task's unique identifier
func (t *MemoGenerationTask) ID() uuid.UUID {
	return t.id
}

// Type returns the task type identifier
func (t *MemoGenerationTask) Type() string {
	return TaskTypeMemoGeneration
}

// Payload returns the task data as a byte slice
func (t *MemoGenerationTask) Payload() []byte {
	payload := memoGenerationPayload{
		MemoID: t.memoID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		// If marshal fails, return an empty payload with error logged
		t.logger.Error("failed to marshal task payload", "error", err)
		return []byte{}
	}

	return data
}

// Status returns the current task status
// We convert the string to TaskStatus to fulfill the Task interface
func (t *MemoGenerationTask) Status() TaskStatus {
	return TaskStatus(t.status)
}

// Execute implements the Task interface but does nothing yet
// This will be implemented in T107
func (t *MemoGenerationTask) Execute(ctx context.Context) error {
	// Implementation will be added in T107
	return fmt.Errorf("not implemented")
}
