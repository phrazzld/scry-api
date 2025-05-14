//go:build exported_core_functions

package testdb

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FindProjectRoot is the public function for locating the project root directory.
// It uses various strategies to find the root, with special handling for CI environments.
func FindProjectRoot() (string, error) {
	// Create a logger with relevant context
	logger := slog.Default().With(
		slog.String("function", "FindProjectRoot"),
		slog.Bool("ci_environment", isCIEnvironment()),
		slog.Bool("github_actions", isGitHubActionsCI()),
	)

	// Log initial working directory for context
	startDir, err := os.Getwd()
	if err != nil {
		logger.Error("failed to get current working directory",
			slog.String("error", err.Error()))
		startDir = "<unknown>"
	}
	logger.Info("starting project root detection",
		slog.String("working_directory", startDir))

	// Keep track of paths we've checked for debugging
	checkedPaths := []string{}
	checkedEnvVars := []string{}

	// Try to detect repository name for use in various strategies
	repoName := checkAndGetRepo(logger)
	logger.Debug("using repository name for detection strategies",
		slog.String("repository", repoName))

	// 1. Check for explicit project root environment variable (highest priority)
	logger.Debug("checking for explicit project root environment variable")
	if projectRoot := os.Getenv("SCRY_PROJECT_ROOT"); projectRoot != "" {
		logger.Info("SCRY_PROJECT_ROOT environment variable is set",
			slog.String("value", projectRoot))

		checkedEnvVars = append(checkedEnvVars, "SCRY_PROJECT_ROOT="+projectRoot)

		// Validate the specified path
		if validateProjectRoot(projectRoot, logger) {
			logger.Info("validated project root specified by SCRY_PROJECT_ROOT",
				slog.String("project_root", projectRoot))
			return projectRoot, nil
		} else {
			logger.Warn("SCRY_PROJECT_ROOT is set but does not point to a valid project root",
				slog.String("project_root", projectRoot))
		}
	} else {
		logger.Debug("SCRY_PROJECT_ROOT environment variable is not set")
	}

	// GitHub Actions CI-specific handling (higher priority in CI)
	if isGitHubActionsCI() {
		logger.Info("detected GitHub Actions CI environment, using specialized detection logic")

		// 2a. First check GitHub workspace (most reliable in GitHub Actions)
		githubWorkspace := os.Getenv("GITHUB_WORKSPACE")
		if githubWorkspace != "" {
			logger.Info("GITHUB_WORKSPACE environment variable is set",
				slog.String("value", githubWorkspace))

			checkedEnvVars = append(checkedEnvVars, "GITHUB_WORKSPACE="+githubWorkspace)

			// First try: Direct workspace path (most common setup)
			if validateProjectRoot(githubWorkspace, logger) {
				logger.Info("project root found at GitHub workspace root",
					slog.String("project_root", githubWorkspace))
				return githubWorkspace, nil
			}

			// Second try: Repository might be in a subdirectory (monorepo setup)
			possiblePaths := []string{
				filepath.Join(githubWorkspace, repoName),
				filepath.Join(githubWorkspace, "src", repoName),
				filepath.Join(githubWorkspace, "go", "src", repoName),
			}

			for _, path := range possiblePaths {
				checkedPaths = append(checkedPaths, filepath.Join(path, "go.mod"))
				logger.Debug("checking possible monorepo location",
					slog.String("path", path))

				if validateProjectRoot(path, logger) {
					logger.Info("project root found in repository subdirectory",
						slog.String("project_root", path))
					return path, nil
				}
			}

			// Third try: If workspace exists but go.mod not found, investigate the directory
			// This is critical for diagnosing checkout issues or unexpected directory structures
			entries, readErr := os.ReadDir(githubWorkspace)
			if readErr != nil {
				logger.Warn("failed to read GitHub workspace directory contents",
					slog.String("error", readErr.Error()))
			} else {
				// Log directory contents for diagnostics
				names := make([]string, 0, len(entries))
				for _, entry := range entries {
					names = append(names, entry.Name())
				}
				logger.Info("GitHub workspace directory contents",
					slog.Any("entries", names))

				// Look for any directory that might have go.mod (could be nested)
				for _, entry := range entries {
					if entry.IsDir() {
						subdir := filepath.Join(githubWorkspace, entry.Name())
						if validateProjectRoot(subdir, logger) {
							logger.Info("found project root in GitHub workspace subdirectory",
								slog.String("project_root", subdir),
								slog.String("subdirectory", entry.Name()))
							return subdir, nil
						}
					}
				}
			}

			// GitHub workspace exists but no project root found - this is unusual
			logger.Warn("GitHub workspace exists but no valid project root found",
				slog.String("workspace", githubWorkspace))
		} else {
			logger.Warn("GITHUB_WORKSPACE environment variable not set despite GitHub Actions detection")
		}

		// 2b. Check additional GitHub Actions environment variables that might help
		// GitHub runner workspace is another possible location
		if runnerWorkspace := os.Getenv("RUNNER_WORKSPACE"); runnerWorkspace != "" {
			logger.Info("checking RUNNER_WORKSPACE location",
				slog.String("path", runnerWorkspace))

			checkedEnvVars = append(checkedEnvVars, "RUNNER_WORKSPACE="+runnerWorkspace)

			// Try the runner workspace directly and with repo name
			possiblePaths := []string{
				runnerWorkspace,
				filepath.Join(runnerWorkspace, repoName),
			}

			for _, path := range possiblePaths {
				checkedPaths = append(checkedPaths, filepath.Join(path, "go.mod"))
				if validateProjectRoot(path, logger) {
					logger.Info("project root found via RUNNER_WORKSPACE",
						slog.String("project_root", path))
					return path, nil
				}
			}
		}
	}

	// 3. Check for GitLab CI environment
	logger.Debug("checking for GitLab CI environment")
	if gitlabProjectDir := os.Getenv("CI_PROJECT_DIR"); gitlabProjectDir != "" {
		logger.Info("CI_PROJECT_DIR environment variable is set",
			slog.String("value", gitlabProjectDir))

		checkedEnvVars = append(checkedEnvVars, "CI_PROJECT_DIR="+gitlabProjectDir)

		if validateProjectRoot(gitlabProjectDir, logger) {
			logger.Info("project root found via CI_PROJECT_DIR",
				slog.String("project_root", gitlabProjectDir))
			return gitlabProjectDir, nil
		} else {
			logger.Warn("CI_PROJECT_DIR is set but does not point to a valid project root",
				slog.String("path", gitlabProjectDir))
		}
	} else {
		logger.Debug("CI_PROJECT_DIR environment variable is not set")
	}

	// 4. Start with current working directory and traverse upwards
	logger.Info("falling back to directory traversal strategy")
	dir, err := os.Getwd()
	if err != nil {
		logger.Error("failed to get current directory",
			slog.String("error", err.Error()))
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	logger.Debug("beginning traversal from current directory",
		slog.String("directory", dir))

	// Traverse up to find go.mod
	for {
		checkedPaths = append(checkedPaths, filepath.Join(dir, "go.mod"))

		if validateProjectRoot(dir, logger) {
			logger.Info("found project root via directory traversal",
				slog.String("project_root", dir))
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding go.mod
			break
		}
		dir = parent
	}

	// 5. Check a few common locations (lower priority)
	// These are common paths in development environments and CI systems
	commonLocations := []string{
		"/go/src/github.com/phrazzld/scry-api",
		filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "phrazzld", "scry-api"),
	}

	for _, location := range commonLocations {
		checkedPaths = append(checkedPaths, filepath.Join(location, "go.mod"))
		logger.Debug("checking common location", slog.String("path", location))

		if validateProjectRoot(location, logger) {
			logger.Info("found project root in common location",
				slog.String("project_root", location))
			return location, nil
		}
	}

	// If we got here, we didn't find a valid project root
	logger.Error("failed to find project root using all strategies",
		slog.Any("checked_paths", checkedPaths),
		slog.Any("checked_env_vars", checkedEnvVars))

	return "", fmt.Errorf(
		"failed to find project root: no valid project root found after trying %d paths and %d environment variables",
		len(checkedPaths),
		len(checkedEnvVars),
	)
}

