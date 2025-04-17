package task

import (
	"errors"
	"fmt"
	"log/slog"
)

// Common errors returned by the TaskQueue
var (
	ErrQueueClosed = errors.New("task queue is closed")
	ErrQueueFull   = errors.New("task queue is full")
)

// TaskQueue implements a buffered task queue that satisfies both
// TaskQueueReader and TaskQueueWriter interfaces
type TaskQueue struct {
	tasks  chan Task
	logger *slog.Logger
	closed bool
}

// NewTaskQueue creates a new task queue with the specified buffer size
func NewTaskQueue(size int, logger *slog.Logger) *TaskQueue {
	return &TaskQueue{
		tasks:  make(chan Task, size),
		logger: logger,
		closed: false,
	}
}

// Enqueue adds a task to the queue for processing
// Returns an error if the queue is full or closed
func (q *TaskQueue) Enqueue(task Task) error {
	if q.closed {
		return ErrQueueClosed
	}

	// Try to add the task to the channel
	select {
	case q.tasks <- task:
		q.logger.Debug("task enqueued",
			"task_id", task.ID(),
			"task_type", task.Type(),
			"queue_len", len(q.tasks),
			"queue_cap", cap(q.tasks))
		return nil
	default:
		// Channel is full
		return fmt.Errorf("%w: queue capacity %d reached", ErrQueueFull, cap(q.tasks))
	}
}

// Close closes the task queue, preventing further task submission
func (q *TaskQueue) Close() {
	if !q.closed {
		q.closed = true
		close(q.tasks)
		q.logger.Info("task queue closed")
	}
}

// GetChannel returns a read-only channel for consuming tasks
func (q *TaskQueue) GetChannel() <-chan Task {
	return q.tasks
}
