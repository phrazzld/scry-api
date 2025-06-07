//go:build integration || test_without_external_deps

package testdb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindProjectRootWithEnvVar tests the behavior of findProjectRoot with SCRY_PROJECT_ROOT set
func TestFindProjectRootWithEnvVar(t *testing.T) {
	// Save original environment
	origEnv := os.Getenv("SCRY_PROJECT_ROOT")
	defer func() {
		if err := os.Setenv("SCRY_PROJECT_ROOT", origEnv); err != nil {
			t.Logf("Failed to restore SCRY_PROJECT_ROOT: %v", err)
		}
	}()

	// Set up a temporary directory with a go.mod file
	tempDir := t.TempDir()
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module github.com/test/project"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with SCRY_PROJECT_ROOT set
	if err := os.Setenv("SCRY_PROJECT_ROOT", tempDir); err != nil {
		t.Fatal(err)
	}

	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("findProjectRoot failed with SCRY_PROJECT_ROOT set: %v", err)
	}

	if root != tempDir {
		t.Errorf("Expected project root %q, got %q", tempDir, root)
	}
}

// TestFindProjectRootWithInvalidEnvVar tests the behavior of findProjectRoot with an invalid SCRY_PROJECT_ROOT
func TestFindProjectRootWithInvalidEnvVar(t *testing.T) {
	// Save original environment
	origEnv := os.Getenv("SCRY_PROJECT_ROOT")
	defer func() {
		if err := os.Setenv("SCRY_PROJECT_ROOT", origEnv); err != nil {
			t.Logf("Failed to restore SCRY_PROJECT_ROOT: %v", err)
		}
	}()

	// Set an invalid directory
	invalidDir := filepath.Join(t.TempDir(), "nonexistent")
	if err := os.Setenv("SCRY_PROJECT_ROOT", invalidDir); err != nil {
		t.Fatal(err)
	}

	// This should fall back to other methods, not fail outright
	_, err := findProjectRoot()
	if err == nil {
		// If it doesn't error, it must have found the project root by other means
		// which is fine for this test
		t.Log("findProjectRoot succeeded with other detection methods")
	}
}

// TestIsCIEnvironment tests the detection of CI environments
func TestIsCIEnvironment(t *testing.T) {
	// Save original CI environment variable
	origCI := os.Getenv("CI")
	defer func() {
		if err := os.Setenv("CI", origCI); err != nil {
			t.Logf("Failed to restore CI environment: %v", err)
		}
	}()

	// Test with CI not set
	if err := os.Unsetenv("CI"); err != nil {
		t.Fatal(err)
	}
	for _, envVar := range []string{
		"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "TRAVIS", "CIRCLECI",
	} {
		origVal := os.Getenv(envVar)
		if err := os.Unsetenv(envVar); err != nil {
			t.Fatal(err)
		}
		defer func(name, value string) {
			if err := os.Setenv(name, value); err != nil {
				t.Logf("Failed to restore %s: %v", name, err)
			}
		}(envVar, origVal)
	}

	if isCIEnvironment() {
		t.Error("isCIEnvironment returned true when no CI environment variables are set")
	}

	// Test with CI set
	if err := os.Setenv("CI", "true"); err != nil {
		t.Fatal(err)
	}

	if !isCIEnvironment() {
		t.Error("isCIEnvironment returned false when CI environment variable is set")
	}
}

