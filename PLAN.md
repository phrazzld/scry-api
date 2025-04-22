# Remediation Plan – Sprint 25

## Executive Summary
This plan addresses critical issues identified in the Card Review API code review, focusing on production stability, developer experience, and maintainability. Starting with the critical mock LLM usage in production code, we'll then improve error handling, consolidate repository abstractions, and refactor tests. The changes are ordered to maximize stability gains while minimizing regression risks, targeting both immediate safety concerns and long-term maintainability.

## Strike List
| Seq | CR-ID | Title | Effort | Owner |
|-----|-------|-------|--------|-------|
| 1 | cr-01 | Remove Mock LLM from Main Application Path | s | backend |
| 2 | cr-02 | Replace Panics with Error Returns | xs | backend |
| 3 | cr-07 | Consolidate Repository Interfaces/Adapters | m | backend |
| 4 | cr-08 | Use Centralized Mock in Card Handler Tests | xs | backend |
| 5 | cr-03,04,05,06 | Refactor Card Review API Tests | m | backend |
| 6 | cr-09 | Add Deterministic DB Ordering | xs | backend |
| 7 | cr-18 | Consolidate Duplicate Test Helper | xs | backend |
| 8 | cr-14, cr-15 | Improve API Error Response Security | s | backend |
| 9 | cr-17 | Move Adapter Creation | s | backend |
| 10 | cr-10, cr-11, cr-12 | Harden Pre-commit Hook Environment | s | backend |
| 11 | cr-16 | Remove Unused Import Workaround | xs | backend |

## Detailed Remedies

### cr-01 Remove Mock LLM from Main Application Path
- **Problem:** The application uses mock LLM generator in the main application path, risking deployment with non-functional AI.
- **Impact:** Critical production failure if deployed; core flashcard generation functionality would use test data instead of actual AI-generated content.
- **Chosen Fix:** Use build tags to conditionally compile generator initialization based on the build environment.
- **Steps:**
  1. Create `internal/platform/gemini/generator_prod.go` (no build tag) containing `createGenerator` that returns the real implementation.
  2. Create `internal/platform/gemini/generator_test.go` (with `//go:build test_without_external_deps`) containing a test implementation.
  3. Update `cmd/server/main.go` to call `gemini.createGenerator()` instead of directly initializing a mock.
  4. Ensure CI/CD pipeline doesn't include the test build tag for production builds.
- **Done-When:** Production build uses real LLM; test build uses mock; no direct mock initialization in main.

### cr-02 Replace Panics with Error Returns
- **Problem:** Several constructors panic when provided with nil dependencies instead of returning errors.
- **Impact:** Application crashes during startup with minimal diagnostics if dependencies are misconfigured.
- **Chosen Fix:** Modify constructors to return errors instead of panicking, allowing for graceful handling.
- **Steps:**
  1. Update `NewCardReviewService` and `NewCardHandler` signatures to return `(..., error)`.
  2. Replace panic calls with `return nil, fmt.Errorf("dependency %s cannot be nil", "depName")`.
  3. Update all callers to handle potential errors.
  4. Add error handling in application setup to log detailed diagnostic information.
- **Done-When:** Application handles initialization errors gracefully with clear error messages.

### cr-07 Consolidate Repository Interfaces/Adapters
- **Problem:** Multiple repository interfaces and adapters create confusion and unnecessary abstraction layers.
- **Impact:** Increased complexity, overlapping responsibilities, unclear architectural boundaries.
- **Chosen Fix:** Consolidate repository interfaces by adding key methods to existing store interfaces.
- **Steps:**
  1. Add `GetForUpdate` method to `store.UserCardStatsStore` interface.
  2. Ensure Postgres implementation properly implements the new method.
  3. Remove separate repository adapters in the card_review package.
  4. Update `CardReviewService` to use the store interfaces directly.
  5. Update main.go to remove adapter creation and inject store implementations directly.
- **Done-When:** No separate repository interfaces exist; services use store interfaces directly.

### cr-08 Use Centralized Mock in Card Handler Tests
- **Problem:** Tests define local mock implementations instead of using the centralized mocks.
- **Impact:** Duplication of mock logic, inconsistent test behavior, maintenance burden.
- **Chosen Fix:** Replace local mocks with centralized mocks from the mocks package.
- **Steps:**
  1. Remove the local `mockCardReviewService` from `api/card_handler_test.go`.
  2. Import and use `mocks.MockCardReviewService` with functional options.
  3. Update assertions to use the standard mock's tracking capabilities.
- **Done-When:** All tests use centralized mocks consistently.

### cr-03,04,05,06 Refactor Card Review API Tests
- **Problem:** API tests are verbose with duplicate setup, manual validation, and complex assertions.
- **Impact:** Tests are difficult to maintain, understand, and extend.
- **Chosen Fix:** Leverage testutils helpers to simplify test code and improve maintainability.
- **Steps:**
  1. Replace manual server/router setup with `testutils.SetupCardReviewTestServer`.
  2. Replace manual HTTP request execution with helper methods like `ExecuteGetNextCardRequest`.
  3. Replace manual response validation with `AssertCardResponse` and similar helpers.
  4. Simplify service call count assertions.
