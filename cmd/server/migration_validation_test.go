package main

import (
	"database/sql"
	"os"
	"testing"
)

// TestMigrationValidation tests the migration validation functionality
func TestMigrationValidation(t *testing.T) {
	// Skip if no database URL is set
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try alternate environment variables
		if altURL := os.Getenv("SCRY_TEST_DB_URL"); altURL != "" {
			dbURL = altURL
		} else if altURL := os.Getenv("SCRY_DATABASE_URL"); altURL != "" {
			dbURL = altURL
		} else {
			t.Skip("Skipping database validation test - no database URL provided")
		}
	}

	// Create a test database connection
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close database connection: %v", err)
		}
	}()

	// Test if verifyAppliedMigrations returns nil when migrations are properly applied
	// This test assumes migrations have been applied to the test database
	// and only checks that the function executes without error
	err = verifyAppliedMigrations(db, nil)
	if err != nil {
		t.Fatalf("verifyAppliedMigrations failed on properly migrated database: %v", err)
	}

	// Test that getMigrationsPath returns a valid path
	path, err := getMigrationsPath()
	if err != nil {
		t.Fatalf("getMigrationsPath failed: %v", err)
	}
	if path == "" {
		t.Fatal("getMigrationsPath returned empty path")
	}
	if !directoryExists(path) {
		t.Fatalf("getMigrationsPath returned path that doesn't exist: %s", path)
	}
	t.Logf("Found migrations path: %s", path)

	// Verify we can enumerate migration files
	migData, err := enumerateMigrationFiles(path)
	if err != nil {
		t.Fatalf("Failed to enumerate migration files: %v", err)
	}
	if migData.SQLCount == 0 {
		t.Fatalf("No SQL migration files found in %s", path)
	}
	t.Logf("Found %d SQL migration files", migData.SQLCount)
	t.Logf("Latest version: %s", migData.LatestVersion)
}
