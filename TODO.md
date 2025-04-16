# TODO

## 1. Fix Migration Logger Exit Handling (Medium Risk)
- [x] **T001:** Refactor slogGooseLogger.Fatalf to remove os.Exit(1)
    - **Action:** Modify the `Fatalf` method in `cmd/server/main.go`'s `slogGooseLogger` struct. Remove the `os.Exit(1)` call and ensure it only logs the error using `slog.Error`.
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 1
    - **Note:** This task was already completed. The `Fatalf` method in `cmd/server/main.go` (lines 327-331) only logs the error and doesn't call `os.Exit(1)` as noted in the comment.

- [x] **T002:** Modify runMigrations to return errors
    - **Action:** Update the `runMigrations` function signature in `cmd/server/main.go` to return an `error`. Ensure that errors encountered during `goose` operations (e.g., `goose.Up`, `goose.Down`) are returned instead of handled internally by the logger's `Fatalf`.
    - **Depends On:** [T001]
    - **AC Ref:** PLAN.md Section 1
    - **Note:** This task was already completed. The `runMigrations` function already returns errors from all goose operations (lines 484-486 for Up, and similar for Down, Status, Create, Version). The function signature already returns an error.

- [x] **T003:** Update main function migration handling
    - **Action:** In the `main` function in `cmd/server/main.go`, update the block that calls `runMigrations`. Check the returned error. If an error is present, log it using the application logger and then call `os.Exit(1)`.
    - **Depends On:** [T002]
    - **AC Ref:** PLAN.md Section 1
    - **Note:** This task was already completed. The `main` function already checks the returned error from `runMigrations` (lines 79-86), logs it using the application logger, and calls `os.Exit(1)` if an error is present.

## 2. Fix Integration Test Database Management (Low Risk)
- [x] **T004:** Modify TestMain to initialize shared testDB
    - **Action:** In the `cmd/server` package (likely `main_task_test.go` based on context), modify or ensure the `TestMain` function initializes a package-level `var testDB *sql.DB`. This variable should hold the database connection used by integration tests within that package. Handle connection errors appropriately. Ensure the connection is closed during cleanup.
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 4
    - **Note:** This task was already completed. The `TestMain` function in `main_task_test.go` already initializes a package-level `var testDB *sql.DB` variable (line 29) and properly handles connection errors and cleanup (lines 129-133).

- [x] **T005:** Refactor setupTestServer to accept *sql.DB
    - **Action:** Modify the `setupTestServer` function in `cmd/server/auth_integration_test.go` to accept a `*sql.DB` parameter. Update the function to use this provided database connection instead of calling `testutils.GetTestDB()` internally.
    - **Depends On:** [T004]
    - **AC Ref:** PLAN.md Section 4

- [x] **T006:** Update integration tests to use shared testDB
    - **Action:** Update integration test functions like `TestAuthIntegration` in `cmd/server/auth_integration_test.go` to retrieve the test server by calling the refactored `setupTestServer` with the shared `testDB` variable initialized in `TestMain`. Include checks to skip tests if `testDB` is nil.
    - **Depends On:** [T005]
    - **AC Ref:** PLAN.md Section 4

## 3. Simplify Password Validation Logic (Low Risk)
- [x] **T007:** Create ValidatePassword function for plaintext validation
    - **Action:** In `internal/domain/user.go`, create a new standalone function `ValidatePassword(password string) error` that checks only the plaintext password length requirements. Return `ErrPasswordTooShort` or `ErrPasswordTooLong` appropriately.
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 3

- [x] **T008:** Refactor User.Validate for persistence checks
    - **Action:** Modify the `Validate` method of the `User` struct in `internal/domain/user.go`. Remove the plaintext password length check. Ensure it validates only fields relevant for persistence: non-nil ID, valid email format, and non-empty `HashedPassword`. Return appropriate errors (`ErrInvalidID`, `ErrInvalidEmail`, `ErrPasswordRequired`).
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 3
    - **Note:** Modified the `Validate` method to focus only on persistence-relevant validations. Updated tests to reflect the new validation logic. The plaintext password length check is now handled entirely by the `ValidatePassword` function.

