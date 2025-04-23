# TODO List for Card Review API Improvements (Sprint 25)

## Production Stability

### T001: Create Gemini Generator Factory with Build Tags
- [x] Create factory function to return appropriate LLM generator implementation based on build tags
- [x] Add new generator factory to `internal/platform/gemini/gemini_utils.go` with exported `NewGenerator` function
- [x] Use build tags to conditionally compile real vs mock implementations
- Effort: Small
- Depends On: None

### T002: Fix Main.go Generator Initialization
- [x] Update `cmd/server/main.go` to use factory function instead of direct mock import
- [x] Add error handling for generator creation failures
- [x] Remove direct use of `mocks.NewMockGeneratorWithDefaultCards`
- Effort: Small
- Depends On: T001

### T003: Validate LLM Configuration in Production
- [x] Add enhanced validation in gemini factory to ensure API keys exist in production
- [x] Add specific error messages for missing/invalid configuration
- [x] Ensure tests still work with minimal configuration
- Effort: Extra Small
- Depends On: T001

### T004: Refactor Card Review Service Constructor
- [x] Update `NewCardReviewService` to return errors instead of panicking on nil dependencies
- [x] Change function signature to return `(*cardReviewService, error)`
- [x] Replace panic calls with formatted error returns
- Effort: Small
- Depends On: None

### T005: Update Service Init Error Handling
- [x] Modify all call sites of service constructors to check for errors
- [x] Add appropriate error handling in `cmd/server/main.go`
- [x] Update logs with context about initialization failures
- Effort: Small
- Depends On: T004

## Repository Abstraction

### T006: Analyze Repository Interfaces
- [x] Review current repository interfaces, adapters, and store implementations
- [x] Document redundant abstractions and inconsistencies
- [x] Identify target pattern for standardization
- Effort: Small
- Depends On: None

### T007: Create Repository Pattern Documentation
- [x] Document standardized repository approach in `internal/domain/README.md`
- [x] Define interface naming conventions and responsibilities
- [x] Create examples of proper repository usage
- Effort: Small
- Depends On: T006

### T008: Refactor Card Review Repository Interface
- [x] Update card review service to depend directly on store interfaces
- [x] Remove redundant adapter layers between store and service
- [x] Modify service implementation to call store methods directly
- Effort: Medium
- Depends On: T007

### T009: Extend UserCardStatsStore Interface
- [x] Add `GetForUpdate` method to `store.UserCardStatsStore` interface
- [x] Implement method in postgres store with proper locking
- [x] Update unit tests for the new method
- Effort: Small
- Depends On: None

### T010: Implement Transaction Handling
- [x] Create consistent transaction mechanism across repositories
- [x] Ensure proper rollback on errors
- [x] Update service methods to use transactions for atomic operations
- Effort: Medium
- Depends On: T008

## Test Improvements

### T011: Extract Common Test Setup
- [x] Create `testutils.SetupCardReviewTestServer` helper
- [x] Move duplicate setup code from API tests to helper
- [x] Ensure proper cleanup in all test paths
- Effort: Small
- Depends On: None

### T012: Implement Table-Driven API Tests
- [x] Refactor card review API tests to use table-driven approach
- [x] Create test case structs with inputs and expected outputs
- [x] Use `t.Run` for individual test isolation
- Effort: Small
- Depends On: T011

### T013: Use API Test Helpers
- [x] Create helpers for API request execution and response validation
- [x] Replace manual HTTP client code with standardized helpers
- [x] Improve assertion messages for test failures
- Effort: Small
- Depends On: T012

### T014: Consolidate Domain Test Helpers
- [x] Move `createCardWithStats` helper to testutils package
- [x] Update all test files to use shared implementation
- [x] Remove duplicate code across test files
- Effort: Extra Small
- Depends On: T011

### T015: Add Test Cleanup Mechanisms
- [x] Review all tests for proper resource cleanup
- [x] Use `t.Cleanup` consistently for all test resources
- [x] Ensure database connections are properly closed
- Effort: Extra Small
- Depends On: T012

## Code Quality

### T016: Add Deterministic SQL Ordering
- [x] Add secondary sort keys to GetNextReviewCard query
- [x] Ensure ORDER BY includes card ID for deterministic ordering
- [x] Add test case with identical timestamps to verify
- Effort: Extra Small
- Depends On: None

### T017: Update Pre-commit Hooks
- [  ] Enhance pre-commit checks for panic usage
- [  ] Add verification of SQL query deterministic ordering
- [  ] Improve hook reliability and error messages
- Effort: Small
- Depends On: T004, T016

### T018: Remove Unused Import Workarounds
- [  ] Eliminate var assignments used to prevent unused import errors
- [  ] Remove unnecessary imports throughout codebase
- [  ] Fix any resulting compilation errors
- Effort: Extra Small
- Depends On: None

### T019: Review Repository Injection in Main
- [  ] Verify all dependency injection happens in main.go
- [  ] Ensure no adapters are created in router setup
- [  ] Document DI approach for future reference
- Effort: Extra Small
- Depends On: T008

### T020: Improve Error Handling in API Responses
- [  ] Prevent internal error details from leaking in responses
- [  ] Add structured error responses with appropriate status codes
- [  ] Ensure sensitive data (DB errors, SQL) is not exposed
- Effort: Small
- Depends On: None
