package task

import (
	"context"
	"errors"
	"testing"
	"time"

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
	logger := setupTestLogger()
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
	logger := setupTestLogger()
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

func TestWorkerPool_Start_Stop(t *testing.T) {
	logger := setupTestLogger()
	taskQueue := newMockTaskQueue()
	config := WorkerPoolConfig{
		WorkerCount: 2,
	}

	pool := NewWorkerPool(taskQueue, config, logger)

	// Start the worker pool
	pool.Start()

	// Give workers a moment to initialize
	time.Sleep(50 * time.Millisecond)

	// Stop the worker pool
	pool.Stop()

	// This test mainly checks that Start and Stop don't panic
	// The actual functionality is tested in other tests
}

func TestWorkerPool_ProcessTask_Success(t *testing.T) {
	logger := setupTestLogger()
	taskQueue := newMockTaskQueue()
	config := WorkerPoolConfig{
		WorkerCount: 1,
	}

	// Counter to track completed tasks
	completed := make(chan struct{})

	// Create a task that signals completion
	task := newMockTask()
	task.execFn = func(ctx context.Context) error {
		completed <- struct{}{}
		return nil
	}

	// Create and start the worker pool
	pool := NewWorkerPool(taskQueue, config, logger)
	pool.Start()

	// Add a task to the queue
	taskQueue.ch <- task

	// Wait for task completion or timeout
	select {
	case <-completed:
		// Task completed successfully
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for task to complete")
	}

	// Clean up
	pool.Stop()
}

func TestWorkerPool_ProcessTask_Error(t *testing.T) {
	logger := setupTestLogger()
	taskQueue := newMockTaskQueue()
	config := WorkerPoolConfig{
		WorkerCount: 1,
	}

	// Error channel to check if handler was called
	errorHandled := make(chan error)

	// Create a task that returns an error
	expectedErr := errors.New("test error")
	task := newMockTask()
	task.execFn = func(ctx context.Context) error {
		return expectedErr
	}

	// Create worker pool with error handler
	pool := NewWorkerPool(taskQueue, config, logger)
	pool.SetErrorHandler(func(task Task, err error) {
		errorHandled <- err
	})
	pool.Start()

	// Add the task to the queue
	taskQueue.ch <- task

	// Wait for error handler to be called or timeout
	select {
	case err := <-errorHandled:
		assert.Equal(t, expectedErr, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for error handler")
	}

	// Clean up
	pool.Stop()
}

func TestWorkerPool_ProcessTask_Panic(t *testing.T) {
	logger := setupTestLogger()
	taskQueue := newMockTaskQueue()
	config := WorkerPoolConfig{
		WorkerCount: 1,
	}

	// Error channel to check if handler was called
	errorHandled := make(chan error)

	// Create a task that panics
	task := newMockTask()
	task.execFn = func(ctx context.Context) error {
		panic("test panic")
	}

	// Create worker pool with error handler
	pool := NewWorkerPool(taskQueue, config, logger)
	pool.SetErrorHandler(func(task Task, err error) {
		errorHandled <- err
	})
	pool.Start()

	// Add the task to the queue
	taskQueue.ch <- task

	// Wait for error handler to be called or timeout
	select {
	case err := <-errorHandled:
		// Verify the error is a panic-related error
		assert.Contains(t, err.Error(), "panic")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for error handler after panic")
	}

	// Clean up
	pool.Stop()
}

func TestWorkerPool_Shutdown_DuringTask(t *testing.T) {
	logger := setupTestLogger()
	taskQueue := newMockTaskQueue()
	config := WorkerPoolConfig{
		WorkerCount: 1,
	}

	// Create a channel to signal when the task starts execution
	taskStarted := make(chan struct{})
	// Create a channel to signal when we can allow the task to finish
	allowFinish := make(chan struct{})
	// Create a channel to signal when the task has completed
	taskCompleted := make(chan struct{})

	// Create a task that blocks until signaled
	task := newMockTask()
	task.execFn = func(ctx context.Context) error {
		// Signal that task execution has started
		close(taskStarted)

		// Wait for context cancellation or allowFinish signal
		select {
		case <-ctx.Done():
			close(taskCompleted)
			return ctx.Err()
		case <-allowFinish:
			close(taskCompleted)
			return nil
		}
	}

	// Create and start the worker pool
	pool := NewWorkerPool(taskQueue, config, logger)
	pool.Start()

	// Add the task to the queue
	taskQueue.ch <- task

	// Wait for the task to start executing
	select {
	case <-taskStarted:
		// Task has started executing
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for task to start")
	}

	// Start a goroutine to stop the worker pool
	stopDone := make(chan struct{})
	go func() {
		pool.Stop()
		close(stopDone)
	}()

	// Wait for the task to be notified of cancellation
	select {
	case <-taskCompleted:
		// Task has been canceled, as expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for task to be canceled")
	}

	// Now Stop() should complete
	select {
	case <-stopDone:
		// This is expected, Stop completed after the task
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for worker pool to stop")
	}
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	logger := setupTestLogger()
	taskQueue := newMockTaskQueue()
	config := WorkerPoolConfig{
		WorkerCount: 1,
	}

	// Create a task that checks for context cancellation
	contextCanceled := make(chan struct{})
	task := newMockTask()
	task.execFn = func(ctx context.Context) error {
		<-ctx.Done() // Block until context is canceled
		close(contextCanceled)
		return ctx.Err()
	}

	// Create and start the worker pool
	pool := NewWorkerPool(taskQueue, config, logger)
	pool.Start()

	// Add the task to the queue
	taskQueue.ch <- task

	// Give the worker a moment to start processing the task
	time.Sleep(50 * time.Millisecond)

	// Stop the worker pool (this should cancel the context)
	pool.Stop()

	// Check if the context cancellation was detected by the task
	select {
	case <-contextCanceled:
		// Context was canceled, as expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for context cancellation")
	}
}
