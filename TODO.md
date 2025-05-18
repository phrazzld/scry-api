# TODO

*This Todo List is managed by the claude.ai/code agent. Do not update directly.*

## Completed Tasks

### CI Failure Resolution Tasks (2025-05-16)

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

- [x] **Task 6.6: Document Build Tag Usage Policy**
  - **Priority**: Medium (P2)
  - **Description**: Create clear guidelines for Go build tag usage
  - **Implementation**:
    1. Create `docs/BUILD_TAGS.md` with approved patterns
    2. Document that core application logic should not use restrictive build tags
    3. Link from development guidelines
  - **Verification**:
    - Documentation exists and is referenced in code reviews
  - **Dependencies**: None

## New CI Failure Resolution Tasks (2025-05-16)

### CI/CD
- [x] **T001 · Bugfix · P0: update ci workflow to use 'go run ./cmd/server' for database migrations**
    - **Context:** `CI Resolution Plan > Resolution Steps > Issue 1: Incorrect Go Command for Running Migrations in CI`
    - **Action:**
        1. Open the CI workflow file: `.github/workflows/ci.yml`.
        2. Locate the "Reset and prepare database" step and change the line executing the migration from `go run cmd/server/main.go ...` to `go run ./cmd/server ...`, preserving all necessary flags.
    - **Done‑when:**
        1. The "Reset and prepare database" step in CI completes without "undefined" compilation errors.
        2. Database migrations are successfully applied, confirmed by logs or subsequent successful test steps.
    - **Verification:**
        1. Trigger the CI pipeline (e.g., by pushing the commit).
        2. Observe the "Reset and prepare database" step logs for successful execution using the package path and no compilation errors.
    - **Depends‑on:** none

- [x] **T002 · Feature · P0: add dedicated 'go build ./cmd/server' step to ci pipeline**
    - **Context:** `CI Resolution Plan > Resolution Steps > Issue 2: Missing Early Build Verification for cmd/server Package in CI`
    - **Action:**
        1. Open the CI workflow file: `.github/workflows/ci.yml`.
        2. After Go environment setup and dependency installation, and *before* the "Reset and prepare database" step, add a new step:
           ```yaml
           - name: Build main application
             run: go build ./cmd/server
           ```
    - **Done‑when:**
        1. The new "Build main application" step is executed and passes in the CI pipeline.
    - **Verification:**
        1. Trigger the CI pipeline and confirm the new build step passes.
        2. (Optional) Temporarily introduce a syntax error in a non-`main.go` file within the `cmd/server` package, confirm the new build step fails early, then revert the error.
    - **Depends‑on:** none

- [x] **T008 · Chore · P2: enhance ci log verbosity for go command execution steps**
    - **Context:** `CI Resolution Plan > Prevention Measures > Bullet 6 (Improve CI Observability)`
    - **Action:**
        1. Review CI steps in `.github/workflows/ci.yml` that execute Go commands (e.g., build, run, test) or other critical script executions.
        2. Modify these steps to ensure the exact commands being run are logged (e.g., using `echo "Running: <command>"` or `set -x`) and their full standard output/error streams are captured and visible.
    - **Done‑when:**
        1. CI logs for key Go execution steps clearly display the full command executed.
        2. CI logs show comprehensive output from these Go commands, aiding easier diagnosis of failures.
    - **Verification:**
        1. Trigger a CI run and inspect the logs for the specified steps to confirm improved command and output visibility.
    - **Depends‑on:** none

### Documentation
- [x] **T003 · Chore · P1: update project documentation with correct 'go run ./cmd/server' command and centralize command guidance**
    - **Context:** `CI Resolution Plan > Resolution Steps > Issue 3: Incorrect or Misleading Commands in Documentation` and `Prevention Measures > Bullet 3 (Standardize and Document Run Commands)`
    - **Action:**
        1. Identify all project documents (e.g., `README.md`, `CLAUDE.md`, `docs/DEVELOPMENT_GUIDE.md`) that provide instructions for building, running, or migrating the application.
        2. Search for instances of `go run cmd/server/main.go` and replace them with `go run ./cmd/server`, ensuring all associated flags are correctly documented.
        3. Establish or update a single source of truth (e.g., in `README.md` or a dedicated `DEVELOPMENT.md`) for common development and CI commands, emphasizing package-based execution.
    - **Done‑when:**
        1. All relevant project documentation uses `go run ./cmd/server` for running the application/migrations.
        2. A designated single source of truth for common commands is established, clear, and accurate.
    - **Verification:**
        1. Review the updated documentation for accuracy, clarity, and consistency.
        2. Ensure a new developer can easily find and use the correct commands by referencing the documentation.
    - **Depends‑on:** [T001]

