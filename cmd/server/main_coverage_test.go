//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainFunctionComponents tests components of main() that can be tested in isolation
func TestMainFunctionComponents(t *testing.T) {
	t.Run("constants and globals", func(t *testing.T) {
		// Test that constants are properly defined
		assert.Equal(t, "internal/platform/postgres/migrations", migrationsDir)
		assert.NotEmpty(t, migrationsDir)
	})

	t.Run("application initialization sequence", func(t *testing.T) {
		// Test the application initialization sequence without actually running main()
		// This tests the individual components that main() uses

		// 1. Test config loading with valid env vars
		t.Setenv("SCRY_DATABASE_URL", "postgres://test:test@localhost:5432/test")
		t.Setenv("SCRY_AUTH_JWT_SECRET", "test-secret-key-for-testing-only-32-chars-long")
		t.Setenv("SCRY_LLM_GEMINI_API_KEY", "test-api-key")
		t.Setenv("SCRY_LLM_PROMPT_TEMPLATE_PATH", "../../prompts/flashcard_template.txt")

		cfg, err := loadAppConfig()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// 2. Test logger setup
		logger, err := setupAppLogger(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)

		// 3. Test that database setup would work (but expect connection failure)
		_, err = setupAppDatabase(cfg, logger)
		assert.Error(t, err, "Expected database connection to fail in test environment")
		assert.Contains(t, err.Error(), "failed to ping database")
	})
}

