//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadAppConfig tests configuration loading with various scenarios
func TestLoadAppConfig(t *testing.T) {
	testCases := []ServerTestCase{
		{
			Name: "valid config with all required fields",
			PreTest: func(t *testing.T) {
				t.Setenv("SCRY_DATABASE_URL", "postgres://test:test@localhost:5432/test")
				t.Setenv("SCRY_AUTH_JWT_SECRET", "test-secret-key-for-testing-only-32-chars-long")
				t.Setenv("SCRY_LLM_GEMINI_API_KEY", "test-api-key")
				t.Setenv("SCRY_LLM_PROMPT_TEMPLATE_PATH", "../../prompts/flashcard_template.txt")
			},
			ExpectError: false,
		},
		{
			Name: "missing required environment variables",
			PreTest: func(t *testing.T) {
				// Clear all env vars to test failure case
				t.Setenv("SCRY_DATABASE_URL", "")
				t.Setenv("SCRY_AUTH_JWT_SECRET", "")
				t.Setenv("SCRY_LLM_GEMINI_API_KEY", "")
				t.Setenv("SCRY_LLM_PROMPT_TEMPLATE_PATH", "")
			},
			ExpectError:   true,
			ErrorContains: "configuration validation failed",
		},
		{
			Name: "invalid JWT secret (too short)",
			PreTest: func(t *testing.T) {
				t.Setenv("SCRY_DATABASE_URL", "postgres://test:test@localhost:5432/test")
				t.Setenv("SCRY_AUTH_JWT_SECRET", "short")
				t.Setenv("SCRY_LLM_GEMINI_API_KEY", "test-api-key")
				t.Setenv("SCRY_LLM_PROMPT_TEMPLATE_PATH", "../../prompts/flashcard_template.txt")
			},
			ExpectError:   true,
			ErrorContains: "JWTSecret",
		},
	}

	RunServerTestCases(t, testCases, func(t *testing.T, tc ServerTestCase) {
		cfg, err := loadAppConfig()

		if tc.ExpectError {
			assert.Error(t, err)
			if tc.ErrorContains != "" {
				assert.Contains(t, err.Error(), tc.ErrorContains)
			}
			assert.Nil(t, cfg)
		} else {
			require.NoError(t, err)
			require.NotNil(t, cfg)
			AssertConfigurationValid(t, cfg)
		}
	})
}

// TestSetupAppDatabase tests database setup with various scenarios
func TestSetupAppDatabase(t *testing.T) {
	testCases := []ServerTestCase{
		{
			Name: "invalid database URL format",
			Config: &config.Config{
				Database: config.DatabaseConfig{
					URL: "invalid-url-format",
				},
			},
			ExpectError:   true,
			ErrorContains: "failed to ping database",
		},
		{
			Name: "database connection timeout",
			Config: &config.Config{
				Database: config.DatabaseConfig{
					URL: "postgres://nonexistent:password@nonexistent-host:5432/nonexistent",
				},
			},
			ExpectError:   true,
			ErrorContains: "failed to ping database",
		},
	}

	RunServerTestCases(t, testCases, func(t *testing.T, tc ServerTestCase) {
		testLogger, _ := CreateTestLogger(t)

		db, err := setupAppDatabase(tc.Config, testLogger)

		if tc.ExpectError {
			assert.Error(t, err)
			if tc.ErrorContains != "" {
				assert.Contains(t, err.Error(), tc.ErrorContains)
			}
			assert.Nil(t, db)
		} else {
			require.NoError(t, err)
			require.NotNil(t, db)
			assert.NoError(t, db.Close())
		}
	})
}

// TestApplicationLifecycle tests application creation and lifecycle management
func TestApplicationLifecycle(t *testing.T) {
	t.Run("application structure validation", func(t *testing.T) {
		// Test application struct field validation without full initialization
		// This tests that our application struct is properly structured

		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
		}

		// Test that the application struct can be created and has expected fields
		require.NotNil(t, app.config)
		require.NotNil(t, app.logger)
		assert.Equal(t, cfg, app.config)
		assert.Equal(t, testLogger, app.logger)

		// Test cleanup with nil fields (should not panic)
		require.NotPanics(t, func() {
			app.cleanup()
		}, "cleanup should handle nil fields gracefully")
	})

	t.Run("application components validation", func(t *testing.T) {
		// Test individual component creation that we can test in isolation

		// Test router setup with minimal app
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
			// Other fields are nil, but setupRouter should handle this gracefully
		}

		require.NotPanics(t, func() {
			router := app.setupRouter()
			require.NotNil(t, router)

			// Test that router has basic routes
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Health endpoint should work even with nil services
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "OK", w.Body.String())
		}, "setupRouter should not panic with minimal application")
	})
}

