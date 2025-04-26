# Todo

## internal/store interfaces
- [x] **T001 · Feature · P1: define card store interface methods**
    - **Context:** PLAN.md > Detailed Build Steps > 1a
    - **Action:**
        1. In `internal/store/card.go`, add to `CardStore`:
            - `GetByID(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)`
            - `UpdateContent(ctx context.Context, cardID uuid.UUID, content json.RawMessage) error`
            - `Delete(ctx context.Context, cardID uuid.UUID) error`
    - **Done-when:**
        1. All three methods are declared.
        2. Code compiles.
    - **Depends-on:** none

- [x] **T002 · Feature · P1: define user card stats store interface methods**
    - **Context:** PLAN.md > Detailed Build Steps > 1b
    - **Action:**
        1. In `internal/store/user_card_stats.go`, add to `UserCardStatsStore`:
            - `GetForUpdate(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error)`
            - `Update(ctx context.Context, stats *domain.UserCardStats) error`
    - **Done-when:**
        1. Both methods are declared.
        2. Code compiles.
    - **Depends-on:** none

- [x] **T003 · Feature · P1: define SRSService interface method**
    - **Context:** PLAN.md > Detailed Build Steps > 1c
    - **Action:**
        1. In `internal/domain/srs/service.go`, declare `PostponeReview(stats *domain.UserCardStats, days int, now time.Time) (*domain.UserCardStats, error)` on `SRSService`.
    - **Done-when:**
        1. Method signature is present.
        2. Code compiles.
    - **Depends-on:** none

- [x] **T004 · Feature · P1: define CardService interface**
    - **Context:** PLAN.md > Detailed Build Steps > 1d
    - **Action:**
        1. In `internal/service/card_service.go`, declare `CardService` with methods:
            - `UpdateCardContent(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error`
            - `DeleteCard(ctx context.Context, userID, cardID uuid.UUID) error`
            - `PostponeCard(ctx context.Context, userID, cardID uuid.UUID, days int) (*domain.UserCardStats, error)`
    - **Done-when:**
        1. Interface is defined.
        2. Code compiles.
    - **Depends-on:** none

## internal/domain/srs
- [x] **T005 · Feature · P1: implement SRS postpone review logic**
    - **Context:** PLAN.md > Detailed Build Steps > 2a
    - **Action:**
        1. In `internal/domain/srs/service_impl.go`, implement `PostponeReview` to add `days` to `stats.NextReviewAt` and update `stats.UpdatedAt`.
        2. Validate `days >= 1`.
    - **Done-when:**
        1. Logic compiles and runs.
        2. Interface `SRSService` is satisfied by the implementation.
    - **Depends-on:** [T003]

- [x] **T006 · Test · P1: add unit tests for SRS postpone logic**
    - **Context:** PLAN.md > Detailed Build Steps > 2b
    - **Action:**
        1. Write tests in `internal/domain/srs/service_impl_test.go` covering standard cases, large `days`, DST, leap years.
    - **Done-when:**
        1. Tests pass.
        2. Coverage ≥ 95% for `service_impl.go`.
    - **Depends-on:** [T005]

## internal/platform/postgres
- [x] **T007 · Feature · P2: implement PostgresCardStore.GetByID**
    - **Context:** PLAN.md > Detailed Build Steps > 3a & 3f
    - **Action:**
        1. In `internal/platform/postgres/card_store.go`, implement `GetByID` using `SELECT * FROM cards WHERE id=$1`.
        2. Map `sql.ErrNoRows` to `store.ErrNotFound`.
    - **Done-when:**
        1. Method compiles.
        2. Returns `store.ErrNotFound` on missing row.
    - **Depends-on:** [T001]

- [x] **T008 · Feature · P2: implement PostgresCardStore.UpdateContent**
    - **Context:** PLAN.md > Detailed Build Steps > 3b
    - **Action:**
        1. Implement `UPDATE cards SET content=$1, updated_at=now() WHERE id=$2` in `card_store.go`.
        2. Handle `json.RawMessage` parameter.
    - **Done-when:**
        1. Method compiles.
        2. `updated_at` is set correctly.
    - **Depends-on:** [T001]

