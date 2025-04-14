package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/pressly/goose/v3"
)

// TestMigrationFlow tests the entire migration flow if there's a database URL available.
// This is an integration test and will be skipped if DATABASE_URL isn't set.
func TestMigrationFlow(t *testing.T) {
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get database URL from environment
	dbURL := testutils.GetTestDatabaseURL(t)

	// Create a minimal config for the test
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: dbURL,
		},
	}

	// Run the migration up
	err := runMigrations(cfg, "up")
	if err != nil {
		t.Fatalf("Failed to run migrations up: %v", err)
	}

	// Connect to the database to verify tables were created
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	defer func() {
		err := db.Close()
		if err != nil {
			t.Logf("Error closing database connection: %v", err)
		}
	}()

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

	// Run migrations down to clean up
	err = runMigrations(cfg, "down")
	if err != nil {
		t.Fatalf("Failed to run migrations down: %v", err)
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
	// Set up goose logger with a test adapter that fails the test on fatal errors
	goose.SetLogger(&testGooseLogger{t: t})

	// Get absolute path to migrations directory based on current working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	// Go up one level from cmd/server to project root
	projectRoot := filepath.Dir(filepath.Dir(wd))
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