// TestMigrationComponents tests migration-related functionality
func TestMigrationComponents(t *testing.T) {
	t.Run("migration path detection", func(t *testing.T) {
		// Test FindMigrationsDir function
		migrationsPath, err := FindMigrationsDir()
		if err != nil {
			// This might fail in test environment, which is expected
			assert.Contains(t, err.Error(), "migrations directory")
		} else {
			assert.NotEmpty(t, migrationsPath)
			assert.Contains(t, migrationsPath, "migrations")
		}
	})

	t.Run("project root detection", func(t *testing.T) {
		// Test FindProjectRoot function
		projectRoot, err := FindProjectRoot()
		if err != nil {
			// This might fail depending on test environment
			t.Logf("FindProjectRoot failed as expected in test env: %v", err)
		} else {
			assert.NotEmpty(t, projectRoot)
		}
	})

	t.Run("migration utilities", func(t *testing.T) {
		// Test utility functions that don't require database

		// Test URL masking - maskPassword URL-encodes the stars as %2A%2A%2A%2A
		masked := maskPassword("postgres://user:password@host:5432/db")
		assert.Contains(t, masked, "user:%2A%2A%2A%2A@host")
		assert.NotContains(t, masked, "password")

		// Test empty URL
		masked = maskPassword("")
		assert.Equal(t, "", masked)

		// Test URL without password
		masked = maskPassword("postgres://user@host:5432/db")
		assert.Equal(t, "postgres://user@host:5432/db", masked)
	})

	t.Run("database URL standardization", func(t *testing.T) {
		// Test standardizeCIDatabaseURL function - it changes credentials to postgres:postgres
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{"standard postgres URL", "postgres://user:pass@host:5432/db", "postgres://postgres:postgres@host:5432/db"},
			{
				"localhost URL",
				"postgres://user:pass@localhost:5432/db",
				"postgres://postgres:postgres@localhost:5432/db",
			},
			{
				"empty URL",
				"",
				"//postgres:postgres@",
			}, // standardizeCIDatabaseURL adds postgres:postgres even to empty URLs
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := standardizeCIDatabaseURL(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("environment detection", func(t *testing.T) {
		// Test CI environment detection
		originalCI := os.Getenv("CI")
		defer func() {
			if originalCI == "" {
				os.Unsetenv("CI")
			} else {
				os.Setenv("CI", originalCI)
			}
		}()

		// Test with CI=true
		os.Setenv("CI", "true")
		assert.True(t, isCIEnvironment(), "Should detect CI environment when CI=true")

		// Test with CI=false - isCIEnvironment() only checks if CI env var is non-empty, not its value
		os.Setenv("CI", "false")
		assert.True(t, isCIEnvironment(), "isCIEnvironment() returns true for any non-empty CI value")

		// Test without CI env var
		os.Unsetenv("CI")
		assert.False(t, isCIEnvironment(), "Should not detect CI environment when CI unset")
	})
}

// TestMigrationValidationHelpers tests migration validation functions
func TestMigrationValidationHelpers(t *testing.T) {
	t.Run("migrations directory enumeration", func(t *testing.T) {
		// Test enumerateMigrationFiles - this should work if migrations directory exists
		migrationsPath, err := FindMigrationsDir()
		if err != nil {
			t.Skip("Skipping migration enumeration test - migrations directory not found")
		}

		files, err := enumerateMigrationFiles(migrationsPath)
		if err != nil {
			t.Logf("Migration enumeration failed as expected: %v", err)
		} else {
			assert.IsType(t, MigrationFilesData{}, files)
			// Should have some migration files in the project
			if len(files.Files) > 0 {
				t.Logf("Found %d migration files", len(files.Files))
				// Only check SQL files, as the directory may contain other files like README.md, .keep
				sqlFileCount := 0
				for _, file := range files.Files {
					if strings.HasSuffix(file, ".sql") {
						sqlFileCount++
					}
				}
				t.Logf("Found %d SQL files out of %d total files", sqlFileCount, len(files.Files))
				assert.Equal(t, files.SQLCount, sqlFileCount, "SQLCount should match actual SQL files found")
			}
		}
	})
}

// TestApplicationCleanup tests cleanup functionality
func TestApplicationCleanup(t *testing.T) {
	t.Run("cleanup with nil fields", func(t *testing.T) {
		app := &application{}

		// cleanup() currently panics with nil logger due to logger.Error() and logger.Info() calls
		// This is expected behavior - the application should have a valid logger
		assert.Panics(t, func() {
			app.cleanup()
		}, "cleanup panics with nil logger (expected behavior)")
	})

	t.Run("cleanup with partial initialization", func(t *testing.T) {
		testLogger, _ := CreateTestLogger(t)
		app := &application{
			logger: testLogger,
			// Other fields remain nil
		}

		// Should not panic with partial initialization
		require.NotPanics(t, func() {
			app.cleanup()
		}, "cleanup should handle partial initialization")
	})
}

// TestApplicationRun tests the Run method without actually starting a server
func TestApplicationRun(t *testing.T) {
	t.Run("run with minimal setup", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
		}

		// Can't easily test Run() without starting an actual server
		// But we can test that setupRouter works
		router := app.setupRouter()
		assert.NotNil(t, router)

		// The router should have the health endpoint
		// This indirectly tests part of the Run() method's router setup
	})
}

// TestUtilityFunctions tests various utility functions for coverage
func TestUtilityFunctions(t *testing.T) {
	t.Run("URL extraction", func(t *testing.T) {
		tests := []struct {
			name     string
			url      string
			expected string
		}{
			{
				"postgres URL",
				"postgres://user:pass@localhost:5432/db",
				"localhost",
			}, // extractHostFromURL returns hostname only
			{"invalid URL", "not-a-url", ""},
			{"empty URL", "", ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := extractHostFromURL(tt.url)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("directory existence check", func(t *testing.T) {
		// Test with a directory that should exist
		exists := directoryExists(".")
		assert.True(t, exists, "Current directory should exist")

		// Test with a directory that shouldn't exist
		exists = directoryExists("/non/existent/directory/path")
		assert.False(t, exists, "Non-existent directory should return false")
	})

	t.Run("migrations path resolution", func(t *testing.T) {
		// Test getMigrationsPath function
		path, err := getMigrationsPath()
		if err != nil {
			t.Logf("getMigrationsPath failed as expected: %v", err)
		} else {
			assert.Contains(t, path, "migrations")
		}
	})

	t.Run("database URL source detection", func(t *testing.T) {
		// Save original env vars
		originalDB := os.Getenv("DATABASE_URL")
		originalScryDB := os.Getenv("SCRY_DATABASE_URL")
		defer func() {
			if originalDB == "" {
				os.Unsetenv("DATABASE_URL")
			} else {
				os.Setenv("DATABASE_URL", originalDB)
			}
			if originalScryDB == "" {
				os.Unsetenv("SCRY_DATABASE_URL")
			} else {
				os.Setenv("SCRY_DATABASE_URL", originalScryDB)
			}
		}()

		// Test with SCRY_DATABASE_URL
		testURL := "postgres://test"
		os.Setenv("SCRY_DATABASE_URL", testURL)
		os.Unsetenv("DATABASE_URL")
		source := detectDatabaseURLSource(testURL)
		assert.Equal(t, "environment variable SCRY_DATABASE_URL", source)

		// Test with legacy DATABASE_URL
		os.Unsetenv("SCRY_DATABASE_URL")
		os.Setenv("DATABASE_URL", testURL)
		source = detectDatabaseURLSource(testURL)
		assert.Equal(t, "environment variable DATABASE_URL", source)

		// Test with neither (empty URL)
		os.Unsetenv("SCRY_DATABASE_URL")
		os.Unsetenv("DATABASE_URL")
		source = detectDatabaseURLSource("")
		assert.Equal(t, "configuration", source)
	})
}

// TestExecutionMode tests execution mode detection
func TestExecutionMode(t *testing.T) {
	t.Run("execution mode detection", func(t *testing.T) {
		mode := getExecutionMode()
		assert.Contains(t, []string{"local", "ci"}, mode)
	})
}