- [x] **T009:** Update NewUser to use ValidatePassword
    - **Action:** Modify the `NewUser` function in `internal/domain/user.go`. Call the new `ValidatePassword` function (from T007) on the input password. Return an error immediately if validation fails. Remove any password validation logic previously within `NewUser`.
    - **Depends On:** [T007]
    - **AC Ref:** PLAN.md Section 3
    - **Note:** Modified `NewUser` to validate password format using the `ValidatePassword` function before creating the user object. This improves separation of concerns by handling input validation before persistence validation.

- [x] **T010:** Update PostgresUserStore.Create for correct password handling
    - **Action:** Review and update the `Create` method in `internal/platform/postgres/user_store.go`. Ensure it correctly hashes the `Password` field from the input `User` struct, stores it in the `HashedPassword` field, and clears the `Password` field before saving to the database. Ensure it calls the refactored `User.Validate` (from T008) before hashing and saving.
    - **Depends On:** [T008, T009]
    - **AC Ref:** PLAN.md Section 3
    - **Note:** Updated both `Create` and `Update` methods to explicitly validate password format using `domain.ValidatePassword()` before hashing. Moved the persistence validation (`User.Validate()`) to occur after password hashing to ensure the user object is in the correct state for validation.

## 4. Address Unused AuthResponse Field (Low Risk)
- [x] **T011:** Update AuthHandler.Register to populate ExpiresAt
    - **Action:** Modify the `Register` method in `internal/api/auth_handler.go`. After generating the JWT token, calculate the expiration time using `cfg.Auth.TokenLifetimeMinutes`. Populate the `ExpiresAt` field in the `AuthResponse` struct before sending the JSON response. Ensure `ExpiresAt` is formatted appropriately (e.g., RFC3339).
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 6
    - **Note:** Updated `AuthHandler` to include `authConfig`, allowing access to token lifetime settings. Added expiration time calculation to the `Register` method and included it in the response. Used RFC3339 format for the expiration timestamp.

- [x] **T012:** Update AuthHandler.Login to populate ExpiresAt
    - **Action:** Modify the `Login` method in `internal/api/auth_handler.go`. After generating the JWT token, calculate the expiration time using `cfg.Auth.TokenLifetimeMinutes`. Populate the `ExpiresAt` field in the `AuthResponse` struct before sending the JSON response. Ensure `ExpiresAt` is formatted appropriately (e.g., RFC3339).
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 6
    - **Note:** Updated `Login` method to calculate token expiration time based on `authConfig.TokenLifetimeMinutes` and included it in the response using RFC3339 format, consistent with the `Register` method's implementation.

## 5. Improve Server Initialization Structure (Low Risk)
- [x] **T013:** Create appDependencies struct
    - **Action:** Define a new struct `appDependencies` in `cmd/server/main.go` to hold shared application dependencies like Config, Logger, DB, UserStore, JWTService, etc.
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 2
    - **Note:** Created the `appDependencies` struct with fields for all shared dependencies identified in the codebase. The struct is currently unused (which is expected) but will be used in subsequent tasks to simplify dependency management.

- [x] **T014:** Extract loadConfig function
    - **Action:** Create a new function `loadConfig() (*config.Config, error)` in `cmd/server/main.go`. Move the configuration loading logic (using `config.Load()`) from `initializeApp` into this new function.
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 2
    - **Note:** Created the `loadConfig` function that encapsulates the config loading logic and updated both `initializeApp` and the migration code in `main` to use this new function. This improves code organization and reduces duplication.

- [x] **T015:** Extract setupLogger function
    - **Action:** Create a new function `setupLogger(cfg *config.Config) *slog.Logger` in `cmd/server/main.go`. Move the logger setup logic (using `logger.Setup()`) from `initializeApp` into this new function. Handle potential errors from `logger.Setup`.
    - **Depends On:** [T014]
    - **AC Ref:** PLAN.md Section 2
    - **Note:** Created the `setupLogger` function to encapsulate logger configuration and initialization. Updated both the `initializeApp` function and migration code in `main` to use this function, which provides consistent logger setup throughout the application.

- [x] **T016:** Extract setupDatabase function
    - **Action:** Create a new function `setupDatabase(cfg *config.Config, logger *slog.Logger) (*sql.DB, error)` in `cmd/server/main.go`. Move the database connection opening and ping logic from `initializeApp` (or `startServer`) into this function. Return the `*sql.DB` instance or an error.
    - **Depends On:** [T014, T015]
    - **AC Ref:** PLAN.md Section 2
    - **Note:** Created the `setupDatabase` function that encapsulates database initialization, connection pool configuration, and connectivity validation. Updated both the `initializeApp` and `startServer` functions to use this function, eliminating code duplication and ensuring consistent database setup across the application.

