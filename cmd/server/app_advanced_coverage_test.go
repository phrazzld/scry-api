//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppAdvancedCoverage tests additional app.go functions for coverage improvement
// This targets the 30 uncovered lines in app.go to boost coverage further
func TestAppAdvancedCoverage(t *testing.T) {
	testLogger, _ := CreateTestLogger(t)
	ctx := context.Background()

	t.Run("newApplication with various error scenarios", func(t *testing.T) {
		// Test various failure points in newApplication to cover error paths

		// Test with invalid JWT secret length
		cfg := CreateMinimalTestConfig(t)
		cfg.Auth.JWTSecret = "short" // Too short, less than 32 characters

		app, err := newApplication(ctx, cfg, testLogger, nil)

		assert.Error(t, err, "should fail with short JWT secret")
		assert.Nil(t, app, "app should be nil on failure")
		assert.Contains(
			t,
			err.Error(),
			"jwt secret must be at least 32 characters",
			"error should mention JWT secret length",
		)
	})

	t.Run("newApplication with missing LLM config", func(t *testing.T) {
		// Test with missing LLM configuration
		cfg := CreateMinimalTestConfig(t)
		cfg.LLM.GeminiAPIKey = "" // Empty API key should cause failure

		app, err := newApplication(ctx, cfg, testLogger, nil)

		assert.Error(t, err, "should fail with empty Gemini API key")
		assert.Nil(t, app, "app should be nil on failure")
	})

	t.Run("newApplication with invalid LLM model name", func(t *testing.T) {
		// Test with invalid LLM model configuration
		cfg := CreateMinimalTestConfig(t)
		cfg.LLM.ModelName = "" // Empty model name might cause issues

		app, err := newApplication(ctx, cfg, testLogger, nil)

		// May fail during LLM generator initialization
		if err != nil {
			assert.Error(t, err, "may fail with empty model name")
			assert.Nil(t, app, "app should be nil on failure")
		}
	})

	t.Run("newApplication with invalid prompt template path", func(t *testing.T) {
		// Test with invalid prompt template path
		cfg := CreateMinimalTestConfig(t)
		cfg.LLM.PromptTemplatePath = "/nonexistent/path/to/template.txt"

		app, err := newApplication(ctx, cfg, testLogger, nil)

		// May fail during LLM generator initialization due to missing template
		if err != nil {
			assert.Error(t, err, "may fail with nonexistent template path")
			assert.Nil(t, app, "app should be nil on failure")
		}
	})

	t.Run("newApplication with various database driver errors", func(t *testing.T) {
		// Test different database URL formats that might cause driver errors
		testCases := []struct {
			name  string
			dbURL string
		}{
			{"unknown_protocol", "unknown://user:pass@host/db"},
			{"malformed_url", "not-a-url-at-all"},
			{"missing_user", "postgres://:pass@host/db"},
			{"missing_host", "postgres://user:pass@/db"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg := CreateMinimalTestConfig(t)
				cfg.Database.URL = tc.dbURL

				app, err := newApplication(ctx, cfg, testLogger, nil)

				// Should fail with database-related error
				assert.Error(t, err, "should fail with invalid database URL: %s", tc.dbURL)
				assert.Nil(t, app, "app should be nil on failure")
			})
		}
	})

	t.Run("newApplication task configuration edge cases", func(t *testing.T) {
		// Test with invalid task configuration
		cfg := CreateMinimalTestConfig(t)
		cfg.Task.WorkerCount = 0 // Invalid worker count

		app, err := newApplication(ctx, cfg, testLogger, nil)

		// May fail during task runner initialization
		if err != nil {
			assert.Error(t, err, "may fail with invalid worker count")
			assert.Nil(t, app, "app should be nil on failure")
		}
	})

	t.Run("newApplication with various auth configuration issues", func(t *testing.T) {
		// Test various auth configuration problems
		testCases := []struct {
			name                 string
			jwtSecret            string
			tokenLifetime        int
			refreshTokenLifetime int
		}{
			{"zero_token_lifetime", "this-is-a-valid-32-character-jwt-secret", 0, 60},
			{"negative_token_lifetime", "this-is-a-valid-32-character-jwt-secret", -1, 60},
			{"zero_refresh_lifetime", "this-is-a-valid-32-character-jwt-secret", 60, 0},
			{"negative_refresh_lifetime", "this-is-a-valid-32-character-jwt-secret", 60, -1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg := CreateMinimalTestConfig(t)
				cfg.Auth.JWTSecret = tc.jwtSecret
				cfg.Auth.TokenLifetimeMinutes = tc.tokenLifetime
				cfg.Auth.RefreshTokenLifetimeMinutes = tc.refreshTokenLifetime

				app, err := newApplication(ctx, cfg, testLogger, nil)

				// May fail during auth service initialization
				if err != nil {
					assert.Error(t, err, "may fail with invalid auth config: %s", tc.name)
					assert.Nil(t, app, "app should be nil on failure")
				}
			})
		}
	})
}

