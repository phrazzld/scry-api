package config

import (
	"testing"

	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadDefaults verifies that the Load function sets the expected default values
// for port and log level when no environment variables are set.
func TestLoadDefaults(t *testing.T) {
	// Setup environment with required fields but not the ones with defaults
	cleanup := testutils.SetupEnv(t, map[string]string{
		// Set required fields
		"SCRY_DATABASE_URL":       "postgresql://user:pass@localhost:5432/testdb",
		"SCRY_AUTH_JWT_SECRET":    "thisisasecretkeythatis32charslong!!",
		"SCRY_LLM_GEMINI_API_KEY": "test-api-key",
		// Explicitly unset the ones we want to test defaults for
		"SCRY_SERVER_PORT":      "",
		"SCRY_SERVER_LOG_LEVEL": "",
	})
	defer cleanup()

	// Load configuration
	cfg, err := Load()

	// Verify
	require.NoError(t, err, "Load() should not return an error with default values")
	require.NotNil(t, cfg, "Load() should return a non-nil config")
	assert.Equal(t, 8080, cfg.Server.Port, "Default server port should be 8080")
	assert.Equal(t, "info", cfg.Server.LogLevel, "Default log level should be 'info'")
}

// TestLoadFromEnv verifies that the Load function correctly reads values from environment variables.
func TestLoadFromEnv(t *testing.T) {
	// Setup environment
	cleanup := testutils.SetupEnv(t, map[string]string{
		"SCRY_SERVER_PORT":        "9090",
		"SCRY_SERVER_LOG_LEVEL":   "debug",
		"SCRY_DATABASE_URL":       "postgresql://user:pass@localhost:5432/testdb",
		"SCRY_AUTH_JWT_SECRET":    "thisisasecretkeythatis32charslong!!",
		"SCRY_LLM_GEMINI_API_KEY": "test-api-key",
	})
	defer cleanup()

	// Load configuration
	cfg, err := Load()

	// Verify
	require.NoError(t, err, "Load() should not return an error with valid environment variables")
	require.NotNil(t, cfg, "Load() should return a non-nil config")
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
	assert.Equal(t, "test-api-key", cfg.LLM.GeminiAPIKey, "Gemini API key should be loaded from environment variables")
}

// TestLoadValidationErrors verifies that the Load function correctly validates the configuration.
func TestLoadValidationErrors(t *testing.T) {
	// Test cases with invalid values
	testCases := []struct {
		name           string
		envVars        map[string]string
		expectError    bool
		errorSubstring string
	}{
		{
			name: "Missing required fields",
			envVars: map[string]string{
				"SCRY_SERVER_PORT":      "9090",
				"SCRY_SERVER_LOG_LEVEL": "debug",
				// Missing Database URL, JWT Secret, and Gemini API Key
			},
			expectError:    true,
			errorSubstring: "validation failed",
		},
		{
			name: "Invalid port number",
			envVars: map[string]string{
				"SCRY_SERVER_PORT":        "999999", // Port out of range
				"SCRY_SERVER_LOG_LEVEL":   "debug",
				"SCRY_DATABASE_URL":       "postgresql://user:pass@localhost:5432/testdb",
				"SCRY_AUTH_JWT_SECRET":    "thisisasecretkeythatis32charslong!!",
				"SCRY_LLM_GEMINI_API_KEY": "test-api-key",
			},
			expectError:    true,
			errorSubstring: "validation failed",
		},
		{
			name: "Invalid log level",
			envVars: map[string]string{
				"SCRY_SERVER_PORT":        "9090",
				"SCRY_SERVER_LOG_LEVEL":   "invalid-level", // Invalid log level
				"SCRY_DATABASE_URL":       "postgresql://user:pass@localhost:5432/testdb",
				"SCRY_AUTH_JWT_SECRET":    "thisisasecretkeythatis32charslong!!",
				"SCRY_LLM_GEMINI_API_KEY": "test-api-key",
			},
			expectError:    true,
			errorSubstring: "validation failed",
		},
		{
			name: "Short JWT secret",
			envVars: map[string]string{
				"SCRY_SERVER_PORT":        "9090",
				"SCRY_SERVER_LOG_LEVEL":   "debug",
				"SCRY_DATABASE_URL":       "postgresql://user:pass@localhost:5432/testdb",
				"SCRY_AUTH_JWT_SECRET":    "tooshort", // Too short JWT secret
				"SCRY_LLM_GEMINI_API_KEY": "test-api-key",
			},
			expectError:    true,
			errorSubstring: "validation failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup environment
			cleanup := testutils.SetupEnv(t, tc.envVars)
			defer cleanup()

			// Load configuration
			cfg, err := Load()

			// Verify
			if tc.expectError {
				assert.Error(t, err, "Load() should return an error with invalid configuration")
				if err != nil {
					assert.Contains(
						t,
						err.Error(),
						tc.errorSubstring,
						"Error message should contain expected substring",
					)
				}
				assert.Nil(t, cfg, "Config should be nil when an error occurs")
			} else {
				assert.NoError(t, err, "Load() should not return an error with valid configuration")
				assert.NotNil(t, cfg, "Load() should return a non-nil config")
			}
		})
	}
}