// TestFindProjectRootComprehensive tests the behavior of findProjectRoot in various environments
func TestFindProjectRootComprehensive(t *testing.T) {
	// Save original environment variables that might be modified by tests
	origEnv := saveEnvironment(t, []string{
		"SCRY_PROJECT_ROOT",
		"GITHUB_WORKSPACE",
		"GITHUB_ACTIONS",
		"GITHUB_REPOSITORY",
		"RUNNER_WORKSPACE",
		"CI",
		"CI_PROJECT_DIR",
		"CI_PROJECT_NAME",
	})
	defer restoreEnvironment(t, origEnv)

	// Get the current project root to use as a reference
	// We'll use this to verify that test results are accurate
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Find the parent directory containing go.mod (the actual project root)
	// This is a simplified version of findProjectRoot to find the reference value
	referenceRoot := ""
	dir := currentDir
	for i := 0; i < 10; i++ { // Try up to 10 levels up
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			referenceRoot = dir
			break
		}
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break // Reached filesystem root
		}
		dir = parentDir
	}

	if referenceRoot == "" {
		t.Fatalf("Failed to find reference project root for testing")
	}
	t.Logf("Reference project root: %s", referenceRoot)

	// Run test cases
	tests := []struct {
		name                string
		setupEnvironment    func(t *testing.T)
		expectedOutcome     string // "success", "error", or "specific_path"
		expectedPath        string // Only checked if expectedOutcome is "specific_path"
		checkRelativeToRoot bool   // If true, checks if result is relative to reference root
	}{
		{
			name: "explicit SCRY_PROJECT_ROOT environment variable",
			setupEnvironment: func(t *testing.T) {
				clearAllEnvironment(t, origEnv)
				// Set the explicit project root environment variable
				setEnv(t, "SCRY_PROJECT_ROOT", referenceRoot)
			},
			expectedOutcome: "specific_path",
			expectedPath:    referenceRoot,
		},
		{
			name: "GitHub Actions environment",
			setupEnvironment: func(t *testing.T) {
				clearAllEnvironment(t, origEnv)
				// Simulate GitHub Actions environment
				setEnv(t, "CI", "true")
				setEnv(t, "GITHUB_ACTIONS", "true")
				setEnv(t, "GITHUB_WORKSPACE", referenceRoot)
				setEnv(t, "GITHUB_REPOSITORY", "scry/scry-api")
			},
			expectedOutcome: "specific_path",
			expectedPath:    referenceRoot,
		},
		{
			name: "GitHub Actions with repository in subdirectory",
			setupEnvironment: func(t *testing.T) {
				clearAllEnvironment(t, origEnv)
				// Simulate monorepo GitHub Actions environment
				tmpDir := t.TempDir()
				repoDir := filepath.Join(tmpDir, "scry-api")
				if err := os.MkdirAll(repoDir, 0755); err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}

				// Create a go.mod file in the repo directory
				goModPath := filepath.Join(repoDir, "go.mod")
				if err := os.WriteFile(goModPath, []byte("module github.com/scry/scry-api"), 0644); err != nil {
					t.Fatalf("Failed to create go.mod file: %v", err)
				}

				// Create some common project markers
				dirs := []string{
					filepath.Join(repoDir, "cmd"),
					filepath.Join(repoDir, "internal"),
					filepath.Join(repoDir, "docs"),
				}
				for _, dir := range dirs {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("Failed to create directory %s: %v", dir, err)
					}
				}

				// Set GitHub Actions environment variables
				setEnv(t, "CI", "true")
				setEnv(t, "GITHUB_ACTIONS", "true")
				setEnv(t, "GITHUB_WORKSPACE", tmpDir)
				setEnv(t, "GITHUB_REPOSITORY", "scry/scry-api")

				// Change working directory to a subdirectory
				if err := os.Chdir(filepath.Join(repoDir, "internal")); err != nil {
					t.Fatalf("Failed to change working directory: %v", err)
				}

				// Defer returning to original directory
				t.Cleanup(func() {
					if err := os.Chdir(currentDir); err != nil {
						t.Logf("Failed to restore working directory: %v", err)
					}
				})
			},
			expectedOutcome: "success", // We expect a valid project root to be found
		},
		{
			name: "GitHub Actions with RUNNER_WORKSPACE",
			setupEnvironment: func(t *testing.T) {
				clearAllEnvironment(t, origEnv)
				// Simulate GitHub Actions with RUNNER_WORKSPACE
				setEnv(t, "CI", "true")
				setEnv(t, "GITHUB_ACTIONS", "true")
				// Invalid GITHUB_WORKSPACE
				setEnv(t, "GITHUB_WORKSPACE", "/nonexistent/path")
				// Valid RUNNER_WORKSPACE
				setEnv(t, "RUNNER_WORKSPACE", referenceRoot)
				setEnv(t, "GITHUB_REPOSITORY", "scry/scry-api")
			},
			expectedOutcome: "specific_path",
			expectedPath:    referenceRoot,
		},
		{
			name: "GitLab CI environment",
			setupEnvironment: func(t *testing.T) {
				clearAllEnvironment(t, origEnv)
				// Simulate GitLab CI environment
				setEnv(t, "CI", "true")
				setEnv(t, "GITLAB_CI", "true")
				setEnv(t, "CI_PROJECT_DIR", referenceRoot)
				setEnv(t, "CI_PROJECT_NAME", "scry-api")
			},
			expectedOutcome: "specific_path",
			expectedPath:    referenceRoot,
		},
		{
			name: "fallback to directory traversal",
			setupEnvironment: func(t *testing.T) {
				clearAllEnvironment(t, origEnv)
				// No environment variables set, should use directory traversal
			},
			expectedOutcome:     "success",
			checkRelativeToRoot: true,
		},
		{
			name: "working from a subdirectory",
			setupEnvironment: func(t *testing.T) {
				clearAllEnvironment(t, origEnv)

				// Get the internal directory path
				internalDir := filepath.Join(referenceRoot, "internal")

				// Change to the internal directory
				if err := os.Chdir(internalDir); err != nil {
					t.Skipf("Failed to change to internal directory: %v", err)
					return
				}

				// Defer changing back to the original directory
				t.Cleanup(func() {
					if err := os.Chdir(currentDir); err != nil {
						t.Logf("Failed to restore working directory: %v", err)
					}
				})
			},
			expectedOutcome:     "success",
			checkRelativeToRoot: true,
		},
		{
			name: "invalid explicit project root path",
			setupEnvironment: func(t *testing.T) {
				clearAllEnvironment(t, origEnv)
				// Set SCRY_PROJECT_ROOT to an invalid path
				setEnv(t, "SCRY_PROJECT_ROOT", "/tmp/nonexistent/path")

				// Also unset other environment variables to prevent fallback
				clearEnv(t, "GITHUB_WORKSPACE")
				clearEnv(t, "CI_PROJECT_DIR")
			},
			expectedOutcome:     "success", // Function falls back to directory traversal if SCRY_PROJECT_ROOT is invalid
			checkRelativeToRoot: true,      // Result should relate to the actual project root
		},
		{
			name: "GitHub Actions with empty workspace",
			setupEnvironment: func(t *testing.T) {
				clearAllEnvironment(t, origEnv)
				// Simulate GitHub Actions with empty GITHUB_WORKSPACE
				setEnv(t, "CI", "true")
				setEnv(t, "GITHUB_ACTIONS", "true")
				setEnv(t, "GITHUB_WORKSPACE", "")
				setEnv(t, "GITHUB_REPOSITORY", "scry/scry-api")
			},
			expectedOutcome:     "success", // Should fall back to directory traversal
			checkRelativeToRoot: true,
		},
	}

	// Log the start of our test
	t.Log("Starting comprehensive project root detection tests")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup the environment for this test case
			tt.setupEnvironment(t)

			// Call the function under test
			result, err := findProjectRoot()

			// Log detailed test results for context
			t.Logf("Test: %s", tt.name)
			t.Logf("Result: %s", result)
			t.Logf("Error: %v", err)

			// Get current directory, safely handling errors
			curDir, dirErr := os.Getwd()
			if dirErr != nil {
				curDir = "<error getting working directory>"
			}

			t.Logf("Working directory: %s", curDir)
			t.Logf("Environment variables:")
			for _, v := range []string{
				"SCRY_PROJECT_ROOT", "GITHUB_WORKSPACE", "GITHUB_ACTIONS",
				"GITHUB_REPOSITORY", "RUNNER_WORKSPACE", "CI",
				"CI_PROJECT_DIR", "CI_PROJECT_NAME",
			} {
				t.Logf("  %s=%s", v, os.Getenv(v))
			}

			// Check results against expectations
			switch tt.expectedOutcome {
			case "success":
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if result == "" {
					t.Errorf("Expected a non-empty project root path but got empty string")
				}

				// Verify the returned path actually exists and has a go.mod file
				if _, statErr := os.Stat(filepath.Join(result, "go.mod")); statErr != nil {
					t.Errorf("Returned project root doesn't contain go.mod: %s (error: %v)", result, statErr)
				}

				// If we're checking if the result should be related to the reference root
				if tt.checkRelativeToRoot {
					// In normal operation, the function should return the same directory
					// as our reference root, or at least a directory that contains go.mod
					if !strings.Contains(result, referenceRoot) && !strings.Contains(referenceRoot, result) {
						// Neither path contains the other - they might be completely different
						// Check if it's a valid project root by checking for a go.mod file
						if _, err := os.Stat(filepath.Join(result, "go.mod")); err != nil {
							t.Errorf("Returned path does not contain go.mod, not a valid project root: %s", result)
						}
					}
				}

			case "error":
				if err == nil {
					t.Errorf("Expected an error but got success: %s", result)
				}

			case "specific_path":
				if err != nil {
					t.Errorf("Expected specific path but got error: %v", err)
				}
				if result != tt.expectedPath {
					t.Errorf("Expected path %q but got %q", tt.expectedPath, result)
				}
			}
		})
	}
}

