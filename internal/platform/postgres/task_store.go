package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
)

// PostgresTaskStore implements the task.TaskStore interface using PostgreSQL
type PostgresTaskStore struct {
	db store.DBTX
}

// NewPostgresTaskStore creates a new PostgresTaskStore
func NewPostgresTaskStore(db store.DBTX) *PostgresTaskStore {
	return &PostgresTaskStore{
		db: db,
	}
}

// SaveTask persists a task to the database
func (s *PostgresTaskStore) SaveTask(ctx context.Context, task task.Task) error {
	log := logger.FromContext(ctx)
	
	// Insert the task into the database
	query := `
		INSERT INTO tasks (id, type, payload, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	// Convert payload to JSONB-compatible format
	payload := task.Payload()
	
	now := time.Now().UTC()
	
	_, err := s.db.ExecContext(ctx, query,
		task.ID(),
		task.Type(),
		payload,
		task.Status(),
		now,
		now,
	)
	
	if err != nil {
		log.Error("failed to save task",
			"task_id", task.ID(),
			"task_type", task.Type(),
			"error", err)
		return fmt.Errorf("failed to save task to database: %w", err)
	}
	
	return nil
}

// UpdateTaskStatus updates the status of a task in the database
func (s *PostgresTaskStore) UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status task.TaskStatus, errorMsg string) error {
	log := logger.FromContext(ctx)
	
	query := `
		UPDATE tasks
		SET status = $1, error_message = $2, updated_at = $3
		WHERE id = $4
	`
	
	now := time.Now().UTC()
	
	result, err := s.db.ExecContext(ctx, query,
		status,
		errorMsg,
		now,
		taskID,
	)
	
	if err != nil {
		log.Error("failed to update task status",
			"task_id", taskID,
			"status", status,
			"error", err)
		return fmt.Errorf("failed to update task status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected",
			"task_id", taskID,
			"error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		log.Warn("no task found with ID to update status",
			"task_id", taskID)
		return nil // Task not found, treat as no-op
	}
	
	return nil
}

// GetPendingTasks retrieves all tasks with "pending" status
func (s *PostgresTaskStore) GetPendingTasks(ctx context.Context) ([]task.Task, error) {
	return s.getTasksByStatus(ctx, task.TaskStatusPending, 0)
}

// GetProcessingTasks retrieves tasks with "processing" status
func (s *PostgresTaskStore) GetProcessingTasks(ctx context.Context, olderThan time.Duration) ([]task.Task, error) {
	return s.getTasksByStatus(ctx, task.TaskStatusProcessing, olderThan)
}

// getTasksByStatus is a helper method to get tasks by status with optional age filter
func (s *PostgresTaskStore) getTasksByStatus(ctx context.Context, status task.TaskStatus, olderThan time.Duration) ([]task.Task, error) {
	log := logger.FromContext(ctx)
	
	var query string
	var args []interface{}
	
	if olderThan > 0 {
		// Get tasks older than the specified duration
		query = `
			SELECT id, type, payload, status, error_message, created_at, updated_at
			FROM tasks
			WHERE status = $1 AND updated_at < $2
			ORDER BY created_at ASC
		`
		args = []interface{}{status, time.Now().UTC().Add(-olderThan)}
	} else {
		// Get all tasks with the given status
		query = `
			SELECT id, type, payload, status, error_message, created_at, updated_at
			FROM tasks
			WHERE status = $1
			ORDER BY created_at ASC
		`
		args = []interface{}{status}
	}
	
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("failed to query tasks by status",
			"status", status,
			"error", err)
		return nil, fmt.Errorf("failed to query tasks by status: %w", err)
	}
	defer rows.Close()
	
	var tasks []task.Task
	
	for rows.Next() {
		var id uuid.UUID
		var taskType string
		var payload []byte
		var taskStatus task.TaskStatus
		var errorMessage sql.NullString
		var createdAt time.Time
		var updatedAt time.Time
		
		if err := rows.Scan(&id, &taskType, &payload, &taskStatus, &errorMessage, &createdAt, &updatedAt); err != nil {
			log.Error("failed to scan task row",
				"status", status,
				"error", err)
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}
		
		// Create a DatabaseTask with the retrieved data
		t := &databaseTask{
			id:           id,
			taskType:     taskType,
			payload:      payload,
			status:       taskStatus,
			errorMessage: errorMessage.String,
			createdAt:    createdAt,
			updatedAt:    updatedAt,
		}
		
		tasks = append(tasks, t)
	}
	
	if err := rows.Err(); err != nil {
		log.Error("error iterating task rows",
			"status", status,
			"error", err)
		return nil, fmt.Errorf("error iterating task rows: %w", err)
	}
	
	return tasks, nil
}

// databaseTask implements the task.Task interface for tasks loaded from the database
type databaseTask struct {
	id           uuid.UUID
	taskType     string
	payload      []byte
	status       task.TaskStatus
	errorMessage string
	createdAt    time.Time
	updatedAt    time.Time
	executeFn    func(ctx context.Context) error
}

// ID returns the task's unique identifier
func (t *databaseTask) ID() uuid.UUID {
	return t.id
}

// Type returns the task type identifier
func (t *databaseTask) Type() string {
	return t.taskType
}

// Payload returns the task data as a byte slice
func (t *databaseTask) Payload() []byte {
	return t.payload
}

// Status returns the current task status
func (t *databaseTask) Status() task.TaskStatus {
	return t.status
}

// Execute runs the task logic
// Note: For recovered tasks, the execution function needs to be set
// by the task registry/factory before execution
func (t *databaseTask) Execute(ctx context.Context) error {
	if t.executeFn != nil {
		return t.executeFn(ctx)
	}
	
	// If no execute function is set, this is a "dummy" recovered task
	// that hasn't been properly initialized with its execution function.
	// In a real implementation, you'd have a registry or factory that
	// creates concrete task instances based on the task type.
	return errors.New("no execution function defined for recovered task")
}

// Creates a concrete task from a database task
// This will be used in a future implementation to create a task instance
// from a database record, with full context about how to execute it
func createTaskFromDatabaseTask(dbTask *databaseTask) task.Task {
	switch dbTask.taskType {
	case "generate_cards":
		// Example of how to parse a GenerateCardsTask payload
		var payload struct {
			UserID string `json:"user_id"`
			MemoID string `json:"memo_id"`
		}
		
		if err := json.Unmarshal(dbTask.payload, &payload); err != nil {
			// If we can't parse the payload, return the database task as is
			return dbTask
		}
		
		// Create a new GenerateCardsTask (would be defined elsewhere)
		// generateCardsTask := NewGenerateCardsTask(...)
		// return generateCardsTask
		
		// For now, just return the database task
		return dbTask
		
	default:
		// For unknown task types, return the database task as is
		return dbTask
	}
}