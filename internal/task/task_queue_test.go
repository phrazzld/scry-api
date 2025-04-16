package task

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockTask implements the Task interface for testing
type mockTask struct {
	id       uuid.UUID
	taskType string
	payload  []byte
	status   TaskStatus
	execFn   func(ctx context.Context) error
}

func (m *mockTask) ID() uuid.UUID {
	return m.id
}

func (m *mockTask) Type() string {
	return m.taskType
}

func (m *mockTask) Payload() []byte {
	return m.payload
}

func (m *mockTask) Status() TaskStatus {
	return m.status
}

func (m *mockTask) Execute(ctx context.Context) error {
	if m.execFn != nil {
		return m.execFn(ctx)
	}
	return nil
}

func newMockTask() *mockTask {
	return &mockTask{
		id:       uuid.New(),
		taskType: "mock",
		payload:  []byte("test payload"),
		status:   TaskStatusPending,
		execFn:   nil,
	}
}

func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func TestNewTaskQueue(t *testing.T) {
	logger := setupTestLogger()
	queueSize := 10
	queue := NewTaskQueue(queueSize, logger)

	assert.NotNil(t, queue)
	assert.Equal(t, queueSize, cap(queue.tasks))
	assert.False(t, queue.closed)
}

func TestEnqueue(t *testing.T) {
	logger := setupTestLogger()
	queue := NewTaskQueue(2, logger)

	// Test successful enqueue
	task1 := newMockTask()
	err := queue.Enqueue(task1)
	assert.NoError(t, err)

	task2 := newMockTask()
	err = queue.Enqueue(task2)
	assert.NoError(t, err)

	// Test queue full
	task3 := newMockTask()
	err = queue.Enqueue(task3)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrQueueFull)

	// Dequeue one item to make space
	<-queue.tasks

	// Now we should be able to enqueue again
	err = queue.Enqueue(task3)
	assert.NoError(t, err)
}

func TestClose(t *testing.T) {
	logger := setupTestLogger()
	queue := NewTaskQueue(10, logger)

	// Enqueue a task
	task := newMockTask()
	err := queue.Enqueue(task)
	assert.NoError(t, err)

	// Close the queue
	queue.Close()
	assert.True(t, queue.closed)

	// Try to enqueue after closing
	err = queue.Enqueue(newMockTask())
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrQueueClosed)

	// Make sure we can still read from the queue
	received := <-queue.GetChannel()
	assert.Equal(t, task.ID(), received.ID())

	// After draining the channel, the next read should return the zero value
	// since the channel is closed
	select {
	case _, ok := <-queue.GetChannel():
		assert.False(t, ok, "Channel should be closed")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for closed channel read")
	}
}

func TestGetChannel(t *testing.T) {
	logger := setupTestLogger()
	queue := NewTaskQueue(10, logger)

	task := newMockTask()
	err := queue.Enqueue(task)
	assert.NoError(t, err)

	// Get the read-only channel
	ch := queue.GetChannel()

	// Read from the channel
	receivedTask := <-ch
	assert.Equal(t, task.ID(), receivedTask.ID())
	assert.Equal(t, task.Type(), receivedTask.Type())
}

func TestConcurrentEnqueue(t *testing.T) {
	logger := setupTestLogger()
	queueSize := 100
	queue := NewTaskQueue(queueSize, logger)

	// Start multiple goroutines to enqueue tasks
	taskCount := 50
	doneCh := make(chan struct{})

	go func() {
		for i := 0; i < taskCount; i++ {
			task := newMockTask()
			err := queue.Enqueue(task)
			assert.NoError(t, err)
		}
		close(doneCh)
	}()

	// Wait for all tasks to be enqueued
	<-doneCh

	// Verify we can read all the tasks
	count := 0
	for i := 0; i < taskCount; i++ {
		select {
		case <-queue.GetChannel():
			count++
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timed out waiting for task")
		}
	}

	assert.Equal(t, taskCount, count, "Should read all enqueued tasks")
}