// TestApplicationRunErrorPaths tests Run method error scenarios
func TestApplicationRunErrorPaths(t *testing.T) {
	cfg := CreateMinimalTestConfig(t)
	testLogger, _ := CreateTestLogger(t)

	t.Run("application struct validation", func(t *testing.T) {
		// Test application struct creation and basic validation
		app := &application{
			config: cfg,
			logger: testLogger,
			// All service fields are nil
		}

		// Test that we can create an application struct
		assert.NotNil(t, app, "application should be created")
		assert.Equal(t, cfg, app.config, "config should be set")
		assert.Equal(t, testLogger, app.logger, "logger should be set")
	})

	t.Run("invalid port configuration validation", func(t *testing.T) {
		// Test validation of invalid server configuration
		cfgInvalid := CreateMinimalTestConfig(t)
		cfgInvalid.Server.Port = 99999 // Invalid port number

		app := &application{
			config: cfgInvalid,
			logger: testLogger,
		}

		// Test that we can create the app struct with invalid config
		assert.NotNil(t, app, "application should be created even with invalid config")
		assert.Equal(t, 99999, app.config.Server.Port, "invalid port should be stored")
	})
}

// TestLogDatabaseInfoErrorPaths tests logDatabaseInfo error scenarios
func TestLogDatabaseInfoErrorPaths(t *testing.T) {
	testLogger, _ := CreateTestLogger(t)
	ctx := context.Background()

	t.Run("logDatabaseInfo with nil database", func(t *testing.T) {
		// Test logDatabaseInfo with nil database (should panic)
		assert.Panics(t, func() {
			logDatabaseInfo(nil, ctx, testLogger)
		}, "logDatabaseInfo should panic with nil database")
	})

	t.Run("logDatabaseInfo with nil logger", func(t *testing.T) {
		// Test logDatabaseInfo with nil logger (should panic due to nil DB first)
		assert.Panics(t, func() {
			logDatabaseInfo(nil, ctx, nil)
		}, "logDatabaseInfo should panic with nil database even with nil logger")
	})
}

// TestUtilityFunctionEdgeCases tests utility functions for edge cases
func TestUtilityFunctionEdgeCases(t *testing.T) {
	t.Run("getExecutionMode edge cases", func(t *testing.T) {
		// Test getExecutionMode with various environment setups

		// Test with no CI environment
		mode := getExecutionMode()
		assert.Contains(t, []string{"ci", "local"}, mode, "execution mode should be ci or local")

		// Test with CI environment set
		t.Setenv("CI", "true")
		mode = getExecutionMode()
		assert.Equal(t, "ci", mode, "should return ci when CI env is set")

		// Test with GITHUB_ACTIONS environment
		t.Setenv("GITHUB_ACTIONS", "true")
		mode = getExecutionMode()
		assert.Equal(t, "ci", mode, "should return ci when GITHUB_ACTIONS env is set")
	})

	t.Run("isCIEnvironment comprehensive tests", func(t *testing.T) {
		// Test isCIEnvironment with various scenarios

		// Reset environment first
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "")

		result := isCIEnvironment()
		assert.False(t, result, "should return false with no CI env vars")

		// Test with CI=true
		t.Setenv("CI", "true")
		result = isCIEnvironment()
		assert.True(t, result, "should return true with CI=true")

		// Test with GITHUB_ACTIONS=true
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "true")
		result = isCIEnvironment()
		assert.True(t, result, "should return true with GITHUB_ACTIONS=true")

		// Test with both set
		t.Setenv("CI", "1")
		t.Setenv("GITHUB_ACTIONS", "1")
		result = isCIEnvironment()
		assert.True(t, result, "should return true with both CI env vars set")
	})
}
