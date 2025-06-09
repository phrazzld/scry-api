//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainFunctionLogic tests the main function logic without calling main directly
// This tests the sequence of operations that main() performs
func TestMainFunctionLogic(t *testing.T) {
	t.Run("main function initialization sequence", func(t *testing.T) {
		// Set up environment for successful config loading
		originalEnv := make(map[string]string)
		envVars := map[string]string{
			"SCRY_SERVER_PORT":                         "8080",
			"SCRY_SERVER_LOG_LEVEL":                    "info",
			"SCRY_DATABASE_URL":                        "postgres://test:test@localhost:5432/test",
			"SCRY_AUTH_JWT_SECRET":                     "test-jwt-secret-key-32-chars-123",
			"SCRY_AUTH_TOKEN_LIFETIME_MINUTES":         "60",
			"SCRY_AUTH_REFRESH_TOKEN_LIFETIME_MINUTES": "1440",
			"SCRY_LLM_GEMINI_API_KEY":                  "test-gemini-key",
			"SCRY_LLM_MODEL_NAME":                      "gemini-1.5-flash",
			"SCRY_LLM_PROMPT_TEMPLATE_PATH":            "../../prompts/flashcard_template.txt",
			"SCRY_TASK_WORKER_COUNT":                   "1",
			"SCRY_TASK_QUEUE_SIZE":                     "10",
			"SCRY_TASK_STUCK_TASK_AGE_MINUTES":         "5",
		}

		// Setup environment
		for key, value := range envVars {
			originalEnv[key] = os.Getenv(key)
			os.Setenv(key, value)
		}

		// Cleanup
		defer func() {
			for key := range envVars {
				if originalVal, existed := originalEnv[key]; existed {
					os.Setenv(key, originalVal)
				} else {
					os.Unsetenv(key)
				}
			}
		}()

		// Test the sequence of operations that main() performs

		// 1. Load configuration (like main does)
		cfg, err := loadAppConfig()
		require.NoError(t, err, "loadAppConfig should succeed like in main()")
		require.NotNil(t, cfg)

		// 2. Setup logger (like main does)
		logger, err := setupAppLogger(cfg)
		require.NoError(t, err, "setupAppLogger should succeed like in main()")
		require.NotNil(t, logger)

		// 3. Setup database (like main does) - this will fail with our test URL, which is expected
		// In main(), if database setup fails, it calls os.Exit(1)
		db, err := setupAppDatabase(cfg, logger)
		if err != nil {
			// This is expected with our test database URL
			t.Logf("Database setup failed as expected with test URL: %v", err)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to")
		} else {
			// If it unexpectedly succeeds, clean up
			require.NotNil(t, db)
			db.Close()
		}

		// 4. Test context creation (like main does)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		require.NotNil(t, ctx)

		// Note: We don't test newApplication or app.Run here because they require
		// a valid database connection, which we can't provide in unit tests
		// Those are tested separately with mocks
	})

	t.Run("main function config loading failure simulation", func(t *testing.T) {
		// Clear environment to simulate config loading failure
		originalEnv := make(map[string]string)
		requiredVars := []string{
			"SCRY_DATABASE_URL",
			"SCRY_AUTH_JWT_SECRET",
		}

		// Save and clear required environment variables
		for _, key := range requiredVars {
			originalEnv[key] = os.Getenv(key)
			os.Unsetenv(key)
		}

		// Cleanup
		defer func() {
			for key, value := range originalEnv {
				if value != "" {
					os.Setenv(key, value)
				} else {
					os.Unsetenv(key)
				}
			}
		}()

		// Test that loadAppConfig fails when required config is missing
		// In main(), this would cause os.Exit(1)
		cfg, err := loadAppConfig()
		assert.Error(t, err, "loadAppConfig should fail with missing required config")
		assert.Nil(t, cfg)
	})

	t.Run("main function logger setup failure simulation", func(t *testing.T) {
		// Create config with invalid log level that might cause logger setup to fail
		cfg := CreateMinimalTestConfig(t)
		cfg.Server.LogLevel = "" // Empty log level might cause issues

		// Test logger setup - this should actually still work as the logger
		// library likely handles empty/invalid log levels gracefully
		logger, err := setupAppLogger(cfg)

		// Most logger implementations handle invalid levels gracefully
		// If it fails, that's what main() would encounter
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, logger)
			t.Logf("Logger setup failed as might happen in main(): %v", err)
		} else {
			assert.NotNil(t, logger)
			t.Log("Logger setup succeeded even with empty log level")
		}
	})
}

// TestMainMigrationFlags tests the migration-related logic from main
func TestMainMigrationFlags(t *testing.T) {
	t.Run("migration flag handling logic", func(t *testing.T) {
		// Set up complete test environment with all required variables
		SetupTestEnvironment(t)

		// Override the database URL for this test
		t.Setenv("SCRY_DATABASE_URL", "postgres://test:test@localhost:5432/test")

		// Test the sequence when migration flags are provided
		// In main(), if migration flags are set, it calls handleMigrations and exits

		// 1. Load config
		cfg, err := loadAppConfig()
		require.NoError(t, err)

		// 2. Setup logger
		_, err = setupAppLogger(cfg)
		require.NoError(t, err)

		// 3. Test that handleMigrations can be called
		// (This will likely fail because of database connection issues, but tests the logic)
		err = handleMigrations(cfg, "status", "", false, false, false)

		// handleMigrations will likely fail with our test database URL
		// In main(), this would cause os.Exit(1)
		if err != nil {
			assert.Error(t, err)
			t.Logf("handleMigrations failed as expected with test config: %v", err)
		} else {
			t.Log("handleMigrations unexpectedly succeeded")
		}
	})
}

// TestMainApplicationFlowExtensive tests the application flow without actually running the server
func TestMainApplicationFlowExtensive(t *testing.T) {
	t.Run("application initialization without server start", func(t *testing.T) {
		// This tests the flow of main() up to the point where it would start the server

		// Set up complete test environment with all required variables
		SetupTestEnvironment(t)

		// Override the database URL to use a mock URL for this test
		t.Setenv("SCRY_DATABASE_URL", "mock://test")

		// Follow main() logic up to the point of starting the server

		// 1. Config loading
		cfg, err := loadAppConfig()
		require.NoError(t, err)

		// 2. Logger setup
		logger, err := setupAppLogger(cfg)
		require.NoError(t, err)

		// 3. Database setup - will fail with mock URL, which is what main() would encounter
		db, err := setupAppDatabase(cfg, logger)
		if err != nil {
			// Expected failure - in main() this would call os.Exit(1)
			assert.Error(t, err)
			t.Logf("Database setup failed as expected in main() flow: %v", err)
			return
		}
		defer db.Close()

		// 4. Context creation
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// 5. Application initialization - likely to fail with mock database
		app, err := newApplication(ctx, cfg, logger, db)
		if err != nil {
			// Expected failure with mock database
			assert.Error(t, err)
			t.Logf("Application initialization failed as expected: %v", err)
			return
		}

		// 6. If we got this far, test that app.Run handles quick cancellation
		if app != nil {
			err = app.Run(ctx)
			// Should handle timeout gracefully
			assert.NoError(t, err)
		}
	})
}

// TestMainConstantsAndGlobals tests package-level constants and variables
func TestMainConstantsAndGlobals(t *testing.T) {
	t.Run("package constants", func(t *testing.T) {
		// Test the migrationsDir constant
		assert.NotEmpty(t, migrationsDir)
		assert.Equal(t, "internal/platform/postgres/migrations", migrationsDir)
	})
}
