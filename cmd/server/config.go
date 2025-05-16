//go:build exported_core_functions

package main

import (
	"fmt"

	"log/slog"

	"github.com/phrazzld/scry-api/internal/config"
)

// loadAppConfig loads the application configuration from environment variables or config file.
// Returns the loaded config and any loading error.
func loadAppConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Log basic configuration details after successful loading
	slog.Info("Server configuration loaded",
		"port", cfg.Server.Port,
		"log_level", cfg.Server.LogLevel)

	// Log additional configuration details at debug level if available
	if cfg.Database.URL != "" {
		slog.Debug("Database configuration", "url_present", true)
	}
	if cfg.Auth.JWTSecret != "" {
		slog.Debug("Auth configuration", "jwt_secret_present", true)
	}

	return cfg, nil
}
