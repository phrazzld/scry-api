package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/pressly/goose/v3"
)

// executeMigration executes database migrations using goose
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
	if testDBURL = GetTestDatabaseURL(); testDBURL != "" {
		migrationLogger.Info("Using standardized test database URL",
			"source", "GetTestDatabaseURL")
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
	migrationsPath, migrationsErr := FindMigrationsDir()
	if migrationsErr == nil && directoryExists(migrationsPath) {
		migrationsDirPath = migrationsPath
		migrationLogger.Info("Using standardized migrations directory path",
			"path", migrationsDirPath,
			"source", "FindMigrationsDir")
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
	migrationTableName := MigrationTableName
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

// runMigrations is a wrapper around executeMigration for backward compatibility
func runMigrations(cfg *config.Config, command string, verbose bool, args ...string) error {
	return executeMigration(cfg, command, verbose, args...)
}
