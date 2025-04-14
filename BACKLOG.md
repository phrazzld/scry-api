# Scry API - MVP Backlog

**Repository:** `scry-api` (Go Backend)
**Version:** 1.0-MVP
**Date:** 2025-04-13

This backlog outlines the major work items required to build the Minimum Viable Product (MVP) for the Scry Go backend API, hosted entirely on DigitalOcean. Items should generally be tackled in order, aligning with our core principles and engineering guidelines. Each item represents a meaningful chunk of work, intended to be broken down further into detailed plans and tasks.

* **1. CI/CD Setup:**
    * Set up GitHub Actions for continuous integration.
    * Configure workflows for:
      * Building and testing the code on every push and PR.
      * Linting code with golangci-lint.
      * Running security scanning.
      * Automating deployment to DigitalOcean on main branch merges.
    * Implement pre-commit hooks (using pre-commit framework) to ensure code quality:
      * gofmt/goimports for consistent formatting.
      * golangci-lint for catching issues early.
      * Commit message validation for conventional commits.
      * Potentially run fast tests pre-commit.

* **2. Logging Framework Setup:**
    * Set up basic structured logging framework (e.g., `log/slog`, `zerolog`, `zap`).
    * Configure the logging system using the application configuration.
    * Implement appropriate log levels and contextual logging helpers.

* **3. Database Setup & Migrations:**
    * Provision DigitalOcean Managed PostgreSQL instance.
    * Enable `pgvector` extension on the DO Managed Postgres instance.
    * Set up database migration tooling (e.g., `golang-migrate`).
    * Define core domain models/structs in Go (`internal/domain`: `User`, `Memo`, `Card`, `UserCardStats`) adhering to type standards (`CODING_STANDARDS.md` Section 2).
    * Create initial database schema migration script defining `users`, `memos` (including `status` field: 'pending', 'processing', 'completed', 'completed_with_errors', 'failed'), `cards` (with `content` JSONB structure), `user_card_stats` tables, columns, constraints, and essential indexes (esp. on `user_card_stats` for `next_review_at`).

* **4. Core Domain Logic Implementation (SRS):**
    * Define `srs.Service` interface within the core domain/application layer.
    * Implement basic SRS algorithm logic (SM-2 variant) within the `srs` service. *Action Required: Define precise MVP parameters (initial intervals, ease factors, lapse handling) in a separate design doc before implementation.* Adhere to pure function principles where possible (`CODING_STANDARDS.md` Section 4).

* **5. Authentication Implementation:**
    * Implement `store.UserStore` interface and PostgreSQL implementation (`internal/platform/postgres`) for user CRUD, including secure password hashing (`bcrypt`).
    * Implement JWT generation logic within an `auth.Service`.
    * Implement User Registration endpoint (`POST /auth/register`) in `internal/api`, utilizing `auth.Service` and `store.UserStore`.
    * Implement User Login endpoint (`POST /auth/login`).
    * Implement Authentication Middleware (JWT validation) for protecting relevant API routes.

* **6. Asynchronous Task Runner Setup:**
    * Implement basic in-memory background task queue & worker pool (`internal/task`) using goroutines/channels.
    * Implement recovery mechanism: On application startup, query `memos` table for entries with `status = 'processing'`, enqueue generation tasks for these Memos to handle potential restarts during processing. Define clear locking or timestamp logic if needed to prevent duplicate processing in multi-instance scenarios (though MVP likely single instance).

* **7. Generation Service Implementation (`llm` -> `generation`):**
    * Define `generation.Generator` interface (e.g., `GenerateCardsFromMemo(...)`) within the core application layer.
    * Implement `geminiGenerator` struct implementing the `Generator` interface (`internal/platform/gemini` or similar).
        * Load prompt templates from external configuration (not hardcoded).
        * Implement logic to call Gemini API using the configured model(s).
        * Implement error handling & basic retry logic for transient Gemini API errors.
        * Securely load and manage the Gemini API key via configuration.
    * Design service to be swappable per `ARCHITECTURE_GUIDELINES.md` Section 3 (Dependency Inversion).

