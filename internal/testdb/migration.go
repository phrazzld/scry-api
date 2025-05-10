//go:build integration || test_without_external_deps

package testdb

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindMigrationsDir locates the migrations directory based on the project root.
// This is a helper function for applying migrations.
func FindMigrationsDir() (string, error) {
	// Get the project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}

	// Build the migrations directory path
	migrationsDir := filepath.Join(projectRoot, "internal", "platform", "postgres", "migrations")

	// Verify the directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
	}

	return migrationsDir, nil
}
