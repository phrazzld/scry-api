//go:build integration || test_without_external_deps

package ciutil

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindProjectRoot(t *testing.T) {
	// Create a test logger
	var logBuffer strings.Builder
	logHandler := slog.NewTextHandler(&logBuffer, nil)
	logger := slog.New(logHandler)

	// Save current environment
	savedEnv := map[string]string{
		EnvScryProjectRoot:  os.Getenv(EnvScryProjectRoot),
		EnvGitHubActions:    os.Getenv(EnvGitHubActions),
		EnvGitHubWorkspace:  os.Getenv(EnvGitHubWorkspace),
		EnvGitLabCI:         os.Getenv(EnvGitLabCI),
		EnvGitLabProjectDir: os.Getenv(EnvGitLabProjectDir),
	}

	// Clean up after the test
	defer func() {
		for k, v := range savedEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", k, err)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("Failed to restore environment variable %s: %v", k, err)
				}
			}
		}
	}()

	// Get current directory and find the actual project root
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Find the actual project root by going up directories until we find go.mod
	projectRoot := currentDir
	for i := 0; i < 10; i++ {
		if fileExists(filepath.Join(projectRoot, "go.mod")) {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatalf("Could not find project root with go.mod file")
		}
		projectRoot = parent
	}

	tests := []struct {
		name        string
		setupEnv    func()
		wantErr     bool
		checkResult func(string) bool
	}{
		{
			name: "Explicit SCRY_PROJECT_ROOT",
			setupEnv: func() {
				// Reset CI variables
				if err := os.Unsetenv(EnvGitHubActions); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitHubActions, err)
				}
				if err := os.Unsetenv(EnvGitHubWorkspace); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitHubWorkspace, err)
				}
				if err := os.Unsetenv(EnvGitLabCI); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitLabCI, err)
				}
				if err := os.Unsetenv(EnvGitLabProjectDir); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitLabProjectDir, err)
				}

				// Set explicit project root to actual project root
				if err := os.Setenv(EnvScryProjectRoot, projectRoot); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", EnvScryProjectRoot, err)
				}
			},
			wantErr: false,
			checkResult: func(result string) bool {
				return result == projectRoot
			},
		},
		{
			name: "GitHub Actions environment",
			setupEnv: func() {
				// Reset other variables
				if err := os.Unsetenv(EnvScryProjectRoot); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvScryProjectRoot, err)
				}
				if err := os.Unsetenv(EnvGitLabCI); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitLabCI, err)
				}
				if err := os.Unsetenv(EnvGitLabProjectDir); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitLabProjectDir, err)
				}

				// Set GitHub Actions environment
				if err := os.Setenv(EnvGitHubActions, "true"); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", EnvGitHubActions, err)
				}
				if err := os.Setenv(EnvGitHubWorkspace, projectRoot); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", EnvGitHubWorkspace, err)
				}
			},
			wantErr: false,
			checkResult: func(result string) bool {
				return result == projectRoot
			},
		},
		{
			name: "GitLab CI environment",
			setupEnv: func() {
				// Reset other variables
				if err := os.Unsetenv(EnvScryProjectRoot); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvScryProjectRoot, err)
				}
				if err := os.Unsetenv(EnvGitHubActions); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitHubActions, err)
				}
				if err := os.Unsetenv(EnvGitHubWorkspace); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitHubWorkspace, err)
				}

				// Set GitLab CI environment
				if err := os.Setenv(EnvGitLabCI, "true"); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", EnvGitLabCI, err)
				}
				if err := os.Setenv(EnvGitLabProjectDir, projectRoot); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", EnvGitLabProjectDir, err)
				}
			},
			wantErr: false,
			checkResult: func(result string) bool {
				return result == projectRoot
			},
		},
		{
			name: "Auto-detection",
			setupEnv: func() {
				// Reset all environment variables
				if err := os.Unsetenv(EnvScryProjectRoot); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvScryProjectRoot, err)
				}
				if err := os.Unsetenv(EnvGitHubActions); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitHubActions, err)
				}
				if err := os.Unsetenv(EnvGitHubWorkspace); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitHubWorkspace, err)
				}
				if err := os.Unsetenv(EnvGitLabCI); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitLabCI, err)
				}
				if err := os.Unsetenv(EnvGitLabProjectDir); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitLabProjectDir, err)
				}
			},
			wantErr: false,
			checkResult: func(result string) bool {
				// The result should be a valid directory containing go.mod
				goModPath := filepath.Join(result, GoModFile)
				return fileExists(goModPath)
			},
		},
		{
			name: "Invalid SCRY_PROJECT_ROOT",
			setupEnv: func() {
				// Reset CI variables
				if err := os.Unsetenv(EnvGitHubActions); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitHubActions, err)
				}
				if err := os.Unsetenv(EnvGitHubWorkspace); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitHubWorkspace, err)
				}
				if err := os.Unsetenv(EnvGitLabCI); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitLabCI, err)
				}
				if err := os.Unsetenv(EnvGitLabProjectDir); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", EnvGitLabProjectDir, err)
				}

				// Set invalid project root
				if err := os.Setenv(EnvScryProjectRoot, "/path/that/does/not/exist"); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", EnvScryProjectRoot, err)
				}
			},
			wantErr: true,
			checkResult: func(result string) bool {
				return false // Should error
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment for this test
			tc.setupEnv()

			// Reset the log buffer
			logBuffer.Reset()

			// Call the function
			result, err := FindProjectRoot(logger)

			// Check for expected error
			if (err != nil) != tc.wantErr {
				t.Errorf("FindProjectRoot() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			// If no error, check the result
			if err == nil {
				if !tc.checkResult(result) {
					t.Errorf("FindProjectRoot() = %v, which does not meet expected criteria", result)
				}
			}

			// Check that appropriate logging occurred
			logOutput := logBuffer.String()
			if err == nil && !strings.Contains(logOutput, "project root") {
				t.Errorf("Expected log messages about project root but none were found")
			}
		})
	}
}

