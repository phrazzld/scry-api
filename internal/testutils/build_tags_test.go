//go:build integration

package testutils_test

import (
	"errors"
	"testing"

	"github.com/phrazzld/scry-api/internal/testutils"
)

// TestBuildTagFunctionAvailability verifies that critical functions needed by other packages
// are available with the integration build tag. This is important to ensure CI can access
// these functions.
func TestBuildTagFunctionAvailability(t *testing.T) {
	// This test exists primarily to verify that key functions are available
	// when building with the integration tag. If this test compiles, it means
	// the functions are accessible.

	// Skip actual execution in environments without a database configured
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - DATABASE_URL not set")
	}

	// Test IsIntegrationTestEnvironment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Error("IsIntegrationTestEnvironment should return true in integration test")
	}

	// Test AssertNoErrorLeakage (if this compiles, the function is available)
	testutils.AssertNoErrorLeakage(t, errors.New("test error"))

	// Test MustGetTestDatabaseURL (just ensure it's available)
	_ = testutils.MustGetTestDatabaseURL()

	// Note: We don't need to call every function - the main purpose is
	// to verify these function references compile successfully with the
	// integration build tag.

	t.Log("All required functions are available with the integration build tag")
}
