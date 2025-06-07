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

func TestFindProjectRoot_GitHubActionsScenarios(t *testing.T) {
	// Save original environment
	originalGitHubActions := os.Getenv(ciutil.EnvGitHubActions)
	originalGitHubWorkspace := os.Getenv(ciutil.EnvGitHubWorkspace)
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)

	// Cleanup function
	cleanup := func() {
		_ = os.Setenv(ciutil.EnvGitHubActions, originalGitHubActions)
		_ = os.Setenv(ciutil.EnvGitHubWorkspace, originalGitHubWorkspace)
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
	}
	defer cleanup()

	tests := []struct {
		name            string
		gitHubActions   string
		gitHubWorkspace string
		setupWorkspace  bool
		expectError     bool
		errorContains   string
	}{
		{
			name:            "GitHub_Actions_valid_workspace",
			gitHubActions:   "true",
			gitHubWorkspace: "",
			setupWorkspace:  true,
			expectError:     false,
		},
		{
			name:            "GitHub_Actions_invalid_workspace",
			gitHubActions:   "true",
			gitHubWorkspace: "/invalid/path/that/does/not/exist",
			setupWorkspace:  false,
			expectError:     true,
			errorContains:   "invalid project root",
		},
		{
			name:            "GitHub_Actions_no_workspace",
			gitHubActions:   "true",
			gitHubWorkspace: "",
			setupWorkspace:  false,
			expectError:     false, // Should fall back to auto-detection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear SCRY_PROJECT_ROOT to test GitHub Actions path
			_ = os.Setenv(ciutil.EnvScryProjectRoot, "")
			_ = os.Setenv(ciutil.EnvGitHubActions, tt.gitHubActions)

			var workspaceDir string
			if tt.setupWorkspace {
				// Create a temporary directory with go.mod for valid workspace
				tempDir := t.TempDir()
				goModPath := filepath.Join(tempDir, "go.mod")
				err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
				require.NoError(t, err)
				workspaceDir = tempDir
			} else {
				workspaceDir = tt.gitHubWorkspace
			}

			_ = os.Setenv(ciutil.EnvGitHubWorkspace, workspaceDir)

			result, err := ciutil.FindProjectRoot(slog.Default())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.setupWorkspace {
					assert.Equal(t, workspaceDir, result)
				}
			}
		})
	}
}

func TestFindProjectRoot_GitLabCIScenarios(t *testing.T) {
	// Save original environment
	originalGitLabCI := os.Getenv(ciutil.EnvGitLabCI)
	originalGitLabProjectDir := os.Getenv(ciutil.EnvGitLabProjectDir)
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)
	originalGitHubActions := os.Getenv(ciutil.EnvGitHubActions)

	// Cleanup function
	cleanup := func() {
		_ = os.Setenv(ciutil.EnvGitLabCI, originalGitLabCI)
		_ = os.Setenv(ciutil.EnvGitLabProjectDir, originalGitLabProjectDir)
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
		_ = os.Setenv(ciutil.EnvGitHubActions, originalGitHubActions)
	}
	defer cleanup()

	tests := []struct {
		name             string
		gitLabCI         string
		gitLabProjectDir string
		setupProjectDir  bool
		expectError      bool
		errorContains    string
	}{
		{
			name:             "GitLab_CI_valid_project_dir",
			gitLabCI:         "true",
			gitLabProjectDir: "",
			setupProjectDir:  true,
			expectError:      false,
		},
		{
			name:             "GitLab_CI_invalid_project_dir",
			gitLabCI:         "true",
			gitLabProjectDir: "/invalid/path/that/does/not/exist",
			setupProjectDir:  false,
			expectError:      true,
			errorContains:    "invalid project root",
		},
		{
			name:             "GitLab_CI_no_project_dir",
			gitLabCI:         "true",
			gitLabProjectDir: "",
			setupProjectDir:  false,
			expectError:      false, // Should fall back to auto-detection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear higher priority env vars to test GitLab CI path
			_ = os.Setenv(ciutil.EnvScryProjectRoot, "")
			_ = os.Setenv(ciutil.EnvGitHubActions, "")
			_ = os.Setenv(ciutil.EnvGitLabCI, tt.gitLabCI)

			var projectDir string
			if tt.setupProjectDir {
				// Create a temporary directory with go.mod for valid project dir
				tempDir := t.TempDir()
				goModPath := filepath.Join(tempDir, "go.mod")
				err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
				require.NoError(t, err)
				projectDir = tempDir
			} else {
				projectDir = tt.gitLabProjectDir
			}

			_ = os.Setenv(ciutil.EnvGitLabProjectDir, projectDir)

			result, err := ciutil.FindProjectRoot(slog.Default())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.setupProjectDir {
					assert.Equal(t, projectDir, result)
				}
			}
		})
	}
}

