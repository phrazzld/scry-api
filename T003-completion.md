# T003 · Test · P1: Audit Integration Test Coverage Gaps Post-Mock Deletion

## Coverage Gap Analysis

### API Endpoints and Routes

Based on the API routes defined in `cmd/server/main.go`, the following endpoints are available:

```
/api/auth/register          [POST]
/api/auth/login             [POST]
/api/auth/refresh           [POST]
/api/memos                  [POST]
/api/cards/next             [GET]
/api/cards/{id}/answer      [POST]
/api/cards/{id}             [PUT, DELETE]
/api/cards/{id}/postpone    [POST]
/health                     [GET]
```

### Current Integration Test Coverage

After reviewing all existing integration test files, the current coverage is:

1. **Auth Endpoints**:
   - `auth_integration_test.go` covers:
     - ✅ `/api/auth/register` - basic flow
     - ✅ `/api/auth/login` - basic flow
     - ❌ `/api/auth/refresh` - not covered by integration tests

2. **Card Endpoints**:
   - `get_card_api_integration_test.go` covers:
     - ✅ `/api/cards/next` - success case and no cards due

   - `card_review_api_test.go` covers:
     - ✅ `/api/cards/next` - extensive test cases
     - ✅ `/api/cards/{id}/answer` - extensive test cases with validation

   - `card_api_integration_test.go` covers:
     - ✅ `/api/cards/{id}` [PUT] - edit card tests
     - ✅ `/api/cards/{id}` [DELETE] - delete card tests
     - ✅ `/api/cards/{id}/postpone` - postpone card tests

3. **Memo Endpoints**:
   - ❌ `/api/memos` - not covered by integration tests

### Coverage Gaps Identified

Based on this analysis, the following coverage gaps exist:

1. **Missing Endpoint Coverage**:
   - `/api/auth/refresh` - No integration tests for token refresh functionality
   - `/api/memos` [POST] - No integration tests for memo creation

2. **Auth Coverage Limitations**:
   - `/api/auth/register` - Limited test cases in integration tests (only happy path)
   - `/api/auth/login` - Limited test cases in integration tests (only happy path)

3. **Error Case Coverage Gaps**:
   - Missing tests for server errors in the auth endpoints
   - Missing tests for validation errors in several endpoints

### Critical Scenarios Previously Covered by Mock Tests

The deleted mock-based test files (`auth_api_test.go` and `card_api_test.go`) covered the following critical scenarios that are now missing from integration tests:

1. **Auth Endpoint Scenarios** (from `auth_api_test.go`):
   - Registration validation errors (invalid email format, password too short)
   - Email already exists error
   - Database errors during registration
   - Token generation errors
   - Login with non-existent user
   - Login with incorrect password
   - Refresh token validation (expired token, invalid token)

2. **Card Endpoint Scenarios** (from `card_api_test.go`):
   - Card content validation errors
   - JSON unmarshaling errors for card content
   - Database failures
   - Detailed error case handling tests

### Priority Assessment for T004 Implementation

Based on the identified gaps, the following test scenarios should be prioritized in T004:

1. **High Priority**:
   - Implement integration tests for `/api/auth/refresh` (missing entirely)
   - Implement integration tests for `/api/memos` (missing entirely)
   - Add validation error tests for auth endpoints (missing validation test coverage)

2. **Medium Priority**:
   - Add error case tests for auth endpoints (server errors, database errors)
   - Expand card endpoint tests for additional edge cases

3. **Lower Priority**:
   - Additional performance or stress testing scenarios
   - Advanced error injection tests (these can be addressed separately if needed)

## Conclusion

The current integration test suite has good coverage for card-related endpoints but significant gaps in auth endpoints and memo creation. The next task (T004) should focus on implementing tests for these missing scenarios to maintain comprehensive test coverage after the removal of the mock-based tests.
