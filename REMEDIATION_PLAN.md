```md
# Remediation Plan – Sprint <n>

## Executive Summary
This plan prioritizes immediate fixes for critical blockers preventing server startup (cr-02) and hindering effective testing (cr-04, cr-03, cr-01). We then address high-impact issues undermining dependency management (cr-06), dependency injection clarity (cr-07), and test coverage (cr-05). The sequence tackles quick wins first, followed by foundational test framework repairs, enabling the core policy enforcement (cr-01) and subsequent test additions (cr-05), ensuring a stable, testable, and maintainable state aligned with our development philosophy.

## Strike List
| Seq | CR‑ID | Title                                                       | Effort | Owner?  |
|:----|:------|:------------------------------------------------------------|:-------|:--------|
| 1   | cr‑02 | Fix Initialization Order Bug Prevents Server Start          | xs     | backend |
| 2   | cr‑04 | Fix Missing t.Helper() Call in Test Rollback Obscures Failures | xs     | backend |
| 3   | cr‑07 | Eliminate Awkward Dependency Injection Cast in main.go      | s      | backend |
| 4   | cr‑06 | Reduce Massive Dependency Bloat Introduced                  | s      | backend |
| 5   | cr‑03 | Fix Test Transaction Helper (TxDB) Violates DBTX Contract   | s      | backend |
| 6   | cr‑09 | Remove Redundant Test Transaction Helpers                   | s      | backend |
| 7   | cr‑08 | Standardize Inconsistent Test Build Tags                    | s      | backend |
| 8   | cr‑01 | Eliminate Mocking of Internal Components Violates Core Testing Policy | l      | backend |
| 9   | cr‑05 | Add Missing Integration Tests for New Service Logic           | m      | backend |
| 10  | cr‑10 | Refactor Duplicate Request Handling Logic in API Handlers     | s      | backend |
| 11  | cr‑11 | Split Oversized Test File (`card_management_api_test.go`) | s      | backend |
| 12  | cr‑13 | Remove Hardcoded Test Password Hash                         | xs     | backend |
| 13  | cr‑15 | Fix Non-idiomatic Test Mock Naming                          | xs     | backend |
| 14  | cr‑14 | Fix Comment Typos                                           | xs     | backend |
| 15  | cr‑12 | Trim Overly Verbose GoDoc Comments                          | s      | backend |
| 16  | cr‑16 | Ensure Logging Includes Correlation IDs                     | s      | backend |

## Detailed Remedies

### cr‑02 Fix Initialization Order Bug Prevents Server Start
- **Problem:** `cardService` is instantiated before its dependency `srsService` in `startServer`.
- **Impact:** Runtime panic/failure due to nil dependency injection, preventing server startup. BLOCKER.
- **Chosen Fix:** Reorder instantiation in `startServer`.
- **Steps:**
  1. In `cmd/server/main.go`, locate the `startServer` function.
  2. Move the block initializing `srsService` to *before* the block initializing `cardService`.
  3. Ensure the valid `srsService` instance is passed to `service.NewCardService`.
- **Done‑When:** Server starts without panic related to nil `srsService` dependency. CI startup checks pass.

### cr‑04 Fix Missing t.Helper() Call in Test Rollback Obscures Failures
- **Problem:** The deferred transaction rollback in `testutils.WithTx` lacks `t.Helper()`.
- **Impact:** Failed rollbacks report errors pointing to the `defer` line, not the test code causing the issue, hindering debugging. BLOCKER.
- **Chosen Fix:** Create/reinstate a helper function with `t.Helper()` for rollback.
- **Steps:**
  1. Define `AssertRollbackNoError(t *testing.T, tx *sql.Tx)` in `internal/testutils/db.go`.
  2. Place `t.Helper()` at the start of this function.
  3. Include the rollback logic (`tx.Rollback()`) and appropriate error checking (`err != sql.ErrTxDone`) inside.
  4. Replace the inline `defer tx.Rollback()` in `WithTx` with `defer AssertRollbackNoError(t, tx)`.
- **Done‑When:** Rollback errors in tests correctly point to the specific test function line number.

### cr‑07 Eliminate Awkward Dependency Injection Cast in main.go
- **Problem:** `deps.CardService` is retrieved and immediately type-asserted (`deps.CardService.(service.CardService)`), indicating the DI container stores it as `interface{}` or a mismatched type.
- **Impact:** Reduces compile-time type safety, obscures wiring errors until runtime, complicates understanding the dependency graph. HIGH severity.
- **Chosen Fix:** Define the `CardService` field in `appDependencies` with its specific interface type.
- **Steps:**
  1. Modify the `appDependencies` struct definition in `cmd/server/main.go`.
  2. Change the type of the `CardService` field from `interface{}` (or other incorrect type) to `service.CardService`.
  3. Ensure the `CardService` field is populated with the correct type during setup in `startServer`.
  4. Remove the type assertion `.(service.CardService)` in `setupRouter`.
- **Done‑When:** `appDependencies.CardService` field has the specific type `service.CardService`, the type assertion is removed, and the application compiles and runs correctly.

### cr‑06 Reduce Massive Dependency Bloat Introduced
- **Problem:** `go.mod` and `go.sum` have significantly increased, likely due to conflicting versions (e.g., pgx/v4 and v5) or unnecessary transitive dependencies.
- **Impact:** Increases build times, vulnerability surface area, and maintenance overhead. Violates dependency management standards. HIGH severity.
- **Chosen Fix:** Audit and tidy dependencies.
- **Steps:**
  1. Run `go mod tidy`.
  2. Inspect `go.mod` for explicit dependencies that are no longer needed or cause conflicts (e.g., remove `pgx/v4` if `pgx/v5` is the standard).
  3. Use `go mod graph` or `go mod why` to understand why unexpected transitive dependencies are included and address the root cause if possible.
  4. Ensure test-only dependencies are correctly managed (usually handled by `go mod tidy`).
- **Done‑When:** `go mod tidy` runs cleanly, `go.mod` reflects only necessary direct dependencies, and the number of indirect dependencies is minimized and justified. CI build times potentially decrease.

### cr‑03 Fix Test Transaction Helper (TxDB) Violates DBTX Contract
- **Problem:** The `testutils.TxDB` wrapper's `Begin()`/`BeginTx()` methods incorrectly return the existing transaction, violating the `DBTX` interface contract.
- **Impact:** Prevents realistic testing of nested/managed transactions, hides potential bugs, limits integration test effectiveness. BLOCKER.
- **Chosen Fix:** Remove the `TxDB` wrapper and pass `*sql.Tx` directly via `WithTx`.
- **Steps:**
  1. Delete the `TxDB` struct definition and methods from `internal/testutils/api_helpers.go`.
  2. Modify the `testutils.WithTx` function signature: change the test function parameter from `fn func(tx store.DBTX)` to `fn func(t *testing.T, tx *sql.Tx)`.
  3. Update all call sites of `testutils.WithTx` to match the new signature.
  4. Inside the test function `fn` passed to `WithTx`, obtain transactional store/repository instances by calling their respective `WithTx(tx)` methods, passing the received `*sql.Tx`.
- **Done‑When:** `TxDB` type is removed. `testutils.WithTx` passes `*sql.Tx` directly. Tests compile and correctly use the provided transaction to obtain transactional stores/repositories.

### cr‑09 Remove Redundant Test Transaction Helpers
- **Problem:** `internal/platform/postgres/card_store_crud_test.go` defines local helpers (`withTxForCardTest`, `getTestDBForCardStore`) duplicating functionality in `testutils`.
- **Impact:** Violates DRY, adds unnecessary code, risks divergence from standard test setup. MEDIUM severity.
- **Chosen Fix:** Consolidate on standard `testutils` helpers.
- **Steps:**
  1. Delete the local helper functions `withTxForCardTest` and `getTestDBForCardStore`.
  2. Refactor tests within `card_store_crud_test.go` to use `testutils.GetTestDBWithT` and `testutils.WithTx` directly.
- **Done‑When:** Redundant helpers are removed, and tests use the canonical helpers from `internal/testutils`.

### cr‑08 Standardize Inconsistent Test Build Tags
- **Problem:** Inconsistent build tags (`test_without_external_deps` vs. none) for tests requiring database interaction.
- **Impact:** Creates confusion, hinders reliable selective test execution (e.g., separating unit from integration tests). MEDIUM severity.
- **Chosen Fix:** Standardize on a single, meaningful build tag for integration tests.
- **Steps:**
  1. Define a standard build tag for tests requiring external dependencies like a database (e.g., `integration`).
  2. Apply `//go:build integration` consistently to the top of all test files that interact with the database (e.g., `internal/platform/postgres/*_test.go`, `internal/service/*_tx_test.go`, relevant `cmd/server/*_api_test.go`).
  3. Remove the misleading `test_without_external_deps` tag.
  4. Update CI/CD scripts and documentation to recognize and utilize the `integration` tag for running these tests separately if needed.
