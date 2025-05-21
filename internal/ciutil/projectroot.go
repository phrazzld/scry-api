package ciutil

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Project root marker files
const (
	GoModFile    = "go.mod" // Primary marker file for Go projects
	GitDirectory = ".git"   // Git directory marker
)

// Common errors for project root detection
var (
	ErrProjectRootNotFound = errors.New("unable to find project root")
	ErrInvalidProjectRoot  = errors.New("invalid project root: no go.mod file found")
)

// FindProjectRoot returns the absolute path to the project root directory.
// It checks several sources in the following order:
//
// 1. SCRY_PROJECT_ROOT environment variable (explicit override)
// 2. GITHUB_WORKSPACE environment variable (GitHub Actions)
// 3. CI_PROJECT_DIR environment variable (GitLab CI)
// 4. Auto-detection by traversing directories upward looking for go.mod
//
// It returns an error if the project root cannot be determined.
func FindProjectRoot(logger *slog.Logger) (string, error) {
	// Get current working directory as starting point
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	if logger != nil {
		logger.Debug("Starting project root detection from working directory",
			"working_dir", workingDir,
		)
	}

	// Check explicit environment variable override first
	if projectRoot := os.Getenv(EnvScryProjectRoot); projectRoot != "" {
		if logger != nil {
			logger.Info("Using project root from SCRY_PROJECT_ROOT environment variable",
				"project_root", projectRoot,
			)
		}

		// Verify the directory exists and contains a go.mod file
		if !isValidProjectRoot(projectRoot) {
			return "", fmt.Errorf("%w at %s", ErrInvalidProjectRoot, projectRoot)
		}

		return projectRoot, nil
	}

	// Check GitHub Actions workspace
	if IsGitHubActions() {
		githubWorkspace := os.Getenv(EnvGitHubWorkspace)
		if githubWorkspace != "" {
			if logger != nil {
				logger.Info("Using project root from GitHub Actions workspace",
					"github_workspace", githubWorkspace,
				)
			}

			// Verify the directory exists and contains a go.mod file
			if !isValidProjectRoot(githubWorkspace) {
				return "", fmt.Errorf("%w at %s", ErrInvalidProjectRoot, githubWorkspace)
			}

			return githubWorkspace, nil
		}
	}

	// Check GitLab CI project directory
	if IsGitLabCI() {
		gitlabProjectDir := os.Getenv(EnvGitLabProjectDir)
		if gitlabProjectDir != "" {
			if logger != nil {
				logger.Info("Using project root from GitLab CI project directory",
					"gitlab_project_dir", gitlabProjectDir,
				)
			}

			// Verify the directory exists and contains a go.mod file
			if !isValidProjectRoot(gitlabProjectDir) {
				return "", fmt.Errorf("%w at %s", ErrInvalidProjectRoot, gitlabProjectDir)
			}

			return gitlabProjectDir, nil
		}
	}

	// Auto-detect by traversing directories upward
	if logger != nil {
		logger.Info("No environment variables found for project root, auto-detecting...")
	}

	projectRoot, err := findProjectRootByTraversal(workingDir, logger)
	if err != nil {
		return "", err
	}

	return projectRoot, nil
}

// findProjectRootByTraversal looks for project markers by traversing directories upward.
// It starts from the given directory and looks for go.mod or .git.
// Returns the absolute path to the project root if found, an error otherwise.
func findProjectRootByTraversal(startDir string, logger *slog.Logger) (string, error) {
	currentDir := startDir
	maxIterations := 10 // Limit traversal to prevent infinite loops

	for i := 0; i < maxIterations; i++ {
		if logger != nil {
			logger.Debug("Checking directory for project markers",
				"dir", currentDir,
				"iteration", i+1,
			)
		}

		// Check for go.mod (primary marker)
		goModPath := filepath.Join(currentDir, GoModFile)
		if fileExists(goModPath) {
			if logger != nil {
				logger.Info("Found project root with go.mod",
					"project_root", currentDir,
					"go_mod_path", goModPath,
				)
			}
			return currentDir, nil
		}

		// Check for .git directory (secondary marker)
		gitDirPath := filepath.Join(currentDir, GitDirectory)
		if dirExists(gitDirPath) {
			if logger != nil {
				logger.Info("Found potential project root with .git directory",
					"project_root", currentDir,
					"git_dir_path", gitDirPath,
				)
			}

			// Prefer directories with go.mod, but accept .git if we're sure it's valid
			return currentDir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)

		// If we've reached the filesystem root, stop traversing
		if parentDir == currentDir {
			break
		}

		currentDir = parentDir
	}

	// If we got here, we couldn't find the project root
	if logger != nil {
		logger.Error("Failed to find project root by directory traversal",
			"start_dir", startDir,
			"max_iterations", maxIterations,
		)
	}

	return "", ErrProjectRootNotFound
}

// isValidProjectRoot checks if the given directory exists and contains a go.mod file.
func isValidProjectRoot(dir string) bool {
	if !dirExists(dir) {
		return false
	}

	goModPath := filepath.Join(dir, GoModFile)
	return fileExists(goModPath)
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FindMigrationsDir returns the absolute path to the migrations directory.
// It first finds the project root, then appends the path to the migrations directory.
func FindMigrationsDir(logger *slog.Logger) (string, error) {
	projectRoot, err := FindProjectRoot(logger)
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}

	// The standard path to migrations directory is:
	// internal/platform/postgres/migrations
	migrationsPath := filepath.Join(projectRoot, "internal", "platform", "postgres", "migrations")

	if logger != nil {
		logger.Debug("Resolved migrations directory path",
			"project_root", projectRoot,
			"migrations_path", migrationsPath,
		)
	}

	// Verify the directory exists
	if !dirExists(migrationsPath) {
		return "", fmt.Errorf("migrations directory not found at %s", migrationsPath)
	}

	return migrationsPath, nil
}
