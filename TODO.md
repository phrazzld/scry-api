# TODO List

This file contains detailed, atomic tasks that need to be addressed in the codebase. Each task is designed to be highly focused and independently addressable.

## Status Key
- [ ] Not started
- [~] In progress
- [x] Completed

## New Tasks

### Critical Issues

- [x] **Fix Import Cycle in Config Package Tests**:
  - **Files**: `/internal/config/load_test.go`
  - **Issue**: Import cycle detected in config package tests causing linter errors
  - **Description**: The config package's test file was importing testutils, which created an import cycle through the chain of dependencies.
  - **Acceptance Criteria**:
    - Break the circular dependency by moving test to config_test package
    - Implement local version of needed test utilities
    - Ensure tests pass without import cycle errors
    - Ensure linter passes without this error
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

- [x] **Fix Error Handling in Response Body Close**:
  - **Files**: `/cmd/server/auth_integration_test.go`
  - **Issue**: Error return values of response Body.Close() are not being checked
  - **Description**: The linter (errcheck) is flagging that error return values from response Body.Close() calls aren't being checked in test files.
  - **Acceptance Criteria**:
    - Update all instances of `defer resp.Body.Close()` to check return errors
    - Ensure proper handling of these errors in test cleanup
    - Ensure linter passes without these warnings
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

- [x] **Fix Unchecked os.Setenv Calls in Integration Tests**:
  - **File**: `/cmd/server/main_integration_test.go`
  - **Issue**: Error return values of os.Setenv are not being checked
  - **Description**: The linter (errcheck) is flagging that several calls to os.Setenv don't check return values.
  - **Acceptance Criteria**:
    - Update all os.Setenv calls to check return errors
    - Add proper error handling or test failures when environment variables can't be set
    - Ensure linter passes without these warnings
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

- [x] **Fix Unchecked Database Close in Tests**:
  - **Files**: `/cmd/server/main_task_test.go`, `/internal/testutils/db.go`
  - **Issue**: Error return values of db.Close() are not being checked
  - **Description**: The linter (errcheck) is flagging that db.Close() calls don't check return values.
  - **Acceptance Criteria**:
    - Update all db.Close() calls to check return errors
    - Add proper error handling for database close failures
    - Ensure linter passes without these warnings
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

- [x] **Fix Unchecked Store Method Calls in Task Tests**:
  - **File**: `/internal/task/runner_test.go`
  - **Issue**: Error return values for store methods not being checked
  - **Description**: The linter (errcheck) is flagging that several store method calls like SaveTask() and UpdateTaskStatus() don't check return values.
  - **Acceptance Criteria**:
    - Update all store method calls to check return errors
    - Add proper assertion/failure handling when store operations fail
    - Ensure linter passes without these warnings
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

- [x] **Improve Switch Statement in Auth Middleware**:
  - **File**: `/internal/api/middleware/auth.go`
  - **Issue**: Could use tagged switch on err (staticcheck QF1002)
  - **Description**: The linter suggests using a tagged switch on err for better readability.
  - **Acceptance Criteria**:
    - Refactor the switch statement to use a tagged switch on err
    - Ensure functionality remains the same
    - Ensure linter passes without this warning
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

- [x] **Remove Unused Function in Main Integration Test**:
  - **File**: `/cmd/server/main_integration_test.go`
  - **Issue**: Function createTempConfigFile is unused
  - **Description**: The linter shows that createTempConfigFile function is defined but never used.
  - **Acceptance Criteria**:
    - Either implement proper usage of the function
    - Or remove it if it's not needed
    - Or mark it appropriately to indicate it's for future use
    - Ensure linter passes without this warning
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

### Completed Tasks

- [x] **Fix Import Cycle in Config and Logger Packages**:
  - **Files**: `/internal/config/load_test.go`, `/internal/platform/logger/logger.go`, `/internal/platform/logger/logger_test.go`
  - **Issue**: Import cycle detected between config and logger packages
  - **Description**: There's a circular dependency where `config` imports `testutils`, which imports `platform/postgres`, which imports `platform/logger`, which imports `config`.
  - **Acceptance Criteria**:
    - Break the circular dependency by restructuring the imports
    - Consider creating a separate configuration type for logger that doesn't require the full config package
    - Ensure tests pass without import cycle errors
    - Maintain the same functionality after the restructuring
  - **Dependencies**: None
  - **Estimated Complexity**: Moderate

- [x] **Remove Unused Task Creation Function**:
  - **File**: `/internal/platform/postgres/task_store.go`
  - **Issue**: Function `createTaskFromDatabaseTask` is flagged as unused by linter
  - **Description**: The function is defined but never used, causing a linter warning.
  - **Acceptance Criteria**:
    - Either implement proper usage of the function
    - Or remove it and document why it's not needed
    - Or mark it appropriately to indicate it's for future use
    - Ensure linter passes without this warning
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

- [x] **Fix Test DB Connection in Integration Tests**:
  - **File**: `/cmd/server/main_integration_test.go`
  - **Issue**: Integration test fails with connection refused error
  - **Description**: Tests are unable to connect to the test database, possibly due to incorrect connection parameters or missing test database setup.
  - **Acceptance Criteria**:
    - Fix the database connection in tests
    - Ensure integration tests can run successfully
    - Update documentation on how to set up test database if needed
    - Consider adding a flag to skip integration tests when database is not available
  - **Dependencies**: None
  - **Estimated Complexity**: Moderate

- [x] **Update Deprecated Test Setup in Task Tests**:
  - **File**: `/cmd/server/main_task_test.go`
  - **Issue**: Still using deprecated `setupTestDB` and `teardownTestDB` functions
  - **Description**: Test uses old pattern for database setup and teardown that should be updated to use transaction-based approach.
  - **Acceptance Criteria**:
    - Refactor test to use `testutils.WithTx` instead of direct setup/teardown
    - Enable parallel testing if appropriate
    - Ensure tests still pass after refactoring
    - Remove unused functions
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

- [x] **Fix Code Formatting Issues**:
  - **Files**: Multiple files in `internal/platform/postgres/` package
  - **Issue**: Files are not properly formatted according to gofmt
  - **Description**: The linter shows formatting errors in task_store.go, user_store_get_test.go, user_store_test.go, and user_store_update_delete_test.go.
  - **Acceptance Criteria**:
    - Run gofmt on all affected files
    - Ensure there are no formatting errors when running golangci-lint
  - **Dependencies**: None
  - **Estimated Complexity**: Simple

- [x] **Implement Asynchronous Task Runner**:
  - **Issue**: Need to implement a background task processing system
  - **Description**: According to BACKLOG.md, we need to implement an in-memory background task queue with a worker pool using goroutines and channels, along with a recovery mechanism for processing tasks.
  - **Acceptance Criteria**:
    - Implement a basic in-memory task queue
    - Implement a worker pool using goroutines and channels
    - Implement a recovery mechanism for handling application restarts
    - Ensure tasks can be processed asynchronously
    - Add comprehensive tests for the task processing system
    - Ensure proper error handling and logging
  - **Dependencies**: None
  - **Estimated Complexity**: Complex

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
