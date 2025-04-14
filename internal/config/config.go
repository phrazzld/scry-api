package config

// Config holds all application configuration.
// It organizes settings into logical groups for better maintainability.
type Config struct {
	Server   ServerConfig   `mapstructure:"server" validate:"required"`
	Database DatabaseConfig `mapstructure:"database" validate:"required"`
	Auth     AuthConfig     `mapstructure:"auth" validate:"required"`
	LLM      LLMConfig      `mapstructure:"llm" validate:"required"`
}

// ServerConfig contains all server-related configuration settings.
type ServerConfig struct {
	Port     int    `mapstructure:"port" validate:"required,gt=0,lt=65536"`
	LogLevel string `mapstructure:"log_level" validate:"required,oneof=debug info warn error fatal"`
	// Add other server settings as needed (e.g., timeouts)
}

// DatabaseConfig contains all database-related configuration settings.
type DatabaseConfig struct {
	URL string `mapstructure:"url" validate:"required,url"`
	// Add other DB settings as needed (e.g., pool size)
}

// AuthConfig contains all authentication and authorization settings.
type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret" validate:"required,min=32"`
	// Add other auth settings as needed (e.g., token expiry)
}

// LLMConfig contains all LLM integration related settings.
type LLMConfig struct {
	GeminiAPIKey string `mapstructure:"gemini_api_key" validate:"required"`
	// Add other LLM settings as needed (e.g., model name)
}
