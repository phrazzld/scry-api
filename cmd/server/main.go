package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/events"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"
)

// Constant for the migrations directory path
// Used in the migration command implementation
// This is a relative path from the project root
const migrationsDir = "internal/platform/postgres/migrations"

// TaskFactoryEventHandler is an event handler that creates tasks when events are emitted
type TaskFactoryEventHandler struct {
	taskFactory *task.MemoGenerationTaskFactory
	taskRunner  *task.TaskRunner
	logger      *slog.Logger
}

// HandleEvent processes events by creating and submitting tasks
func (h *TaskFactoryEventHandler) HandleEvent(ctx context.Context, event *events.TaskRequestEvent) error {
	// Only handle memo generation events for now
	if event.Type != task.TaskTypeMemoGeneration {
		h.logger.Debug("ignoring event with unsupported type", "event_type", event.Type, "event_id", event.ID)
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
		h.logger.Error("invalid memo ID", "error", err, "memo_id", payload.MemoID, "event_id", event.ID)
		return fmt.Errorf("invalid memo ID: %w", err)
	}

	// Create the task
	h.logger.Debug("creating task for memo", "memo_id", memoID, "event_id", event.ID)
	task, err := h.taskFactory.CreateTask(memoID)
	if err != nil {
		h.logger.Error("failed to create task", "error", err, "memo_id", memoID, "event_id", event.ID)
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Submit the task to the runner
	h.logger.Debug("submitting task to runner", "task_id", task.ID(), "memo_id", memoID, "event_id", event.ID)
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

// appDependencies holds all the shared application dependencies
// to simplify passing them around between functions.
type appDependencies struct {
	// Configuration
	Config *config.Config

	// Core services
	Logger *slog.Logger
	DB     *sql.DB

	// Stores (using interfaces for proper abstraction)
	UserStore          store.UserStore
	TaskStore          task.TaskStore // Using the interface defined in task.TaskStore
	MemoStore          store.MemoStore
	CardStore          store.CardStore
	UserCardStatsStore store.UserCardStatsStore

	// Repository interfaces for card operations
	CardRepository task.CardRepository // Interface for card operations

	// Services
	JWTService       auth.JWTService
	PasswordVerifier auth.PasswordVerifier
	Generator        task.Generator // Interface for card generation

	// Event system
	EventEmitter events.EventEmitter

	// Task handling
	TaskRunner *task.TaskRunner
}

// main is the entry point for the scry-api server.
// It will be responsible for initializing configuration, setting up logging,
// establishing database connections, injecting dependencies, and starting the
// HTTP server.
func main() {
	// Define migration-related command-line flags
	// These will be used in a future task to implement the migration functionality
	migrateCmd := flag.String("migrate", "", "Run database migrations (up|down|create|status|version)")
	migrationName := flag.String("name", "", "Name for the new migration (only used with 'create')")
	flag.Parse()

	// If a migration command was specified, execute it and exit
	if *migrateCmd != "" {
		slog.Info("Migration requested",
			"command", *migrateCmd,
			"name", *migrationName)

		// Load configuration for migration
		cfg, err := loadConfig()
		if err != nil {
			slog.Error("Failed to load configuration for migration",
				"error", err)
			os.Exit(1)
		}

		// Set up logging with the shared logger setup function
		_, err = setupLogger(cfg)
		if err != nil {
			slog.Error("Failed to set up logger for migration",
				"error", err)
			os.Exit(1)
		}

		// Execute the migration command
		err = runMigrations(cfg, *migrateCmd, *migrationName)
		if err != nil {
			slog.Error("Migration failed",
				"command", *migrateCmd,
				"error", err)
			os.Exit(1)
		}

		slog.Info("Migration completed successfully",
			"command", *migrateCmd)
		os.Exit(0)
	}

	// IMPORTANT: Log messages here use Go's default slog handler (plain text)
	// rather than our custom JSON handler. This is intentional - we can't set up
	// the custom JSON logger until we've loaded configuration, but we still want
	// to log the application startup. This creates a consistent initialization
	// sequence where even initialization errors can be logged.
	slog.Info("Scry API Server starting...")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set up structured logging using the configured log level
	// After this point, all slog calls will use the JSON structured logger
	_, err = setupLogger(cfg)
	if err != nil {
		slog.Error("Failed to set up logger", "error", err)
		os.Exit(1)
	}

	// Log configuration details using structured logging
	slog.Info("Server configuration loaded",
		"port", cfg.Server.Port,
		"log_level", cfg.Server.LogLevel)

	// Log additional configuration details at debug level if available
	if cfg.Database.URL != "" {
		slog.Debug("Database configuration", "url_present", true)
	}
	if cfg.Auth.JWTSecret != "" {
		slog.Debug("Auth configuration", "jwt_secret_present", true)
	}

	// Establish database connection
	_, err = setupDatabase(cfg, slog.Default())
	if err != nil {
		slog.Error("Failed to setup database", "error", err)
		os.Exit(1)
	}
	slog.Info("Database connection established")

	// Initialize JWT authentication service
	_, err = setupJWTService(cfg)
	if err != nil {
		slog.Error("Failed to initialize JWT service", "error", err)
		os.Exit(1)
	}
	slog.Info("JWT authentication service initialized",
		"token_lifetime_minutes", cfg.Auth.TokenLifetimeMinutes)

	// At this point, all required services have been initialized
	slog.Info("Scry API Server initialized successfully",
		"port", cfg.Server.Port)

	// Start the server
	startServer(cfg)
}

// setupRouter creates and configures the application router with all routes and middleware.
// It accepts the application dependencies to create handlers and register routes.
// Returns the configured router.
func setupRouter(deps *appDependencies) *chi.Mux {
	// Create a router
	r := chi.NewRouter()

	// Apply standard middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Create the password verifier
	passwordVerifier := auth.NewBcryptVerifier()

	// Create API handlers (user service will be created later when needed)
	authHandler := api.NewAuthHandler(deps.UserStore, deps.JWTService, passwordVerifier, &deps.Config.Auth)
	authMiddleware := authmiddleware.NewAuthMiddleware(deps.JWTService)

	// Create adapter for the store to be used in the service layer
	memoRepoAdapter := service.NewMemoRepositoryAdapter(deps.MemoStore, deps.DB)

	// Create the memo service with the event emitter from dependencies
	memoService := service.NewMemoService(memoRepoAdapter, deps.TaskRunner, deps.EventEmitter, deps.Logger)
	memoHandler := api.NewMemoHandler(memoService)

	// Register routes
	r.Route("/api", func(r chi.Router) {
		// Authentication endpoints (public)
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.RefreshToken)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			// Memo endpoints
			r.Post("/memos", memoHandler.CreateMemo)
		})
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			deps.Logger.Error("Failed to write health check response", "error", err)
		}
	})

	return r
}

