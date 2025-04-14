// Package main implements the entry point for the Scry API server
// which handles users' spaced repetition flashcards and provides
// LLM integration for card generation.
package main

import (
	"fmt"
	"log"
	"log/slog" // Will be used in subsequent tasks
	"os"       // Will be used in subsequent tasks

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/logger" // Will be used in subsequent tasks
)

// main is the entry point for the scry-api server.
// It will be responsible for initializing configuration, setting up logging,
// establishing database connections, injecting dependencies, and starting the
// HTTP server.
func main() {
	// Suppress unused import warnings - these will be used in subsequent tasks
	_ = slog.LevelInfo
	_ = os.Stdout
	_ = logger.Setup

	fmt.Println("Scry API Server Starting...")

	// Call the core initialization logic
	_, err := initializeApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

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

	// Log configuration details
	fmt.Printf("Server configuration: Port=%d, LogLevel=%s\n",
		cfg.Server.Port, cfg.Server.LogLevel)

	// Future initialization steps would happen here
	// (logging, database, services, etc.)
	// - Setting up a proper logger using the configured log level
	// - Establishing database connection using Database.URL
	// - Configuring authentication with Auth.JWTSecret
	// - Initializing LLM client with LLM.GeminiAPIKey
	// - Injecting these dependencies into service layer
	// - Starting the HTTP server on the configured port

	return cfg, nil
}

// Another test comment
