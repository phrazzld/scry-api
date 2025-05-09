//go:build integration || test_without_external_deps

package testdb

import (
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

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