// startServer starts the HTTP server with graceful shutdown.
// It's split from main() to improve readability and testability.
func startServer(cfg *config.Config) {
	// Step 1: Set up database connection and logger
	logger, err := setupLogger(cfg)
	if err != nil {
		slog.Error("Failed to set up logger", "error", err)
		os.Exit(1)
	}

	db, err := setupDatabase(cfg, logger)
	if err != nil {
		logger.Error("Failed to setup database", "error", err)
		os.Exit(1)
	}

	// Ensure database connection is closed when the server shuts down
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Error closing database connection", "error", err)
		}
	}()

	// Step 2: Initialize JWT service
	jwtService, err := setupJWTService(cfg)
	if err != nil {
		logger.Error("Failed to initialize JWT service", "error", err)
		os.Exit(1)
	}

	// Step 3: Initialize stores and other dependencies
	userStore := postgres.NewPostgresUserStore(db, bcrypt.DefaultCost)
	taskStore := postgres.NewPostgresTaskStore(db) // Concrete implementation that satisfies task.TaskStore
	memoStore := postgres.NewPostgresMemoStore(db, logger)
	cardStore := postgres.NewPostgresCardStore(db, logger)
	userCardStatsStore := postgres.NewPostgresUserCardStatsStore(db, logger)
	passwordVerifier := auth.NewBcryptVerifier()

	// Create a mock generator service for card generation
	mockGenerator := mocks.NewMockGeneratorWithDefaultCards(uuid.Nil, uuid.Nil)

	// Step 4: Populate the application dependencies struct
	deps := &appDependencies{
		Config:             cfg,
		Logger:             logger,
		DB:                 db,
		UserStore:          userStore,
		TaskStore:          taskStore,
		MemoStore:          memoStore,
		CardStore:          cardStore,
		UserCardStatsStore: userCardStatsStore,
		// MemoRepository removed - using MemoStore with adapter instead
		CardRepository:   cardStore, // Now using the real CardStore implementation
		Generator:        mockGenerator,
		JWTService:       jwtService,
		PasswordVerifier: passwordVerifier,
	}

	// Step 5: Set up task runner using the new setup function
	taskRunner, err := setupTaskRunner(deps)
	if err != nil {
		logger.Error("Failed to setup task runner", "error", err)
		os.Exit(1)
	}

	// Update dependencies with the task runner
	deps.TaskRunner = taskRunner

	// Step 6: Set up event emitter
	eventEmitter := events.NewInMemoryEventEmitter(logger)

	// Create a memo repository adapter
	memoRepoAdapter := service.NewMemoRepositoryAdapter(deps.MemoStore, deps.DB)

	// Create a memo service adapter
	memoServiceAdapter, err := task.NewMemoServiceAdapter(memoRepoAdapter)
	if err != nil {
		logger.Error("Failed to create memo service adapter", "error", err)
		os.Exit(1)
	}

	// Create the task factory
	memoTaskFactory := task.NewMemoGenerationTaskFactory(
		memoServiceAdapter,
		deps.Generator,
		deps.CardRepository,
		logger,
	)

	// Create and register task factory event handler
	taskFactoryHandler := &TaskFactoryEventHandler{
		taskFactory: memoTaskFactory,
		taskRunner:  taskRunner,
		logger:      logger.With("component", "task_factory_event_handler"),
	}

	// Register the event handler with the event emitter
	eventEmitter.RegisterHandler(taskFactoryHandler)

	// Add event emitter to dependencies
	deps.EventEmitter = eventEmitter

	// Ensure task runner is stopped when the server shuts down
	defer taskRunner.Stop()

	// Step 7: Set up router using the new setup function
	router := setupRouter(deps)

	// Step 8: Configure and create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// Step 9: Start server in a goroutine to allow for graceful shutdown
	go func() {
		logger.Info("Starting server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Step 10: Set up graceful shutdown
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignal

	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", "error", err)
	}

	logger.Info("Server shutdown completed")
}