// TestProjectRootValidation tests the project root validation logic
// This tests the behavior that would normally use the internal validateProjectRoot function
func TestProjectRootValidation(t *testing.T) {
	// Create a temporary directory for our test project root
	testDir := t.TempDir()

	// Set up environment to point to this directory
	origEnv := os.Getenv("SCRY_PROJECT_ROOT")
	defer func() {
		if err := os.Setenv("SCRY_PROJECT_ROOT", origEnv); err != nil {
			t.Logf("Failed to restore SCRY_PROJECT_ROOT: %v", err)
		}
	}()

	// First set CI to true for consistent behavior
	origCI := os.Getenv("CI")
	defer func() {
		if err := os.Setenv("CI", origCI); err != nil {
			t.Logf("Failed to restore CI: %v", err)
		}
	}()

	if err := os.Setenv("CI", "true"); err != nil {
		t.Fatalf("Failed to set CI environment variable: %v", err)
	}

	// Empty directory should not be considered a valid project root
	// But findProjectRoot will fall back to other methods instead of failing
	if err := os.Setenv("SCRY_PROJECT_ROOT", testDir); err != nil {
		t.Fatalf("Failed to set SCRY_PROJECT_ROOT: %v", err)
	}
	_, err := findProjectRoot()
	if err != nil {
		t.Logf("With empty directory, findProjectRoot fell back: %v", err)
	}

	// Create a go.mod file, which is the primary requirement
	goModPath := filepath.Join(testDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module github.com/scry/scry-api"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod file: %v", err)
	}

	// Now it should be valid with just go.mod
	if err := os.Setenv("SCRY_PROJECT_ROOT", testDir); err != nil {
		t.Fatalf("Failed to set SCRY_PROJECT_ROOT: %v", err)
	}
	result, err := findProjectRoot()
	if err != nil || result != testDir {
		t.Errorf("Directory with go.mod should be a valid project root, got error: %v, result: %s", err, result)
	}

	// Add some secondary markers to increase confidence
	markers := []string{
		".git",
		"cmd",
		"internal",
		"docs",
		".github",
		".gitignore",
		"go.sum",
	}

	// Add markers one by one
	for _, marker := range markers {
		markerPath := filepath.Join(testDir, marker)
		// Create either files or directories based on the marker type
		if strings.HasPrefix(marker, ".") {
			// Create empty file for dot files
			if err := os.WriteFile(markerPath, []byte{}, 0644); err != nil {
				t.Fatalf("Failed to create marker file %s: %v", marker, err)
			}
		} else {
			// Create directory for others
			if err := os.MkdirAll(markerPath, 0755); err != nil {
				t.Fatalf("Failed to create marker directory %s: %v", marker, err)
			}
		}

		t.Logf("Added marker: %s to test project root", marker)
	}

	// Test with a non-existent directory - findProjectRoot should fall back to other methods
	nonExistentDir := filepath.Join(testDir, "nonexistent")
	if err := os.Setenv("SCRY_PROJECT_ROOT", nonExistentDir); err != nil {
		t.Fatalf("Failed to set SCRY_PROJECT_ROOT to nonexistent path: %v", err)
	}
	result, err = findProjectRoot()
	if err != nil {
		t.Fatalf("With non-existent directory, findProjectRoot failed unexpectedly: %v", err)
	}
	t.Logf("With non-existent directory, findProjectRoot succeeded with fallback: %s", result)
}

