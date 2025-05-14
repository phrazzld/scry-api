//go:build exported_core_functions

package testdb

import (
	"log/slog"
	"net/url"
	"os"
	"strings"
)

// GetTestDatabaseURL returns the database URL to use for tests, with standardized credentials.
// This version is for use in production code and exports the functionality with exported_core_functions tag.
func GetTestDatabaseURL() string {
	// Start with the most basic logger
	logger := slog.Default().With("component", "database")
	
	// Check if we're in a CI environment (GitHub Actions, GitLab CI, etc.)
	inCI := isCIEnvironmentInternal()
	
	// Log environment detection for debugging
	logger.Info("Database environment detection", 
		"ci_environment", inCI)
	
	// Try to get database URL from environment variables in order of precedence
	candidateURLs := []string{
		os.Getenv("DATABASE_URL"),
		os.Getenv("SCRY_TEST_DB_URL"),
		os.Getenv("SCRY_DATABASE_URL"),
	}
	
	var dbURL string
	for _, candidate := range candidateURLs {
		if candidate != "" {
			dbURL = candidate
			break
		}
	}
	
	// If no URL found, return empty string - callers will need to handle this
	if dbURL == "" {
		logger.Warn("No database URL found in environment variables")
		return ""
	}
	
	// In CI environments, standardize database credentials to 'postgres'
	if inCI {
		logger.Info("Standardizing database URL for CI environment")
		
		// Parse the URL
		parsedURL, err := url.Parse(dbURL)
		if err != nil {
			logger.Error("Failed to parse database URL", "error", err)
			return dbURL // Return original URL on error
		}
		
		// Set standardized CI credentials
		parsedURL.User = url.UserPassword("postgres", "postgres")
		
		// Update the URL
		dbURL = parsedURL.String()
		
		// Log the standardized URL (with password masked)
		safeURL := strings.Replace(dbURL, "postgres:postgres", "postgres:****", 1)
		logger.Info("Using standardized database URL in CI", "url", safeURL)
	}
	
	return dbURL
}

// isCIEnvironmentInternal returns true if the code is running in a CI environment
// This is a separate function to avoid name conflicts with the one in project_root.go
func isCIEnvironmentInternal() bool {
	return os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("GITLAB_CI") != ""
}

// maskDatabaseURL masks the password in a database URL for safe logging
func maskDatabaseURL(dbURL string) string {
	// Parse the URL
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return "invalid-url"
	}
	
	// Mask the password if user info exists
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		parsedURL.User = url.UserPassword(username, "****")
		return parsedURL.String()
	}
	
	return dbURL
}