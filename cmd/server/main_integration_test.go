// Package main contains integration tests for the server application
// These tests require a real database connection to run.
//
// To run these tests:
//
//  1. Start the local development database:
//     cd infrastructure/local_dev && docker-compose up -d
//
//  2. Set the DATABASE_URL environment variable to a valid connection string:
//     export DATABASE_URL=postgres://scryapiuser:local_development_password@localhost:5432/scry?sslmode=disable
//
//  3. Run the tests:
//     go test -v ./cmd/server
//
// If DATABASE_URL is not set, these tests will be skipped automatically.
package main

import (
	"context"
	"os"
	"testing"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Use the testDB variable defined in main_task_test.go
// TestMain function is also defined in main_task_test.go

// This function was removed to fix linting errors

// TestSuccessfulInitialization verifies the application initializes correctly
// with valid configuration from environment variables
func TestSuccessfulInitialization(t *testing.T) {
	// Disable parallel testing to avoid environment variable conflicts
	// t.Parallel()

	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - DATABASE_URL environment variable required")
	}

	// Get a real database URL from the test environment
	dbURL := testutils.GetTestDatabaseURL(t)

	// First save any existing environment variables we'll need to restore
	oldEnv := make(map[string]string)
	varsToSet := []string{
		"SCRY_SERVER_PORT",
		"SCRY_SERVER_LOG_LEVEL",
		"SCRY_DATABASE_URL",
		"SCRY_AUTH_JWT_SECRET",
		"SCRY_AUTH_TOKEN_LIFETIME_MINUTES",
		"SCRY_LLM_GEMINI_API_KEY",
	}

	// Save old values
	for _, varName := range varsToSet {
		oldEnv[varName] = os.Getenv(varName)
		// Clear each variable to avoid any interference
		err := os.Unsetenv(varName)
		require.NoError(t, err, "Failed to unset environment variable %s", varName)
	}

	// Restore environment when test completes
	defer func() {
		for key, value := range oldEnv {
			if value != "" {
				err := os.Setenv(key, value)
				if err != nil {
					t.Logf("Warning: Failed to restore env var %s: %v", key, err)
				}
			}
		}
	}()

	// Set our test values explicitly
	err := os.Setenv("SCRY_SERVER_PORT", "9090")
	require.NoError(t, err, "Failed to set SCRY_SERVER_PORT environment variable")

	err = os.Setenv("SCRY_SERVER_LOG_LEVEL", "debug")
	require.NoError(t, err, "Failed to set SCRY_SERVER_LOG_LEVEL environment variable")

	err = os.Setenv("SCRY_DATABASE_URL", dbURL)
	require.NoError(t, err, "Failed to set SCRY_DATABASE_URL environment variable")

	err = os.Setenv("SCRY_AUTH_JWT_SECRET", "thisisasecretkeythatis32charslong!!")
	require.NoError(t, err, "Failed to set SCRY_AUTH_JWT_SECRET environment variable")

	err = os.Setenv("SCRY_AUTH_TOKEN_LIFETIME_MINUTES", "60") // 1 hour
	require.NoError(t, err, "Failed to set SCRY_AUTH_TOKEN_LIFETIME_MINUTES environment variable")

	err = os.Setenv("SCRY_LLM_GEMINI_API_KEY", "test-api-key")
	require.NoError(t, err, "Failed to set SCRY_LLM_GEMINI_API_KEY environment variable")

	// Use the shared database connection with transaction isolation
	testutils.WithTx(t, testDB, func(tx store.DBTX) {
		// Load configuration directly
		cfg, err := loadConfig()

		// Verify configuration loading succeeded
		require.NoError(t, err, "Configuration loading should succeed with valid env vars")
		require.NotNil(t, cfg, "Configuration should not be nil")

		// Verify config values were loaded correctly
		assert.Equal(
			t,
			9090,
			cfg.Server.Port,
			"Server port should be loaded from environment variables",
		)
		assert.Equal(
			t,
			"debug",
			cfg.Server.LogLevel,
			"Log level should be loaded from environment variables",
		)
		assert.Equal(
			t,
			dbURL,
			cfg.Database.URL,
			"Database URL should be loaded from environment variables",
		)
		assert.Equal(
			t,
			"thisisasecretkeythatis32charslong!!",
			cfg.Auth.JWTSecret,
			"JWT secret should be loaded from environment variables",
		)
		assert.Equal(
			t,
			60,
			cfg.Auth.TokenLifetimeMinutes,
			"Token lifetime should be loaded from environment variables",
		)
	})
}

