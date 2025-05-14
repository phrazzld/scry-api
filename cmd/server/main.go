//go:build exported_core_functions

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
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx driver for database/sql
	"github.com/phrazzld/scry-api/internal/api"
	apiMiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/events"
	"github.com/phrazzld/scry-api/internal/platform/gemini"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/phrazzld/scry-api/internal/testdb" // Import for standardized functions
	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"
)

// migrationsDir is the relative path to migrations from the project root
// This is deprecated in favor of testdb.FindMigrationsDir() but kept for compatibility
// with existing code that expects this constant
const migrationsDir = "internal/platform/postgres/migrations"

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
	CardRepository store.CardStore // Interface for card operations

	// Services
	JWTService        auth.JWTService
	PasswordVerifier  auth.PasswordVerifier
	Generator         task.Generator                // Interface for card generation
	SRSService        srs.Service                   // Interface for SRS algorithm operations
	CardService       service.CardService           // Interface for card service operations
	MemoService       service.MemoService           // Interface for memo service operations
	CardReviewService card_review.CardReviewService // Interface for card review operations

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
	migrateCmd := flag.String(
		"migrate",
		"",
		"Run database migrations (up|down|create|status|version)",
	)
	migrationName := flag.String("name", "", "Name for the new migration (only used with 'create')")
	verbose := flag.Bool("verbose", false, "Enable verbose logging for migrations")
	verifyOnly := flag.Bool("verify-migrations", false, "Only verify migrations without applying them")
	validateMigrations := flag.Bool("validate-migrations", false, "Validate applied migrations (returns non-zero exit code on failure)")
	flag.Parse()

	// If a migration command was specified or validation is requested, handle and exit
	if *migrateCmd != "" || *validateMigrations {
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

		// Handle different migration-related commands
		if *validateMigrations {
			// Validate applied migrations (mostly used in CI)
			slog.Info("Validating applied migrations",
				"verbose", *verbose,
				"mode", getExecutionMode())
			err = validateAppliedMigrations(cfg, *verbose)
		} else if *verifyOnly {
			// Only verify migrations without applying them
			slog.Info("Verifying migrations only (not applying)",
				"command", *migrateCmd,
				"verbose", *verbose)
			err = verifyMigrations(cfg, *verbose)
		} else {
			// Normal migration execution
			slog.Info("Executing migrations",
				"command", *migrateCmd,
				"verbose", *verbose,
				"name", *migrationName)
			err = runMigrations(cfg, *migrateCmd, *verbose, *migrationName)
		}

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
	r.Use(
		apiMiddleware.NewTraceMiddleware(deps.Logger),
	) // Add trace IDs for improved error handling

	// Create the password verifier
	passwordVerifier := auth.NewBcryptVerifier()

	// Create API handlers (user service will be created later when needed)
	authHandler := api.NewAuthHandler(
		deps.UserStore,
		deps.JWTService,
		passwordVerifier,
		&deps.Config.Auth,
		deps.Logger,
	)
	authMiddleware := apiMiddleware.NewAuthMiddleware(deps.JWTService)

	// Use memo service from dependencies, which has been properly initialized in startServer
	memoHandler := api.NewMemoHandler(deps.MemoService, deps.Logger)

	// Use card service directly from dependencies (now properly typed as service.CardService)
	cardHandler := api.NewCardHandler(deps.CardReviewService, deps.CardService, deps.Logger)

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

			// Card review endpoints
			r.Get("/cards/next", cardHandler.GetNextReviewCard)
			r.Post("/cards/{id}/answer", cardHandler.SubmitAnswer)

			// Card management endpoints
			r.Put("/cards/{id}", cardHandler.EditCard)
			r.Delete("/cards/{id}", cardHandler.DeleteCard)
			r.Post("/cards/{id}/postpone", cardHandler.PostponeCard)
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
	taskStore := postgres.NewPostgresTaskStore(
		db,
	) // Concrete implementation that satisfies task.TaskStore
	memoStore := postgres.NewPostgresMemoStore(db, logger)
	cardStore := postgres.NewPostgresCardStore(db, logger)
	userCardStatsStore := postgres.NewPostgresUserCardStatsStore(db, logger)
	passwordVerifier := auth.NewBcryptVerifier()

	// Create the appropriate generator service for card generation based on build tags
	generator, err := gemini.NewGenerator(
		context.Background(),
		logger.With("component", "llm_generator"),
		cfg.LLM,
	)
	if err != nil {
		logger.Error("Failed to initialize LLM generator", "error", err)
		os.Exit(1)
	}
	logger.Info("LLM generator initialized successfully")

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
		Generator:        generator,
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
	// Add event emitter to dependencies immediately so it can be used by services
	deps.EventEmitter = eventEmitter

	// Create a memo repository adapter
	memoRepoAdapter := service.NewMemoRepositoryAdapter(deps.MemoStore, deps.DB)

	// Create memo service
	memoService, err := service.NewMemoService(
		memoRepoAdapter,
		deps.TaskRunner,
		deps.EventEmitter,
		logger,
	)
	if err != nil {
		logger.Error("Failed to create memo service", "error", err)
		os.Exit(1)
	}
	deps.MemoService = memoService

	// Create a memo service adapter for tasks
	memoServiceAdapter, err := task.NewMemoServiceAdapter(memoRepoAdapter)
	if err != nil {
		logger.Error("Failed to create memo service adapter", "error", err)
		os.Exit(1)
	}

	// Create SRS service with default parameters
	srsService, err := srs.NewDefaultService()
	if err != nil {
		logger.Error("Failed to create SRS service", "error", err)
		os.Exit(1)
	}
	// Store SRS service in dependencies for use by other services
	deps.SRSService = srsService

	// Create a card repository adapter for the card service
	cardRepoAdapter := service.NewCardRepositoryAdapter(deps.CardStore, deps.DB)
	statsRepoAdapter := service.NewStatsRepositoryAdapter(deps.UserCardStatsStore)

	// Create the card service using SRS service from dependencies
	cardService, err := service.NewCardService(cardRepoAdapter, statsRepoAdapter, deps.SRSService, logger)
	if err != nil {
		logger.Error("Failed to create card service", "error", err)
		os.Exit(1)
	}
	deps.CardService = cardService

	// Create card review service with direct store dependencies
	cardReviewService, err := card_review.NewCardReviewService(
		deps.CardStore,
		deps.UserCardStatsStore,
		deps.SRSService, // Use SRS service from dependencies
		logger,
	)
	if err != nil {
		logger.Error("Failed to create card review service", "error", err)
		os.Exit(1)
	}
	deps.CardReviewService = cardReviewService

	// Create the task factory - ensuring all services are initialized first
	memoTaskFactory := task.NewMemoGenerationTaskFactory(
		memoServiceAdapter,
		deps.Generator,
		deps.CardService, // CardService is now properly initialized with srsService
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

// executeMigration is a centralized function that executes migration commands.
// It encapsulates all the common logic for database connectivity, error handling,
// and migration execution. This is the core migration execution logic that is used
// by both runMigrations and verifyMigrations.
func executeMigration(cfg *config.Config, command string, verbose bool, args ...string) error {
	// Use a correlation ID for all migration logs to allow tracing the entire operation
	correlationID := uuid.New().String()
	migrationLogger := slog.Default().With(
		"correlation_id", correlationID,
		"component", "migrations",
		"command", command,
	)

	startTime := time.Now()
	migrationLogger.Info("Starting migration operation",
		"operation", fmt.Sprintf("goose %s", command),
		"verbose", verbose,
		"mode", getExecutionMode())

	// Configure goose to use the custom slog logger adapter
	goose.SetLogger(&slogGooseLogger{})

	// Determine database URL to use - prioritize standardized function if in test context
	// or fall back to config URL
	dbURL := cfg.Database.URL

	// Try to use the standardized function if in test mode or CI
	testDBURL := ""

	// Use the standardized function when available
	if testDBURL = testdb.GetTestDatabaseURL(); testDBURL != "" {
		migrationLogger.Info("Using standardized test database URL",
			"source", "testdb.GetTestDatabaseURL")
		dbURL = testDBURL
	}

	// Validate database URL
	if dbURL == "" {
		migrationLogger.Error("Database URL is empty",
			"error", "missing configuration",
			"resolution", "check DATABASE_URL environment variable or config file")
		return fmt.Errorf("database URL is empty: check your configuration")
	}

	// Always log database URL (masked) for diagnostics
	// Mask the password in the URL for safe logging
	safeURL := maskDatabaseURL(dbURL)
	dbURLSource := detectDatabaseURLSource(dbURL)
	migrationLogger.Info("Using database URL",
		"url", safeURL,
		"source", dbURLSource,
		"host", extractHostFromURL(dbURL))

	// Open a database connection using the database URL
	migrationLogger.Info("Opening database connection for migrations")
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		migrationLogger.Error("Failed to open database connection",
			"error", err,
			"error_type", fmt.Sprintf("%T", err))
		return fmt.Errorf(
			"failed to open database connection: %w (check connection string format and credentials)",
			err,
		)
	}

	// Ensure the database connection is closed when the function returns
	defer func() {
		migrationLogger.Debug("Closing database connection")
		err := db.Close()
		if err != nil {
			migrationLogger.Error("Error closing database connection",
				"error", err,
				"error_type", fmt.Sprintf("%T", err))
		}

		// Log the total execution time at the end
		duration := time.Since(startTime)
		migrationLogger.Info("Migration operation completed",
			"operation", fmt.Sprintf("goose %s", command),
			"duration_ms", duration.Milliseconds(),
			"success", err == nil)
	}()

	// Set connection pool parameters
	db.SetMaxOpenConns(5)                  // Limit connections to avoid overwhelming the database
	db.SetMaxIdleConns(2)                  // Keep a few connections ready
	db.SetConnMaxLifetime(time.Minute * 5) // Recreate connections that have been open too long
	migrationLogger.Debug("Configured database connection pool",
		"max_open_conns", 5,
		"max_idle_conns", 2,
		"conn_max_lifetime_minutes", 5)

	// Verify database connectivity with a ping
	migrationLogger.Debug("Verifying database connection with ping")
	pingStartTime := time.Now()
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		pingDuration := time.Since(pingStartTime)
		migrationLogger.Error("Database ping failed",
			"error", err,
			"duration_ms", pingDuration.Milliseconds(),
			"error_type", fmt.Sprintf("%T", err))

		// Check for specific error types and provide targeted advice
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

	pingDuration := time.Since(pingStartTime)
	migrationLogger.Info("Database connection verified successfully",
		"duration_ms", pingDuration.Milliseconds())

	// In CI or verbose mode, log database connection information
	if verbose || os.Getenv("CI") != "" {
		logDatabaseInfo(db, pingCtx, migrationLogger)
	}

	// Try to use the standardized function for migrations directory
	migrationsDirPath := ""

	// First try using the standardized FindMigrationsDir function
	migrationsPath, migrationsErr := testdb.FindMigrationsDir()
	if migrationsErr == nil && directoryExists(migrationsPath) {
		migrationsDirPath = migrationsPath
		migrationLogger.Info("Using standardized migrations directory path",
			"path", migrationsDirPath,
			"source", "testdb.FindMigrationsDir")
	} else {
		// Fall back to the traditional approach if the standardized function fails
		migrationLogger.Debug("Standardized migrations path detection failed, using fallback",
			"error", migrationsErr)

		// Fallback to the constant path
		migrationsDirPath = migrationsDir

		// Get the absolute path for better logging
		absPath, err := filepath.Abs(migrationsDirPath)
		if err != nil {
			migrationLogger.Warn("Could not resolve absolute path for migrations directory",
				"relative_path", migrationsDirPath,
				"error", err)
			// Continue with relative path
		} else {
			migrationsDirPath = absPath
		}

		// Check if directory exists before proceeding
		if !directoryExists(migrationsDirPath) {
			// Try to find migrations relative to current directory
			cwd, _ := os.Getwd()
			altPath := filepath.Join(cwd, migrationsDir)
			migrationLogger.Warn("Migrations directory not found at specified path, trying alternative",
				"original_path", migrationsDirPath,
				"alternative_path", altPath)

			if directoryExists(altPath) {
				migrationsDirPath = altPath
				migrationLogger.Info("Found migrations at alternative path", "path", migrationsDirPath)
			} else {
				migrationLogger.Error("Failed to locate migrations directory",
					"original_path", migrationsDirPath,
					"alternative_path", altPath)
				return fmt.Errorf("migrations directory not found at %s or %s", migrationsDirPath, altPath)
			}
		}
	}

	// Log the migration directory path we're using
	migrationLogger.Info("Using migrations directory",
		"path", migrationsDirPath,
		"exists", directoryExists(migrationsDirPath))

	// List available migration files (always do this for better logging)
	migFilesData, err := enumerateMigrationFiles(migrationsDirPath)
	if err != nil {
		migrationLogger.Warn("Failed to read migrations directory", "error", err)
	} else {
		// Log migration files information
		migrationLogger.Info("Found migration files",
			"count", len(migFilesData.Files),
			"sql_count", migFilesData.SQLCount,
			"newest_file", migFilesData.NewestFile,
			"oldest_file", migFilesData.OldestFile)

		// In verbose mode, log all files
		if verbose || os.Getenv("CI") != "" {
			migrationLogger.Info("Migration files list", "files", migFilesData.Files)
		}
	}

	// Set the migration directory
	migrationLogger.Debug("Setting up migration configuration", "dir", migrationsDirPath)

	// Set the dialect
	dialectStartTime := time.Now()
	if err := goose.SetDialect("postgres"); err != nil {
		migrationLogger.Error("Failed to set dialect", "error", err)
		return fmt.Errorf("failed to set dialect: %w", err)
	}
	migrationLogger.Debug("Set database dialect",
		"dialect", "postgres",
		"duration_ms", time.Since(dialectStartTime).Milliseconds())

	// Set the migration table name using the standardized constant
	migrationTableName := testdb.MigrationTableName
	goose.SetTableName(migrationTableName)
	migrationLogger.Debug("Set migration table name", "table", migrationTableName)

	// Execute the requested migration command
	migrationLogger.Info("Starting migration command execution",
		"command", command,
		"args", args)

	// Log current database migration version before executing command
	var currentVersion string
	versionErr := db.QueryRow(fmt.Sprintf("SELECT version_id FROM %s ORDER BY version_id DESC LIMIT 1", migrationTableName)).
		Scan(&currentVersion)
	if versionErr != nil {
		if errors.Is(versionErr, sql.ErrNoRows) {
			migrationLogger.Info("No migrations currently applied", "status", "clean database")
			currentVersion = "0"
		} else {
			migrationLogger.Warn("Failed to retrieve current migration version",
				"error", versionErr,
				"error_type", fmt.Sprintf("%T", versionErr))
		}
	} else {
		migrationLogger.Info("Current database migration version", "version", currentVersion)
	}

	// Execute the command with timing
	commandStartTime := time.Now()

	switch command {
	case "up":
		migrationLogger.Info("Applying pending migrations")
		err = goose.Up(db, migrationsDirPath)
	case "down":
		migrationLogger.Info("Rolling back one migration version")
		err = goose.Down(db, migrationsDirPath)
	case "reset":
		migrationLogger.Info("Resetting all migrations (roll back to zero)")
		err = goose.Reset(db, migrationsDirPath)
	case "status":
		migrationLogger.Info("Checking migration status")
		err = goose.Status(db, migrationsDirPath)
	case "version":
		migrationLogger.Info("Retrieving current migration version")
		err = goose.Version(db, migrationsDirPath)
	case "create":
		// The migration name is required when creating a new migration
		if len(args) == 0 || args[0] == "" {
			migrationLogger.Error("Migration create command requires a name parameter")
			return fmt.Errorf("migration name is required for 'create' command")
		}

		// Define the migration type (SQL by default)
		migrationName := args[0]
		migrationLogger.Info("Creating new migration",
			"name", migrationName,
			"type", "sql",
			"directory", migrationsDirPath)
		err = goose.Create(db, migrationsDirPath, migrationName, "sql")
	default:
		migrationLogger.Error("Unknown migration command",
			"command", command,
			"valid_commands", []string{"up", "down", "reset", "status", "version", "create"})
		return fmt.Errorf(
			"unknown migration command: %s (expected up, down, reset, status, version, or create)",
			command,
		)
	}

	// Log command execution time
	commandDuration := time.Since(commandStartTime)

	if err != nil {
		migrationLogger.Error("Migration command failed",
			"command", command,
			"error", err,
			"error_type", fmt.Sprintf("%T", err),
			"duration_ms", commandDuration.Milliseconds())
		return fmt.Errorf("migration command '%s' failed: %w", command, err)
	}

	migrationLogger.Info("Migration command executed successfully",
		"command", command,
		"duration_ms", commandDuration.Milliseconds())

	// Check if migration version changed
	if command == "up" || command == "down" || command == "reset" {
		// Get new version
		var newVersion string
		newVersionErr := db.QueryRow(fmt.Sprintf("SELECT version_id FROM %s ORDER BY version_id DESC LIMIT 1", migrationTableName)).
			Scan(&newVersion)
		if newVersionErr != nil {
			if errors.Is(newVersionErr, sql.ErrNoRows) {
				migrationLogger.Info(
					"Database schema is now at base version",
					"new_version",
					"0",
					"previous_version",
					currentVersion,
				)
			} else {
				migrationLogger.Warn("Failed to retrieve new migration version", "error", newVersionErr)
			}
		} else if newVersion != currentVersion {
			migrationLogger.Info("Database schema version changed",
				"previous_version", currentVersion,
				"new_version", newVersion)
		} else {
			migrationLogger.Info("Database schema version unchanged", "version", newVersion)
		}
	}

	// Verify migrations applied successfully
	if command == "up" && (verbose || os.Getenv("CI") != "") {
		if verifyErr := verifyAppliedMigrations(db, migrationLogger); verifyErr != nil {
			migrationLogger.Error("Migration verification failed",
				"error", verifyErr,
				"error_type", fmt.Sprintf("%T", verifyErr))
			return fmt.Errorf("migration verification failed: %w", verifyErr)
		}
	}

	return nil
}

// logDatabaseInfo logs detailed information about the database connection
// This is used for diagnostics in CI and with verbose mode
func logDatabaseInfo(db *sql.DB, ctx context.Context, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info("Gathering detailed database information")
	infoStartTime := time.Now()

	// Log database version
	var version string
	versionErr := db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if versionErr != nil {
		logger.Warn("Failed to query database version",
			"error", versionErr,
			"error_type", fmt.Sprintf("%T", versionErr))
	} else {
		logger.Info("Database version information",
			"system", "PostgreSQL",
			"version", version)
	}

	// Log database user
	var user string
	userErr := db.QueryRowContext(ctx, "SELECT current_user").Scan(&user)
	if userErr != nil {
		logger.Warn("Failed to query current database user",
			"error", userErr,
			"error_type", fmt.Sprintf("%T", userErr))
	} else {
		logger.Info("Database connection credentials", "user", user)
	}

	// Log database name
	var dbName string
	dbNameErr := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName)
	if dbNameErr != nil {
		logger.Warn("Failed to query current database name",
			"error", dbNameErr,
			"error_type", fmt.Sprintf("%T", dbNameErr))
	} else {
		logger.Info("Connected to database", "database", dbName)
	}

	// Check if migrations table exists
	migrationTable := testdb.MigrationTableName
	var migTableExists bool
	tableQuery := fmt.Sprintf(
		"SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = '%s')",
		migrationTable,
	)
	tableErr := db.QueryRowContext(ctx, tableQuery).Scan(&migTableExists)
	if tableErr != nil {
		logger.Warn("Failed to check for migrations table",
			"error", tableErr,
			"table", migrationTable)
	} else {
		logger.Info("Migration tracking table",
			"table", migrationTable,
			"exists", migTableExists)

		// If table exists, check the migration count
		if migTableExists {
			var migrationCount int
			countErr := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", migrationTable)).Scan(&migrationCount)
			if countErr != nil {
				logger.Warn("Failed to count applied migrations", "error", countErr)
			} else {
				logger.Info("Migration status summary", "count", migrationCount)
			}
		}
	}

	logger.Debug("Database information gathering completed",
		"duration_ms", time.Since(infoStartTime).Milliseconds())
}

