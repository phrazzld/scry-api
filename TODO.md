# CI Failure Resolution Tasks

## Issue 1: Database User & URL Standardization

- [x] **Task 1.1: Enhance Logging in `GetTestDatabaseURL()`**
  - **Priority**: High
  - **Description**: Add detailed structured logging to `GetTestDatabaseURL()` to improve visibility into its decision process
  - **Implementation**:
    - Modify function to log environment variables being checked (DATABASE_URL, SCRY_TEST_DB_URL, etc.)
    - Log the source of each connection string component
    - Log the final constructed URL (with password masked)
    - Use structured logging with appropriate context fields
  - **Verification**:
    - Run CI job and confirm diagnostic information is present in logs
  - **Dependencies**: None

- [x] **Task 1.2: Refactor `GetTestDatabaseURL()` for CI Environment Awareness**
  - **Priority**: High
  - **Description**: Refactor the function to properly prioritize CI environment variables and always use 'postgres' user in CI
  - **Implementation**:
    - Strengthen CI environment detection (check both `CI` and `GITHUB_ACTIONS` env vars)
    - Explicitly use 'postgres' as username and password when in CI environment
    - Parse the identified URL with robust error handling
    - Reconstruct URL with standardized credentials
    - Update all relevant environment variables with standardized URL
  - **Verification**:
    - CI logs should show standardized URL with 'postgres' user
    - Database connection should succeed in CI
  - **Dependencies**: Task 1.1

- [x] **Task 1.3: Add Unit Tests for `GetTestDatabaseURL()` Covering CI Scenarios**
  - **Priority**: Medium
  - **Description**: Create comprehensive unit tests for database URL standardization behavior in CI
  - **Implementation**:
    - Add tests that mock CI environment variables
    - Test with various input URLs (root user, other users, missing credentials)
    - Verify standardized output consistently uses 'postgres' in CI
    - Test fallback mechanisms and error handling
  - **Verification**:
    - CI job successfully executes the new tests
    - Test coverage for `GetTestDatabaseURL()` increases
  - **Dependencies**: Task 1.2

- [x] **Task 1.4: Add Integration Test for Basic Database Connection**
  - **Priority**: Medium
  - **Description**: Create an early integration test to verify database connectivity
  - **Implementation**:
    - Create test that calls `GetTestDatabaseURL()`
    - Attempt to establish database connection
    - Execute a simple test query (SELECT 1)
    - Assert success
  - **Verification**:
    - Test passes consistently in CI environment
  - **Dependencies**: Task 1.2

## Issue 2: Project Root Detection

- [x] **Task 2.1: Enhance Logging in `findProjectRoot()`**
  - **Priority**: High
  - **Description**: Add detailed logging to trace project root detection logic
  - **Implementation**:
    - Log the initial working directory
    - Log each path checked and markers sought (go.mod, .git)
    - Log the outcome of each check
    - Log the final determined project root or error message
  - **Verification**:
    - CI logs should show detailed project root detection process
  - **Dependencies**: None

- [x] **Task 2.2: Refactor `findProjectRoot()` for CI Robustness**
  - **Priority**: High
  - **Description**: Make project root detection more reliable in CI environments
  - **Implementation**:
    - Prioritize CI-specific environment variables (GITHUB_WORKSPACE, CI_PROJECT_DIR)
    - Add explicit check that the detected path contains go.mod
    - Improve the fallback auto-detection mechanism
    - Provide clear error messages for troubleshooting
  - **Verification**:
    - CI logs confirm correct project root identification
    - Subsequent steps (migrations) correctly find files
  - **Dependencies**: Task 2.1

- [x] **Task 2.3: Add Unit Tests for `findProjectRoot()` Covering CI Scenarios**
  - **Priority**: Medium
  - **Description**: Create tests for project root detection in CI-like environments
  - **Implementation**:
    - Create tests that mock CI environment variables
    - Simulate different filesystem structures using temporary directories
    - Test explicit variable detection and fallback mechanisms
    - Verify error handling
  - **Verification**:
    - Tests pass in CI environment
    - Increased test coverage for `findProjectRoot()`
  - **Dependencies**: Task 2.2

