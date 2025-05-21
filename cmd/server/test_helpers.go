//go:build exported_core_functions

package main

import (
	"os"

	"github.com/phrazzld/scry-api/internal/config"
)

// loadConfig is a backwards compatibility function for tests.
// It simply calls loadAppConfig to match the original signature.
func loadConfig() (*config.Config, error) {
	return loadAppConfig()
}

// IsIntegrationTestEnvironment is a backwards compatibility helper.
// It checks if DATABASE_URL is set.
func IsIntegrationTestEnvironment() bool {
	return os.Getenv("DATABASE_URL") != ""
}

// Simplified versions of migration functions for tests that maintain
// backward compatibility with the original API

// Note: we've removed the test helper for runMigrations to avoid redeclaration conflicts
// The tests will now call the main function in migrations.go with the required parameters
