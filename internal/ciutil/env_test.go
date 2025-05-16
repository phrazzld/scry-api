package ciutil

import (
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestIsCI(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "No CI env vars",
			envVars:  map[string]string{},
			expected: false,
		},
		{
			name:     "Generic CI",
			envVars:  map[string]string{EnvCI: "true"},
			expected: true,
		},
		{
			name:     "GitHub Actions",
			envVars:  map[string]string{EnvGitHubActions: "true"},
			expected: true,
		},
		{
			name:     "GitLab CI",
			envVars:  map[string]string{EnvGitLabCI: "true"},
			expected: true,
		},
		{
			name:     "Jenkins",
			envVars:  map[string]string{EnvJenkinsURL: "https://jenkins.example.com"},
			expected: true,
		},
		{
			name:     "Travis CI",
			envVars:  map[string]string{EnvTravisCI: "true"},
			expected: true,
		},
		{
			name:     "Circle CI",
			envVars:  map[string]string{EnvCircleCI: "true"},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Save current environment
			savedEnv := map[string]string{}
			for k := range tc.envVars {
				savedEnv[k] = os.Getenv(k)
			}

			// Set up test environment
			for k, v := range tc.envVars {
				os.Setenv(k, v)
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

			// Test the function
			if got := IsCI(); got != tc.expected {
				t.Errorf("IsCI() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestIsGitHubActions(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "No GitHub Actions env vars",
			envVars:  map[string]string{},
			expected: false,
		},
		{
			name:     "GitHub Actions flag only",
			envVars:  map[string]string{EnvGitHubActions: "true"},
			expected: false,
		},
		{
			name:     "GitHub Workspace only",
			envVars:  map[string]string{EnvGitHubWorkspace: "/github/workspace"},
			expected: false,
		},
		{
			name: "Both GitHub Actions and Workspace",
			envVars: map[string]string{
				EnvGitHubActions:   "true",
				EnvGitHubWorkspace: "/github/workspace",
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Save current environment
			savedEnv := map[string]string{}
			for k := range tc.envVars {
				savedEnv[k] = os.Getenv(k)
			}

			// Set up test environment
			for k, v := range tc.envVars {
				os.Setenv(k, v)
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

			// Test the function
			if got := IsGitHubActions(); got != tc.expected {
				t.Errorf("IsGitHubActions() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestGetEnvWithFallbacks(t *testing.T) {
	// Create a test logger that captures warnings
	var logBuffer strings.Builder
	logHandler := slog.NewTextHandler(&logBuffer, nil)
	logger := slog.New(logHandler)

	tests := []struct {
		name             string
		envVars          map[string]string
		fallbacks        []string
		defaultValue     string
		expected         string
		expectDepWarning bool
	}{
		{
			name:             "No env vars set, use default",
			envVars:          map[string]string{},
			fallbacks:        []string{"PRIMARY_VAR", "LEGACY_VAR"},
			defaultValue:     "default_value",
			expected:         "default_value",
			expectDepWarning: false,
		},
		{
			name:             "Primary var set",
			envVars:          map[string]string{"PRIMARY_VAR": "primary_value"},
			fallbacks:        []string{"PRIMARY_VAR", "LEGACY_VAR"},
			defaultValue:     "default_value",
			expected:         "primary_value",
			expectDepWarning: false,
		},
		{
			name:             "Legacy var set, expect warning",
			envVars:          map[string]string{"LEGACY_VAR": "legacy_value"},
			fallbacks:        []string{"PRIMARY_VAR", "LEGACY_VAR"},
			defaultValue:     "default_value",
			expected:         "legacy_value",
			expectDepWarning: true,
		},
		{
			name: "Both vars set, primary takes precedence",
			envVars: map[string]string{
				"PRIMARY_VAR": "primary_value",
				"LEGACY_VAR":  "legacy_value",
			},
			fallbacks:        []string{"PRIMARY_VAR", "LEGACY_VAR"},
			defaultValue:     "default_value",
			expected:         "primary_value",
			expectDepWarning: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Save current environment
			savedEnv := map[string]string{}
			for k := range tc.envVars {
				savedEnv[k] = os.Getenv(k)
			}

			// Set up test environment
			for k, v := range tc.envVars {
				os.Setenv(k, v)
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

			// Reset the log buffer
			logBuffer.Reset()

			// Test the function
			got := GetEnvWithFallbacks(tc.fallbacks, tc.defaultValue, logger)
			if got != tc.expected {
				t.Errorf("GetEnvWithFallbacks() = %v, want %v", got, tc.expected)
			}

			// Check for warning log
			logOutput := logBuffer.String()
			hasWarning := strings.Contains(logOutput, "legacy")

			if tc.expectDepWarning && !hasWarning {
				t.Errorf("Expected deprecation warning but none was logged")
			} else if !tc.expectDepWarning && hasWarning {
				t.Errorf("Unexpected deprecation warning was logged: %s", logOutput)
			}
		})
	}
}

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Database URL",
			input:    "postgres://username:password@localhost:5432/dbname",
			expected: "postgres://username:****@localhost:5432/dbname",
		},
		{
			name:     "MySQL URL",
			input:    "mysql://dbuser:secret@dbhost:3306/mydb",
			expected: "mysql://dbuser:****@dbhost:3306/mydb",
		},
		{
			name:     "API Key",
			input:    "api_key_12345678901234",
			expected: "api_****1234",
		},
		{
			name:     "JWT Secret",
			input:    "jwt_secret_very_long_and_secure",
			expected: "jwt_****cure",
		},
		{
			name:     "Regular text",
			input:    "not_sensitive_data",
			expected: "not_sensitive_data",
		},
		{
			name:     "Short text",
			input:    "short",
			expected: "short",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MaskSensitiveValue(tc.input)
			if got != tc.expected {
				t.Errorf("MaskSensitiveValue() = %v, want %v", got, tc.expected)
			}
		})
	}
}