// checkAndGetRepo attempts to determine the repository name from environment variables
// or falls back to the default "scry-api" for this project.
func checkAndGetRepo(logger *slog.Logger) string {
	// Check environment variables that might contain the repository name
	envVars := []string{
		"GITHUB_REPOSITORY",
		"CI_PROJECT_NAME",
		"REPO_NAME",
		"SCRY_REPO_NAME",
	}

	// Log the check
	logger.Debug("checking environment variables for repository name")

	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			// For GITHUB_REPOSITORY, format is owner/repo
			if envVar == "GITHUB_REPOSITORY" && strings.Contains(val, "/") {
				parts := strings.Split(val, "/")
				if len(parts) >= 2 {
					logger.Debug("detected repository name from environment variable",
						slog.String("source", envVar),
						slog.String("repository", parts[1]))
					return parts[1]
				}
			} else {
				logger.Debug("detected repository name from environment variable",
					slog.String("source", envVar),
					slog.String("repository", val))
				return val
			}
		}
	}

	// Default for this project
	return "scry-api"
}

// validateProjectRoot checks if the given directory is a valid project root.
// A valid project root must contain a go.mod file.
func validateProjectRoot(dir string, logger *slog.Logger) bool {
	// Check if go.mod exists in the directory
	goModPath := filepath.Join(dir, "go.mod")
	_, err := os.Stat(goModPath)
	isValid := err == nil

	// Log the result for debugging
	if isValid {
		logger.Debug("found valid project root with go.mod",
			slog.String("directory", dir),
			slog.String("go_mod_path", goModPath))
	} else {
		logger.Debug("directory is not a valid project root",
			slog.String("directory", dir),
			slog.String("go_mod_path", goModPath),
			slog.String("error", err.Error()))
	}

	return isValid
}

// isCIEnvironment returns true if the code is running in a CI environment.
func isCIEnvironment() bool {
	// Common CI environment variables
	ciEnvVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_URL",
		"CIRCLECI",
		"TRAVIS",
		"TF_BUILD", // Azure DevOps
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// isGitHubActionsCI returns true if the code is running in GitHub Actions.
func isGitHubActionsCI() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}
