package config

import (
	"fmt"
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
	// Initialize a new viper instance
	v := viper.New()

	// --- Set default values ---
	// These defaults are used if the setting is not found in any other source
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.log_level", "info")
	v.SetDefault("auth.bcrypt_cost", 10) // Default bcrypt cost (same as bcrypt.DefaultCost)

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
	v.SetEnvPrefix("SCRY")                             // all env vars must be prefixed with SCRY_
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // maps nested config keys to env vars with underscores
	v.AutomaticEnv()                                   // read in environment variables that match

	// Explicitly bind critical environment variables to ensure they are properly mapped
	// This provides more reliable binding for essential configuration values
	bindEnvs := []struct {
		key    string // config key in dot notation (e.g., "server.port")
		envVar string // environment variable name (e.g., "SCRY_SERVER_PORT")
	}{
		{"database.url", "SCRY_DATABASE_URL"},
		{"auth.jwt_secret", "SCRY_AUTH_JWT_SECRET"},
		{"auth.bcrypt_cost", "SCRY_AUTH_BCRYPT_COST"},
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