### Backend (`cmd/server`)
- [x] **T004 · Refactor · P2: refactor 'cmd/server/main.go' for improved modularity and adherence to length guidelines**
    - **Context:** `CI Resolution Plan > Resolution Steps > Issue 4: (Recommended) Refactor cmd/server/main.go for Maintainability and Adherence to Standards`
    - **Action:**
        1. Identify logical sections within `cmd/server/main.go` (e.g., config loading, logger setup, database setup, router setup, application struct definition).
        2. Create new `.go` files in `cmd/server/` (e.g., `app.go`, `config_loader.go`, `logger_setup.go`) and move the relevant functions, types, and constants to these new files, ensuring each file starts with `package main`.
        3. Ensure `cmd/server/main.go` primarily contains the `main()` function orchestrating calls; run `go fmt ./cmd/server/...` and `goimports -w ./cmd/server`.
    - **Done‑when:**
        1. `cmd/server/main.go` is significantly shorter and primarily orchestrates calls to functions in other files.
        2. The `cmd/server` package compiles successfully with `go build ./cmd/server`.
        3. `golangci-lint run ./cmd/server/...` passes without new issues (including file length violations for `main.go`).
        4. The application runs as expected locally and the CI pipeline (with T001 & T002 fixes) passes with the refactored code.
    - **Verification:**
        1. Locally run `go build ./cmd/server` to confirm compilation.
        2. Run `golangci-lint run ./cmd/server/...` locally.
        3. Run the application locally using `go run ./cmd/server`.
        4. Confirm the CI pipeline passes with the refactored code.
    - **Depends‑on:** [T001, T002]

- [x] **T007 · Chore · P2: audit 'cmd/server' build tag usage against 'docs/BUILD_TAGS.md'**
    - **Context:** `CI Resolution Plan > Prevention Measures > Bullet 5 (Build Tag Policy Adherence)`
    - **Action:**
        1. Review all `.go` files within the `cmd/server` package and other core application logic areas.
        2. Compare current build tag usage with the policies outlined in `docs/BUILD_TAGS.md`.
        3. Identify any core application logic files inadvertently excluded by restrictive build tags or any other non-compliance.
    - **Done‑when:**
        1. Audit of build tag usage in `cmd/server` and core logic is complete.
        2. A summary of findings (compliance or list of non-compliant files/tags) is documented. (Further tickets to be created if non-compliance requires code changes).
    - **Verification:**
        1. Manually inspect files and compare build tag usage with the documented policy in `docs/BUILD_TAGS.md`.
    - **Depends‑on:** none

### Tooling
- [x] **T005 · Chore · P1: add 'go build ./cmd/server' pre-commit hook**
    - **Context:** `CI Resolution Plan > Prevention Measures > Bullet 2 (Strengthen Pre-commit Hooks)`
    - **Action:**
        1. Edit the `.pre-commit-config.yaml` file.
        2. Add a new hook that executes `go build -o /dev/null ./cmd/server` to verify the main application package builds.
    - **Done‑when:**
        1. The pre-commit hook configuration includes the `go build ./cmd/server` check.
        2. The hook successfully runs and passes on clean code, and fails if the `cmd/server` build is broken.
    - **Verification:**
        1. Run pre-commit hooks on a commit; ensure the new hook executes and passes.
        2. Temporarily introduce a build error in the `cmd/server` package, run pre-commit hooks, and verify the new hook fails, preventing the commit. Revert the error.
    - **Depends‑on:** [T002]

- [x] **T006 · Chore · P1: synchronize pre-commit 'golangci-lint' configuration with ci settings**
    - **Context:** `CI Resolution Plan > Prevention Measures > Bullet 2 (Strengthen Pre-commit Hooks)`
    - **Action:**
        1. Compare the `golangci-lint` configuration (e.g., `.golangci.yml`, command-line arguments, linter version) used in CI with the configuration in `.pre-commit-config.yaml`.
        2. Update the pre-commit hook configuration for `golangci-lint` to mirror the CI setup, including considerations for build tags and linter versions/settings.
    - **Done‑when:**
        1. The `golangci-lint` execution in pre-commit hooks uses the same effective configuration (version, linters, settings) as the CI pipeline.
    - **Verification:**
        1. Review both CI and pre-commit configurations for `golangci-lint` to confirm alignment.
        2. Ensure pre-commit linting passes/fails consistently with CI for the same code state by testing with code that triggers specific linters.
    - **Depends‑on:** none

