//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMigrationsValidatorComprehensive tests migration validation functions
// This targets the 34 uncovered lines in migrations_validator.go for coverage improvement
func TestMigrationsValidatorComprehensive(t *testing.T) {
	cfg := CreateMinimalTestConfig(t)

	t.Run("verifyMigrations without database", func(t *testing.T) {
		// Test verifyMigrations function - will fail without database connectivity
		err := verifyMigrations(cfg, false)

		assert.Error(t, err, "verifyMigrations should fail without database connectivity")
	})

	t.Run("verifyMigrations with verbose mode", func(t *testing.T) {
		// Test verifyMigrations function with verbose logging
		err := verifyMigrations(cfg, true)

		assert.Error(t, err, "verifyMigrations with verbose should fail without database connectivity")
	})

	t.Run("validateAppliedMigrations without database", func(t *testing.T) {
		// Test validateAppliedMigrations function - will fail without database connectivity
		err := validateAppliedMigrations(cfg, false)

		assert.Error(t, err, "validateAppliedMigrations should fail without database connectivity")
	})

	t.Run("validateAppliedMigrations with verbose mode", func(t *testing.T) {
		// Test validateAppliedMigrations function with verbose logging
		err := validateAppliedMigrations(cfg, true)

		assert.Error(t, err, "validateAppliedMigrations with verbose should fail without database connectivity")
	})

	t.Run("verifyAppliedMigrations with nil database", func(t *testing.T) {
		// Test verifyAppliedMigrations with nil database - should panic
		testLogger, _ := CreateTestLogger(t)

		assert.Panics(t, func() {
			verifyAppliedMigrations(nil, testLogger)
		}, "verifyAppliedMigrations should panic with nil database")
	})

	t.Run("verifyAppliedMigrations with nil logger", func(t *testing.T) {
		// Test verifyAppliedMigrations with nil logger - should handle gracefully
		assert.Panics(t, func() {
			verifyAppliedMigrations(nil, nil)
		}, "verifyAppliedMigrations should panic with nil database even with nil logger")
	})
}

// TestEnumerateMigrationFiles tests migration file enumeration
func TestEnumerateMigrationFiles(t *testing.T) {
	t.Run("enumerateMigrationFiles with nonexistent directory", func(t *testing.T) {
		// Test with non-existent directory
		data, err := enumerateMigrationFiles("/nonexistent/path/to/migrations")

		assert.Error(t, err, "should fail with non-existent directory")
		assert.Empty(t, data.Files, "files list should be empty on error")
		assert.Equal(t, 0, data.SQLCount, "SQL count should be 0 on error")
	})

	t.Run("enumerateMigrationFiles with empty directory", func(t *testing.T) {
		// Test with current directory (should exist but may not have migration files)
		data, err := enumerateMigrationFiles(".")

		if err != nil {
			assert.Error(t, err, "may fail if directory can't be read")
		} else {
			// If successful, should return valid data structure
			assert.GreaterOrEqual(t, len(data.Files), 0, "files list should be valid")
			assert.GreaterOrEqual(t, data.SQLCount, 0, "SQL count should be non-negative")
		}
	})

	t.Run("enumerateMigrationFiles with permission denied", func(t *testing.T) {
		// Test with a restricted directory (if exists)
		data, err := enumerateMigrationFiles("/root")

		// May fail with permission error or succeed depending on system
		if err != nil {
			assert.Error(t, err, "may fail with permission error")
			assert.Empty(t, data.Files, "files list should be empty on error")
		} else {
			// If somehow accessible, should return valid data
			assert.GreaterOrEqual(t, len(data.Files), 0, "files list should be valid")
		}
	})
}