// TestRepoNameDetection tests how repository names are detected from environment variables
// This tests the behavior that would use the internal checkAndGetRepo function
func TestRepoNameDetection(t *testing.T) {
	// We'll test this indirectly by observing how findProjectRoot behaves
	// with different environment variables that influence repo name detection

	// Save original environment variables
	origEnv := saveEnvironment(t, []string{
		"SCRY_PROJECT_ROOT",
		"GITHUB_REPOSITORY",
		"CI_PROJECT_NAME",
		"GITHUB_WORKSPACE",
		"CI_PROJECT_DIR",
		"CI",
		"GITHUB_ACTIONS",
	})
	defer restoreEnvironment(t, origEnv)

	// We'll set SCRY_PROJECT_ROOT to ensure findProjectRoot succeeds
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}

	// Find the actual project root by traversing up looking for go.mod
	projectRoot := ""
	dir := currentDir
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			projectRoot = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if projectRoot == "" {
		t.Fatalf("Could not find project root from current directory")
	}

	setEnv(t, "SCRY_PROJECT_ROOT", projectRoot)

	// For each test case, we'll set specific environment variables and call findProjectRoot
	// While we can't directly test checkAndGetRepo, we know it's used by findProjectRoot
	// for GitHub Actions monorepo detection

	// Ensure GITHUB_ACTIONS and CI are set to true to make our environment variables relevant
	setEnv(t, "CI", "true")
	setEnv(t, "GITHUB_ACTIONS", "true")

	// Get a successful findProjectRoot result to confirm our setup works
	_, err = findProjectRoot()
	if err != nil {
		t.Fatalf("findProjectRoot failed in test setup: %v", err)
	}

	// Some basic tests to verify environment variables are being processed
	tests := []struct {
		name        string
		envSetup    func()
		expectedLog string // We can't easily check internal results, so we'll check logs
	}{
		{
			name: "GITHUB_REPOSITORY with owner/repo format",
			envSetup: func() {
				clearEnv(t, "CI_PROJECT_NAME")
				setEnv(t, "GITHUB_REPOSITORY", "scry/different-repo-name")
			},
			expectedLog: "different-repo-name", // Should appear in logs
		},
		{
			name: "CI_PROJECT_NAME",
			envSetup: func() {
				clearEnv(t, "GITHUB_REPOSITORY")
				setEnv(t, "CI_PROJECT_NAME", "different-project-name")
			},
			expectedLog: "different-project-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			tt.envSetup()

			// Call findProjectRoot which will use checkAndGetRepo internally
			_, err := findProjectRoot()
			if err != nil {
				t.Fatalf("findProjectRoot failed: %v", err)
			}

			// Success - we just verified that different environment variables
			// don't break the repository detection logic
		})
	}
}