- [x] **T009 · Feature · P3: develop local script to simulate key ci pipeline checks**
    - **Context:** `CI Resolution Plan > Prevention Measures > Bullet 7 (Local CI Simulation Script (Optional))`
    - **Action:**
        1. Create a new script (e.g., `scripts/run-ci-checks.sh` or a `Makefile` target like `make ci-local`).
        2. Implement commands in the script to execute key CI checks locally, such as linting (`golangci-lint`), formatting (`go fmt`), building the main application (`go build ./cmd/server`), and running tests (`go test ./...`).
        3. Document the usage of this script for developers.
    - **Done‑when:**
        1. A script exists that allows developers to run a defined subset of key CI checks locally with a single command.
        2. The script successfully executes these checks and reports pass/fail status accurately.
        3. Usage instructions for the script are added to project documentation (e.g., `README.md` or `DEVELOPMENT.md`).
    - **Verification:**
        1. Run the script locally on clean code and confirm it passes all checks.
        2. Introduce a lint error, then a test failure, then a build error, confirming the script catches each issue in turn.
    - **Depends‑on:** none

## CI Failure Resolution - 2025-05-17

### Critical Database Driver Fix
- [x] **T010 · Bugfix · P0: add missing pgx driver import to migrations executor**
    - **Context:** CI failing with `sql: unknown driver "pgx" (forgotten import?)` in `cmd/server/migrations_executor.go`
    - **Action:**
        1. Open file `cmd/server/migrations_executor.go`
        2. Add import statement: `import _ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver`
        3. Run `goimports` to fix import ordering
        4. Commit with message: `fix(db): add missing pgx driver import for migrations executor`
    - **Done‑when:**
        1. The pgx driver blank import is present and correctly formatted
        2. Database migrations execute successfully locally
        3. CI pipeline passes all checks
    - **Verification:**
        1. Run `go run ./cmd/server -migrate=status` locally without "unknown driver" error
        2. Execute `./scripts/run-ci-checks.sh` and confirm it passes
        3. Push fix and verify CI "Reset and prepare database" step passes
    - **Depends‑on:** none

### Build & Configuration Verification
- [x] **T011 · Chore · P1: audit build tags in cmd/server package**
    - **Context:** Ensure core application files don't have restrictive build tags preventing migration execution
    - **Action:**
        1. Review all files in `cmd/server/` package
        2. Check for build tags on: `app.go`, `config.go`, `logger.go`, `database.go`, `migrations_*.go`
        3. Remove any restrictive build tags from core logic files
        4. Ensure all files have `package main` declaration
    - **Done‑when:**
        1. Core application files have no restrictive build tags
        2. All migration-critical files are included in build
    - **Verification:**
        1. Run `go build ./cmd/server` with no "undefined" errors
    - **Depends‑on:** none

### CI & Testing Improvements
- [x] **T012 · Feature · P2: update local CI script with migration checks**
    - **Context:** Strengthen `scripts/run-ci-checks.sh` to catch database driver issues
    - **Action:**
        1. Add `go build ./cmd/server` to the script
        2. Add migration smoke test: `go run ./cmd/server -migrate=status`
        3. Ensure script fails on missing drivers or build errors
    - **Done‑when:**
        1. Script runs build verification
        2. Script runs migration smoke test
    - **Verification:**
        1. Run script locally and verify new checks execute
    - **Depends‑on:** [T010]

- [x] **T013 · Feature · P2: add migration smoke test to CI pipeline**
    - **Context:** Add early database connectivity check to CI workflow
    - **Action:**
        1. Edit CI workflow file (`.github/workflows/ci.yml`)
        2. Add "Migration Smoke Test" step before database reset
        3. Step should run: `go run ./cmd/server -migrate=status`
    - **Done‑when:**
        1. CI includes migration smoke test step
        2. Step runs before full database operations
    - **Verification:**
        1. Push changes and verify new CI step executes
    - **Depends‑on:** [T010]

### Documentation & Standards
- [x] **T014 · Docs · P3: update Go development philosophy with driver import guidance**
    - **Context:** Document requirement for database driver imports
    - **Action:**
        1. Edit `docs/DEVELOPMENT_PHILOSOPHY_APPENDIX_GO.md`
        2. Add section: "Any Go main package using database/sql MUST ensure required database drivers are explicitly imported via blank imports (`_ "driver/path"`) within that package's scope."
    - **Done‑when:**
        1. Documentation includes database driver import requirement
    - **Verification:**
        1. Review documentation for clarity
    - **Depends‑on:** none

- [x] **T015 · Docs · P3: update code review checklist**
    - **Context:** Add database driver and build tag checks to review process
    - **Action:**
        1. Locate project's code review checklist or create one
        2. Add item: "Verify database driver imports in files using sql.Open()"
        3. Add item: "Check for restrictive build tags on core application files"
    - **Done‑when:**
        1. Checklist includes both verification items
    - **Verification:**
        1. Document review confirms updates
    - **Depends‑on:** none

