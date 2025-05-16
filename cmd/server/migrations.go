package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/pressly/goose/v3"
)

// handleMigrations handles the execution of database migrations.
// It's called from main() when migration-related flags are detected.
// Returns an error if migrations fail or validation is unsuccessful.
func handleMigrations(
	cfg *config.Config,
	migrateCmd string,
	migrationName string,
	verbose bool,
	verifyOnly bool,
	validateMigrations bool,
) error {
	// Handle different migration-related commands
	if validateMigrations {
		// Validate applied migrations (mostly used in CI)
		slog.Info("Validating applied migrations",
			"verbose", verbose,
			"mode", getExecutionMode())
		return validateAppliedMigrations(cfg, verbose)
	} else if verifyOnly {
		// Only verify migrations without applying them
		slog.Info("Verifying migrations only (not applying)",
			"command", migrateCmd,
			"verbose", verbose)
		return verifyMigrations(cfg, verbose)
	} else if migrateCmd != "" {
		// Normal migration execution
		slog.Info("Executing migrations",
			"command", migrateCmd,
			"verbose", verbose,
			"name", migrationName)
		return runMigrations(cfg, migrateCmd, verbose, migrationName)
	}

	return nil
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
	migrationTable := MigrationTableName
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
	migrationTable := MigrationTableName
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
	testDBURL := GetTestDatabaseURL()
	if testDBURL != "" {
		logger.Info("Using standardized test database URL",
			"source", "GetTestDatabaseURL")
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
	migrationsPath, err := FindMigrationsDir()
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
