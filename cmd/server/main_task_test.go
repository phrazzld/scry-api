package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// customTestTask implements task.Task for testing
type customTestTask struct {
	id       uuid.UUID
	payload  []byte
	status   task.TaskStatus
	executed bool
	mu       sync.Mutex
}

func newCustomTestTask() *customTestTask {
	return &customTestTask{
		id:       uuid.New(),
		payload:  []byte(`{"test":"data"}`),
		status:   task.TaskStatusPending,
		executed: false,
	}
}

func (t *customTestTask) ID() uuid.UUID {
	return t.id
}

func (t *customTestTask) Type() string {
	return "custom_test_task"
}

func (t *customTestTask) Payload() []byte {
	return t.payload
}

func (t *customTestTask) Status() task.TaskStatus {
	return t.status
}

func (t *customTestTask) Execute(ctx context.Context) error {
	// Mark as executed and simulate processing
	t.mu.Lock()
	t.executed = true
	t.mu.Unlock()
	
	// Simulate some work
	time.Sleep(10 * time.Millisecond)
	
	return nil
}

func (t *customTestTask) WasExecuted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.executed
}

// TestTaskRunnerIntegration tests that the task runner can be initialized, 
// started, and tasks can be submitted and processed.
func TestTaskRunnerIntegration(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("Skipping integration test - DATABASE_URL environment variable required")
	}

	// Set up the configuration
	cfg := &config.Config{
		Task: config.TaskConfig{
			WorkerCount:          2,
			QueueSize:            10,
			StuckTaskAgeMinutes:  30,
		},
	}

	// Set up the database connection
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Set up the task store
	taskStore := postgres.NewPostgresTaskStore(db)

	// Set up the task runner
	taskRunner := task.NewTaskRunner(taskStore, task.TaskRunnerConfig{
		WorkerCount:  cfg.Task.WorkerCount,
		QueueSize:    cfg.Task.QueueSize,
		StuckTaskAge: time.Duration(cfg.Task.StuckTaskAgeMinutes) * time.Minute,
	}, slog.Default())

	// Start the task runner
	err := taskRunner.Start()
	require.NoError(t, err, "Failed to start task runner")

	// Create and submit a test task
	testTask := newCustomTestTask()
	err = taskRunner.Submit(context.Background(), testTask)
	require.NoError(t, err, "Failed to submit task")

	// Wait for the task to be processed
	startTime := time.Now()
	for !testTask.WasExecuted() && time.Since(startTime) < 2*time.Second {
		time.Sleep(50 * time.Millisecond)
	}

	// Check that the task was executed
	assert.True(t, testTask.WasExecuted(), "Task should have been executed")

	// Stop the task runner
	taskRunner.Stop()
}

// Helper functions for integration test setup
func setupTestDB(t *testing.T) *sql.DB {
	// Get the database URL from environment for testing
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL environment variable is required for integration tests")
	}
	
	// Connect to the database
	db, err := sql.Open("pgx", dbURL)
	require.NoError(t, err, "Failed to open database connection")
	
	// Set reasonable connection pool settings for tests
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Return the database connection
	return db
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
	if db != nil {
		err := db.Close()
		if err != nil {
			t.Logf("Warning: failed to close database connection: %v", err)
		}
	}
}