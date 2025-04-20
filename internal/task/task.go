package task

import (
	"context"
	"database/sql"
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

// Task type constants
const (
	// TaskTypeMemoGeneration represents the task type for generating flashcards from memos
	TaskTypeMemoGeneration = "memo_generation"
)

// Task represents a unit of background work to be processed
// Version: 1.0
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

// TaskQueueReader provides read-only access to the task channel
// allowing workers to consume tasks without the ability to enqueue
// Version: 1.0
type TaskQueueReader interface {
	// GetChannel returns a read-only channel for consuming tasks
	GetChannel() <-chan Task
}

// TaskQueueWriter provides write access to the task queue
// allowing services to enqueue tasks for processing
// Version: 1.0
type TaskQueueWriter interface {
	// Enqueue adds a task to the queue for processing
	// Returns an error if the queue is full or closed
	Enqueue(task Task) error

	// Close closes the task queue, preventing further task submission
	Close()
}

// TaskStore defines the interface for persisting tasks
// Version: 1.0
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

	// WithTx returns a new TaskStore instance that uses the provided transaction.
	// This allows for multiple operations to be executed within a single transaction.
	// The transaction should be created and managed by the caller (typically a service).
	WithTx(tx *sql.Tx) TaskStore
}
