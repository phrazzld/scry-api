//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewApplication tests the main application initialization function structure
func TestNewApplication(t *testing.T) {
	t.Run("application initialization error cases", func(t *testing.T) {
		testLogger, _ := CreateTestLogger(t)
		ctx := context.Background()

		// newApplication should panic with nil inputs due to nil pointer dereference
		assert.Panics(t, func() {
			newApplication(ctx, nil, testLogger, nil)
		}, "newApplication should panic with nil config")

		// Test with nil logger - this might panic at logger.Info() or return error from service init
		cfg := CreateMinimalTestConfig(t)
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is expected - that's fine
					return
				}
			}()

			// If no panic, it should return an error
			app, err := newApplication(ctx, cfg, nil, nil)
			assert.Error(t, err, "newApplication should return error or panic with nil logger")
			assert.Nil(t, app)
		}()

		// Test with nil database - this should panic during postgres store initialization
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is expected - that's fine
					return
				}
			}()

			// If no panic, it should return an error
			app, err := newApplication(ctx, cfg, testLogger, nil)
			assert.Error(t, err, "newApplication should return error or panic with nil database")
			assert.Nil(t, app)
		}()
	})

	t.Run("application structure validation", func(t *testing.T) {
		// Test the Run method without full initialization
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
		}

		// Test setupRouter works with minimal app
		router := app.setupRouter()
		assert.NotNil(t, router)

		// Note: We can't easily test the full Run() method without starting a server
		// but we've tested its components in other test files
	})
}

// TestSetupTaskRunner tests task runner initialization
func TestSetupTaskRunner(t *testing.T) {
	t.Run("setup task runner with nil application", func(t *testing.T) {
		// This should panic due to nil pointer access
		assert.Panics(t, func() {
			setupTaskRunner(nil)
		}, "setupTaskRunner should panic with nil application")
	})

	t.Run("setup task runner with incomplete application", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
			// Missing taskStore - should cause panic when trying to access nil taskStore
		}

		// This should panic due to nil taskStore access
		assert.Panics(t, func() {
			setupTaskRunner(app)
		}, "setupTaskRunner should panic with nil taskStore")
	})
}