- [x] **T017:** Extract setupJWTService function
    - **Action:** Create a new function `setupJWTService(cfg *config.Config) (auth.JWTService, error)` in `cmd/server/main.go`. Move the JWT service initialization logic (using `auth.NewJWTService()`) from `initializeApp` into this function.
    - **Depends On:** [T014]
    - **AC Ref:** PLAN.md Section 2
    - **Note:** Created the `setupJWTService` function that encapsulates JWT service initialization. Updated both the `initializeApp` and `startServer` functions to use this function, ensuring consistent JWT service setup and reducing code duplication.

- [x] **T018:** Extract setupRouter function
    - **Action:** Create a new function `setupRouter(deps *appDependencies) *chi.Mux` in `cmd/server/main.go`. Move the router creation (`chi.NewRouter()`), middleware setup, and route registration logic from `startServer` into this function. The function should accept the `appDependencies` struct to access handlers and services.
    - **Depends On:** [T013]
    - **AC Ref:** PLAN.md Section 2
    - **Note:** Created the `setupRouter` function that centralizes router creation, middleware configuration, and route registration. The function accepts the `appDependencies` struct to access necessary handlers and services, and returns the configured router. Updated the `startServer` function to use this function, improving code organization and maintainability.

- [x] **T019:** Extract setupTaskRunner function
    - **Action:** Create a new function `setupTaskRunner(deps *appDependencies) (*task.Runner, error)` in `cmd/server/main.go`. Move the task runner initialization logic (including creating the task store and runner) from `startServer` into this function. Accept `appDependencies` for necessary components like DB, Config, Logger.
    - **Depends On:** [T013]
    - **AC Ref:** PLAN.md Section 2
    - **Note:** Created the `setupTaskRunner` function that encapsulates task runner initialization, configuration, and startup. The function accepts the `appDependencies` struct to access necessary components like DB, Config, and Logger. Updated the `startServer` function to use this function, improving code organization and making dependency flow more explicit with the two-stage approach (create deps, then create task runner, then update deps with the runner).

- [x] **T020:** Remove initializeApp function
    - **Action:** Delete the `initializeApp` function from `cmd/server/main.go` as its logic has been extracted into smaller functions. Update the `main` function to call the new helper functions directly for initial setup before `startServer`.
    - **Depends On:** [T014, T015, T016, T017]
    - **AC Ref:** PLAN.md Section 2
    - **Note:** Removed the `initializeApp` function and moved its functionality directly into the `main` function by calling the extracted setup functions (loadConfig, setupLogger, setupDatabase, setupJWTService) in sequence. Also updated the integration test to use `loadConfig` directly instead of `initializeApp`. This further improves code organization by removing an unnecessary abstraction layer.

- [x] **T021:** Refactor startServer function
    - **Action:** Modify the `startServer` function in `cmd/server/main.go`. Remove the extracted logic. Update it to:
        1. Initialize dependencies using the new helper functions (`setupDatabase`, `setupJWTService`, etc.) and populate the `appDependencies` struct.
        2. Call `setupRouter` and `setupTaskRunner` using the `appDependencies`.
        3. Start the task runner.
        4. Configure and start the `http.Server`.
        5. Handle graceful shutdown.
        Ensure `db.Close()` and `taskRunner.Stop()` are called appropriately on shutdown.
    - **Depends On:** [T013, T016, T017, T018, T019, T020]
    - **AC Ref:** PLAN.md Section 2
    - **Note:** Refactored the `startServer` function to use a more structured, step-by-step approach with clear comments documenting each stage of the server initialization. Improved variable naming for clarity (e.g., `router` instead of `r`, `server` instead of `srv`, `shutdownSignal` instead of `stop`) and added more comprehensive code documentation. The function now clearly follows the dependency flow pattern: initialize dependencies → build dependency struct → set up task runner → set up router → start server → handle shutdown.

