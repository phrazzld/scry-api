//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadAppConfigExtensive tests the loadAppConfig function
func TestLoadAppConfigExtensive(t *testing.T) {
	testCases := []struct {
		name          string
		envVars       map[string]string
		expectError   bool
		errorContains string
		setupFunc     func(t *testing.T)
		cleanupFunc   func(t *testing.T)
	}{
		{
			name: "successful config load with environment variables",
			envVars: map[string]string{
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
			},
			expectError: false,
		},
		{
			name: "minimal environment variables",
			envVars: map[string]string{
				"SCRY_DATABASE_URL":    "postgres://test:test@localhost:5432/test",
				"SCRY_AUTH_JWT_SECRET": "test-jwt-secret-key-32-chars-123",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup environment variables
			originalEnv := make(map[string]string)
			for key, value := range tc.envVars {
				originalEnv[key] = os.Getenv(key)
				os.Setenv(key, value)
			}

			// Setup function
			if tc.setupFunc != nil {
				tc.setupFunc(t)
			}

			// Cleanup function
			defer func() {
				if tc.cleanupFunc != nil {
					tc.cleanupFunc(t)
				}
				// Restore original environment
				for key := range tc.envVars {
					if originalVal, existed := originalEnv[key]; existed {
						os.Setenv(key, originalVal)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			// Test loadAppConfig
			cfg, err := loadAppConfig()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)

				// Validate basic config structure
				assert.Greater(t, cfg.Server.Port, 0)
				assert.NotEmpty(t, cfg.Server.LogLevel)
				if tc.envVars["SCRY_DATABASE_URL"] != "" {
					assert.Equal(t, tc.envVars["SCRY_DATABASE_URL"], cfg.Database.URL)
				}
			}
		})
	}
}

// TestSetupAppLoggerExtensive tests the setupAppLogger function
func TestSetupAppLoggerExtensive(t *testing.T) {
	testCases := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "successful logger setup with info level",
			config: &config.Config{
				Server: config.ServerConfig{
					LogLevel: "info",
				},
			},
			expectError: false,
		},
		{
			name: "successful logger setup with debug level",
			config: &config.Config{
				Server: config.ServerConfig{
					LogLevel: "debug",
				},
			},
			expectError: false,
		},
		{
			name: "successful logger setup with warn level",
			config: &config.Config{
				Server: config.ServerConfig{
					LogLevel: "warn",
				},
			},
			expectError: false,
		},
		{
			name: "successful logger setup with error level",
			config: &config.Config{
				Server: config.ServerConfig{
					LogLevel: "error",
				},
			},
			expectError: false,
		},
		{
			name: "invalid log level should still work (defaults to info)",
			config: &config.Config{
				Server: config.ServerConfig{
					LogLevel: "invalid",
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := setupAppLogger(tc.config)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				require.NoError(t, err)
				require.NotNil(t, logger)

				// Test that logger works
				require.NotPanics(t, func() {
					logger.Info("test message")
					logger.Debug("debug message")
					logger.Warn("warn message")
					logger.Error("error message")
				})
			}
		})
	}
}

// TestSetupRouterExtensive tests the setupRouter method on application
func TestSetupRouterExtensive(t *testing.T) {
	t.Run("router setup with minimal application", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		logger, _ := CreateTestLogger(t)

		// Create minimal application for router testing
		app := &application{
			config: cfg,
			logger: logger,
			// Services can be nil for router setup test
		}

		// Test that setupRouter works even with nil services
		require.NotPanics(t, func() {
			router := app.setupRouter()
			require.NotNil(t, router)

			// Test that router responds to health check
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "OK", w.Body.String())
		})
	})

	t.Run("router setup with full application mock", func(t *testing.T) {
		cfg := CreateTestConfig(t)
		logger, _ := CreateTestLogger(t)

		// Create mock services for more complete router testing
		app := &application{
			config:            cfg,
			logger:            logger,
			userStore:         nil, // These would be mocked in a full test
			jwtService:        nil,
			passwordVerifier:  nil,
			memoService:       nil,
			cardReviewService: nil,
			cardService:       nil,
		}

		// Router should handle nil services gracefully
		require.NotPanics(t, func() {
			router := app.setupRouter()
			require.NotNil(t, router)
		}, "setupRouter should handle nil services gracefully")
	})
}

// TestStartHTTPServerExtensive tests the startHTTPServer function with immediate cancellation
func TestStartHTTPServerExtensive(t *testing.T) {
	t.Run("server starts and stops immediately", func(t *testing.T) {
		cfg := CreateTestConfig(t)
		cfg.Server.Port = 0 // Use random available port
		logger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: logger,
		}

		// Create a router
		router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test"))
		})

		// Create context that cancels immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Start server - should handle cancellation gracefully
		err := app.startHTTPServer(ctx, router)

		// Server should shut down gracefully even with immediate cancellation
		assert.NoError(t, err)
	})

	t.Run("server starts and stops with timeout", func(t *testing.T) {
		cfg := CreateTestConfig(t)
		cfg.Server.Port = 0 // Use random available port
		logger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: logger,
		}

		router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Start server - should handle timeout gracefully
		err := app.startHTTPServer(ctx, router)
		assert.NoError(t, err)
	})
}

// TestApplicationCleanupExtensive tests the cleanup method
func TestApplicationCleanupExtensive(t *testing.T) {
	t.Run("cleanup with nil components", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		logger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: logger,
			// All other fields nil
		}

		// Cleanup should not panic with nil components
		require.NotPanics(t, func() {
			app.cleanup()
		})
	})

	t.Run("cleanup with mock components", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		logger, _ := CreateTestLogger(t)

		app := &application{
			config:     cfg,
			logger:     logger,
			taskRunner: nil, // Would be a mock in full test
		}

		// Cleanup should handle nil task runner gracefully
		require.NotPanics(t, func() {
			app.cleanup()
		})
	})
}

// TestApplicationRunExtensive tests the Run method with mock setup
func TestApplicationRunExtensive(t *testing.T) {
	t.Run("run with immediate cancellation", func(t *testing.T) {
		cfg := CreateTestConfig(t)
		cfg.Server.Port = 0 // Use random available port
		logger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: logger,
		}

		// Create context that cancels immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Run should handle immediate cancellation
		err := app.Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("run with timeout", func(t *testing.T) {
		cfg := CreateTestConfig(t)
		cfg.Server.Port = 0 // Use random available port
		logger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: logger,
		}

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Run should handle timeout gracefully
		err := app.Run(ctx)
		assert.NoError(t, err)
	})
}
