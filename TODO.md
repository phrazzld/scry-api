# TODO List

This file contains detailed, atomic tasks that need to be addressed in the codebase. Each task is designed to be highly focused and independently addressable.

## Status Key
- [ ] Not started
- [~] In progress
- [x] Completed

## New Tasks

### Authentication Tasks

- [x] **Implement JWT Authentication Service**:
  - **Issue**: Need to implement JWT token generation and validation
  - **Description**: According to BACKLOG.md, we need to implement a JWT authentication service for user authentication. This service will be used by the authentication middleware and API endpoints.
  - **Acceptance Criteria**:
    - Implement JWT generation with proper claims structure
    - Implement JWT validation logic
    - Implement token refresh mechanism if needed
    - Add configuration for JWT secrets and token lifetimes
    - Add comprehensive tests for the authentication service
  - **Dependencies**: User Store implementation (completed)
  - **Estimated Complexity**: Complex

- [x] **Implement Authentication API Endpoints and Middleware**:
  - **Issue**: Need to implement user registration and login endpoints and JWT authentication middleware
  - **Description**: According to BACKLOG.md, we need to implement two authentication endpoints and middleware: Registration (`POST /auth/register`), Login (`POST /auth/login`), and JWT validation middleware for protected routes. These will leverage the existing User Store and JWT Authentication Service.
  - **Acceptance Criteria**:
    - Implement `POST /auth/register` endpoint that validates input and creates user accounts
    - Implement `POST /auth/login` endpoint that authenticates users and returns JWT tokens
    - Implement JWT authentication middleware for protecting API routes
    - Integrate middleware with the router
    - Add comprehensive validation for all request payloads
    - Return proper HTTP status codes and error messages
    - Add thorough integration tests for authentication flows
    - Ensure all error scenarios are properly handled and tested
  - **Dependencies**: User Store implementation (completed), JWT Authentication Service (completed)
  - **Estimated Complexity**: Complex

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

- [x] **Add Tests for Middleware Components**:
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

- [x] **Add Tests for Missing Packages**:
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

- [x] **Standardize Test Helper Functions**:
  - **Issue**: Test helper functions might not be consistent across test files
  - **Description**: Review test helper functions across the codebase to ensure they follow consistent patterns, particularly regarding transaction-based isolation.
  - **Acceptance Criteria**:
    - Identify all test helper functions in test files
    - Standardize their signatures and naming conventions
    - Move common helpers to `testutils` package if appropriate
    - Update all tests to use the standardized helpers
  - **Dependencies**: None
  - **Estimated Complexity**: Moderate

- [x] **Add Comprehensive Error Handling in Tests**:
  - **Issue**: Some tests may not have thorough error handling
  - **Description**: Ensure all tests properly handle errors, including from deferred functions.
  - **Acceptance Criteria**:
    - Review all deferred function calls in tests
    - Implement proper error handling for deferred functions
    - Use appropriate error-checking assertion functions
    - Document any special error handling patterns
  - **Dependencies**: None
  - **Estimated Complexity**: Moderate
