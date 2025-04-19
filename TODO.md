# Todo

## Store Error Contracts
- [x] **T001 · refactor · p1: define base and entity-specific errors**
    - **Context:** cr‑01 Centralize and standardize entity-specific error contracts
    - **Action:**
        1. Define base errors (`ErrNotFound`, `ErrDuplicate`, etc.) in `internal/store/errors.go`.
        2. Define entity-specific errors (`ErrUserNotFound`, `ErrMemoNotFound`, etc.) wrapping base errors.
    - **Done‑when:**
        1. Centralized error definitions exist in `internal/store/errors.go`.
    - **Depends‑on:** none

- [x] **T002 · refactor · p1: update UserStore to use standardized errors**
    - **Context:** cr‑01 Centralize and standardize entity-specific error contracts
    - **Action:**
        1. Update `internal/platform/postgres/user_store.go` to return standardized errors from `internal/store/errors.go`.
        2. Update corresponding tests to assert against standardized errors using `errors.Is()`.
    - **Done‑when:**
        1. `PostgresUserStore` uses standardized errors.
        2. Tests pass with new error assertions.
    - **Depends‑on:** [T001]

- [x] **T003 · refactor · p1: update MemoStore to use standardized errors**
    - **Context:** cr‑01 Centralize and standardize entity-specific error contracts
    - **Action:**
        1. Update `internal/platform/postgres/memo_store.go` to return standardized errors.
        2. Update corresponding tests to assert against standardized errors using `errors.Is()`.
    - **Done‑when:**
        1. `PostgresMemoStore` uses standardized errors.
        2. Tests pass with new error assertions.
    - **Depends‑on:** [T001]

- [x] **T004 · refactor · p1: update CardStore to use standardized errors**
    - **Context:** cr‑01 Centralize and standardize entity-specific error contracts
    - **Action:**
        1. Update `internal/platform/postgres/card_store.go` to return standardized errors.
        2. Update corresponding tests to assert against standardized errors using `errors.Is()`.
    - **Done‑when:**
        1. `PostgresCardStore` uses standardized errors.
        2. Tests pass with new error assertions.
    - **Depends‑on:** [T001]

- [x] **T005 · refactor · p1: update UserCardStatsStore to use standardized errors**
    - **Context:** cr‑01 Centralize and standardize entity-specific error contracts
    - **Action:**
        1. Update `internal/platform/postgres/stats_store.go` to return standardized errors.
        2. Update corresponding tests to assert against standardized errors using `errors.Is()`.
    - **Done‑when:**
        1. `PostgresUserCardStatsStore` uses standardized errors.
        2. Tests pass with new error assertions.
    - **Depends‑on:** [T001]

## Store Error Mapping
- [x] **T006 · refactor · p1: refactor UserStore to use error mapping helpers**
    - **Context:** cr‑03 Consistent error mapping with helpers
    - **Action:**
        1. Replace direct database error checks in `internal/platform/postgres/user_store.go` with calls to helpers.
        2. Ensure proper error wrapping (`fmt.Errorf("%w", err)`).
    - **Done‑when:**
        1. `PostgresUserStore` uses centralized error mapping helpers.
        2. Tests pass verifying correct error mapping.
    - **Depends‑on:** [T001]

- [x] **T007 · refactor · p1: refactor MemoStore to use error mapping helpers**
    - **Context:** cr‑03 Consistent error mapping with helpers
    - **Action:**
        1. Replace direct database error checks in `internal/platform/postgres/memo_store.go` with calls to helpers.
        2. Ensure proper error wrapping.
    - **Done‑when:**
        1. `PostgresMemoStore` uses centralized error mapping helpers.
        2. Tests pass verifying correct error mapping.
    - **Depends‑on:** [T001]

- [x] **T008 · refactor · p1: refactor CardStore to use error mapping helpers**
    - **Context:** cr‑03 Consistent error mapping with helpers
    - **Action:**
        1. Replace direct database error checks in `internal/platform/postgres/card_store.go` with calls to helpers.
        2. Ensure proper error wrapping.
    - **Done‑when:**
        1. `PostgresCardStore` uses centralized error mapping helpers.
        2. Tests pass verifying correct error mapping.
    - **Depends‑on:** [T001]

- [x] **T009 · refactor · p1: refactor UserCardStatsStore to use error mapping helpers**
    - **Context:** cr‑03 Consistent error mapping with helpers
    - **Action:**
        1. Replace direct database error checks in `internal/platform/postgres/stats_store.go` with calls to helpers.
        2. Ensure proper error wrapping.
    - **Done‑when:**
        1. `PostgresUserCardStatsStore` uses centralized error mapping helpers.
        2. Tests pass verifying correct error mapping.
    - **Depends‑on:** [T001]