- [x] **T009 · Feature · P2: implement PostgresCardStore.Delete**
    - **Context:** PLAN.md > Detailed Build Steps > 3c
    - **Action:**
        1. Implement `DELETE FROM cards WHERE id=$1` in `card_store.go`.
    - **Done-when:**
        1. Method compiles.
    - **Depends-on:** [T001]

- [x] **T010 · Feature · P2: implement PostgresUserCardStatsStore.GetForUpdate**
    - **Context:** PLAN.md > Detailed Build Steps > 3d & 3f
    - **Action:**
        1. In `internal/platform/postgres/user_card_stats_store.go`, implement `SELECT ... FOR UPDATE`.
        2. Map `sql.ErrNoRows` to `store.ErrNotFound`.
    - **Done-when:**
        1. Method compiles.
        2. Locks row and returns correctly.
    - **Depends-on:** [T002]

- [x] **T011 · Feature · P2: implement PostgresUserCardStatsStore.Update**
    - **Context:** PLAN.md > Detailed Build Steps > 3e
    - **Action:**
        1. Implement `UPDATE user_card_stats SET next_review_at=$1, updated_at=now() WHERE user_id=$2 AND card_id=$3`.
    - **Done-when:**
        1. Method compiles.
    - **Depends-on:** [T002]

- [x] **T012 · Test · P1: add integration tests for Postgres store methods**
    - **Context:** PLAN.md > Detailed Build Steps > 3g; Risk: cascade delete, transactionality
    - **Action:**
        1. Use real DB with transaction rollback to test `GetByID`, `UpdateContent`, `Delete`, `GetForUpdate`, `Update`.
        2. Assert data correctness, `ErrNotFound` mapping, and rollback safety.
    - **Done-when:**
        1. Integration tests pass.
        2. No side-effects remain after rollback.
    - **Depends-on:** [T007, T008, T009, T010, T011]

## internal/service/card_service
- [x] **T013 · Feature · P1: define cardServiceImpl struct and errors**
    - **Context:** PLAN.md > Detailed Build Steps > 4a, 4e
    - **Action:**
        1. In `internal/service/card_service.go`, define basic `cardServiceImpl` with core fields.
        2. In `internal/service/errors.go`, declare `ErrNotOwned` and `ErrStatsNotFound`.
    - **Done-when:**
        1. Struct and errors are defined.
        2. Code compiles.
    - **Note:** SRSService field will be added during T016 implementation to avoid unused field warning.
    - **Depends-on:** [T001, T002, T003, T004]

- [x] **T014 · Feature · P1: implement UpdateCardContent method**
    - **Context:** PLAN.md > Detailed Build Steps > 4b; Security: authorization; Logging
    - **Action:**
        1. Fetch card via `CardStore.GetByID`; if `OwnerID != userID`, return `ErrNotOwned`.
        2. Call `CardStore.UpdateContent`.
        3. Add DEBUG entry/exit and ERROR logging.
    - **Done-when:**
        1. Method enforces ownership and updates content.
        2. Errors and logs behave as specified.
    - **Depends-on:** [T007, T008, T013]

- [x] **T015 · Feature · P1: implement DeleteCard method**
    - **Context:** PLAN.md > Detailed Build Steps > 4c; Security: authorization; Logging
    - **Action:**
        1. Fetch card; enforce `OwnerID == userID`; else `ErrNotOwned`.
        2. Call `CardStore.Delete`.
        3. Add DEBUG and ERROR logs.
    - **Done-when:**
        1. Method deletes card only if owned.
        2. Errors and logs behave as specified.
    - **Depends-on:** [T007, T009, T013]

- [x] **T016 · Feature · P0: implement PostponeCard method**
    - **Context:** PLAN.md > Detailed Build Steps > 4d; Concurrency; Transactionality; Logging
    - **Action:**
        1. Wrap in `UserCardStatsStore.WithTx`.
        2. Inside TX: `GetForUpdate`, call `SRSService.PostponeReview`, then `UserCardStatsStore.Update`.
        3. Return updated `UserCardStats`.
        4. Add DEBUG and ERROR logs.
    - **Done-when:**
        1. Next review date is postponed atomically.
        2. Appropriate errors returned and logged.
    - **Depends-on:** [T005, T010, T011, T013]

