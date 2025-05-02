```markdown
# Todo

## Test Helper Consolidation (cr-02)
- [x] **T001 · Refactor · P1: consolidate test helpers into internal/testutils**
    - **Context:** cr-02 Consolidate Test Helper Sprawl (Steps 1-5)
    - **Action:**
        1. Identify reusable helpers in `cmd/server/api_test_helpers_test.go`, `cmd/server/auth_test_helpers.go`, `internal/testutils/`, `internal/testdb/`. Create `internal/testutils/db` and `internal/testutils/api` sub-packages.
        2. Move/refactor identified helpers into the appropriate `internal/testutils` sub-package, ensuring consistency and clarity. Delete `cmd/server/api_test_helpers_test.go` and `cmd/server/auth_test_helpers.go`.
        3. Refactor all tests (unit and integration) across the codebase to import and use helpers exclusively from `internal/testutils/...`.
    - **Done‑when:**
        1. Helper files in `cmd/server` are deleted.
        2. All tests compile and pass using only helpers from `internal/testutils`.
        3. CI green (`go test -race ./...`, `go test -race -tags=integration ./...`, `golangci-lint run ./...`).
    - **Depends‑on:** none

## API Testing (cr-01)
- [x] **T002 · Chore · P1: delete mock-based API test files**
    - **Context:** cr-01 Remove Contradictory Mock-Based API Tests (Steps 2-3)
    - **Action:**
        1. Delete `cmd/server/auth_api_test.go`.
        2. Delete `cmd/server/card_api_test.go`.
    - **Done‑when:**
        1. Files are deleted from the repository.
        2. Code compiles and CI is green.
    - **Depends‑on:** [T001]

- [ ] **T003 · Test · P1: audit integration test coverage gaps post-mock deletion**
    - **Context:** cr-01 Remove Contradictory Mock-Based API Tests (Step 4)
    - **Action:**
        1. Run coverage tools on existing integration tests (`cmd/server/auth_integration_test.go`, `cmd/server/card_api_integration_test.go`, etc.).
        2. Identify critical scenarios previously covered only by the deleted mock tests (T002).
        3. Document the specific coverage gaps found.
    - **Done‑when:**
        1. A list of specific, critical coverage gaps (or confirmation of none) is documented.
    - **Depends‑on:** [T002]

- [ ] **T004 · Test · P1: add integration tests to cover identified gaps**
    - **Context:** cr-01 Remove Contradictory Mock-Based API Tests (Step 5)
    - **Action:**
        1. Implement necessary test cases within the integration tests (`*_integration_test.go`) to fill gaps identified in T003.
        2. Use only the consolidated helpers from `internal/testutils`.
    - **Done‑when:**
        1. Documented coverage gaps are filled with new integration tests.
        2. Integration tests provide verified, equivalent (or better) coverage for the relevant API endpoints.
        3. CI green (`go test -race -tags=integration ./...`, code coverage meets threshold).
    - **Depends‑on:** [T003]

## Data Storage (cr-04)
- [x] **T005 · Bugfix · P0: refactor PostgresCardStore to prevent nil pointer panic**
    - **Context:** cr-04 Fix Potential Nil Pointer Panic in CardRepositoryAdapter (Step 1)
    - **Action:**
        1. Modify `internal/platform/postgres/card_store.go`: Add `sqlDB *sql.DB` field to `PostgresCardStore`, update `NewPostgresCardStore` to initialize it.
        2. Update `WithTx` method to copy the `sqlDB` field value to the new transactional store instance.
        3. Ensure the `DB()` method returns the stored `sqlDB` field.
    - **Done‑when:**
        1. `PostgresCardStore.DB()` never returns nil when the store was initialized with a valid DB, even after `WithTx` is called.
        2. Code compiles.
    - **Depends‑on:** none

- [x] **T006 · Test · P0: add unit tests verifying PostgresCardStore.DB() non-nil after WithTx**
    - **Context:** cr-04 Fix Potential Nil Pointer Panic in CardRepositoryAdapter (Step 2)
    - **Action:**
        1. Add/verify unit tests in `internal/platform/postgres/card_store_test.go` specifically testing that `DB()` returns a non-nil `*sql.DB` after calling `WithTx`.
    - **Done‑when:**
        1. Tests explicitly verify non-nil return from `DB()` after `WithTx`.
        2. All tests in `internal/platform/postgres` pass.
        3. CI green.
    - **Depends‑on:** [T005]

## Service Layer (cr-05)
- [x] **T007 · Audit · P1: audit service transaction error handling patterns**
    - **Context:** cr-05 Standardize Service Transaction Error Handling (Step 1)
    - **Action:**
        1. Find all usages of `store.RunInTransaction` within the `internal/service/...` packages.
        2. Document the current error return patterns (direct store errors, wrapped errors, sentinel errors) for each usage.
    - **Done‑when:**
        1. A documented inventory of `RunInTransaction` usages and their error handling patterns exists.
    - **Depends‑on:** none

- [x] **T008 · Refactor · P1: standardize error wrapping in service transaction callbacks**
    - **Context:** cr-05 Standardize Service Transaction Error Handling (Step 2)
    - **Action:**
        1. Refactor the anonymous functions passed to `RunInTransaction` based on the audit (T007).
        2. Wrap underlying store/other errors using relevant service-specific error constructors (`%w`) before returning.
        3. Return defined service sentinel errors (e.g., `service.ErrNotOwned`) directly when appropriate.
    - **Done‑when:**
        1. All audited transaction callbacks consistently wrap non-sentinel errors or return defined sentinel errors.
        2. Code compiles and relevant unit tests pass.
    - **Depends‑on:** [T007]

- [x] **T009 · Refactor · P1: update API handlers/callers to use errors.Is/As for service errors**
    - **Context:** cr-05 Standardize Service Transaction Error Handling (Step 3)
    - **Action:**
        1. Update calling layers (e.g., API handlers in `cmd/server`) to consistently use `errors.Is` or `errors.As` to check for the specific service-level errors (wrapped or sentinel) returned by methods using `RunInTransaction`.
    - **Done‑when:**
        1. Callers reliably check errors using `errors.Is`/`errors.As`.
        2. Tests verifying error handling in callers pass.
        3. CI green.
    - **Depends‑on:** [T008]

- [ ] **T010 · Chore · P2: document standardized service error handling approach**
    - **Context:** cr-05 Standardize Service Transaction Error Handling (Step 4)
    - **Action:**
        1. Add documentation to the relevant service package(s) or a central contribution guide detailing the standard for error handling in `RunInTransaction` callbacks and how callers should check errors.
    - **Done‑when:**
        1. Documentation is written and committed.
    - **Depends‑on:** [T008]

## Logging & Security (cr-03)
- [ ] **T011 · Audit · P1: analyze SQL query patterns for sensitive data**
    - **Context:** cr-03 Improve SQL Redaction for Sensitive Data Leakage (Step 1)
    - **Action:**
        1. Analyze common SQL query patterns generated by the application where sensitive data might appear (e.g., literals in `WHERE`, `INSERT VALUES`, `UPDATE SET`).
        2. Document the specific patterns found (e.g., `WHERE user_id = '...'`, `VALUES(..., 'sensitive_token', ...)`).
    - **Done‑when:**
        1. A documented list of sensitive data patterns in SQL queries exists.
    - **Depends‑on:** none

- [ ] **T012 · Feature · P1: implement enhanced SQL redaction logic (or contingency)**
    - **Context:** cr-03 Improve SQL Redaction for Sensitive Data Leakage (Steps 2-3)
    - **Action:**
        1. Refine the regex patterns in `internal/redact/redact.go` based on T011 to specifically match and replace common sensitive value formats (e.g., `'...'`, numeric literals after `=`/`>`/`<`) with `[REDACTED]`.
        2. **Contingency:** If reliable regex proves too complex/brittle during implementation, replace the logic to substitute any string identified as likely SQL (e.g., starting with SELECT/INSERT/UPDATE/DELETE) with `"[SQL_QUERY_REDACTED]"`. Document the decision.
    - **Done‑when:**
        1. SQL redaction logic in `internal/redact/redact.go` is updated using the chosen approach (refined regex or placeholder).
        2. Code compiles.
    - **Depends‑on:** [T011]

- [ ] **T013 · Test · P1: add comprehensive table-driven tests for SQL redaction**
    - **Context:** cr-03 Improve SQL Redaction for Sensitive Data Leakage (Step 4)
    - **Action:**
        1. Add extensive table-driven tests in `internal/redact/redact_test.go`.
        2. Cover various SQL statements with embedded sensitive data (strings, numbers) in different clauses, validating the effectiveness and precision of the chosen redaction method (T012).
    - **Done‑when:**
        1. Test cases cover diverse sensitive data patterns and SQL structures.
        2. All redaction tests pass.
        3. CI green.
    - **Depends‑on:** [T012]

- [ ] **T014 · Chore · P1: manually verify SQL redaction effectiveness in logs**
    - **Context:** cr-03 Improve SQL Redaction for Sensitive Data Leakage (Done-When)
    - **Action:**
        1. Deploy code with updated redaction (T012, T013) to a non-production environment (dev/staging).
        2. Trigger application logic generating SQL queries with potentially sensitive data.
        3. Inspect application logs.
    - **Done‑when:**
        1. Manual inspection confirms sensitive data is effectively redacted according to the chosen strategy.
        2. Debugging utility of logs is confirmed as acceptable.
    - **Verification:**
        1. Check logs for specific examples of sensitive data (e.g., user IDs, tokens) and confirm they are redacted.
        2. Check that non-sensitive parts of queries remain readable (if using regex).
    - **Depends‑on:** [T013]

---

### Clarifications & Assumptions
- [ ] **Issue:** Define "critical scenarios" for integration test coverage audit (T003).
    - **Context:** cr-01 Remove Contradictory Mock-Based API Tests, Step 4
    - **Blocking?:** yes - Requires clarification before T004 can be completed effectively. Assume core success/failure paths for each endpoint initially.
- [ ] **Issue:** Define criteria for triggering the contingency plan (blanket redaction) in T012.
    - **Context:** cr-03 Improve SQL Redaction for Sensitive Data Leakage, Step 3
    - **Blocking?:** no - Assume implementer makes the call based on perceived regex complexity/brittleness vs. security risk.
- [ ] **Issue:** What are the established project code coverage thresholds?
    - **Context:** Validation Checklist / T004 Done-when
    - **Blocking?:** yes - Needed to confirm T004 completion.
```
