```markdown
# Remediation Plan – Sprint <n>

## Executive Summary
This plan targets critical code review findings undermining testability, modularity, security, and robustness. We will first eliminate contradictory mock tests (cr-01) and consolidate fragmented test helpers (cr-02) to establish a sane testing foundation. Subsequently, we address high-severity risks: a potential nil pointer panic (cr-04), inconsistent service error handling (cr-05), and insufficient SQL redaction (cr-03), restoring stability and security.

## Strike List
| Seq | CR‑ID | Title                                              | Effort | Owner?  |
|-----|-------|----------------------------------------------------|--------|---------|
| 1   | cr‑02 | Consolidate Test Helper Sprawl                     | m      | backend |
| 2   | cr‑01 | Remove Contradictory Mock-Based API Tests          | s      | backend |
| 3   | cr‑04 | Fix Potential Nil Pointer Panic in CardRepositoryAdapter | s      | backend |
| 4   | cr‑05 | Standardize Service Transaction Error Handling     | m      | backend |
| 5   | cr‑03 | Improve SQL Redaction for Sensitive Data Leakage   | m      | backend |

## Detailed Remedies

### cr‑02 Consolidate Test Helper Sprawl
- **Problem:** Test setup logic (DB, API server, data creation) is duplicated and scattered across `cmd/server`, `internal/testutils`, `internal/testdb`.
- **Impact:** High maintenance burden, risk of divergence, hinders test writing/understanding; violates `Mandatory Modularity`, `Simplicity First`.
- **Chosen Fix:** Consolidate all reusable test helpers into the `internal/testutils` package, using sub-packages (e.g., `db`, `api`) for organization. Delete helpers from `cmd/server/`. Refactor all tests to use this single source.
- **Steps:**
  1. Identify all reusable helpers in `cmd/server/api_test_helpers_test.go`, `cmd/server/auth_test_helpers.go`, `internal/testutils/`, `internal/testdb/`.
  2. Create sub-packages `internal/testutils/db` and `internal/testutils/api`.
  3. Move and refactor identified helpers into the appropriate `internal/testutils` sub-package, ensuring consistency and clarity.
  4. Delete `cmd/server/api_test_helpers_test.go` and `cmd/server/auth_test_helpers.go`.
  5. Refactor all tests (unit and integration) across the codebase to import and use helpers exclusively from `internal/testutils/...`.
- **Done‑When:** Helper files in `cmd/server` are deleted. All tests compile and pass using only helpers from `internal/testutils`. CI green.

### cr‑01 Remove Contradictory Mock-Based API Tests
- **Problem:** New API tests (`auth_api_test.go`, `card_api_test.go`) use extensive internal mocks, violating the "no internal mocking" policy and contradicting the integration test strategy.
- **Impact:** Fragile, misleading tests; high maintenance cost; undermines test reliability and architectural discipline; violates `Design for Testability`, `Simplicity First`, `Coding Standards (Consistency)`.
- **Chosen Fix:** Delete the mock-based API test files (`cmd/server/auth_api_test.go`, `cmd/server/card_api_test.go`). Ensure equivalent or better coverage exists via integration tests using real dependencies.
- **Steps:**
  1. **Prerequisite:** `cr-02` (Helper Consolidation) must be complete.
  2. Delete `cmd/server/auth_api_test.go`.
  3. Delete `cmd/server/card_api_test.go`.
  4. Audit existing integration tests (`cmd/server/auth_integration_test.go`, `cmd/server/card_api_integration_test.go`, etc.) using coverage tools to identify any critical scenarios previously covered only by the deleted mock tests.
  5. Add necessary test cases to the *integration* tests to fill identified coverage gaps, using the consolidated helpers from `internal/testutils`.
- **Done‑When:** Mock-based API tests are deleted. Integration tests provide verified, equivalent (or better) coverage for the relevant API endpoints. CI green.

### cr‑04 Fix Potential Nil Pointer Panic in CardRepositoryAdapter
- **Problem:** `PostgresCardStore.DB()` can return `nil` when the store is created via `WithTx`, risking a panic if `BeginTx` is called on this nil value (e.g., within `store.RunInTransaction`).
- **Impact:** Runtime panic leading to application crash under specific transactional scenarios; violates `Coding Standards (Robustness)`.
- **Chosen Fix:** Refactor `PostgresCardStore` and its `WithTx` method to always retain and propagate the original `*sql.DB` connection reference, ensuring `DB()` never returns nil if the store was initialized with a valid DB connection.
- **Steps:**
  1. Modify `internal/platform/postgres/card_store.go`:
     - Ensure the `PostgresCardStore` struct has a field to reliably store the original `*sql.DB` (e.g., `sqlDB *sql.DB`).
     - Update `NewPostgresCardStore` to initialize this field correctly.
     - Update the `WithTx` method to copy the `sqlDB` field value from the original store to the new transactional store instance.
     - Ensure the `DB()` method returns the stored `sqlDB` field.
  2. Add/verify unit tests for `PostgresCardStore` specifically testing that `DB()` returns a non-nil `*sql.DB` instance after calling `WithTx` on a store originally created with a valid `*sql.DB`.
- **Done‑When:** `PostgresCardStore.DB()` never returns nil when initialized with a valid DB. Tests verify this behavior under transactional conditions. CI green.

### cr‑05 Standardize Service Transaction Error Handling
- **Problem:** Error handling within `store.RunInTransaction` callbacks across different services is inconsistent—mixing direct returns of store/sentinel errors with wrapped service-specific errors.
- **Impact:** Callers (like API handlers) cannot reliably check for specific error types using `errors.Is`/`errors.As`; violates `Coding Standards (Consistency)`, `Error Handling` principles.
- **Chosen Fix:** Standardize by *always* wrapping errors returned from the transaction function within service methods, unless returning a well-defined service-level sentinel error (e.g., `service.ErrNotFound`, `service.ErrNotOwned`).
- **Steps:**
  1. Audit all usages of `store.RunInTransaction` within the `internal/service/...` packages.
  2. Refactor the anonymous functions passed to `RunInTransaction`:
     - If an underlying store error (e.g., `store.ErrCardNotFound`) or other error occurs, wrap it using a relevant service-specific error constructor (e.g., `NewSubmitAnswerError(...)`, `NewCardServiceError(...) %w`) before returning.
     - Explicitly return defined service sentinel errors (like `service.ErrNotOwned`) directly when appropriate for the domain logic.
  3. Update calling layers (e.g., API handlers) to consistently use `errors.Is` or `errors.As` to check for the specific service-level errors (wrapped or sentinel).
  4. Document this standard approach in service package documentation or a contribution guide.
- **Done‑When:** Error returns from service transaction functions are consistent. Callers reliably check errors using `errors.Is`/`errors.As`. Tests verify correct error propagation and handling. CI green.

### cr‑03 Improve SQL Redaction for Sensitive Data Leakage
- **Problem:** The simple SQL redaction regex in `internal/redact/redact.go` is likely insufficient, risking leakage of sensitive data within complex queries or over-redaction.
- **Impact:** Potential security vulnerability (sensitive data in logs); hinders debugging; violates `Security Considerations`, `Logging Strategy`.
- **Chosen Fix:** Enhance the redaction logic to be more targeted towards sensitive data patterns (quoted strings, numeric values after comparisons) rather than just keywords. If precise regex becomes too complex/fragile, fall back to a simple `"[SQL_QUERY_REDACTED]"` placeholder.
- **Steps:**
  1. Analyze common SQL query patterns generated by the application where sensitive data might appear (e.g., literals in `WHERE`, `INSERT VALUES`, `UPDATE SET`).
  2. Refine the regex patterns in `internal/redact/redact.go` to specifically match and replace common sensitive value formats (e.g., `'...'`, numeric literals after `=`/`>`/`<`, etc.) with a `[REDACTED]` placeholder. Avoid overly broad matching of SQL keywords.
  3. **Contingency:** If achieving reliable redaction via regex proves too difficult or brittle, replace the SQL matching logic entirely to substitute any string identified as likely SQL (e.g., starting with SELECT/INSERT/UPDATE/DELETE) with the fixed placeholder `"[SQL_QUERY_REDACTED]"`.
  4. Add extensive table-driven tests in `internal/redact/redact_test.go` covering various SQL statements with embedded sensitive data (strings, numbers) in different clauses to validate the effectiveness and precision of the chosen redaction method.
- **Done‑When:** SQL redaction is demonstrably more robust. Test cases cover diverse sensitive data patterns. Manual inspection of logs (dev/staging) confirms no leakage and acceptable debugging utility. CI green.

## Standards Alignment
- **Simplicity First:** Achieved by removing complex mocks (cr-01), consolidating helpers (cr-02), adopting clear error handling (cr-05), and potentially simplifying redaction (cr-03 contingency).
- **Modularity is Mandatory:** Enforced by centralizing test helpers (cr-02) into `internal/testutils`.
- **Design for Testability:** Upheld by eliminating internal mocks (cr-01) and promoting integration testing.
- **Coding Standards:** Addressed via robust nil checks (cr-04) and consistent error handling (cr-05).
- **Security Considerations:** Directly improved by enhancing SQL redaction (cr-03).
- **Error Handling:** Standardized across services for predictability (cr-05).

## Validation Checklist
- [ ] All automated tests (unit + integration) pass (`go test -race ./...`, `go test -race -tags=integration ./...`).
- [ ] Static analysis (`golangci-lint run ./...`) reports no new issues.
- [ ] Code coverage meets or exceeds established project thresholds.
- [ ] Manual review of logs in a non-production environment confirms effective SQL redaction (cr-03).
- [ ] Manual review confirms deletion of mock tests (cr-01) and consolidation of helpers (cr-02).
- [ ] No new `nolint` directives introduced related to these fixes.
```
