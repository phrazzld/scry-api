# TODO

## 1. Define Store Interface and Errors
- [x] **Define UserStore Interface:**
  - **Action:** Define the `UserStore` interface in `internal/store/user.go` with methods `Create`, `GetByID`, `GetByEmail`, `Update`, `Delete` as specified in the plan. Include comments explaining each method and its potential errors.
  - **Depends On:** None
  - **AC Ref:** PLAN.md Section 1

- [x] **Define Common Store Errors:**
  - **Action:** Define the `ErrUserNotFound` and `ErrEmailExists` error variables in `internal/store/user.go`.
  - **Depends On:** None
  - **AC Ref:** PLAN.md Section 1

## 2. Implement PostgreSQL User Store Structure
- [x] **Create PostgresUserStore Struct:**
  - **Action:** Create the `PostgresUserStore` struct in `internal/platform/postgres/user_store.go`, including the `db *sql.DB` field and the `uniqueViolationCode` constant. Import necessary packages.
  - **Depends On:** Define UserStore Interface
  - **AC Ref:** PLAN.md Section 2

- [x] **Implement NewPostgresUserStore Constructor:**
  - **Action:** Implement the `NewPostgresUserStore(db *sql.DB) *PostgresUserStore` constructor function in `internal/platform/postgres/user_store.go`.
  - **Depends On:** Create PostgresUserStore Struct
  - **AC Ref:** PLAN.md Section 2

## 3. Implement Data Validation
- [ ] **Implement Domain-Level Password Validation:**
  - **Action:** Enhance the `Validate()` method on the `domain.User` struct (`internal/domain/user.go`) to include checks for password complexity requirements.
  - **Depends On:** None
  - **AC Ref:** PLAN.md Section 4.1

## 4. Implement Store Methods
- [ ] **Implement PostgresUserStore Create Method:**
  - **Action:** Implement the `Create(ctx context.Context, user *domain.User) error` method on `PostgresUserStore`. Include calling `user.Validate()`, hashing the password using `bcrypt` (clearing the plaintext password field), executing the SQL INSERT statement using parameterized queries, and handling potential unique constraint violations (returning `store.ErrEmailExists`). Log errors appropriately using `slog`.
  - **Depends On:** Implement NewPostgresUserStore Constructor, Implement Domain-Level Password Validation, Define Common Store Errors
  - **AC Ref:** PLAN.md Sections 2.1, 3, 4.2

- [ ] **Implement PostgresUserStore GetByID Method:**
  - **Action:** Implement the `GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)` method on `PostgresUserStore`. Use parameterized SQL SELECT query, map the row to a `domain.User` struct, and handle the "not found" case by returning `store.ErrUserNotFound`.
  - **Depends On:** Implement NewPostgresUserStore Constructor, Define Common Store Errors
  - **AC Ref:** PLAN.md Section 2.2

- [ ] **Implement PostgresUserStore GetByEmail Method:**
  - **Action:** Implement the `GetByEmail(ctx context.Context, email string) (*domain.User, error)` method on `PostgresUserStore`. Use parameterized SQL SELECT query, map the row to a `domain.User` struct, and handle the "not found" case by returning `store.ErrUserNotFound`.
  - **Depends On:** Implement NewPostgresUserStore Constructor, Define Common Store Errors
  - **AC Ref:** PLAN.md Section 2.2

- [ ] **Implement PostgresUserStore Update Method:**
  - **Action:** Implement the `Update(ctx context.Context, user *domain.User) error` method on `PostgresUserStore`. Include calling `user.Validate()`, checking if the password needs rehashing (using `bcrypt`) and updating it if necessary, executing the SQL UPDATE statement using parameterized queries, handling potential unique constraint violations for email (returning `store.ErrEmailExists`), and handling "not found" cases (returning `store.ErrUserNotFound`). Log errors appropriately.
  - **Depends On:** Implement NewPostgresUserStore Constructor, Implement Domain-Level Password Validation, Define Common Store Errors
  - **AC Ref:** PLAN.md Sections 2.3, 3, 4.2

- [ ] **Implement PostgresUserStore Delete Method:**
  - **Action:** Implement the `Delete(ctx context.Context, id uuid.UUID) error` method on `PostgresUserStore`. Use parameterized SQL DELETE statement and handle "not found" cases by checking rows affected or using a specific query, returning `store.ErrUserNotFound` if the user doesn't exist.
  - **Depends On:** Implement NewPostgresUserStore Constructor, Define Common Store Errors
  - **AC Ref:** PLAN.md Section 2.4

