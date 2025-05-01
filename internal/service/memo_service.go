package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/events"
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
	CreateMemoAndEnqueueTask(
		ctx context.Context,
		userID uuid.UUID,
		text string,
	) (*domain.Memo, error)

	// UpdateMemoStatus updates a memo's status and handles related business logic
	UpdateMemoStatus(ctx context.Context, memoID uuid.UUID, status domain.MemoStatus) error

	// GetMemo retrieves a memo by its ID
	GetMemo(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error)
}

// Common sentinel errors for MemoService
var (
	// ErrMemoNotFound indicates that the memo does not exist
	ErrMemoNotFound = errors.New("memo not found")
)

// MemoServiceError wraps errors from the memo service with context.
type MemoServiceError struct {
	// Operation is the operation that failed (e.g., "create_memo", "update_memo_status")
	Operation string
	// Message is a human-readable description of the error
	Message string
	// Err is the underlying error that caused the failure
	Err error
}

// Error implements the error interface for MemoServiceError.
func (e *MemoServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("memo service %s failed: %s: %v", e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("memo service %s failed: %s", e.Operation, e.Message)
}

// Unwrap returns the wrapped error to support errors.Is/errors.As.
func (e *MemoServiceError) Unwrap() error {
	return e.Err
}

// NewMemoServiceError creates a new MemoServiceError.
// It returns known sentinel errors directly without wrapping.
func NewMemoServiceError(operation, message string, err error) error {
	if err == nil {
		return nil
	}

	// Check for service-defined sentinel errors
	if errors.Is(err, ErrMemoNotFound) {
		return ErrMemoNotFound
	}

	// Check for store-level sentinel errors that should be mapped to service-level ones
	if errors.Is(err, store.ErrMemoNotFound) {
		return ErrMemoNotFound
	}

	// If not a sentinel to be returned directly, wrap it
	return &MemoServiceError{
		Operation: operation,
		Message:   message,
		Err:       err,
	}
}

// memoServiceImpl implements the MemoService interface
type memoServiceImpl struct {
	memoRepo     MemoRepository
	taskRunner   TaskRunner
	eventEmitter events.EventEmitter
	logger       *slog.Logger
}

// NewMemoService creates a new MemoService
// It returns an error if any of the required dependencies are nil.
func NewMemoService(
	memoRepo MemoRepository,
	taskRunner TaskRunner,
	eventEmitter events.EventEmitter,
	logger *slog.Logger,
) (MemoService, error) {
	// Validate dependencies
	if memoRepo == nil {
		return nil, &MemoServiceError{
			Operation: "create_service",
			Message:   "memoRepo cannot be nil",
			Err:       nil,
		}
	}
	if taskRunner == nil {
		return nil, &MemoServiceError{
			Operation: "create_service",
			Message:   "taskRunner cannot be nil",
			Err:       nil,
		}
	}
	if eventEmitter == nil {
		return nil, &MemoServiceError{
			Operation: "create_service",
			Message:   "eventEmitter cannot be nil",
			Err:       nil,
		}
	}

	// Use provided logger or create default
	if logger == nil {
		logger = slog.Default()
	}

	return &memoServiceImpl{
		memoRepo:     memoRepo,
		taskRunner:   taskRunner,
		eventEmitter: eventEmitter,
		logger:       logger.With("component", "memo_service"),
	}, nil
}

// CreateMemoAndEnqueueTask creates a new memo with pending status and emits an event for processing
// Uses a transaction for the memo creation part to ensure atomicity
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
		return nil, NewMemoServiceError("create_memo", "failed to create memo object", err)
	}

	// 2. Save the memo to the database using a transaction
	err = store.RunInTransaction(ctx, s.memoRepo.DB(), func(ctx context.Context, tx *sql.Tx) error {
		// Get a transactional repo
		txRepo := s.memoRepo.WithTx(tx)

		// Create the memo within the transaction
		err := txRepo.Create(ctx, memo)
		if err != nil {
			s.logger.Error("failed to create memo in transaction",
				"error", err,
				"user_id", userID,
				"memo_id", memo.ID)
			return NewMemoServiceError("create_memo", "failed to save memo to database", err)
		}
		return nil
	})

	if err != nil {
		// Error is already wrapped in the transaction
		return nil, err
	}

	s.logger.Info("memo created successfully with pending status",
		"memo_id", memo.ID,
		"user_id", userID)

	// 3. Create a payload for the event
	payload := struct {
		MemoID uuid.UUID `json:"memo_id"`
	}{
		MemoID: memo.ID,
	}

	// 4. Create and emit a TaskRequestEvent
	event, err := events.NewTaskRequestEvent(task.TaskTypeMemoGeneration, payload)
	if err != nil {
		s.logger.Error("failed to create memo generation event",
			"error", err,
			"memo_id", memo.ID,
			"user_id", userID)
		return nil, NewMemoServiceError("create_memo", "failed to create event", err)
	}

	// 5. Emit the event
	err = s.eventEmitter.EmitEvent(ctx, event)
	if err != nil {
		s.logger.Error("failed to emit memo generation event",
			"error", err,
			"memo_id", memo.ID,
			"user_id", userID,
			"event_id", event.ID)
		return nil, NewMemoServiceError("create_memo", "failed to emit event", err)
	}

	s.logger.Info("memo generation event emitted successfully",
		"memo_id", memo.ID,
		"user_id", userID,
		"event_id", event.ID)

	return memo, nil
}

// GetMemo retrieves a memo by its ID
func (s *memoServiceImpl) GetMemo(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error) {
	memo, err := s.memoRepo.GetByID(ctx, memoID)
	if err != nil {
		s.logger.Error("failed to retrieve memo",
			"error", err,
			"memo_id", memoID)

		if errors.Is(err, store.ErrMemoNotFound) {
			return nil, ErrMemoNotFound
		}
		return nil, NewMemoServiceError("get_memo", "failed to retrieve memo", err)
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
func (s *memoServiceImpl) UpdateMemoStatus(
	ctx context.Context,
	memoID uuid.UUID,
	status domain.MemoStatus,
) error {
	// Use a transaction to ensure atomicity
	return store.RunInTransaction(
		ctx,
		s.memoRepo.DB(),
		func(ctx context.Context, tx *sql.Tx) error {
			// Get a transactional repo
			txRepo := s.memoRepo.WithTx(tx)

			// Retrieve the memo first
			memo, err := txRepo.GetByID(ctx, memoID)
			if err != nil {
				s.logger.Error("failed to retrieve memo for status update",
					"error", err,
					"memo_id", memoID,
					"target_status", status)

				if errors.Is(err, store.ErrMemoNotFound) {
					return ErrMemoNotFound
				}
				return NewMemoServiceError("update_memo_status", "failed to retrieve memo", err)
			}

			// Update the memo's status
			err = memo.UpdateStatus(status)
			if err != nil {
				s.logger.Error("failed to update memo status",
					"error", err,
					"memo_id", memoID,
					"current_status", memo.Status,
					"target_status", status)
				return NewMemoServiceError(
					"update_memo_status",
					fmt.Sprintf("failed to update memo status to %s", status),
					err,
				)
			}

			// Save the updated memo using the transactional repo
			err = txRepo.Update(ctx, memo)
			if err != nil {
				s.logger.Error("failed to save updated memo status",
					"error", err,
					"memo_id", memoID,
					"status", status)
				return NewMemoServiceError(
					"update_memo_status",
					fmt.Sprintf("failed to save memo with status %s", status),
					err,
				)
			}

			s.logger.Info("memo status updated successfully in transaction",
				"memo_id", memoID,
				"status", status)
			return nil
		},
	)
}
