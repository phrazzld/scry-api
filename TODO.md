# Todo

## Service/Task Decoupling

- [x] **T028 · feature · p1: define TaskRequestEvent and EventHandler in internal/events**
    - **Context:** cr-01 step 1
    - **Action:**
        1. Create `internal/events` package
        2. Define `TaskRequestEvent` struct to contain task creation details
        3. Define `EventHandler` interface with method to handle events
    - **Done-when:**
        1. `internal/events` package exists with properly defined types
        2. Unit tests verify event structure and handler interface
    - **Depends-on:** none

- [x] **T029 · refactor · p1: replace TaskFactory dependency with EventEmitter in MemoService**
    - **Context:** cr-01 step 2
    - **Action:**
        1. Define `EventEmitter` interface in `internal/events`
        2. Update `MemoService` to use `EventEmitter` instead of direct `TaskFactory` dependency
        3. Remove `SetTaskFactory` method from `MemoService`
    - **Done-when:**
        1. `MemoService` no longer has direct `TaskFactory` dependency
        2. Service uses `EventEmitter` to publish task creation events
    - **Depends-on:** [T028]

- [x] **T030 · feature · p1: create TaskFactoryEventHandler in task package**
    - **Context:** cr-01 step 3
    - **Action:**
        1. Implement `TaskFactoryEventHandler` in `task` package that subscribes to events
        2. Connect handler to the existing `TaskFactory` implementation
        3. Add tests for the event handler implementation
    - **Done-when:**
        1. `TaskFactoryEventHandler` correctly creates tasks in response to events
        2. Tests verify handler behavior with different event types
    - **Depends-on:** [T028]

- [x] **T031 · chore · p1: wire event system in application initialization**
    - **Context:** cr-01 step 4
    - **Action:**
        1. Update `main.go` to create and configure the event emitter
        2. Register `TaskFactoryEventHandler` with event system
        3. Remove any direct wiring between `MemoService` and `TaskFactory`
    - **Done-when:**
        1. Application initializes without circular dependency
        2. Services and tasks operate through the event system
    - **Depends-on:** [T029, T030]

- [x] **T032 · refactor · p1: unexport MemoServiceImpl**
    - **Context:** cr-01 step 5
    - **Action:**
        1. Rename `MemoServiceImpl` to unexported `memoServiceImpl`
        2. Update constructor to return interface type only
    - **Done-when:**
        1. Implementation is unexported and only accessible through interface
        2. No compilation errors or test failures
    - **Depends-on:** [T029]

## Transaction Boundary Management

- [x] **T033 · refactor · p1: remove transaction logic from CardStore.CreateMultiple**
    - **Context:** cr-02 steps 1-2
    - **Action:**
        1. Remove `BeginTx`, `Commit`, and `Rollback` code
        2. Remove transaction detection logic
        3. Modify method to assume it's operating within a transaction
    - **Done-when:**
        1. Method contains no transaction management code
        2. Tests confirm behavior within transaction context
    - **Depends-on:** none

- [x] **T034 · chore · p3: document transaction assumption in CardStore.CreateMultiple**
    - **Context:** cr-02 step 3
    - **Action:**
        1. Add clear documentation that method must run within a transaction
        2. Document expected behavior if called outside a transaction
    - **Done-when:**
        1. Method has comprehensive documentation about transaction requirements
    - **Depends-on:** [T033]

- [x] **T035 · refactor · p1: update CardStore.CreateMultiple callers**
    - **Context:** cr-02 step 4
    - **Action:**
        1. Find all callers of `CardStore.CreateMultiple`
        2. Ensure callers use `store.RunInTransaction` with `CardStore.WithTx`
    - **Done-when:**
        1. All callers properly manage the transaction context
    - **Depends-on:** [T033]

## Cross-Platform Pre-commit Hooks

- [x] **T036 · chore · p1: restore standard pre-commit hooks**
    - **Context:** cr-03 steps 1-3
    - **Action:**
        1. Remove custom `fix-trailing-whitespace` and `fix-end-of-file` hooks
        2. Add back standard `trailing-whitespace` and `end-of-file-fixer` hooks
        3. Configure hooks to ensure cross-platform compatibility
    - **Done-when:**
        1. Hooks run successfully on both macOS and Linux
        2. Pre-commit configuration passes validation
    - **Depends-on:** none

## MemoServiceAdapter Validation

- [x] **T037 · refactor · p2: improve MemoServiceAdapter constructor validation**
    - **Context:** cr-04 steps 1-3
    - **Action:**
        1. Add type assertions in `NewMemoServiceAdapter` to verify interface compliance
        2. Return clear, descriptive errors on validation failure
        3. Document required repository methods in comments
    - **Done-when:**
        1. Constructor fails fast with clear errors for incompatible repositories
        2. Documentation clearly lists all required methods
    - **Depends-on:** none

## Fix GetNextReviewCard Panic