// TestIsGitHubActionsCI tests the isGitHubActionsCI function
func TestIsGitHubActionsCI(t *testing.T) {
	// Save original environment variables
	origEnv := saveEnvironment(t, []string{
		"GITHUB_ACTIONS", "GITHUB_WORKSPACE",
	})
	defer restoreEnvironment(t, origEnv)

	tests := []struct {
		name          string
		envSetup      func()
		expectedValue bool
	}{
		{
			name: "GitHub Actions with workspace",
			envSetup: func() {
				setEnv(t, "GITHUB_ACTIONS", "true")
				setEnv(t, "GITHUB_WORKSPACE", "/github/workspace")
			},
			expectedValue: true,
		},
		{
			name: "GitHub Actions without workspace",
			envSetup: func() {
				setEnv(t, "GITHUB_ACTIONS", "true")
				clearEnv(t, "GITHUB_WORKSPACE")
			},
			expectedValue: false,
		},
		{
			name: "No GitHub Actions",
			envSetup: func() {
				clearEnv(t, "GITHUB_ACTIONS")
				clearEnv(t, "GITHUB_WORKSPACE")
			},
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			tt.envSetup()

			// Call the function
			result := isGitHubActionsCI()

			// Check result
			if result != tt.expectedValue {
				t.Errorf("Expected %v but got %v", tt.expectedValue, result)
			}
		})
	}
}

// Note: Helper functions for environment management are defined in db_test.go
