# TODO

*This Todo List is managed by the claude.ai/code agent. Do not update directly.*

## Recent Completed Tasks

- [x] **T023 · Test · P1: Fix failing redaction tests**
    - **Context:** Pre-push hook failing due to redaction test failures
    - **Action:** Updated SQL redaction patterns to fix failing tests
    - **Verification:** Tests in internal/redact package now pass

- [x] **T024 · Test · P1: Fix API error redaction tests**
    - **Context:** API error redaction tests failing after changes
    - **Action:** Updated error_redaction_test.go to be compatible with new patterns
    - **Verification:** All API error redaction tests now pass

## CI Workflow (2025-05-20)

- [x] **T001 · Bugfix · P0: update CI migration command to use `go run ./cmd/server`**
    - **Context:** CI failures due to refactoring cmd/server into multiple files
    - **Action:**
        1. Locate the CI workflow step responsible for database migrations
        2. Change the command from `go run ./cmd/server/main.go -migrate=...` to `go run ./cmd/server -migrate=...`
    - **Done‑when:** The CI migration step completes successfully using the updated command
    - **Verification:** Review CI logs to confirm the command works and migration step passes
    - **Depends‑on:** none

- [x] **T002 · Bugfix · P0: add early build verification step for `cmd/server` in CI**
    - **Context:** CI failures due to build issues not caught early in the pipeline
    - **Action:** Add a new step in the CI workflow to build the main application using `go build ./cmd/server`
    - **Done‑when:** CI pipeline includes and passes the build verification step
    - **Verification:** Confirm the CI pipeline fails early when introducing a build error
    - **Depends‑on:** none

- [x] **T003 · Bugfix · P1: enable CGo via `CGO_ENABLED=1` for CI integration tests**
    - **Context:** Database tests failing due to disabled CGo in CI
    - **Action:** Add the environment variable `CGO_ENABLED=1` to relevant CI jobs/steps
    - **Done‑when:** CGo is enabled for database-dependent test execution
    - **Verification:** CI logs confirm `CGO_ENABLED=1` is active for test steps
    - **Depends‑on:** none

- [x] **T004 · Chore · P1: ensure CI runner has required C libraries for CGo**
    - **Context:** Database tests failing due to missing C libraries for CGo
    - **Action:** Verify and install required C libraries (`gcc`, `libpq-dev`) in CI environment
    - **Done‑when:** CGo-dependent packages can successfully compile in CI
    - **Verification:** Test for presence of required libraries in CI environment
    - **Depends‑on:** none

- [x] **T005 · Chore · P1: improve CI test logging and error reporting**
    - **Context:** CI failures providing insufficient diagnostic information
    - **Action:** Increase test verbosity and ensure error messages are captured
    - **Done‑when:** CI test logs provide detailed information about failures
    - **Verification:** Review CI logs for improved clarity and detail
    - **Depends‑on:** none

- [x] **T006 · Chore · P1: add CI artifacts for failed test runs**
    - **Context:** Diagnostic information from CI failures not easily accessible
    - **Action:** Configure CI to upload test logs and reports as artifacts on failure
    - **Done‑when:** Artifacts are available for download on failed CI runs
    - **Verification:** Confirm artifacts are uploaded when tests fail
    - **Depends‑on:** none

## Test Environment & Failures

- [ ] **T007 · Bugfix · P1: debug database URL standardization in CI environment**
    - **Context:** Potential issues with database URL construction in CI
    - **Action:** Review database connection URL construction and parameters
    - **Done‑when:** Database connection logic correctly handles URL parameters
    - **Verification:** Tests that previously failed due to URL issues now pass
    - **Depends‑on:** [T003, T004]

- [ ] **T008 · Test · P1: address remaining specific test failures**
    - **Context:** Test failures not resolved by environment configuration fixes
    - **Action:** Investigate and fix root causes of persistent test failures
    - **Done‑when:** All previously failing tests now pass in CI
    - **Verification:** CI pipeline shows all test suites passing
    - **Depends‑on:** [T001, T002, T003, T004, T007]

## Documentation & Developer Tooling

- [ ] **T012 · Chore · P2: standardize `go run ./cmd/server` command**
    - **Context:** Inconsistent usage of command to run the server
    - **Action:** Replace all instances of `go run ./cmd/server/main.go` with `go run ./cmd/server`
    - **Done‑when:** All documentation and scripts use the standardized command
    - **Verification:** No old command instances remain in docs/scripts
    - **Depends‑on:** none

- [ ] **T013 · Chore · P2: add pre-commit hook for `go build ./cmd/server`**
    - **Context:** Build issues not caught before commits
    - **Action:** Implement a pre-commit hook that runs `go build ./cmd/server`
    - **Done‑when:** The pre-commit hook blocks commits if build fails
    - **Verification:** Test that commits with build errors are prevented
    - **Depends‑on:** none

- [x] **T014 · Chore · P2: document CGo requirements**
    - **Context:** Undocumented CGo dependencies causing test failures
    - **Action:** Document CGo requirements and necessary C libraries
    - **Done‑when:** Requirements are clearly documented for developers
    - **Verification:** Documentation review confirms completeness
    - **Depends‑on:** none

- [ ] **T015 · Feature · P2: create local CI simulation script**
    - **Context:** Difficulty replicating CI environment locally
    - **Action:** Create a script to run key CI checks locally
    - **Done‑when:** Script successfully simulates CI checks
    - **Verification:** Script catches the same issues as CI would
    - **Depends‑on:** none

- [ ] **T016 · Chore · P2: update code review checklist**
    - **Context:** Code reviews not catching potential CI issues
    - **Action:** Add CI/build considerations to review checklist
    - **Done‑when:** Checklist includes new CI-related checks
    - **Verification:** Review the updated guidelines
    - **Depends‑on:** none

## Prevention Best Practices

1. Run CI-specific tests early in the pipeline
2. Enhance logging in CI environment
3. Simulate CI environment locally before pushing
4. Never bypass pre-commit hooks
5. Run build and test checks before pushing code