- **Done-When:** Tests are significantly more concise, use consistent patterns, and remain comprehensive.

### cr-09 Add Deterministic DB Ordering
- **Problem:** The GetNextReviewCard SQL query relies on implicit ordering when timestamps match.
- **Impact:** Non-deterministic behavior in tests and production when cards have identical due times.
- **Chosen Fix:** Add a secondary sort key to ensure consistent ordering.
- **Steps:**
  1. Modify the ORDER BY clause to `ORDER BY ucs.next_review_at ASC, c.id ASC`.
  2. Add a comment explaining the reason for the secondary sort key.
  3. Create or update tests to verify deterministic ordering.
- **Done-When:** Query includes secondary sort key; tests confirm deterministic ordering.

### cr-18 Consolidate Duplicate Test Helper
- **Problem:** The `createCardWithStats` helper function is duplicated across test files.
- **Impact:** Violates DRY principle, potential for inconsistencies if logic needs updates.
- **Chosen Fix:** Move the helper to the testutils package.
- **Steps:**
  1. Create `CreateTestCardWithStats` in `internal/testutils/domain_helpers.go`.
  2. Remove duplicate implementations.
  3. Update test files to use the centralized helper.
- **Done-When:** No duplicate helper implementations exist; tests use the centralized helper.

### cr-14, cr-15 Improve API Error Response Security
- **Problem:** API handlers expose raw internal data and validation errors in responses.
- **Impact:** Information leakage useful to attackers; poor user experience.
- **Chosen Fix:** Return standardized, user-friendly error messages instead of raw errors.
- **Steps:**
  1. Update `cardToResponse` to handle unmarshal errors safely (return nil or generic error structure).
  2. Modify validation error handling to return user-friendly messages without internal details.
  3. Update tests to verify the improved error responses.
- **Done-When:** API responses contain user-friendly errors without exposing internal details.

### cr-17 Move Adapter Creation
- **Problem:** Repository adapters are created within router setup rather than during application initialization.
- **Impact:** Obscures dependency graph, violates dependency injection principles.
- **Chosen Fix:** Move adapter creation to application initialization (or remove adapters entirely if cr-07 is implemented).
- **Steps:**
  1. If adapters are still needed after cr-07, move their creation to startServer.
  2. Update setupRouter to accept fully constructed dependencies.
- **Done-When:** Clear dependency graph with adapters created during application initialization.

### cr-10, cr-11, cr-12 Harden Pre-commit Hook Environment
- **Problem:** Pre-commit hooks rely on user shell profiles and use hardcoded paths.
- **Impact:** Brittle development environment setup, inconsistent behavior across systems.
- **Chosen Fix:** Make hook scripts more self-contained and portable.
- **Steps:**
  1. Remove shell profile sourcing from run_glance.sh.
  2. Use mktemp for temporary files instead of hardcoded paths.
  3. Define minimal PATH setup within the script.
  4. Test the hook in different environments.
- **Done-When:** Hooks run consistently across different environments without user-specific configuration.

### cr-16 Remove Unused Import Workaround
- **Problem:** A variable is defined solely to suppress an "imported and not used" error.
- **Impact:** Code clarity issue, obscures the actual reason for the import.
- **Chosen Fix:** Remove the unnecessary workaround.
- **Steps:**
  1. Remove `sqlRef := sql.IsolationLevel(0)` and related code.
  2. Verify code compiles correctly.
- **Done-When:** Workaround is removed; code compiles without warnings.

## Standards Alignment
- **Simplicity First:** Consolidating repository interfaces (cr-07) and removing unnecessary code (cr-16) directly improve simplicity. Refactoring tests (cr-03-06) makes them easier to understand and maintain.
- **Modularity:** Fixing mock LLM usage (cr-01) enforces architectural boundaries. Moving adapter creation (cr-17) improves dependency injection clarity.
- **Design for Testability:** Comprehensive test improvements (cr-03-06, cr-08, cr-18) enable easier testing and maintenance. Using build tags for LLM initialization (cr-01) allows proper testing without affecting production.
- **Error Handling:** Replacing panics with errors (cr-02) and improving API error responses (cr-14, cr-15) align with best practices for robust error handling.
- **Security:** Fixing mock LLM usage (cr-01) prevents potential production issues. Improving API error responses (cr-14, cr-15) reduces information leakage.

## Validation Checklist
- [ ] All automated tests pass (`go test ./...` and with test tags).
- [ ] Static analysis passes (`golangci-lint run`) with no new issues.
- [ ] Build succeeds without test tags and uses real LLM integration.
- [ ] Build with test tags uses mock LLM integration.
- [ ] API endpoints return user-friendly errors for validation failures.
- [ ] Pre-commit hooks run successfully in a clean environment.
- [ ] No panics during application startup with invalid configuration.
- [ ] SQL query ordering is deterministic for cards with identical due dates.
