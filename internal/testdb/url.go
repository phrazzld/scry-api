//go:build (integration || test_without_external_deps) && !exported_core_functions

// Package testdb provides utilities specifically for database testing.
package testdb

import (
	"log/slog"
	"strings"

	"github.com/phrazzld/scry-api/internal/ciutil"
)

// This file contains database URL handling and standardization utilities.

// GetTestDatabaseURL returns a database URL suitable for testing.
// It attempts to retrieve a URL from environment variables in a specific order.
// For CI environments, the URL is standardized using ciutil.GetTestDatabaseURL.
func GetTestDatabaseURL() string {
	// Detect environment type for logging and configuration
	inCI := isCIForDatabaseURL()

	// Get default logger with environment context
	logger := slog.Default().With(
		slog.String("function", "testdb.GetTestDatabaseURL"),
		slog.Bool("ci_environment", inCI),
	)

	// Use the standardized implementation from ciutil
	dbURL := ciutil.GetTestDatabaseURL(logger)

	// If no database URL is found and we're in CI, log an error
	if dbURL == "" && inCI {
		envVars := []string{ciutil.EnvDatabaseURL, ciutil.EnvScryTestDBURL, ciutil.EnvScryDatabaseURL}
		logger.Error("no database URL found in CI environment",
			slog.String("checked_variables", strings.Join(envVars, ", ")),
			slog.String("impact", "tests will fail"),
			slog.String("resolution", "set at least one database URL environment variable"),
		)
	}

	return dbURL
}

// Use the existing utility functions from env.go
// (they are already defined in env.go, so we don't redeclare them here)

// Helper function that uses isCIEnvironment from env.go
func isCIForDatabaseURL() bool {
	return isCIEnvironment()
}

// maskDatabaseURL masks sensitive information in a database URL for safe logging
// Format: postgres://username:password@hostname:port/database?parameters
func maskDatabaseURL(dbURL string) string {
	// If empty or invalid format, return safely
	if dbURL == "" {
		return ""
	}

	// Use ciutil's masking function if available
	return ciutil.MaskSensitiveValue(dbURL)
}