## Issue 3: Migration Execution

- [x] **Task 3.1: Ensure Migration Tool Uses Standardized Inputs**
  - **Priority**: High
  - **Description**: Ensure migrations use correct database URL and file paths
  - **Implementation**:
    - Review migration initialization code
    - Ensure it uses the enhanced `GetTestDatabaseURL()` and `findProjectRoot()`
    - Correctly construct path to migration files relative to project root
    - Consider centralizing migration logic in a dedicated function
  - **Verification**:
    - Code review confirms standardized functions are used
    - CI logs show correct parameters
  - **Dependencies**: Task 1.2, Task 2.2

- [x] **Task 3.2: Add Comprehensive Logging to Migration Execution**
  - **Priority**: Medium
  - **Description**: Improve visibility into migration process
  - **Implementation**:
    - Log database URL being used (masked)
    - Log resolved path to migration files
    - Log discovered migration files
    - Log migration application status (before/after) with success/failure
  - **Verification**:
    - CI logs show detailed migration information
  - **Dependencies**: Task 3.1

- [x] **Task 3.3: Verify Full Migration Execution in CI**
  - **Priority**: High
  - **Description**: Confirm migrations run successfully in CI
  - **Implementation**:
    - Ensure CI workflow includes explicit migration step
    - Add post-migration verification (query schema_migrations table)
    - Make CI fail if migrations aren't successfully applied
  - **Verification**:
    - CI job completes successfully
    - Logs confirm migrations applied without errors
  - **Dependencies**: Task 1.2, Task 2.2, Task 3.1, Task 3.2

## General CI Improvements

- [x] **Task 4.1: Document CI Environment Configuration**
  - **Priority**: Medium
  - **Description**: Document all CI environment variables and configuration
  - **Implementation**:
    - Create/update docs/ci_environment.md
    - Document all relevant environment variables (purpose, format, usage)
    - Include troubleshooting guide for common CI issues
  - **Verification**:
    - Documentation review for clarity and completeness
  - **Dependencies**: Tasks 1.2, 2.2

- [x] **Task 4.2: Implement CI Pre-flight Checks**
  - **Priority**: Low
  - **Description**: Add early CI stage to validate environment setup
  - **Implementation**:
    - Create script to verify critical environment variables
    - Check database connectivity before main tests
    - Verify project root detection
    - Run as initial CI step
  - **Verification**:
    - CI pipeline catches configuration issues early
  - **Dependencies**: Tasks 1.4, 3.3

- [x] **Task 4.3: Standardize Environment Variable Usage**
  - **Priority**: Medium
  - **Description**: Establish consistent environment variable conventions
  - **Implementation**:
    - Define naming conventions for environment variables
    - Document variable precedence and default values
    - Update code to follow these conventions
  - **Verification**:
    - Code review confirms consistency
  - **Dependencies**: Task 4.1

## Issue 5: Code Organization and Size

- [x] **Task 5.1: Refactor cmd/server/main.go into Smaller Files**
  - **Priority**: Medium
  - **Description**: Break down the large main.go file (1108 lines) into smaller, more modular files
  - **Implementation**:
    - Analyze the file to identify logical components
    - Extract migration-related logic into dedicated files
    - Extract API handlers into separate files
    - Extract configuration and initialization logic into appropriate files
    - Ensure consistent error handling across all files
  - **Verification**:
    - All functionality remains intact
    - Code passes all tests and linting checks
    - File size is under the 1000-line limit
  - **Dependencies**: None

- [x] **Task 5.2: Refactor internal/testdb/db.go into Smaller Files**
  - **Priority**: Medium
  - **Description**: Break down the large db.go file (1069 lines) into smaller, more modular files
  - **Implementation**:
    - Analyze the file to identify logical components
    - Extract database initialization logic into dedicated files
    - Separate test utility functions into domain-specific files
    - Maintain clear documentation of exported functions
  - **Verification**:
    - All functionality remains intact
    - Code passes all tests and linting checks
    - File size is under the 1000-line limit
  - **Dependencies**: None

