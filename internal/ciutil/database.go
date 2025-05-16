package ciutil

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
)

const (
	// StandardCIUser is the standard username used in CI environments
	StandardCIUser = "postgres"

	// StandardCIPassword is the standard password used in CI environments
	StandardCIPassword = "postgres"

	// StandardCIHost is the standard host used in CI environments
	StandardCIHost = "localhost"

	// StandardCIPort is the standard port used in CI environments
	StandardCIPort = "5432"

	// StandardCIDatabase is the standard database name used in CI environments
	StandardCIDatabase = "scry_test"

	// StandardCIOptions contains standard connection options for CI environments
	StandardCIOptions = "sslmode=disable"

	// MigrationTableName is the name of the database table that tracks migrations
	MigrationTableName = "schema_migrations"
)

// GetTestDatabaseURL returns a database URL for testing purposes.
// It checks several environment variables in the following order:
// 1. DATABASE_URL (standard, non-prefixed)
// 2. SCRY_TEST_DB_URL (preferred, standardized name)
// 3. SCRY_DATABASE_URL (fallback)
//
// If running in a CI environment, it standardizes the URL to use postgres:postgres credentials.
// If no environment variables are set, it returns an empty string.
func GetTestDatabaseURL(logger *slog.Logger) string {
	envVars := []string{EnvDatabaseURL, EnvScryTestDBURL, EnvScryDatabaseURL}

	// Check if we have any database URL set
	var dbURL string

	for i, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			dbURL = val

			// Log a deprecation warning if not using the standardized name
			if i != 1 && logger != nil { // EnvScryTestDBURL is at index 1
				logger.Warn("Using non-standardized database URL environment variable",
					"used_var", envVar,
					"preferred_var", EnvScryTestDBURL,
					"value", MaskSensitiveValue(val),
				)
			} else if logger != nil {
				logger.Info("Using database URL from environment variable",
					"var", envVar,
					"value", MaskSensitiveValue(val),
				)
			}

			break
		}
	}

	// If no database URL is set, return empty string
	if dbURL == "" {
		if logger != nil {
			logger.Info("No database URL environment variables found")
		}
		return ""
	}

	// If running in CI, standardize the URL
	if IsCI() {
		standardizedURL, err := standardizeDatabaseURL(dbURL, logger)
		if err != nil {
			if logger != nil {
				logger.Error("Failed to standardize database URL",
					"error", err,
					"original_url", MaskSensitiveValue(dbURL),
				)
			}
			return dbURL
		}

		if standardizedURL != dbURL {
			if logger != nil {
				logger.Info("Standardized database URL for CI environment",
					"original", MaskSensitiveValue(dbURL),
					"standardized", MaskSensitiveValue(standardizedURL),
				)
			}

			// Update environment variables for consistency
			updateDatabaseEnvironmentVariables(standardizedURL, logger)

			return standardizedURL
		}
	}

	return dbURL
}

// standardizeDatabaseURL ensures the database URL uses standard credentials in CI environments.
// It parses the URL, replaces username and password with 'postgres', and returns the standardized URL.
func standardizeDatabaseURL(dbURL string, logger *slog.Logger) (string, error) {
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Check if this is a postgres URL
	if parsedURL.Scheme != "postgres" {
		if logger != nil {
			logger.Warn("Non-postgres database URL detected",
				"scheme", parsedURL.Scheme,
			)
		}
		return dbURL, nil
	}

	// Extract user info
	username := ""
	password := ""

	if parsedURL.User != nil {
		username = parsedURL.User.Username()
		password, _ = parsedURL.User.Password()
	}

	// Check if standardization is needed
	if username == StandardCIUser && password == StandardCIPassword {
		return dbURL, nil // Already using standard credentials
	}

	// Standardize the URL for CI
	standardizedURL := *parsedURL
	standardizedURL.User = url.UserPassword(StandardCIUser, StandardCIPassword)

	// In CI environments, also standardize host and database if not explicitly set
	if IsCI() {
		host := parsedURL.Hostname()
		port := parsedURL.Port()

		if host == "" || host == "localhost" || host == "127.0.0.1" {
			// Keep the local hostname but ensure port is standard
			if port == "" {
				hostPort := fmt.Sprintf("%s:%s", host, StandardCIPort)
				standardizedURL.Host = hostPort
			}
		}

		// Extract path (database name) and standardize if empty
		path := strings.TrimPrefix(parsedURL.Path, "/")
		if path == "" {
			standardizedURL.Path = "/" + StandardCIDatabase
		}

		// Add standard options if none are provided
		if parsedURL.RawQuery == "" {
			standardizedURL.RawQuery = StandardCIOptions
		}
	}

	return standardizedURL.String(), nil
}

// updateDatabaseEnvironmentVariables updates all database-related environment variables
// to maintain consistency across the application.
func updateDatabaseEnvironmentVariables(standardizedURL string, logger *slog.Logger) {
	// List of environment variables to update
	envVars := []string{EnvDatabaseURL, EnvScryTestDBURL, EnvScryDatabaseURL}

	for _, envVar := range envVars {
		if os.Getenv(envVar) != "" {
			if logger != nil {
				logger.Debug("Updating environment variable",
					"var", envVar,
					"value", MaskSensitiveValue(standardizedURL),
				)
			}
			os.Setenv(envVar, standardizedURL)
		}
	}
}