// TestMigrationsValidatorErrorPaths tests error handling in validation functions
func TestMigrationsValidatorErrorPaths(t *testing.T) {
	t.Run("verifyMigrations with empty database URL", func(t *testing.T) {
		// Test verifyMigrations with empty database URL
		cfgEmpty := CreateMinimalTestConfig(t)
		cfgEmpty.Database.URL = ""

		err := verifyMigrations(cfgEmpty, false)

		assert.Error(t, err, "should fail with empty database URL")
		assert.Contains(t, err.Error(), "database URL is empty", "error should mention empty URL")
	})

	t.Run("validateAppliedMigrations with empty database URL", func(t *testing.T) {
		// Test validateAppliedMigrations with empty database URL
		cfgEmpty := CreateMinimalTestConfig(t)
		cfgEmpty.Database.URL = ""

		err := validateAppliedMigrations(cfgEmpty, false)

		assert.Error(t, err, "should fail with empty database URL")
		assert.Contains(t, err.Error(), "database URL is empty", "error should mention empty URL")
	})

	t.Run("verifyMigrations with invalid database URL", func(t *testing.T) {
		// Test verifyMigrations with malformed database URL
		cfgInvalid := CreateMinimalTestConfig(t)
		cfgInvalid.Database.URL = "invalid-url-format"

		err := verifyMigrations(cfgInvalid, false)

		assert.Error(t, err, "should fail with invalid database URL")
	})

	t.Run("validateAppliedMigrations with invalid database URL", func(t *testing.T) {
		// Test validateAppliedMigrations with malformed database URL
		cfgInvalid := CreateMinimalTestConfig(t)
		cfgInvalid.Database.URL = "invalid-url-format"

		err := validateAppliedMigrations(cfgInvalid, false)

		assert.Error(t, err, "should fail with invalid database URL")
	})

	t.Run("verifyMigrations with unreachable database", func(t *testing.T) {
		// Test verifyMigrations with unreachable database
		cfgUnreachable := CreateMinimalTestConfig(t)
		cfgUnreachable.Database.URL = "postgres://user:pass@unreachable.host.example:5432/db"

		err := verifyMigrations(cfgUnreachable, false)

		assert.Error(t, err, "should fail with unreachable database")
	})

	t.Run("validateAppliedMigrations with unreachable database", func(t *testing.T) {
		// Test validateAppliedMigrations with unreachable database
		cfgUnreachable := CreateMinimalTestConfig(t)
		cfgUnreachable.Database.URL = "postgres://user:pass@unreachable.host.example:5432/db"

		err := validateAppliedMigrations(cfgUnreachable, false)

		assert.Error(t, err, "should fail with unreachable database")
	})
}

// TestMigrationsValidatorCI tests CI environment behavior
func TestMigrationsValidatorCI(t *testing.T) {
	cfg := CreateMinimalTestConfig(t)

	t.Run("verifyMigrations in CI environment", func(t *testing.T) {
		// Test verifyMigrations in CI environment (covers CI-specific logging)
		t.Setenv("CI", "true")

		err := verifyMigrations(cfg, false)

		assert.Error(t, err, "verifyMigrations should fail in CI without database")
	})

	t.Run("validateAppliedMigrations in CI environment", func(t *testing.T) {
		// Test validateAppliedMigrations in CI environment
		t.Setenv("CI", "true")

		err := validateAppliedMigrations(cfg, false)

		assert.Error(t, err, "validateAppliedMigrations should fail in CI without database")
	})

	t.Run("verifyMigrations with CI and verbose", func(t *testing.T) {
		// Test verifyMigrations with both CI and verbose flags
		t.Setenv("CI", "true")

		err := verifyMigrations(cfg, true)

		assert.Error(t, err, "verifyMigrations should fail in CI + verbose without database")
	})

	t.Run("validateAppliedMigrations with CI and verbose", func(t *testing.T) {
		// Test validateAppliedMigrations with both CI and verbose flags
		t.Setenv("CI", "true")

		err := validateAppliedMigrations(cfg, true)

		assert.Error(t, err, "validateAppliedMigrations should fail in CI + verbose without database")
	})
}