## Store Security
- [x] **T010 · bugfix · p0: audit and update stores to prevent internal error detail leakage**
    - **Context:** cr‑04 Prevent internal error detail leakage
    - **Action:**
        1. Audit all error return points in store implementations.
        2. Ensure internal error details are logged using structured logging.
        3. Ensure only standardized errors are returned to callers.
    - **Done‑when:**
        1. All store methods log detailed internal errors but return only standardized, opaque errors.
    - **Depends‑on:** [T002, T003, T004, T005, T006, T007, T008, T009]

- [x] **T011 · test · p0: add tests verifying no internal details leak in store errors**
    - **Context:** cr‑04 Prevent internal error detail leakage
    - **Action:**
        1. Add specific tests for each store implementation.
        2. Trigger various database errors (e.g., constraint violations).
        3. Assert that the returned error message does not contain internal database details.
    - **Done‑when:**
        1. Tests pass confirming no internal error details are exposed by store methods.
    - **Depends‑on:** [T010]

## Dependency Injection & Modularity
- [x] **T012 · refactor · p1: define task.TaskStore interface**
    - **Context:** cr‑05 Use TaskStore interface in dependency injection
    - **Action:**
        1. Define a `TaskStore` interface in `internal/task/task.go` with necessary methods.
        2. Ensure `PostgresTaskStore` implicitly satisfies this interface.
    - **Done‑when:**
        1. `internal/task/task.go` contains the `TaskStore` interface definition.
    - **Depends‑on:** none

- [x] **T013 · refactor · p1: update dependency injection to use task.TaskStore interface**
    - **Context:** cr‑05 Use TaskStore interface in dependency injection
    - **Action:**
        1. Update `appDependencies` in `cmd/server/main.go` to use interface type.
        2. Adjust initialization code to assign concrete implementation to interface field.
    - **Done‑when:**
        1. `cmd/server/main.go` uses the interface for dependency injection.
        2. Application compiles and tests pass.
    - **Depends‑on:** [T012]

- [x] **T014 · refactor · p1: decouple MemoStore.Update from task package logic**
    - **Context:** cr‑06 Decouple MemoStore.Update from task package
    - **Action:**
        1. Identify task-specific logic in `MemoStore.Update`.
        2. Move this logic to the appropriate service layer component.
        3. Refactor `MemoStore.Update` to focus solely on data persistence.
    - **Done‑when:**
        1. `MemoStore.Update` has no task package dependency.
        2. Business logic resides in the service layer.
    - **Depends‑on:** none

- [x] **T015 · refactor · p1: update service layer to handle memo status transitions**
    - **Context:** cr‑06 Decouple MemoStore.Update from task package
    - **Action:**
        1. Move memo status transition logic to service layer.
        2. Update all callers as needed.
    - **Done‑when:**
        1. All business logic for memo status lives outside the store.
        2. Tests pass with the refactored code.
    - **Depends‑on:** [T014]

## Testing Utilities
- [ ] **T016 · refactor · p1: remove password hashing logic from testutils.MustInsertUser**
    - **Context:** cr‑07 Remove domain logic from test utilities
    - **Action:**
        1. Modify `MustInsertUser` to accept a pre-hashed password or use UserStore.Create.
        2. Update all test callers to provide hashed passwords or adapt to the new helper.
    - **Done‑when:**
        1. `MustInsertUser` no longer contains password hashing logic.
        2. All tests using `MustInsertUser` pass.
    - **Depends‑on:** none

- [ ] **T017 · refactor · p2: centralize duplicate test helpers into internal/testutils**
    - **Context:** cr‑11 Clean up duplicate test helpers
    - **Action:**
        1. Identify duplicated test helper functions across test files.
        2. Move these helpers into `internal/testutils`.
        3. Update all call sites to use the centralized helpers.
    - **Done‑when:**
        1. No duplicate test code remains; all tests use centralized utilities.
    - **Depends‑on:** none

## User Store Simplification
- [ ] **T018 · refactor · p1: simplify UserStore.Update logic to remove internal fetch**
    - **Context:** cr‑08 Simplify user update logic
    - **Action:**
        1. Remove DB fetch in PostgresUserStore.Update.
        2. Expect complete user object (including HashedPassword) from service layer.
    - **Done‑when:**
        1. User update logic is straightforward and delegates completeness to caller.
    - **Depends‑on:** none

- [ ] **T019 · refactor · p1: update service layer to provide complete user object for updates**
    - **Context:** cr‑08 Simplify user update logic
    - **Action:**
        1. Change service code to always pass a fully-populated User object to Update.
    - **Done‑when:**
        1. Service and store layers are cleanly separated.
        2. All tests pass with simplified update logic.
    - **Depends‑on:** [T018]