## 5. Testing Implementation
- [ ] **Set Up User Store Test File and Helpers:**
  - **Action:** Create the test file `internal/platform/postgres/user_store_test.go`. Include necessary imports (`testing`, `testify`, `uuid`, domain, store, postgres, testutils, etc.). Set up test helper functions, potentially including setup/teardown logic for a test PostgreSQL database (e.g., using `testcontainers-go` or similar, connecting via `testutils.GetTestDatabaseURL`).
  - **Depends On:** Implement NewPostgresUserStore Constructor
  - **AC Ref:** PLAN.md Section 5

- [ ] **Write Integration Tests for Create Method:**
  - **Action:** Implement integration tests in `user_store_test.go` covering the `Create` method. Test cases should include successful creation, attempting to create a user with an existing email (expecting `store.ErrEmailExists`), and validation failures passed from `domain.User.Validate()`. Verify data integrity and password hashing.
  - **Depends On:** Implement PostgresUserStore Create Method, Set Up User Store Test File and Helpers
  - **AC Ref:** PLAN.md Section 5

- [ ] **Write Integration Tests for GetByID Method:**
  - **Action:** Implement integration tests in `user_store_test.go` covering the `GetByID` method. Test cases should include successful retrieval and attempting to retrieve a non-existent user (expecting `store.ErrUserNotFound`).
  - **Depends On:** Implement PostgresUserStore GetByID Method, Set Up User Store Test File and Helpers
  - **AC Ref:** PLAN.md Section 5

- [ ] **Write Integration Tests for GetByEmail Method:**
  - **Action:** Implement integration tests in `user_store_test.go` covering the `GetByEmail` method. Test cases should include successful retrieval and attempting to retrieve a user by a non-existent email (expecting `store.ErrUserNotFound`).
  - **Depends On:** Implement PostgresUserStore GetByEmail Method, Set Up User Store Test File and Helpers
  - **AC Ref:** PLAN.md Section 5

- [ ] **Write Integration Tests for Update Method:**
  - **Action:** Implement integration tests in `user_store_test.go` covering the `Update` method. Test cases should include successful update (with and without password change), attempting to update a non-existent user (expecting `store.ErrUserNotFound`), attempting to update email to an existing one (expecting `store.ErrEmailExists`), and validation failures. Verify data integrity and password rehashing.
  - **Depends On:** Implement PostgresUserStore Update Method, Set Up User Store Test File and Helpers
  - **AC Ref:** PLAN.md Section 5

- [ ] **Write Integration Tests for Delete Method:**
  - **Action:** Implement integration tests in `user_store_test.go` covering the `Delete` method. Test cases should include successful deletion and attempting to delete a non-existent user (expecting `store.ErrUserNotFound`). Verify the user is actually removed.
  - **Depends On:** Implement PostgresUserStore Delete Method, Set Up User Store Test File and Helpers
  - **AC Ref:** PLAN.md Section 5

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS
- [ ] **Issue/Assumption:** Assumed `domain.User` struct exists but needs enhancement for password complexity validation.
  - **Context:** PLAN.md Section 4.1 mentions `domain.User` should have `Validate()` checking password complexity, implying the struct exists but validation needs adding/updating.

- [ ] **Issue/Assumption:** Assumed `bcrypt.DefaultCost` is the appropriate cost factor for password hashing.
  - **Context:** PLAN.md Section 3 shows `bcrypt.DefaultCost` in the example. Confirm if this is sufficient or if a configurable/higher cost is needed.

- [ ] **Issue/Assumption:** Assumed the `sql.DB` dependency for `NewPostgresUserStore` will be provided externally.
  - **Context:** PLAN.md Section 2 shows `NewPostgresUserStore` accepting `*sql.DB`. The plan doesn't cover where this DB connection pool is created and managed.

- [ ] **Issue/Assumption:** Assumed integration tests will use a real PostgreSQL instance.
  - **Context:** PLAN.md Section 5 mentions "integration tests with a real PostgreSQL database".

- [ ] **Issue/Assumption:** Password complexity rules are not specified.
  - **Context:** PLAN.md Section 4.1 mentions checking "Password complexity requirements" but doesn't define them (e.g., min length, character types). Assuming a basic length check (e.g., 8 chars) for now.

- [ ] **Issue/Assumption:** "Other business rules" for domain validation are not specified.
  - **Context:** PLAN.md Section 4.1 mentions "Other business rules". Assuming only email format and password complexity are required for the `User` domain validation at this stage.
