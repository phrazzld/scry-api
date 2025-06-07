//go:build test_without_external_deps

package ciutil_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/phrazzld/scry-api/internal/ciutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindProjectRoot_EnvironmentVariableOverride(t *testing.T) {
	// Save original environment
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)

	// Cleanup function
	cleanup := func() {
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
	}
	defer cleanup()

	// Create a temporary directory with go.mod for testing
	tempDir := t.TempDir()
	goModPath := filepath.Join(tempDir, "go.mod")
	err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	// Test with SCRY_PROJECT_ROOT environment variable set
	_ = os.Setenv(ciutil.EnvScryProjectRoot, tempDir)

	result, err := ciutil.FindProjectRoot(slog.Default())
	assert.NoError(t, err)
	assert.Equal(t, tempDir, result)
}

func TestFindProjectRoot_InvalidEnvironmentVariable(t *testing.T) {
	// Save original environment
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)

	// Cleanup function
	cleanup := func() {
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
	}
	defer cleanup()

	// Test with invalid SCRY_PROJECT_ROOT environment variable
	_ = os.Setenv(ciutil.EnvScryProjectRoot, "/path/that/does/not/exist")

	_, err := ciutil.FindProjectRoot(slog.Default())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid project root")
}

func TestFindMigrationsDir_EnvironmentVariableOverride(t *testing.T) {
	// Save original environment
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)

	// Cleanup function
	cleanup := func() {
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
	}
	defer cleanup()

	// Create a temporary directory structure for testing
	tempDir := t.TempDir()
	goModPath := filepath.Join(tempDir, "go.mod")
	err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	// Create migrations directory
	migrationsDir := filepath.Join(tempDir, "internal", "platform", "postgres", "migrations")
	err = os.MkdirAll(migrationsDir, 0755)
	require.NoError(t, err)

	// Create a test migration file
	migrationFile := filepath.Join(migrationsDir, "001_test.sql")
	err = os.WriteFile(migrationFile, []byte("CREATE TABLE test();"), 0644)
	require.NoError(t, err)

	// Test with SCRY_PROJECT_ROOT environment variable set
	_ = os.Setenv(ciutil.EnvScryProjectRoot, tempDir)

	result, err := ciutil.FindMigrationsDir(slog.Default())
	assert.NoError(t, err)
	assert.Equal(t, migrationsDir, result)
}

func TestFindMigrationsDir_InvalidEnvironmentVariable(t *testing.T) {
	// Save original environment
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)

	// Cleanup function
	cleanup := func() {
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
	}
	defer cleanup()

	// Test with invalid SCRY_PROJECT_ROOT environment variable
	_ = os.Setenv(ciutil.EnvScryProjectRoot, "/path/that/does/not/exist")

	_, err := ciutil.FindMigrationsDir(slog.Default())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find project root")
}
