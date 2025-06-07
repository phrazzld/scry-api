//go:build !integration || test_without_external_deps

// Package testutils provides common utilities for testing across the application.
// It centralizes repeated test setup and teardown logic to avoid duplication
// and standardize testing practices.
package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// IsIntegrationTestEnvironment returns true if the environment is configured
// for running integration tests with a database connection.
// Integration tests should check this and skip if not in an integration test environment.
func IsIntegrationTestEnvironment() bool {
	return os.Getenv("DATABASE_URL") != ""
}

// GetTestDatabaseURL returns the database URL for integration tests.
// If DATABASE_URL environment variable is set, it's used directly.
// If not, it returns an error via the testing.T's Fatalf method.
// This version is designed for use within individual test functions.
func GetTestDatabaseURL(t *testing.T) string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL environment variable is required for this test")
	}
	return dbURL
}

// MustGetTestDatabaseURL returns the database URL for integration tests.
// This version is designed for use in TestMain functions where a testing.T is not available.
// It panics if DATABASE_URL is not set.
func MustGetTestDatabaseURL() string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// ALLOW-PANIC
		panic("DATABASE_URL environment variable is required for integration tests")
	}
	return dbURL
}

// SetupEnv sets up environment variables for testing and returns a cleanup function.
// It captures the original environment variable values, sets new values for the test,
// and returns a function that will restore the original values when called.
//
// Parameters:
//   - t: The testing.T instance for the current test
//   - envVars: A map of environment variable names to the values they should be set to
//
// Returns:
//   - A cleanup function that should be deferred to restore the original environment
//     variables after the test completes.
//
// Example usage:
//
//	func TestSomething(t *testing.T) {
//	    cleanup := testutils.SetupEnv(t, map[string]string{
//	        "SOME_ENV_VAR": "test-value",
//	    })
//	    defer cleanup()
//
//	    // Test code that depends on environment variables
//	}
func SetupEnv(t *testing.T, envVars map[string]string) func() {
	// Save current environment values
	originalValues := make(map[string]string)
	for name := range envVars {
		originalValues[name] = os.Getenv(name)
	}

	// Set new environment variables
	for name, value := range envVars {
		err := os.Setenv(name, value)
		require.NoError(t, err, "Failed to set environment variable %s", name)
	}

	// Return cleanup function
	return func() {
		// Restore original environment
		for name, value := range originalValues {
			if value == "" {
				err := os.Unsetenv(name)
				if err != nil {
					// In a real application, we might want to log this
					// For tests, we'll ignore these errors as they're unlikely
					// and won't affect test results
					t.Logf("Warning: Failed to unset env var %s: %v", name, err)
				}
			} else {
				err := os.Setenv(name, value)
				if err != nil {
					t.Logf("Warning: Failed to restore env var %s: %v", name, err)
				}
			}
		}
	}
}
