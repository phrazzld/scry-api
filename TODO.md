# TODO List

This file contains detailed, atomic tasks that need to be addressed in the codebase. Each task is designed to be highly focused and independently addressable.

## Status Key
- [ ] Not started
- [~] In progress
- [x] Completed

## Urgent Tasks

### Linting Tasks

- [x] **Remove Unused Test Setup Functions**:
  - **File**: `/internal/platform/postgres/user_store_test.go`
  - **Issue**: Functions `setupTestDB` and `teardownTestDB` are flagged as unused by golangci-lint
  - **Description**: These functions are marked as deprecated with comments indicating that `testutils.WithTx` should be used instead, but they're still generating linting errors.
  - **Acceptance Criteria**:
    - Remove both functions completely OR
    - Use build tags or other techniques to retain them for reference without triggering linter errors
    - Verify golangci-lint runs without these specific errors
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

### Build Tasks

- [x] **Complete Full Integration Test Changes**:
  - **File**: `/internal/platform/postgres/user_store_test.go` and potentially others
  - **Issue**: The conversion to transaction-based testing might not be complete across all tests
  - **Description**: While some tests have been updated to use transaction-based isolation with `t.Parallel()`, there may be other tests or test utilities that still need updating.
  - **Acceptance Criteria**:
    - Audit all test files to ensure consistent use of the transaction-based approach
    - Update any tests still using the old `setupTestDB`/`teardownTestDB` pattern
    - Ensure all tests properly use `t.Parallel()` where appropriate
    - All tests pass when run with `-race` flag
  - **Dependencies**: None
  - **Estimated Complexity**: Moderate

### Missing Tests

- [ ] **Add Tests for Middleware Components**:
  - **Issue**: Authentication middleware tests appear to be missing
  - **Description**: According to BACKLOG.md, the "Authentication Middleware" task includes a sub-task for adding tests, but there doesn't appear to be relevant test coverage.
  - **Acceptance Criteria**:
    - Create integration tests for JWT validation middleware
    - Test both valid and invalid token scenarios
    - Test expired token handling
    - Test proper authorization flow
    - Test role-based access control (if implemented)
  - **Dependencies**: Authentication Middleware implementation
  - **Estimated Complexity**: Moderate

- [ ] **Add Tests for Missing Packages**:
  - **Issue**: Several packages have no test files
  - **Description**: The following packages show "[no test files]" when running `go test ./...`:
    - `github.com/phrazzld/scry-api/internal/api`
    - `github.com/phrazzld/scry-api/internal/generation`
    - `github.com/phrazzld/scry-api/internal/service`
    - `github.com/phrazzld/scry-api/internal/store`
    - `github.com/phrazzld/scry-api/internal/task`
  - **Acceptance Criteria**:
    - Evaluate each package to determine appropriate test coverage
    - Implement unit tests for core functionality in each package
    - Create integration tests where appropriate
    - Ensure at least 70% code coverage for critical components
  - **Dependencies**: Implementation of the respective packages
  - **Estimated Complexity**: Complex

## Non-Urgent Tasks

### Documentation Tasks

- [x] **Add Package Documentation for Missing Packages**:
  - **Issue**: Several packages have a `doc.go` file but may need actual implementation documentation
  - **Description**: Ensure comprehensive package documentation exists for all packages, particularly those that show as having a `doc.go` file but potentially lacking implementation details.
  - **Acceptance Criteria**:
    - Review all `doc.go` files for completeness
    - Add or update package documentation where missing or incomplete
    - Ensure documentation follows project standards
    - Verify documentation with `godoc` tool
  - **Dependencies**: None
  - **Estimated Complexity**: Simple to Moderate

### Refactoring Tasks

- [ ] **Standardize Test Helper Functions**:
  - **Issue**: Test helper functions might not be consistent across test files
  - **Description**: Review test helper functions across the codebase to ensure they follow consistent patterns, particularly regarding transaction-based isolation.
  - **Acceptance Criteria**:
    - Identify all test helper functions in test files
    - Standardize their signatures and naming conventions
    - Move common helpers to `testutils` package if appropriate
    - Update all tests to use the standardized helpers
  - **Dependencies**: None
  - **Estimated Complexity**: Moderate

- [ ] **Add Comprehensive Error Handling in Tests**:
  - **Issue**: Some tests may not have thorough error handling
  - **Description**: Ensure all tests properly handle errors, including from deferred functions.
  - **Acceptance Criteria**:
    - Review all deferred function calls in tests
    - Implement proper error handling for deferred functions
    - Use appropriate error-checking assertion functions
    - Document any special error handling patterns
  - **Dependencies**: None
  - **Estimated Complexity**: Moderate
