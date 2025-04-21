# Plan: Card Review API Implementation

## Chosen Approach (One‑liner)
Implement dedicated CardReviewService to orchestrate SRS logic and data operations with clear separation of concerns, explicit transaction handling, and comprehensive error mapping.

## Architecture Blueprint
- **Modules / Packages**
  - `internal/domain`: Core entities and SRS domain logic (existing)
  - `internal/store`: Store interfaces - `CardStore`, `UserCardStatsStore` (require additions)
  - `internal/platform/postgres`: PostgreSQL implementations (require additions)
  - `internal/service`: Add `CardReviewService` for orchestration
  - `internal/api`: HTTP handlers for review endpoints
  - `cmd/server`: Main application setup and dependency injection

- **Public Interfaces / Contracts**
  - `store.CardStore` interface update:
    ```go
    type CardStore interface {
        // ... existing methods ...
        GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
        WithTxCardStore(tx *sql.Tx) CardStore // Ensure consistent tx pattern
    }
    ```
  - `store.UserCardStatsStore` (existing methods):
    ```go
    type UserCardStatsStore interface {
        // ... existing methods ...
        Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)
        Update(ctx context.Context, stats *domain.UserCardStats) error
        WithTx(tx *sql.Tx) UserCardStatsStore // For transactional use
    }
    ```
  - New `service.CardReviewService`:
    ```go
    type CardReviewService interface {
        // GetNextCard retrieves the next card due for review
        // Returns store.ErrNotFound if no cards are due
        GetNextCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error)

        // SubmitAnswer processes a user's answer and updates the SRS stats
        // Returns errors for not found, validation failures, or DB issues
        SubmitAnswer(ctx context.Context, userID uuid.UUID, cardID uuid.UUID,
                    outcome domain.ReviewOutcome) (*domain.UserCardStats, error)
    }
    ```
  - API Request/Response types:
    ```go
    // SubmitAnswerRequest defines the payload for submitting a card answer
    type SubmitAnswerRequest struct {
        Outcome string `json:"outcome" validate:"required,oneof=again hard good easy"`
    }

    // Card/stats responses can use domain types directly initially
    // More tailored DTOs can be added later if needed
    ```

- **Data Flow Diagram**
  ```
  GET /cards/next:
  Client → API Handler → CardReviewService → CardStore → Database
                                          ↑
                                          └ Returns card or ErrNotFound

  POST /cards/{id}/answer:
  Client → API Handler → CardReviewService → Begin Transaction
                                          ↓
                             Get stats/card → SRS calculation → Update stats
                                          ↑
                                          └ Commit/Rollback → Return result
  ```

- **Error & Edge‑Case Strategy**
  - `GetNextReviewCard`:
    - No cards due: Return `store.ErrCardNotFound` → HTTP 204 No Content
    - DB error: Return mapped error → HTTP 500 Internal Server Error
  - `SubmitAnswer`:
    - Invalid cardID format or invalid outcome: HTTP 400 Bad Request
    - Card/stats not found or mismatch: HTTP 404 Not Found
    - SRS calculation error: HTTP 500 Internal Server Error
    - DB update error: HTTP 500 Internal Server Error
  - Use standardized error types from `store` package
  - Always wrap errors with contextual information using `fmt.Errorf("%w", err)`

## Detailed Build Steps
1. Define `GetNextReviewCard` method in `store.CardStore` interface (internal/store/card.go)
   - Add method signature to interface
   - Add detailed documentation about expected behavior

2. Implement `GetNextReviewCard` in `PostgresCardStore` (internal/platform/postgres/card_store.go)
   - Write SQL query to join cards with user_card_stats and filter by next_review_at
   - Map sql.ErrNoRows to store.ErrCardNotFound
   - Add proper logging and error mapping for database errors

3. Create `CardReviewService` interface in `internal/service/card_review_service.go`
   - Define methods as outlined in architecture blueprint
   - Add comprehensive documentation about behavior and error cases

4. Implement `cardReviewServiceImpl` in `internal/service/card_review_service.go`
   - Create struct with dependencies: CardStore, UserCardStatsStore, SRS Service, Logger
   - Implement `GetNextCard` method
   - Implement `SubmitAnswer` method with transaction handling via store.RunInTransaction
   - Add proper error handling, logging, and correlation ID propagation