- **Done‑When:** All database-dependent tests consistently use the `integration` build tag. CI can selectively run unit vs. integration tests.

### cr‑01 Eliminate Mocking of Internal Components Violates Core Testing Policy
- **Problem:** Tests mock internal interfaces (`CardService`, `CardRepository`, `StatsRepository`, `SRSService`), violating the "NO Mocking Internal Collaborators" policy.
- **Impact:** Leads to brittle tests, hides coupling, potentially masks design flaws, and bypasses true integration verification. CI *must* reject this. BLOCKER.
- **Chosen Fix:** Eliminate all mocks for internal collaborators; refactor tests to use real implementations.
- **Steps:**
  1. Delete all mock types and files associated with internal interfaces (e.g., `MockCardService`, `MockCardRepository`, `internal/mocks/...`, `internal/service/mocks_test.go`).
  2. Systematically refactor tests in `internal/api`, `internal/service`:
      *   Replace mock instantiations with instantiations of *real* services and repositories.
      *   Use test database fixtures (`testutils.GetTestDBWithT`, `testutils.WithTx`) to provide real stores/repositories to services under test.
      *   For API handler tests, provide real service implementations to the handlers.
      *   Focus test assertions on observable behavior and state changes (e.g., database state, HTTP responses, returned values/errors) rather than mock interactions.
  3. Reserve mocks *only* for true external system boundaries if absolutely necessary (e.g., a hypothetical external payment gateway API), ideally tested at a higher system-test level.