## 6. Centralize Mock Implementations (Low Risk)
- [x] **T022:** Create internal/mocks package
    - **Action:** Create the directory `internal/mocks`. Add a `doc.go` file explaining the purpose of the package.
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 5
    - **Note:** Created the `internal/mocks` directory and added a comprehensive `doc.go` file that explains the purpose of the package, key features, usage examples, and guidelines for adding new mocks. This establishes a foundation for centralizing mock implementations that will be moved from individual test files in subsequent tasks.

- [x] **T023:** Move MockJWTService to internal/mocks
    - **Action:** Move the `MockJWTService` struct and its methods from `internal/api/auth_handler_test.go` (and potentially `internal/api/middleware/auth_test.go`) to a new file `internal/mocks/jwt_service.go`. Ensure the package declaration is `package mocks`. Update imports as needed.
    - **Depends On:** [T022]
    - **AC Ref:** PLAN.md Section 5
    - **Note:** Created `internal/mocks/jwt_service.go` with a centralized `MockJWTService` implementation that replaces the duplicated mocks in both test files. Updated the implementation to use exported fields (Token, Err, ValidateErr, Claims) for easier test configuration. Added function field options (GenerateTokenFn, ValidateTokenFn) for more flexible test behavior customization. Updated all test files to use the new centralized mock.

- [ ] **T024:** Move MockUserStore to internal/mocks
    - **Action:** Move the `MockUserStore` struct and its methods from `internal/api/auth_handler_test.go` to a new file `internal/mocks/user_store.go`. Ensure the package declaration is `package mocks`. Update imports as needed.
    - **Depends On:** [T022]
    - **AC Ref:** PLAN.md Section 5

- [ ] **T025:** Move MockPasswordVerifier to internal/mocks
    - **Action:** Move the `MockPasswordVerifier` struct and its methods from `internal/api/auth_handler_test.go` to a new file `internal/mocks/password_verifier.go`. Ensure the package declaration is `package mocks`. Update imports as needed.
    - **Depends On:** [T022]
    - **AC Ref:** PLAN.md Section 5

- [ ] **T026:** Update auth_handler_test.go to use centralized mocks
    - **Action:** Modify `internal/api/auth_handler_test.go`. Remove the local mock definitions. Update the import statements to use `github.com/phrazzld/scry-api/internal/mocks`. Ensure tests still instantiate and use the mocks correctly from the new package.
    - **Depends On:** [T023, T024, T025]
    - **AC Ref:** PLAN.md Section 5

- [ ] **T027:** Update middleware/auth_test.go to use centralized mocks
    - **Action:** Modify `internal/api/middleware/auth_test.go`. Remove the local mock definition for `MockJWTService`. Update the import statements to use `github.com/phrazzld/scry-api/internal/mocks`. Ensure tests still instantiate and use the `MockJWTService` correctly from the new package.
    - **Depends On:** [T023]
    - **AC Ref:** PLAN.md Section 5

## 7. Improve Email Validation (Low Risk)
- [ ] **T028:** Add TODO comment to validateEmailFormat
    - **Action:** Add the following comment above the `validateEmailFormat` function in `internal/domain/user.go`:
      ```go
      // validateEmailFormat performs basic validation of email format.
      // TODO: This is a basic implementation that only checks for @ symbol presence.
      // In the future, this should be replaced with a more robust email validation
      // library or regex that follows RFC 5322 standards, with consideration for
      // internationalized email addresses (IDN) according to RFC 6530.
      ```
    - **Depends On:** None
    - **AC Ref:** PLAN.md Section 7

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS
- [ ] **Issue/Assumption:** Acceptance Criteria References
    - **Context:** The `PLAN.md` document does not have explicit AC IDs.
    - **Assumption:** The `AC Ref` fields in this `TODO.md` refer to the relevant section number and associated issue description within the `PLAN.md` document (e.g., "PLAN.md Section 1"). This implies successful completion of the tasks under a section addresses the issue described in that section of the plan.

- [ ] **Issue/Assumption:** Mock Implementations Scope
    - **Context:** PLAN.md Section 5 mentions moving `MockUserStore`, `MockPasswordVerifier`, and `MockJWTService`.
    - **Assumption:** These are the only mocks currently defined inline within the specified test files (`auth_handler_test.go`, `middleware/auth_test.go`) that need centralization. If other inline mocks exist, they should also be moved as part of these tasks or new tasks created.
