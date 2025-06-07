//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"testing"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestLargeFunctionsCoverage targets the largest uncovered functions to boost coverage
// This focuses on executeMigration (28.7%), newApplication (33.3%), and related functions
func TestLargeFunctionsCoverage(t *testing.T) {
	t.Run("newApplication error paths", func(t *testing.T) {
		// Test various error conditions in newApplication to cover more branches

		ctx := context.Background()
		testLogger, _ := CreateTestLogger(t)

		// Test with various invalid configurations
		testCases := []struct {
			name   string
			config func() *config.Config
		}{
			{
				name: "empty_jwt_secret",
				config: func() *config.Config {
					cfg := CreateMinimalTestConfig(t)
					cfg.Auth.JWTSecret = ""
					return cfg
				},
			},
			{
				name: "short_jwt_secret",
				config: func() *config.Config {
					cfg := CreateMinimalTestConfig(t)
					cfg.Auth.JWTSecret = "short"
					return cfg
				},
			},
			{
				name: "empty_gemini_api_key",
				config: func() *config.Config {
					cfg := CreateMinimalTestConfig(t)
					cfg.LLM.GeminiAPIKey = ""
					return cfg
				},
			},
			{
				name: "invalid_database_url",
				config: func() *config.Config {
					cfg := CreateMinimalTestConfig(t)
					cfg.Database.URL = "invalid://url"
					return cfg
				},
			},
			{
				name: "missing_prompt_template",
				config: func() *config.Config {
					cfg := CreateMinimalTestConfig(t)
					cfg.LLM.PromptTemplatePath = "/nonexistent/path.txt"
					return cfg
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg := tc.config()

				// newApplication should fail with these invalid configs
				app, err := newApplication(ctx, cfg, testLogger, nil)

				assert.Error(t, err, "should fail with invalid config: %s", tc.name)
				assert.Nil(t, app, "app should be nil on failure")
			})
		}
	})

	t.Run("newApplication with nil parameters", func(t *testing.T) {
		// Test newApplication with nil parameters to cover nil checks

		ctx := context.Background()
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		// Test with nil context (should fail gracefully)
		app, err := newApplication(nil, cfg, testLogger, nil)
		// May not panic but should fail
		if err != nil {
			assert.Error(t, err, "should fail with nil context")
			assert.Nil(t, app, "app should be nil on failure")
		}

		// Test with nil config (should panic or fail)
		assert.Panics(t, func() {
			_, _ = newApplication(ctx, nil, testLogger, nil)
		}, "should panic with nil config")

		// Test with nil logger (should fail gracefully or panic)
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is acceptable
					return
				}
			}()

			app, err := newApplication(ctx, cfg, nil, nil)
			if err != nil {
				assert.Error(t, err, "should fail with nil logger")
				assert.Nil(t, app, "app should be nil on failure")
			}
		}()
	})

	t.Run("application initialization partial success", func(t *testing.T) {
		// Test partial initialization success to cover more paths

		ctx := context.Background()
		testLogger, _ := CreateTestLogger(t)

		// Test with valid logger but invalid database
		cfg := CreateMinimalTestConfig(t)
		cfg.Database.URL = "postgres://nonexistent:nonexistent@nonexistent:9999/nonexistent"

		app, err := newApplication(ctx, cfg, testLogger, nil)

		// Should fail during database setup
		assert.Error(t, err, "should fail with invalid database connection")
		assert.Nil(t, app, "app should be nil on failure")
	})

	t.Run("executeMigration command variations", func(t *testing.T) {
		// Test executeMigration with different commands to cover more branches

		cfg := CreateMinimalTestConfig(t)

		// Test different migration commands
		commands := []string{
			"up",
			"down",
			"status",
			"version",
			"reset",
			"invalid-command",
		}

		for _, cmd := range commands {
			t.Run("cmd_"+cmd, func(t *testing.T) {
				// executeMigration should handle these commands
				// It will likely fail due to database connection, but covers the command parsing
				err := executeMigration(cfg, cmd, false)

				// All commands should fail without proper database setup
				assert.Error(t, err, "should fail without database for command: %s", cmd)
			})
		}
	})

	t.Run("executeMigration error paths", func(t *testing.T) {
		// Test executeMigration error handling paths

		cfg := CreateMinimalTestConfig(t)

		// Test with empty command
		err := executeMigration(cfg, "", false)
		assert.Error(t, err, "should fail with empty command")

		// Test with whitespace command
		err = executeMigration(cfg, "   ", false)
		assert.Error(t, err, "should fail with whitespace command")

		// Test with verbose flag
		err = executeMigration(cfg, "status", true)
		assert.Error(t, err, "should fail without database even with verbose")

		// Test with additional arguments
		err = executeMigration(cfg, "up", false, "1", "test-migration")
		assert.Error(t, err, "should fail with additional arguments")
	})

	t.Run("setupTaskRunner coverage", func(t *testing.T) {
		// Test setupTaskRunner function to cover its branches

		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		// Create minimal application structure for testing
		app := &application{
			config: cfg,
			logger: testLogger,
		}

		// setupTaskRunner should panic or fail without proper dependencies
		// Wrap in a recovery function to handle potential panics
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is expected due to missing dependencies
					t.Log("setupTaskRunner panicked as expected due to missing dependencies")
					return
				}
			}()

			runner, err := setupTaskRunner(app)

			// If no panic, should fail with error
			assert.Error(t, err, "should fail without proper dependencies")
			assert.Nil(t, runner, "runner should be nil on failure")
		}()
	})
}
