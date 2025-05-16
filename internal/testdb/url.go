//go:build (integration || test_without_external_deps) && !exported_core_functions

// Package testdb provides utilities specifically for database testing.
package testdb

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// This file contains database URL handling and standardization utilities.

// GetTestDatabaseURL returns a database URL suitable for testing.
// It attempts to retrieve a URL from environment variables in a specific order.
// For CI environments, specifically GitHub Actions, the URL is standardized:
// it enforces both username and password to be 'postgres'.
func GetTestDatabaseURL() string {
	// Detect environment type for logging and configuration
	inCI := isCIEnvironment()
	inGitHubActions := isGitHubActionsCI()

	// Get default logger with environment context
	logger := slog.Default().With(
		slog.String("function", "GetTestDatabaseURL"),
		slog.Bool("ci_environment", inCI),
		slog.Bool("github_actions", inGitHubActions),
	)

	// Check environment variables in priority order
	envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}

	// Log environment variables in CI for diagnostics
	if inCI {
		// Collect environment variable values for structured logging
		envValues := make(map[string]string)
		for _, envVar := range envVars {
			value := os.Getenv(envVar)
			if value != "" {
				envValues[envVar] = maskDatabaseURL(value)
			} else {
				envValues[envVar] = "<not set>"
			}
		}

		// Log all environment variables in a single structured log entry
		logger.Debug("checking database URL environment variables",
			slog.Any("environment_variables", envValues),
		)
	}

	// Search for a valid database URL in the priority order
	for _, envVar := range envVars {
		dbURL := os.Getenv(envVar)
		if dbURL == "" {
			continue // Skip empty environment variables
		}

		// Found a database URL
		logger.Info("found database URL",
			slog.String("source", envVar),
			slog.String("url", maskDatabaseURL(dbURL)),
		)

		// If not in CI, return the URL as-is
		if !inCI {
			return dbURL
		}

		// CI environment handling - standardize the database URL
		standardizedURL, err := standardizeDatabaseURL(dbURL, inGitHubActions, logger)
		if err != nil {
			logger.Error("failed to standardize database URL",
				slog.String("url", maskDatabaseURL(dbURL)),
				slog.String("error", err.Error()),
			)

			// For GitHub Actions, return a fallback URL if standardization fails
			if inGitHubActions {
				fallbackURL := "postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable"
				logger.Warn("using fallback database URL for GitHub Actions",
					slog.String("fallback_url", maskDatabaseURL(fallbackURL)),
				)

				// Update all environment variables with the fallback URL
				updateEnvironmentVariables(envVars, fallbackURL, logger)
				return fallbackURL
			}

			// For other CI environments, return the original URL if we can't standardize
			return dbURL
		}

		// If URL was successfully standardized
		if standardizedURL != dbURL {
			logger.Info("standardized database URL for CI",
				slog.String("original_url", maskDatabaseURL(dbURL)),
				slog.String("standardized_url", maskDatabaseURL(standardizedURL)),
			)

			// Update all environment variables with the standardized URL
			updateEnvironmentVariables(envVars, standardizedURL, logger)
		}

		return standardizedURL
	}

	// No valid URL found
	if inCI {
		logger.Error("no database URL found in CI environment",
			slog.String("checked_variables", strings.Join(envVars, ", ")),
			slog.String("impact", "tests will fail"),
			slog.String("resolution", "set at least one database URL environment variable"),
		)
	}
	return ""
}

// standardizeDatabaseURL ensures the database URL uses the correct credentials for CI.
// For GitHub Actions, it enforces 'postgres' as both username and password.
// For other CI environments, it ensures 'postgres' is used as the username at minimum.
func standardizeDatabaseURL(dbURL string, isGitHubActions bool, logger *slog.Logger) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Check if URL contains user info
	if parsedURL.User == nil {
		// Add default user info if none is present
		parsedURL.User = url.UserPassword("postgres", "postgres")
		logger.Debug("adding default postgres credentials to URL with no user info")
		return parsedURL.String(), nil
	}

	// Get current username and password
	username := parsedURL.User.Username()
	password, passwordSet := parsedURL.User.Password()

	// Log detected credentials (masking the password)
	logger.Debug("detected database credentials",
		slog.String("username", username),
		slog.Bool("password_set", passwordSet),
	)

	// Determine if standardization is needed
	needsUpdate := false

	// For GitHub Actions, standardize both username and password to 'postgres'
	if isGitHubActions {
		if username != "postgres" || (passwordSet && password != "postgres") {
			parsedURL.User = url.UserPassword("postgres", "postgres")
			logger.Debug("standardizing GitHub Actions credentials",
				slog.String("username", "postgres"),
				slog.String("password", "****"),
			)
			needsUpdate = true
		}
	} else if username != "postgres" {
		// For other CI environments, only standardize the username
		parsedURL.User = url.UserPassword("postgres", password)
		logger.Debug("standardizing CI username only",
			slog.String("username", "postgres"),
		)
		needsUpdate = true
	}

	// Return standardized URL if updated, or original URL if no update needed
	if needsUpdate {
		return parsedURL.String(), nil
	}
	return dbURL, nil
}

// updateEnvironmentVariables updates all database-related environment variables
// with the standardized URL for consistency across the application.
func updateEnvironmentVariables(envVars []string, standardizedURL string, logger *slog.Logger) {
	for _, envVar := range envVars {
		oldValue := os.Getenv(envVar)
		if oldValue == "" {
			continue // Skip unset variables
		}

		// Only update and log if we're actually changing something
		if oldValue != standardizedURL {
			logger.Debug("updating environment variable",
				slog.String("variable", envVar),
				slog.String("old_value", maskDatabaseURL(oldValue)),
				slog.String("new_value", maskDatabaseURL(standardizedURL)),
			)

			if err := os.Setenv(envVar, standardizedURL); err != nil {
				logger.Error("failed to set environment variable",
					slog.String("variable", envVar),
					slog.String("error", err.Error()),
				)
			}
		}
	}
}

// maskDatabaseURL masks sensitive information in a database URL for safe logging
// Format: postgres://username:password@hostname:port/database?parameters
func maskDatabaseURL(dbURL string) string {
	// If empty or invalid format, return safely
	if dbURL == "" {
		return ""
	}

	// Try regex matching first for consistent output format
	re := regexp.MustCompile(`://([^:]+):([^@]+)@`)
	if re.MatchString(dbURL) {
		return re.ReplaceAllString(dbURL, "://$1:****@")
	}

	// Fall back to URL parsing if regex doesn't match
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		// If both approaches fail, return a generic masked version
		return "database-url-with-masked-credentials"
	}

	// For properly parsed URLs, mask the password
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		parsedURL.User = url.UserPassword(username, "****")
		return parsedURL.String()
	}

	// If no user info is found, return the original URL
	return dbURL
}
