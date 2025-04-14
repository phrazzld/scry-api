package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupEnv sets up environment variables for testing and returns a cleanup function
func setupEnv(t *testing.T, envVars map[string]string) func() {
	// Save current environment values
	originalValues := make(map[string]string)
	for name := range envVars {
		originalValues[name] = os.Getenv(name)
	}

	// Set new environment variables
	for name, value := range envVars {
		err := os.Setenv(name, value)
		require.NoError(t, err, "Failed to set environment variable %s", name)
	}

	// Return cleanup function
	return func() {
		// Restore original environment
		for name, value := range originalValues {
			if value == "" {
				os.Unsetenv(name)
			} else {
				os.Setenv(name, value)
			}
		}
	}
}

// createTempConfigFile creates a temporary config.yaml file with the given content
func createTempConfigFile(t *testing.T, content string) (string, func()) {
	tempDir := t.TempDir()
	configPath := tempDir + "/config.yaml"

	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err, "Failed to create temporary config file")

	// Return the directory path and a cleanup function
	return tempDir, func() {
		// t.TempDir() handles cleanup automatically
	}
}

// TestSuccessfulInitialization verifies the application initializes correctly
// with valid configuration from environment variables
func TestSuccessfulInitialization(t *testing.T) {
	// Setup required environment variables
	cleanup := setupEnv(t, map[string]string{
		"SCRY_SERVER_PORT":        "9090",
		"SCRY_SERVER_LOG_LEVEL":   "debug",
		"SCRY_DATABASE_URL":       "postgresql://user:pass@localhost:5432/testdb",
		"SCRY_AUTH_JWT_SECRET":    "thisisasecretkeythatis32charslong!!",
		"SCRY_LLM_GEMINI_API_KEY": "test-api-key",
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
	assert.Equal(t, "postgresql://user:pass@localhost:5432/testdb", cfg.Database.URL, "Database URL should be loaded from environment variables")
	assert.Equal(t, "thisisasecretkeythatis32charslong!!", cfg.Auth.JWTSecret, "JWT secret should be loaded from environment variables")
	assert.Equal(t, "test-api-key", cfg.LLM.GeminiAPIKey, "Gemini API key should be loaded from environment variables")
}

// loadConfigWithFile wraps config.Load with the ability to specify a config file path
// This avoids the need to change the working directory in tests
func loadConfigWithFile(configPath string) (*config.Config, error) {
	// This function is nearly identical to config.Load but allows specifying a config file
	// It is exported for testing purposes only
	v := viper.New()

	// Set default values
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.log_level", "info")

	// Set the specific config file rather than looking in the working directory
	v.SetConfigType("yaml")
	if configPath != "" {
		v.SetConfigFile(configPath)
		
		// Attempt to read the config file
		if err := v.ReadInConfig(); err != nil {
			// Only log non-file-not-found errors (though this shouldn't happen with an explicit file)
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				fmt.Printf("Warning: error reading config file: %v\n", err)
			}
		}
	}

	// Configure environment variables
	v.SetEnvPrefix("SCRY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicitly bind critical environment variables
	bindEnvs := []struct {
		key    string
		envVar string
	}{
		{"database.url", "SCRY_DATABASE_URL"},
		{"auth.jwt_secret", "SCRY_AUTH_JWT_SECRET"},
		{"llm.gemini_api_key", "SCRY_LLM_GEMINI_API_KEY"},
		{"server.port", "SCRY_SERVER_PORT"},
		{"server.log_level", "SCRY_SERVER_LOG_LEVEL"},
	}

	for _, env := range bindEnvs {
		err := v.BindEnv(env.key, env.envVar)
		if err != nil {
			return nil, fmt.Errorf("error binding environment variable %s: %w", env.envVar, err)
		}
	}

	// Unmarshal and validate
	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// initializeAppWithConfigFile is a version of initializeApp that uses the specific config file
func initializeAppWithConfigFile(configPath string) (*config.Config, error) {
	// Load configuration with the specific file
	cfg, err := loadConfigWithFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Log configuration details
	fmt.Printf("Server configuration: Port=%d, LogLevel=%s\n",
		cfg.Server.Port, cfg.Server.LogLevel)

	return cfg, nil
}

// TestEnvironmentVariablePrecedence verifies that environment variables take precedence over config file values
func TestEnvironmentVariablePrecedence(t *testing.T) {
	// Create a temporary config file with one set of values
	configYaml := `
server:
  port: 7070
  log_level: info
database:
  url: postgresql://db_user:db_pass@db-host:5432/db
auth:
  jwt_secret: thisisasecretkeythatis32charslong!!
llm:
  gemini_api_key: config-file-api-key
`
	tempDir, cleanupFile := createTempConfigFile(t, configYaml)
	defer cleanupFile()
	
	configPath := tempDir + "/config.yaml"

	// Setup environment variables with different values
	// The environment variables should take precedence over the config file
	envCleanup := setupEnv(t, map[string]string{
		"SCRY_SERVER_PORT":        "9090", // Different from config.yaml
		"SCRY_DATABASE_URL":       "postgresql://user:pass@localhost:5432/testdb",
		"SCRY_AUTH_JWT_SECRET":    "thisisasecretkeythatis32charslong!!",
		"SCRY_LLM_GEMINI_API_KEY": "test-api-key",
		// Deliberately not setting SCRY_SERVER_LOG_LEVEL to test config file value
	})
	defer envCleanup()

	// Initialize the application with the specific config file
	cfg, err := initializeAppWithConfigFile(configPath)

	// Verify initialization succeeded
	require.NoError(t, err, "Application initialization should succeed")
	require.NotNil(t, cfg, "Configuration should not be nil")

	// Verify environment variable took precedence
	assert.Equal(t, 9090, cfg.Server.Port, "Server port should come from environment variable (precedence)")
	// Verify config file value was used when env var not set
	assert.Equal(t, "info", cfg.Server.LogLevel, "Log level should come from config file when env var not set")
}

// TestInvalidConfiguration tests initialization with invalid configuration
func TestInvalidConfiguration(t *testing.T) {
	testCases := []struct {
		name      string
		envVars   map[string]string
		errorText string
	}{
		{
			name: "Missing required fields",
			envVars: map[string]string{
				"SCRY_SERVER_PORT":      "9090",
				"SCRY_SERVER_LOG_LEVEL": "debug",
				// Missing Database URL, JWT Secret, and Gemini API Key
			},
			errorText: "validation failed",
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
			errorText: "validation failed",
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
			errorText: "validation failed",
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
			errorText: "validation failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := setupEnv(t, tc.envVars)
			defer cleanup()

			// Initialize the application
			cfg, err := initializeApp()

			// Verify initialization failed as expected
			assert.Error(t, err, "Application initialization should fail with invalid config")
			assert.Nil(t, cfg, "Configuration should be nil on error")
			assert.Contains(t, err.Error(), tc.errorText, "Error message should contain expected text")
		})
	}
}
