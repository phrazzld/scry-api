package task

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockTaskQueue implements TaskQueueReader for testing
type mockTaskQueue struct {
	ch chan Task
}

func newMockTaskQueue() *mockTaskQueue {
	return &mockTaskQueue{
		ch: make(chan Task, 10),
	}
}

func (m *mockTaskQueue) GetChannel() <-chan Task {
	return m.ch
}

func TestNewWorkerPool(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Test with valid configuration
	taskQueue := newMockTaskQueue()
	config := WorkerPoolConfig{
		WorkerCount: 5,
	}

	pool := NewWorkerPool(taskQueue, config, logger)

	assert.NotNil(t, pool)
	assert.Equal(t, 5, pool.workerCount)
	assert.Equal(t, taskQueue, pool.taskQueue)
	assert.NotNil(t, pool.ctx)
	assert.NotNil(t, pool.cancel)
	assert.NotNil(t, pool.logger)
	assert.Nil(t, pool.errorHandler)

	// Test with invalid worker count (should default to 1)
	invalidConfig := WorkerPoolConfig{
		WorkerCount: 0,
	}

	pool = NewWorkerPool(taskQueue, invalidConfig, logger)
	assert.Equal(t, 1, pool.workerCount)

	// Test with negative worker count (should default to 1)
	invalidConfig.WorkerCount = -5
	pool = NewWorkerPool(taskQueue, invalidConfig, logger)
	assert.Equal(t, 1, pool.workerCount)
}

func TestSetErrorHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	taskQueue := newMockTaskQueue()
	config := DefaultWorkerPoolConfig()
	pool := NewWorkerPool(taskQueue, config, logger)

	// Initially the error handler should be nil
	assert.Nil(t, pool.errorHandler)

	// Set a custom error handler
	pool.SetErrorHandler(func(task Task, err error) {
		// This is just a test for setting the handler
		// The actual functionality will be tested in later tasks
	})

	// The error handler should now be set
	assert.NotNil(t, pool.errorHandler)
}
