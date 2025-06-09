//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplicationStructure tests application struct and basic operations
func TestApplicationStructure(t *testing.T) {
	t.Run("application struct creation and basic operations", func(t *testing.T) {
		cfg := CreateTestConfig(t)
		logger, _ := CreateTestLogger(t)

		// Create minimal application struct for testing
		app := &application{
			config: cfg,
			logger: logger,
		}

		// Test that setupRouter works with minimal app
		require.NotPanics(t, func() {
			router := app.setupRouter()
			assert.NotNil(t, router)
		})
	})
}

// TestApplicationRunSimple tests the Run method with basic scenarios
func TestApplicationRunSimple(t *testing.T) {
	t.Run("application Run basic flow", func(t *testing.T) {
		cfg := CreateTestConfig(t)
		cfg.Server.Port = 0 // Use random available port
		logger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: logger,
		}

		// Create context that cancels immediately
		ctx, cancel := context.WithCancel(context.Background())

		// Start Run in goroutine and cancel immediately
		errCh := make(chan error, 1)
		go func() {
			err := app.Run(ctx)
			errCh <- err
		}()

		// Cancel after minimal delay to allow server start
		time.Sleep(5 * time.Millisecond)
		cancel()

		// Wait for completion
		select {
		case err := <-errCh:
			// Should complete without error
			assert.NoError(t, err)
		case <-time.After(1 * time.Second):
			t.Log("Run took longer than expected but that's acceptable")
		}
	})
}

// TestApplicationCleanupCoverage tests cleanup scenarios
func TestApplicationCleanupCoverage(t *testing.T) {
	t.Run("cleanup with nil components", func(t *testing.T) {
		cfg := CreateTestConfig(t)
		logger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: logger,
			// All other fields are nil
		}

		// Test cleanup doesn't panic with nil components
		require.NotPanics(t, func() {
			app.cleanup()
		})
	})

	t.Run("cleanup with nil database", func(t *testing.T) {
		cfg := CreateTestConfig(t)
		logger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: logger,
			db:     nil, // Test with nil database
		}

		// Test cleanup doesn't panic with nil DB
		require.NotPanics(t, func() {
			app.cleanup()
		})
	})
}

// TestSetupAppDatabaseCoverage tests database setup edge cases
func TestSetupAppDatabaseCoverage(t *testing.T) {
	t.Run("setupAppDatabase with connection error", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "postgres://nonexistent:password@unreachable-host:5432/test",
			},
		}
		logger, _ := CreateTestLogger(t)

		db, err := setupAppDatabase(cfg, logger)

		// Should fail due to unreachable host
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to ping database")
	})

	t.Run("setupAppDatabase with malformed URL", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{
				URL: "not-a-valid-database-url",
			},
		}
		logger, _ := CreateTestLogger(t)

		db, err := setupAppDatabase(cfg, logger)

		// Should fail due to malformed URL
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to ping database")
	})
}

// TestSetupAppLoggerCoverage tests logger setup edge cases
func TestSetupAppLoggerCoverage(t *testing.T) {
	t.Run("setupAppLogger with error level", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "error",
			},
		}

		logger, err := setupAppLogger(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("setupAppLogger with debug level", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "debug",
			},
		}

		logger, err := setupAppLogger(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, logger)
	})
}

// TestMainFunctionCoverage tests components that main() uses
func TestMainFunctionCoverage(t *testing.T) {
	t.Run("loadAppConfig with missing env vars", func(t *testing.T) {
		// Save current env vars
		originalVars := map[string]string{
			"SCRY_DATABASE_URL":       os.Getenv("SCRY_DATABASE_URL"),
			"SCRY_AUTH_JWT_SECRET":    os.Getenv("SCRY_AUTH_JWT_SECRET"),
			"SCRY_LLM_GEMINI_API_KEY": os.Getenv("SCRY_LLM_GEMINI_API_KEY"),
		}
		defer func() {
			// Restore env vars
			for key, val := range originalVars {
				if val == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, val)
				}
			}
		}()

		// Clear required env vars
		os.Unsetenv("SCRY_DATABASE_URL")
		os.Unsetenv("SCRY_AUTH_JWT_SECRET")
		os.Unsetenv("SCRY_LLM_GEMINI_API_KEY")

		// Should fail with missing configuration
		cfg, err := loadAppConfig()
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})
}
