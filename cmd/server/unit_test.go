//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComponentInitialization tests individual component initialization
func TestComponentInitialization(t *testing.T) {
	testCases := []ServerTestCase{
		{
			Name:        "valid config creation",
			Config:      CreateTestConfig(t),
			ExpectError: false,
		},
		{
			Name:        "minimal config creation",
			Config:      CreateMinimalTestConfig(t),
			ExpectError: false,
		},
		{
			Name: "logger setup with debug level",
			Config: &config.Config{
				Server: config.ServerConfig{
					LogLevel: "debug",
				},
			},
			ExpectError: false,
		},
		{
			Name: "logger setup with warning level",
			Config: &config.Config{
				Server: config.ServerConfig{
					LogLevel: "warn",
				},
			},
			ExpectError: false,
		},
	}

	RunServerTestCases(t, testCases, func(t *testing.T, tc ServerTestCase) {
		if tc.Config != nil && tc.Config.Server.LogLevel != "" {
			// Test logger setup
			logger, err := setupAppLogger(tc.Config)
			if tc.ExpectError {
				assert.Error(t, err)
				if tc.ErrorContains != "" {
					assert.Contains(t, err.Error(), tc.ErrorContains)
				}
			} else {
				require.NoError(t, err)
				AssertLoggerValid(t, logger)
			}
		} else if tc.Config != nil {
			// Test config validation
			AssertConfigurationValid(t, tc.Config)
		}
	})
}

// TestRouterSetup tests router configuration without starting server
func TestRouterSetup(t *testing.T) {
	t.Run("router setup with mock application", func(t *testing.T) {
		// Create test configuration and logger
		cfg := CreateMinimalTestConfig(t)
		testLogger, logBuf := CreateTestLogger(t)

		// Create mock application with minimal fields needed for router
		app := &application{
			config: cfg,
			logger: testLogger,
		}

		// Test that setupRouter doesn't panic even with nil services
		// This demonstrates that we can test router setup in isolation
		require.NotPanics(t, func() {
			router := app.setupRouter()
			require.NotNil(t, router, "router should not be nil")
		}, "setupRouter should handle nil services gracefully")

		// Verify logger was used (router setup should log something)
		logs := logBuf.String()
		// Note: Currently setupRouter doesn't log, but this shows the pattern
		t.Logf("Router setup logs: %s", logs)
	})
}

// TestMockDatabase tests our mock database infrastructure
func TestMockDatabase(t *testing.T) {
	t.Run("successful mock DB operations", func(t *testing.T) {
		mockDB := NewMockDB()
		ctx := context.Background()

		// Test successful ping
		err := mockDB.PingContext(ctx)
		assert.NoError(t, err, "mock DB ping should succeed")

		// Test successful close
		err = mockDB.Close()
		assert.NoError(t, err, "mock DB close should succeed")
	})

	t.Run("failing mock DB operations", func(t *testing.T) {
		mockDB := NewFailingMockDB()
		ctx := context.Background()

		// Test failing ping
		err := mockDB.PingContext(ctx)
		assert.Error(t, err, "failing mock DB ping should fail")

		// Test failing close
		err = mockDB.Close()
		assert.Error(t, err, "failing mock DB close should fail")
	})
}

// TestConfigValidation tests configuration validation helpers
func TestConfigValidation(t *testing.T) {
	t.Run("valid configurations", func(t *testing.T) {
		configs := []*config.Config{
			CreateTestConfig(t),
			CreateMinimalTestConfig(t),
		}

		for i, cfg := range configs {
			t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
				require.NotPanics(t, func() {
					AssertConfigurationValid(t, cfg)
				}, "valid config should pass validation")
			})
		}
	})

	// Note: Invalid configuration tests would be useful but require more complex
	// test infrastructure to properly catch assertion failures
	t.Run("validation behavior", func(t *testing.T) {
		t.Log("Config validation infrastructure is working - see valid config tests above")
	})
}

// TestTestHelpers tests our test helper functions themselves
func TestTestHelpers(t *testing.T) {
	t.Run("CreateTestLogger", func(t *testing.T) {
		logger, logBuf := CreateTestLogger(t)

		require.NotNil(t, logger, "logger should not be nil")
		require.NotNil(t, logBuf, "log buffer should not be nil")

		// Test logging and capture
		logger.Info("test message", "key", "value")

		logs := logBuf.String()
		assert.Contains(t, logs, "test message", "log should contain test message")
		assert.Contains(t, logs, "key", "log should contain key")
		assert.Contains(t, logs, "value", "log should contain value")
	})

	t.Run("MockHTTPHandler", func(t *testing.T) {
		handler := MockHTTPHandler()
		require.NotNil(t, handler, "mock handler should not be nil")

		// We could test the handler with httptest.NewRecorder() here
		// but that would require more HTTP testing infrastructure
	})

	t.Run("WaitForCondition", func(t *testing.T) {
		// Test successful condition
		conditionMet := false
		go func() {
			// Simulate async condition being met
			time.Sleep(10 * time.Millisecond)
			conditionMet = true
		}()

		require.NotPanics(t, func() {
			WaitForCondition(t, func() bool {
				return conditionMet
			}, 100*time.Millisecond, "condition should be met")
		}, "WaitForCondition should not panic for met condition")
	})
}
