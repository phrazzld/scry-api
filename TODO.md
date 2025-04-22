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

- [x] **T076 · Chore · P1: configure dependency injection and routes**
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

- [x] **T077 · Test · P2: add unit tests for PostgresCardStore.GetNextReviewCard**
    - **Context:** PLAN.md > Testing Strategy > Unit Tests
    - **Action:**
        1. Create unit tests for store implementation
        2. Cover: card found, no card due, database error scenarios
    - **Done-when:**
        1. Tests pass and cover specified scenarios
        2. Edge cases and error handling are tested
    - **Depends-on:** [T070]

- [x] **T078 · Test · P2: add unit tests for CardReviewService methods**
    - **Context:** PLAN.md > Testing Strategy > Unit Tests
    - **Action:**
        1. Create unit tests for `GetNextCard` and `SubmitAnswer` with mocked dependencies
        2. Cover happy paths and various error scenarios
    - **Done-when:**
        1. Tests pass using mocked dependencies
        2. Error handling and edge cases are covered
        3. Ownership verification is tested
    - **Depends-on:** [T072, T073]

- [x] **T079 · Test · P2: add unit tests for card review API handlers**
    - **Context:** PLAN.md > Testing Strategy > Unit Tests
    - **Action:**
        1. Create unit tests for both handler methods with mocked service
        2. Test input validation, response status codes, and response bodies
    - **Done-when:**
        1. Tests pass using mocked service
        2. Status code mappings are verified
        3. Input validation is tested
    - **Depends-on:** [T074, T075]

- [x] **T080 · Test · P2: add integration tests for card review API endpoints**
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
    - **Note:** After implementation review, we determined a better approach is to use API tests with mocked dependencies instead of database-dependent integration tests. See tasks T083-T088 for the new implementation approach.

    **Testing Approach Lessons Learned:**

    1. **Database Independence**:
       - API tests with mocked dependencies provide faster, more reliable test runs
       - Removes the need for DATABASE_URL to be set in test environments
       - Eliminates test flakiness caused by database state or connection issues
       - Allows CI to run tests without database setup

    2. **Improved Separation of Concerns**:
       - API tests focus solely on HTTP layer behavior (request/response, status codes, content)
       - Service and store layers are tested separately with dedicated unit tests
       - Clear boundaries make it easier to identify where issues occur

    3. **Enhanced Test Maintainability**:
       - Mock with functional options pattern provides flexible configuration
       - Test helpers with builder pattern make test setup more concise and readable
       - Table-driven tests make adding new scenarios straightforward
       - Consistent patterns across tests reduce cognitive load

    4. **Better Test Coverage**:
       - Mocks make it easier to test edge cases (errors, rare conditions)
       - Clearer verification of call tracking ensures correct component interaction
       - Fine-grained control over test conditions
       - Build tags (`test_without_external_deps`) allow conditional compilation

    5. **Developer Experience**:
       - Tests run faster (no database setup/teardown overhead)
       - Less environment setup required to run tests
       - Isolated failures make debugging easier
       - Clear test patterns are more approachable for new developers

- [x] **T081 · Chore · P2: ensure query performance with proper indexing**
    - **Context:** PLAN.md > Risk Matrix > Query performance issues
    - **Action:**
        1. Analyze the query for `GetNextReviewCard`
        2. Verify index exists on `user_card_stats` for `user_id` and `next_review_at`
        3. Create migration for index if needed
    - **Done-when:**
        1. Proper index exists or is added via migration
        2. Query performance is analyzed and acceptable
    - **Depends-on:** [T070]
    - **Note:** Analysis confirmed that appropriate indexes are already in place:
        - `idx_cards_user_id` on `cards(user_id)` for the cards table filter
        - `idx_stats_user_next_review_at` on `user_card_stats(user_id, next_review_at)` for both filtering and sorting on user_card_stats
        - No new migration needed as indexes were already included in the initial table creation

- [x] **T082 · Chore · P2: implement concurrency protection for stats updates**
    - **Context:** PLAN.md > Risk Matrix > Concurrency issues in stats updates
    - **Action:**
        1. Review transaction isolation in `SubmitAnswer`
        2. Implement `FOR UPDATE` locking if needed when fetching stats
    - **Done-when:**
        1. Concurrency issues are analyzed and addressed
        2. Appropriate locking mechanism is implemented if needed
    - **Depends-on:** [T073]
    - **Note:** Added `GetForUpdate` method with `SELECT FOR UPDATE` to the `UserCardStatsStore` interface
      and its implementation. Updated the `SubmitAnswer` method to use this new method, providing
      row-level locking to prevent concurrent modifications to the same stats record.

