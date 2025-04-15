# TODO

## Design Principles (CORE_PRINCIPLES.md)
- [x] **Refactor UserCardStats Mutability:** Remove mutable methods from `UserCardStats` and rely solely on `srs.Service`.
  - **Action:** Delete the `UpdateReview` and `PostponeReview` methods from `internal/domain/user_card_stats.go`. Refactor any code that currently calls these methods to use the corresponding methods in `internal/domain/srs/service.go` instead, ensuring immutability is maintained. Update relevant tests.
  - **Depends On:** None
  - **AC Ref:** Design Principles Issue 1

- [x] **Correct Ease Factor DB Constraint:** Align the database check constraint for `ease_factor` with the defined algorithm minimum.
  - **Action:** Modify the SQL `CHECK` constraint in `internal/platform/postgres/migrations/20250415000004_create_user_card_stats_table.sql` from `CHECK (ease_factor > 1.0 AND ease_factor <= 2.5)` to `CHECK (ease_factor >= 1.3 AND ease_factor <= 2.5)`. Ensure the corresponding down migration (if applicable) is correct or add a new migration if necessary.
  - **Depends On:** None
  - **AC Ref:** Design Principles Issue 2

- [x] **Remove Redundant Local Dev Test Helpers:** Eliminate helper functions in `local_postgres_test.go` that duplicate existing configuration files.
  - **Action:** Delete the `generateDockerComposeYml` and `generateInitScript` functions from `infrastructure/local_dev/local_postgres_test.go`. Update the tests (e.g., `TestLocalPostgresSetup`) to assume the `docker-compose.yml` and `init-scripts/01-init.sql` files exist in their expected locations relative to the test file.
  - **Depends On:** None
  - **AC Ref:** Design Principles Issue 3

## Architectural Patterns (ARCHITECTURE_GUIDELINES.md)
- [x] **Refactor slogGooseLogger Fatalf:** Prevent `slogGooseLogger.Fatalf` from exiting the application directly.
  - **Action:** Remove the `os.Exit(1)` call from the `Fatalf` method in `cmd/server/main.go`'s `slogGooseLogger`. Modify the `runMigrations` function to return the error encountered during `goose` operations. Update the `main` function's migration handling block to check for errors returned by `runMigrations` and call `os.Exit(1)` there if an error occurred.
  - **Depends On:** None
  - **AC Ref:** Architectural Patterns Issue 1

- [x] **Add Explicit DB Password Management in Terraform:** Introduce a Terraform variable for the database user password.
  - **Action:** Define a new `variable "database_password"` in `infrastructure/terraform/variables.tf` (mark as sensitive). Update the `digitalocean_database_user` resource in `infrastructure/terraform/main.tf` to use this variable for the password instead of relying on auto-generation. Update `terraform.tfvars.example` and any relevant documentation.
  - **Depends On:** None
  - **AC Ref:** Architectural Patterns Issue 2

## Code Quality (CODING_STANDARDS.md)
- [x] **Enhance DB Connection Error Handling:** Add specific error type checks for database connection attempts.
  - **Action:** In `cmd/server/main.go` within the `runMigrations` function's `db.PingContext` error handling block (lines ~220-248), add specific checks using `errors.Is` or type assertions for common connection errors (e.g., `context.DeadlineExceeded`, `pgconn.PgError` for authentication failures, network errors) to provide more informative error messages.
  - **Depends On:** None
  - **AC Ref:** Code Quality Issue 1

- [x] **Add TODO for Robust Email Validation:** Mark the basic email validation for future improvement.
  - **Action:** Add a `// TODO:` comment above the `validateEmailFormat` function in `internal/domain/user.go` indicating that the current implementation is basic and should be replaced with a more robust validation library in a future task.
  - **Depends On:** None
  - **AC Ref:** Code Quality Issue 2

## Test Quality (TESTING_STRATEGY.md)
- [x] **Use Relative Paths in Migration Syntax Test:** Refactor `TestMigrationsValidSyntax` to avoid absolute paths.
  - **Action:** Modify the path construction logic in `cmd/server/migrations_test.go` (lines ~79-83) for `TestMigrationsValidSyntax`. Instead of constructing an absolute path based on `os.Getwd()`, use a relative path from the test file's location or determine the project root reliably. Consider using `filepath.Abs` on the relative path if an absolute path is still required by `goose.CollectMigrations`.
  - **Depends On:** None
  - **AC Ref:** Test Quality Issue 1

- [x] **Use filepath.Join in Local Postgres Test:** Refactor `TestLocalPostgresSetup` to use `filepath.Join`.
  - **Action:** Modify the path construction logic in `infrastructure/local_dev/local_postgres_test.go` (line ~23). Replace the hardcoded relative path concatenation for finding `docker-compose.yml` with `filepath.Join(".", "docker-compose.yml")` or similar to correctly refer to the file relative to the working directory.
  - **Depends On:** None
  - **AC Ref:** Test Quality Issue 2

- [x] **Enhance Terraform Test Validation:** Improve Terraform tests to verify database connectivity.
  - **Action:** Modify the `TestTerraformDatabaseInfrastructure` test in `infrastructure/terraform/test/terraform_test.go`. After `terraform.InitAndApply`, use the `connection_string` output to establish a database connection, perform a `Ping()` to verify connectivity, and optionally attempt to run a simple query or apply migrations.
  - **Depends On:** Add Explicit DB Password Management in Terraform
  - **AC Ref:** Test Quality Issue 3

## Documentation Practices (DOCUMENTATION_APPROACH.md)
- [x] **Add Godoc Comments to SRS Algorithm Functions:** Document core SRS calculation functions.
  - **Action:** Add comprehensive Godoc comments to the functions `calculateNewEaseFactor`, `calculateNewInterval`, `calculateNextReviewDate`, and `calculateNextStats` in `internal/domain/srs/algorithm.go`. Explain the purpose, parameters, return values, and any relevant algorithmic details for each function.
  - **Depends On:** None
  - **AC Ref:** Documentation Practices Issue 1

- [ ] **Document SRS Lapse Handling Multiplier:** Clarify the 'Good' outcome multiplier after a lapse in SRS design docs.
  - **Action:** Update the `docs/design/srs_algorithm.md` document. Add a specific point under "Lapse Handling" or within the interval calculation description explaining the use of the `1.5` multiplier for the "Good" outcome immediately following an "Again" outcome (lapse). Include the rationale for this specific value.
  - **Depends On:** Refactor UserCardStats Mutability
  - **AC Ref:** Documentation Practices Issue 2

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS
- [ ] **Issue/Assumption:** Acceptance Criteria References
  - **Context:** The `PLAN.md` (Code Review) does not have explicit AC IDs.
  - **Assumption:** The `AC Ref` fields in this `TODO.md` refer to the specific numbered issues within each section of the `PLAN.md` (Code Review) document (e.g., "Design Principles Issue 1", "Test Quality Issue 3").

- [ ] **Issue/Assumption:** Exit handling in slogGooseLogger.Fatalf
  - **Context:** PLAN.md Section Architectural Patterns 1 shows `slogGooseLogger.Fatalf` calling `os.Exit(1)`, but `main.go` already handles exits.
  - **Assumption:** The `slogGooseLogger.Fatalf` implementation should only log the error using `slog.Error` and not call `os.Exit(1)`. The `runMigrations` function will return errors to `main` which handles program exit consistently.
