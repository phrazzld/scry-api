//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMigrationsHelpersCoverageBoost tests remaining uncovered functions in migrations_helpers.go
// This targets the 13 uncovered lines to boost coverage toward 70% target
func TestMigrationsHelpersCoverageBoost(t *testing.T) {
	t.Run("GetTestDatabaseURL CI environment paths", func(t *testing.T) {
		// Test GetTestDatabaseURL CI-specific behavior

		// Test with DATABASE_URL in CI environment
		t.Setenv("CI", "true")
		t.Setenv("DATABASE_URL", "postgres://test:test@127.0.0.1:5432/testdb")

		url := GetTestDatabaseURL()

		// Should standardize the URL for CI by forcing postgres credentials
		expected := "postgres://postgres:postgres@127.0.0.1:5432/testdb"
		assert.Equal(t, expected, url, "should standardize to postgres credentials in CI")
	})

	t.Run("GetTestDatabaseURL GITHUB_ACTIONS environment", func(t *testing.T) {
		// Test with GITHUB_ACTIONS environment variable
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "true")
		t.Setenv("DATABASE_URL", "postgres://user:pass@127.0.0.1:5432/db")

		url := GetTestDatabaseURL()

		// Should standardize the URL for GitHub Actions by forcing postgres credentials
		expected := "postgres://postgres:postgres@127.0.0.1:5432/db"
		assert.Equal(t, expected, url, "should standardize to postgres credentials in GITHUB_ACTIONS environment")
	})

	t.Run("GetTestDatabaseURL CI defaults", func(t *testing.T) {
		// Test CI default database URL when no DATABASE_URL or SCRY_TEST_DB_URL is set
		t.Setenv("CI", "true")
		t.Setenv("DATABASE_URL", "")
		t.Setenv("SCRY_TEST_DB_URL", "")

		url := GetTestDatabaseURL()

		// Should return CI default URL
		expected := "postgres://postgres:postgres@localhost:5432/scry_api_test?sslmode=disable"
		assert.Equal(t, expected, url, "should return CI default URL when no env vars set")
	})

	t.Run("GetTestDatabaseURL SCRY_TEST_DB_URL precedence", func(t *testing.T) {
		// Test that SCRY_TEST_DB_URL is used when DATABASE_URL is not set
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "")
		t.Setenv("DATABASE_URL", "")
		t.Setenv("SCRY_TEST_DB_URL", "postgres://test:test@localhost:5432/scry_test")

		url := GetTestDatabaseURL()

		expected := "postgres://test:test@localhost:5432/scry_test"
		assert.Equal(t, expected, url, "should use SCRY_TEST_DB_URL when DATABASE_URL not set")
	})

	t.Run("GetTestDatabaseURL no environment variables", func(t *testing.T) {
		// Test behavior when no relevant environment variables are set
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "")
		t.Setenv("DATABASE_URL", "")
		t.Setenv("SCRY_TEST_DB_URL", "")

		url := GetTestDatabaseURL()

		// Should return default local development URL when no configuration is available
		expected := "postgres://testuser:testpass@localhost:5432/scry_api_test?sslmode=disable"
		assert.Equal(t, expected, url, "should return default URL when no configuration available")
	})
}