- **Done‑When:** No mocks for internal interfaces exist in the codebase. Unit/integration tests use real collaborators. CI passes policy checks.

### cr‑05 Add Missing Integration Tests for New Service Logic
- **Problem:** Transactional behavior and database interactions of `UpdateCardContent`, `DeleteCard`, `PostponeCard` are untested against a real database.
- **Impact:** Risks undetected data integrity issues, concurrency bugs, incorrect ownership logic, and faulty transactional behavior in production. HIGH severity.
- **Chosen Fix:** Add integration tests using `testutils.WithTx` and real store implementations.
- **Steps:**
  1. In `internal/service/card_service_tx_test.go` (or create if needed), add test functions for `UpdateCardContent`, `DeleteCard`, `PostponeCard`.
  2. Use `testutils.WithTx` (which now passes `*sql.Tx` after cr-03 fix).
  3. Inside the test function, instantiate the real `service.CardService` using real `postgres` store implementations initialized with the provided `*sql.Tx`.
  4. Mocking the `srsService` dependency *might* be acceptable here if its logic is complex and tested separately, but database interactions *must* use real stores.
  5. Cover happy paths and key error conditions (ownership failure, not found errors, invalid input for Postpone).
  6. Assert expected database state changes and return values/errors within the transaction.
  7. Ensure tests are tagged with `//go:build integration`.
- **Done‑When:** New integration tests cover the specified service methods, verifying database interactions and logic against a real database. Test coverage metrics improve.

