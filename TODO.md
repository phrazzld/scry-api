# TODO

## Dependencies
- [x] **Add pressly/goose dependency:** Add goose migration framework to the project
  - **Action:** Run `go get github.com/pressly/goose/v3` to add the goose library to `go.mod`.
  - **Depends On:** None
  - **AC Ref:** Section 2.1.1

- [x] **Add PostgreSQL driver:** Add pgx PostgreSQL driver to the project
  - **Action:** Run `go get github.com/jackc/pgx/v5/stdlib` to add the PostgreSQL driver compatible with `database/sql` to `go.mod`.
  - **Depends On:** None
  - **AC Ref:** Section 2.1.2

- [x] **Tidy go.mod file:** Ensure go.mod is consistent
  - **Action:** Run `go mod tidy` to ensure `go.mod` and `go.sum` are consistent after adding dependencies.
  - **Depends On:** Add pressly/goose dependency, Add PostgreSQL driver
  - **AC Ref:** Section 2.1.2

## Migration Directory Structure
- [x] **Create migrations directory:** Create directory for SQL migration files
  - **Action:** Create the directory `internal/platform/postgres/migrations` for storing SQL migration files.
  - **Depends On:** None
  - **AC Ref:** Section 2.2.1

- [x] **Add .keep file:** Ensure empty migrations directory is tracked in Git
  - **Action:** Add an empty `.keep` file to `internal/platform/postgres/migrations` to ensure the directory is tracked by Git even when empty.
  - **Depends On:** Create migrations directory
  - **AC Ref:** Section 2.2.2

## Migration Command Handling
- [x] **Define migration command-line flags:** Add command flags to main.go
  - **Action:** Use the standard `flag` package in `cmd/server/main.go` to define `-migrate` (string) and `-name` (string) flags for controlling migration operations. Parse flags early in `main`.
  - **Depends On:** None
  - **AC Ref:** Section 2.3.1, 3.1

- [x] **Add conditional migration execution logic:** Modify main.go to handle migration commands
  - **Action:** Modify `cmd/server/main.go` after flag parsing to check if the `-migrate` flag was provided. If it was, call the `runMigrations` function and exit the application based on the result. If not, proceed with normal server startup.
  - **Depends On:** Define migration command-line flags, Define runMigrations function signature
  - **AC Ref:** Section 2.3.1, 3.1

- [x] **Define runMigrations function signature:** Define function to encapsulate migration logic
  - **Action:** Define the function `runMigrations(cfg *config.Config, command string, args ...string) error` in `cmd/server/main.go`. This function will encapsulate all migration logic.
  - **Depends On:** None
  - **AC Ref:** Section 2.3.2, 3.1

- [x] **Implement database connection logic:** Connect to database in runMigrations
  - **Action:** Inside `runMigrations`, use the provided `cfg *config.Config` to get the `cfg.Database.URL`. Open a `database/sql` connection using `sql.Open("pgx", cfg.Database.URL)`. Ensure the connection is closed using `defer db.Close()`. Ping the database to verify connectivity.
  - **Depends On:** Define runMigrations function signature, Add PostgreSQL driver
  - **AC Ref:** Section 2.3.2, 3.1

- [x] **Implement error handling for DB connection:** Add robust error handling
  - **Action:** Add robust error handling for `sql.Open` and `db.Ping` within `runMigrations`. Return descriptive errors using `fmt.Errorf` with `%w` for wrapping.
  - **Depends On:** Implement database connection logic
  - **AC Ref:** Section 2.3.2, 5.1

## Logging Integration
- [x] **Define slogGooseLogger struct and methods:** Create custom logger adapter for goose
  - **Action:** Create the `slogGooseLogger` struct and implement the `Printf(format string, v ...interface{})` and `Fatalf(format string, v ...interface{})` methods to adapt `goose`'s logging output to the application's `slog` logger. Ensure `Fatalf` logs an error but does *not* call `os.Exit(1)` directly (let main.go handle exits).
  - **Depends On:** None
  - **AC Ref:** Section 2.4.1, 3.1

- [x] **Set goose logger in runMigrations:** Configure goose to use the custom logger
  - **Action:** Instantiate `slogGooseLogger` and call `goose.SetLogger(&slogGooseLogger{})` at the beginning of the `runMigrations` function.
  - **Depends On:** Define runMigrations function signature, Define slogGooseLogger struct and methods
  - **AC Ref:** Section 2.4.2, 3.1

## Migration Command Implementation
- [x] **Implement up command logic:** Add support for applying migrations
  - **Action:** Add a case for "up" in the `switch command` block within `runMigrations`. Call `goose.Up(db, migrationsDir)` and return its result.
  - **Depends On:** Implement database connection logic, Set goose logger in runMigrations
  - **AC Ref:** Section 2.5, 3.1

- [x] **Implement down command logic:** Add support for rolling back migrations
  - **Action:** Add a case for "down" in the `switch command` block within `runMigrations`. Call `goose.Down(db, migrationsDir)` and return its result.
  - **Depends On:** Implement database connection logic, Set goose logger in runMigrations
  - **AC Ref:** Section 2.5, 3.1

