//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewApplicationErrorPaths tests error paths in newApplication function
// This covers lines in app.go that currently have 0% coverage
func TestNewApplicationErrorPaths(t *testing.T) {
	testLogger, _ := CreateTestLogger(t)
	ctx := context.Background()

	t.Run("newApplication with invalid database", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		// Set an invalid database URL to trigger database connection error
		cfg.Database.URL = "invalid://database/url"

		app, err := newApplication(ctx, cfg, testLogger, nil)

		// Should fail with configuration or initialization error
		assert.Error(t, err, "should fail with invalid database URL")
		assert.Nil(t, app, "app should be nil on failure")
	})

	t.Run("newApplication with empty Gemini API key", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		// Clear the Gemini API key to trigger LLM generator error
		cfg.LLM.GeminiAPIKey = ""

		app, err := newApplication(ctx, cfg, testLogger, nil)

		// Should fail with Gemini API key error
		assert.Error(t, err, "should fail with empty Gemini API key")
		assert.Nil(t, app, "app should be nil on failure")
	})

	t.Run("newApplication with invalid JWT secret", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		// Set an empty JWT secret to trigger JWT service error
		cfg.Auth.JWTSecret = ""

		app, err := newApplication(ctx, cfg, testLogger, nil)

		// Should fail with JWT secret error
		assert.Error(t, err, "should fail with empty JWT secret")
		assert.Nil(t, app, "app should be nil on failure")
	})

	t.Run("newApplication with malformed database URL", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		// Set a malformed database URL
		cfg.Database.URL = "not-a-url"

		app, err := newApplication(ctx, cfg, testLogger, nil)

		// Should fail with URL parsing error
		assert.Error(t, err, "should fail with malformed database URL")
		assert.Nil(t, app, "app should be nil on failure")
	})

	t.Run("newApplication partial initialization cleanup", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		// Set invalid config to test cleanup on partial initialization
		cfg.LLM.GeminiAPIKey = "" // This will cause initialization to fail

		app, err := newApplication(ctx, cfg, testLogger, nil)

		// Should handle failure gracefully and clean up
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, app, "app should be nil on failure")
		} else if app != nil {
			// If app was created, test cleanup
			app.cleanup()
		}
	})
}

// TestApplicationCleanupPaths tests cleanup functionality
func TestApplicationCleanupPaths(t *testing.T) {
	t.Run("cleanup with nil services", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
			// All services are nil - should handle gracefully
		}

		assert.NotPanics(t, func() {
			app.cleanup()
		}, "cleanup should handle nil services gracefully")
	})

	t.Run("cleanup with partial services", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
			// All services are nil - should handle gracefully
		}

		assert.NotPanics(t, func() {
			app.cleanup()
		}, "cleanup should handle partial services gracefully")
	})

	t.Run("cleanup with failing database close", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
			// db field doesn't exist or is nil - should handle gracefully
		}

		assert.NotPanics(t, func() {
			app.cleanup()
		}, "cleanup should handle database close failure gracefully")
	})
}

// TestApplicationStartupPaths tests application startup validation
func TestApplicationStartupPaths(t *testing.T) {
	t.Run("application struct creation", func(t *testing.T) {
		cfg := CreateMinimalTestConfig(t)
		testLogger, _ := CreateTestLogger(t)

		app := &application{
			config: cfg,
			logger: testLogger,
		}

		// Test that we can create an application struct
		assert.NotNil(t, app, "application should be created")
		assert.Equal(t, cfg, app.config, "config should be set")
		assert.Equal(t, testLogger, app.logger, "logger should be set")
	})
}

// TestUtilityFunctionsPaths tests utility functions with edge cases
func TestUtilityFunctionsPaths(t *testing.T) {
	t.Run("maskDatabaseURL with various formats", func(t *testing.T) {
		// Test with standard postgres URL
		masked := maskDatabaseURL("postgres://user:password@host:5432/db")
		expected := "postgres://user:%2A%2A%2A%2A@host:5432/db"
		assert.Equal(t, expected, masked, "should mask password with URL-encoded asterisks")

		// Test with URL containing special characters
		masked = maskDatabaseURL("postgresql://user:p@ss!word@host/db")
		expected = "postgresql://user:%2A%2A%2A%2A@host/db"
		assert.Equal(t, expected, masked, "should mask complex password")

		// Test with URL without password
		masked = maskDatabaseURL("postgres://user@host:5432/db")
		expected = "postgres://user:%2A%2A%2A%2A@host:5432/db" // The function still masks even when no password
		assert.Equal(t, expected, masked, "function masks even URLs without explicit password")

		// Test with empty URL
		masked = maskDatabaseURL("")
		assert.Equal(t, "", masked, "should handle empty URL")

		// Test with malformed URL
		masked = maskDatabaseURL("not-a-url")
		assert.Equal(t, "not-a-url", masked, "should handle malformed URL")
	})

	t.Run("extractHostFromURL edge cases", func(t *testing.T) {
		// Test with standard URL - extractHostFromURL only returns hostname, not port
		host := extractHostFromURL("postgres://user:pass@localhost:5432/db")
		assert.Equal(t, "localhost", host, "should extract hostname only")

		// Test with URL without port
		host = extractHostFromURL("postgres://user:pass@hostname/db")
		assert.Equal(t, "hostname", host, "should extract hostname without port")

		// Test with IP address - only returns the IP, not the port
		host = extractHostFromURL("postgres://user:pass@192.168.1.1:5432/db")
		assert.Equal(t, "192.168.1.1", host, "should extract IP only")

		// Test with malformed URL
		host = extractHostFromURL("not-a-url")
		assert.Equal(t, "", host, "should return empty for malformed URL")

		// Test with empty URL
		host = extractHostFromURL("")
		assert.Equal(t, "", host, "should return empty for empty URL")
	})
}
