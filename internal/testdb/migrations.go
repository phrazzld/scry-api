//go:build (integration || test_without_external_deps) && !exported_core_functions

// Package testdb provides utilities specifically for database testing.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

// This file contains database migration utilities for test setup.

// testGooseLogger implements a minimal logger interface for goose
type testGooseLogger struct {
	t *testing.T
}

// Printf implements the required logging method for goose's SetLogger
func (l *testGooseLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.t.Log("Goose: " + strings.TrimSpace(msg))
}

// Fatalf implements the required logging method for goose's SetLogger
func (l *testGooseLogger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.t.Fatal("Goose fatal error: " + strings.TrimSpace(msg))
}

// ApplyMigrations runs migrations without using testing.T
// This exists for backward compatibility with code that was written
// before the testdb package was created.
// The function includes enhanced error handling and diagnostics.
func ApplyMigrations(db *sql.DB, migrationsDir string) error {
	// Verify database connection is active
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		// We can't use formatDBConnectionError here since we don't have the URL
		// Instead, create a descriptive error message
		return fmt.Errorf("database connection failed before migrations: %w", err)
	}

	// Verify migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
	}

	// Configure goose
	goose.SetTableName("schema_migrations")
	goose.SetBaseFS(os.DirFS(migrationsDir))

	// Run migrations with comprehensive error handling
	if err := goose.Up(db, "."); err != nil {
		// Create detailed error with migration information
		migrationFiles := ""
		if entries, err := os.ReadDir(migrationsDir); err == nil && len(entries) > 0 {
			names := make([]string, 0, len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					names = append(names, entry.Name())
				}
			}
			migrationFiles = fmt.Sprintf(" (available migrations: %v)", names)
		}

		return fmt.Errorf("failed to run migrations in %s%s: %w", migrationsDir, migrationFiles, err)
	}

	return nil
}