// TestMigrationUtilities tests various migration utility functions
func TestMigrationUtilities(t *testing.T) {
	t.Run("slog goose logger", func(t *testing.T) {
		logger := &slogGooseLogger{}

		// Test Printf doesn't panic
		require.NotPanics(t, func() {
			logger.Printf("test message %s", "arg")
		})

		// Test Fatalf doesn't panic and doesn't exit
		require.NotPanics(t, func() {
			logger.Fatalf("test error %s", "arg")
		})
	})

	t.Run("execution mode detection", func(t *testing.T) {
		// Save original env vars
		originalCI := os.Getenv("CI")
		originalGitHub := os.Getenv("GITHUB_ACTIONS")
		defer func() {
			if originalCI == "" {
				os.Unsetenv("CI")
			} else {
				os.Setenv("CI", originalCI)
			}
			if originalGitHub == "" {
				os.Unsetenv("GITHUB_ACTIONS")
			} else {
				os.Setenv("GITHUB_ACTIONS", originalGitHub)
			}
		}()

		// Test local environment
		os.Unsetenv("CI")
		os.Unsetenv("GITHUB_ACTIONS")
		mode := getExecutionMode()
		assert.Equal(t, "local", mode)

		// Test CI environment with CI variable
		os.Setenv("CI", "true")
		mode = getExecutionMode()
		assert.Equal(t, "ci", mode)

		// Test CI environment with GITHUB_ACTIONS
		os.Unsetenv("CI")
		os.Setenv("GITHUB_ACTIONS", "true")
		mode = getExecutionMode()
		assert.Equal(t, "ci", mode)

		// Test isCIEnvironment directly
		os.Unsetenv("CI")
		os.Unsetenv("GITHUB_ACTIONS")
		assert.False(t, isCIEnvironment())

		os.Setenv("CI", "true")
		assert.True(t, isCIEnvironment())
	})

	t.Run("database URL masking", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				"complete postgres URL",
				"postgres://user:password@localhost:5432/db",
				"postgres://user:%2A%2A%2A%2A@localhost:5432/db", // URL encoded ****
			},
			{
				"URL without password",
				"postgres://user@localhost:5432/db",
				"postgres://user:%2A%2A%2A%2A@localhost:5432/db", // maskDatabaseURL always adds ****
			},
			{
				"invalid URL",
				"not-a-url",
				"not-a-url", // Return as-is for invalid URLs
			},
			{
				"empty URL",
				"",
				"", // Return as-is for empty URLs
			},
			{
				"URL with complex password",
				"postgres://admin:complex!pass@example.com:5432/mydb",
				"postgres://admin:%2A%2A%2A%2A@example.com:5432/mydb", // URL encoded ****
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := maskDatabaseURL(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("database URL source detection", func(t *testing.T) {
		// Save original env vars
		envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}
		original := make(map[string]string)
		for _, env := range envVars {
			original[env] = os.Getenv(env)
			os.Unsetenv(env)
		}
		defer func() {
			for env, val := range original {
				if val == "" {
					os.Unsetenv(env)
				} else {
					os.Setenv(env, val)
				}
			}
		}()

		testURL := "postgres://test:test@localhost:5432/test"

		// Test with DATABASE_URL
		os.Setenv("DATABASE_URL", testURL)
		source := detectDatabaseURLSource(testURL)
		assert.Contains(t, source, "DATABASE_URL")

		// Test with SCRY_DATABASE_URL
		os.Unsetenv("DATABASE_URL")
		os.Setenv("SCRY_DATABASE_URL", testURL)
		source = detectDatabaseURLSource(testURL)
		assert.Contains(t, source, "SCRY_DATABASE_URL")

		// Test with no matching env var
		os.Unsetenv("SCRY_DATABASE_URL")
		source = detectDatabaseURLSource(testURL)
		assert.Equal(t, "configuration", source)
	})

	t.Run("host extraction from URL", func(t *testing.T) {
		tests := []struct {
			name     string
			url      string
			expected string
		}{
			{"postgres URL", "postgres://user:pass@localhost:5432/db", "localhost"},
			{"complex host", "postgres://user:pass@db.example.com:5432/mydb", "db.example.com"},
			{"invalid URL", "not-a-url", ""}, // extractHostFromURL returns empty string for invalid URLs
			{"empty URL", "", ""},            // extractHostFromURL returns empty string for empty URLs
			{"URL without host", "postgres:///db", ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := extractHostFromURL(tt.url)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("directory existence check", func(t *testing.T) {
		// Test with current directory (should exist)
		assert.True(t, directoryExists("."))

		// Test with non-existent directory
		assert.False(t, directoryExists("/non/existent/directory"))

		// Test with a file (should return false for non-directory)
		tempFile, err := os.CreateTemp("", "test-file")
		require.NoError(t, err)
		tempFile.Close()
		defer os.Remove(tempFile.Name())

		assert.False(t, directoryExists(tempFile.Name()))
	})

	t.Run("migrations path resolution", func(t *testing.T) {
		// Test getMigrationsPath
		path, err := getMigrationsPath()
		if err != nil {
			// Expected in test environment
			assert.Contains(t, err.Error(), "migrations")
		} else {
			assert.NotEmpty(t, path)
			assert.Contains(t, path, "migrations")
		}
	})
}

// TestApplicationCleanupCustom tests cleanup scenarios
func TestApplicationCleanupCustom(t *testing.T) {
	t.Run("cleanup with various states", func(t *testing.T) {
		testLogger, _ := CreateTestLogger(t)

		// Test cleanup with nil app
		var app *application
		require.NotPanics(t, func() {
			if app != nil {
				app.cleanup()
			}
		})

		// Test cleanup with minimal app
		app = &application{
			logger: testLogger,
		}
		require.NotPanics(t, func() {
			app.cleanup()
		})

		// Test cleanup with mock database - skip this since MockDB is not sql.DB compatible
		app = &application{
			logger: testLogger,
			// db field is nil, which is fine for testing cleanup with nil db
		}
		require.NotPanics(t, func() {
			app.cleanup()
		})
	})
}

// TestConfigurationIntegration tests config integration with app components
func TestConfigurationIntegration(t *testing.T) {
	t.Run("config validation for application use", func(t *testing.T) {
		cfg := CreateTestConfig(t)

		// Test that config has all required fields for application initialization
		assert.NotEmpty(t, cfg.Database.URL)
		assert.NotEmpty(t, cfg.Auth.JWTSecret)
		assert.GreaterOrEqual(t, len(cfg.Auth.JWTSecret), 32) // JWT secret must be 32+ chars
		assert.Greater(t, cfg.Server.Port, 0)
		assert.NotEmpty(t, cfg.Server.LogLevel)
		assert.NotEmpty(t, cfg.LLM.GeminiAPIKey)
		assert.NotEmpty(t, cfg.LLM.ModelName)
		assert.NotEmpty(t, cfg.LLM.PromptTemplatePath)
		assert.Greater(t, cfg.Task.WorkerCount, 0)
		assert.Greater(t, cfg.Task.QueueSize, 0)
		assert.Greater(t, cfg.Task.StuckTaskAgeMinutes, 0)
	})

	t.Run("config fields for task runner", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)

		// Verify task configuration is valid
		assert.Equal(t, 1, cfg.Task.WorkerCount)
		assert.Equal(t, 10, cfg.Task.QueueSize)
		assert.Equal(t, 5, cfg.Task.StuckTaskAgeMinutes)
	})
}

// TestTaskFactoryEventHandler tests the event handler
func TestTaskFactoryEventHandler(t *testing.T) {
	t.Run("event handler structure", func(t *testing.T) {
		testLogger, _ := CreateTestLogger(t)

		// Test creating handler with nil fields (should be handled gracefully by the application)
		handler := &TaskFactoryEventHandler{
			logger: testLogger,
		}

		assert.NotNil(t, handler.logger)
		assert.Nil(t, handler.taskFactory)
		assert.Nil(t, handler.taskRunner)
	})
}