5. Create `CardHandler` in `internal/api/card_handler.go`
   - Implement `GetNextReviewCard` handler for GET /cards/next endpoint
   - Implement `SubmitAnswer` handler for POST /cards/{id}/answer endpoint
   - Add input validation for request body and URL parameters
   - Implement proper error mapping to HTTP status codes

6. Update dependency injection in `cmd/server/main.go`
   - Instantiate CardReviewService
   - Instantiate CardHandler
   - Register routes in router

7. Write unit tests for new components
   - Test `PostgresCardStore.GetNextReviewCard` with different scenarios
   - Test `CardReviewService` methods with mocked dependencies
   - Test `CardHandler` with mocked service and http test utilities

8. Write integration tests for API endpoints
   - Test full flow for GET /cards/next with various scenarios
   - Test full flow for POST /cards/{id}/answer with various scenarios
   - Verify database state changes after successful operations

## Testing Strategy
- **Unit Tests:**
  - `PostgresCardStore.GetNextReviewCard`: Test SQL query logic
  - `CardReviewService`: Test orchestration logic and error handling
    - Mock `CardStore`, `UserCardStatsStore`, `srs.Service`
    - Test various scenarios including happy path and error cases
  - `CardHandler`: Test handler logic, input validation, and response formatting
    - Mock `CardReviewService`
    - Test HTTP status code mapping for different error types

- **Integration Tests:**
  - `PostgresCardStore.GetNextReviewCard`: Test with real database
    - Use transaction isolation via testutils.WithTx
    - Set up test data with various next_review_at values
  - API endpoints: Test end-to-end with HTTP requests
    - Test error cases and edge conditions
    - Verify database state changes after successful operations

- **What to mock:**
  - In unit tests: Mock dependencies at the same layer or external layers
  - In integration tests: Use real implementations with test database

- **Coverage targets:**
  - Core logic (service layer): 90%+ coverage
  - Edge cases and error handling: 80%+ coverage
  - Focus on the review flow to ensure it works correctly

## Logging & Observability
- **Log events:**
  - CardStore:
    - Start/end of `GetNextReviewCard` with userID
    - Card found or not found status
    - Database errors with context
  - CardReviewService:
    - Start/end of service method calls
    - Transaction boundaries (start, commit, rollback)
    - Error conditions with context
  - CardHandler:
    - Request details (method, path, userID)
    - Response status and errors

- **Structured fields per action:**
  - Common: correlation_id, request_id, user_id
  - Card-specific: card_id, memo_id
  - Stats-specific: interval, ease_factor, next_review_at, review_count
  - Errors: error message, error type

- **Correlation ID propagation:**
  - Use existing pattern of context.Context to propagate correlation ID
  - Include correlation ID in all log entries
  - Pass context through all layers of call stack

## Security & Config
- **Input validation hotspots:**
  - `cardID` in URL path (validate UUID format)
  - `outcome` in request body (validate allowed values)
  - Use validator tags for request struct validation

- **Ownership check:**
  - Verify that requested card/stats belong to authenticated user
  - Return 404 Not Found (not 403 Forbidden) for non-owned resources to avoid leaking existence

- **Authorization:**
  - Use existing auth middleware to ensure authenticated requests

## Documentation
- **Code self-doc patterns:**
  - Add comprehensive godoc comments for all new interfaces, structs, and methods
  - Document error cases and return values
  - Document transaction handling and resource ownership checks

- **Interface documentation:**
  - Clearly document contract between layers
  - Document error types that can be returned

- **Testing documentation:**
  - Document test setup and verification approach
  - Document test data requirements

## Risk Matrix

| Risk | Severity | Mitigation |
|------|----------|------------|
| Incorrect SRS calculation | High | Leverage existing, tested SRS service; Add focused tests |
| Next review card query incorrect | Medium | Write comprehensive tests with various test data scenarios |
| Query performance issues | Medium | Ensure proper indexing on user_card_stats (user_id, next_review_at); Test with larger datasets |
| Concurrency issues in stats updates | Medium | Use proper transaction isolation; Include FOR UPDATE locking if needed |
| Error mapping inconsistencies | Low | Standardize error types and handling across layers |
| Security: User accesses other user's cards | Low | Implement strict ownership validation in service layer |

## Open Questions
- Should we implement pagination for the `GetNextReviewCard` endpoint to return multiple cards at once? Single card is simpler for now, can be extended later.
- Should we store the review history or just update the stats? Just updating stats is sufficient for initial implementation.