## Transaction Management
- [ ] **T020 · feature · p1: implement WithTx method on all store interfaces**
    - **Context:** cr‑02 Implement WithTx and clarify transaction boundaries
    - **Action:**
        1. Add `WithTx(*sql.Tx)` methods to all store interfaces in `internal/store`.
    - **Done‑when:**
        1. All store interfaces define WithTx.
    - **Depends‑on:** none

- [ ] **T021 · feature · p1: implement WithTx method in all store implementations**
    - **Context:** cr‑02 Implement WithTx and clarify transaction boundaries
    - **Action:**
        1. Implement WithTx method in each Postgres store implementation.
    - **Done‑when:**
        1. All stores construct new instances with the provided transaction.
    - **Depends‑on:** [T020]

- [ ] **T022 · refactor · p1: update services to manage transaction boundaries explicitly**
    - **Context:** cr‑02 Implement WithTx and clarify transaction boundaries
    - **Action:**
        1. Refactor service layer to create store instances with transactions as needed.
    - **Done‑when:**
        1. Transaction boundaries are explicit and managed at the service layer.
    - **Depends‑on:** [T021]

- [ ] **T023 · test · p1: add integration tests for transaction atomicity**
    - **Context:** cr‑02 Implement WithTx and clarify transaction boundaries
    - **Action:**
        1. Write tests to verify atomicity and rollback/commit behavior with WithTx.
    - **Done‑when:**
        1. Transactions are proven atomic in test scenarios.
    - **Depends‑on:** [T021]

## Documentation
- [ ] **T024 · chore · p2: document transaction pattern and ownership in README**
    - **Context:** cr‑02 Implement WithTx and clarify transaction boundaries
    - **Action:**
        1. Update store/README.md to document transaction pattern.
        2. Provide clear examples of transaction usage.
    - **Done‑when:**
        1. Documentation matches new transaction pattern.
    - **Depends‑on:** [T022]

- [ ] **T025 · chore · p2: update documentation to match current interfaces and error patterns**
    - **Context:** cr‑09 Update documentation across store implementations
    - **Action:**
        1. Update store/README.md and doc.go files to reflect latest interfaces and error contracts.
    - **Done‑when:**
        1. Documentation accurately reflects current interfaces, patterns, and best practices.
    - **Depends‑on:** [T001, T024]

## Testing Improvements
- [ ] **T026 · test · p2: add missing unit tests for error utilities**
    - **Context:** cr‑10 Add missing unit tests for error utilities
    - **Action:**
        1. Create `internal/platform/postgres/errors_test.go`.
        2. Write table-driven tests for all error mapping and helper functions.
    - **Done‑when:**
        1. Coverage for error mapping utilities is comprehensive.
    - **Depends‑on:** none

## Cleanup
- [ ] **T027 · chore · p2: remove MockCardRepository and associated dead code**
    - **Context:** cr‑12 Remove dead code (MockCardRepository)
    - **Action:**
        1. Remove MockCardRepository from main.go and associated test files.
        2. Update tests to use real CardStore with transaction isolation.
    - **Done‑when:**
        1. No dead code remains; all tests use real implementations.
    - **Depends‑on:** [T021]

- [ ] **T028 · test · p2: add compile-time interface checks to all store implementations**
    - **Context:** cr‑13 Add compile-time interface checks
    - **Action:**
        1. Add `var _ store.Interface = (*Implementation)(nil)` assertions.
    - **Done‑when:**
        1. All store implementations are compile-time checked.
    - **Depends‑on:** none

- [ ] **T029 · chore · p3: mark stub GetNextReviewCard implementation with TODO and panic**
    - **Context:** cr‑14 Mark stub GetNextReviewCard implementation
    - **Action:**
        1. Add TODO and panic with clear message in stub.
        2. Create issue/task for implementation.
    - **Done‑when:**
        1. Method is clearly marked as unimplemented.
    - **Depends‑on:** none

- [ ] **T030 · chore · p3: clean up minor issues (logging, TODOs, dead functions)**
    - **Context:** cr‑15 Fix minor issues (logging, TODOs, dead functions)
    - **Action:**
        1. Reduce logging level for routine success to Debug.
        2. Move/fix misplaced TODO comments.
        3. Remove unused functions.
        4. Add interface versioning comments.
    - **Done‑when:**
        1. Codebase passes style checks and is clean.
    - **Depends‑on:** none

### Clarifications & Assumptions
- [ ] **Issue:** Should all store interfaces expose WithTx, or only those used transactionally?
    - **Context:** cr‑02 Implement WithTx and clarify transaction boundaries
    - **Blocking?:** yes

- [ ] **Issue:** Are test helpers allowed to call direct SQL for setup, or must everything use UserStore.Create?
    - **Context:** cr‑07 Remove domain logic from test utilities
    - **Blocking?:** no