- [x] **T038 · bugfix · p1: replace panic with error in GetNextReviewCard**
    - **Context:** cr-05 steps 1-2
    - **Action:**
        1. Replace `panic` with `return nil, store.ErrNotImplemented`
        2. Update callers to handle the error case properly
    - **Done-when:**
        1. Method returns appropriate error without panicking
        2. Tests verify error handling
    - **Depends-on:** none

## UserCardStats Orchestration

- [ ] **T039 · refactor · p1: remove UserCardStats creation from CardStore**
    - **Context:** cr-06 step 1
    - **Action:**
        1. Remove code that inserts `UserCardStats` from `CardStore.CreateMultiple`
        2. Ensure tests are updated to reflect the change
    - **Done-when:**
        1. `CardStore.CreateMultiple` only manages card entities
    - **Depends-on:** none

- [ ] **T040 · feature · p1: create CardService and orchestration method**
    - **Context:** cr-06 steps 2-3
    - **Action:**
        1. Create new `CardService` interface and implementation in `internal/service/card_service.go`
        2. Implement `CreateCards` method that handles both card and stats creation
        3. Use `store.RunInTransaction` with repositories' `WithTx` methods to ensure atomicity
    - **Done-when:**
        1. `CardService.CreateCards` orchestrates both operations in a single transaction
        2. Tests verify atomic behavior
    - **Depends-on:** [T039]

- [ ] **T041 · refactor · p2: update callers to use new orchestration method**
    - **Context:** cr-06 step 4
    - **Action:**
        1. Find all callers that previously relied on CardStore.CreateMultiple for stats
        2. Update them to use the new service orchestration method
    - **Done-when:**
        1. All callers use the service method for orchestration
    - **Depends-on:** [T040]

## Test Helper Consolidation

- [ ] **T042 · refactor · p2: centralize duplicate test helpers**
    - **Context:** cr-07 steps 1-3
    - **Action:**
        1. Identify all duplicated helper functions across test files
        2. Move them to appropriate files in `internal/testutils`
        3. Update all tests to use the centralized helpers
    - **Done-when:**
        1. No duplicate test helpers exist in individual test files
        2. Tests pass using centralized utilities
    - **Depends-on:** none

## AssertNoErrorLeakage Relocation

- [ ] **T043 · refactor · p3: move AssertNoErrorLeakage to postgres package**
    - **Context:** cr-08 steps 1-2
    - **Action:**
        1. Move `AssertNoErrorLeakage` function to `internal/platform/postgres/errors_test.go`
        2. Update all imports and references
    - **Done-when:**
        1. Helper is co-located with the code it tests
        2. Tests pass with updated imports
    - **Depends-on:** none

## Standardize bcrypt Cost

- [ ] **T044 · refactor · p2: standardize bcrypt cost in test helpers**
    - **Context:** cr-09 steps 1-3
    - **Action:**
        1. Add `bcryptCost` parameter to `MustInsertUser` and `CreateTestStores`
        2. Pass the configured value from application settings in all test cases
    - **Done-when:**
        1. Test helpers use consistent bcrypt cost values
        2. Tests pass with standardized values
    - **Depends-on:** none

## Remove Unnecessary sql.DB Mock

- [ ] **T045 · refactor · p3: eliminate sql.DB mock**
    - **Context:** cr-10 steps 1-2
    - **Action:**
        1. Remove `internal/mocks/db.go` file
        2. Update tests to use store.DBTX interface instead
    - **Done-when:**
        1. No direct mocking of sql.DB is used in tests
        2. All tests pass with interface-based approach
    - **Depends-on:** none

## Document Cascade Delete Dependencies

- [ ] **T046 · chore · p3: document cascade delete behavior**
    - **Context:** cr-11 steps 1-2
    - **Action:**
        1. Update interface documentation in `store/card.go`
        2. Add clear comments in the implementation about dependency on cascade deletes
    - **Done-when:**
        1. Cascade delete behavior is clearly documented in both interface and implementation
    - **Depends-on:** none

## Fix Trivial and Misleading Tests

- [ ] **T047 · test · p3: clean up test code quality issues**
    - **Context:** cr-12 steps 1-2
    - **Action:**
        1. Remove `TestDBTXInterface` test
        2. Fix misleading comments in `TestableGeminiGenerator`
    - **Done-when:**
        1. Tests are meaningful and comments match implementation
    - **Depends-on:** none

## Documentation Improvements

- [ ] **T048 · chore · p3: improve documentation quality**
    - **Context:** cr-13 steps 1-2
    - **Action:**
        1. Update generic TODO comments to be specific and actionable
        2. Fix all other documentation inconsistencies
    - **Done-when:**
        1. All TODOs have clear next steps
        2. Documentation is accurate and consistent
    - **Depends-on:** none

### Clarifications & Assumptions

- [x] **Issue:** Need to determine the best service to handle UserCardStats orchestration
    - **Context:** cr-06 step 2
    - **Resolution:** Create a new CardService in internal/service/card_service.go that orchestrates both Card and UserCardStats creation within a transaction. This follows the same pattern as MemoService with RunInTransaction + WithTx, keeping persistence layer focused on single responsibilities while the service layer handles orchestration.