## CI Failure Resolution - 2025-05-18

### Test Execution Issues
- [ ] **T016 · Test · P0: retrieve and analyze full CI test logs**
    - **Context:** Immediate Actions Needed > Get Full Test Logs
    - **Action:**
        1. Download complete test logs for the failed CI run (Run ID: 15096488474)
        2. Identify specific failing tests, error messages, or compilation errors
        3. Create detailed failure analysis
    - **Done‑when:**
        1. Full test log is available and failing tests are clearly identified
        2. Root cause of test failures is documented
    - **Verification:**
        1. Specific failing tests are identified with error messages
    - **Depends‑on:** none

- [ ] **T017 · Test · P0: fix zero test coverage in internal/testutils packages**
    - **Context:** Root Cause Analysis > Issue 1: Zero Coverage in Test Utilities
    - **Action:**
        1. Review internal/testutils and subpackages for missing or non-executing tests
        2. Determine if these packages should have meaningful coverage or be excluded
        3. Either implement missing tests or properly exclude from coverage metrics
    - **Done‑when:**
        1. All testutils packages have appropriate test coverage or are explicitly excluded from coverage
    - **Verification:**
        1. Coverage report shows >0% if tests exist, or package is excluded from coverage calculations
    - **Depends‑on:** T016

- [ ] **T018 · Test · P0: unskip tests marked as skipped during refactoring**
    - **Context:** Root Cause Analysis > Tests being skipped during refactoring
    - **Action:**
        1. Identify all tests skipped due to "refactoring in progress"
        2. Either complete refactoring and re-enable, or implement temporary fixes to unblock CI
        3. Remove skip annotations and ensure tests run
    - **Done‑when:**
        1. No tests remain skipped due to refactoring
        2. All previously skipped tests execute and pass
    - **Verification:**
        1. Test output shows all previously skipped tests now run and report results
    - **Depends‑on:** T016

### CI Infrastructure
- [x] **T019 · Chore · P1: enable verbose test logging in CI**
    - **Context:** Immediate Actions Needed > Insufficient Log Information
    - **Action:**
        1. Update CI workflow to use verbose test flags (-v)
        2. Ensure test output is not truncated
        3. Configure log artifact retention for failed runs
    - **Done‑when:**
        1. CI test jobs output complete, verbose logs
        2. Logs are downloadable as artifacts after runs
    - **Verification:**
        1. Trigger a test run and confirm logs show all test details
    - **Depends‑on:** none

- [x] **T020 · Chore · P1: split CI test execution by package**
    - **Context:** Prevention Measures > Consider splitting test execution by package
    - **Action:**
        1. Update CI workflow to run tests per package
        2. Report individual package results and coverage
        3. Ensure failures in one package don't halt others
    - **Done‑when:**
        1. Each package's test results are reported independently in CI
        2. Package-level coverage metrics are visible
    - **Verification:**
        1. CI dashboard shows separate results for each package
    - **Depends‑on:** none

### Quality Gates
- [ ] **T021 · Chore · P1: implement pre-push hook for test execution**
    - **Context:** Prevention Measures > Implement pre-push hooks
    - **Action:**
        1. Configure pre-push hook that runs full test suite
        2. Include in .pre-commit-config.yaml
        3. Document installation in CONTRIBUTING.md
    - **Done‑when:**
        1. Developers cannot push if tests fail locally
        2. Hook is part of standard dev setup
    - **Verification:**
        1. Attempt push with broken test; confirm push is blocked
    - **Depends‑on:** none

- [ ] **T022 · Test · P1: enforce test coverage thresholds**
    - **Context:** Prevention Measures > Add test coverage thresholds
    - **Action:**
        1. Set minimum coverage thresholds in CI (70% overall)
        2. Configure per-package thresholds where appropriate
        3. Fail builds if coverage drops below threshold
    - **Done‑when:**
        1. CI fails when coverage is below threshold
        2. Coverage reports show which packages fail requirements
    - **Verification:**
        1. Lower coverage intentionally and confirm CI fails appropriately
    - **Depends‑on:** none

## Prevention Measures

1. Run dedicated CI-specific tests early in the pipeline
2. Enhance logging and observability in CI environment
3. Provide tools for developers to simulate CI environment locally
4. Regularly audit CI pipeline configuration and scripts
5. Ensure strict code review for environment-interacting code
6. Add file size limits to pre-commit hooks to prevent excessive file growth
7. Never bypass pre-commit hooks unless absolutely necessary
8. Run local build and linting checks before pushing code
