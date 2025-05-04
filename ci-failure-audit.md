# CI Failure Audit - PR #26

## Summary

PR #26: "feat: implement card management API endpoints" is currently failing two CI checks:

1. **Lint** - Failed after 43s
2. **Test** - Failed after 55s

The primary issue is related to undefined symbols in the `testutils` package.

## Detailed Analysis

### Lint Failure

The linting process fails because it cannot find several functions in the `testutils` package when running tests:

```
internal/testutils/db_test.go:30:18: undefined: testutils.AssertCloseNoError
internal/testutils/db_test.go:33:18: undefined: testutils.SetupTestDatabaseSchema
internal/testutils/db_test.go:43:20: undefined: testutils.ResetTestData
internal/testutils/db_test.go:57:16: undefined: testutils.ResetTestData
internal/testutils/db_test.go:78:14: undefined: testutils.WithTx
internal/testutils/db_test.go:104:14: undefined: testutils.WithTx
internal/testutils/error_handling_test.go:31:13: undefined: testutils.AssertCloseNoError
internal/testutils/error_handling_test.go:39:13: undefined: testutils.AssertCloseNoError
internal/testutils/error_handling_test.go:86:18: undefined: testutils.AssertCloseNoError
internal/testutils/helpers_test.go:22:20: undefined: testutils.CreateTestUser
```

### Test Failure

The test process fails with similar errors in the same package:

```
FAIL	github.com/phrazzld/scry-api/internal/testutils [build failed]
```

## Root Cause

The issue stems from build tag configurations:

1. The file `internal/testutils/testutils_for_tests.go` has the build tag `//go:build test_internal_only`, but the tests are using the tag `test_without_external_deps` in CI.

2. The functions needed by the tests (`AssertCloseNoError`, `SetupTestDatabaseSchema`, `ResetTestData`, `WithTx`, `CreateTestUser`) are defined in `testutils_for_tests.go` but are not available when building with the `test_without_external_deps` tag.

## Relevant Files

1. `/Users/phaedrus/Development/scry/scry-api/internal/testutils/testutils_for_tests.go` - Contains the utility functions needed by tests
2. `/Users/phaedrus/Development/scry/scry-api/internal/testutils/db_test.go` - Test file that can't find the functions
3. `/Users/phaedrus/Development/scry/scry-api/internal/testutils/error_handling_test.go` - Another test file with missing functions
4. `/Users/phaedrus/Development/scry/scry-api/internal/testutils/helpers_test.go` - Another test file with missing functions

## Proposed Solution

1. **Update build tags**: Modify `testutils_for_tests.go` to include the `test_without_external_deps` tag in addition to or instead of the `test_internal_only` tag:

```go
//go:build test_without_external_deps || test_internal_only
```

Or:

```go
//go:build test_without_external_deps
```

2. **Fix CI configuration**: Alternatively, adjust the CI configuration to use the `test_internal_only` tag for running tests in the testutils package.

## Next Steps

1. Make the appropriate build tag change in `testutils_for_tests.go`
2. Push the change to the PR branch
3. Verify that the CI tests now pass
4. Ensure that all the local tests continue to pass with the updated build tags

## Related Information

This issue appears to be related to the test helper refactoring tasks (T023-T028) mentioned in the previous conversation, where compatibility and testing infrastructure were being updated.