func TestFindProjectRoot_AutoDetectionScenarios(t *testing.T) {
	// Save original environment
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)
	originalGitHubActions := os.Getenv(ciutil.EnvGitHubActions)
	originalGitLabCI := os.Getenv(ciutil.EnvGitLabCI)

	// Cleanup function
	cleanup := func() {
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
		_ = os.Setenv(ciutil.EnvGitHubActions, originalGitHubActions)
		_ = os.Setenv(ciutil.EnvGitLabCI, originalGitLabCI)
	}
	defer cleanup()

	// Clear all environment variables to force auto-detection
	_ = os.Setenv(ciutil.EnvScryProjectRoot, "")
	_ = os.Setenv(ciutil.EnvGitHubActions, "")
	_ = os.Setenv(ciutil.EnvGitLabCI, "")

	t.Run("Auto_detection_finds_project_root", func(t *testing.T) {
		// This should find the actual project root since we're running from within the project
		result, err := ciutil.FindProjectRoot(slog.Default())
		assert.NoError(t, err)
		assert.NotEmpty(t, result)

		// Verify it contains go.mod
		goModPath := filepath.Join(result, "go.mod")
		_, err = os.Stat(goModPath)
		assert.NoError(t, err, "Project root should contain go.mod file")
	})
}

func TestFindProjectRootByTraversal_WithGoMod(t *testing.T) {
	// Create a temporary directory structure with go.mod
	tempDir := t.TempDir()

	// Create subdirectories
	subDir := filepath.Join(tempDir, "sub", "dir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create go.mod in the root
	goModPath := filepath.Join(tempDir, "go.mod")
	err = os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	// Change to subdirectory and test traversal
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Clear environment variables to force auto-detection
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)
	originalGitHubActions := os.Getenv(ciutil.EnvGitHubActions)
	originalGitLabCI := os.Getenv(ciutil.EnvGitLabCI)

	defer func() {
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
		_ = os.Setenv(ciutil.EnvGitHubActions, originalGitHubActions)
		_ = os.Setenv(ciutil.EnvGitLabCI, originalGitLabCI)
	}()

	_ = os.Setenv(ciutil.EnvScryProjectRoot, "")
	_ = os.Setenv(ciutil.EnvGitHubActions, "")
	_ = os.Setenv(ciutil.EnvGitLabCI, "")

	result, err := ciutil.FindProjectRoot(slog.Default())
	assert.NoError(t, err)

	// Use filepath.EvalSymlinks to resolve any symlinks for comparison
	expectedPath, err := filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	actualPath, err := filepath.EvalSymlinks(result)
	require.NoError(t, err)
	assert.Equal(t, expectedPath, actualPath)
}

