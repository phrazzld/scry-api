# todo

## store interfaces
- [x] **t101 · feature · p1: define MemoStore interface**
    - **context:** detailed build steps #1, architecture blueprint
    - **action:**
        1. Define `MemoStore` interface in `internal/store/memo.go` with methods: `Create`, `GetByID`, `UpdateStatus`, `FindMemosByStatus`
        2. Add godoc comments explaining methods, parameters, returns, and error conditions
    - **done‑when:**
        1. Interface defined and documented with clear error contracts
        2. Code compiles
    - **depends‑on:** none

- [x] **t102 · feature · p1: define CardStore interface**
    - **context:** detailed build steps #1, architecture blueprint
    - **action:**
        1. Define `CardStore` interface in `internal/store/card.go` with methods: `CreateMultiple`, `GetByID`, `UpdateContent`, `Delete`, `GetNextReviewCard`
        2. Add godoc comments explaining methods, parameters, returns, and error conditions
    - **done‑when:**
        1. Interface defined and documented with clear error contracts
        2. Code compiles
    - **depends‑on:** none

- [x] **t103 · feature · p1: define UserCardStatsStore interface**
    - **context:** detailed build steps #1, architecture blueprint
    - **action:**
        1. Define `UserCardStatsStore` interface in `internal/store/stats.go` with methods: `Get`, `Update`, `Delete`
        2. Add godoc comments explaining methods, parameters, returns, and error conditions
    - **done‑when:**
        1. Interface defined and documented with clear error contracts
        2. Code compiles
    - **depends‑on:** none

- [x] **t104 · feature · p1: define common store errors**
    - **context:** error & edge-case strategy
    - **action:**
        1. Define common errors (`ErrNotFound`, `ErrDuplicate`, `ErrNotImplemented`) in `internal/store/errors.go`
        2. Add godoc comments explaining when each error occurs
    - **done‑when:**
        1. Common errors defined and documented
        2. Code compiles
    - **depends‑on:** none

## postgres memo store
- [x] **t105 · feature · p1: implement PostgresMemoStore struct and constructor**
    - **context:** detailed build steps #2, architecture blueprint
    - **action:**
        1. Create `PostgresMemoStore` struct with `db store.DBTX` and `logger *slog.Logger`
        2. Implement `NewPostgresMemoStore` constructor
    - **done‑when:**
        1. Struct and constructor implemented and documented
        2. Code compiles
    - **depends‑on:** [t101, t104]

- [ ] **t106 · feature · p2: implement MemoStore Create method**
    - **context:** detailed build steps #2
    - **action:**
        1. Implement `Create` using parameterized SQL INSERT
        2. Validate memo before DB operation
        3. Add structured logging
    - **done‑when:**
        1. Method creates memos in database
        2. Uses parameterized queries for security
        3. Logs success/failure appropriately
    - **depends‑on:** [t105]

- [ ] **t107 · feature · p2: implement MemoStore GetByID method**
    - **context:** detailed build steps #2
    - **action:**
        1. Implement `GetByID` using parameterized SQL SELECT
        2. Map `sql.ErrNoRows` to `store.ErrNotFound`
        3. Add structured logging
    - **done‑when:**
        1. Method retrieves memos or returns ErrNotFound
        2. Uses parameterized queries for security
        3. Logs success/failure appropriately
    - **depends‑on:** [t105]

- [ ] **t108 · feature · p2: implement MemoStore UpdateStatus method**
    - **context:** detailed build steps #2
    - **action:**
        1. Implement `UpdateStatus` using parameterized SQL UPDATE
        2. Check rows affected for not found case
        3. Add structured logging
    - **done‑when:**
        1. Method updates status or returns ErrNotFound
        2. Uses parameterized queries for security
        3. Logs success/failure appropriately
    - **depends‑on:** [t105]

- [ ] **t109 · feature · p2: implement MemoStore FindMemosByStatus method**
    - **context:** detailed build steps #2
    - **action:**
        1. Implement `FindMemosByStatus` using parameterized SQL SELECT
        2. Return empty slice (not nil) if no results
        3. Add structured logging
    - **done‑when:**
        1. Method retrieves memos matching status
        2. Uses parameterized queries for security
        3. Logs success/failure appropriately
    - **depends‑on:** [t105]

