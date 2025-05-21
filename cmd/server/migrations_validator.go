package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/phrazzld/scry-api/internal/config"
)

// MigrationFilesData holds information about migration files in a directory
type MigrationFilesData struct {
	Files         []string
	SQLCount      int
	NewestFile    string
	OldestFile    string
	LatestVersion string // The version number of the latest migration
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

// verifyAppliedMigrations verifies that all migrations have been properly applied
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
