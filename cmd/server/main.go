// Package main implements the entry point for the Scry API server
// which handles users' spaced repetition flashcards and provides
// LLM integration for card generation.
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
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"
)

// Constant for the migrations directory path
// Used in the migration command implementation
// This is a relative path from the project root
const migrationsDir = "internal/platform/postgres/migrations"

// main is the entry point for the scry-api server.
// It will be responsible for initializing configuration, setting up logging,
// establishing database connections, injecting dependencies, and starting the
// HTTP server.
func main() {
	// Define migration-related command-line flags
	// These will be used in a future task to implement the migration functionality
	migrateCmd := flag.String("migrate", "", "Run database migrations (up|down|create|status|version)")
	migrationName := flag.String("name", "", "Name for new migration file (used with -migrate=create)")

	// Parse command-line flags
	flag.Parse()

	// If a migration command was specified, execute it and exit
	if *migrateCmd != "" {
		slog.Info("Migration requested",
			"command", *migrateCmd,
			"name", *migrationName)

		// Load configuration for migration
		cfg, err := config.Load()
		if err != nil {
			slog.Error("Failed to load configuration for migration",
				"error", err)
			os.Exit(1)
		}

		// Set up logging
		_, err = logger.Setup(cfg.Server)
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

	// Call the core initialization logic
	cfg, err := initializeApp()
	if err != nil {
		// Still using the default logger here if initializeApp failed
		// (which may include logger setup failure)
		slog.Error("Failed to initialize application",
			"error", err)
		os.Exit(1)
	}

	// At this point, the JSON structured logger has been configured by initializeApp()
	// All log messages from here on will use the structured JSON format
	slog.Info("Scry API Server initialized successfully",
		"port", cfg.Server.Port)

	// Start the server
	startServer(cfg)
}

// startServer configures and starts the HTTP server.
func startServer(cfg *config.Config) {
	// Open a database connection
	db, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("Error closing database connection", "error", err)
		}
	}()

	// Configure connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		slog.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("Database connection established")

	// Initialize dependencies
	userStore := postgres.NewPostgresUserStore(db, bcrypt.DefaultCost)
	jwtService, err := auth.NewJWTService(cfg.Auth)
	if err != nil {
		slog.Error("Failed to initialize JWT service", "error", err)
		os.Exit(1)
	}

	// Create a router
	r := chi.NewRouter()

	// Apply standard middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Create the password verifier
	passwordVerifier := auth.NewBcryptVerifier()

	// Create API handlers
	authHandler := api.NewAuthHandler(userStore, jwtService, passwordVerifier)
	authMiddleware := authmiddleware.NewAuthMiddleware(jwtService)

	// Register routes
	r.Route("/api", func(r chi.Router) {
		// Authentication endpoints (public)
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			// Add protected routes here
		})
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			slog.Error("Error writing health check response", "error", err)
		}
	})

	// Create and run server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Starting server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}

	slog.Info("Server shutdown completed")
}

// initializeApp loads configuration and sets up application components.
// Returns the loaded config and any initialization error.
func initializeApp() (*config.Config, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set up structured logging using the configured log level
	// After this point, all slog calls will use the JSON structured logger
	_, err = logger.Setup(cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to set up logger: %w", err)
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

	// Initialize services

	// Establish database connection
	db, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database connection: %w", err)
	}

	// Configure connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	slog.Info("Database connection established")

	// Initialize UserStore (will be used in startServer)
	slog.Info("User store initialized")

	// Initialize JWT authentication service (will be used in startServer)
	_, err = auth.NewJWTService(cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize JWT authentication service: %w", err)
	}
	slog.Info("JWT authentication service initialized",
		"token_lifetime_minutes", cfg.Auth.TokenLifetimeMinutes)

	// These services will be initialized in future tasks:
	// - Initializing LLM client with LLM.GeminiAPIKey

	return cfg, nil
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
				"network error connecting to database: %w (check network connectivity and DNS resolution)",
				err,
			)
		}

		// PostgreSQL-specific errors
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "28P01": // ERRCODE_INVALID_PASSWORD
				return fmt.Errorf(
					"database authentication failed: %w (check username and password in connection string)",
					err,
				)
			case "3D000": // ERRCODE_INVALID_CATALOG_NAME
				return fmt.Errorf(
					"database does not exist: %w (check database name in connection string or create the database)",
					err,
				)
			case "42501": // ERRCODE_INSUFFICIENT_PRIVILEGE
				return fmt.Errorf(
					"insufficient privileges to connect to database: %w (check user permissions)",
					err,
				)
			default:
				return fmt.Errorf(
					"PostgreSQL error: %s (code: %s): %w (check PostgreSQL logs for details)",
					pgErr.Message, pgErr.Code, err,
				)
			}
		}

		// Fallback to string-based error detection for errors not caught above
		switch {
		case strings.Contains(err.Error(), "connect: connection refused"):
			return fmt.Errorf(
				"database server refused connection: %w (check if database server is running and accessible)",
				err,
			)
		case strings.Contains(err.Error(), "no such host"):
			return fmt.Errorf(
				"database host not found: %w (check hostname in connection URL and DNS resolution)",
				err,
			)
		case strings.Contains(err.Error(), "authentication failed"):
			return fmt.Errorf(
				"database authentication failed: %w (check username and password in connection string)",
				err,
			)
		case strings.Contains(err.Error(), "database"):
			return fmt.Errorf(
				"database not found: %w (check database name in connection URL)",
				err,
			)
		default:
			// Log the detailed error type for debugging
			slog.Debug("Unhandled database connection error",
				"error_type", fmt.Sprintf("%T", err),
				"error", err.Error(),
			)
			return fmt.Errorf("failed to ping database: %w (type: %T)", err, err)
		}
	}

	// Database connection is established and verified
	slog.Info("Database connection established successfully")

	// Set migrations directory
	slog.Debug("Setting migrations directory", "dir", migrationsDir)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Execute the migration command based on input
	switch command {
	case "up":
		// Run all available migrations
		slog.Info("Running migrations up", "dir", migrationsDir)
		if err := goose.Up(db, migrationsDir); err != nil {
			return fmt.Errorf("migration up failed: %w", err)
		}

	case "down":
		// Rollback the last migration
		slog.Info("Running migrations down (rollback last migration)", "dir", migrationsDir)
		if err := goose.Down(db, migrationsDir); err != nil {
			return fmt.Errorf("migration down failed: %w", err)
		}

	case "status":
		// Show migration status
		slog.Info("Checking migration status", "dir", migrationsDir)
		if err := goose.Status(db, migrationsDir); err != nil {
			return fmt.Errorf("migration status check failed: %w", err)
		}

	case "create":
		// Validate migration name argument
		if len(args) == 0 || args[0] == "" {
			return fmt.Errorf("migration name is required for create command (use -name flag)")
		}

		name := args[0]
		// Create a new migration file
		slog.Info("Creating new migration", "name", name, "dir", migrationsDir)
		if err := goose.Create(db, migrationsDir, name, "sql"); err != nil {
			return fmt.Errorf("migration creation failed: %w", err)
		}

	case "version":
		// Show current migration version
		slog.Info("Checking current migration version", "dir", migrationsDir)
		if err := goose.Version(db, migrationsDir); err != nil {
			return fmt.Errorf("getting migration version failed: %w", err)
		}

	default:
		return fmt.Errorf("unknown migration command: %s (valid commands: up, down, status, create, version)", command)
	}

	return nil
}
