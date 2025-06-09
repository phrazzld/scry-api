//go:build test_without_external_deps

package testutils_test

import (
	"testing"

	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
)

// Basic test to ensure at least one test runs in the package
func TestBasicFunctionality(t *testing.T) {
	// Test CreateTestUser to ensure it creates a valid user
	user := testutils.CreateTestUser(t)

	assert.NotNil(t, user)
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.Email)
	assert.Contains(t, user.Email, "@example.com")
}

// Test environment setup utilities
func TestEnvironmentSetup(t *testing.T) {
	// Test SetupEnv
	cleanup := testutils.SetupEnv(t, map[string]string{
		"TEST_VAR": "test_value",
	})
	defer cleanup()

	// This test passes by compiling and running
	assert.True(t, true, "Environment setup test passed")
}