- [x] **T017 · Test · P1: add unit tests for CardService methods**
    - **Context:** PLAN.md > Detailed Build Steps > 4f; Testing Strategy
    - **Action:**
        1. In `internal/service/card_service_impl_test.go`, mock dependencies.
        2. Cover happy paths and all error paths (`NotFound`, `ErrNotOwned`, store/SRS errors).
    - **Done-when:**
        1. Tests pass with ≥90% coverage.
    - **Depends-on:** [T014, T015, T016]

## internal/api/card_handler
- [x] **T018 · Feature · P2: implement EditCard HTTP handler**
    - **Context:** PLAN.md > Detailed Build Steps > 5a–d; Validation; Logging
    - **Action:**
        1. In `internal/api/card_handler.go`, add `EditCard`.
        2. Decode/validate `EditCardRequest`, extract `userID` and `cardID`.
        3. Call `CardService.UpdateCardContent`, map errors via central handler.
        4. Return 204 No Content.
        5. Log DEBUG start/end and WARN/ERROR with `trace_id`, `user_id`, `card_id`.
    - **Done-when:**
        1. Handler compiles.
        2. Unit tests cover 204, 400, 403, 404, 500.
    - **Depends-on:** [T014]

- [x] **T019 · Feature · P2: implement DeleteCard HTTP handler**
    - **Context:** PLAN.md > Detailed Build Steps > 5a–d; Validation; Logging
    - **Action:**
        1. Add `DeleteCard`, extract context values.
        2. Call `CardService.DeleteCard`, map errors.
        3. Return 204 No Content.
        4. Log DEBUG and WARN/ERROR.
    - **Done-when:**
        1. Handler compiles.
        2. Unit tests cover 204, 403, 404, 500.
    - **Depends-on:** [T015]

- [x] **T020 · Feature · P1: implement PostponeCard HTTP handler**
    - **Context:** PLAN.md > Detailed Build Steps > 5a–d; Validation; Logging
    - **Action:**
        1. Add `PostponeCard`, decode/validate `PostponeCardRequest`.
        2. Extract `userID`, `cardID`, call `CardService.PostponeCard`.
        3. Map errors and return 200 with updated `UserCardStats` JSON.
        4. Log DEBUG start/end and WARN/ERROR.
    - **Done-when:**
        1. Handler compiles.
        2. Unit tests cover 200, 400, 404, 500.
    - **Depends-on:** [T016]

- [ ] **T021 · Test · P2: add unit tests for API handlers**
    - **Context:** PLAN.md > Testing Strategy > Unit Tests
    - **Action:**
        1. In `internal/api/card_handler_test.go`, mock `CardService`.
        2. Test decoding, validation, service calls, and error mapping for each handler.
    - **Done-when:**
        1. Tests pass with ≥90% coverage.
    - **Depends-on:** [T018, T019, T020]

## cmd/server
- [ ] **T022 · Chore · P1: register card management API routes**
    - **Context:** PLAN.md > Detailed Build Steps > 6a
    - **Action:**
        1. In `cmd/server/main.go` (or router setup), add:
            - `PUT /cards/{id}` → `cardHandler.EditCard`
            - `DELETE /cards/{id}` → `cardHandler.DeleteCard`
            - `POST /cards/{id}/postpone` → `cardHandler.PostponeCard`
    - **Done-when:**
        1. Routes are registered.
    - **Depends-on:** [T018, T019, T020]

- [ ] **T023 · Chore · P1: wire CardService into CardHandler via DI**
    - **Context:** PLAN.md > Detailed Build Steps > 6b
    - **Action:**
        1. Instantiate `cardServiceImpl` and inject into `NewCardHandler` in server setup.
    - **Done-when:**
        1. Application compiles and starts with real service.
    - **Depends-on:** [T013, T022]

