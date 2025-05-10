//go:build integration || test_without_external_deps

package testdb

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestGetTestDB(t *testing.T) {
	// Save original environment variables
	origDBURL := os.Getenv("DATABASE_URL")
	origTestDBURL := os.Getenv("SCRY_TEST_DB_URL")
	origSCRYDBURL := os.Getenv("SCRY_DATABASE_URL")

	// Restore environment variables when test completes
	defer func() {
		if err := os.Setenv("DATABASE_URL", origDBURL); err != nil {
			log.Printf("Failed to restore DATABASE_URL: %v", err)
		}
		if err := os.Setenv("SCRY_TEST_DB_URL", origTestDBURL); err != nil {
			log.Printf("Failed to restore SCRY_TEST_DB_URL: %v", err)
		}
		if err := os.Setenv("SCRY_DATABASE_URL", origSCRYDBURL); err != nil {
			log.Printf("Failed to restore SCRY_DATABASE_URL: %v", err)
		}
	}()

	// Test case: no environment variables set
	if err := os.Unsetenv("DATABASE_URL"); err != nil {
		t.Fatalf("Failed to unset DATABASE_URL: %v", err)
	}
	if err := os.Unsetenv("SCRY_TEST_DB_URL"); err != nil {
		t.Fatalf("Failed to unset SCRY_TEST_DB_URL: %v", err)
	}
	if err := os.Unsetenv("SCRY_DATABASE_URL"); err != nil {
		t.Fatalf("Failed to unset SCRY_DATABASE_URL: %v", err)
	}

	db, err := GetTestDB()
	if db != nil {
		t.Error("Expected nil DB when no environment variables are set")
	}
	if err == nil {
		t.Error("Expected error when no environment variables are set")
	} else {
		// Verify error message contains helpful information
		t.Logf("Error message: %v", err)
		if msg := err.Error(); !contains(msg, "DATABASE_URL") || !contains(msg, "SCRY_TEST_DB_URL") || !contains(msg, "SCRY_DATABASE_URL") {
			t.Errorf("Error message doesn't mention all environment variables: %s", msg)
		}
	}
}

func TestMaskDatabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard postgres URL",
			url:      "postgres://user:password@localhost:5432/dbname",
			expected: "postgres://user:****@localhost:5432/dbname",
		},
		{
			name:     "URL with query parameters",
			url:      "postgres://user:password@localhost:5432/dbname?sslmode=disable",
			expected: "postgres://user:****@localhost:5432/dbname?sslmode=disable",
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskDatabaseURL(tt.url)
			if result != tt.expected {
				t.Errorf("maskDatabaseURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

// TestErrorFormatting verifies that error formatting functions provide detailed and helpful error messages
func TestErrorFormatting(t *testing.T) {
	tests := []struct {
		name           string
		errorGenerator func() error
		expectedParts  []string
	}{
		{
			name: "Environment Variable Error",
			errorGenerator: func() error {
				// Force all env vars to be empty for this test
				origDBURL := os.Getenv("DATABASE_URL")
				origTestDBURL := os.Getenv("SCRY_TEST_DB_URL")
				origSCRYDBURL := os.Getenv("SCRY_DATABASE_URL")
				defer func() {
					if err := os.Setenv("DATABASE_URL", origDBURL); err != nil {
						t.Logf("Failed to restore DATABASE_URL: %v", err)
					}
					if err := os.Setenv("SCRY_TEST_DB_URL", origTestDBURL); err != nil {
						t.Logf("Failed to restore SCRY_TEST_DB_URL: %v", err)
					}
					if err := os.Setenv("SCRY_DATABASE_URL", origSCRYDBURL); err != nil {
						t.Logf("Failed to restore SCRY_DATABASE_URL: %v", err)
					}
				}()

				if err := os.Unsetenv("DATABASE_URL"); err != nil {
					t.Logf("Failed to unset DATABASE_URL: %v", err)
				}
				if err := os.Unsetenv("SCRY_TEST_DB_URL"); err != nil {
					t.Logf("Failed to unset SCRY_TEST_DB_URL: %v", err)
				}
				if err := os.Unsetenv("SCRY_DATABASE_URL"); err != nil {
					t.Logf("Failed to unset SCRY_DATABASE_URL: %v", err)
				}

				return formatEnvVarError()
			},
			expectedParts: []string{
				"Database connection failed",
				"Required environment variables missing",
				"DATABASE_URL",
				"SCRY_TEST_DB_URL",
				"Please ensure one of",
			},
		},
		{
			name: "Database Connection Error",
			errorGenerator: func() error {
				baseErr := fmt.Errorf("connection refused")
				dbURL := "postgres://user:password@localhost:5432/invalid_db"
				return formatDBConnectionError(baseErr, dbURL)
			},
			expectedParts: []string{
				"Database connection failed",
				"connection refused",
				"Database URL used",
				"postgres://user:****@localhost",
				"PostgreSQL service is running",
				"Credentials and connection string are correct",
			},
		},
		{
			name: "Project Root Error",
			errorGenerator: func() error {
				checkedPaths := []string{"/path/to/go.mod", "/another/path/go.mod"}
				checkedEnvVars := []string{"SCRY_PROJECT_ROOT=not_found"}
				return formatProjectRootError(checkedPaths, checkedEnvVars)
			},
			expectedParts: []string{
				"Could not find go.mod",
				"Checked environment variables",
				"Checked paths",
				"To fix this, set SCRY_PROJECT_ROOT",
			},
		},
		{
			name: "Migration Error",
			errorGenerator: func() error {
				baseErr := fmt.Errorf("migration failed: syntax error in migration file")
				migrationsDir := "/path/to/migrations"
				return formatMigrationError(baseErr, migrationsDir)
			},
			expectedParts: []string{
				"Failed to run database migrations",
				"migration failed: syntax error",
				"Migrations directory",
				"Please check",
				"Migration files exist and are valid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorGenerator()
			if err == nil {
				t.Fatalf("Expected error but got nil")
			}

			errMsg := err.Error()
			t.Logf("Generated error message:\n%s", errMsg)

			// Verify the error contains all expected parts
			for _, part := range tt.expectedParts {
				if !contains(errMsg, part) {
					t.Errorf("Error message doesn't contain expected part: %q", part)
				}
			}

			// Verify the error message length is sufficient for diagnostics
			if len(errMsg) < 100 {
				t.Errorf(
					"Error message is suspiciously short (%d chars), may not contain enough diagnostics",
					len(errMsg),
				)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