### cr‑10 Refactor Duplicate Request Handling Logic in API Handlers
- **Problem:** Repeated code in API handlers (`EditCard`, `DeleteCard`, `PostponeCard`) for common tasks like extracting path parameters, parsing UUIDs, and getting user ID from context.
- **Impact:** Violates DRY, increases maintenance burden, risks inconsistent error handling. MEDIUM severity.
- **Chosen Fix:** Extract common logic into shared helper functions within `internal/api`.
- **Steps:**
  1. Identify the common blocks of code (e.g., `chi.URLParam`, `uuid.Parse`, `auth.UserIDFromContext`).
  2. Create helper functions in `internal/api` (e.g., `getUserID(r *http.Request) (uuid.UUID, error)`, `getPathUUID(r *http.Request, paramName string) (uuid.UUID, error)`). These helpers should handle parsing and potentially return standard errors.
  3. Refactor the `EditCard`, `DeleteCard`, `PostponeCard` handlers to call these helpers instead of duplicating the logic.
- **Done‑When:** Duplicate request setup code is removed from handlers and replaced by calls to shared helper functions.

### cr‑11 Split Oversized Test File (`card_management_api_test.go`)
- **Problem:** `cmd/server/card_management_api_test.go` is excessively long (819 lines), hindering readability and maintainability.
- **Impact:** Difficult to navigate, understand, and modify tests. Violates file length guidelines. MEDIUM severity.
- **Chosen Fix:** Refactor by splitting tests for each endpoint into separate files.
- **Steps:**
  1. Create new test files within `cmd/server/` named after the endpoints they test (e.g., `edit_card_api_test.go`, `delete_card_api_test.go`, `postpone_card_api_test.go`).
  2. Move the test functions related to `PUT /cards/{id}` into `edit_card_api_test.go`, `DELETE /cards/{id}` into `delete_card_api_test.go`, etc.
  3. Identify common setup functions or test helpers used across these tests and move them into a shared file (e.g., `cmd/server/api_test_helpers_test.go`) if appropriate, or keep them within each file if simple enough.
  4. Ensure all new files have the correct package declaration (`package main_test`) and necessary imports.
- **Done‑When:** `card_management_api_test.go` is significantly smaller, and tests are logically grouped into separate files by endpoint.

### cr‑13 Remove Hardcoded Test Password Hash
- **Problem:** A test uses a fixed bcrypt hash.
- **Impact:** Minor security consideration (less ideal than dynamic generation), slight brittleness if bcrypt parameters change. LOW severity.
- **Chosen Fix:** Dynamically hash passwords during test setup.
- **Steps:**
  1. Modify the test helper function responsible for creating test users (e.g., `createTestUser` in `cmd/server/card_management_api_test.go`).
  2. Change it to accept a plain text password string as an argument.
  3. Inside the helper, use `bcrypt.GenerateFromPassword` to hash the provided password string.
  4. Store the resulting hash in the user data before inserting it into the test database.
- **Done‑When:** Tests generate user password hashes dynamically at runtime.

### cr‑15 Fix Non-idiomatic Test Mock Naming
- **Problem:** Inconsistent naming conventions for mock types (e.g., `mockCardService` vs `MockCardService`).
- **Impact:** Violates Go naming standards, minor inconsistency. LOW severity. *(Note: This becomes less relevant after cr-01 removes internal mocks, but should be applied to any remaining valid mocks, e.g., for external systems).*
- **Chosen Fix:** Use standard Go naming conventions.
- **Steps:**
  1. Identify any remaining mock types (after cr-01).
  2. Rename exported mock types to `MockXxx` (e.g., `MockExternalAPI`).
  3. Rename unexported mock types (if any, less common) to `mockXxx`.
- **Done‑When:** All remaining mock types follow standard Go naming conventions.

### cr‑14 Fix Comment Typos
- **Problem:** Minor spelling errors in code comments (e.g., "reques", "defaul").
- **Impact:** Reduces clarity and professionalism. LOW severity.
- **Chosen Fix:** Correct spelling errors.
- **Steps:**
  1. Locate the typos mentioned in the review (`internal/testutils/card_api_helpers.go`).
  2. Correct the spelling errors in the comments.
  3. Perform a quick search for other common typos if time permits.
