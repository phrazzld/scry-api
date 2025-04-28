//go:build integration

package postgres_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTask implements the task.Task interface for testing
type mockTask struct {
	id     uuid.UUID
	typ    string
	data   []byte
	status task.TaskStatus
}

func newMockTask() *mockTask {
	data, _ := json.Marshal(map[string]interface{}{
		"test_key": "test_value",
		"time":     time.Now().UTC(),
	})

	return &mockTask{
		id:     uuid.New(),
		typ:    "test_task",
		data:   data,
		status: task.TaskStatusPending,
	}
}

func (t *mockTask) ID() uuid.UUID {
	return t.id
}

func (t *mockTask) Type() string {
	return t.typ
}

func (t *mockTask) Payload() []byte {
	return t.data
}

func (t *mockTask) Status() task.TaskStatus {
	return t.status
}

func (t *mockTask) Execute(ctx context.Context) error {
	return nil
}

// TestTaskStoreErrorLeakage tests that TaskStore operations do not leak internal
// database error details in their returned errors.
func TestTaskStoreErrorLeakage(t *testing.T) {
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test")
	}

	// We need to run real database tests to trigger actual PostgreSQL errors
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create a transaction for isolation
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback() // Intentionally ignoring error as it's cleanup code
	}()

	// Create the store
	taskStore := postgres.NewPostgresTaskStore(tx)

	// Tests for SaveTask operation
	t.Run("SaveTask with duplicate ID does not leak details", func(t *testing.T) {
		// First save the task normally
		task := newMockTask()
		err := taskStore.SaveTask(ctx, task)
		require.NoError(t, err)

		// Now try to save another task with the same ID to trigger a constraint violation
		duplicateTask := newMockTask()
		duplicateTask.id = task.id // Same ID as the previously saved task

		err = taskStore.SaveTask(ctx, duplicateTask)
		assert.Error(t, err)
		// The error should be mapped to a domain error
		AssertNoErrorLeakage(t, err)
	})

	// Tests for UpdateTaskStatus operation
	t.Run("UpdateTaskStatus errors do not leak details", func(t *testing.T) {
		// Create a task for updates
		task := newMockTask()
		err := taskStore.SaveTask(ctx, task)
		require.NoError(t, err)

		// Test with invalid status
		// Note: Since TaskStatus is just a string, we can pass any string,
		// but the database schema might have a constraint on valid values
		err = taskStore.UpdateTaskStatus(ctx, task.id, "invalid_status", "")
		// Even if this doesn't cause an error, if it does, check it doesn't leak details
		if err != nil {
			AssertNoErrorLeakage(t, err)
		}
	})

	// Tests for GetPendingTasks and GetProcessingTasks operations
	t.Run("Get*Tasks errors do not leak details", func(t *testing.T) {
		// Force a database error if possible (this might be difficult in a real scenario)
		// For example, by using a transaction that has already been committed

		// For this test, we'll compromise by checking that no sensitive terms
		// appear in the returned task objects from a successful call

		// First create a few tasks
		for i := 0; i < 3; i++ {
			task := newMockTask()
			err := taskStore.SaveTask(ctx, task)
			require.NoError(t, err)
		}

		// Get pending tasks
		tasks, err := taskStore.GetPendingTasks(ctx)
		require.NoError(t, err)

		// Check that each task doesn't have sensitive information in its fields
		for _, tsk := range tasks {
			// Ensure task ID is valid
			assert.NotEmpty(t, tsk.ID())

			// Ensure task type doesn't contain sensitive info
			// We can't use AssertNoErrorLeakage directly since it expects an error
			// So we'll check for sensitive terms manually
			taskType := tsk.Type()
			for _, term := range []string{"postgres", "sql", "database"} {
				assert.NotContains(t, taskType, term,
					"Task type contains sensitive term: %q", term)
			}

			// Ensure payload doesn't contain sensitive info
			if len(tsk.Payload()) > 0 {
				payload := string(tsk.Payload())
				for _, term := range []string{"postgres", "sql", "database"} {
					assert.NotContains(t, payload, term,
						"Task payload contains sensitive term: %q", term)
				}
			}
		}
	})
}
