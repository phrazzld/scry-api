// Package config provides configuration management for the Scry API.
//
// This package is responsible for loading, parsing, validating, and providing
// strongly-typed access to application configuration from multiple sources.
// It follows clean architecture principles by isolating configuration details
// from business logic.
//
// Configuration Sources:
//
// The configuration is loaded from multiple sources with the following precedence:
//   1. Environment variables with the "SCRY_" prefix (highest priority)
//   2. Configuration file (config.yaml in the working directory, if present)
//   3. Default values (for non-critical settings, lowest priority)
//
// Key Features:
//
//   - Strongly-typed configuration model using Go structs
//   - Validation using go-playground/validator tags
//   - Flexible configuration via environment variables or YAML files
//   - Clear error messages for missing or invalid configuration
//   - Sensible defaults for non-critical settings
//
// Usage:
//
// To load configuration in the application:
//
//	cfg, err := config.Load()
//	if err != nil {
//	    log.Fatalf("Failed to load configuration: %v", err)
//	}
//	// Use configuration values
//	fmt.Printf("Starting server on port %d\n", cfg.Server.Port)
//
// For local development, create a config.yaml file in the working directory
// or set environment variables with the SCRY_ prefix. See .env.example and
// config.yaml.example for examples.
//
// Extending Configuration:
//
// To add new configuration options:
//   1. Add fields to the appropriate struct in config.go
//   2. Add validation tags as needed
//   3. Update documentation
//   4. If needed, set defaults in the Load() function
//
// All configuration is accessed through the Config struct, which is injected
// into components that need configuration values.
package config