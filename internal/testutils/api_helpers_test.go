//go:build test_without_external_deps || (integration && !compatibility)

package testutils

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestGenerateAuthHeader tests the auth helper functions
func TestGenerateAuthHeader(t *testing.T) {
	// This test ensures the GenerateAuthHeader function works correctly
	testUserID := uuid.New()

	// Test that the function returns a valid Bearer token format
	authHeader, err := GenerateAuthHeader(testUserID)
	if err != nil {
		t.Fatalf("GenerateAuthHeader() error = %v", err)
	}

	// Verify the header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		t.Errorf("GenerateAuthHeader() = %v, want string starting with 'Bearer '", authHeader)
	}

	// Verify the token part is not empty
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		t.Error("GenerateAuthHeader() returned empty token")
	}
}