## Integration Tests (HTTP)
- [ ] **T024 · Test · P0: add HTTP integration tests for edit card endpoint**
    - **Context:** PLAN.md > Detailed Build Steps > 7; Risk: unauthorized edits, JSONB validation
    - **Action:**
        1. In `cmd/server` test suite, use `testutils.WithTx` to test `PUT /cards/{id}`.
        2. Cover 204 success, 401/403, 404, 400 (invalid JSON), and verify `updated_at`.
    - **Done-when:**
        1. Tests pass in CI.
    - **Depends-on:** [T023]

- [ ] **T025 · Test · P0: add HTTP integration tests for delete card endpoint**
    - **Context:** PLAN.md > Detailed Build Steps > 7; Risk: cascade delete, unauthorized deletes
    - **Action:**
        1. Use `testutils.WithTx` to test `DELETE /cards/{id}`.
        2. Cover 204, 401/403, 404, verify card removed and stats cascade-deleted.
    - **Done-when:**
        1. Tests pass in CI.
    - **Depends-on:** [T023]

- [ ] **T026 · Test · P0: add HTTP integration tests for postpone card endpoint**
    - **Context:** PLAN.md > Detailed Build Steps > 7; Risk: race conditions, unauthorized postpones
    - **Action:**
        1. Use `testutils.WithTx` to test `POST /cards/{id}/postpone`.
        2. Cover 200, 400 (`days<1`), 404 (stats not found), auth failures, verify `next_review_at`.
        3. (Optional) Simulate concurrent postpones.
    - **Done-when:**
        1. Tests pass in CI.
    - **Depends-on:** [T023]

## Documentation
- [ ] **T027 · Chore · P2: update OpenAPI specification**
    - **Context:** PLAN.md > Detailed Build Steps > 8a
    - **Action:**
        1. In `docs/openapi.yaml`, add definitions for new endpoints, request/response schemas, status codes, security.
    - **Done-when:**
        1. Spec validates and reflects implementation.
    - **Depends-on:** [T018, T019, T020]

- [ ] **T028 · Chore · P3: add GoDoc comments**
    - **Context:** PLAN.md > Documentation > Code & 8b
    - **Action:**
        1. Add GoDoc to new public interfaces, methods, handlers, DTOs.
    - **Done-when:**
        1. All new public symbols have clear GoDoc.
    - **Depends-on:** [T001, T002, T003, T004, T018, T019, T020]

- [ ] **T029 · Chore · P3: update store README for cascade delete**
    - **Context:** PLAN.md > Detailed Build Steps > 8c; Risk: cascade delete
    - **Action:**
        1. In `internal/store/README.md`, document reliance on `ON DELETE CASCADE` for `user_card_stats`.
    - **Done-when:**
        1. README accurately notes cascade behavior.
    - **Depends-on:** [T009]

- [ ] **T030 · Chore · P3: add CHANGELOG entry**
    - **Context:** PLAN.md > Documentation > CHANGELOG
    - **Action:**
        1. In `CHANGELOG.md`, add entry summarizing Edit, Delete, Postpone endpoints and related interfaces.
    - **Done-when:**
        1. Changelog entry follows project conventions.
    - **Depends-on:** [T026]

### Clarifications & Assumptions
- [ ] **Issue:** Should we allow PATCH-style partial updates or only full PUT?
    - **Context:** PLAN.md > Open Questions
    - **Blocking?:** no
- [ ] **Issue:** Confirm hard-delete semantics vs. soft-delete requirement.
    - **Context:** PLAN.md > Open Questions
    - **Blocking?:** no
- [ ] **Issue:** Determine if batch postpone or admin override features are needed.
    - **Context:** PLAN.md > Open Questions
    - **Blocking?:** no
- [ ] **Issue:** Decide if dedicated audit log for delete/postpone events is required.
    - **Context:** PLAN.md > Open Questions
    - **Blocking?:** no
- [ ] **Issue:** Clarify client UX expectations around multiple postpones.
    - **Context:** PLAN.md > Open Questions
    - **Blocking?:** no