## postgres card store
- [x] **t110 · feature · p1: implement PostgresCardStore struct and constructor**
    - **context:** detailed build steps #3, architecture blueprint
    - **action:**
        1. Create `PostgresCardStore` struct with `db store.DBTX` and `logger *slog.Logger`
        2. Implement `NewPostgresCardStore` constructor
    - **done‑when:**
        1. Struct and constructor implemented and documented
        2. Code compiles
    - **depends‑on:** [t102, t104]

- [ ] **t111 · feature · p1: implement CardStore CreateMultiple method**
    - **context:** detailed build steps #3, error & edge-case strategy
    - **action:**
        1. Implement batch insert for cards and stats within single transaction
        2. Use efficient approach (e.g., pgx.CopyFrom) for batch operations
        3. Ensure atomicity with explicit rollback on errors
    - **done‑when:**
        1. Method creates multiple cards and stats in transaction
        2. All operations succeed or all fail (atomic)
        3. Logs success/failure appropriately
    - **depends‑on:** [t110]

- [ ] **t112 · feature · p2: implement CardStore GetByID method**
    - **context:** detailed build steps #3
    - **action:**
        1. Implement `GetByID` using parameterized SQL SELECT
        2. Handle JSONB content field properly
        3. Map `sql.ErrNoRows` to `store.ErrNotFound`
    - **done‑when:**
        1. Method retrieves cards or returns ErrNotFound
        2. JSONB content is correctly mapped
        3. Logs success/failure appropriately
    - **depends‑on:** [t110]

- [ ] **t113 · feature · p2: implement CardStore UpdateContent method**
    - **context:** detailed build steps #3
    - **action:**
        1. Implement `UpdateContent` using parameterized SQL UPDATE
        2. Validate JSON content
        3. Check rows affected for not found case
    - **done‑when:**
        1. Method updates card content or returns ErrNotFound
        2. JSON content is validated
        3. Logs success/failure appropriately
    - **depends‑on:** [t110]

- [ ] **t114 · feature · p2: implement CardStore Delete method**
    - **context:** detailed build steps #3
    - **action:**
        1. Implement `Delete` using parameterized SQL DELETE
        2. Check rows affected for not found case
        3. Rely on cascade delete for associated stats
    - **done‑when:**
        1. Method deletes card or returns ErrNotFound
        2. Associated stats are also deleted (cascade)
        3. Logs success/failure appropriately
    - **depends‑on:** [t110]

- [ ] **t115 · feature · p3: implement CardStore GetNextReviewCard stub**
    - **context:** detailed build steps #3
    - **action:**
        1. Implement stub that returns `nil, store.ErrNotImplemented`
    - **done‑when:**
        1. Method exists and returns ErrNotImplemented
    - **depends‑on:** [t110]

## postgres user card stats store
- [ ] **t116 · feature · p1: implement PostgresUserCardStatsStore struct and constructor**
    - **context:** detailed build steps #4, architecture blueprint
    - **action:**
        1. Create `PostgresUserCardStatsStore` struct with `db store.DBTX` and `logger *slog.Logger`
        2. Implement `NewPostgresUserCardStatsStore` constructor
    - **done‑when:**
        1. Struct and constructor implemented and documented
        2. Code compiles
    - **depends‑on:** [t103, t104]

- [ ] **t117 · feature · p2: implement UserCardStatsStore Get method**
    - **context:** detailed build steps #4
    - **action:**
        1. Implement `Get` using parameterized SQL SELECT
        2. Map `sql.ErrNoRows` to `store.ErrNotFound`
    - **done‑when:**
        1. Method retrieves stats or returns ErrNotFound
        2. Uses parameterized queries for security
        3. Logs success/failure appropriately
    - **depends‑on:** [t116]

- [ ] **t118 · feature · p2: implement UserCardStatsStore Update method**
    - **context:** detailed build steps #4
    - **action:**
        1. Implement `Update` using parameterized SQL UPDATE
        2. Validate stats before updating
        3. Check rows affected for not found case
    - **done‑when:**
        1. Method updates stats or returns ErrNotFound
        2. Uses parameterized queries for security
        3. Logs success/failure appropriately
    - **depends‑on:** [t116]

