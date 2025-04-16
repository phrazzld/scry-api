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

// Execute runs the memo generation task, handling the complete lifecycle
// from fetching the memo, updating status, generating cards, saving them,
// and finalizing the process. It handles errors at each step and ensures
// appropriate status updates.
func (t *MemoGenerationTask) Execute(ctx context.Context) error {
	// Update task status to processing
	t.status = statusProcessing
	t.logger.Info("starting memo generation task")

	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		t.status = statusFailed
		t.logger.Error("task cancelled by context", "error", err)
		return fmt.Errorf("task cancelled by context: %w", err)
	}

	// 1. Retrieve the memo
	memo, err := t.memoRepo.GetByID(ctx, t.memoID)
	if err != nil {
		t.status = statusFailed
		t.logger.Error("failed to retrieve memo", "error", err)
		return fmt.Errorf("failed to retrieve memo: %w", err)
	}

	t.logger.Info("retrieved memo", "user_id", memo.UserID, "memo_status", memo.Status)

	// 2. Update memo status to processing
	err = memo.UpdateStatus(domain.MemoStatusProcessing)
	if err != nil {
		t.status = statusFailed
		t.logger.Error("failed to update memo status to processing", "error", err)
		return fmt.Errorf("failed to update memo status to processing: %w", err)
	}

	// Save the updated status
	err = t.memoRepo.Update(ctx, memo)
	if err != nil {
		t.status = statusFailed
		t.logger.Error("failed to save memo processing status", "error", err)
		return fmt.Errorf("failed to save memo processing status: %w", err)
	}

	// 3. Generate cards
	t.logger.Info("generating cards from memo text")
	cards, err := t.generator.GenerateCards(ctx, memo.Text, memo.UserID)
	if err != nil {
		// Update memo status to failed on generation error
		_ = updateMemoStatusWithLogging(ctx, memo, domain.MemoStatusFailed, t.memoRepo, t.logger)
		t.status = statusFailed
		t.logger.Error("failed to generate cards", "error", err)
		return fmt.Errorf("failed to generate cards: %w", err)
	}

	// Log the number of cards generated
	t.logger.Info("cards generated", "count", len(cards))

	// 4. Save the generated cards (if any)
	if len(cards) > 0 {
		err = t.cardRepo.CreateMultiple(ctx, cards)
		if err != nil {
			// Update memo status to failed if we couldn't save the cards
			_ = updateMemoStatusWithLogging(ctx, memo, domain.MemoStatusFailed, t.memoRepo, t.logger)
			t.status = statusFailed
			t.logger.Error("failed to save generated cards", "error", err)
			return fmt.Errorf("failed to save generated cards: %w", err)
		}
		t.logger.Info("saved generated cards to database")
	} else {
		t.logger.Info("no cards were generated for this memo")
	}

	// 5. Update memo status to completed
	finalStatus := domain.MemoStatusCompleted
	if len(cards) == 0 {
		// If no cards were generated but no errors occurred, consider it completed but note in logs
		t.logger.Warn("memo processing completed but no cards were generated")
	}

	// Attempt to update the final status
	err = updateMemoStatusWithLogging(ctx, memo, finalStatus, t.memoRepo, t.logger)
	if err != nil {
		// Log the error but don't fail the task - the important work is done
		t.logger.Error("failed to update memo final status, but cards were generated and saved",
			"error", err,
			"cards_generated", len(cards))
	}

	// Update task status to completed
	t.status = statusCompleted
	t.logger.Info("memo generation task completed successfully", "cards_generated", len(cards))
	return nil
}

// updateMemoStatusWithLogging updates a memo's status and logs the outcome
// It's a helper function used by Execute to reduce code duplication
func updateMemoStatusWithLogging(
	ctx context.Context,
	memo *domain.Memo,
	status domain.MemoStatus,
	repo MemoRepository,
	logger *slog.Logger,
) error {
	err := memo.UpdateStatus(status)
	if err != nil {
		logger.Error("failed to set memo status", "status", status, "error", err)
		return fmt.Errorf("failed to set memo status to %s: %w", status, err)
	}

	err = repo.Update(ctx, memo)
	if err != nil {
		logger.Error("failed to save memo status", "status", status, "error", err)
		return fmt.Errorf("failed to save memo status %s: %w", status, err)
	}

	logger.Info("updated memo status", "status", status)
	return nil
}