### Clarifications & Assumptions

- [x] **Issue:** Confirm UserCardStatsStore interface and implementation are complete
    - **Context:** PLAN.md dependencies for CardReviewService
    - **Blocking?:** yes (blocks T073)
    - **Status:** Confirmed, implementation exists in internal/store/stats.go

- [x] **Issue:** Confirm SRS service exists and is injectable
    - **Context:** PLAN.md dependencies for CardReviewService
    - **Blocking?:** yes (blocks T073)
    - **Status:** Confirmed, implementation exists in internal/domain/srs/service.go

- [ ] **Issue:** Decide if pagination is needed for GetNextReviewCard
    - **Context:** PLAN.md > Open Questions
    - **Blocking?:** no (can be added later)

- [x] **T083 · Chore · P1: Extract and improve CardReviewService mock**
    - **Context:** CONSULTANT-PLAN.md > Step 1: Create a Mock for CardReviewService
    - **Action:**
        1. Create a new file `internal/mocks/card_review_service.go`
        2. Extract and adapt the `mockCardReviewService` from `card_handler_test.go`
        3. Enhance the mock with better configurability for test cases
    - **Done-when:**
        1. Mock implementation exists in the mocks package
        2. Mock provides flexible configuration through functional options or similar
        3. Mock correctly implements the CardReviewService interface
    - **Depends-on:** None

- [x] **T084 · Test · P1: Implement API tests for GetNextReviewCard endpoint**
    - **Context:** CONSULTANT-PLAN.md > Step 2: Create API Test for Card Review Endpoints
    - **Action:**
        1. Create a new file `cmd/server/card_review_api_test.go`
        2. Implement test case for the GET /cards/next endpoint
        3. Use MockCardReviewService and MockJWTService for dependencies
        4. Test success case, no cards due case, unauthorized case
    - **Done-when:**
        1. Tests verify HTTP responses without database dependency
        2. Tests cover all expected status codes (200, 204, 401)
        3. Tests validate response body structure
    - **Depends-on:** [T083]

- [x] **T085 · Test · P1: Implement API tests for SubmitAnswer endpoint**
    - **Context:** CONSULTANT-PLAN.md > Step 2: Create API Test for Card Review Endpoints
    - **Action:**
        1. Add tests for POST /cards/{id}/answer endpoint to `card_review_api_test.go`
        2. Use MockCardReviewService and MockJWTService for dependencies
        3. Test success case, not found case, unauthorized case, invalid input case
    - **Done-when:**
        1. Tests verify HTTP responses without database dependency
        2. Tests cover all expected status codes (200, 400, 401, 403, 404)
        3. Tests validate response body structure
    - **Depends-on:** [T083]

- [x] **T086 · Chore · P2: Update testutils for API testing**
    - **Context:** CONSULTANT-PLAN.md > Implementation Plan
    - **Action:**
        1. Enhance `internal/testutils/api_helpers.go` with new helpers for card review API tests
        2. Add helper functions for creating test data (cards, user stats)
        3. Add helper functions for setting up test server with mocked dependencies
    - **Done-when:**
        1. Helpers make API test setup more concise and readable
        2. Common test patterns are extracted into reusable utilities
        3. Test data creation is consistent across tests
    - **Depends-on:** [T083]

- [x] **T087 · Chore · P2: Remove or refactor database-dependent integration tests**
    - **Context:** CONSULTANT-PLAN.md > Implementation Strategy
    - **Action:**
        1. Decide whether to completely remove `card_review_integration_test.go` or keep with build tags
        2. If keeping, update it to be explicit about its database dependency
        3. If removing, ensure all test coverage is maintained in API tests
    - **Done-when:**
        1. No longer have test files that require DATABASE_URL to be set
        2. If integration tests are kept, they're clearly separated from API tests
        3. Test coverage reports show equivalent or better coverage
    - **Depends-on:** [T084, T085]

- [x] **T088 · Chore · P2: Update T080 with lessons learned**
    - **Context:** Original task is now marked completed but we want to document our learning
    - **Action:**
        1. Add comprehensive note to T080 about the testing approach change
        2. Document key lessons: database-independence, mocking services, separation of concerns
    - **Done-when:**
        1. Clear explanation of why we changed approach is documented
        2. Future developers can learn from this experience
    - **Depends-on:** [T084, T085, T086, T087]