// verifyAppliedMigrations verifies and logs information about applied migrations
// Returns an error if verification fails or if there's a problem with the migrations
func verifyAppliedMigrations(db *sql.DB, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info("Verifying applied migrations")
	verifyStartTime := time.Now()

	// Check migration count in the database
	migrationTable := testdb.MigrationTableName
	var migrationCount int
	countErr := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", migrationTable)).Scan(&migrationCount)
	if countErr != nil {
		errMsg := "Failed to verify migration count"
		logger.Error(errMsg,
			"error", countErr,
			"error_type", fmt.Sprintf("%T", countErr))
		return fmt.Errorf("%s: %w", errMsg, countErr)
	}

	logger.Info("Verification in progress",
		"migrations_count", migrationCount,
		"table", migrationTable)

	// Verify migration files on disk
	migrationsPath, err := getMigrationsPath()
	if err != nil {
		errMsg := "Failed to locate migrations directory"
		logger.Error(errMsg, "error", err)
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	migFilesData, err := enumerateMigrationFiles(migrationsPath)
	if err != nil {
		errMsg := "Failed to read migrations directory"
		logger.Error(errMsg, "error", err)
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	// Check if SQL migration count matches the expected number
	sqlFileCount := migFilesData.SQLCount
	if migrationCount < sqlFileCount {
		errMsg := "Not all migrations have been applied"
		logger.Error(errMsg,
			"applied_migrations", migrationCount,
			"expected_migrations", sqlFileCount)
		return fmt.Errorf("%s: found %d applied migrations but expected %d",
			errMsg, migrationCount, sqlFileCount)
	}

	// List applied migrations and check for any that failed to apply
	rows, queryErr := db.Query(
		fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY version_id", migrationTable),
	)
	if queryErr != nil {
		errMsg := "Failed to query migration history"
		logger.Error(errMsg,
			"error", queryErr,
			"error_type", fmt.Sprintf("%T", queryErr))
		return fmt.Errorf("%s: %w", errMsg, queryErr)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			logger.Warn("Failed to close rows", "error", err)
		}
	}()

	// Collect all migrations and check for failures
	appliedMigrations := make([]string, 0, migrationCount)
	failedMigrations := make([]string, 0)

	for rows.Next() {
		var versionID string
		var isApplied bool
		if err := rows.Scan(&versionID, &isApplied); err != nil {
			logger.Warn("Failed to scan migration row", "error", err)
			continue
		}

		if isApplied {
			appliedMigrations = append(appliedMigrations, versionID)
		} else {
			failedMigrations = append(failedMigrations, versionID)
		}
	}

	// Check for any errors during row iteration
	if err := rows.Err(); err != nil {
		errMsg := "Error while iterating migration rows"
		logger.Error(errMsg, "error", err)
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	// Log the collected migrations
	logger.Info("Applied migration versions",
		"versions", appliedMigrations,
		"count", len(appliedMigrations))

	// If there are any failed migrations, return an error
	if len(failedMigrations) > 0 {
		errMsg := "Some migrations failed to apply"
		logger.Error(errMsg,
			"failed_versions", failedMigrations,
			"count", len(failedMigrations))
		return fmt.Errorf("%s: %v", errMsg, failedMigrations)
	}

	// Verify the latest migration matches the expected latest migration
	if len(appliedMigrations) > 0 {
		latestApplied := appliedMigrations[len(appliedMigrations)-1]
		if migFilesData.LatestVersion != "" && latestApplied != migFilesData.LatestVersion {
			errMsg := "Latest applied migration does not match expected latest version"
			logger.Error(errMsg,
				"latest_applied", latestApplied,
				"expected_latest", migFilesData.LatestVersion)
			return fmt.Errorf("%s: got %s but expected %s",
				errMsg, latestApplied, migFilesData.LatestVersion)
		}
	}

	logger.Info("Migration verification completed successfully",
		"duration_ms", time.Since(verifyStartTime).Milliseconds(),
		"migrations_applied", len(appliedMigrations))

	return nil
}

// runMigrations handles the execution of database migrations.
// It connects to the database using configuration from cfg, then executes
// the specified migration command (up, down, status, create, version).
// The args parameter is used for command-specific arguments, such as
// the migration name when creating a new migration.
//
// This function encapsulates all migration-related logic and will be expanded
// in future tasks to handle different migration commands
func runMigrations(cfg *config.Config, command string, verbose bool, args ...string) error {
	return executeMigration(cfg, command, verbose, args...)
}

// verifyMigrations checks if migrations can be applied without actually applying them.
// It validates database connectivity, migration table existence, and migration files.
func verifyMigrations(cfg *config.Config, verbose bool) error {
	slog.Info("Verifying database migrations setup")

	// Use the centralized migration function with the "status" command
	// This will check connectivity, locate migrations, and verify database
	// state without applying any changes
	return executeMigration(cfg, "status", true)
}

// validateAppliedMigrations checks if all migrations have been successfully applied.
// This is primarily used in CI to ensure that migrations are successfully applied.
// Returns an error if any migrations are missing or failed to apply.
func validateAppliedMigrations(cfg *config.Config, verbose bool) error {
	logger := slog.Default().With(
		"component", "migration_validator",
		"mode", getExecutionMode(),
	)

	logger.Info("Starting migration validation")

	// Determine which database URL to use
	dbURL := cfg.Database.URL

	// Try to use the standardized test database URL if available
	testDBURL := testdb.GetTestDatabaseURL()
	if testDBURL != "" {
		logger.Info("Using standardized test database URL",
			"source", "testdb.GetTestDatabaseURL")
		dbURL = testDBURL
	}

	// Open database connection to check migrations
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		logger.Error("Failed to open database connection",
			"error", err,
			"error_type", fmt.Sprintf("%T", err))
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn("Failed to close database connection", "error", err)
		}
	}()

	// Ping the database to verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("Database ping failed",
			"error", err,
			"error_type", fmt.Sprintf("%T", err))
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check and verify the migration status
	err = verifyAppliedMigrations(db, logger)
	if err != nil {
		// Enhanced error messages for CI environment
		if isCIEnvironment() {
			logger.Error("Migration validation failed - CI CRITICAL ERROR",
				"error", err,
				"details", "This will cause CI failure")
			return fmt.Errorf("CI MIGRATION VALIDATION FAILED: %w", err)
		}
		return fmt.Errorf("migration validation failed: %w", err)
	}

	logger.Info("Migration validation completed successfully",
		"result", "all migrations properly applied")
	return nil
}

