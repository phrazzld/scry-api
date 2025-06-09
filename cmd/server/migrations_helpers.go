package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
)

// MigrationTableName is the name of the table used by goose to track migrations.
const MigrationTableName = "schema_migrations"

// GetTestDatabaseURL returns a standardized database URL for testing,
// with proper environment variable precedence and CI environment handling.
func GetTestDatabaseURL() string {
	// Check for explicit DATABASE_URL first
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		slog.Info("using DATABASE_URL from environment",
			"source", "DATABASE_URL",
			"url", maskPassword(dbURL))

		// In CI, always standardize to 'postgres' user
		if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
			standardized := standardizeCIDatabaseURL(dbURL)
			slog.Info("standardized CI database URL",
				"original", maskPassword(dbURL),
				"standardized", maskPassword(standardized))
			return standardized
		}

		return dbURL
	}

	// Check for test-specific database URL
	if testDBURL := os.Getenv("SCRY_TEST_DB_URL"); testDBURL != "" {
		slog.Info("using SCRY_TEST_DB_URL from environment",
			"source", "SCRY_TEST_DB_URL",
			"url", maskPassword(testDBURL))
		return testDBURL
	}

	// Check if we're in CI
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		// CI-specific configuration
		ciURL := "postgres://postgres:postgres@localhost:5432/scry_api_test?sslmode=disable"
		slog.Info("using CI database configuration",
			"source", "CI defaults",
			"url", maskPassword(ciURL))
		return ciURL
	}

	// Default for local development
	defaultURL := "postgres://testuser:testpass@localhost:5432/scry_api_test?sslmode=disable"
	slog.Info("using default test database URL",
		"source", "defaults",
		"url", maskPassword(defaultURL))
	return defaultURL
}

// standardizeCIDatabaseURL ensures the database URL uses 'postgres' as both username and password in CI
func standardizeCIDatabaseURL(dbURL string) string {
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		slog.Error("failed to parse DATABASE_URL in CI, using as-is",
			"error", err,
			"url", maskPassword(dbURL))
		return dbURL
	}

	// Force postgres:postgres in CI
	parsedURL.User = url.UserPassword("postgres", "postgres")

	return parsedURL.String()
}

// maskPassword masks the password in a database URL for logging
func maskPassword(dbURL string) string {
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return dbURL
	}

	if _, hasPassword := parsedURL.User.Password(); hasPassword {
		parsedURL.User = url.UserPassword(parsedURL.User.Username(), "****")
	}

	return parsedURL.String()
}

// FindMigrationsDir attempts to locate the migrations directory relative to the project root.
func FindMigrationsDir() (string, error) {
	projectRoot, err := FindProjectRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}

	migrationsPath := filepath.Join(projectRoot, "internal", "platform", "postgres", "migrations")

	// Verify the migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		return "", fmt.Errorf("migrations directory not found at %s", migrationsPath)
	}

	return migrationsPath, nil
}

// FindProjectRoot locates the project root directory by looking for marker files.
func FindProjectRoot() (string, error) {
	// Check CI environment variables first
	if ci := os.Getenv("CI"); ci == "true" {
		if githubWorkspace := os.Getenv("GITHUB_WORKSPACE"); githubWorkspace != "" {
			return filepath.Clean(githubWorkspace), nil
		}
		if ciProjectDir := os.Getenv("CI_PROJECT_DIR"); ciProjectDir != "" {
			return filepath.Clean(ciProjectDir), nil
		}
	}

	// Start from current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree looking for project root markers
	dir := currentDir
	for {
		// Check for go.mod (Go projects)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Also check for .git to ensure we're at the actual project root
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				return dir, nil
			}
			return dir, nil
		}

		// Check if we've reached the root directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("project root not found (no go.mod found in directory tree)")
}
