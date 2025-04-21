package testutils

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/phrazzld/scry-api/internal/events"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
)

// CreateStoreInstances creates all store instances needed for testing.
// Takes a database transaction and logger, returns userStore, taskStore, memoStore, cardStore, statsStore.
func CreateStoreInstances(
	t *testing.T,
	dbtx store.DBTX,
	logger *slog.Logger,
) (*postgres.PostgresUserStore, *postgres.PostgresTaskStore, *postgres.PostgresMemoStore, *postgres.PostgresCardStore, *postgres.PostgresUserCardStatsStore) {
	t.Helper()
	userStore := postgres.NewPostgresUserStore(dbtx, 10) // BCrypt cost = 10 for faster tests
	taskStore := postgres.NewPostgresTaskStore(dbtx)
	memoStore := postgres.NewPostgresMemoStore(dbtx, logger)
	cardStore := postgres.NewPostgresCardStore(dbtx, logger)
	statsStore := postgres.NewPostgresUserCardStatsStore(dbtx, logger)
	return userStore, taskStore, memoStore, cardStore, statsStore
}

// CreateTaskRunner creates a task runner with the given configuration.
func CreateTaskRunner(
	t *testing.T,
	taskStore task.TaskStore,
	config task.TaskRunnerConfig,
	logger *slog.Logger,
) *task.TaskRunner {
	t.Helper()
	return task.NewTaskRunner(taskStore, config, logger)
}

// TaskFactoryEventHandler is an EventHandler that creates tasks from events
type TaskFactoryEventHandler struct {
	taskFactory *task.MemoGenerationTaskFactory
	taskRunner  *task.TaskRunner
	logger      *slog.Logger
}

// NewTaskFactoryEventHandler creates a new TaskFactoryEventHandler
func NewTaskFactoryEventHandler(
	taskFactory *task.MemoGenerationTaskFactory,
	taskRunner *task.TaskRunner,
	logger *slog.Logger,
) *TaskFactoryEventHandler {
	return &TaskFactoryEventHandler{
		taskFactory: taskFactory,
		taskRunner:  taskRunner,
		logger:      logger.With("component", "task_factory_event_handler"),
	}
}

// HandleEvent processes TaskRequestEvents by creating and submitting tasks
func (h *TaskFactoryEventHandler) HandleEvent(ctx context.Context, event *events.TaskRequestEvent) error {
	// Only handle memo generation events
	if event.Type != task.TaskTypeMemoGeneration {
		h.logger.Debug("ignoring event with unsupported type", "event_type", event.Type, "event_id", event.ID)
		return nil
	}

	// For testing purposes, we'll just log the event and return
	// This will be replaced by a proper handler in the next ticket
	h.logger.Debug("handling event", "event_id", event.ID, "event_type", event.Type)
	return nil
}

// CreateMemoServiceComponents creates all components needed for the memo service.
// Returns task runner, memo service, and task factory.
func CreateMemoServiceComponents(
	t *testing.T,
	dbtx store.DBTX,
	taskStore task.TaskStore,
	memoStore store.MemoStore,
	generator task.Generator,
	cardRepo task.CardRepository,
	taskConfig task.TaskRunnerConfig,
	logger *slog.Logger,
) (*task.TaskRunner, service.MemoService, *task.MemoGenerationTaskFactory) {
	t.Helper()
	// Configure task runner
	taskRunner := task.NewTaskRunner(taskStore, taskConfig, logger)

	// Create the memo service adapter for task package
	memoServiceAdapter, err := task.NewMemoServiceAdapter(memoStore)
	if err != nil {
		t.Fatalf("Failed to create memo service adapter: %v", err)
	}

	// Create the memo generation task factory with the adapter
	memoTaskFactory := task.NewMemoGenerationTaskFactory(
		memoServiceAdapter,
		generator,
		cardRepo,
		logger,
	)

	// Get the real DB from the dbtx to pass to repo adapter
	db, ok := dbtx.(*sql.DB)
	if !ok {
		// If it's not already a *sql.DB, create a dummy one for testing
		db = &sql.DB{}
	}

	// Create the memo repository adapter for service package
	memoRepoAdapter := service.NewMemoRepositoryAdapter(memoStore, db)

	// Create the event emitter
	eventEmitter := events.NewInMemoryEventEmitter(logger)

	// Create the event handler
	eventHandler := NewTaskFactoryEventHandler(memoTaskFactory, taskRunner, logger)

	// Register the event handler with the emitter
	eventEmitter.RegisterHandler(eventHandler)

	// Create the memo service
	memoService := service.NewMemoService(memoRepoAdapter, taskRunner, eventEmitter, logger)

	return taskRunner, memoService, memoTaskFactory
}

// GetDefaultTaskConfig returns a task runner configuration suitable for tests.
func GetDefaultTaskConfig() task.TaskRunnerConfig {
	return task.TaskRunnerConfig{
		WorkerCount:  1, // Use 1 worker for more predictable test execution
		QueueSize:    10,
		StuckTaskAge: 30 * 60 * 1000, // 30 minutes in milliseconds
	}
}

// GetNoopLogger returns a logger that discards all output.
// Useful for tests where you don't want to see log output.
func GetNoopLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
