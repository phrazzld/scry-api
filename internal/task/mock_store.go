package task

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockTaskStore implements the TaskStore interface for testing
type MockTaskStore struct {
	mutex           sync.RWMutex
	tasks           map[uuid.UUID]Task
	taskStatusTimes map[uuid.UUID]time.Time
	SaveFn          func(ctx context.Context, task Task) error
	UpdateStatusFn  func(ctx context.Context, taskID uuid.UUID, status TaskStatus, errorMsg string) error
}

// NewMockTaskStore creates a new MockTaskStore with default implementations
func NewMockTaskStore() *MockTaskStore {
	store := &MockTaskStore{
		tasks:           make(map[uuid.UUID]Task),
		taskStatusTimes: make(map[uuid.UUID]time.Time),
	}

	// Default behavior for SaveTask
	store.SaveFn = func(ctx context.Context, task Task) error {
		store.mutex.Lock()
		defer store.mutex.Unlock()

		mockTask, ok := task.(*MockTask)
		if !ok {
			// If it's not a MockTask, create a new one with same properties
			mockTask = NewMockTask(task.ID(), task.Type(), task.Payload())
			mockTask.TaskStatus = task.Status()
		}

		store.tasks[task.ID()] = mockTask
		store.taskStatusTimes[task.ID()] = time.Now()
		return nil
	}

	// Default behavior for UpdateTaskStatus
	store.UpdateStatusFn = func(ctx context.Context, taskID uuid.UUID, status TaskStatus, errorMsg string) error {
		store.mutex.Lock()
		defer store.mutex.Unlock()

		task, exists := store.tasks[taskID]
		if !exists {
			return nil // Simulate "not found" as a no-op for testing simplicity
		}

		mockTask := task.(*MockTask)
		mockTask.TaskStatus = status
		store.tasks[taskID] = mockTask
		store.taskStatusTimes[taskID] = time.Now()
		return nil
	}

	return store
}

// SaveTask persists a task to the mock store
func (s *MockTaskStore) SaveTask(ctx context.Context, task Task) error {
	return s.SaveFn(ctx, task)
}

// UpdateTaskStatus updates the status of a task in the mock store
func (s *MockTaskStore) UpdateTaskStatus(
	ctx context.Context,
	taskID uuid.UUID,
	status TaskStatus,
	errorMsg string,
) error {
	return s.UpdateStatusFn(ctx, taskID, status, errorMsg)
}

// GetPendingTasks retrieves all tasks with "pending" status
func (s *MockTaskStore) GetPendingTasks(ctx context.Context) ([]Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var pendingTasks []Task
	for _, task := range s.tasks {
		if task.Status() == TaskStatusPending {
			pendingTasks = append(pendingTasks, task)
		}
	}

	return pendingTasks, nil
}

// GetProcessingTasks retrieves tasks with "processing" status
func (s *MockTaskStore) GetProcessingTasks(ctx context.Context, olderThan time.Duration) ([]Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var processingTasks []Task
	now := time.Now()

	for _, task := range s.tasks {
		if task.Status() == TaskStatusProcessing {
			statusTime, exists := s.taskStatusTimes[task.ID()]
			// If olderThan is zero, include all processing tasks
			// Otherwise, only include tasks that have been in this state longer than olderThan
			if olderThan == 0 || (exists && now.Sub(statusTime) > olderThan) {
				processingTasks = append(processingTasks, task)
			}
		}
	}

	return processingTasks, nil
}

// WithTx implements TaskStore.WithTx for the mock store
// In the mock implementation, we just return the same store instance
func (s *MockTaskStore) WithTx(tx *sql.Tx) TaskStore {
	return s
}