- **Done‑When:** Specified typos are corrected.

### cr‑12 Trim Overly Verbose GoDoc Comments
- **Problem:** GoDoc comments restate obvious type information or mechanics, cluttering code and documentation. Inconsistent formatting.
- **Impact:** Reduces signal-to-noise ratio in documentation. LOW severity.
- **Chosen Fix:** Trim comments to focus on non-obvious aspects and standardize format.
- **Steps:**
  1. Review GoDoc comments in specified locations (`internal/api/card_handler.go` DTOs, `internal/store/card.go`, `internal/store/stats.go`).
  2. Remove comments that merely repeat the type signature (e.g., `// ID is the unique identifier`).
  3. Focus comments on *why* something exists, its purpose, or non-obvious constraints/behavior.
  4. Ensure consistent formatting for parameters/returns (e.g., using lists or consistent indentation).
- **Done‑When:** GoDoc comments are concise, informative, consistently formatted, and add value beyond the type signature.

### cr‑16 Ensure Logging Includes Correlation IDs
- **Problem:** Log lines may be missing a consistent `trace_id` or correlation ID.
- **Impact:** Makes tracing requests across logs difficult, hindering debugging and observability. LOW severity.
- **Chosen Fix:** Ensure trace IDs are consistently injected and logged via context.
- **Steps:**
  1. Verify trace ID middleware (e.g., `apiMiddleware.NewTraceMiddleware`) correctly adds a trace ID to the request `context.Context`.
  2. Confirm that logger retrieval functions (e.g., `logger.FromContextOrDefault`) extract the trace ID from the context and add it as a structured field (`slog.String("trace_id", id)`).
  3. Audit key code paths (API handlers, service methods) to ensure the `context.Context` is passed down and used when retrieving the logger.
- **Done‑When:** Structured logs generated during request processing consistently include the `trace_id` field.

## Standards Alignment
- **Testability:** Directly addressed by cr-01 (removing internal mocks), cr-03 (fixing TxDB), cr-04 (fixing rollback helper), cr-05 (adding integration tests), cr-08 (build tags), cr-09 (removing redundant helpers). Enforces the core testing philosophy.
- **Simplicity:** Addressed by cr-07 (removing DI cast), cr-09 (removing redundancy), cr-10 (DRY helpers), cr-11 (smaller files), cr-12 (concise docs).
- **Modularity:** Addressed by cr-01 (exposing real boundaries), cr-05 (testing service boundaries), cr-10 (API helpers).
- **Coding Standards:** Addressed by cr-06 (deps), cr-07 (DI pattern), cr-08 (build tags), cr-11 (file length), cr-15 (naming), cr-14 (typos).
- **Dependency Management:** Directly addressed by cr-06.

## Validation Checklist
- [ ] All automated tests (unit, integration) pass in CI.
- [ ] `golangci-lint` and other static analysis tools pass without new errors.
- [ ] Server starts successfully without initialization panics (cr-02).
- [ ] No mocks for internal project interfaces remain (cr-01).
- [ ] Test transaction helpers (`testutils.WithTx`) function correctly, passing `*sql.Tx` (cr-03).
- [ ] Test rollback errors point to the correct test code line (cr-04).
- [ ] `go mod tidy` reports no changes; dependency graph is clean (cr-06).
- [ ] No type assertions used for core service dependencies in DI setup (cr-07).
- [ ] Integration tests cover new service logic (cr-05).
- [ ] Standard build tag (`integration`) applied consistently (cr-08).
- [ ] Code duplication reduced in API handlers (cr-10) and test helpers (cr-09).
- [ ] Oversized test file split (cr-11).
- [ ] Log entries include `trace_id` where applicable (cr-16).
- [ ] Minor cleanups complete (cr-12, cr-13, cr-14, cr-15).
```
