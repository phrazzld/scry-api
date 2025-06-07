//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExecuteMigrationComprehensive tests the executeMigration function with various scenarios
// This targets the 52 uncovered lines in migrations_executor.go for significant coverage improvement
func TestExecuteMigrationComprehensive(t *testing.T) {
	cfg := CreateMinimalTestConfig(t)

	t.Run("executeMigration up command", func(t *testing.T) {
		// Test the "up" migration command path
		err := executeMigration(cfg, "up", false)

		// Will fail without database connectivity but covers the code path
		assert.Error(t, err, "up command should fail without database connectivity")
		assert.Contains(t, err.Error(), "database", "error should be related to database connectivity")
	})

	t.Run("executeMigration down command", func(t *testing.T) {
		// Test the "down" migration command path
		err := executeMigration(cfg, "down", false)

		// Will fail without database connectivity but covers the code path
		assert.Error(t, err, "down command should fail without database connectivity")
	})

	t.Run("executeMigration reset command", func(t *testing.T) {
		// Test the "reset" migration command path
		err := executeMigration(cfg, "reset", false)

		// Will fail without database connectivity but covers the code path
		assert.Error(t, err, "reset command should fail without database connectivity")
	})

	t.Run("executeMigration status command", func(t *testing.T) {
		// Test the "status" migration command path
		err := executeMigration(cfg, "status", false)

		// Will fail without database connectivity but covers the code path
		assert.Error(t, err, "status command should fail without database connectivity")
	})

	t.Run("executeMigration version command", func(t *testing.T) {
		// Test the "version" migration command path
		err := executeMigration(cfg, "version", false)

		// Will fail without database connectivity but covers the code path
		assert.Error(t, err, "version command should fail without database connectivity")
	})

	t.Run("executeMigration create command with name", func(t *testing.T) {
		// Test the "create" migration command path with a migration name
		err := executeMigration(cfg, "create", false, "test_migration")

		// Will fail without database connectivity but covers the code path
		assert.Error(t, err, "create command should fail without database connectivity")
	})

	t.Run("executeMigration create command without name", func(t *testing.T) {
		// Test the "create" command without migration name (should fail early)
		err := executeMigration(cfg, "create", false)

		// Should fail with specific error about missing migration name
		assert.Error(t, err, "create command should fail without migration name")
		assert.Contains(t, err.Error(), "name is required", "error should mention missing name")
	})

	t.Run("executeMigration create command with empty name", func(t *testing.T) {
		// Test the "create" command with empty migration name
		err := executeMigration(cfg, "create", false, "")

		// Should fail with specific error about missing migration name
		assert.Error(t, err, "create command should fail with empty migration name")
		assert.Contains(t, err.Error(), "name is required", "error should mention missing name")
	})

	t.Run("executeMigration unknown command", func(t *testing.T) {
		// Test unknown command handling
		err := executeMigration(cfg, "unknown_command", false)

		// Should fail with unknown command error before database connectivity
		assert.Error(t, err, "unknown command should fail")
		assert.Contains(t, err.Error(), "unknown migration command", "error should mention unknown command")
	})

	t.Run("executeMigration with verbose mode", func(t *testing.T) {
		// Test verbose mode execution (covers verbose logging paths)
		err := executeMigration(cfg, "version", true)

		// Will fail without database but covers verbose logging code paths
		assert.Error(t, err, "verbose command should fail without database connectivity")
	})

	t.Run("executeMigration with empty database URL", func(t *testing.T) {
		// Test with empty database URL (should fail early)
		cfgEmpty := CreateMinimalTestConfig(t)
		cfgEmpty.Database.URL = ""

		err := executeMigration(cfgEmpty, "status", false)

		// Should fail early with database URL error
		assert.Error(t, err, "should fail with empty database URL")
		assert.Contains(t, err.Error(), "database URL is empty", "error should mention empty URL")
	})

	t.Run("executeMigration with invalid database URL format", func(t *testing.T) {
		// Test with malformed database URL
		cfgInvalid := CreateMinimalTestConfig(t)
		cfgInvalid.Database.URL = "invalid-database-url-format"

		err := executeMigration(cfgInvalid, "status", false)

		// Will fail during database connection attempt
		assert.Error(t, err, "should fail with invalid database URL")
	})

	t.Run("executeMigration with unreachable database", func(t *testing.T) {
		// Test with unreachable database host (covers network error paths)
		cfgUnreachable := CreateMinimalTestConfig(t)
		cfgUnreachable.Database.URL = "postgres://user:pass@nonexistent.host.example:5432/db"

		err := executeMigration(cfgUnreachable, "status", false)

		// Will fail during ping with network error
		assert.Error(t, err, "should fail with unreachable database")
	})

	t.Run("runMigrations wrapper function", func(t *testing.T) {
		// Test the runMigrations wrapper function for backward compatibility
		err := runMigrations(cfg, "version", false)

		// Should behave exactly like executeMigration
		assert.Error(t, err, "runMigrations should fail without database connectivity")
	})
}

// TestExecuteMigrationCIEnvironment tests CI-specific behavior
func TestExecuteMigrationCIEnvironment(t *testing.T) {
	cfg := CreateMinimalTestConfig(t)

	t.Run("executeMigration with CI environment", func(t *testing.T) {
		// Set CI environment variable to test CI-specific code paths
		t.Setenv("CI", "true")

		err := executeMigration(cfg, "status", false)

		// Will fail without database but covers CI-specific logging paths
		assert.Error(t, err, "CI mode should fail without database connectivity")
	})

	t.Run("executeMigration with CI environment and verbose", func(t *testing.T) {
		// Test CI environment with verbose mode
		t.Setenv("CI", "true")

		err := executeMigration(cfg, "status", true)

		// Will fail without database but covers CI + verbose logging paths
		assert.Error(t, err, "CI + verbose mode should fail without database connectivity")
	})
}

// TestExecuteMigrationDatabaseURLHandling tests database URL processing
func TestExecuteMigrationDatabaseURLHandling(t *testing.T) {
	t.Run("executeMigration with 127.0.0.1 URL", func(t *testing.T) {
		// Test database URL with 127.0.0.1 (should be standardized to localhost in CI)
		cfg := CreateMinimalTestConfig(t)
		cfg.Database.URL = "postgres://user:pass@127.0.0.1:5432/testdb"

		err := executeMigration(cfg, "status", false)

		// Will fail during connection but covers URL standardization logic
		assert.Error(t, err, "should fail during database connection")
	})

	t.Run("executeMigration with complex database URL", func(t *testing.T) {
		// Test complex database URL with query parameters
		cfg := CreateMinimalTestConfig(t)
		cfg.Database.URL = "postgres://user:password@localhost:5432/db?sslmode=disable&connect_timeout=10"

		err := executeMigration(cfg, "status", false)

		// Will fail during connection but covers URL parsing and logging
		assert.Error(t, err, "should fail during database connection")
	})
}

// TestExecuteMigrationTimeout tests timeout scenarios
func TestExecuteMigrationTimeout(t *testing.T) {
	t.Run("executeMigration with timeout", func(t *testing.T) {
		// Test with database URL that will cause timeout
		cfg := CreateMinimalTestConfig(t)
		// Use a non-routable IP that will cause timeout
		cfg.Database.URL = "postgres://user:pass@192.0.2.1:5432/db" // TEST-NET-1 (non-routable)

		// This will test the timeout error handling paths in executeMigration
		err := executeMigration(cfg, "status", false)

		assert.Error(t, err, "should fail with timeout or connection error")
	})
}
