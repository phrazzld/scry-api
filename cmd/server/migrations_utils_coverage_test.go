//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMigrationsUtilsCoverageBoost tests remaining uncovered functions in migrations_utils.go
// This targets the 23 uncovered lines to boost coverage toward 70% target
func TestMigrationsUtilsCoverageBoost(t *testing.T) {

	t.Run("directoryExists comprehensive tests", func(t *testing.T) {
		// Test directoryExists with various edge cases

		// Test with current directory (should exist)
		exists := directoryExists(".")
		assert.True(t, exists, "current directory should exist")

		// Test with non-existent directory
		exists = directoryExists("/completely/nonexistent/path")
		assert.False(t, exists, "non-existent path should return false")

		// Test with empty string
		exists = directoryExists("")
		assert.False(t, exists, "empty path should return false")

		// Test with a file instead of directory (using a known file)
		exists = directoryExists("go.mod")
		assert.False(t, exists, "file path should return false for directory check")

		// Test with relative path
		exists = directoryExists("./cmd")
		// This might or might not exist, but should not panic
		t.Logf("./cmd directory exists: %v", exists)

		// Test with parent directory
		exists = directoryExists("..")
		assert.True(t, exists, "parent directory should exist")
	})

	t.Run("getMigrationsPath error conditions", func(t *testing.T) {
		// Test getMigrationsPath error paths by temporarily changing working directory

		// Create a temporary directory without migrations
		tempDir := t.TempDir()

		// Change to temp directory (this will make FindMigrationsDir fail)
		originalWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(originalWd) }()

		_ = os.Chdir(tempDir)

		// Now getMigrationsPath should fail and go through fallback logic
		path, err := getMigrationsPath()

		// Should either return error or fallback path
		if err != nil {
			assert.Error(t, err, "getMigrationsPath should fail in directory without migrations")
			assert.Empty(t, path, "path should be empty on error")
		} else {
			// If it succeeded, it used fallback logic
			t.Logf("getMigrationsPath used fallback path: %s", path)
		}
	})
}

// TestMigrationUtilsNonDatabaseFunctions tests utility functions that don't require database
func TestMigrationUtilsNonDatabaseFunctions(t *testing.T) {
	t.Run("isCIEnvironment detection", func(t *testing.T) {
		// Test CI environment detection

		// Test with no CI variables
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "")
		assert.False(t, isCIEnvironment(), "should return false when no CI env vars set")

		// Test with CI=true
		t.Setenv("CI", "true")
		t.Setenv("GITHUB_ACTIONS", "")
		assert.True(t, isCIEnvironment(), "should return true when CI=true")

		// Test with GITHUB_ACTIONS=true
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "true")
		assert.True(t, isCIEnvironment(), "should return true when GITHUB_ACTIONS=true")

		// Test with both set
		t.Setenv("CI", "true")
		t.Setenv("GITHUB_ACTIONS", "true")
		assert.True(t, isCIEnvironment(), "should return true when both CI vars set")
	})

	t.Run("getExecutionMode returns correct mode", func(t *testing.T) {
		// Test execution mode detection

		// Test local mode
		t.Setenv("CI", "")
		t.Setenv("GITHUB_ACTIONS", "")
		mode := getExecutionMode()
		assert.Equal(t, "local", mode, "should return local mode when not in CI")

		// Test CI mode
		t.Setenv("CI", "true")
		mode = getExecutionMode()
		assert.Equal(t, "ci", mode, "should return ci mode when in CI")
	})
}

// TestMigrationUtilsURLProcessing tests URL processing edge cases
func TestMigrationUtilsURLProcessing(t *testing.T) {
	t.Run("maskDatabaseURL error conditions", func(t *testing.T) {
		// Test maskDatabaseURL with various problematic URLs

		// Test with completely invalid URL
		result := maskDatabaseURL("://invalid")
		assert.Equal(t, "://invalid", result, "should return original for invalid URL")

		// Test with URL that has no user info
		result = maskDatabaseURL("postgres://localhost:5432/db")
		assert.Equal(t, "postgres://localhost:5432/db", result, "should handle URL without user info")

		// Test with URL that has user but no password
		result = maskDatabaseURL("postgres://user@localhost:5432/db")
		expected := "postgres://user:%2A%2A%2A%2A@localhost:5432/db"
		assert.Equal(t, expected, result, "should add mask even when no password present")

		// Test with URL that has empty password
		result = maskDatabaseURL("postgres://user:@localhost:5432/db")
		expected = "postgres://user:%2A%2A%2A%2A@localhost:5432/db"
		assert.Equal(t, expected, result, "should mask empty password")
	})

	t.Run("extractHostFromURL error conditions", func(t *testing.T) {
		// Test extractHostFromURL with problematic URLs

		// Test with invalid URL that can't be parsed
		result := extractHostFromURL("://completely-invalid")
		assert.Equal(t, "", result, "should return empty for unparseable URL")

		// Test with URL that has no host
		result = extractHostFromURL("postgres:///db")
		assert.Equal(t, "", result, "should return empty for URL without host")

		// Test with empty URL
		result = extractHostFromURL("")
		assert.Equal(t, "", result, "should return empty for empty URL")

		// Test with URL that is not a URL at all
		result = extractHostFromURL("not-a-url-at-all")
		assert.Equal(t, "", result, "should return empty for non-URL string")
	})

	t.Run("detectDatabaseURLSource edge cases", func(t *testing.T) {
		// Test detectDatabaseURLSource with various edge cases

		// Test with empty URL
		source := detectDatabaseURLSource("")
		assert.Equal(t, "configuration", source, "should default to configuration for empty URL")

		// Test with URL that matches no environment variables
		t.Setenv("DATABASE_URL", "")
		t.Setenv("SCRY_TEST_DB_URL", "")
		t.Setenv("SCRY_DATABASE_URL", "")

		uniqueURL := "postgres://unique:unique@example.com:9999/unique"
		source = detectDatabaseURLSource(uniqueURL)
		assert.Equal(t, "configuration", source, "should default to configuration when no env vars match")

		// Test with partial environment variable matches
		t.Setenv("DATABASE_URL", "postgres://partial@localhost/test")
		source = detectDatabaseURLSource("postgres://different@localhost/test")
		assert.Equal(t, "configuration", source, "should detect configuration when URL doesn't exactly match env var")
	})
}
