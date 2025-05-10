//go:build migration_helpers

package testdb

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
)

// The MigrationTableName constant is now defined in constants.go

// ApplyMigrations runs the migrations in the specified directory.
// This is a helper function that doesn't require testing.T.
func ApplyMigrations(db *sql.DB, migrationsDir string) error {
	// Set dialect for goose
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Configure goose with silent logger and set migration table name
	goose.SetLogger(&silentLogger{})
	goose.SetTableName(MigrationTableName)

	// Verify migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
	}

	// Run migrations
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// silentLogger is a minimal implementation of goose.Logger that doesn't output anything
type silentLogger struct{}

func (l *silentLogger) Printf(format string, v ...interface{}) {
	// Silence all regular log messages
}

func (l *silentLogger) Fatalf(format string, v ...interface{}) {
	// Don't exit, just return an error through the normal channels
	// Instead, the error is returned from goose.Up
}
