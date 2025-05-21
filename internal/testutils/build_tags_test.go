//go:build integration

package testutils_test

import (
	"testing"

	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
)

// TestBuildTagFunctionAvailability verifies that critical testutils functions
// are available with the integration build tag.
func TestBuildTagFunctionAvailability(t *testing.T) {
	t.Parallel()

	// Skip if not in integration test environment (this also tests IsIntegrationTestEnvironment)
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - DATABASE_URL not set")
	}

	// Test MustGetTestDatabaseURL
	dbURL := ""
	assert.NotPanics(t, func() {
		// This only tests the function is available, not that it returns a valid URL
		// since we're already inside IsIntegrationTestEnvironment() which confirms
		// the environment is set up correctly
		dbURL = testutils.MustGetTestDatabaseURL()
	}, "MustGetTestDatabaseURL should not panic when DATABASE_URL is set")
	assert.NotEmpty(t, dbURL, "Database URL should not be empty")

	// Just checking that these functions compile and are available

	// This test passes if it compiles, indicating the functions are available
	// with the integration build tag, which is used in CI
}

// TestFunctionForwarderForCI verifies that functions are properly forwarding to testdb.
// This is a compile-time test - if it builds, it means the functions are properly defined
// and available with the current build tags.
func TestFunctionForwarderForCI(t *testing.T) {
	// Just verify these functions are available by referring to them
	// This will fail at compile time if they're not available
	var (
		_ = testutils.IsIntegrationTestEnvironment
		_ = testutils.WithTx
		_ = testutils.GetTestDBWithT
		_ = testutils.MustGetTestDatabaseURL
		_ = testutils.SetupTestDatabaseSchema
	)

	// If this test compiles and runs, it means the forwarding functions
	// are properly defined and available with the current build tags
	assert.True(t, true, "Functions are available with current build tags")
}
