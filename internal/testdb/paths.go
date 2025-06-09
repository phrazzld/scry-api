//go:build (integration || test_without_external_deps) && !exported_core_functions

// Package testdb provides utilities specifically for database testing.
package testdb

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// This file contains project root and path detection utilities.

// getCurrentDir returns the current working directory or an error message if it fails
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("error getting current directory: %v", err)
	}
	return dir
}

// validateProjectRoot checks if a directory has the key characteristics of a project root directory.
// Returns true if the directory appears to be a valid project root.
func validateProjectRoot(path string, logger *slog.Logger) bool {
	// Primary indicator: go.mod file
	goModPath := filepath.Join(path, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		logger.Debug("go.mod not found, not a valid project root",
			slog.String("path", path),
			slog.String("error", err.Error()))
		return false
	}

	// Secondary indicators: Check for common project directories or files
	// Not all of these need to exist, but they strengthen our confidence
	indicators := 0
	secondaryMarkers := []string{
		".git",          // Git repository marker
		"cmd",           // Common Go project structure
		"internal",      // Common Go project structure
		"docs",          // Documentation directory
		".github",       // GitHub-specific directory
		".gitignore",    // Git ignore file
		"go.sum",        // Go dependency lockfile
		".golangci.yml", // Linter configuration
	}

	for _, marker := range secondaryMarkers {
		markerPath := filepath.Join(path, marker)
		if _, err := os.Stat(markerPath); err == nil {
			indicators++
			logger.Debug("found secondary project root marker",
				slog.String("marker", marker))
		}
	}

	// Log validation result
	isValid := true // go.mod exists, which is the primary requirement
	logger.Debug("project root validation result",
		slog.String("path", path),
		slog.Bool("has_go_mod", true), // We already checked this above
		slog.Int("secondary_indicators", indicators),
		slog.Bool("is_valid", isValid))

	return isValid
}

