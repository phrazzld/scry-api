package main

import (
	"testing"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// For backwards compatibility in this file
// In new tests, use testutils.CreateTempConfigFile directly
func createTempConfigFile(t *testing.T, content string) (string, func()) {
	return testutils.CreateTempConfigFile(t, content)
}

// TestSuccessfulInitialization verifies the application initializes correctly
// with valid configuration from environment variables
func TestSuccessfulInitialization(t *testing.T) {
	// Setup required environment variables
	cleanup := testutils.SetupEnv(t, map[string]string{
		"SCRY_SERVER_PORT":                 "9090",
		"SCRY_SERVER_LOG_LEVEL":            "debug",
		"SCRY_DATABASE_URL":                "postgresql://user:pass@localhost:5432/testdb",
		"SCRY_AUTH_JWT_SECRET":             "thisisasecretkeythatis32charslong!!",
		"SCRY_AUTH_TOKEN_LIFETIME_MINUTES": "60", // 1 hour
		"SCRY_LLM_GEMINI_API_KEY":          "test-api-key",
	})
	defer cleanup()

	// Initialize the application
	cfg, err := initializeApp()

	// Verify initialization succeeded
	require.NoError(t, err, "Application initialization should succeed with valid config")
	require.NotNil(t, cfg, "Configuration should not be nil")

	// Verify config values were loaded correctly
	assert.Equal(t, 9090, cfg.Server.Port, "Server port should be loaded from environment variables")
	assert.Equal(t, "debug", cfg.Server.LogLevel, "Log level should be loaded from environment variables")
	assert.Equal(
		t,
		"postgresql://user:pass@localhost:5432/testdb",
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
}

// TestValidationErrors verifies that configuration validation works correctly
// and returns appropriate errors
func TestValidationErrors(t *testing.T) {
	// Clean environment to avoid interference
	cleanup := testutils.SetupEnv(t, map[string]string{
		"SCRY_SERVER_PORT":                 "999999",  // Invalid port (too high)
		"SCRY_SERVER_LOG_LEVEL":            "invalid", // Invalid log level
		"SCRY_DATABASE_URL":                "",        // Required field missing
		"SCRY_AUTH_JWT_SECRET":             "short",   // Too short
		"SCRY_AUTH_TOKEN_LIFETIME_MINUTES": "50000",   // Too high
		"SCRY_LLM_GEMINI_API_KEY":          "",        // Required field missing
	})
	defer cleanup()

	// Test loading
	_, err := config.Load()

	// Verify error handling
	require.Error(t, err, "Loading invalid config should return error")
	assert.Contains(t, err.Error(), "validation failed", "Error should mention validation failure")

	// Check error message directly, no need to extract validation errors

	// Check for specific validation failures in the error message
	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "Server.Port", "Error should mention invalid port")
	assert.Contains(t, errorMsg, "Server.LogLevel", "Error should mention invalid log level")
	assert.Contains(t, errorMsg, "Database.URL", "Error should mention missing URL")
	assert.Contains(t, errorMsg, "Auth.JWTSecret", "Error should mention invalid JWT secret")
	assert.Contains(t, errorMsg, "Auth.TokenLifetimeMinutes", "Error should mention invalid token lifetime")
	assert.Contains(t, errorMsg, "LLM.GeminiAPIKey", "Error should mention missing API key")
}
