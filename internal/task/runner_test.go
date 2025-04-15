package task

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRunner_Submit(t *testing.T) {
	t.Parallel()

	// Setup
	store := NewMockTaskStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	config := DefaultTaskRunnerConfig()
	config.QueueSize = 2 // Small queue size to test full queue behavior
	
	runner := NewTaskRunner(store, config, logger)

	// Test cases
	t.Run("successful submission", func(t *testing.T) {
		t.Parallel()
		
		task := CreateMockTaskWithPayload("test task")
		err := runner.Submit(context.Background(), task)
		
		assert.NoError(t, err)
		
		// Verify task was saved to store
		pendingTasks, _ := store.GetPendingTasks(context.Background())
		assert.Contains(t, extractTaskIDs(pendingTasks), task.ID())
	})
	
	t.Run("queue full", func(t *testing.T) {
		t.Parallel()
		
		// Create a runner with a queue size of 1
		smallStore := NewMockTaskStore()
		smallConfig := DefaultTaskRunnerConfig()
		smallConfig.QueueSize = 1
		
		smallRunner := NewTaskRunner(smallStore, smallConfig, logger)
		
		// Fill the queue
		task1 := CreateMockTaskWithPayload("task 1")
		err := smallRunner.Submit(context.Background(), task1)
		require.NoError(t, err)
		
		// Add another task to fill queue
		task2 := CreateMockTaskWithPayload("task 2")
		err = smallRunner.Submit(context.Background(), task2)
		
		// Expect error due to full queue
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "queue is full")
	})
	
	t.Run("store error", func(t *testing.T) {
		t.Parallel()
		
		// Create a store that returns an error on save
		errorStore := NewMockTaskStore()
		errorStore.SaveFn = func(ctx context.Context, task Task) error {
			return errors.New("mock store error")
		}
		
		errorRunner := NewTaskRunner(errorStore, config, logger)
		
		task := CreateMockTaskWithPayload("error task")
		err := errorRunner.Submit(context.Background(), task)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save task")
	})
}

func TestTaskRunner_Start_and_Processing(t *testing.T) {
	t.Parallel()

	// Setup
	store := NewMockTaskStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	config := DefaultTaskRunnerConfig()
	config.WorkerCount = 2
	config.QueueSize = 10
	
	runner := NewTaskRunner(store, config, logger)

	// Create a channel to verify task execution
	taskCompletedChan := make(chan uuid.UUID, 5)
	
	// Use a mutex to protect shared state access
	var mu sync.Mutex 
	taskIDs := make([]uuid.UUID, 0, 3)

	// Add some tasks with custom execution functions
	for i := 0; i < 3; i++ {
		task := CreateMockTaskWithPayload("test task")
		
		// Store the task ID for later verification
		mu.Lock()
		taskIDs = append(taskIDs, task.ID())
		mu.Unlock()
		
		// Set execution function
		task.ExecuteFn = func(ctx context.Context) error {
			taskCompletedChan <- task.ID()
			return nil
		}
		
		err := runner.Submit(context.Background(), task)
		require.NoError(t, err)
	}

	// Start the runner
	err := runner.Start()
	require.NoError(t, err)

	// Collect completed tasks with a timeout
	completedTasks := make(map[uuid.UUID]bool)
	timeout := time.After(2 * time.Second)
	
	// Wait for all tasks to complete
taskWaitLoop:
	for len(completedTasks) < 3 {
		select {
		case taskID := <-taskCompletedChan:
			completedTasks[taskID] = true
		case <-timeout:
			break taskWaitLoop
		}
	}
	
	// Stop the runner
	runner.Stop()
	
	// Verify all tasks were completed
	mu.Lock()
	defer mu.Unlock()
	
	for _, id := range taskIDs {
		assert.True(t, completedTasks[id], "Task %s should have been completed", id)
	}
	assert.Len(t, completedTasks, 3, "All 3 tasks should have been completed")
}

func TestTaskRunner_TaskFailure(t *testing.T) {
	t.Parallel()

	// Setup
	store := NewMockTaskStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	config := DefaultTaskRunnerConfig()
	runner := NewTaskRunner(store, config, logger)

	// Create a channel to track error handler calls
	errorChan := make(chan struct{}, 1)
	
	// Set a custom error handler
	runner.SetErrorHandler(func(task Task, err error) {
		errorChan <- struct{}{}
	})

	// Create task that will fail
	task := CreateMockTaskWithPayload("failing task")
	task.ExecuteFn = func(ctx context.Context) error {
		return errors.New("intentional test failure")
	}
	
	err := runner.Submit(context.Background(), task)
	require.NoError(t, err)

	// Start the runner
	err = runner.Start()
	require.NoError(t, err)

	// Wait for error handler to be called
	select {
	case <-errorChan:
		// Error handler was called as expected
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for error handler to be called")
	}
	
	// Add a small delay to allow for the task status to be updated
	time.Sleep(100 * time.Millisecond)
	
	// Stop the runner
	runner.Stop()

	// Verify task is marked as failed in the store
	var foundFailedTask bool
	taskID := task.ID()
	for id, storedTask := range store.tasks {
		if id == taskID && storedTask.Status() == TaskStatusFailed {
			foundFailedTask = true
			break
		}
	}
	
	assert.True(t, foundFailedTask, "Task should be marked as failed")
}