// getExecutionMode returns a string describing the execution environment
// This helps with log filtering and diagnostic analysis
func getExecutionMode() string {
	if isCIEnvironment() {
		return "ci"
	}
	return "local"
}

// isCIEnvironment returns true if running in a CI environment
func isCIEnvironment() bool {
	return os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""
}

// maskDatabaseURL masks the password in a database URL for safe logging.
func maskDatabaseURL(dbURL string) string {
	// Parse the URL
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return "invalid-url"
	}

	// Mask the password if user info exists
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		parsedURL.User = url.UserPassword(username, "****")
		return parsedURL.String()
	}

	return dbURL
}

// Deprecated: Use the getExecutionMode function at the top level instead

// detectDatabaseURLSource attempts to determine the source of the database URL
func detectDatabaseURLSource(dbURL string) string {
	// Check environment variables in order of preference
	envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}

	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value == dbURL {
			return fmt.Sprintf("environment variable %s", envVar)
		}
	}

	// If we couldn't match to an environment variable
	return "configuration"
}

// extractHostFromURL extracts the hostname from a database URL for logging
func extractHostFromURL(dbURL string) string {
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return "unknown"
	}

	return parsedURL.Hostname()
}

// directoryExists checks if a directory exists at the given path
func directoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// MigrationFilesData holds structured information about migration files
type MigrationFilesData struct {
	Files         []string
	SQLCount      int
	NewestFile    string
	OldestFile    string
	LatestVersion string // The version number of the latest migration
}

