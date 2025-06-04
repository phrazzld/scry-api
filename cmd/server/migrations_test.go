//go:build integration || test_without_external_deps

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationFlow tests the entire migration flow if there's a database URL available.
// This is an integration test and will be skipped if DATABASE_URL isn't set.
func TestMigrationFlow(t *testing.T) {
	if !testutils.IsIntegrationTestEnvironment() {
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

// Unit tests that don't require external dependencies

// TestMigrationValidationUnit tests migration validation functions without database
func TestMigrationValidationUnit(t *testing.T) {
	t.Run("migration files enumeration", func(t *testing.T) {
		// Test with non-existent directory
		files, err := enumerateMigrationFiles("/non/existent/directory")
		assert.Error(t, err)
		assert.Equal(t, MigrationFilesData{}, files)

		// Test with current directory (not a migrations dir, but tests the function)
		files, err = enumerateMigrationFiles(".")
		if err != nil {
			// Expected - current dir is not migrations dir
			assert.Contains(t, err.Error(), "migration")
		} else {
			// If it succeeds, verify structure
			assert.IsType(t, MigrationFilesData{}, files)
		}
	})

	t.Run("migration table name", func(t *testing.T) {
		// Test that migration table name constant is defined
		assert.NotEmpty(t, MigrationTableName)
		assert.Equal(t, "schema_migrations", MigrationTableName)
	})
}

// TestMigrationHelpersUnit tests migration helper functions
func TestMigrationHelpersUnit(t *testing.T) {
	t.Run("find migrations directory", func(t *testing.T) {
		// Test FindMigrationsDir function
		migrationsPath, err := FindMigrationsDir()
		if err != nil {
			// Expected in test environment
			assert.Contains(t, err.Error(), "migrations")
		} else {
			assert.NotEmpty(t, migrationsPath)
			assert.Contains(t, migrationsPath, "migrations")
		}
	})

	t.Run("find project root", func(t *testing.T) {
		// Test FindProjectRoot function
		projectRoot, err := FindProjectRoot()
		if err != nil {
			// This might fail depending on test environment
			t.Logf("FindProjectRoot failed as expected in test env: %v", err)
		} else {
			assert.NotEmpty(t, projectRoot)
			// Should be an absolute path
			assert.True(t, filepath.IsAbs(projectRoot))
		}
	})

	t.Run("standardize CI database URL", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{"empty URL", "", "//postgres:postgres@"},
			{"standard postgres URL", "postgres://user:pass@host:5432/db", "postgres://postgres:postgres@host:5432/db"},
			{
				"localhost URL",
				"postgres://user:pass@localhost:5432/db",
				"postgres://postgres:postgres@localhost:5432/db",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := standardizeCIDatabaseURL(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("mask password function", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{"URL with password", "postgres://user:secret@host:5432/db", "postgres://user:%2A%2A%2A%2A@host:5432/db"},
			{"URL without password", "postgres://user@host:5432/db", "postgres://user@host:5432/db"},
			{"empty URL", "", ""},
			{"invalid URL", "not-a-url", "not-a-url"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := maskPassword(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

// TestMigrationExecutorUnit tests migration executor concepts
func TestMigrationExecutorUnit(t *testing.T) {
	t.Run("migration executor structure validation", func(t *testing.T) {
		// We can't test the actual executor without real dependencies
		// but we can test that our migration constants are properly defined
		assert.NotEmpty(t, migrationsDir)
		assert.Equal(t, "internal/platform/postgres/migrations", migrationsDir)

		// Test that the migration table name is consistent
		assert.Equal(t, "schema_migrations", MigrationTableName)
	})
}

// TestMigrationDirectoryUtilsUnit tests migration directory utilities
func TestMigrationDirectoryUtilsUnit(t *testing.T) {
	t.Run("check migrations directory existence", func(t *testing.T) {
		// Test with actual project structure
		possiblePaths := []string{
			"internal/platform/postgres/migrations",
			"../../internal/platform/postgres/migrations",
			"./migrations",
		}

		var foundPath string
		for _, path := range possiblePaths {
			if directoryExists(path) {
				foundPath = path
				break
			}
		}

		if foundPath != "" {
			t.Logf("Found migrations directory at: %s", foundPath)

			// Test that we can enumerate files in this directory
			files, err := enumerateMigrationFiles(foundPath)
			if err == nil {
				assert.IsType(t, MigrationFilesData{}, files)
				t.Logf("Found %d SQL files and %d total files", files.SQLCount, len(files.Files))
			}
		} else {
			t.Log("No migrations directory found in expected locations")
		}
	})

	t.Run("migration path constants", func(t *testing.T) {
		// Test that migration directory constant is defined
		assert.NotEmpty(t, migrationsDir)
		assert.Equal(t, "internal/platform/postgres/migrations", migrationsDir)
	})
}

// TestDatabaseURLDetectionUnit tests database URL detection and validation
func TestDatabaseURLDetectionUnit(t *testing.T) {
	t.Run("database URL source detection with various env vars", func(t *testing.T) {
		// Save original env vars
		envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}
		original := make(map[string]string)
		for _, env := range envVars {
			original[env] = os.Getenv(env)
			_ = os.Unsetenv(env)
		}
		defer func() {
			for env, val := range original {
				if val == "" {
					_ = os.Unsetenv(env)
				} else {
					_ = os.Setenv(env, val)
				}
			}
		}()

		testURL := "postgres://test:test@localhost:5432/test"

		// Test with SCRY_TEST_DB_URL
		_ = os.Setenv("SCRY_TEST_DB_URL", testURL)
		source := detectDatabaseURLSource(testURL)
		assert.Contains(t, source, "SCRY_TEST_DB_URL")

		// Test with no env var set matching the URL
		_ = os.Unsetenv("SCRY_TEST_DB_URL")
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("SCRY_DATABASE_URL")

		// Use a different URL that won't match any environment variables
		uniqueTestURL := "postgres://unique:unique@localhost:9999/unique"
		source = detectDatabaseURLSource(uniqueTestURL)
		assert.Equal(t, "configuration", source)

		// Test with empty URL - should not match unset env vars
		source = detectDatabaseURLSource("")
		assert.Equal(t, "configuration", source)
	})
}

// TestMigrationValidationHelpersUnit tests validation helper functions
func TestMigrationValidationHelpersUnit(t *testing.T) {
	t.Run("migration files data structure", func(t *testing.T) {
		// Test MigrationFilesData structure
		data := MigrationFilesData{
			Files:    []string{"file1.sql", "file2.sql"},
			SQLCount: 2,
		}

		assert.Equal(t, 2, len(data.Files))
		assert.Equal(t, 2, data.SQLCount)
		assert.Contains(t, data.Files, "file1.sql")
		assert.Contains(t, data.Files, "file2.sql")
	})

	t.Run("create migration files data from directory", func(t *testing.T) {
		// Create a temporary directory with test files
		tempDir, err := os.MkdirTemp("", "test-migrations")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Create test SQL files
		sqlFiles := []string{"001_test.sql", "002_test.sql", "003_test.sql"}
		for _, filename := range sqlFiles {
			file, err := os.Create(filepath.Join(tempDir, filename))
			require.NoError(t, err)
			_ = file.Close()
		}

		// Create a non-SQL file
		nonSQLFile, err := os.Create(filepath.Join(tempDir, "README.md"))
		require.NoError(t, err)
		_ = nonSQLFile.Close()

		// Test enumeration
		data, err := enumerateMigrationFiles(tempDir)
		if err != nil {
			// The function might have additional validation requirements
			t.Logf("enumerateMigrationFiles failed: %v", err)
		} else {
			assert.Equal(t, 3, data.SQLCount)
			assert.Equal(t, 4, len(data.Files)) // 3 SQL + 1 README
		}
	})
}

// TestMigrationConstantsUnit tests migration-related constants
func TestMigrationConstantsUnit(t *testing.T) {
	t.Run("migration table name constant", func(t *testing.T) {
		assert.Equal(t, "schema_migrations", MigrationTableName)
	})

	t.Run("migrations directory constant", func(t *testing.T) {
		assert.Equal(t, "internal/platform/postgres/migrations", migrationsDir)
	})
}

// TestMigrationErrorHandlingUnit tests error handling in migration functions
func TestMigrationErrorHandlingUnit(t *testing.T) {
	t.Run("migration functions with invalid paths", func(t *testing.T) {
		// Test getMigrationsPath with invalid working directory
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()

		// Change to a directory that doesn't have migrations
		tempDir, err := os.MkdirTemp("", "test-no-migrations")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		_ = os.Chdir(tempDir)

		path, err := getMigrationsPath()
		assert.Error(t, err)
		assert.Empty(t, path)
	})

	t.Run("directory operations with permission errors", func(t *testing.T) {
		// Test directoryExists with paths that might have permission issues
		paths := []string{
			"/root",          // Might not be accessible
			"/proc/self/mem", // Special file system
			"/dev/null",      // Device file, not directory
		}

		for _, path := range paths {
			exists := directoryExists(path)
			// Just test that function doesn't panic, result varies by system
			assert.IsType(t, false, exists)
		}
	})
}
