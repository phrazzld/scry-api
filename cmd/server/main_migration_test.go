//go:build integration

package main

import (
	"testing"

	"github.com/phrazzld/scry-api/internal/config"
)

func TestRunMigrationsInvalidCommand(t *testing.T) {
	// Create minimal config for the test
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: "postgres://user:password@localhost:5432/testdb",
		},
	}

	// Test that an invalid migration command returns an error
	err := runMigrations(cfg, "invalid_command")
	if err == nil {
		t.Fatal("Expected error for invalid migration command, got nil")
	}
}

func TestRunMigrationsInvalidConfigURL(t *testing.T) {
	// Create config with empty database URL
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: "",
		},
	}

	// Test that an empty database URL returns an error
	err := runMigrations(cfg, "up")
	if err == nil {
		t.Fatal("Expected error for empty database URL, got nil")
	}
}

func TestRunMigrationsCreateMissingName(t *testing.T) {
	// Create minimal config for the test
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: "postgres://user:password@localhost:5432/testdb",
		},
	}

	// Test that a "create" command without a name returns an error
	err := runMigrations(cfg, "create", "")
	if err == nil {
		t.Fatal("Expected error for missing migration name, got nil")
	}
}

// TestRunMigrationsConnectionError tests the database connection error handling.
// This is a unit test that doesn't require a real database connection.
func TestRunMigrationsConnectionError(t *testing.T) {
	// Create config with invalid database URL
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: "postgres://invalid:invalid@nonexistenthost:5432/nonexistentdb",
		},
	}

	// Test that attempting to connect to a non-existent database returns an error
	err := runMigrations(cfg, "status")
	if err == nil {
		t.Fatal("Expected error for invalid database connection, got nil")
	}
}