- [x] **Implement status command logic:** Add support for checking migration status
  - **Action:** Add a case for "status" in the `switch command` block within `runMigrations`. Call `goose.Status(db, migrationsDir)` and return its result.
  - **Depends On:** Implement database connection logic, Set goose logger in runMigrations
  - **AC Ref:** Section 2.5, 3.1

- [x] **Implement create command logic:** Add support for creating new migrations
  - **Action:** Add a case for "create" in the `switch command` block within `runMigrations`. Check if a migration name was provided via the `-name` flag (passed in `args`). If not, return an error. Call `goose.Create(db, migrationsDir, args[0], "sql")` and return its result.
  - **Depends On:** Implement database connection logic, Set goose logger in runMigrations, Define migration command-line flags
  - **AC Ref:** Section 2.5, 3.1, 3.2

- [x] **Implement version command logic:** Add support for checking migration version
  - **Action:** Add a case for "version" in the `switch command` block within `runMigrations`. Call `goose.Version(db, migrationsDir)` and return its result.
  - **Depends On:** Implement database connection logic, Set goose logger in runMigrations
  - **AC Ref:** Section 2.5, 3.1

- [x] **Implement default case for unknown commands:** Handle invalid migration commands
  - **Action:** Add a `default` case to the `switch command` block in `runMigrations` that returns a formatted error indicating an unknown command was provided.
  - **Depends On:** Define runMigrations function signature
  - **AC Ref:** Section 2.5, 3.1

## Testing
- [x] **Implement unit tests for migration flag parsing:** Test flag parsing logic
  - **Action:** Write unit tests for `main.go` to verify that the `-migrate` and `-name` flags are correctly parsed under various scenarios (present, absent, combined).
  - **Depends On:** Define migration command-line flags
  - **AC Ref:** Section 2.6

- [x] **Implement unit tests for runMigrations command dispatch:** Test command routing logic
  - **Action:** Write unit tests for the `runMigrations` function, focusing on the `switch` statement logic. Mock the `goose` calls to verify that the correct `goose` function is called for each command string ("up", "down", "status", "create", "version", default). Verify argument handling for "create".
  - **Depends On:** Implement all migration command logic tasks
  - **AC Ref:** Section 2.6, 4.3

- [x] **Implement integration test for create command:** Test migration file creation
  - **Action:** Write an integration test that executes the application binary with `-migrate=create -name=test_migration`. Verify that the corresponding SQL file is created in the migrations directory with the correct naming convention and basic structure. Clean up the created file afterwards.
  - **Depends On:** Implement create command logic, Create migrations directory
  - **AC Ref:** Section 2.6, 3.2, 4.3

- [x] **Implement integration tests for migration execution:** Test up/down migration flow
  - **Action:** Write integration tests using a temporary PostgreSQL database (via testcontainers-go). Create a dummy migration file. Run the application with `-migrate=up`, check status/version, run `-migrate=down`, check status/version again to verify the core migration flow.
  - **Depends On:** Implement up, down, status and version command logic
  - **AC Ref:** Section 2.6, 4.3

## Documentation
- [x] **Update README.md with migration command usage:** Document command-line interface
  - **Action:** Add a new section to `README.md` explaining how to use the `-migrate` flag with the available commands (`up`, `down`, `status`, `create`, `version`) and the `-name` flag. Include practical examples.
  - **Depends On:** Implement all migration command logic tasks
  - **AC Ref:** Section 2.7.1, 4.5

- [x] **Document migration file format and naming conventions:** Explain migration file structure
  - **Action:** Add details to the `README.md` explaining the expected SQL migration file format (`-- +goose Up`, `-- +goose Down` sections) and the timestamp-based naming convention generated by the `create` command.
  - **Depends On:** Implement create command logic
  - **AC Ref:** Section 2.7.2, 4.5

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS
- [x] **Issue/Assumption:** Exit handling in slogGooseLogger.Fatalf
  - **Context:** PLAN.md Section 2.4.1 shows `slogGooseLogger.Fatalf` calling `os.Exit(1)`, but `main.go` already handles exits.
  - **Assumption:** The `slogGooseLogger.Fatalf` implementation should only log the error using `slog.Error` and not call `os.Exit(1)`. The `runMigrations` function will return errors to `main` which handles program exit consistently.

- [x] **Issue/Assumption:** Database for integration tests
  - **Context:** PLAN.md Section 2.6 (Add Tests), Section 4.3 (Testability).
  - **Assumption:** Integration tests requiring a database will use `testcontainers-go` to manage a temporary PostgreSQL instance, aligning with the project's testing strategy principles.

- [x] **Issue/Assumption:** No explicit Acceptance Criteria in PLAN.md
  - **Context:** The PLAN.md document does not contain explicitly labeled Acceptance Criteria.
  - **Assumption:** The section numbers provided in the AC Ref fields above refer to the corresponding sections in PLAN.md that define the requirements for each task.