// loadConfig loads the application configuration from environment variables or config file.
// Returns the loaded config and any loading error.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, nil
}

// setupLogger configures and initializes the application logger based on config settings.
// Returns the configured logger or an error if setup fails.
func setupLogger(cfg *config.Config) (*slog.Logger, error) {
	loggerConfig := logger.LoggerConfig{
		Level: cfg.Server.LogLevel,
	}

	l, err := logger.Setup(loggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to set up logger: %w", err)
	}

	return l, nil
}

// setupDatabase establishes a connection to the database and configures connection pools.
// Returns the database connection if successful, or an error if the connection fails.
func setupDatabase(cfg *config.Config, logger *slog.Logger) (*sql.DB, error) {
	// Open database connection
	db, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool with reasonable defaults
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// setupJWTService initializes and configures the JWT authentication service.
// Returns the configured service or an error if setup fails.
func setupJWTService(cfg *config.Config) (auth.JWTService, error) {
	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("JWT secret cannot be empty")
	}

	// Initialize the JWT service with configuration
	jwtService, err := auth.NewJWTService(cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT service: %w", err)
	}

	return jwtService, nil
}

// setupTaskRunner initializes and starts the background task processor.
// Takes a fully populated appDependencies struct and returns a started TaskRunner.
func setupTaskRunner(deps *appDependencies) (*task.TaskRunner, error) {
	// Create the task runner with the configured dependencies
	taskRunner := task.NewTaskRunner(deps.TaskStore, task.TaskRunnerConfig{
		QueueSize:    deps.Config.Task.QueueSize,
		WorkerCount:  deps.Config.Task.WorkerCount,
		StuckTaskAge: time.Duration(deps.Config.Task.StuckTaskAgeMinutes) * time.Minute,
	}, deps.Logger)

	// Start the task runner
	if err := taskRunner.Start(); err != nil {
		return nil, fmt.Errorf("failed to start task runner: %w", err)
	}

	return taskRunner, nil
}

