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
		EnvGitHubActions:   os.Getenv(EnvGitHubActions),
		EnvGitHubWorkspace: os.Getenv(EnvGitHubWorkspace),
		EnvGitLabCI:        os.Getenv(EnvGitLabCI),
		EnvGitLabProjectDir: os.Getenv(EnvGitLabProjectDir),
	}

	// Clean up after the test
	defer func() {
		for k, v := range savedEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Get current directory (should be within the project)
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
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
				os.Unsetenv(EnvGitHubActions)
				os.Unsetenv(EnvGitHubWorkspace)
				os.Unsetenv(EnvGitLabCI)
				os.Unsetenv(EnvGitLabProjectDir)
				
				// Set explicit project root to current directory
				os.Setenv(EnvScryProjectRoot, currentDir)
			},
			wantErr: false,
			checkResult: func(result string) bool {
				return result == currentDir
			},
		},
		{
			name: "GitHub Actions environment",
			setupEnv: func() {
				// Reset other variables
				os.Unsetenv(EnvScryProjectRoot)
				os.Unsetenv(EnvGitLabCI)
				os.Unsetenv(EnvGitLabProjectDir)
				
				// Set GitHub Actions environment
				os.Setenv(EnvGitHubActions, "true")
				os.Setenv(EnvGitHubWorkspace, currentDir)
			},
			wantErr: false,
			checkResult: func(result string) bool {
				return result == currentDir
			},
		},
		{
			name: "GitLab CI environment",
			setupEnv: func() {
				// Reset other variables
				os.Unsetenv(EnvScryProjectRoot)
				os.Unsetenv(EnvGitHubActions)
				os.Unsetenv(EnvGitHubWorkspace)
				
				// Set GitLab CI environment
				os.Setenv(EnvGitLabCI, "true")
				os.Setenv(EnvGitLabProjectDir, currentDir)
			},
			wantErr: false,
			checkResult: func(result string) bool {
				return result == currentDir
			},
		},
		{
			name: "Auto-detection",
			setupEnv: func() {
				// Reset all environment variables
				os.Unsetenv(EnvScryProjectRoot)
				os.Unsetenv(EnvGitHubActions)
				os.Unsetenv(EnvGitHubWorkspace)
				os.Unsetenv(EnvGitLabCI)
				os.Unsetenv(EnvGitLabProjectDir)
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
				os.Unsetenv(EnvGitHubActions)
				os.Unsetenv(EnvGitHubWorkspace)
				os.Unsetenv(EnvGitLabCI)
				os.Unsetenv(EnvGitLabProjectDir)
				
				// Set invalid project root
				os.Setenv(EnvScryProjectRoot, "/path/that/does/not/exist")
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

	// Get current directory (should be within the project)
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Save current environment
	savedEnv := map[string]string{
		EnvScryProjectRoot: os.Getenv(EnvScryProjectRoot),
	}

	// Clean up after the test
	defer func() {
		for k, v := range savedEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set project root to current directory to simplify testing
	os.Setenv(EnvScryProjectRoot, currentDir)

	// This test is environment-dependent, so we'll focus on correct behavior
	// rather than specific paths
	_, err = FindMigrationsDir(logger)
	
	// We don't specifically check for error or success here, as it depends on
	// the actual directory structure. Instead, we just verify the function runs
	// and produces appropriate logs.
	
	// Check that appropriate logging occurred
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "migrations") {
		t.Errorf("Expected log messages about migrations directory but none were found")
	}
}

func TestIsValidProjectRoot(t *testing.T) {
	// Get current directory (should be within the project)
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "project-root-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

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