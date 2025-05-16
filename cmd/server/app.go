package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/events"
	"github.com/phrazzld/scry-api/internal/platform/gemini"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
	"golang.org/x/crypto/bcrypt"
)

// TaskFactoryEventHandler is an event handler that creates tasks when events are emitted
type TaskFactoryEventHandler struct {
	taskFactory *task.MemoGenerationTaskFactory
	taskRunner  *task.TaskRunner
	logger      *slog.Logger
}

// HandleEvent processes events by creating and submitting tasks
func (h *TaskFactoryEventHandler) HandleEvent(
	ctx context.Context,
	event *events.TaskRequestEvent,
) error {
	// Only handle memo generation events for now
	if event.Type != task.TaskTypeMemoGeneration {
		h.logger.Debug(
			"ignoring event with unsupported type",
			"event_type",
			event.Type,
			"event_id",
			event.ID,
		)
		return nil
	}

	// Extract the memo ID from the event payload
	var payload struct {
		MemoID string `json:"memo_id"`
	}

	if err := event.UnmarshalPayload(&payload); err != nil {
		h.logger.Error("failed to unmarshal payload", "error", err, "event_id", event.ID)
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Parse the memo ID
	memoID, err := uuid.Parse(payload.MemoID)
	if err != nil {
		h.logger.Error(
			"invalid memo ID",
			"error",
			err,
			"memo_id",
			payload.MemoID,
			"event_id",
			event.ID,
		)
		return fmt.Errorf("invalid memo ID: %w", err)
	}

	// Create the task
	h.logger.Debug("creating task for memo", "memo_id", memoID, "event_id", event.ID)
	task, err := h.taskFactory.CreateTask(memoID)
	if err != nil {
		h.logger.Error(
			"failed to create task",
			"error",
			err,
			"memo_id",
			memoID,
			"event_id",
			event.ID,
		)
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Submit the task to the runner
	h.logger.Debug(
		"submitting task to runner",
		"task_id",
		task.ID(),
		"memo_id",
		memoID,
		"event_id",
		event.ID,
	)
	if err := h.taskRunner.Submit(ctx, task); err != nil {
		h.logger.Error(
			"failed to submit task",
			"error",
			err,
			"task_id",
			task.ID(),
			"memo_id",
			memoID,
			"event_id",
			event.ID,
		)
		return fmt.Errorf("failed to submit task: %w", err)
	}

	h.logger.Info(
		"task created and submitted successfully",
		"task_id",
		task.ID(),
		"memo_id",
		memoID,
		"event_id",
		event.ID,
	)
	return nil
}

// application holds all the shared application dependencies to simplify management
// and ensure proper cleanup on shutdown.
type application struct {
	// Configuration
	config *config.Config

	// Core services
	logger *slog.Logger
	db     *sql.DB

	// Stores (using interfaces for proper abstraction)
	userStore          store.UserStore
	taskStore          task.TaskStore
	memoStore          store.MemoStore
	cardStore          store.CardStore
	userCardStatsStore store.UserCardStatsStore

	// Service interfaces
	jwtService        auth.JWTService
	passwordVerifier  auth.PasswordVerifier
	generator         task.Generator
	srsService        srs.Service
	cardService       service.CardService
	memoService       service.MemoService
	cardReviewService card_review.CardReviewService

	// Event system
	eventEmitter events.EventEmitter

	// Task handling
	taskRunner *task.TaskRunner
}

// newApplication creates a new application instance with all dependencies initialized.
// It accepts core dependencies like configuration, logger, and database connection that
// must be established before application initialization.
func newApplication(ctx context.Context, cfg *config.Config, logger *slog.Logger, db *sql.DB) (*application, error) {
	app := &application{
		config: cfg,
		logger: logger,
		db:     db,
	}

	// Initialize JWT service
	var err error
	app.jwtService, err = auth.NewJWTService(cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize JWT service: %w", err)
	}
	logger.Info("JWT authentication service initialized",
		"token_lifetime_minutes", cfg.Auth.TokenLifetimeMinutes)

	// Initialize password verifier
	app.passwordVerifier = auth.NewBcryptVerifier()

	// Initialize stores
	app.userStore = postgres.NewPostgresUserStore(db, bcrypt.DefaultCost)
	app.taskStore = postgres.NewPostgresTaskStore(db)
	app.memoStore = postgres.NewPostgresMemoStore(db, logger)
	app.cardStore = postgres.NewPostgresCardStore(db, logger)
	app.userCardStatsStore = postgres.NewPostgresUserCardStatsStore(db, logger)

	// Create the LLM generator service
	app.generator, err = gemini.NewGenerator(
		ctx,
		logger.With("component", "llm_generator"),
		cfg.LLM,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM generator: %w", err)
	}
	logger.Info("LLM generator initialized successfully")

	// Initialize task runner
	app.taskRunner, err = setupTaskRunner(app)
	if err != nil {
		return nil, fmt.Errorf("failed to setup task runner: %w", err)
	}

	// Initialize event emitter
	app.eventEmitter = events.NewInMemoryEventEmitter(logger)

	// Initialize SRS service
	app.srsService, err = srs.NewDefaultService()
	if err != nil {
		return nil, fmt.Errorf("failed to create SRS service: %w", err)
	}

	// Create required adapters
	memoRepoAdapter := service.NewMemoRepositoryAdapter(app.memoStore, app.db)
	cardRepoAdapter := service.NewCardRepositoryAdapter(app.cardStore, app.db)
	statsRepoAdapter := service.NewStatsRepositoryAdapter(app.userCardStatsStore)

	// Initialize memo service
	app.memoService, err = service.NewMemoService(
		memoRepoAdapter,
		app.taskRunner,
		app.eventEmitter,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create memo service: %w", err)
	}

	// Initialize card service
	app.cardService, err = service.NewCardService(
		cardRepoAdapter,
		statsRepoAdapter,
		app.srsService,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create card service: %w", err)
	}

	// Initialize card review service
	app.cardReviewService, err = card_review.NewCardReviewService(
		app.cardStore,
		app.userCardStatsStore,
		app.srsService,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create card review service: %w", err)
	}

	// Create memo service adapter and task factory for task processing
	memoServiceAdapter, err := task.NewMemoServiceAdapter(memoRepoAdapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create memo service adapter: %w", err)
	}

	// Create task factory
	memoTaskFactory := task.NewMemoGenerationTaskFactory(
		memoServiceAdapter,
		app.generator,
		app.cardService,
		logger,
	)

	// Create and register task factory event handler
	taskFactoryHandler := &TaskFactoryEventHandler{
		taskFactory: memoTaskFactory,
		taskRunner:  app.taskRunner,
		logger:      logger.With("component", "task_factory_event_handler"),
	}

	// Register the event handler with the event emitter
	if emitter, ok := app.eventEmitter.(*events.InMemoryEventEmitter); ok {
		emitter.RegisterHandler(taskFactoryHandler)
	} else {
		return nil, fmt.Errorf("unexpected event emitter type, cannot register task handler")
	}

	logger.Info("Application initialized successfully")
	return app, nil
}

// Run starts the application server, handling lifecycle and cleanup.
// It returns an error if the server fails to start or encounters problems.
func (app *application) Run(ctx context.Context) error {
	// Set up router using the application dependencies
	router := app.setupRouter()

	// Start the HTTP server
	err := app.startHTTPServer(ctx, router)
	if err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// setupTaskRunner initializes and starts the background task processor.
// It uses the application struct to access required dependencies.
func setupTaskRunner(app *application) (*task.TaskRunner, error) {
	// Create the task runner with the configured dependencies
	taskRunner := task.NewTaskRunner(app.taskStore, task.TaskRunnerConfig{
		QueueSize:    app.config.Task.QueueSize,
		WorkerCount:  app.config.Task.WorkerCount,
		StuckTaskAge: time.Duration(app.config.Task.StuckTaskAgeMinutes) * time.Minute,
	}, app.logger)

	// Start the task runner
	if err := taskRunner.Start(); err != nil {
		return nil, fmt.Errorf("failed to start task runner: %w", err)
	}

	return taskRunner, nil
}

// cleanup handles graceful shutdown of application resources.
func (app *application) cleanup() {
	// Stop task runner
	if app.taskRunner != nil {
		app.taskRunner.Stop()
	}

	// Close database connection
	if app.db != nil {
		if err := app.db.Close(); err != nil {
			app.logger.Error("Error closing database connection", "error", err)
		}
	}

	app.logger.Info("Application shutdown completed")
}
