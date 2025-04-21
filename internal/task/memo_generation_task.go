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

// Status constants for MemoGenerationTask
// These match the TaskStatus values defined in task.go
const (
	statusPending    = "pending"
	statusProcessing = "processing"
	statusCompleted  = "completed"
	statusFailed     = "failed"
)

// Common errors
var (
	ErrNilMemoService = errors.New("memo service cannot be nil")
	ErrNilGenerator   = errors.New("generator cannot be nil")
	ErrNilCardService = errors.New("card service cannot be nil")
	ErrNilLogger      = errors.New("logger cannot be nil")
	ErrEmptyMemoID    = errors.New("memo ID cannot be empty")
)

// MemoService defines the interface for memo service operations
// This replaces the MemoRepository to ensure proper separation of concerns
type MemoService interface {
	// GetMemo retrieves a memo by its ID
	GetMemo(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error)

	// UpdateMemoStatus updates a memo's status and handles related business logic
	UpdateMemoStatus(ctx context.Context, memoID uuid.UUID, status domain.MemoStatus) error
}

// Generator defines the interface for flashcard generation services
type Generator interface {
	// GenerateCards creates flashcards from memo text
	GenerateCards(ctx context.Context, memoText string, userID uuid.UUID) ([]*domain.Card, error)
}

// CardService defines the interface for card service operations
type CardService interface {
	// CreateCards creates multiple cards and their associated stats in a single transaction
	// This orchestrates both card and stats creation atomically
	CreateCards(ctx context.Context, cards []*domain.Card) error

	// GetCard retrieves a card by its ID
	GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)
}

// memoGenerationPayload represents the serialized data stored in the task
type memoGenerationPayload struct {
	MemoID uuid.UUID `json:"memo_id"`
}

// MemoGenerationTask implements the Task interface for generating
// flashcards from a memo
type MemoGenerationTask struct {
	id          uuid.UUID
	memoID      uuid.UUID
	memoService MemoService
	generator   Generator
	cardService CardService
	logger      *slog.Logger
	status      string // Using string instead of TaskStatus to avoid circular imports
}

// NewMemoGenerationTask creates a new memo generation task
func NewMemoGenerationTask(
	memoID uuid.UUID,
	memoService MemoService,
	generator Generator,
	cardService CardService,
	logger *slog.Logger,
) (*MemoGenerationTask, error) {
	// Validate dependencies
	if memoService == nil {
		return nil, ErrNilMemoService
	}
	if generator == nil {
		return nil, ErrNilGenerator
	}
	if cardService == nil {
		return nil, ErrNilCardService
	}
	if logger == nil {
		return nil, ErrNilLogger
	}

	// Validate memo ID
	if memoID == uuid.Nil {
		return nil, ErrEmptyMemoID
	}

	return &MemoGenerationTask{
		id:          uuid.New(),
		memoID:      memoID,
		memoService: memoService,
		generator:   generator,
		cardService: cardService,
		logger:      logger.With("task_type", TaskTypeMemoGeneration, "memo_id", memoID),
		status:      statusPending,
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
	memo, err := t.memoService.GetMemo(ctx, t.memoID)
	if err != nil {
		t.status = statusFailed
		t.logger.Error("failed to retrieve memo", "error", err)
		return fmt.Errorf("failed to retrieve memo: %w", err)
	}

	t.logger.Info("retrieved memo", "user_id", memo.UserID, "memo_status", memo.Status)

	// 2. Update memo status to processing
	err = t.memoService.UpdateMemoStatus(ctx, t.memoID, domain.MemoStatusProcessing)
	if err != nil {
		t.status = statusFailed
		t.logger.Error("failed to update memo status to processing", "error", err)
		return fmt.Errorf("failed to update memo status to processing: %w", err)
	}

	// 3. Generate cards
	t.logger.Info("generating cards from memo text")
	cards, err := t.generator.GenerateCards(ctx, memo.Text, memo.UserID)
	if err != nil {
		// Update memo status to failed on generation error
		_ = t.memoService.UpdateMemoStatus(ctx, t.memoID, domain.MemoStatusFailed)
		t.status = statusFailed
		t.logger.Error("failed to generate cards", "error", err)
		return fmt.Errorf("failed to generate cards: %w", err)
	}

	// Log the number of cards generated
	t.logger.Info("cards generated", "count", len(cards))

	// 4. Save the generated cards (if any)
	if len(cards) > 0 {
		// Use CardService to create cards and stats in a single transaction
		err = t.cardService.CreateCards(ctx, cards)

		if err != nil {
			// Update memo status to failed if we couldn't save the cards
			_ = t.memoService.UpdateMemoStatus(ctx, t.memoID, domain.MemoStatusFailed)
			t.status = statusFailed
			t.logger.Error("failed to save generated cards and stats", "error", err)
			return fmt.Errorf("failed to save generated cards and stats: %w", err)
		}
		t.logger.Info("saved generated cards and stats to database")
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
	err = t.memoService.UpdateMemoStatus(ctx, t.memoID, finalStatus)
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
