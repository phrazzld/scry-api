//go:build (integration || test_without_external_deps) && !exported_core_functions

// Package testdb provides utilities specifically for database testing.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// This file contains the main test database setup and teardown utilities.

// SetupTestDatabaseSchema applies all database migrations to prepare the database for testing.
// It provides enhanced error messages with diagnostics information for common failures.
func SetupTestDatabaseSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	// Check database connection first
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		dbURL := GetTestDatabaseURL()
		errDetail := formatDBConnectionError(err, dbURL)
		t.Fatalf("Database connection failed before migrations: %v", errDetail)
	}

	// Log database connection info in CI
	if isCIEnvironment() {
		// Verify database version
		var version string
		versionErr := db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
		if versionErr != nil {
			t.Logf("CI debug: Failed to query database version: %v", versionErr)
		} else {
			t.Logf("CI debug: Connected to PostgreSQL: %s", version)
		}

		// Check database user
		var user string
		userErr := db.QueryRowContext(ctx, "SELECT current_user").Scan(&user)
		if userErr != nil {
			t.Logf("CI debug: Failed to query current database user: %v", userErr)
		} else {
			t.Logf("CI debug: Connected as database user: %s", user)
		}

		// Check if migrations table exists
		var migTableExists bool
		tableQuery := fmt.Sprintf(
			"SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = '%s')",
			MigrationTableName,
		)
		tableErr := db.QueryRowContext(ctx, tableQuery).Scan(&migTableExists)
		if tableErr != nil {
			t.Logf("CI debug: Failed to check for migrations table: %v", tableErr)
		} else {
			t.Logf("CI debug: Migrations table '%s' exists: %v", MigrationTableName, migTableExists)
		}
	}

	// Find project root to locate migration files
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Set up goose for migrations
	migrationsDir := filepath.Join(projectRoot, "internal", "platform", "postgres", "migrations")

	// Check that migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		errDetail := fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
		t.Fatalf("Migrations directory error: %v", errDetail)
	}

	// In CI, list available migration files for diagnostics
	if isCIEnvironment() {
		t.Logf("CI debug: Migrations directory: %s", migrationsDir)
		entries, readErr := os.ReadDir(migrationsDir)
		if readErr != nil {
			t.Logf("CI debug: Failed to read migrations directory: %v", readErr)
		} else {
			migFiles := make([]string, 0, len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					migFiles = append(migFiles, entry.Name())
				}
			}
			t.Logf("CI debug: Available migration files: %v", migFiles)
		}
	}

	// Configure goose with custom logger
	goose.SetLogger(&testGooseLogger{t: t})
	goose.SetTableName(MigrationTableName)
	goose.SetBaseFS(os.DirFS(migrationsDir))

	// Try to run migrations with enhanced error reporting
	if err := goose.Up(db, "."); err != nil {
		errDetail := formatMigrationError(err, migrationsDir)
		t.Fatalf("Migration failed: %v", errDetail)
	}

	// Verify migrations were applied successfully
	var migrationCount int
	countErr := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", MigrationTableName)).
		Scan(&migrationCount)
	if countErr != nil {
		t.Fatalf("Failed to verify migrations: %v", countErr)
	}

	t.Logf("Database migrations applied successfully: %d migrations in schema", migrationCount)

	// In CI, list applied migrations for verification
	if isCIEnvironment() {
		rows, queryErr := db.QueryContext(
			ctx,
			fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY version_id", MigrationTableName),
		)
		if queryErr != nil {
			t.Logf("CI debug: Failed to query migration history: %v", queryErr)
		} else {
			defer func() {
				if err := rows.Close(); err != nil {
					t.Logf("Warning: failed to close rows: %v", err)
				}
			}()

			t.Logf("CI debug: Applied migrations:")
			for rows.Next() {
				var versionID string
				var isApplied bool
				if err := rows.Scan(&versionID, &isApplied); err != nil {
					t.Logf("CI debug: Failed to scan migration row: %v", err)
					continue
				}
				t.Logf("CI debug:   Migration %s, applied: %v", versionID, isApplied)
			}
		}
	}
}

// GetTestDBWithT returns a database connection for testing, with t.Helper() support.
// It automatically skips the test if DATABASE_URL is not set, ensuring
// consistent behavior for integration tests.
func GetTestDBWithT(t *testing.T) *sql.DB {
	t.Helper()

	// Skip the test if the database URL is not available
	dbURL := GetTestDatabaseURL()
	if dbURL == "" {
		t.Skip("DATABASE_URL or SCRY_TEST_DB_URL not set - skipping integration test")
	}

	// Open database connection
	db, err := sql.Open("pgx", dbURL)
	require.NoError(t, err, "Failed to open database connection")

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify the connection works
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	err = db.PingContext(ctx)
	require.NoError(t, err, "Database ping failed")

	// Register cleanup to close the database connection
	t.Cleanup(func() {
		CleanupDB(t, db)
	})

	return db
}

// GetTestDB returns a database connection for testing without t.Helper() support.
// This is useful for non-test code that needs database access.
// Returns an error with detailed diagnostics if the database connection cannot be established.
func GetTestDB() (*sql.DB, error) {
	// Check if the database URL is available
	dbURL := GetTestDatabaseURL()
	if dbURL == "" {
		return nil, formatEnvVarError()
	}

	// Open database connection
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w\nDatabase URL format may be incorrect", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify the connection works
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		// Close the connection if ping fails
		closeErr := db.Close()

		// Create the base error with detailed diagnostics
		baseErr := formatDBConnectionError(err, dbURL)

		// Add any additional connection close errors if they occurred
		if closeErr != nil {
			return nil, fmt.Errorf("%v\nAdditional error when closing connection: %w", baseErr, closeErr)
		}

		return nil, baseErr
	}

	return db, nil
}

// CleanupDB properly closes a database connection, logging any errors.
func CleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if db == nil {
		return
	}

	if err := db.Close(); err != nil {
		t.Logf("Warning: failed to close database connection: %v", err)
	}
}
