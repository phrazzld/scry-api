# Todo

## Code Quality (cr-07)
- [x] **T032 · Refactor · P2: refactor card_api_integration_test.go to reduce file size**
    - **Context:** The file exceeds the 500-line limit (currently 519 lines)
    - **Action:**
        1. Split the file into logical test suites
        2. Extract common test utilities to shared helper functions
        3. Apply consistent test patterns across all API tests
    - **Done‑when:**
        1. No individual test file exceeds 500 lines
        2. All tests continue to pass
        3. Code coverage is maintained
    - **Depends‑on:** none

## Card Review API (cr-08)
- [x] **T033 · Fix · P0: update Card Review API tests to fix CI failures**
    - **Context:** CI is failing due to Card Review API tests that are testing functionality not yet implemented
    - **Action:**
        1. Update mock implementations for Card Review API tests to properly handle responses
        2. Fix issues with missing `user_id` field in card response JSON
        3. Implement proper validation error handling in mock server
    - **Done‑when:**
        1. CI passes on the feature/card-management-api branch
        2. Card Review API tests pass with proper mock implementations
    - **Depends‑on:** none

- [ ] **T034 · Feature · P1: implement Card Review API endpoints**
    - **Context:** The Card Review API functionality is needed for the SRS flashcard review workflow
    - **Action:**
        1. Implement `store.CardStore` function `GetNextReviewCard(userID time.Time)`
        2. Implement Fetch Next Card endpoint (`GET /cards/next`)
        3. Implement `store.UserCardStatsStore` function `UpdateStats(userID, cardID, outcome)`
        4. Implement Submit Answer endpoint (`POST /cards/{id}/answer`)
    - **Done‑when:**
        1. All endpoints are implemented and tested
        2. The previously disabled tests pass
        3. CI passes with all tests enabled
    - **Depends‑on:** none

## CI Fixes (cr-06)
- [x] **T029 · Fix · P1: resolve build tag compatibility issues in testutils package**
    - **Context:** CI is using test_without_external_deps build tag, but some internal tests require different tags
    - **Action:**
        1. Standardize build tags across test helper files
        2. Update pre-commit hooks to use consistent build tags
        3. Ensure test files are compatible with both local development and CI environments
    - **Done‑when:**
        1. All tests pass in both local environment and CI pipeline
        2. Pre-commit hooks run successfully with the same build tags as CI
    - **Depends‑on:** none

- [x] **T030 · Fix · P1: resolve linting errors in integration tests and API helpers**
    - **Context:** CI linting job fails due to error handling issues and unused functions
    - **Action:**
        1. Fix error handling for `resp.Body.Close()` in card_api_integration_test.go
        2. Add proper error handling for `json.Encoder.Encode()` in API helper files
        3. Remove or mark unused functions in compatibility.go
    - **Done‑when:**
        1. All linting errors are resolved
        2. CI linting job passes successfully
    - **Depends‑on:** none

- [x] **T031 · Fix · P1: implement proper SQL redaction in error logs**
    - **Context:** SQL queries are not properly redacted in error logs, causing test failures
    - **Action:**
        1. Analyze current SQL redaction implementation
        2. Update the error redaction logic to properly handle SQL queries
        3. Restore and update the `TestErrorLeakage` test to verify redaction
        4. Ensure all database layers properly redact SQL in error messages
    - **Done‑when:**
        1. All SQL queries in error logs are properly redacted with [REDACTED_SQL]
        2. TestErrorLeakage test passes and verifies proper redaction
        3. CI test job passes successfully
    - **Depends‑on:** none

- [x] **T020 · Fix · P0: resolve test failures in CI**
    - **Context:** CI is failing on the latest PR #26 due to test failures
    - **Action:**
        1. Investigate the root cause of test failures in the CI pipeline
        2. Implement fixes to ensure tests pass in the CI environment
        3. Update test configuration if necessary
    - **Done‑when:**
        1. CI pipeline passes all tests successfully
        2. Changes are merged to the feature branch
    - **Depends‑on:** T023, T024, T025, T026, T027

### Test Helper Refactoring Tasks
- [x] **T023 · Refactor · P1: refactor testutils API helpers to remove compatibility stubs**
    - **Context:** CI failures are partly due to stub implementations in integration test helpers
    - **Action:**
        1. Remove stub implementations in `integration_helpers.go` and `integration_test_helpers.go`
        2. Implement actual functionality for all helper methods
        3. Update affected tests to use correct function calls
    - **Done‑when:**
        1. All stub implementations are replaced with actual implementations
        2. Tests using these helpers pass locally
    - **Depends‑on:** none

- [x] **T024 · Refactor · P1: remove deprecated CardWithStatsOptions struct and old test helpers**
    - **Context:** Deprecated test helpers are causing inconsistencies in tests
    - **Action:**
        1. Remove deprecated `CardWithStatsOptions` struct
        2. Refactor tests to use the current pattern for creating test cards
        3. Update documentation to reflect the new approach
    - **Done‑when:**
        1. Deprecated struct and associated helpers are removed
        2. All tests consistently use the same pattern
    - **Depends‑on:** none

- [x] **T025 · Refactor · P1: audit and deduplicate JWT service and auth helpers**
    - **Context:** Multiple JWT service and auth helpers create confusion and inconsistency
    - **Action:**
        1. Identify all JWT service and auth helper implementations
        2. Consolidate them into a single, reusable implementation
        3. Update imports and usage across the codebase
    - **Done‑when:**
        1. Single, consistent JWT and auth helper implementation exists
        2. All tests using these helpers pass locally
    - **Depends‑on:** none

- [x] **T026 · Refactor · P1: standardize test server setup and request helpers**
    - **Context:** Test server setup and request helpers lack standardization
    - **Action:**
        1. Create consistent interfaces for test server setup
        2. Standardize request helper functions across all test files
        3. Ensure proper cleanup of resources in all test setup/teardown
    - **Done‑when:**
        1. Test server setup is consistent across all tests
        2. Request helpers follow a standardized pattern
    - **Depends‑on:** T023, T025

- [x] **T027 · Refactor · P1: remove all references to compatibility.go and integration_helpers.go stubs**
    - **Context:** References to compatibility layer stubs need to be removed
    - **Action:**
        1. Search for and replace all references to compatibility layer stubs
        2. Update imports to use the standardized helpers
        3. Ensure tests are using the actual implementations
    - **Done‑when:**
        1. No references to compatibility stubs remain
        2. All tests pass with the updated helpers
    - **Depends‑on:** T023, T024, T026

- [x] **T028 · Chore · P1: final validation and task completion**
    - **Context:** Verify that all refactoring tasks resolve the CI failures
    - **Action:**
        1. Run the entire test suite to verify all tests pass
        2. Check CI pipeline to confirm the issue is resolved
        3. Document the changes and update relevant documentation
    - **Done‑when:**
        1. CI pipeline passes all tests
        2. Documentation is updated to reflect changes
        3. T020 is marked complete
    - **Depends‑on:** T023, T024, T025, T026, T027
