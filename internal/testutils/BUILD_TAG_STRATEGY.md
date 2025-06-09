# Build Tag Strategy for Testing Packages

This document explains the build tag strategy used for the test utilities in the Scry API project, specifically focusing on the relationship between `testutils` and `testdb` packages and their function exports.

## Overview

The Scry API project uses build tags to control which files are included in specific build scenarios. For testing, this is particularly important as we need to ensure test utilities are available in various environments while preventing circular dependencies and function redeclarations.

## Recent Improvements

We've made improvements to the build tag strategy to resolve issues with function visibility in CI environments:

1. Added `integration_exports.go` (build tag: `integration && !test_without_external_deps && !integration_test_internal`)
   - Contains critical functions needed by postgres integration tests
   - Ensures these functions are available in CI environments
   - Prevents conflicts with internal test functions

2. Added test verification to ensure no regression of function availability

3. Updated documentation to clarify build tag usage and standard patterns

## Key Build Tags

| Tag                       | Purpose                                                  | Usage                                                                          |
|---------------------------|----------------------------------------------------------|--------------------------------------------------------------------------------|
| `integration`             | Mark tests requiring external services                    | Integration tests that need real database connections                          |
| `integration_test_internal` | Internal tag to control function declarations             | Used to prevent duplicate function declarations during refactoring              |
| `test_without_external_deps` | Tests that simulate external dependencies                | CI environments, local testing without API keys                                |
| `exported_core_functions` | Functions that need to be available in production code    | Exported utilities that may be used outside of tests                            |

## File Structure

### testutils Package

The `testutils` package is being refactored to improve its structure. It currently consists of:

1. **compatibility.go** (build tag: `integration_test_internal`)
   - Contains the original implementation of test utilities
   - Only included when explicitly building with `integration_test_internal`
   - Disabled in normal builds to prevent function redeclarations

2. **db_forwarding.go** (build tag: `!integration_test_internal`)
   - Contains forwarding functions that delegate to the `testdb` package
   - Included in all builds EXCEPT when `integration_test_internal` is defined
   - Ensures critical functions are always available in CI environments

3. Other utility files with no build tags
   - General utilities that are always available

### testdb Package

The `testdb` package contains the actual implementation of database testing utilities:

1. **db.go** (build tag: `(integration || test_without_external_deps) && !exported_core_functions`)
   - Core implementation for test-only usage
   - Available in integration tests and mocked tests

2. **db_exports.go** (build tag: `exported_core_functions`)
   - Contains functions that need to be available in production code
   - Used when building with `exported_core_functions` tag

## Potential Issues

### Function Availability in CI

CI environments need access to functions like:
- `IsIntegrationTestEnvironment()`
- `WithTx()`
- `MustGetTestDatabaseURL()`
- `SetupTestDatabaseSchema()`

The most common issue is when these functions aren't available during CI builds due to build tag configurations. This can happen if:
1. The build tags exclude the files containing these functions in CI environments
2. The dependencies between functions create circular references across build tags

### Function Redeclarations

During refactoring, it's easy to accidentally have two implementations of the same function included in a build, causing redeclaration errors.

## Best Practices

1. **Use Consistent Build Tags**
   - Align build tags across related files
   - Document the intended behavior of each tag

2. **Prefer Negative Conditions for Critical Functions**
   - Use tags like `!integration_test_internal` instead of `forwarding_functions`
   - This ensures the functions are available by default

3. **Document Dependencies Between Functions**
   - Clearly indicate which functions depend on others
   - Ensure dependencies are available under the same build configurations

4. **Test in CI-like Environments**
   - Verify function availability with the build tags used in CI
   - Use explicit go build commands with specific tags to check for issues

## Troubleshooting

If you encounter function redeclaration errors or undefined functions:

1. Check which build tags are active in your environment
   ```bash
   go list -f '{{.BuildTags}}' ./...
   ```

2. Verify that the required functions are available under those tags
   ```bash
   # To see which files will be included with specific tags
   go list -f '{{.GoFiles}}' -tags=integration ./internal/testutils
   ```

3. For "undefined function" errors in integration tests:
   - Make sure the function is defined in `integration_exports.go`
   - Verify the build tags allow it to be included in your build

4. For function redeclaration errors:
   - Check for multiple files defining the same function
   - Ensure build tag combinations exclude conflicting definitions

5. For testing in a CI-like environment:
   ```bash
   # Build with the same tags used in CI
   go build -tags=integration ./internal/platform/postgres/...
   ```
