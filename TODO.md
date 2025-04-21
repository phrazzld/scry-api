# Todo

## Card Review API

- [x] **T069 · Feature · P1: define GetNextReviewCard in CardStore interface**
    - **Context:** PLAN.md > Detailed Build Steps > 1; Public Interfaces
    - **Action:**
        1. Add `GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)` to `CardStore` interface
        2. Add godoc explaining it returns the next due card or `store.ErrCardNotFound`
    - **Done-when:**
        1. Interface method signature added to `internal/store/card.go`
        2. Godoc comment added explaining behavior and errors
    - **Depends-on:** none

- [x] **T070 · Feature · P1: implement GetNextReviewCard in PostgresCardStore**
    - **Context:** PLAN.md > Detailed Build Steps > 2; Error & Edge-Case Strategy
    - **Action:**
        1. Implement `GetNextReviewCard` in `internal/platform/postgres/card_store.go`
        2. Write SQL query joining `cards` and `user_card_stats` filtering by `user_id` and `next_review_at <= NOW()`
        3. Map `sql.ErrNoRows` to `store.ErrCardNotFound`; wrap other DB errors
    - **Done-when:**
        1. Method returns the correct card based on `next_review_at`
        2. Returns `store.ErrCardNotFound` when no cards are due
        3. Database errors are mapped and logged appropriately
    - **Depends-on:** [T069]

- [x] **T071 · Feature · P1: create CardReviewService interface**
    - **Context:** PLAN.md > Detailed Build Steps > 3; Public Interfaces
    - **Action:**
        1. Create `internal/service/card_review/service.go`
        2. Define `CardReviewService` interface with `GetNextCard` and `SubmitAnswer` methods
        3. Add godoc explaining method behavior, parameters, and potential errors
    - **Done-when:**
        1. Interface matches the plan specification
        2. Comprehensive godoc comments document behavior and error cases
    - **Depends-on:** none

- [x] **T072 · Feature · P1: implement GetNextCard in CardReviewService**
    - **Context:** PLAN.md > Detailed Build Steps > 4; Data Flow Diagram (GET)
    - **Action:**
        1. Create `cardReviewServiceImpl` struct with `CardStore` and `Logger` dependencies
        2. Implement the `GetNextCard` method that calls `CardStore.GetNextReviewCard`
        3. Add proper error handling and logging with context propagation
    - **Done-when:**
        1. Method correctly calls store layer and returns appropriate results
        2. Error handling preserves original error types (e.g., `store.ErrCardNotFound`)
        3. Logging includes structured fields and correlation ID
    - **Depends-on:** [T069, T070, T071]

- [x] **T073 · Feature · P1: implement SubmitAnswer in CardReviewService**
    - **Context:** PLAN.md > Detailed Build Steps > 4; Data Flow Diagram (POST)
    - **Action:**
        1. Add `UserCardStatsStore` and `srs.Service` dependencies to `cardReviewServiceImpl`
        2. Implement `SubmitAnswer` with transaction handling via `store.RunInTransaction`
        3. Verify card ownership, calculate new stats with SRS service, update stats in DB
    - **Done-when:**
        1. Operations are wrapped in a single transaction
        2. Card ownership is verified against authenticated user
        3. SRS calculation is performed and stats are updated
        4. Error handling and logging are comprehensive
    - **Depends-on:** [T071]

- [x] **T074 · Feature · P1: implement GET /cards/next handler**
    - **Context:** PLAN.md > Detailed Build Steps > 5; Error & Edge-Case Strategy
    - **Action:**
        1. Create `CardHandler` struct in `internal/api/card_handler.go`
        2. Implement `GetNextReviewCard` handler for GET /cards/next endpoint
        3. Map service responses to appropriate HTTP status codes (200, 204, 500)
    - **Done-when:**
        1. Handler extracts user ID from context
        2. Handler calls service layer correctly
        3. Returns HTTP 200 with card on success
        4. Returns HTTP 204 when no cards are due
        5. Returns HTTP 500 on other errors
    - **Depends-on:** [T072]

