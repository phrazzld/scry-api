//go:build (integration || test_without_external_deps) && !exported_core_functions

// Package testdb provides utilities specifically for database testing.
package testdb

import "os"

// This file contains environment detection utilities for determining test execution context.

// IsIntegrationTestEnvironment returns true if any of the database URL environment
// variables are set, indicating that integration tests can be run.
func IsIntegrationTestEnvironment() bool {
	// Check if any of the database URL environment variables are set
	envVars := []string{"DATABASE_URL", "SCRY_TEST_DB_URL", "SCRY_DATABASE_URL"}

	for _, envVar := range envVars {
		if len(os.Getenv(envVar)) > 0 {
			return true
		}
	}

	return false
}

// ShouldSkipDatabaseTest returns true if the database connection environment variables
// are not set, indicating that database integration tests should be skipped.
// This provides a consistent way for tests to check for database availability.
func ShouldSkipDatabaseTest() bool {
	return !IsIntegrationTestEnvironment()
}

// isCIEnvironment returns true if running in any type of CI environment.
// It checks for common CI environment variables across multiple platforms.
// Use isGitHubActionsCI() for GitHub Actions specific detection.
func isCIEnvironment() bool {
	// Check common CI environment variables
	ciVars := []string{
		"CI",             // Generic
		"GITHUB_ACTIONS", // GitHub Actions
		"GITLAB_CI",      // GitLab CI
		"JENKINS_URL",    // Jenkins
		"TRAVIS",         // Travis CI
		"CIRCLECI",       // Circle CI
	}

	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// isGitHubActionsCI returns true if specifically running in GitHub Actions CI.
// This is used for GitHub Actions-specific configuration settings.
func isGitHubActionsCI() bool {
	return os.Getenv("GITHUB_ACTIONS") != "" && os.Getenv("GITHUB_WORKSPACE") != ""
}