* **8. Memo & Card Generation Implementation:**
    * Implement `store.MemoStore` and `store.CardStore` interfaces and Postgres implementations for Memo/Card/Stats persistence and status updates.
    * Implement Background Job logic (`internal/task/processor.go` or similar): `GenerateCardsFromMemo(memoID, userID, memoText)`:
        * Update Memo status to `processing` in DB.
        * Call `generation.Generator.GenerateCardsFromMemo`.
        * Parse/validate the returned Card data structures.
        * Attempt to save the batch of generated Cards and their initial `user_card_stats` (with `next_review_at = NOW()`) to the DB, ideally within a transaction if feasible for the batch.
        * **Error Handling:**
            * If saving an individual Card/Stat fails, retry that save operation (e.g., 2-3 times). Log error on each failure.
            * If *some* cards save successfully after retries but others fail permanently, log failed card details clearly. Update Memo status to `completed_with_errors`. Log failure ratio. **Do not** retry the LLM call.
            * If *all* card saves fail after retries, or the LLM call fails irrecoverably, log the error extensively. Update Memo status to `failed`. Trigger critical alert.
            * On full success, update Memo status to `completed`.
    * Implement Submit Memo endpoint (`POST /memos`): Authenticates user, saves Memo via `store` with `pending` status, enqueues `GenerateCardsFromMemo` job via `task` service, returns HTTP 202 Accepted.

* **9. Card Review API Implementation:**
    * Implement `store.CardStore` function `GetNextReviewCard(userID time.Time)` using the defined query logic (filtering by `next_review_at`, ordering).
    * Implement Fetch Next Card endpoint (`GET /cards/next`), using the store function and handling the 204 No Content case.
    * Implement `store.UserCardStatsStore` function `UpdateStats(userID, cardID, outcome)`.
    * Implement Submit Answer endpoint (`POST /cards/{id}/answer`), validating the outcome, calling the `srs.Service` to calculate new stats, and updating the DB via the store.

* **10. Card Management API Implementation:**
    * Implement `store.CardStore` function for updating Card content (`content` JSONB).
    * Implement Edit Card endpoint (`PUT /cards/{id}`).
    * Implement `store.CardStore` function for deleting a Card and its associated `user_card_stats`.
    * Implement Delete Card endpoint (`DELETE /cards/{id}`).
    * Implement Postpone Card endpoint (`POST /cards/{id}/postpone`), calculating and updating `next_review_at` via the store.

* **11. API Structure & Server Setup:**
    * Set up main HTTP server entry point (`cmd/server/main.go`).
    * Implement request routing using `chi` router.
    * Integrate standard middleware: Logging (request logging), Recovery (panic recovery), Auth (JWT validation), CORS.
    * Implement basic request body validation.

* **12. Testing:**
    * Adhere strictly to `TESTING_STRATEGY.md`.
    * Implement unit tests for `srs` service, utility functions, domain logic.
    * Implement integration tests for critical API flows (Auth register/login, Memo submission & generation trigger, Card review cycle: get next -> answer -> get next). Mock only at external boundaries (define `Generator` interface mock, potentially `Store` interface mock *if not* using test containers). Consider `testcontainers-go` for PostgreSQL integration tests.

* **13. Deployment & Infrastructure (DigitalOcean):**
    * Create comprehensive `Dockerfile` for the Go application.
    * Configure deployment via DigitalOcean App Platform, including environment variables (DB connection string, JWT secret, Gemini API key, log level, etc.).
    * Ensure DO Managed Postgres is configured correctly (version, extensions).

* **14. Monitoring & Logging (Basic):**
    * Implement structured logging (JSON) throughout the application, including contextual information (user ID, request ID, memo ID, card ID).
    * Ensure critical errors (panic recovery, background job failures, failed LLM calls, DB errors) are logged at ERROR/FATAL level.
    * Set up basic alerting via DO App Platform monitoring or external service for critical error logs.
    * Define a basic health check endpoint (`/healthz`).

## Completed Items

* **Project Setup & Configuration (Completed):**
    * ✅ Initialize Go module (`scry-api`) and standard project structure (e.g., `/cmd/server`, `/internal/domain`, `/internal/service`, `/internal/store`, `/internal/api`, `/internal/generation`, `/internal/task`, `/internal/config`, `/internal/platform/postgres`).

* **Configuration Management Implementation (Completed):**
    * ✅ Implement configuration loading (env vars primary, config files for local dev via Viper) adhering to `ARCHITECTURE_GUIDELINES.md` Section 6.
    * ✅ Create a strongly-typed configuration structure containing all application settings.
    * ✅ Implement validation logic to ensure all required configuration values are present and valid.