- [x] **T075 · Feature · P1: implement POST /cards/{id}/answer handler**
    - **Context:** PLAN.md > Detailed Build Steps > 5; Input validation hotspots
    - **Action:**
        1. Implement `SubmitAnswer` handler in `CardHandler`
        2. Validate UUID format and request body (outcome field)
        3. Map service responses to appropriate HTTP status codes (200, 400, 404, 500)
    - **Done-when:**
        1. Handler validates card ID and request body
        2. Handler extracts user ID from context
        3. Returns appropriate HTTP status codes for different scenarios
        4. Returns updated stats on success
    - **Depends-on:** [T073]

- [ ] **T076 · Chore · P1: configure dependency injection and routes**
    - **Context:** PLAN.md > Detailed Build Steps > 6
    - **Action:**
        1. In `cmd/server/main.go`, instantiate `CardReviewService` with dependencies
        2. Instantiate `CardHandler` with the service
        3. Register API routes with authentication middleware
    - **Done-when:**
        1. Service and handler are properly instantiated
        2. Routes `/cards/next` and `/cards/{id}/answer` are registered
        3. Routes are protected by authentication middleware
    - **Depends-on:** [T074, T075]

- [ ] **T077 · Test · P2: add unit tests for PostgresCardStore.GetNextReviewCard**
    - **Context:** PLAN.md > Testing Strategy > Unit Tests
    - **Action:**
        1. Create unit tests for store implementation
        2. Cover: card found, no card due, database error scenarios
    - **Done-when:**
        1. Tests pass and cover specified scenarios
        2. Edge cases and error handling are tested
    - **Depends-on:** [T070]

- [ ] **T078 · Test · P2: add unit tests for CardReviewService methods**
    - **Context:** PLAN.md > Testing Strategy > Unit Tests
    - **Action:**
        1. Create unit tests for `GetNextCard` and `SubmitAnswer` with mocked dependencies
        2. Cover happy paths and various error scenarios
    - **Done-when:**
        1. Tests pass using mocked dependencies
        2. Error handling and edge cases are covered
        3. Ownership verification is tested
    - **Depends-on:** [T072, T073]

- [ ] **T079 · Test · P2: add unit tests for card review API handlers**
    - **Context:** PLAN.md > Testing Strategy > Unit Tests
    - **Action:**
        1. Create unit tests for both handler methods with mocked service
        2. Test input validation, response status codes, and response bodies
    - **Done-when:**
        1. Tests pass using mocked service
        2. Status code mappings are verified
        3. Input validation is tested
    - **Depends-on:** [T074, T075]

- [ ] **T080 · Test · P2: add integration tests for card review API endpoints**
    - **Context:** PLAN.md > Testing Strategy > Integration Tests
    - **Action:**
        1. Set up integration test environment with test database
        2. Test both endpoints with various scenarios
        3. Verify database state changes after operations
    - **Done-when:**
        1. Integration tests pass against real database
        2. HTTP status codes and response bodies are verified
        3. Database state changes are verified after operations
    - **Depends-on:** [T076]

- [ ] **T081 · Chore · P2: ensure query performance with proper indexing**
    - **Context:** PLAN.md > Risk Matrix > Query performance issues
    - **Action:**
        1. Analyze the query for `GetNextReviewCard`
        2. Verify index exists on `user_card_stats` for `user_id` and `next_review_at`
        3. Create migration for index if needed
    - **Done-when:**
        1. Proper index exists or is added via migration
        2. Query performance is analyzed and acceptable
    - **Depends-on:** [T070]

- [ ] **T082 · Chore · P2: implement concurrency protection for stats updates**
    - **Context:** PLAN.md > Risk Matrix > Concurrency issues in stats updates
    - **Action:**
        1. Review transaction isolation in `SubmitAnswer`
        2. Implement `FOR UPDATE` locking if needed when fetching stats
    - **Done-when:**
        1. Concurrency issues are analyzed and addressed
        2. Appropriate locking mechanism is implemented if needed
    - **Depends-on:** [T073]

### Clarifications & Assumptions

- [ ] **Issue:** Confirm UserCardStatsStore interface and implementation are complete
    - **Context:** PLAN.md dependencies for CardReviewService
    - **Blocking?:** yes (blocks T073)

- [ ] **Issue:** Confirm SRS service exists and is injectable
    - **Context:** PLAN.md dependencies for CardReviewService
    - **Blocking?:** yes (blocks T073)

- [ ] **Issue:** Decide if pagination is needed for GetNextReviewCard
    - **Context:** PLAN.md > Open Questions
    - **Blocking?:** no (can be added later)