- [ ] **t119 · feature · p2: implement UserCardStatsStore Delete method**
    - **context:** detailed build steps #4
    - **action:**
        1. Implement `Delete` using parameterized SQL DELETE
        2. Check rows affected for not found case
    - **done‑when:**
        1. Method deletes stats or returns ErrNotFound
        2. Uses parameterized queries for security
        3. Logs success/failure appropriately
    - **depends‑on:** [t116]

## testing
- [ ] **t120 · test · p2: implement test utilities for transaction isolation**
    - **context:** testing strategy
    - **action:**
        1. Implement or verify `testutils.WithTx` helper for transaction-isolated tests
        2. Set up test database connection handling
    - **done‑when:**
        1. Helper is available for integration tests
        2. Tests can run in isolated transactions
    - **depends‑on:** none

- [ ] **t121 · test · p2: write integration tests for MemoStore**
    - **context:** testing strategy, build step #7
    - **action:**
        1. Create `memo_store_test.go` with tests for all methods
        2. Cover success, not found, error cases
        3. Use transaction isolation via `testutils.WithTx`
    - **done‑when:**
        1. All memo store methods have test coverage
        2. Tests pass against real database
    - **depends‑on:** [t106, t107, t108, t109, t120]

- [ ] **t122 · test · p2: write integration tests for CardStore**
    - **context:** testing strategy, build step #7
    - **action:**
        1. Create `card_store_test.go` with tests for all methods
        2. Cover success, not found, error cases, transaction atomicity
        3. Use transaction isolation via `testutils.WithTx`
    - **done‑when:**
        1. All card store methods have test coverage
        2. Tests pass against real database
    - **depends‑on:** [t111, t112, t113, t114, t115, t120]

- [ ] **t123 · test · p2: write integration tests for UserCardStatsStore**
    - **context:** testing strategy, build step #7
    - **action:**
        1. Create `user_card_stats_store_test.go` with tests for all methods
        2. Cover success, not found, error cases
        3. Use transaction isolation via `testutils.WithTx`
    - **done‑when:**
        1. All user card stats store methods have test coverage
        2. Tests pass against real database
    - **depends‑on:** [t117, t118, t119, t120]

## cross-cutting concerns
- [ ] **t124 · feature · p1: implement common error mapping utilities**
    - **context:** error & edge-case strategy, build step #5
    - **action:**
        1. Create helpers in `errors.go` to map SQL/Postgres errors to domain errors
        2. Handle common cases: not found, unique constraint violations
    - **done‑when:**
        1. Error mapping is centralized and reused across stores
        2. Tests cover mapping behavior
    - **depends‑on:** [t104]

- [ ] **t125 · chore · p2: verify database indexes for performance**
    - **context:** risk matrix (Missing indexes)
    - **action:**
        1. Review schema for appropriate indexes on IDs, foreign keys, status fields
        2. Create migration for any missing indexes if needed
    - **done‑when:**
        1. Required indexes are confirmed to exist
    - **depends‑on:** none

- [ ] **t126 · chore · p2: update main.go for dependency injection**
    - **context:** build step #9
    - **action:**
        1. Update application setup code to wire in new store implementations
        2. Inject DB connection pool and logger
    - **done‑when:**
        1. Application compiles with new store implementations
    - **depends‑on:** [t105, t110, t116]

- [ ] **t127 · chore · p3: update store-related documentation**
    - **context:** build step #8, documentation
    - **action:**
        1. Create/update README.md files for store interfaces and implementations
        2. Document usage patterns and error handling
    - **done‑when:**
        1. README files exist and contain accurate information
    - **depends‑on:** [t101, t102, t103, t105, t110, t116]

### clarifications & assumptions
- [ ] **issue:** clarify if CreateMultiple should handle both cards and stats atomically
    - **context:** open question
    - **blocking?:** yes (for t111)

- [ ] **issue:** determine if additional reporting/filtering functionality is needed
    - **context:** open question
    - **blocking?:** no

- [ ] **issue:** decide whether optimistic concurrency handling is required
    - **context:** open question
    - **blocking?:** no
