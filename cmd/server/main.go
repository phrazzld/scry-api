package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
)

// migrationsDir is the relative path to migrations from the project root
// This is deprecated in favor of testdb.FindMigrationsDir() but kept for compatibility
// with existing code that expects this constant
const migrationsDir = "internal/platform/postgres/migrations"

// main is the entry point for the scry-api server.
// It handles initialization, migration commands, and starting the server.
func main() {
	// Define migration-related command-line flags
	migrateCmd := flag.String(
		"migrate",
		"",
		"Run database migrations (up|down|create|status|version)",
	)
	migrationName := flag.String("name", "", "Name for the new migration (only used with 'create')")
	verbose := flag.Bool("verbose", false, "Enable verbose logging for migrations")
	verifyOnly := flag.Bool("verify-migrations", false, "Only verify migrations without applying them")
	validateMigrations := flag.Bool(
		"validate-migrations",
		false,
		"Validate applied migrations (returns non-zero exit code on failure)",
	)
	flag.Parse()

	// IMPORTANT: Log messages here use Go's default slog handler (plain text)
	// rather than our custom JSON handler. This is intentional - we can't set up
	// the custom JSON logger until we've loaded configuration, but we still want
	// to log the application startup. This creates a consistent initialization
	// sequence where even initialization errors can be logged.
	slog.Info("Scry API Server starting...")

	// Load configuration
	cfg, err := loadAppConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set up structured logging using the configured log level
	// After this point, all slog calls will use the JSON structured logger
	logger, err := setupAppLogger(cfg)
	if err != nil {
		slog.Error("Failed to set up logger", "error", err)
		os.Exit(1)
	}

	// If a migration command was specified or validation is requested, handle and exit
	if *migrateCmd != "" || *verifyOnly || *validateMigrations {
		err = handleMigrations(cfg, *migrateCmd, *migrationName, *verbose, *verifyOnly, *validateMigrations)
		if err != nil {
			logger.Error("Migration failed",
				"command", *migrateCmd,
				"error", err)
			os.Exit(1)
		}

		logger.Info("Migration completed successfully",
			"command", *migrateCmd)
		os.Exit(0)
	}

	// Establish database connection
	db, err := setupAppDatabase(cfg, logger)
	if err != nil {
		logger.Error("Failed to setup database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Error closing database connection", "error", err)
		}
	}()

	// Create application context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize application with all dependencies
	app, err := newApplication(ctx, cfg, logger, db)
	if err != nil {
		logger.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	// Start the application server
	err = app.Run(ctx)
	if err != nil {
		logger.Error("Application failed", "error", err)
		os.Exit(1)
	}
}
