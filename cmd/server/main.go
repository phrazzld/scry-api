package main

import (
	"fmt"
	"log"

	"github.com/phrazzld/scry-api/internal/config"
)

// main is the entry point for the scry-api server.
// It will be responsible for initializing configuration, setting up logging,
// establishing database connections, injecting dependencies, and starting the
// HTTP server.
func main() {
	fmt.Println("Scry API Server Starting...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Log configuration details
	fmt.Printf("Server configuration: Port=%d, LogLevel=%s\n", 
		cfg.Server.Port, cfg.Server.LogLevel)
	
	// TODO: Initialize application components with configuration
	// In the future, this will include:
	// - Setting up a proper logger using the configured log level
	// - Establishing database connection using Database.URL
	// - Configuring authentication with Auth.JWTSecret
	// - Initializing LLM client with LLM.GeminiAPIKey
	// - Injecting these dependencies into service layer
	// - Starting the HTTP server on the configured port
}