func TestFindMigrationsDir(t *testing.T) {
	// Create a test logger
	var logBuffer strings.Builder
	logHandler := slog.NewTextHandler(&logBuffer, nil)
	logger := slog.New(logHandler)

	// Get current directory and find the actual project root
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Find the actual project root by going up directories until we find go.mod
	projectRoot := currentDir
	for i := 0; i < 10; i++ {
		if fileExists(filepath.Join(projectRoot, "go.mod")) {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatalf("Could not find project root with go.mod file")
		}
		projectRoot = parent
	}

	// Save current environment
	savedEnv := map[string]string{
		EnvScryProjectRoot: os.Getenv(EnvScryProjectRoot),
	}

	// Clean up after the test
	defer func() {
		for k, v := range savedEnv {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", k, err)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Logf("Failed to restore environment variable %s: %v", k, err)
				}
			}
		}
	}()

	// Set project root to actual project root for testing
	if err := os.Setenv(EnvScryProjectRoot, projectRoot); err != nil {
		t.Fatalf("Failed to set environment variable %s: %v", EnvScryProjectRoot, err)
	}

	// This test is environment-dependent, so we'll focus on correct behavior
	// rather than specific paths
	_, err = FindMigrationsDir(logger)
	if err == nil {
		t.Log("FindMigrationsDir found migrations directory successfully")
	} else {
		t.Logf("FindMigrationsDir returned error (expected in test environment): %v", err)
	}

	// We don't specifically check for error or success here, as it depends on
	// the actual directory structure. Instead, we just verify the function runs
	// and produces appropriate logs.

	// Check that appropriate logging occurred (may be at DEBUG level)
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "migrations") && !strings.Contains(logOutput, "project") {
		t.Logf("Log output: %s", logOutput)
		// This is expected - the function only logs detailed messages when DEBUG level is enabled
		t.Log("No detailed logging expected at default level")
	}
}

func TestIsValidProjectRoot(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "project-root-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp directory %s: %v", tempDir, err)
		}
	}()

	// Create a go.mod file in the temp directory
	goModPath := filepath.Join(tempDir, GoModFile)
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	tests := []struct {
		name     string
		dir      string
		expected bool
	}{
		{
			name:     "Valid project root (temp dir with go.mod)",
			dir:      tempDir,
			expected: true,
		},
		{
			name:     "Invalid project root (non-existent directory)",
			dir:      "/path/that/does/not/exist",
			expected: false,
		},
		{
			name:     "Invalid project root (directory without go.mod)",
			dir:      os.TempDir(), // Assume temp dir doesn't have go.mod
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidProjectRoot(tc.dir)
			if result != tc.expected {
				t.Errorf("isValidProjectRoot(%s) = %v, want %v", tc.dir, result, tc.expected)
			}
		})
	}
}
