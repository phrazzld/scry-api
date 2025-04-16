package task

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MockTask is a simple implementation of the Task interface for testing
type MockTask struct {
	TaskID      uuid.UUID
	TaskType    string
	TaskPayload []byte
	TaskStatus  TaskStatus
	ExecuteFn   func(ctx context.Context) error
}

// NewMockTask creates a new MockTask with the given ID and type
func NewMockTask(id uuid.UUID, taskType string, payload []byte) *MockTask {
	return &MockTask{
		TaskID:      id,
		TaskType:    taskType,
		TaskPayload: payload,
		TaskStatus:  TaskStatusPending,
		ExecuteFn:   func(ctx context.Context) error { return nil },
	}
}

// ID returns the task's unique identifier
func (t *MockTask) ID() uuid.UUID {
	return t.TaskID
}

// Type returns the task type identifier
func (t *MockTask) Type() string {
	return t.TaskType
}

// Payload returns the task data as a byte slice
func (t *MockTask) Payload() []byte {
	return t.TaskPayload
}

// Status returns the current task status
func (t *MockTask) Status() TaskStatus {
	return t.TaskStatus
}

// Execute runs the task logic
func (t *MockTask) Execute(ctx context.Context) error {
	return t.ExecuteFn(ctx)
}

// MockPayload is a sample payload structure used for testing
type MockPayload struct {
	Message string    `json:"message"`
	Created time.Time `json:"created"`
}

// CreateMockTaskWithPayload is a helper function to create a MockTask with a structured payload
func CreateMockTaskWithPayload(message string) *MockTask {
	payload := MockPayload{
		Message: message,
		Created: time.Now().UTC(),
	}

	data, _ := json.Marshal(payload)
	return NewMockTask(uuid.New(), "mock_task", data)
}
