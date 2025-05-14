package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Load loads application configuration from multiple sources and returns a validated Config struct.
//
// The function follows this order of precedence for configuration values:
//  1. Environment variables with the "SCRY_" prefix (highest priority)
//  2. Values from config.yaml file (if present)
//  3. Default values (lowest priority)
//
// Environment variables are automatically mapped from nested config values using underscores.
// For example, the config field Config.Server.Port maps to the env var SCRY_SERVER_PORT.
//
// Default values are set for non-critical settings:
//   - server.port: 8080
//   - server.log_level: "info"
//
// Required values that must be provided (no defaults):
//   - database.url: PostgreSQL connection string
//   - auth.jwt_secret: JWT signing key (32+ characters)
//   - llm.gemini_api_key: Google Gemini API key
//
// The function performs validation on the loaded configuration to ensure:
//   - All required fields are present
//   - Values meet validation rules (port ranges, string lengths, etc.)
//
// Returns:
//   - A pointer to the populated and validated Config struct
//   - An error if loading or validation fails, with context about the failure
func Load() (*Config, error) {
	return LoadWithLogger(nil)
}

// LoadWithLogger loads configuration with the provided logger for detailed logging.
// This is the same as Load() but allows for logging configuration loading details.
// See Load() for full documentation.
func LoadWithLogger(logger *slog.Logger) (*Config, error) {
	// Initialize a new viper instance
	v := viper.New()

	// --- Set default values ---
	// These defaults are used if the setting is not found in any other source
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.log_level", "info")
	v.SetDefault(
		"auth.bcrypt_cost",
		10,
	) // Default bcrypt cost (same as bcrypt.DefaultCost)
	v.SetDefault(
		"auth.token_lifetime_minutes",
		60,
	) // Default access token lifetime (1 hour)
	v.SetDefault(
		"auth.refresh_token_lifetime_minutes",
		10080,
	) // Default refresh token lifetime (7 days)
	v.SetDefault("llm.model_name", "gemini-2.0-flash") // Default Gemini model
	v.SetDefault(
		"llm.max_retries",
		3,
	) // Default number of retries for transient errors
	v.SetDefault("llm.retry_delay_seconds", 2) // Default base delay between retries
	v.SetDefault("task.worker_count", 2)       // Default worker count
	v.SetDefault("task.queue_size", 100)       // Default queue size
	v.SetDefault(
		"task.stuck_task_age_minutes",
		30,
	) // Default stuck task age (30 minutes)

	// --- Configure config file (optional, for local dev) ---
	// Looks for config.yaml in the working directory
	v.SetConfigName("config") // name of config file (without extension)
	v.SetConfigType("yaml")   // specifies the format of the config file
	v.AddConfigPath(".")      // look for config in the working directory

	// Attempt to read the config file
	// Only ignore "file not found" errors, as the config file is optional
	// Log other errors as warnings since they might indicate permission issues or malformed YAML
	if err := v.ReadInConfig(); err != nil {
		// Check if the error is a ConfigFileNotFoundError, which we can safely ignore
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// This is not a "file not found" error, so it might be important
			// In a production app, we'd log this as a warning
			fmt.Printf("Warning: error reading config file: %v\n", err)
		}
		// Continue loading from other sources regardless of error type
	}

	// --- Configure environment variables ---
	// Environment variables take precedence over config file values
	v.SetEnvPrefix("SCRY") // all env vars must be prefixed with SCRY_
	v.SetEnvKeyReplacer(
		strings.NewReplacer(".", "_"),
	) // maps nested config keys to env vars with underscores
	v.AutomaticEnv() // read in environment variables that match

	// Explicitly bind critical environment variables to ensure they are properly mapped
	// This provides more reliable binding for essential configuration values
	bindEnvs := []struct {
		key            string   // config key in dot notation (e.g., "server.port")
		envVar         string   // primary environment variable (e.g., "SCRY_SERVER_PORT")
		legacyEnvVars  []string // legacy environment variables for backward compatibility
	}{
		{"database.url", "SCRY_DATABASE_URL", []string{"DATABASE_URL"}},
		{"auth.jwt_secret", "SCRY_AUTH_JWT_SECRET", nil},
		{"auth.bcrypt_cost", "SCRY_AUTH_BCRYPT_COST", nil},
		{"auth.token_lifetime_minutes", "SCRY_AUTH_TOKEN_LIFETIME_MINUTES", nil},
		{"auth.refresh_token_lifetime_minutes", "SCRY_AUTH_REFRESH_TOKEN_LIFETIME_MINUTES", nil},
		{"llm.gemini_api_key", "SCRY_LLM_GEMINI_API_KEY", nil},
		{"llm.model_name", "SCRY_LLM_MODEL_NAME", nil},
		{"llm.prompt_template_path", "SCRY_LLM_PROMPT_TEMPLATE_PATH", nil},
		{"llm.max_retries", "SCRY_LLM_MAX_RETRIES", nil},
		{"llm.retry_delay_seconds", "SCRY_LLM_RETRY_DELAY_SECONDS", nil},
		{"server.port", "SCRY_SERVER_PORT", nil},
		{"server.log_level", "SCRY_SERVER_LOG_LEVEL", []string{"LOG_LEVEL"}},
		{"task.worker_count", "SCRY_TASK_WORKER_COUNT", nil},
		{"task.queue_size", "SCRY_TASK_QUEUE_SIZE", nil},
		{"task.stuck_task_age_minutes", "SCRY_TASK_STUCK_TASK_AGE_MINUTES", nil},
	}

	// Bind environment variables with support for legacy variable names
	for _, env := range bindEnvs {
		// First bind the primary standardized environment variable
		err := v.BindEnv(env.key, env.envVar)
		if err != nil {
			return nil, fmt.Errorf("error binding environment variable %s: %w", env.envVar, err)
		}

		// If this variable has legacy alternatives, check for their existence
		// and log appropriate deprecation warnings
		if len(env.legacyEnvVars) > 0 && os.Getenv(env.envVar) == "" {
			for _, legacyEnvVar := range env.legacyEnvVars {
				legacyValue := os.Getenv(legacyEnvVar)
				if legacyValue != "" {
					// Set the standardized env var to maintain consistency
					os.Setenv(env.envVar, legacyValue)

					// Log deprecation warning if logger is provided
					if logger != nil {
						logger.Warn("Using legacy environment variable",
							"legacy_var", legacyEnvVar,
							"preferred_var", env.envVar,
							"config_key", env.key,
						)
					} else {
						fmt.Printf("Warning: Using legacy environment variable %s. "+
							"Please use %s instead.\n", legacyEnvVar, env.envVar)
					}

					break
				}
			}
		}
	}

	// Log which environment variables are set (at debug level)
	if logger != nil {
		for _, env := range bindEnvs {
			if v.Get(env.key) != nil {
				// Mask sensitive values in logs
				value := fmt.Sprintf("%v", v.Get(env.key))
				if strings.Contains(env.key, "secret") ||
				   strings.Contains(env.key, "password") ||
				   strings.Contains(env.key, "key") ||
				   strings.Contains(env.key, "url") {
					// Show only first and last few characters
					if len(value) > 8 {
						value = value[:4] + "****" + value[len(value)-4:]
					} else {
						value = "****" // Very short sensitive value
					}
				}

				logger.Debug("Configuration value loaded",
					"key", env.key,
					"env_var", env.envVar,
					"value", value,
					"source", v.GetString(fmt.Sprintf("source.%s", env.key)),
				)
			}
		}
	}

	// --- Unmarshal configuration into the Config struct ---
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// --- Validate the configuration values against defined rules ---
	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}
