# TODO

## Password Validation Simplification

- [x] **Update validatePasswordComplexity function:** Implement length-based password validation
  - **Action:** Modify the `validatePasswordComplexity` function in `internal/domain/user.go` to enforce a minimum length of 12 characters and a maximum length of 72 characters (bcrypt's practical limit), removing all character class requirements
  - **Depends On:** None
  - **AC Ref:** Password Validation Simplification.2

- [x] **Simplify password validation in User model:** Update User.Validate method
  - **Action:** Update the `Validate` method in `internal/domain/user.go` to use the simplified length check when `Password` field is present, ensuring it still checks for `HashedPassword` presence if `Password` is empty
  - **Depends On:** Update validatePasswordComplexity function
  - **AC Ref:** Password Validation Simplification.2, Password Validation Simplification.4

- [x] **Add Godoc comments to password validation function:** Document the new approach
  - **Action:** Add clear Godoc comments to the validatePasswordComplexity function in `internal/domain/user.go`, explaining the length-based approach and rationale
  - **Depends On:** Update validatePasswordComplexity function
  - **AC Ref:** Password Validation Simplification.4, Documentation Improvements.1

- [x] **Update password validation domain tests:** Adjust tests for new validation rules
  - **Action:** Modify existing tests in `internal/domain/user_test.go` to verify the new length-based password validation rules, including edge cases (too short, too long, exact limits)
  - **Depends On:** Simplify password validation in User model
  - **AC Ref:** Password Validation Simplification.4

## Code Quality Improvements

- [x] **Refactor transaction handling in PostgresUserStore.Create:** Use named error returns
  - **Action:** Modify the `Create` method in `internal/platform/postgres/user_store.go` to use named return values for errors and update the deferred rollback function to check this named error variable before rolling back
  - **Depends On:** None
  - **AC Ref:** Code Quality Improvements.1

- [ ] **Refactor transaction handling in PostgresUserStore.Update:** Use named error returns
  - **Action:** Modify the `Update` method in `internal/platform/postgres/user_store.go` to use named return values for errors and update the deferred rollback function to check this named error variable before rolling back
  - **Depends On:** None
  - **AC Ref:** Code Quality Improvements.1

- [ ] **Optimize password hash fetching in PostgresUserStore.Update:** Conditionally fetch password hash
  - **Action:** Refactor `internal/platform/postgres/user_store.go:225-240` to only query the existing `hashed_password` when necessary (i.e., when the input `user.Password` field is empty)
  - **Depends On:** None
  - **AC Ref:** Core Principles and Design Improvements.1

- [ ] **Add context.Context to test helper functions:** Improve helper function signatures
  - **Action:** Modify the test helper functions (`insertTestUser`, `getUserByID`, `countUsers`) in `internal/platform/postgres/user_store_test.go` to accept `ctx context.Context` as the first argument and pass it to the appropriate database methods
  - **Depends On:** None
  - **AC Ref:** Code Quality Improvements.2

- [ ] **Remove unnecessary nolint:unused directives:** Clean up code
  - **Action:** Remove the `//nolint:unused` comments from helper functions in `internal/platform/postgres/user_store_test.go` as they are actively used within the test file
  - **Depends On:** Add context.Context to test helper functions
  - **AC Ref:** Code Quality Improvements.3

- [x] **Add TODO comment for improving validateEmailFormat:** Track technical debt
  - **Action:** Add a `// TODO:` comment above the `validateEmailFormat` function in `internal/domain/user.go` indicating it's a basic implementation that should be replaced with more robust validation in the future
  - **Depends On:** None
  - **AC Ref:** Code Quality Improvements.4

## Testing Improvements

- [ ] **Refactor setupTestDB to use project migrations:** Eliminate schema duplication
  - **Action:** Modify the `setupTestDB` function in `internal/platform/postgres/user_store_test.go` to use the project's migrations instead of direct `CREATE TABLE` SQL, ensuring tests run against the canonical schema definition
  - **Depends On:** None
  - **AC Ref:** Testing Improvements.1

- [ ] **Add password validation integration tests:** Test validation in store methods
  - **Action:** Add test cases to `TestPostgresUserStore_Create` and `TestPostgresUserStore_Update` in `internal/platform/postgres/user_store_test.go` that explicitly verify password validation rejects passwords that don't meet length requirements
  - **Depends On:** Update password validation domain tests
  - **AC Ref:** Testing Improvements.2

- [ ] **Refactor insertTestUser to use PostgresUserStore.Create:** Improve test helper
  - **Action:** Modify the `insertTestUser` helper function to use the `PostgresUserStore.Create` method instead of direct SQL, leveraging automatic password hashing
  - **Depends On:** Refactor setupTestDB to use project migrations
  - **AC Ref:** Testing Improvements.3, Testing Improvements.4

- [ ] **Refactor getUserByID to use PostgresUserStore.GetByID:** Improve test helper
  - **Action:** Modify the `getUserByID` helper function to use the `PostgresUserStore.GetByID` method instead of direct SQL queries
  - **Depends On:** Refactor setupTestDB to use project migrations
  - **AC Ref:** Testing Improvements.3

- [ ] **Improve test data isolation:** Enable parallel testing
  - **Action:** Implement a better test isolation strategy that allows for parallel test execution, such as using transaction-based isolation or unique database/schema names for each test
  - **Depends On:** Refactor setupTestDB to use project migrations
  - **AC Ref:** Testing Improvements.5

## Architecture Improvements

- [ ] **Add configuration option for bcrypt cost:** Make security tunable
  - **Action:** Add a `bcrypt_cost` integer field to `AuthConfig` in `internal/config/config.go` with appropriate validation. Update `PostgresUserStore.Create` and `PostgresUserStore.Update` to use this configured cost instead of `bcrypt.DefaultCost`
  - **Depends On:** None
  - **AC Ref:** Architecture Improvements.1

## Documentation and Organization

- [ ] **Break down Authentication Implementation in BACKLOG.md:** Improve task tracking
  - **Action:** Edit `BACKLOG.md` to replace the single "Authentication Implementation" item with separate, granular tasks: User Store Implementation, JWT Authentication Service, Authentication API Endpoints, and Authentication Middleware
  - **Depends On:** None
  - **AC Ref:** Core Principles and Design Improvements.2

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS

- [ ] **Issue/Assumption:** Acceptance Criteria References
  - **Context:** The `PLAN.md` does not have explicit AC IDs.
  - **Assumption:** The `AC Ref` fields in this `TODO.md` refer to the specific numbered items within each section of the `PLAN.md` document (e.g., "Password Validation Simplification.2" refers to item 2 in that section).

- [ ] **Issue/Assumption:** Test Helper countUsers Refactoring
  - **Context:** Testing Improvements item 3 suggests refactoring helpers to use store methods. However, `countUsers` is primarily for verification purposes.
  - **Assumption:** The `countUsers` helper will retain direct SQL access for verification purposes, but will be updated to accept `context.Context` as per Code Quality Improvements item 2.
