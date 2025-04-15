package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTask implements the task.Task interface for testing
type testTask struct {
	id     uuid.UUID
	typ    string
	data   []byte
	status task.TaskStatus
}

func newTestTask() *testTask {
	data, _ := json.Marshal(map[string]interface{}{
		"test_key": "test_value",
		"time":     time.Now().UTC(),
	})

	return &testTask{
		id:     uuid.New(),
		typ:    "test_task",
		data:   data,
		status: task.TaskStatusPending,
	}
}

func (t *testTask) ID() uuid.UUID {
	return t.id
}

func (t *testTask) Type() string {
	return t.typ
}

func (t *testTask) Payload() []byte {
	return t.data
}

func (t *testTask) Status() task.TaskStatus {
	return t.status
}

func (t *testTask) Execute(ctx context.Context) error {
	return nil
}

// isIntegrationTestEnvironment returns true if the environment is configured
// for running integration tests with a database connection
func isIntegrationTestEnvironment() bool {
	return os.Getenv("DATABASE_URL") != ""
}

// getTestDatabaseURL returns the database URL for integration tests
func getTestDatabaseURL(t *testing.T) string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL environment variable is required for this test")
	}
	return dbURL
}

// Integration tests for PostgresTaskStore
func TestPostgresTaskStore_Integration(t *testing.T) {
	if !isIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - DATABASE_URL environment variable required")
	}

	// Get database connection
	dbURL := getTestDatabaseURL(t)
	db, err := sql.Open("pgx", dbURL)
	require.NoError(t, err, "Failed to open database connection")
	defer func() {
		err := db.Close()
		if err != nil {
			t.Logf("Error closing database connection: %v", err)
		}
	}()

	// Run test with transaction-based isolation
	tx, err := db.Begin()
	require.NoError(t, err, "Failed to begin transaction")
	
	defer func() {
		err := tx.Rollback()
		if err != nil && err != sql.ErrTxDone {
			t.Logf("Error rolling back transaction: %v", err)
		}
	}()
	
	ctx := context.Background()
	store := NewPostgresTaskStore(tx)

	t.Run("SaveTask", func(t *testing.T) {
		// Create a test task
		testTask := newTestTask()

		// Save the task
		err := store.SaveTask(ctx, testTask)
		require.NoError(t, err, "Failed to save task")

		// Verify task was saved correctly
		var count int
		err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE id = $1", testTask.ID()).Scan(&count)
		require.NoError(t, err, "Failed to count tasks")
		assert.Equal(t, 1, count, "Task should be saved in the database")

		// Verify task data
		var typ string
		var status string
		var payload []byte
		err = tx.QueryRowContext(ctx, "SELECT type, status, payload FROM tasks WHERE id = $1", testTask.ID()).
			Scan(&typ, &status, &payload)
		require.NoError(t, err, "Failed to query task data")
		assert.Equal(t, testTask.Type(), typ)
		assert.Equal(t, string(testTask.Status()), status)
		assert.Equal(t, testTask.Payload(), payload)
	})

	t.Run("UpdateTaskStatus", func(t *testing.T) {
		// Create and save a test task
		testTask := newTestTask()
		err := store.SaveTask(ctx, testTask)
		require.NoError(t, err, "Failed to save task")

		// Update task status
		err = store.UpdateTaskStatus(ctx, testTask.ID(), task.TaskStatusProcessing, "test error message")
		require.NoError(t, err, "Failed to update task status")

		// Verify task status was updated
		var status string
		var errorMsg string
		err = tx.QueryRowContext(ctx, "SELECT status, error_message FROM tasks WHERE id = $1", testTask.ID()).
			Scan(&status, &errorMsg)
		require.NoError(t, err, "Failed to query task status")
		assert.Equal(t, string(task.TaskStatusProcessing), status)
		assert.Equal(t, "test error message", errorMsg)
	})

	t.Run("GetPendingTasks", func(t *testing.T) {
		// Create and save multiple test tasks with different statuses
		pendingTask1 := newTestTask()
		pendingTask2 := newTestTask()
		processingTask := newTestTask()

		// Save tasks
		require.NoError(t, store.SaveTask(ctx, pendingTask1))
		require.NoError(t, store.SaveTask(ctx, pendingTask2))
		require.NoError(t, store.SaveTask(ctx, processingTask))

		// Update one task to processing status
		require.NoError(t, store.UpdateTaskStatus(ctx, processingTask.ID(), task.TaskStatusProcessing, ""))

		// Get pending tasks
		pendingTasks, err := store.GetPendingTasks(ctx)
		require.NoError(t, err, "Failed to get pending tasks")

		// Extract task IDs for easier comparison
		pendingIDs := make(map[uuid.UUID]bool)
		for _, t := range pendingTasks {
			pendingIDs[t.ID()] = true
		}

		// Verify we got at least the two pending tasks we just created
		// (there might be more from other tests)
		assert.True(t, pendingIDs[pendingTask1.ID()], "Pending task 1 should be returned")
		assert.True(t, pendingIDs[pendingTask2.ID()], "Pending task 2 should be returned")
		assert.False(t, pendingIDs[processingTask.ID()], "Processing task should not be returned")
	})

	t.Run("GetProcessingTasks", func(t *testing.T) {
		// Create and save test tasks
		pendingTask := newTestTask()
		newProcessingTask := newTestTask()
		oldProcessingTask := newTestTask()

		// Save tasks
		require.NoError(t, store.SaveTask(ctx, pendingTask))
		require.NoError(t, store.SaveTask(ctx, newProcessingTask))
		require.NoError(t, store.SaveTask(ctx, oldProcessingTask))

		// Update to processing status
		require.NoError(t, store.UpdateTaskStatus(ctx, newProcessingTask.ID(), task.TaskStatusProcessing, ""))
		require.NoError(t, store.UpdateTaskStatus(ctx, oldProcessingTask.ID(), task.TaskStatusProcessing, ""))

		// Make the old processing task's updated_at field older
		_, err := tx.ExecContext(ctx,
			"UPDATE tasks SET updated_at = $1 WHERE id = $2",
			time.Now().UTC().Add(-15*time.Minute), oldProcessingTask.ID())
		require.NoError(t, err, "Failed to update task timestamp")

		// Test getting all processing tasks
		allProcessingTasks, err := store.GetProcessingTasks(ctx, 0)
		require.NoError(t, err, "Failed to get processing tasks")

		// Extract task IDs
		allProcessingIDs := make(map[uuid.UUID]bool)
		for _, t := range allProcessingTasks {
			allProcessingIDs[t.ID()] = true
		}

		// Should include both processing tasks
		assert.True(t, allProcessingIDs[newProcessingTask.ID()], "New processing task should be returned")
		assert.True(t, allProcessingIDs[oldProcessingTask.ID()], "Old processing task should be returned")
		assert.False(t, allProcessingIDs[pendingTask.ID()], "Pending task should not be returned")

		// Test getting only old processing tasks
		oldProcessingTasks, err := store.GetProcessingTasks(ctx, 10*time.Minute)
		require.NoError(t, err, "Failed to get old processing tasks")

		// Extract task IDs
		oldProcessingIDs := make(map[uuid.UUID]bool)
		for _, t := range oldProcessingTasks {
			oldProcessingIDs[t.ID()] = true
		}

		// Should include only the old processing task
		assert.False(t, oldProcessingIDs[newProcessingTask.ID()], "New processing task should not be returned")
		assert.True(t, oldProcessingIDs[oldProcessingTask.ID()], "Old processing task should be returned")
		assert.False(t, oldProcessingIDs[pendingTask.ID()], "Pending task should not be returned")
	})
}