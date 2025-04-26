package task

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// TaskRunnerConfig holds configuration for the task runner
type TaskRunnerConfig struct {
	// WorkerCount determines how many concurrent workers process tasks
	WorkerCount int

	// QueueSize determines the buffer size for the in-memory task queue
	QueueSize int

	// StuckTaskAge defines how long a task can be in processing state
	// before it's considered stuck and reset
	StuckTaskAge time.Duration

	// StuckTaskCheckInterval defines how often to check for stuck tasks
	// If zero, defaults to 5 minutes
	StuckTaskCheckInterval time.Duration
}

// DefaultTaskRunnerConfig returns a TaskRunnerConfig with reasonable defaults
func DefaultTaskRunnerConfig() TaskRunnerConfig {
	return TaskRunnerConfig{
		WorkerCount:            2,
		QueueSize:              100,
		StuckTaskAge:           30 * time.Minute,
		StuckTaskCheckInterval: 5 * time.Minute,
	}
}

// TaskRunner manages background task processing
type TaskRunner struct {
	store      TaskStore
	taskChan   chan Task
	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	config     TaskRunnerConfig
	logger     *slog.Logger
	errHandler func(task Task, err error)
}

// NewTaskRunner creates a new TaskRunner
func NewTaskRunner(store TaskStore, config TaskRunnerConfig, logger *slog.Logger) *TaskRunner {
	// Apply default check interval if not specified
	if config.StuckTaskCheckInterval == 0 {
		config.StuckTaskCheckInterval = 5 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TaskRunner{
		store:      store,
		taskChan:   make(chan Task, config.QueueSize),
		ctx:        ctx,
		cancelFunc: cancel,
		wg:         sync.WaitGroup{},
		config:     config,
		logger:     logger,
		errHandler: func(task Task, err error) {
			// Default error handler just logs the error
			logger.Error("task execution failed",
				"task_id", task.ID(),
				"task_type", task.Type(),
				"error", err)
		},
	}
}

// SetErrorHandler allows setting a custom error handler function
func (r *TaskRunner) SetErrorHandler(handler func(task Task, err error)) {
	r.errHandler = handler
}

// Submit adds a new task to the queue
func (r *TaskRunner) Submit(ctx context.Context, task Task) error {
	// Save task to database first
	if err := r.store.SaveTask(ctx, task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// Then add to in-memory queue
	select {
	case r.taskChan <- task:
		return nil
	default:
		// Queue is full, return error
		return fmt.Errorf("task queue is full, try again later")
	}
}

// Start initializes the worker pool and begins processing tasks
func (r *TaskRunner) Start() error {
	// Recover unfinished tasks from previous runs
	if err := r.Recover(); err != nil {
		return fmt.Errorf("failed to recover tasks: %w", err)
	}

	// Start worker goroutines
	for i := 0; i < r.config.WorkerCount; i++ {
		r.wg.Add(1)
		go r.worker(i)
	}

	// Start goroutine to check for stuck tasks periodically
	r.wg.Add(1)
	go r.stuckTaskMonitor()

	return nil
}

// Stop gracefully shuts down the task runner
func (r *TaskRunner) Stop() {
	r.cancelFunc()
	r.wg.Wait()
	close(r.taskChan)
}

// Recover loads any unfinished tasks from the database
func (r *TaskRunner) Recover() error {
	ctx := context.Background()

	// Get tasks that were in "pending" state
	pendingTasks, err := r.store.GetPendingTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending tasks: %w", err)
	}

	// Get tasks that were in "processing" state (potentially interrupted by a crash)
	processingTasks, err := r.store.GetProcessingTasks(
		ctx,
		0,
	) // Get all processing tasks regardless of age
	if err != nil {
		return fmt.Errorf("failed to get processing tasks: %w", err)
	}

	// Log recovery statistics
	r.logger.Info("recovering unfinished tasks",
		"pending_count", len(pendingTasks),
		"processing_count", len(processingTasks))

	// Requeue pending tasks
	for _, task := range pendingTasks {
		select {
		case r.taskChan <- task:
			// Successfully requeued
		default:
			// Queue is full, log error
			r.logger.Error("failed to requeue pending task, queue is full",
				"task_id", task.ID(),
				"task_type", task.Type())
		}
	}

	// Reset processing tasks back to pending state and requeue them
	for _, task := range processingTasks {
		// Update status in database to pending
		if err := r.store.UpdateTaskStatus(ctx, task.ID(), TaskStatusPending, "Reset after recovery"); err != nil {
			r.logger.Error("failed to reset processing task status",
				"task_id", task.ID(),
				"task_type", task.Type(),
				"error", err)
			continue
		}

		// Requeue
		select {
		case r.taskChan <- task:
			// Successfully requeued
		default:
			// Queue is full, log error
			r.logger.Error("failed to requeue processing task, queue is full",
				"task_id", task.ID(),
				"task_type", task.Type())
		}
	}

	return nil
}

// worker processes tasks from the queue
func (r *TaskRunner) worker(id int) {
	defer r.wg.Done()

	r.logger.Debug("starting worker", "worker_id", id)

	for {
		select {
		case <-r.ctx.Done():
			// Context cancelled, stop worker
			r.logger.Debug("stopping worker", "worker_id", id)
			return

		case task, ok := <-r.taskChan:
			if !ok {
				// Channel closed, stop worker
				r.logger.Debug("task channel closed, stopping worker", "worker_id", id)
				return
			}

			// Process the task
			r.processTask(task, id)
		}
	}
}

// processTask handles execution of a single task
func (r *TaskRunner) processTask(task Task, workerID int) {
	ctx := context.Background()
	logger := r.logger.With(
		"task_id", task.ID(),
		"task_type", task.Type(),
		"worker_id", workerID,
	)

	// Update task status to processing
	if err := r.store.UpdateTaskStatus(ctx, task.ID(), TaskStatusProcessing, ""); err != nil {
		logger.Error("failed to update task status to processing", "error", err)
		return
	}

	logger.Info("processing task")

	// Execute task
	err := task.Execute(ctx)

	if err != nil {
		// Task failed
		logger.Error("task execution failed", "error", err)
		if updateErr := r.store.UpdateTaskStatus(ctx, task.ID(), TaskStatusFailed, err.Error()); updateErr != nil {
			logger.Error("failed to update task status to failed", "error", updateErr)
		}

		// Call error handler
		r.errHandler(task, err)
	} else {
		// Task completed successfully
		logger.Info("task completed successfully")
		if updateErr := r.store.UpdateTaskStatus(ctx, task.ID(), TaskStatusCompleted, ""); updateErr != nil {
			logger.Error("failed to update task status to completed", "error", updateErr)
		}
	}
}

// stuckTaskMonitor periodically checks for tasks that have been in "processing"
// state for too long and resets them
func (r *TaskRunner) stuckTaskMonitor() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.config.StuckTaskCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			// Context cancelled, stop monitor
			return

		case <-ticker.C:
			ctx := context.Background()

			// Find tasks that have been in "processing" state for too long
			stuckTasks, err := r.store.GetProcessingTasks(ctx, r.config.StuckTaskAge)
			if err != nil {
				r.logger.Error("failed to check for stuck tasks", "error", err)
				continue
			}

			if len(stuckTasks) > 0 {
				r.logger.Info("found stuck tasks", "count", len(stuckTasks))

				// Reset each stuck task
				for _, task := range stuckTasks {
					if err := r.store.UpdateTaskStatus(ctx, task.ID(), TaskStatusPending,
						"Reset after being stuck in processing state"); err != nil {
						r.logger.Error("failed to reset stuck task status",
							"task_id", task.ID(),
							"task_type", task.Type(),
							"error", err)
						continue
					}

					// Requeue
					select {
					case r.taskChan <- task:
						// Successfully requeued
						r.logger.Info("requeued stuck task",
							"task_id", task.ID(),
							"task_type", task.Type())
					default:
						// Queue is full, log error
						r.logger.Error("failed to requeue stuck task, queue is full",
							"task_id", task.ID(),
							"task_type", task.Type())
					}
				}
			}
		}
	}
}