// slogGooseLogger adapts slog for goose's logger interface
type slogGooseLogger struct{}

// Printf implements the goose.Logger Printf method by forwarding messages to slog.Info
func (l *slogGooseLogger) Printf(format string, v ...interface{}) {
	slog.Info(fmt.Sprintf(format, v...))
}

// Fatalf implements the goose.Logger Fatalf method by forwarding error messages to slog.Error
// Note: Unlike the standard Fatalf behavior, this does NOT call os.Exit
// to allow main.go to handle application exit consistently
func (l *slogGooseLogger) Fatalf(format string, v ...interface{}) {
	slog.Error(fmt.Sprintf(format, v...))
	// Deliberately NOT calling os.Exit(1) here
	// The error will be returned to main which will handle the exit
}

// runMigrations handles the execution of database migrations.
// It connects to the database using configuration from cfg, then executes
// the specified migration command (up, down, status, create, version).
// The args parameter is used for command-specific arguments, such as
// the migration name when creating a new migration.
//
// This function encapsulates all migration-related logic and will be expanded
// in future tasks to handle different migration commands
func runMigrations(cfg *config.Config, command string, args ...string) error {
	// Configure goose to use the custom slog logger adapter
	goose.SetLogger(&slogGooseLogger{})

	// pgx driver is automatically registered with database/sql
	// when the stdlib package is imported

	// Validate database URL before attempting to connect
	if cfg.Database.URL == "" {
		return fmt.Errorf("database URL is empty: check your configuration")
	}

	// Open a database connection using the configured Database URL
	slog.Info("Opening database connection for migrations")
	db, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		return fmt.Errorf(
			"failed to open database connection: %w (check connection string format and credentials)",
			err,
		)
	}

	// Ensure the database connection is closed when the function returns
	defer func() {
		slog.Debug("Closing database connection")
		err := db.Close()
		if err != nil {
			slog.Error("Error closing database connection",
				"error", err,
				"error_type", fmt.Sprintf("%T", err))
		}
	}()

	// Set connection pool parameters
	db.SetMaxOpenConns(5)                  // Limit connections to avoid overwhelming the database
	db.SetMaxIdleConns(2)                  // Keep a few connections ready
	db.SetConnMaxLifetime(time.Minute * 5) // Recreate connections that have been open too long

	// Verify database connectivity with a ping
	slog.Debug("Verifying database connection with ping")
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		// Check for specific error types and provide targeted advice

		// Context deadline exceeded error (connection timeout)
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf(
				"database ping timed out after 5s: %w (check network connectivity, firewall rules, and server load)",
				err,
			)
		}

		// Network-related errors
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return fmt.Errorf(
					"network timeout connecting to database: %w (check network latency and database server load)",
					err,
				)
			}
			return fmt.Errorf(
				"network error connecting to database: %w (check hostname, port, and network connectivity)",
				err,
			)
		}

		// Return a generic error message if no specific error type was matched
		return fmt.Errorf(
			"failed to connect to database: %w (check connection string, credentials, and database availability)",
			err,
		)
	}

	// Set the migration directory
	slog.Debug("Setting up migration directory", "dir", migrationsDir)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Execute the requested migration command
	slog.Info("Executing migration command", "command", command)

	switch command {
	case "up":
		err = goose.Up(db, migrationsDir)
	case "down":
		err = goose.Down(db, migrationsDir)
	case "reset":
		err = goose.Reset(db, migrationsDir)
	case "status":
		err = goose.Status(db, migrationsDir)
	case "version":
		err = goose.Version(db, migrationsDir)
	case "create":
		// The migration name is required when creating a new migration
		if len(args) == 0 || args[0] == "" {
			return fmt.Errorf("migration name is required for 'create' command")
		}

		// Define the migration type (SQL by default)
		migrationName := args[0]
		err = goose.Create(db, migrationsDir, migrationName, "sql")
	default:
		return fmt.Errorf(
			"unknown migration command: %s (expected up, down, reset, status, version, or create)",
			command,
		)
	}

	if err != nil {
		return fmt.Errorf("migration command '%s' failed: %w", command, err)
	}

	return nil
}
