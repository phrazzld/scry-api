package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// slogGooseLogger adapts the goose logger interface to use slog
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

// logDatabaseInfo logs detailed database connection and state information
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