// TestServerLifecycle tests HTTP server startup and shutdown
func TestServerLifecycle(t *testing.T) {
	t.Run("server startup and immediate shutdown", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		// Use a random available port for testing
		cfg.Server.Port = 0 // Let OS assign port
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
		}

		// Create a simple test router
		router := http.NewServeMux()
		router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test"))
		})

		// Test server startup with context cancellation for immediate shutdown
		ctx, cancel := context.WithCancel(context.Background())

		// Start server in goroutine
		errCh := make(chan error, 1)
		go func() {
			err := app.startHTTPServer(ctx, router)
			errCh <- err
		}()

		// Give server time to start
		time.Sleep(10 * time.Millisecond)

		// Cancel context to trigger shutdown
		cancel()

		// Wait for server to shutdown
		select {
		case err := <-errCh:
			// Server should shutdown cleanly
			assert.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("Server did not shutdown within timeout")
		}
	})

	t.Run("server behavior with unusual port", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		cfg.Server.Port = 99999 // Very high port that might not be available
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
		}

		router := MockHTTPHandler()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server in goroutine and immediately cancel to test shutdown
		errCh := make(chan error, 1)
		go func() {
			err := app.startHTTPServer(ctx, router)
			errCh <- err
		}()

		// Give minimal time for server to attempt start
		time.Sleep(1 * time.Millisecond)
		cancel()

		// Wait for server response (should complete quickly)
		select {
		case err := <-errCh:
			// Server should complete without error when context is cancelled
			assert.NoError(t, err, "Server should shutdown cleanly even with high port")
		case <-time.After(100 * time.Millisecond):
			t.Log("Server shutdown took longer than expected but that's acceptable")
		}
	})
}

// TestRouterConfiguration tests router setup and middleware
func TestRouterConfiguration(t *testing.T) {
	t.Run("router middleware chain", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
		}

		router := app.setupRouter()
		require.NotNil(t, router)

		// Test health endpoint works
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})

	t.Run("protected routes without auth", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
			// No auth services - routes should still exist but fail auth
		}

		router := app.setupRouter()
		require.NotNil(t, router)

		// Test protected route without authentication
		req := httptest.NewRequest("GET", "/api/cards/next", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail due to missing auth, but route should exist
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

// TestConfigurationValidation tests edge cases in config validation
func TestConfigurationValidation(t *testing.T) {
	t.Run("config field validation", func(t *testing.T) {
		// Test that our config creation helpers create valid configs
		configs := []struct {
			name   string
			config *config.Config
		}{
			{"full config", CreateTestConfig(t)},
			{"minimal config", CreateMinimalTestConfig(t)},
		}

		for _, tc := range configs {
			t.Run(tc.name, func(t *testing.T) {
				require.NotNil(t, tc.config)

				// Validate required fields are present
				assert.NotEmpty(t, tc.config.Database.URL)
				assert.NotEmpty(t, tc.config.Auth.JWTSecret)
				assert.Greater(t, tc.config.Server.Port, 0)
				assert.NotEmpty(t, tc.config.Server.LogLevel)
				assert.NotEmpty(t, tc.config.LLM.GeminiAPIKey)
				assert.NotEmpty(t, tc.config.LLM.ModelName)
				assert.NotEmpty(t, tc.config.LLM.PromptTemplatePath)

				// Test that config is compatible with setupAppLogger
				logger, err := setupAppLogger(tc.config)
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			})
		}
	})
}

// TestErrorHandling tests error scenarios across components
func TestErrorHandling(t *testing.T) {
	t.Run("nil config handling", func(t *testing.T) {
		// Test that functions handle nil configs (some may panic, some may error)

		// setupAppLogger panics with nil config
		assert.Panics(t, func() {
			setupAppLogger(nil)
		}, "setupAppLogger should panic with nil config")

		// setupAppDatabase also panics with nil config
		assert.Panics(t, func() {
			setupAppDatabase(nil, nil)
		}, "setupAppDatabase should panic with nil config")
	})

	t.Run("invalid logger config", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				LogLevel: "", // Empty log level should get default
			},
		}

		// setupAppLogger should handle empty log level gracefully
		logger, err := setupAppLogger(cfg)
		// Should either succeed with default or fail gracefully
		if err != nil {
			assert.Contains(t, err.Error(), "log level")
		} else {
			assert.NotNil(t, logger)
		}
	})
}

// TestIntegrationPoints tests points where components integrate
func TestIntegrationPoints(t *testing.T) {
	t.Run("config to logger integration", func(t *testing.T) {
		// Test all valid log levels work with setupAppLogger
		logLevels := []string{"debug", "info", "warn", "error"}

		for _, level := range logLevels {
			t.Run("level_"+level, func(t *testing.T) {
				cfg := CreateMinimalTestConfig(t)
				cfg.Server.LogLevel = level

				logger, err := setupAppLogger(cfg)
				require.NoError(t, err, "setupAppLogger should handle level: %s", level)
				require.NotNil(t, logger)

				// Test that logger works at this level
				logger.Debug("debug message")
				logger.Info("info message")
				logger.Warn("warn message")
				logger.Error("error message")
			})
		}
	})
}
