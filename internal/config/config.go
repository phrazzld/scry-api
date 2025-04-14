package config

// Config holds all application configuration organized into logical groups.
// It is loaded from environment variables and/or configuration files
// using the Load() function and validated to ensure required values are present.
// Each field group is represented by a nested configuration struct with its own
// validation rules using the "validate" tag.
//
// The "mapstructure" tags define how the configuration values are mapped from
// various sources like environment variables (with the SCRY_ prefix) and configuration
// files in YAML format.
type Config struct {
	// Server contains HTTP server settings like port and logging level
	Server ServerConfig `mapstructure:"server" validate:"required"`

	// Database contains database connection settings
	Database DatabaseConfig `mapstructure:"database" validate:"required"`

	// Auth contains authentication and authorization settings
	Auth AuthConfig `mapstructure:"auth" validate:"required"`

	// LLM contains language model integration settings
	LLM LLMConfig `mapstructure:"llm" validate:"required"`
}

// ServerConfig defines server-related settings for the HTTP API.
// These settings control the behavior of the web server that handles
// client requests.
type ServerConfig struct {
	// Port specifies the TCP port number the HTTP server will listen on.
	// Valid values are between 1 and 65535, with common HTTP ports being
	// 8080, 3000, etc. Default is 8080 if not specified.
	Port int `mapstructure:"port" validate:"required,gt=0,lt=65536"`

	// LogLevel controls the verbosity of application logging.
	// Accepts "debug", "info", "warn", "error", or "fatal" in order
	// of increasing severity. Default is "info" if not specified.
	LogLevel string `mapstructure:"log_level" validate:"required,oneof=debug info warn error fatal"`
	// Add other server settings as needed (e.g., timeouts, middleware configs)
}

// DatabaseConfig defines settings related to the database connection.
// These settings determine how the application connects to the PostgreSQL database.
type DatabaseConfig struct {
	// URL is the PostgreSQL database connection string.
	// Format: postgres://username:password@host:port/database
	// Required for establishing a connection to the database.
	URL string `mapstructure:"url" validate:"required,url"`
	// Add other DB settings as needed (e.g., max connections, timeout, retry policy)
}

// AuthConfig defines authentication and authorization settings.
// These settings are used for securing API endpoints and managing user authentication.
type AuthConfig struct {
	// JWTSecret is the secret key used to sign and verify JWT tokens.
	// Must be at least 32 characters long to ensure adequate security.
	// This value should be kept secret and never committed to source control.
	JWTSecret string `mapstructure:"jwt_secret" validate:"required,min=32"`
	// Add other auth settings as needed (e.g., token expiry duration, refresh token settings)
}

// LLMConfig defines settings for Language Model integration.
// These settings control how the application interacts with LLM services.
type LLMConfig struct {
	// GeminiAPIKey is the API key for accessing Google's Gemini AI service.
	// Required for making requests to the Gemini API endpoints.
	// This value should be kept secret and never committed to source control.
	GeminiAPIKey string `mapstructure:"gemini_api_key" validate:"required"`
	// Add other LLM settings as needed (e.g., model name, request timeout, rate limiting)
}
