//go:build integration

package testutils_test

import (
	"errors"
	"testing"

	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
)

// TestAssertNoErrorLeakage verifies that the AssertNoErrorLeakage function works as expected
func TestAssertNoErrorLeakage(t *testing.T) {
	// Skip in non-integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping test in non-integration environment")
	}

	// Create a testing.T that will record failures instead of failing the test
	mockT := new(testing.T)

	// Test with a clean error (should not fail)
	testutils.AssertNoErrorLeakage(mockT, errors.New("clean error message"))
	assert.False(t, mockT.Failed(), "AssertNoErrorLeakage should not fail on clean error")

	// Test with a leaky error (should fail)
	mockT = new(testing.T)
	testutils.AssertNoErrorLeakage(mockT, errors.New("error with postgres details"))
	assert.True(t, mockT.Failed(), "AssertNoErrorLeakage should fail on error with leak")

	// Test with nil error (should not fail)
	mockT = new(testing.T)
	testutils.AssertNoErrorLeakage(mockT, nil)
	assert.False(t, mockT.Failed(), "AssertNoErrorLeakage should not fail on nil error")
}