func TestTaskRunner_Recover(t *testing.T) {
	t.Parallel()

	// Setup
	store := NewMockTaskStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	// Add some pending and processing tasks to the store
	pendingTask := CreateMockTaskWithPayload("pending task")
	processingTask := CreateMockTaskWithPayload("processing task")
	
	// Save tasks with appropriate status
	store.SaveTask(context.Background(), pendingTask)
	
	// Save processing task and update its status
	store.SaveTask(context.Background(), processingTask)
	store.UpdateTaskStatus(context.Background(), processingTask.ID(), TaskStatusProcessing, "")
	
	// Create a channel to track task execution
	taskCompletedChan := make(chan uuid.UUID, 5)
	
	// Create a new runner
	config := DefaultTaskRunnerConfig()
	runner := NewTaskRunner(store, config, logger)
	
	// Set ExecuteFn for all tasks to signal completion
	for _, storedTask := range store.tasks {
		mockTask := storedTask.(*MockTask)
		mockTask.ExecuteFn = func(ctx context.Context) error {
			taskCompletedChan <- storedTask.ID()
			return nil
		}
	}
	
	// Start the runner which will trigger recovery
	err := runner.Start()
	require.NoError(t, err)
	
	// Expected task IDs to be completed
	expectedTasks := map[uuid.UUID]bool{
		pendingTask.ID(): false,
		processingTask.ID(): false,
	}
	
	// Collect completed tasks with a timeout
	timeout := time.After(2 * time.Second)
	
	// Wait for all tasks to be executed
taskWaitLoop:
	for {
		allCompleted := true
		for _, completed := range expectedTasks {
			if !completed {
				allCompleted = false
				break
			}
		}
		
		if allCompleted {
			break taskWaitLoop
		}
		
		select {
		case taskID := <-taskCompletedChan:
			expectedTasks[taskID] = true
		case <-timeout:
			break taskWaitLoop
		}
	}
	
	// Stop the runner
	runner.Stop()
	
	// Verify all tasks were executed
	assert.True(t, expectedTasks[pendingTask.ID()], "Pending task should have been completed")
	assert.True(t, expectedTasks[processingTask.ID()], "Processing task should have been completed")
}

func TestTaskRunner_StuckTasks(t *testing.T) {
	t.Parallel()

	// Setup
	store := NewMockTaskStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	// Create a task and mark it as processing but set its timestamp to be old
	stuckTask := CreateMockTaskWithPayload("stuck task")
	store.SaveTask(context.Background(), stuckTask)
	store.UpdateTaskStatus(context.Background(), stuckTask.ID(), TaskStatusProcessing, "")
	
	// Manually set the task's status time to be old (30 minutes ago)
	store.taskStatusTimes[stuckTask.ID()] = time.Now().Add(-30 * time.Minute)
	
	// Create a channel to track task execution
	taskCompletedChan := make(chan uuid.UUID, 5)
	
	// Set ExecuteFn to signal completion
	mockTask := store.tasks[stuckTask.ID()].(*MockTask)
	mockTask.ExecuteFn = func(ctx context.Context) error {
		taskCompletedChan <- stuckTask.ID()
		return nil
	}
	
	// Create a new runner with a very short stuck task check interval
	config := DefaultTaskRunnerConfig()
	config.StuckTaskAge = 15 * time.Minute         // Consider tasks older than 15 minutes as stuck
	config.StuckTaskCheckInterval = 100 * time.Millisecond // Check very frequently for test
	
	runner := NewTaskRunner(store, config, logger)
	
	// Start the runner
	err := runner.Start()
	require.NoError(t, err)
	
	// Wait for the stuck task to be executed with a timeout
	select {
	case taskID := <-taskCompletedChan:
		assert.Equal(t, stuckTask.ID(), taskID, "Stuck task should have been executed")
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for stuck task to be executed")
	}
	
	// Stop the runner
	runner.Stop()
}

// Helper function to extract task IDs from a slice of tasks
func extractTaskIDs(tasks []Task) []uuid.UUID {
	ids := make([]uuid.UUID, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID()
	}
	return ids
}