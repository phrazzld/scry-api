package config

import (
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestLoadWithLegacyEnvironmentVariables(t *testing.T) {
	// Create a buffer for log output to capture warnings
	var logBuffer strings.Builder
	logHandler := slog.NewTextHandler(&logBuffer, nil)
	logger := slog.New(logHandler)

	// Save the current environment
	savedEnv := map[string]string{
		"SCRY_DATABASE_URL":     os.Getenv("SCRY_DATABASE_URL"),
		"DATABASE_URL":          os.Getenv("DATABASE_URL"),
		"SCRY_SERVER_LOG_LEVEL": os.Getenv("SCRY_SERVER_LOG_LEVEL"),
		"LOG_LEVEL":             os.Getenv("LOG_LEVEL"),
		"SCRY_AUTH_JWT_SECRET":  os.Getenv("SCRY_AUTH_JWT_SECRET"),
	}

	// Restore the environment after the test
	defer func() {
		for key, value := range savedEnv {
			if value == "" {
				if err := os.Unsetenv(key); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", key, err)
				}
			} else {
				if err := os.Setenv(key, value); err != nil {
					t.Logf("Failed to restore environment variable %s: %v", key, err)
				}
			}
		}
	}()

	// Test cases with various environment variable configurations
	tests := []struct {
		name             string
		envVars          map[string]string
		expectWarning    bool
		expectedLogLevel string
		expectedDBURL    string
	}{
		{
			name: "Standard environment variables",
			envVars: map[string]string{
				"SCRY_DATABASE_URL":     "postgres://standard:pwd@localhost:5432/standard",
				"SCRY_SERVER_LOG_LEVEL": "debug",
				"SCRY_AUTH_JWT_SECRET":  "standard-jwt-secret-at-least-32-chars-long",
			},
			expectWarning:    false,
			expectedLogLevel: "debug",
			expectedDBURL:    "postgres://standard:pwd@localhost:5432/standard",
		},
		{
			name: "Legacy environment variables",
			envVars: map[string]string{
				"DATABASE_URL":         "postgres://legacy:pwd@localhost:5432/legacy",
				"LOG_LEVEL":            "error",
				"SCRY_AUTH_JWT_SECRET": "legacy-jwt-secret-at-least-32-chars-long",
			},
			expectWarning:    true,
			expectedLogLevel: "error",
			expectedDBURL:    "postgres://legacy:pwd@localhost:5432/legacy",
		},
		{
			name: "Mixed standard and legacy (standard takes precedence)",
			envVars: map[string]string{
				"SCRY_DATABASE_URL":     "postgres://standard:pwd@localhost:5432/standard",
				"DATABASE_URL":          "postgres://legacy:pwd@localhost:5432/legacy",
				"SCRY_SERVER_LOG_LEVEL": "info",
				"LOG_LEVEL":             "debug",
				"SCRY_AUTH_JWT_SECRET":  "mixed-jwt-secret-at-least-32-chars-long",
			},
			expectWarning:    false,
			expectedLogLevel: "info",
			expectedDBURL:    "postgres://standard:pwd@localhost:5432/standard",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear environment variables that might interfere
			for key := range savedEnv {
				if err := os.Unsetenv(key); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", key, err)
				}
			}

			// Set environment variables for this test
			for key, value := range tc.envVars {
				if err := os.Setenv(key, value); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", key, err)
				}
			}

			// Reset log buffer
			logBuffer.Reset()

			// Load configuration
			cfg, err := LoadWithLogger(logger)
			if err != nil {
				t.Fatalf("LoadWithLogger() error = %v", err)
			}

			// Check if configuration was loaded correctly
			if cfg.Server.LogLevel != tc.expectedLogLevel {
				t.Errorf("Expected LogLevel = %s, got %s", tc.expectedLogLevel, cfg.Server.LogLevel)
			}

			if cfg.Database.URL != tc.expectedDBURL {
				t.Errorf("Expected Database.URL = %s, got %s", tc.expectedDBURL, cfg.Database.URL)
			}

			// Check for deprecation warnings
			logOutput := logBuffer.String()
			hasWarning := strings.Contains(logOutput, "legacy environment variable")

			if tc.expectWarning && !hasWarning {
				t.Errorf("Expected deprecation warning but none was logged")
			}

			if !tc.expectWarning && hasWarning {
				t.Errorf("Unexpected deprecation warning was logged: %s", logOutput)
			}
		})
	}
}

