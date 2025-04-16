package task

import (
	"context"
	"log/slog"
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
