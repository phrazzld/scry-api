package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresTaskStore(t *testing.T) {
	tests := []struct {
		name  string
		db    store.DBTX
		check func(t *testing.T, store *PostgresTaskStore)
	}{
		{
			name: "valid_db",
			db:   &sql.DB{},
			check: func(t *testing.T, store *PostgresTaskStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
			},
		},
		{
			name: "nil_db_still_creates_store",
			db:   nil,
			check: func(t *testing.T, store *PostgresTaskStore) {
				assert.NotNil(t, store)
				assert.Nil(t, store.db)
			},
		},
		{
			name: "mock_dbtx",
			db:   &mockDBTX{},
			check: func(t *testing.T, store *PostgresTaskStore) {
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewPostgresTaskStore(tt.db)
			if tt.check != nil {
				tt.check(t, store)
			}
		})
	}
}

func TestPostgresTaskStore_WithTx(t *testing.T) {
	// Note: We can't create a real *sql.Tx without a database connection,
	// so we'll test the behavior by checking the store structure.
	// The actual transaction behavior is tested in integration tests.

	originalDB := &sql.DB{}
	store := NewPostgresTaskStore(originalDB)

	// Verify the method exists and the store has expected fields
	assert.NotNil(t, store)
	assert.Equal(t, originalDB, store.db)
}

func TestDatabaseTask_Getters(t *testing.T) {
	taskID := uuid.New()
	taskType := "test_task"
	payload := []byte(`{"test": "data"}`)
	status := task.TaskStatusPending
	executeFn := func(ctx context.Context) error {
		return nil
	}

	dbTask := &databaseTask{
		id:        taskID,
		taskType:  taskType,
		payload:   payload,
		status:    status,
		executeFn: executeFn,
	}

	// Test ID()
	assert.Equal(t, taskID, dbTask.ID())

	// Test Type()
	assert.Equal(t, taskType, dbTask.Type())

	// Test Payload()
	assert.Equal(t, payload, dbTask.Payload())

	// Test Status()
	assert.Equal(t, status, dbTask.Status())
}

func TestDatabaseTask_Execute(t *testing.T) {
	tests := []struct {
		name      string
		executeFn func(ctx context.Context) error
		wantErr   bool
		errMsg    string
	}{
		{
			name: "execute_with_function",
			executeFn: func(ctx context.Context) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "execute_with_error",
			executeFn: func(ctx context.Context) error {
				return errors.New("execution failed")
			},
			wantErr: true,
			errMsg:  "execution failed",
		},
		{
			name:      "execute_without_function",
			executeFn: nil,
			wantErr:   true,
			errMsg:    "no execution function defined for recovered task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbTask := &databaseTask{
				id:        uuid.New(),
				taskType:  "test",
				payload:   []byte("{}"),
				status:    task.TaskStatusPending,
				executeFn: tt.executeFn,
			}

			err := dbTask.Execute(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