// getMigrationsPath returns the path to the migrations directory
func getMigrationsPath() (string, error) {
	// First try to use the standardized migrations directory function
	migrationsPath, err := testdb.FindMigrationsDir()
	if err == nil && directoryExists(migrationsPath) {
		// Found migrations directory using the standardized function
		return migrationsPath, nil
	}

	// Log the failure for diagnostic purposes
	slog.Debug("Standardized FindMigrationsDir failed, falling back to traditional method",
		"error", err)

	// Start with the constant path relative to the working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Try the standard path first
	stdPath := filepath.Join(cwd, migrationsDir)
	if directoryExists(stdPath) {
		return stdPath, nil
	}

	// If that fails, try to find the project root by traversing up
	dir := cwd
	for i := 0; i < 10; i++ { // Try up to 10 levels up
		// Check for go.mod file to identify project root
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Found project root, now look for migrations
			migPath := filepath.Join(dir, migrationsDir)
			if directoryExists(migPath) {
				return migPath, nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached filesystem root
		}
		dir = parent
	}

	// Check GitHub workspace environment variable (CI environment)
	if ghWorkspace := os.Getenv("GITHUB_WORKSPACE"); ghWorkspace != "" {
		migPath := filepath.Join(ghWorkspace, migrationsDir)
		if directoryExists(migPath) {
			return migPath, nil
		}
	}

	// If all else fails, return an error
	return "", fmt.Errorf("could not locate migrations directory")
}

// enumerateMigrationFiles lists and categorizes migration files in a directory
func enumerateMigrationFiles(dirPath string) (MigrationFilesData, error) {
	result := MigrationFilesData{}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return result, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		result.Files = append(result.Files, name)

		// Count SQL files and extract version information
		if filepath.Ext(name) == ".sql" {
			result.SQLCount++

			// Track oldest and newest based on filename (migrations are typically named with timestamps)
			if result.OldestFile == "" || name < result.OldestFile {
				result.OldestFile = name
			}
			if result.NewestFile == "" || name > result.NewestFile {
				result.NewestFile = name
			}

			// Extract the version number from the filename
			// Migration files typically have format: YYYYMMDDHHMMSS_description.sql
			parts := strings.SplitN(name, "_", 2)
			if len(parts) > 0 {
				version := parts[0]
				// Check if it looks like a valid version (numeric)
				if _, err := strconv.ParseInt(version, 10, 64); err == nil {
					// Keep track of the latest version
					if result.LatestVersion == "" || version > result.LatestVersion {
						result.LatestVersion = version
					}
				}
			}
		}
	}

	// Sort files by name for consistent output
	sort.Strings(result.Files)

	return result, nil
}