// TestMigrationsHelpersEdgeCases tests edge cases in helper functions
func TestMigrationsHelpersEdgeCases(t *testing.T) {
	t.Run("standardizeCIDatabaseURL edge cases", func(t *testing.T) {
		// Test standardizeCIDatabaseURL with edge cases

		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "localhost_host",
				input:    "postgres://user:pass@localhost:5432/db",
				expected: "postgres://postgres:postgres@localhost:5432/db",
			},
			{
				name:     "127_0_0_1_host",
				input:    "postgres://user:pass@127.0.0.1:5432/db",
				expected: "postgres://postgres:postgres@127.0.0.1:5432/db",
			},
			{
				name:     "with_query_params",
				input:    "postgres://user:pass@127.0.0.1:5432/db?sslmode=disable&timeout=30",
				expected: "postgres://postgres:postgres@127.0.0.1:5432/db?sslmode=disable&timeout=30",
			},
			{
				name:     "empty_string",
				input:    "",
				expected: "",
			},
			{
				name:     "other_ip_host",
				input:    "postgres://user:pass@192.168.1.1:5432/db",
				expected: "postgres://postgres:postgres@192.168.1.1:5432/db",
			},
			{
				name:     "postgresql_protocol",
				input:    "postgresql://user:pass@127.0.0.1:5433/db",
				expected: "postgresql://postgres:postgres@127.0.0.1:5433/db",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := standardizeCIDatabaseURL(tc.input)
				assert.Equal(t, tc.expected, result, "standardization failed for %s", tc.name)
			})
		}
	})

	t.Run("maskPassword function edge cases", func(t *testing.T) {
		// Test maskPassword function with various URL formats

		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "standard_url",
				input:    "postgres://user:password@host:5432/db",
				expected: "postgres://user:****@host:5432/db",
			},
			{
				name:     "url_with_special_chars",
				input:    "postgres://user:p@ss!word@host:5432/db",
				expected: "postgres://user:****@host:5432/db",
			},
			{
				name:     "url_without_password",
				input:    "postgres://user@host:5432/db",
				expected: "postgres://user@host:5432/db",
			},
			{
				name:     "url_with_empty_password",
				input:    "postgres://user:@host:5432/db",
				expected: "postgres://user:****@host:5432/db",
			},
			{
				name:     "malformed_url",
				input:    "not-a-url",
				expected: "not-a-url",
			},
			{
				name:     "empty_string",
				input:    "",
				expected: "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := maskPassword(tc.input)
				assert.Equal(t, tc.expected, result, "password masking failed for %s", tc.name)
			})
		}
	})

	t.Run("FindMigrationsDir error conditions", func(t *testing.T) {
		// Test FindMigrationsDir error handling

		// Change to a directory that doesn't have migrations
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()

		// Create and change to temporary directory
		tempDir := t.TempDir()
		_ = os.Chdir(tempDir)

		// Now FindMigrationsDir should fail
		path, err := FindMigrationsDir()

		// Should return error when migrations directory not found
		assert.Error(t, err, "FindMigrationsDir should fail when migrations directory not found")
		assert.Empty(t, path, "path should be empty on error")
		assert.Contains(t, err.Error(), "migrations", "error should mention migrations")
	})

	t.Run("FindProjectRoot error conditions", func(t *testing.T) {
		// Test FindProjectRoot error handling

		// Change to a directory that doesn't have project root indicators
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()

		// Create and change to temporary directory without go.mod
		tempDir := t.TempDir()
		_ = os.Chdir(tempDir)

		// Now FindProjectRoot should fail
		path, err := FindProjectRoot()

		// Should return error when project root not found
		assert.Error(t, err, "FindProjectRoot should fail when project root not found")
		assert.Empty(t, path, "path should be empty on error")
	})
}

// TestMigrationsHelpersEnvironmentHandling tests environment variable handling
func TestMigrationsHelpersEnvironmentHandling(t *testing.T) {
	t.Run("GetTestDatabaseURL with complex CI scenarios", func(t *testing.T) {
		// Test various CI environment combinations

		// Test CI=true with complex DATABASE_URL
		t.Setenv("CI", "true")
		t.Setenv("GITHUB_ACTIONS", "")
		complexURL := "postgresql://testuser:complex!password@127.0.0.1:5433/test_db?sslmode=require&connect_timeout=10"
		t.Setenv("DATABASE_URL", complexURL)

		url := GetTestDatabaseURL()
		expected := "postgresql://testuser:complex!password@localhost:5433/test_db?sslmode=require&connect_timeout=10"
		assert.Equal(t, expected, url, "should handle complex URL standardization in CI")

		// Test GITHUB_ACTIONS=true with DATABASE_URL
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "true")
		simpleURL := "postgres://github:actions@127.0.0.1:5432/testdb"
		t.Setenv("DATABASE_URL", simpleURL)

		url = GetTestDatabaseURL()
		expected = "postgres://github:actions@localhost:5432/testdb"
		assert.Equal(t, expected, url, "should standardize URL in GITHUB_ACTIONS environment")
	})

	t.Run("GetTestDatabaseURL precedence order", func(t *testing.T) {
		// Test environment variable precedence order

		// Set all environment variables
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "")
		t.Setenv("DATABASE_URL", "postgres://database:url@localhost:5432/db1")
		t.Setenv("SCRY_TEST_DB_URL", "postgres://scry:test@localhost:5432/db2")

		// DATABASE_URL should take precedence
		url := GetTestDatabaseURL()
		expected := "postgres://database:url@localhost:5432/db1"
		assert.Equal(t, expected, url, "DATABASE_URL should take precedence over SCRY_TEST_DB_URL")

		// Clear DATABASE_URL, SCRY_TEST_DB_URL should be used
		t.Setenv("DATABASE_URL", "")
		url = GetTestDatabaseURL()
		expected = "postgres://scry:test@localhost:5432/db2"
		assert.Equal(t, expected, url, "SCRY_TEST_DB_URL should be used when DATABASE_URL is empty")
	})
}
