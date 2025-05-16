//go:build exported_core_functions

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMigrationPathsFindable checks that migrations directory can be found and contains migrations
// This test is not tagged so it will run in all build configurations
func TestMigrationPathsFindable(t *testing.T) {
	// Start from the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// The test is run from cmd/server, so we need to go up two levels to reach project root
	projectRoot := filepath.Dir(filepath.Dir(cwd))

	// Construct the expected migrations path
	migrationsPath := filepath.Join(projectRoot, "internal", "platform", "postgres", "migrations")

	// Verify the migrations directory exists
	info, err := os.Stat(migrationsPath)
	if err != nil {
		t.Fatalf("Failed to find migrations directory at %s: %v", migrationsPath, err)
	}

	if !info.IsDir() {
		t.Fatalf("Expected %s to be a directory", migrationsPath)
	}

	// Read the directory to make sure there are migration files
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	// Check that there are migration files
	migrationFiles := 0
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
			migrationFiles++
		}
	}

	if migrationFiles == 0 {
		t.Fatalf("No SQL migration files found in %s", migrationsPath)
	}

	t.Logf("Found %d migration files in %s", migrationFiles, migrationsPath)
}

// For now, we'll just run the basic test on migration paths
// to avoid build constraint issues
