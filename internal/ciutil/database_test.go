package ciutil

import (
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestGetTestDatabaseURL(t *testing.T) {
	// Create a test logger that captures output
	var logBuffer strings.Builder
	logHandler := slog.NewTextHandler(&logBuffer, nil)
	logger := slog.New(logHandler)

	tests := []struct {
		name     string
		envVars  map[string]string
		isCI     bool
		expected string
	}{
		{
			name:     "No database URL set",
			envVars:  map[string]string{},
			isCI:     false,
			expected: "",
		},
		{
			name:     "DATABASE_URL set (non-CI)",
			envVars:  map[string]string{EnvDatabaseURL: "postgres://user:password@localhost:5432/testdb"},
			isCI:     false,
			expected: "postgres://user:password@localhost:5432/testdb",
		},
		{
			name:     "SCRY_TEST_DB_URL set (non-CI)",
			envVars:  map[string]string{EnvScryTestDBURL: "postgres://test:test123@localhost:5432/testdb"},
			isCI:     false,
			expected: "postgres://test:test123@localhost:5432/testdb",
		},
		{
			name:     "SCRY_DATABASE_URL set (non-CI)",
			envVars:  map[string]string{EnvScryDatabaseURL: "postgres://app:app123@localhost:5432/appdb"},
			isCI:     false,
			expected: "postgres://app:app123@localhost:5432/appdb",
		},
		{
			name: "Multiple database URLs set (precedence order)",
			envVars: map[string]string{
				EnvDatabaseURL:     "postgres://primary:primary@localhost:5432/primary",
				EnvScryTestDBURL:   "postgres://test:test@localhost:5432/test",
				EnvScryDatabaseURL: "postgres://app:app@localhost:5432/app",
			},
			isCI:     false,
			expected: "postgres://primary:primary@localhost:5432/primary",
		},
		{
			name: "DATABASE_URL set (CI environment)",
			envVars: map[string]string{
				EnvDatabaseURL: "postgres://user:password@localhost:5432/testdb",
				EnvCI:          "true",
			},
			isCI:     true,
			expected: "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable",
		},
		{
			name:     "DATABASE_URL with non-standard host (CI environment)",
			envVars:  map[string]string{EnvDatabaseURL: "postgres://user:password@db:5432/testdb", EnvCI: "true"},
			isCI:     true,
			expected: "postgres://postgres:postgres@db:5432/testdb?sslmode=disable",
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
			got := GetTestDatabaseURL(logger)

			// In CI mode, we standardize the URL, so just check that the username and password are correct
			if tc.isCI && got != "" {
				if !strings.Contains(got, "postgres:postgres") {
					t.Errorf(
						"GetTestDatabaseURL() = %v, doesn't contain standard CI credentials",
						MaskSensitiveValue(got),
					)
				}
			} else if got != tc.expected {
				t.Errorf("GetTestDatabaseURL() = %v, want %v", MaskSensitiveValue(got), MaskSensitiveValue(tc.expected))
			}

			// Verify logging behavior
			logOutput := logBuffer.String()

			// Check for deprecation warnings for non-standardized vars
			if tc.envVars[EnvDatabaseURL] != "" && !strings.Contains(logOutput, "non-standardized") {
				t.Errorf("Expected deprecation warning for DATABASE_URL but none was logged")
			}

			if tc.envVars[EnvScryDatabaseURL] != "" && tc.envVars[EnvDatabaseURL] == "" &&
				tc.envVars[EnvScryTestDBURL] == "" && !strings.Contains(logOutput, "non-standardized") {
				t.Errorf("Expected deprecation warning for SCRY_DATABASE_URL but none was logged")
			}
		})
	}
}

func TestStandardizeDatabaseURL(t *testing.T) {
	// Create a test logger
	var logBuffer strings.Builder
	logHandler := slog.NewTextHandler(&logBuffer, nil)
	logger := slog.New(logHandler)

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Already standardized URL",
			input:    "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable",
			expected: "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable",
			wantErr:  false,
		},
		{
			name:     "URL with different username/password",
			input:    "postgres://user:password@localhost:5432/testdb",
			expected: "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable",
			wantErr:  false,
		},
		{
			name:     "URL with different host",
			input:    "postgres://user:password@db:5432/testdb",
			expected: "postgres://postgres:postgres@db:5432/testdb?sslmode=disable",
			wantErr:  false,
		},
		{
			name:     "URL with missing database",
			input:    "postgres://user:password@localhost:5432",
			expected: "postgres://postgres:postgres@localhost:5432/scry_test?sslmode=disable",
			wantErr:  false,
		},
		{
			name:     "Non-postgres URL",
			input:    "mysql://user:password@localhost:3306/testdb",
			expected: "mysql://user:password@localhost:3306/testdb",
			wantErr:  false,
		},
		{
			name:     "Invalid URL",
			input:    "not-a-url",
			expected: "",
			wantErr:  true,
		},
	}

	// Set CI environment for testing
	os.Setenv(EnvCI, "true")
	defer os.Unsetenv(EnvCI)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := standardizeDatabaseURL(tc.input, logger)

			if (err != nil) != tc.wantErr {
				t.Errorf("standardizeDatabaseURL() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr && got != tc.expected {
				t.Errorf("standardizeDatabaseURL() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestUpdateDatabaseEnvironmentVariables(t *testing.T) {
	// Create a test logger
	var logBuffer strings.Builder
	logHandler := slog.NewTextHandler(&logBuffer, nil)
	logger := slog.New(logHandler)

	// Save current environment
	savedEnv := map[string]string{
		EnvDatabaseURL:     os.Getenv(EnvDatabaseURL),
		EnvScryTestDBURL:   os.Getenv(EnvScryTestDBURL),
		EnvScryDatabaseURL: os.Getenv(EnvScryDatabaseURL),
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

	// Set up test environment
	testVars := map[string]string{
		EnvDatabaseURL:     "postgres://user1:pass1@localhost:5432/db1",
		EnvScryTestDBURL:   "postgres://user2:pass2@localhost:5432/db2",
		EnvScryDatabaseURL: "postgres://user3:pass3@localhost:5432/db3",
	}

	for k, v := range testVars {
		os.Setenv(k, v)
	}

	// Test the function
	standardizedURL := "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable"
	updateDatabaseEnvironmentVariables(standardizedURL, logger)

	// Check that all environment variables have been updated
	for k := range testVars {
		if got := os.Getenv(k); got != standardizedURL {
			t.Errorf("Environment variable %s = %v, want %v", k, got, standardizedURL)
		}
	}

	// Verify logging behavior
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "Updating environment variable") {
		t.Errorf("Expected update log messages but none were found")
	}
}
