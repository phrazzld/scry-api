package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Load configuration from environment variables and optionally config files.
// Environment variables take precedence over values from config files.
// Returns a populated Config struct or an error if loading/validation fails.
func Load() (*Config, error) {
	// Initialize a new viper instance
	v := viper.New()

	// --- Set default values ---
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.log_level", "info")

	// --- Configure config file (optional, for local dev) ---
	v.SetConfigName("config") // name of config file (without extension)
	v.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	v.AddConfigPath(".")      // look for config in the working directory

	// Attempt to read the config file but ignore errors if it doesn't exist
	_ = v.ReadInConfig() // Find and read the config file, ignore file not found error

	// --- Configure environment variables ---
	v.SetEnvPrefix("SCRY")                             // e.g., SCRY_SERVER_PORT=8081
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // Replace dots with underscores (e.g., server.port -> SERVER_PORT)
	v.AutomaticEnv()                                   // Read in environment variables that match

	// Explicitly bind environment variables for critical configuration
	// This ensures they are properly identified and bound
	bindEnvs := []struct {
		key      string
		envVar   string
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

	// --- Unmarshal and validate ---
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		// Provide more context on validation errors if possible
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}
