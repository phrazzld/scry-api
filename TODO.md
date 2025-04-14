# TODO

## Core Principles Issues
- [x] **Ignore or Delete Transient Plan Files:**
  - **Action:** Update the `.gitignore` file to include the pattern `*-PLAN.md` OR manually delete the existing `*-PLAN.md` files (e.g., `configure-slog-handler-options-PLAN.md`, `create-initial-logger-go-file-PLAN.md`, etc.) from the repository to reduce clutter. Ensure `PLAN.md`, `TODO.md`, `CODE_REVIEW.md` are also appropriately handled (ignored or removed if transient).
  - **Depends On:** None
  - **AC Ref:** Core Principles Issue 1 (Transient Plan Files Management)

- [x] **Remove Unnecessary `nolint:unused` Directive on `loggerKey`:**
  - **Action:** Remove the `// nolint:unused` comment from the `loggerKey` struct definition in `internal/platform/logger/logger.go:28` as the type is used.
  - **Depends On:** None
  - **AC Ref:** Core Principles Issue 2 (Unnecessary Directive)

## Architecture & Design Issues
- [x] **Align Log Level Validation and Implementation:**
  - **Action:** Review and update the log level validation logic in `internal/config/config.go:37` (validator tag `oneof=...`), the example in `config.yaml.example`, and any related documentation (e.g., Godoc in `logger.go`) to ensure they consistently reflect the supported log levels (debug, info, warn, error) and do not include "fatal".
  - **Depends On:** None
  - **AC Ref:** Architecture & Design Issue 1 (Log Level Validation vs. Implementation)

- [x] **Clarify Initial Logging Behavior:**
  - **Action:** Add comments in `cmd/server/main.go` near lines 18 and 24-26 explaining *why* the initial log messages use the default `slog` handler (plain text) before the custom JSON handler is configured during `initializeApp`.
  - **Depends On:** None
  - **AC Ref:** Architecture & Design Issue 2 (Initial Logging Before Setup)

## Testing Strategy Issues
- [ ] **Simplify Logger Test Setup:**
  - **Action:** Refactor the test setup in `internal/platform/logger/logger_test.go`. Prioritize using the `setupTestLogger` approach (direct buffer injection) over `setupLogCapture` (OS pipe redirection). Minimize or eliminate the use of the global `testFixture` variable and OS stream redirection (`os.Pipe`, `os.Stdout = ...`) to improve test isolation and simplicity.
  - **Depends On:** None
  - **AC Ref:** Testing Strategy Issue 1 (Test Setup Complexity)

- [ ] **Enhance Log Level Filtering Assertions in Tests:**
  - **Action:** Modify `TestValidLogLevelParsing` in `internal/platform/logger/logger_test.go` (around lines 542-547) to include assertions that verify log messages are correctly *filtered* based on the configured level. For example, when the level is "warn", assert that "debug" and "info" messages are *not* present in the output, while "warn" and "error" messages *are*.
  - **Depends On:** Simplify Logger Test Setup
  - **AC Ref:** Testing Strategy Issue 2 (Limited Assertions in Log Level Tests)

- [ ] **Remove Unnecessary `nolint:unused` Directives in Test Code:**
  - **Action:** Remove the `// nolint:unused` comments from the `stderrReader` and `stdoutReader` fields in the `testSetup` struct definition in `internal/platform/logger/logger_test.go` (lines 55-56) as they are intended for future use or potentially used indirectly. If any fields are truly unused after refactoring, remove them.
  - **Depends On:** Simplify Logger Test Setup
  - **AC Ref:** Testing Strategy Issue 3 (Unused Directives in Test Code)

## Code Quality Issues
- [ ] **Replace HTML Entities in Test Code Comments:**
  - **Action:** Search for and replace all instances of the HTML entity `&gt;` with the literal `>` character within comments in the `internal/platform/logger/logger_test.go` file.
  - **Depends On:** None
  - **AC Ref:** Code Quality Issue 1 (HTML Entity in Code Comments)

## Documentation Issues
- [ ] **Clarify Godoc for `logger.Setup` Error Return:**
  - **Action:** Update the Godoc comment for the `Setup` function in `internal/platform/logger/logger.go` (around lines 56-57) to explicitly state that the returned `error` is currently always `nil` but is included in the signature for potential future extensions (e.g., adding file logging which might fail).
  - **Depends On:** None
  - **AC Ref:** Documentation Issue 1 (Minor Godoc Clarity in Setup Function)

- [ ] **Fix Formatting Inconsistency in `BACKLOG.md`:**
  - **Action:** Review `BACKLOG.md` around line 13 (the "have pre-commit hook warn..." item) and adjust its formatting (e.g., indentation, bullet style) to be consistent with other top-level backlog items, or integrate it into a more relevant section if appropriate.
  - **Depends On:** None
  - **AC Ref:** Documentation Issue 2 (Backlog Formatting Inconsistency)

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS
- [ ] **Issue/Assumption:** The code review (`PLAN.md`) serves as the primary source of requirements for these TODO tasks.
  - **Context:** The instructions requested decomposition of `PLAN.md`, which contains code review feedback rather than a feature plan. Tasks are derived from the "Suggestion" or "Suggested Solution" fields.
- [ ] **Issue/Assumption:** The `Nil Context Handling` point (Code Quality #2) requires no action as the review notes it was already fixed.
  - **Context:** The review explicitly states "this was already fixed".
