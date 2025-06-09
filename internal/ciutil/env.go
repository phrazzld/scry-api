package ciutil

import (
	"os"
	"strings"

	"log/slog"
)

// Common environment variable names used across the codebase.
// These constants ensure consistent access and prevent typos.
const (
	// CI environment detection variables
	EnvCI               = "CI"
	EnvGitHubActions    = "GITHUB_ACTIONS"
	EnvGitHubWorkspace  = "GITHUB_WORKSPACE"
	EnvGitLabCI         = "GITLAB_CI"
	EnvGitLabProjectDir = "CI_PROJECT_DIR"
	EnvJenkinsURL       = "JENKINS_URL"
	EnvTravisCI         = "TRAVIS"
	EnvCircleCI         = "CIRCLECI"

	// Project-specific environment variables
	EnvScryProjectRoot = "SCRY_PROJECT_ROOT"

	// Database connection environment variables
	EnvDatabaseURL     = "DATABASE_URL"
	EnvScryTestDBURL   = "SCRY_TEST_DB_URL" // Preferred standardized name
	EnvScryDatabaseURL = "SCRY_DATABASE_URL"

	// Log settings
	EnvLogLevel  = "LOG_LEVEL"
	EnvLogFormat = "LOG_FORMAT"

	// Common default values
	DefaultLogLevel  = "info"
	DefaultLogFormat = "json"
)

// IsCI returns true if the current environment is a CI environment.
// It checks for common CI environment variables across different CI providers.
func IsCI() bool {
	return os.Getenv(EnvCI) != "" ||
		os.Getenv(EnvGitHubActions) != "" ||
		os.Getenv(EnvGitLabCI) != "" ||
		os.Getenv(EnvJenkinsURL) != "" ||
		os.Getenv(EnvTravisCI) != "" ||
		os.Getenv(EnvCircleCI) != ""
}

// IsGitHubActions returns true if the current environment is GitHub Actions.
func IsGitHubActions() bool {
	return os.Getenv(EnvGitHubActions) != "" && os.Getenv(EnvGitHubWorkspace) != ""
}

// IsGitLabCI returns true if the current environment is GitLab CI.
func IsGitLabCI() bool {
	return os.Getenv(EnvGitLabCI) != "" && os.Getenv(EnvGitLabProjectDir) != ""
}

// GetEnvWithFallbacks returns the value of the first non-empty environment variable
// from the provided list. If no environment variables are set, it returns the defaultValue.
//
// This function is useful for handling multiple possible environment variable names,
// which is especially important during the transition to standardized naming conventions.
func GetEnvWithFallbacks(envVars []string, defaultValue string, logger *slog.Logger) string {
	for i, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			// Log a deprecation warning if a non-primary environment variable is used
			if i > 0 && logger != nil {
				logger.Warn("Using legacy environment variable",
					"used_var", envVar,
					"preferred_var", envVars[0],
					"value", MaskSensitiveValue(val),
				)
			}
			return val
		}
	}
	return defaultValue
}

// MaskSensitiveValue masks sensitive data in values like database URLs to prevent
// exposing credentials in logs. This should be used whenever potentially sensitive
// environment variable values are logged.
func MaskSensitiveValue(value string) string {
	// If it looks like a database URL, mask the password
	if strings.HasPrefix(value, "postgres://") || strings.HasPrefix(value, "mysql://") {
		parts := strings.Split(value, "@")
		if len(parts) >= 2 {
			credentials := strings.Split(parts[0], ":")
			if len(credentials) >= 3 {
				// Format: postgres://username:password@host:port/database
				protocol := credentials[0]
				username := credentials[1]
				masked := protocol + ":" + username + ":****@" + strings.Join(parts[1:], "@")
				return masked
			}
		}
	}

	// For non-database URL values that might contain tokens or keys
	if len(value) > 8 && (strings.Contains(value, "key") ||
		strings.Contains(value, "token") ||
		strings.Contains(value, "secret")) {
		return value[:4] + "****" + value[len(value)-4:]
	}

	return value
}
