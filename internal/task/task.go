package task

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the current state of a task
type TaskStatus string

// Possible task status values
const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

// Task represents a unit of background work to be processed
type Task interface {
	// ID returns the task's unique identifier
	ID() uuid.UUID

	// Type returns the task type identifier
	Type() string

	// Payload returns the task data as a byte slice
	Payload() []byte

	// Status returns the current task status
	Status() TaskStatus

	// Execute runs the task logic
	Execute(ctx context.Context) error
}

// TaskStore defines the interface for persisting tasks
type TaskStore interface {
	// SaveTask persists a task to the database
	SaveTask(ctx context.Context, task Task) error

	// UpdateTaskStatus updates the status of a task
	UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status TaskStatus, errorMsg string) error

	// GetPendingTasks retrieves all tasks with "pending" status
	GetPendingTasks(ctx context.Context) ([]Task, error)

	// GetProcessingTasks retrieves tasks with "processing" status
	// If olderThan is non-zero, only returns tasks that have been in this state
	// longer than the specified duration
	GetProcessingTasks(ctx context.Context, olderThan time.Duration) ([]Task, error)
}