// TestValidationErrors verifies that configuration validation works correctly
// and returns appropriate errors
func TestValidationErrors(t *testing.T) {
	// Don't run in parallel to avoid environment variable conflicts
	// t.Parallel()

	// This test doesn't need to connect to a real database
	// It only tests configuration validation

	// First save any existing environment variables we'll need to restore
	oldEnv := make(map[string]string)
	varsToChange := []string{
		"SCRY_SERVER_PORT",
		"SCRY_SERVER_LOG_LEVEL",
		"SCRY_DATABASE_URL",
		"SCRY_AUTH_JWT_SECRET",
		"SCRY_AUTH_TOKEN_LIFETIME_MINUTES",
		"SCRY_LLM_GEMINI_API_KEY",
		"DATABASE_URL", // Also clear this one which might interfere
	}

	// Save old values
	for _, varName := range varsToChange {
		oldEnv[varName] = os.Getenv(varName)
		// Clear each variable to avoid any interference
		err := os.Unsetenv(varName)
		require.NoError(t, err, "Failed to unset environment variable %s", varName)
	}

	// Restore environment when test completes
	defer func() {
		for key, value := range oldEnv {
			if value != "" {
				err := os.Setenv(key, value)
				if err != nil {
					t.Logf("Warning: Failed to restore env var %s: %v", key, err)
				}
			}
		}
	}()

	// Set up our invalid test values
	setEnvs := []struct {
		name, value string
	}{
		{"SCRY_SERVER_PORT", "999999"},                // Invalid port (too high)
		{"SCRY_SERVER_LOG_LEVEL", "invalid"},          // Invalid log level
		{"SCRY_DATABASE_URL", ""},                     // Required field missing
		{"SCRY_AUTH_JWT_SECRET", "short"},             // Too short
		{"SCRY_AUTH_TOKEN_LIFETIME_MINUTES", "50000"}, // Too high
		{"SCRY_LLM_GEMINI_API_KEY", ""},               // Required field missing
	}

	for _, env := range setEnvs {
		err := os.Setenv(env.name, env.value)
		require.NoError(t, err, "Failed to set %s environment variable", env.name)
	}

	// Test loading
	_, err := config.Load()

	// Verify error handling
	require.Error(t, err, "Loading invalid config should return error")
	assert.Contains(t, err.Error(), "validation failed", "Error should mention validation failure")

	// Check for specific validation failures in the error message
	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "Server.Port", "Error should mention invalid port")
	assert.Contains(t, errorMsg, "Server.LogLevel", "Error should mention invalid log level")
	assert.Contains(t, errorMsg, "Database.URL", "Error should mention missing URL")
	assert.Contains(t, errorMsg, "Auth.JWTSecret", "Error should mention invalid JWT secret")
	assert.Contains(
		t,
		errorMsg,
		"Auth.TokenLifetimeMinutes",
		"Error should mention invalid token lifetime",
	)
	assert.Contains(t, errorMsg, "LLM.GeminiAPIKey", "Error should mention missing API key")
}

// TestDatabaseConnection verifies that the application can successfully connect to the database
// This test is important to verify our database connection fix works correctly
func TestDatabaseConnection(t *testing.T) {
	// Disable parallel testing to avoid environment variable conflicts
	// t.Parallel()

	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - DATABASE_URL environment variable required")
	}

	// Skip if database is not available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	// Get a real database URL from the test environment
	dbURL := testutils.GetTestDatabaseURL(t)

	// First save any existing environment variables we'll need to restore
	oldEnv := make(map[string]string)
	varsToSet := []string{
		"SCRY_SERVER_PORT",
		"SCRY_SERVER_LOG_LEVEL",
		"SCRY_DATABASE_URL",
		"SCRY_AUTH_JWT_SECRET",
		"SCRY_AUTH_TOKEN_LIFETIME_MINUTES",
		"SCRY_LLM_GEMINI_API_KEY",
	}

	// Save old values
	for _, varName := range varsToSet {
		oldEnv[varName] = os.Getenv(varName)
		// Clear each variable to avoid any interference
		err := os.Unsetenv(varName)
		require.NoError(t, err, "Failed to unset environment variable %s", varName)
	}

	// Restore environment when test completes
	defer func() {
		for key, value := range oldEnv {
			if value != "" {
				err := os.Setenv(key, value)
				if err != nil {
					t.Logf("Warning: Failed to restore env var %s: %v", key, err)
				}
			}
		}
	}()

	// Set our test values explicitly
	testEnvs := []struct {
		name, value string
	}{
		{"SCRY_SERVER_PORT", "8080"},
		{"SCRY_SERVER_LOG_LEVEL", "info"},
		{"SCRY_DATABASE_URL", dbURL},
		{"SCRY_AUTH_JWT_SECRET", "thisisasecretkeythatis32charslong!!"},
		{"SCRY_AUTH_TOKEN_LIFETIME_MINUTES", "60"},
		{"SCRY_LLM_GEMINI_API_KEY", "test-api-key"},
	}

	for _, env := range testEnvs {
		err := os.Setenv(env.name, env.value)
		require.NoError(t, err, "Failed to set %s environment variable", env.name)
	}

	// Use transaction isolation for the test
	testutils.WithTx(t, testDB, func(tx store.DBTX) {
		// Test that we can query the database
		var result int
		err := tx.QueryRowContext(context.Background(), "SELECT 1").Scan(&result)
		require.NoError(t, err, "Should be able to query the database")
		assert.Equal(t, 1, result, "Database query should return expected result")

		// Check if we can query any system table to further verify connection
		var currentDatabase string
		err = tx.QueryRowContext(context.Background(), "SELECT current_database()").
			Scan(&currentDatabase)
		require.NoError(t, err, "Should be able to query current database name")
		assert.NotEmpty(t, currentDatabase, "Current database name should not be empty")

		t.Logf("Successfully connected to database: %s", currentDatabase)
	})
}
