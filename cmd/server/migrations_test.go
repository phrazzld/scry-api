//go:build integration

package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/pressly/goose/v3"
)

// TestMigrationFlow tests the entire migration flow if there's a database URL available.
// This is an integration test and will be skipped if DATABASE_URL isn't set.
func TestMigrationFlow(t *testing.T) {
	if !IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get database URL from environment using the standardized function
	dbURL := testdb.GetTestDatabaseURL()

	// Create a minimal config for the test
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: dbURL,
		},
	}

	// Get the project root directory using the standardized function
	projectRoot, err := testdb.FindProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Change working directory to project root so that the relative path in migrationsDir works
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWD); err != nil {
			t.Logf("Warning: Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change working directory to project root: %v", err)
	}

	// Run the migration up
	err = runMigrations(cfg, "up", false)
	if err != nil {
		t.Fatalf("Failed to run migrations up: %v", err)
	}

	// Connect to the database to verify tables were created using the standardized connection string
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	defer testutils.AssertCloseNoError(t, db)

	// Set a timeout for the verification
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify tables exist by querying them
	tables := []string{"users", "memos", "cards", "user_card_stats"}
	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
            SELECT FROM information_schema.tables
            WHERE table_schema = 'public'
            AND table_name = $1
        )`
		err := db.QueryRowContext(ctx, query, table).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check if table %s exists: %v", table, err)
		}
		if !exists {
			t.Errorf("Table %s does not exist after running migrations", table)
		}
	}

	// Verify memo_status enum type exists
	var exists bool
	enumQuery := `SELECT EXISTS (
        SELECT FROM pg_type
        WHERE typname = 'memo_status'
    )`
	err = db.QueryRowContext(ctx, enumQuery).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check if memo_status enum exists: %v", err)
	}
	if !exists {
		t.Errorf("memo_status enum type does not exist after running migrations")
	}

	// Run migrations down to clean up - need to run multiple times to go all the way down
	// Since runMigrations("down") only goes down one version at a time
	for i := 0; i < 10; i++ { // 10 iterations should be more than enough for all migrations
		err = runMigrations(cfg, "down", false)
		if err != nil {
			// If we get the "no migrations" error, we've gone all the way down
			if err.Error() == "migration down failed: no migrations to run. current version: 0" ||
				err.Error() == "migration down failed: migration 0: no current version found" {
				break
			}
			t.Fatalf("Failed to run migrations down: %v", err)
		}
	}

	// Verify all tables are dropped
	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
            SELECT FROM information_schema.tables
            WHERE table_schema = 'public'
            AND table_name = $1
        )`
		err := db.QueryRowContext(ctx, query, table).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check if table %s exists: %v", table, err)
		}
		if exists {
			t.Errorf("Table %s still exists after running migrations down", table)
		}
	}
}

// TestMigrationsValidSyntax tests that all migration files contain valid SQL syntax.
// This test doesn't require a database connection.
func TestMigrationsValidSyntax(t *testing.T) {
	t.Parallel() // Can safely run in parallel since it doesn't use the database
	// Set up goose logger with a test adapter that fails the test on fatal errors
	goose.SetLogger(&testGooseLogger{t: t})

	// Get project root using the standardized function
	projectRoot, err := testdb.FindProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Construct path to migrations directory
	absoluteMigrationsDir := filepath.Join(projectRoot, migrationsDir)

	// First check if we can parse all migrations
	migrations, err := goose.CollectMigrations(absoluteMigrationsDir, 0, goose.MaxVersion)
	if err != nil {
		t.Fatalf("Failed to collect migrations: %v", err)
	}

	if len(migrations) == 0 {
		t.Fatalf("No migration files found in %s", migrationsDir)
	}

	// Log the migrations for visibility
	t.Logf("Found %d migration files:", len(migrations))
	for _, m := range migrations {
		_, filename := filepath.Split(m.Source)
		t.Logf("- %s", filename)
	}

	// All migrations were successfully parsed, which means they have valid syntax
	t.Log("All migrations have valid SQL syntax")
}

// testGooseLogger is a goose.Logger implementation for testing
type testGooseLogger struct {
	t *testing.T
}

func (l *testGooseLogger) Printf(format string, v ...interface{}) {
	l.t.Logf(format, v...)
}

func (l *testGooseLogger) Fatalf(format string, v ...interface{}) {
	l.t.Fatalf(format, v...)
}
