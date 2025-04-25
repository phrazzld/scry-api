package task

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/events"
)

// TaskFactoryEventHandler implements the events.EventHandler interface
// to handle task creation events and delegate them to the appropriate task factory.
type TaskFactoryEventHandler struct {
	// We use interface{} instead of concrete types to avoid circular dependencies
	// The expected types are *MemoGenerationTaskFactory and *TaskRunner
	taskFactory interface {
		CreateTask(memoID uuid.UUID) (interface{}, error)
	}
	taskRunner interface {
		Submit(ctx context.Context, task interface{}) error
	}
	logger *slog.Logger
}

// NewTaskFactoryEventHandler creates a new event handler that uses the given task factory
// to create tasks, and submits them to the provided task runner.
func NewTaskFactoryEventHandler(
	taskFactory interface {
		CreateTask(memoID uuid.UUID) (interface{}, error)
	},
	taskRunner interface {
		Submit(ctx context.Context, task interface{}) error
	},
	logger *slog.Logger,
) *TaskFactoryEventHandler {
	return &TaskFactoryEventHandler{
		taskFactory: taskFactory,
		taskRunner:  taskRunner,
		logger:      logger.With("component", "task_factory_event_handler"),
	}
}

// HandleEvent processes events by creating and submitting tasks.
// It extracts the payload from the event, creates the appropriate task,
// and submits it to the runner for execution.
func (h *TaskFactoryEventHandler) HandleEvent(
	ctx context.Context,
	event *events.TaskRequestEvent,
) error {
	// Only handle memo generation events for now
	if event.Type != "memo_generation" { // Using string literal instead of constant to avoid circular imports
		h.logger.Debug(
			"ignoring event with unsupported type",
			"event_type",
			event.Type,
			"event_id",
			event.ID,
		)
		return nil
	}

	// Extract the memo ID from the event payload
	var payload struct {
		MemoID string `json:"memo_id"`
	}

	if err := event.UnmarshalPayload(&payload); err != nil {
		h.logger.Error("failed to unmarshal payload", "error", err, "event_id", event.ID)
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Parse the memo ID
	memoID, err := uuid.Parse(payload.MemoID)
	if err != nil {
		h.logger.Error(
			"invalid memo ID",
			"error",
			err,
			"memo_id",
			payload.MemoID,
			"event_id",
			event.ID,
		)
		return fmt.Errorf("invalid memo ID: %w", err)
	}

	// Create the task
	h.logger.Debug("creating task for memo", "memo_id", memoID, "event_id", event.ID)
	task, err := h.taskFactory.CreateTask(memoID)
	if err != nil {
		h.logger.Error(
			"failed to create task",
			"error",
			err,
			"memo_id",
			memoID,
			"event_id",
			event.ID,
		)
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Get task ID for logging - we know the task has an ID() method that returns uuid.UUID
	taskID := task.(interface{ ID() uuid.UUID }).ID()

	// Submit the task to the runner
	h.logger.Debug(
		"submitting task to runner",
		"task_id",
		taskID,
		"memo_id",
		memoID,
		"event_id",
		event.ID,
	)
	if err := h.taskRunner.Submit(ctx, task); err != nil {
		h.logger.Error(
			"failed to submit task",
			"error", err,
			"task_id", taskID,
			"memo_id", memoID,
			"event_id", event.ID,
		)
		return fmt.Errorf("failed to submit task: %w", err)
	}

	h.logger.Info(
		"task created and submitted successfully",
		"task_id", taskID,
		"memo_id", memoID,
		"event_id", event.ID,
	)
	return nil
}

// Ensure TaskFactoryEventHandler implements events.EventHandler
var _ events.EventHandler = (*TaskFactoryEventHandler)(nil)
