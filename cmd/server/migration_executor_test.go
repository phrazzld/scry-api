//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExecuteMigration tests migration execution functionality
func TestExecuteMigration(t *testing.T) {
	cfg := CreateMinimalTestConfig(t)

	t.Run("invalid command", func(t *testing.T) {
		err := executeMigration(cfg, "invalid_command", false)
		assert.Error(t, err, "should fail with invalid command")
	})

	t.Run("help command", func(t *testing.T) {
		// Help command should work without database
		err := executeMigration(cfg, "help", false)
		assert.NoError(t, err, "help command should succeed")
	})

	t.Run("version command", func(t *testing.T) {
		// Version command should work without database
		err := executeMigration(cfg, "version", false)
		assert.NoError(t, err, "version command should succeed")
	})

	t.Run("create command without database", func(t *testing.T) {
		// Create command should fail without proper database setup
		err := executeMigration(cfg, "create", false, "test_migration")
		assert.Error(t, err, "create command should fail without database")
	})

	t.Run("status command without database", func(t *testing.T) {
		// Status command should fail without database connectivity
		err := executeMigration(cfg, "status", false)
		assert.Error(t, err, "status command should fail without database")
	})

	t.Run("up command without database", func(t *testing.T) {
		// Up command should fail without database connectivity
		err := executeMigration(cfg, "up", false)
		assert.Error(t, err, "up command should fail without database")
	})

	t.Run("down command without database", func(t *testing.T) {
		// Down command should fail without database connectivity
		err := executeMigration(cfg, "down", false)
		assert.Error(t, err, "down command should fail without database")
	})

	t.Run("verbose flag handling", func(t *testing.T) {
		// Test verbose flag - should still fail but with verbose output
		err := executeMigration(cfg, "status", true)
		assert.Error(t, err, "status command should fail without database even with verbose flag")
	})
}

// TestMigrationValidationFunctions tests migration validation functions
func TestMigrationValidationFunctions(t *testing.T) {
	cfg := CreateMinimalTestConfig(t)

	t.Run("verifyMigrations without database", func(t *testing.T) {
		// This should fail when database is not accessible
		err := verifyMigrations(cfg, false)
		assert.Error(t, err, "should fail without database connectivity")
	})

	t.Run("verifyMigrations with verbose", func(t *testing.T) {
		// This should fail when database is not accessible but with verbose logging
		err := verifyMigrations(cfg, true)
		assert.Error(t, err, "should fail without database connectivity even with verbose")
	})

	t.Run("validateAppliedMigrations without database", func(t *testing.T) {
		// This should fail when database is not accessible
		err := validateAppliedMigrations(cfg, false)
		assert.Error(t, err, "should fail without database connectivity")
	})

	t.Run("validateAppliedMigrations with verbose", func(t *testing.T) {
		// This should fail when database is not accessible but with verbose logging
		err := validateAppliedMigrations(cfg, true)
		assert.Error(t, err, "should fail without database connectivity even with verbose")
	})

	t.Run("verifyAppliedMigrations with nil database", func(t *testing.T) {
		// verifyAppliedMigrations panics with nil database when trying to use it
		// This is expected behavior - the function expects a valid database
		assert.Panics(t, func() {
			verifyAppliedMigrations(nil, nil)
		}, "verifyAppliedMigrations panics with nil database (expected behavior)")
	})
}

// TestMigrationHelpers tests helper functions for migration handling
func TestMigrationHelpers(t *testing.T) {
	cfg := CreateMinimalTestConfig(t)

	t.Run("handleMigrations with no operation", func(t *testing.T) {
		err := handleMigrations(cfg, "", "", false, false, false)
		assert.Error(t, err, "should fail when no operation is specified")
		assert.Contains(t, err.Error(), "no migration operation specified")
	})

	t.Run("handleMigrations with verify only", func(t *testing.T) {
		err := handleMigrations(cfg, "", "", false, true, false)
		// This will fail because we don't have a real database, but tests the code path
		assert.Error(t, err, "should fail without proper setup but test the code path")
	})

	t.Run("handleMigrations with validate migrations", func(t *testing.T) {
		err := handleMigrations(cfg, "", "", false, false, true)
		// This will fail because we don't have a real database, but tests the code path
		assert.Error(t, err, "should fail without proper setup but test the code path")
	})

	t.Run("handleMigrations with migrate command", func(t *testing.T) {
		err := handleMigrations(cfg, "help", "", false, false, false)
		// Help command will fail without database connectivity but tests the code path
		assert.Error(t, err, "help command should fail without database connectivity")
	})

	t.Run("handleMigrations with create command", func(t *testing.T) {
		err := handleMigrations(cfg, "create", "test_migration", false, false, false)
		// This will fail without database but tests the code path
		assert.Error(t, err, "create command should fail without database but test the code path")
	})

	t.Run("handleMigrations with verbose flag", func(t *testing.T) {
		err := handleMigrations(cfg, "help", "", true, false, false)
		// Help command with verbose will fail without database connectivity but tests the code path
		assert.Error(t, err, "help command with verbose should fail without database connectivity")
	})
}
