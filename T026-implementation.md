# Implementation Details for T026: Resolve undefined function errors in postgres tests

## Issue Analysis

The CI failures were caused by critical functions needed by postgres tests being unavailable in the integration test environment. Specifically:

1. The `AssertNoErrorLeakage` function was used in multiple error leakage tests in the postgres package, but wasn't properly exported from testutils
2. Build tag configurations in the testutils package prevented some functions from being available during CI builds

## Solution Implemented

### 1. Added AssertNoErrorLeakage to integration_exports.go

Added the AssertNoErrorLeakage function to integration_exports.go, which has the following build tag:

```go
//go:build integration && !test_without_external_deps && !integration_test_internal
```

This ensures the function is available for integration tests while avoiding conflicts with other implementations.

### 2. Added strings Import

Added the missing strings import to integration_exports.go, as the AssertNoErrorLeakage function uses strings.Contains.

### 3. Created Test for Error Leakage Function

Added error_leakage_test.go to verify that the AssertNoErrorLeakage function works correctly:

```go
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
```

### 4. Created Build Tags Test

Added build_tags_test.go to verify that all critical functions needed by postgres tests are available with the integration build tag:

```go
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
```

### 5. Updated BUILD_TAG_STRATEGY.md

Enhanced the BUILD_TAG_STRATEGY.md document with:
- Information about the recent improvements to the build tag strategy
- Detailed troubleshooting guidance with example commands
- Best practices for dealing with build tag issues

### 6. Verified the Fix Works

1. Ran tests on testutils package with integration tag
2. Built postgres package with integration tag
3. Verified no compilation errors
4. Ran tests with the AssertNoErrorLeakage function to verify it's working correctly

## Testing Strategy

1. Created specific test for AssertNoErrorLeakage function
2. Added build tag function availability test
3. Verified postgres package compiles successfully with all required functions
4. Ran the entire test suite to ensure nothing was broken by our changes

## Conclusion

The implemented solution resolves the undefined function errors in postgres tests by ensuring the AssertNoErrorLeakage function is available with the correct build tags. The solution:

1. Uses the existing integration_exports.go file to provide consistent function exports
2. Follows the established build tag pattern
3. Provides tests to verify the functions are available
4. Enhances documentation to prevent similar issues in the future

This fix completes task T026 and should resolve the CI failures related to undefined functions in postgres tests.