## Issue 6: CI Failure Resolution (PR: feature/card-management-api)

Based on CI failure analysis, these tasks address compilation errors and linting violations.

- [x] **Task 6.1: Fix Compilation Errors in cmd/server/main.go**
  - **Priority**: Critical (P0)
  - **Description**: Fix undefined functions preventing migration command execution
  - **Implementation**:
    1. Verify that `loadAppConfig`, `setupAppLogger`, `handleMigrations`, `setupAppDatabase`, `newApplication` exist in files under `cmd/server/`
    2. Ensure all files declare `package main`
    3. Remove any restrictive build tags (e.g., `//go:build exported_core_functions`) from core application files
    4. Update imports in `main.go` to match refactored structure
  - **Verification**:
    - `go build ./cmd/server/...` succeeds
    - `go run cmd/server/main.go -migrate=up` executes without undefined errors
  - **Dependencies**: None

- [x] **Task 6.2: Fix errcheck Violations**
  - **Priority**: High (P1)
  - **Description**: Add error handling for unchecked function returns
  - **Implementation**:
    1. `internal/ciutil/database.go:187`: Check `os.Setenv` error and log if non-nil
    2. `internal/ciutil/database_test.go:83,90,92`: Add `t.Fatalf` for `os.Setenv` errors and `t.Logf` for `os.Unsetenv`
    3. `internal/ciutil/projectroot_test.go:221`: Wrap `os.RemoveAll` in defer with error check
    4. `internal/config/load_test.go:168,192,217`: Check errors for `os.Remove` and `file.Close()`
  - **Verification**:
    - `golangci-lint run --build-tags=test_without_external_deps ./...` reports no errcheck violations
  - **Dependencies**: Task 6.1

- [x] **Task 6.3: Fix ineffassign Violation**
  - **Priority**: High (P1)
  - **Description**: Fix ineffectual assignment in internal/ciutil/projectroot_test.go:202
  - **Implementation**:
    1. Change error handling to verify that `FindMigrationsDir` returns expected error
    2. Use pattern: `if err == nil { t.Fatal("expected error finding migrations without project root") }`
  - **Verification**:
    - `golangci-lint run --build-tags=test_without_external_deps ./...` reports no ineffassign violations
  - **Dependencies**: Task 6.1

- [x] **Task 6.4: Add Early Build Verification Step to CI**
  - **Priority**: High (P1)
  - **Description**: Add CI step to catch compilation errors before tests
  - **Implementation**:
    1. Modify GitHub Actions workflow to include `go build ./cmd/...` step
    2. Place this step after checkout but before linting and tests
  - **Verification**:
    - CI pipeline fails early if compilation errors exist
  - **Dependencies**: None

- [x] **Task 6.5: Enforce Pre-commit Hooks**
  - **Priority**: Medium (P2)
  - **Description**: Configure pre-commit hooks for linting and build checks
  - **Implementation**:
    1. Update `.pre-commit-config.yaml` to run `golangci-lint` and `go build`
    2. Document installation instructions in README.md
    3. Ensure hooks run on all commits
  - **Verification**:
    - Commits fail locally if linting or build errors exist
  - **Dependencies**: None

- [ ] **Task 6.6: Document Build Tag Usage Policy**
  - **Priority**: Medium (P2)
  - **Description**: Create clear guidelines for Go build tag usage
  - **Implementation**:
    1. Create `docs/BUILD_TAGS.md` with approved patterns
    2. Document that core application logic should not use restrictive build tags
    3. Link from development guidelines
  - **Verification**:
    - Documentation exists and is referenced in code reviews
  - **Dependencies**: None

## Prevention Measures

1. Run dedicated CI-specific tests early in the pipeline
2. Enhance logging and observability in CI environment
3. Provide tools for developers to simulate CI environment locally
4. Regularly audit CI pipeline configuration and scripts
5. Ensure strict code review for environment-interacting code
6. Add file size limits to pre-commit hooks to prevent excessive file growth
7. Never bypass pre-commit hooks unless absolutely necessary
8. Run local build and linting checks before pushing code
