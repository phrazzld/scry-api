package main

import (
	"fmt"
	"log/slog"

	"github.com/phrazzld/scry-api/internal/config"
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
			"verbose", verbose)

		// For the create command, we need to pass the migration name
		var args []string
		if migrateCmd == "create" && migrationName != "" {
			args = append(args, migrationName)
		}

		// Execute the migration command
		return runMigrations(cfg, migrateCmd, verbose, args...)
	}

	// No migration operation requested
	return fmt.Errorf("no migration operation specified")
}