func TestLoadFailsWithMissingRequiredConfig(t *testing.T) {
	// Save the current environment
	savedEnv := map[string]string{
		"SCRY_AUTH_JWT_SECRET": os.Getenv("SCRY_AUTH_JWT_SECRET"),
		"SCRY_DATABASE_URL":    os.Getenv("SCRY_DATABASE_URL"),
		"DATABASE_URL":         os.Getenv("DATABASE_URL"),
	}

	// Restore the environment after the test
	defer func() {
		for key, value := range savedEnv {
			if value == "" {
				if err := os.Unsetenv(key); err != nil {
					t.Logf("Failed to unset environment variable %s: %v", key, err)
				}
			} else {
				if err := os.Setenv(key, value); err != nil {
					t.Logf("Failed to restore environment variable %s: %v", key, err)
				}
			}
		}
	}()

	// Clear required environment variables
	for key := range savedEnv {
		if err := os.Unsetenv(key); err != nil {
			t.Logf("Failed to unset environment variable %s: %v", key, err)
		}
	}

	// Attempt to load with missing required values
	_, err := Load()
	if err == nil {
		t.Errorf("Expected Load() to fail with missing required values, but it succeeded")
	}

	// The error should mention validation
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestLoadConfigFromYAML(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config.*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Failed to remove temp file %s: %v", tempFile.Name(), err)
		}
	}()

	// Write sample configuration to the file
	configContent := `
server:
  port: 9090
  log_level: debug
database:
  url: postgres://yaml:pwd@localhost:5432/yamldb
auth:
  jwt_secret: yaml-jwt-secret-at-least-32-chars-long
  bcrypt_cost: 12
llm:
  gemini_api_key: yaml-test-key
  model_name: gemini-2.0-flash
  prompt_template_path: prompts/test_template.txt
task:
  worker_count: 4
  queue_size: 200
  stuck_task_age_minutes: 15
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Save current environment and working directory
	savedEnv := map[string]string{
		"SCRY_DATABASE_URL":    os.Getenv("SCRY_DATABASE_URL"),
		"DATABASE_URL":         os.Getenv("DATABASE_URL"),
		"SCRY_SERVER_PORT":     os.Getenv("SCRY_SERVER_PORT"),
		"SCRY_AUTH_JWT_SECRET": os.Getenv("SCRY_AUTH_JWT_SECRET"),
	}
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Move to the directory containing the config file
	tempDir := strings.TrimSuffix(tempFile.Name(), strings.TrimPrefix(tempFile.Name(), os.TempDir()))
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Rename the temp file to config.yaml
	configPath := tempDir + "/config.yaml"
	if err := os.Rename(tempFile.Name(), configPath); err != nil {
		t.Fatalf("Failed to rename config file: %v", err)
	}
	defer func() {
		if err := os.Remove(configPath); err != nil {
			t.Logf("Failed to remove config file %s: %v", configPath, err)
		}
	}()

	// Clear environment variables to ensure we're loading from the file
	for key := range savedEnv {
		if err := os.Unsetenv(key); err != nil {
			t.Logf("Failed to unset environment variable %s: %v", key, err)
		}
	}

	// Create a logger to capture output
	var logBuffer strings.Builder
	logHandler := slog.NewTextHandler(&logBuffer, nil)
	logger := slog.New(logHandler)

	// Load configuration
	cfg, err := LoadWithLogger(logger)
	if err != nil {
		t.Fatalf("LoadWithLogger() error = %v", err)
	}

	// Restore working directory and environment
	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("Failed to restore working directory: %v", err)
	}
	for key, value := range savedEnv {
		if value == "" {
			if err := os.Unsetenv(key); err != nil {
				t.Logf("Failed to unset environment variable %s: %v", key, err)
			}
		} else {
			if err := os.Setenv(key, value); err != nil {
				t.Logf("Failed to restore environment variable %s: %v", key, err)
			}
		}
	}

	// Verify the configuration was loaded from the file
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected Server.Port = 9090, got %d", cfg.Server.Port)
	}
	if cfg.Database.URL != "postgres://yaml:pwd@localhost:5432/yamldb" {
		t.Errorf("Expected Database.URL from YAML, got %s", cfg.Database.URL)
	}
	if cfg.Auth.BCryptCost != 12 {
		t.Errorf("Expected Auth.BCryptCost = 12, got %d", cfg.Auth.BCryptCost)
	}
	if cfg.Task.WorkerCount != 4 {
		t.Errorf("Expected Task.WorkerCount = 4, got %d", cfg.Task.WorkerCount)
	}
}