func TestFindProjectRootByTraversal_WithGitDir(t *testing.T) {
	// Create a temporary directory structure with .git directory
	tempDir := t.TempDir()

	// Create subdirectories
	subDir := filepath.Join(tempDir, "sub", "dir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create .git directory in the root
	gitDirPath := filepath.Join(tempDir, ".git")
	err = os.MkdirAll(gitDirPath, 0755)
	require.NoError(t, err)

	// Change to subdirectory and test traversal
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Clear environment variables to force auto-detection
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)
	originalGitHubActions := os.Getenv(ciutil.EnvGitHubActions)
	originalGitLabCI := os.Getenv(ciutil.EnvGitLabCI)

	defer func() {
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
		_ = os.Setenv(ciutil.EnvGitHubActions, originalGitHubActions)
		_ = os.Setenv(ciutil.EnvGitLabCI, originalGitLabCI)
	}()

	_ = os.Setenv(ciutil.EnvScryProjectRoot, "")
	_ = os.Setenv(ciutil.EnvGitHubActions, "")
	_ = os.Setenv(ciutil.EnvGitLabCI, "")

	result, err := ciutil.FindProjectRoot(slog.Default())
	assert.NoError(t, err)

	// Use filepath.EvalSymlinks to resolve any symlinks for comparison
	expectedPath, err := filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	actualPath, err := filepath.EvalSymlinks(result)
	require.NoError(t, err)
	assert.Equal(t, expectedPath, actualPath)
}

func TestFindProjectRootByTraversal_NotFound(t *testing.T) {
	// Create a temporary directory without any project markers
	tempDir := t.TempDir()

	// Create subdirectories
	subDir := filepath.Join(tempDir, "sub", "dir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Change to subdirectory and test traversal
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Clear environment variables to force auto-detection
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)
	originalGitHubActions := os.Getenv(ciutil.EnvGitHubActions)
	originalGitLabCI := os.Getenv(ciutil.EnvGitLabCI)

	defer func() {
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
		_ = os.Setenv(ciutil.EnvGitHubActions, originalGitHubActions)
		_ = os.Setenv(ciutil.EnvGitLabCI, originalGitLabCI)
	}()

	_ = os.Setenv(ciutil.EnvScryProjectRoot, "")
	_ = os.Setenv(ciutil.EnvGitHubActions, "")
	_ = os.Setenv(ciutil.EnvGitLabCI, "")

	_, err = ciutil.FindProjectRoot(slog.Default())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to find project root")
}

func TestFindMigrationsDir_CoverageScenarios(t *testing.T) {
	// Save original environment
	originalScryProjectRoot := os.Getenv(ciutil.EnvScryProjectRoot)

	// Cleanup function
	cleanup := func() {
		_ = os.Setenv(ciutil.EnvScryProjectRoot, originalScryProjectRoot)
	}
	defer cleanup()

	t.Run("FindMigrationsDir_with_valid_project_root", func(t *testing.T) {
		// Create a temporary directory structure
		tempDir := t.TempDir()

		// Create go.mod
		goModPath := filepath.Join(tempDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
		require.NoError(t, err)

		// Create migrations directory structure
		migrationsDir := filepath.Join(tempDir, "internal", "platform", "postgres", "migrations")
		err = os.MkdirAll(migrationsDir, 0755)
		require.NoError(t, err)

		// Create a test migration file
		migrationFile := filepath.Join(migrationsDir, "001_test.sql")
		err = os.WriteFile(migrationFile, []byte("CREATE TABLE test();"), 0644)
		require.NoError(t, err)

		// Set project root to temp directory
		_ = os.Setenv(ciutil.EnvScryProjectRoot, tempDir)

		result, err := ciutil.FindMigrationsDir(slog.Default())
		assert.NoError(t, err)
		assert.Equal(t, migrationsDir, result)
	})

	t.Run("FindMigrationsDir_with_missing_migrations_dir", func(t *testing.T) {
		// Create a temporary directory structure without migrations
		tempDir := t.TempDir()

		// Create go.mod
		goModPath := filepath.Join(tempDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
		require.NoError(t, err)

		// Set project root to temp directory
		_ = os.Setenv(ciutil.EnvScryProjectRoot, tempDir)

		_, err = ciutil.FindMigrationsDir(slog.Default())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "migrations directory not found")
	})

	t.Run("FindMigrationsDir_with_invalid_project_root", func(t *testing.T) {
		// Set invalid project root
		_ = os.Setenv(ciutil.EnvScryProjectRoot, "/invalid/path/that/does/not/exist")

		_, err := ciutil.FindMigrationsDir(slog.Default())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find project root")
	})
}
