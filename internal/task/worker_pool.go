package task

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
)

// WorkerPool manages a pool of worker goroutines that process tasks
// from a task queue. It handles graceful shutdown and worker lifecycle.
type WorkerPool struct {
	// taskQueue provides read access to the tasks to be processed
	taskQueue TaskQueueReader

	// workerCount is the number of concurrent workers to start
	workerCount int

	// wg tracks active worker goroutines for clean shutdown
	wg sync.WaitGroup

	// ctx is used for cancellation and shutdown signaling
	ctx context.Context

	// cancel is the function to call to cancel the context
	cancel context.CancelFunc

	// logger for structured logging
	logger *slog.Logger

	// errorHandler is called when a task execution fails
	// If nil, errors are only logged
	errorHandler func(task Task, err error)
}

// WorkerPoolConfig holds configuration options for the worker pool
type WorkerPoolConfig struct {
	// WorkerCount determines how many concurrent worker goroutines to start
	// If zero or negative, defaults to 1
	WorkerCount int
}

// DefaultWorkerPoolConfig returns a WorkerPoolConfig with reasonable defaults
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		WorkerCount: 2,
	}
}

// NewWorkerPool creates a new worker pool with the specified configuration
func NewWorkerPool(taskQueue TaskQueueReader, config WorkerPoolConfig, logger *slog.Logger) *WorkerPool {
	// Apply defaults for invalid config values
	workerCount := config.WorkerCount
	if workerCount <= 0 {
		workerCount = 1
		logger.Warn("invalid worker count specified, using default",
			"specified_count", config.WorkerCount,
			"default_count", 1)
	}

	// Create a cancelable context for shutdown coordination
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		taskQueue:    taskQueue,
		workerCount:  workerCount,
		wg:           sync.WaitGroup{},
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
		errorHandler: nil, // Default to nil, can be set later with SetErrorHandler
	}
}

// SetErrorHandler allows setting a custom error handler for task execution failures
func (p *WorkerPool) SetErrorHandler(handler func(task Task, err error)) {
	p.errorHandler = handler
}

// Start launches worker goroutines to process tasks from the queue.
// It returns immediately after starting the workers.
func (p *WorkerPool) Start() {
	p.logger.Info("starting worker pool", "worker_count", p.workerCount)

	// Start the requested number of worker goroutines
	for i := 0; i < p.workerCount; i++ {
		workerID := i
		p.wg.Add(1)
		go p.runWorker(workerID)
	}
}

// Stop signals all workers to shut down and waits for them to complete.
// It blocks until all workers have finished processing their current tasks.
func (p *WorkerPool) Stop() {
	p.logger.Info("stopping worker pool, waiting for in-progress tasks to complete")

	// Signal all workers to stop
	p.cancel()

	// Wait for all workers to finish
	p.wg.Wait()

	p.logger.Info("worker pool stopped")
}

// runWorker processes tasks from the queue until context is cancelled or the queue is closed.
// It handles the worker goroutine lifecycle.
func (p *WorkerPool) runWorker(workerID int) {
	// Ensure WaitGroup is decremented when the worker exits
	defer p.wg.Done()

	p.logger.Debug("worker started", "worker_id", workerID)

	// Process tasks until context is cancelled or channel is closed
	for {
		select {
		case <-p.ctx.Done():
			// Context was cancelled (Stop was called)
			p.logger.Debug("worker shutting down due to context cancellation", "worker_id", workerID)
			return

		case task, ok := <-p.taskQueue.GetChannel():
			// Check if channel was closed
			if !ok {
				p.logger.Debug("worker shutting down due to closed task queue", "worker_id", workerID)
				return
			}

			// Process the task
			p.processTask(task, workerID)
		}
	}
}

// processTask executes a single task with panic recovery and error handling.
func (p *WorkerPool) processTask(task Task, workerID int) {
	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic during task execution: %v\n%s", r, debug.Stack())
			p.logger.Error("recovered from panic in task execution",
				"worker_id", workerID,
				"task_id", task.ID(),
				"task_type", task.Type(),
				"panic", r,
				"stack", string(debug.Stack()))

			// Call error handler if set
			if p.errorHandler != nil {
				p.errorHandler(task, err)
			}
		}
	}()

	// Get task-specific logger with context
	logger := p.logger.With(
		"worker_id", workerID,
		"task_id", task.ID(),
		"task_type", task.Type(),
	)

	logger.Info("processing task")

	// Execute the task with the pool's context for cancellation
	err := task.Execute(p.ctx)

	if err != nil {
		// Task execution failed
		logger.Error("task execution failed", "error", err)

		// Call error handler if set
		if p.errorHandler != nil {
			p.errorHandler(task, err)
		}
	} else {
		// Task completed successfully
		logger.Info("task completed successfully")
	}
}