// checkAndGetRepo tries to detect the repository name from common patterns and environment variables
func checkAndGetRepo(logger *slog.Logger) string {
	// First check environment variables that might contain the repo name
	envVars := []string{"GITHUB_REPOSITORY", "CI_PROJECT_NAME"}
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

func findProjectRoot() (string, error) {
	// Create a logger with relevant context
	logger := slog.Default().With(
		slog.String("function", "findProjectRoot"),
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

	// 4a. Try common project subdirectories first if we might be in a nested location
	// This is particularly helpful in CI where tests might run in a subdirectory
	if isCIEnvironment() {
		logger.Debug("checking if current directory is a known project subdirectory")

		// Check if we're in a subdirectory of the project
		commonDirs := []string{"internal", "cmd", "pkg", "test", "scripts"}
		for _, subdir := range commonDirs {
			if strings.HasSuffix(dir, subdir) || strings.Contains(dir, "/"+subdir+"/") {
				logger.Info("detected common project subdirectory pattern",
					slog.String("pattern", subdir),
					slog.String("current_dir", dir))

				// Try to find project root by traversing up
				potentialRoot := dir
				logger.Debug("traversing up from subdirectory",
					slog.String("starting_dir", potentialRoot))

				// Increase max levels to improve reliability
				for i := 0; i < 8; i++ { // Go up max 8 levels (increased from 5)
					potentialRoot = filepath.Dir(potentialRoot)
					checkedPaths = append(checkedPaths, filepath.Join(potentialRoot, "go.mod"))

					logger.Debug("checking potential project root",
						slog.String("path", potentialRoot),
						slog.String("go_mod_path", filepath.Join(potentialRoot, "go.mod")),
						slog.Int("level", i+1))

					if validateProjectRoot(potentialRoot, logger) {
						logger.Info("project root found by subdirectory traversal",
							slog.String("project_root", potentialRoot),
							slog.String("original_subdirectory", subdir))
						return potentialRoot, nil
					}
				}
				logger.Debug("reached maximum traversal depth without finding project root")
				break
			}
		}
	}

	// 4b. Standard traversal - go up until we find go.mod
	maxAttempts := 15 // Increased from 10 to improve reliability
	attempts := 0
	logger.Info("performing standard directory traversal",
		slog.String("starting_dir", dir),
		slog.Int("max_attempts", maxAttempts))

	for attempts < maxAttempts {
		attempts++
		logger.Debug("checking directory in traversal",
			slog.String("directory", dir),
			slog.Int("attempt", attempts))

		checkedPaths = append(checkedPaths, filepath.Join(dir, "go.mod"))

		if validateProjectRoot(dir, logger) {
			logger.Info("project root found by standard traversal",
				slog.String("project_root", dir),
				slog.Int("attempts", attempts))
			return dir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(dir)
		// If we're at the root and haven't found go.mod, we've gone too far
		if parentDir == dir {
			logger.Warn("reached filesystem root without finding project root",
				slog.String("path", dir))
			break
		}
		dir = parentDir
	}

	// 5. Advanced pattern matching strategies as last resort
	logger.Info("trying advanced pattern matching strategies")
	currentDir, _ := os.Getwd()

	// 5a. If all else fails, try to find project by name patterns in the current path
	knownRepoPatterns := []string{repoName, "scry-api", "scry", "scry-api-go"}

	for _, segment := range knownRepoPatterns {
		logger.Debug("checking for project name pattern in path",
			slog.String("pattern", segment),
			slog.String("path", currentDir))

		if strings.Contains(currentDir, segment) {
			// Extract the project root by finding the segment in the path
			idx := strings.Index(currentDir, segment)
			if idx != -1 {
				possibleRoot := currentDir[:idx+len(segment)]
				checkedPaths = append(checkedPaths, filepath.Join(possibleRoot, "go.mod"))

				logger.Debug("found pattern match, checking validity",
					slog.String("pattern", segment),
					slog.String("possible_root", possibleRoot))

				if validateProjectRoot(possibleRoot, logger) {
					logger.Info("project root found by pattern matching",
						slog.String("project_root", possibleRoot),
						slog.String("matched_pattern", segment))
					return possibleRoot, nil
				}
			}
		}
	}

	// 5b. Last resort for CI: try common absolute paths that might contain the project
	if isCIEnvironment() {
		logger.Info("checking common CI absolute paths as last resort")

		// Common paths in various CI systems
		commonCIPaths := []string{
			"/github/workspace", // GitHub Actions
			"/home/runner/work", // GitHub Actions
			"/builds",           // GitLab
			"/workspace",        // Generic CI
			"/go/src",           // Common Go path
		}

		for _, basePath := range commonCIPaths {
			possiblePaths := []string{
				basePath,
				filepath.Join(basePath, repoName),
			}

			for _, path := range possiblePaths {
				// Skip if path doesn't exist
				if _, err := os.Stat(path); err != nil {
					continue
				}

				checkedPaths = append(checkedPaths, filepath.Join(path, "go.mod"))
				logger.Debug("checking common CI path",
					slog.String("path", path))

				if validateProjectRoot(path, logger) {
					logger.Info("project root found in common CI path",
						slog.String("project_root", path))
					return path, nil
				}
			}
		}
	}

	// 6. No project root found - provide comprehensive error information
	logger.Error("failed to find project root by any method",
		slog.Int("checked_paths_count", len(checkedPaths)),
		slog.Int("checked_env_vars_count", len(checkedEnvVars)))

	// Add specific guidance for CI environments
	if isCIEnvironment() {
		logger.Error("CI-specific guidance for project root detection failure",
			slog.String("recommendation", "Set SCRY_PROJECT_ROOT explicitly in CI configuration"),
			slog.String("github_actions_tip", "Check checkout action configuration, ensure it has fetch-depth:0"))
	}

	// Use our standardized error formatting function
	return "", formatProjectRootError(checkedPaths, checkedEnvVars)
}
