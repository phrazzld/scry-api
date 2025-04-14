// Package main implements the entry point for the Scry API server
// which handles users' spaced repetition flashcards and provides
// LLM integration for card generation.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/logger"
)

// main is the entry point for the scry-api server.
// It will be responsible for initializing configuration, setting up logging,
// establishing database connections, injecting dependencies, and starting the
// HTTP server.
func main() {
	// Note: This log message will use the default logger since our custom logger
	// isn't set up yet. It will appear as plain text, not JSON.
	slog.Info("Scry API Server starting...")

	// Call the core initialization logic
	cfg, err := initializeApp()
	if err != nil {
		// Use structured logging for the error
		slog.Error("Failed to initialize application",
			"error", err)
		os.Exit(1)
	}

	// Log successful initialization
	slog.Info("Scry API Server initialized successfully",
		"port", cfg.Server.Port)

	// Server would start here after initialization
	// This would be added in a future task
}

// initializeApp loads configuration and sets up application components.
// Returns the loaded config and any initialization error.
func initializeApp() (*config.Config, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set up structured logging using the configured log level
	_, err = logger.Setup(cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to set up logger: %w", err)
	}

	// Log configuration details using structured logging
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

	// Future initialization steps would happen here
	// (database, services, etc.)
	// - Establishing database connection using Database.URL
	// - Configuring authentication with Auth.JWTSecret
	// - Initializing LLM client with LLM.GeminiAPIKey
	// - Injecting these dependencies into service layer
	// - Starting the HTTP server on the configured port

	return cfg, nil
}

// Another test comment
