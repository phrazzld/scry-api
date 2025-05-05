# T030 - Fix Linting Errors in Integration Tests and API Helpers

## Changes Made

1. Fixed error handling for `resp.Body.Close()` in card_api_integration_test.go
   - Added proper deferred error handling with t.Logf for failures

2. Added error handling for JSON encoding operations in API helper files
   - Updated setup.go to properly check and handle errors from json.NewEncoder(w).Encode() calls
   - Added error handling for encoding errors in all API response functions

3. Fixed unused function declarations in compatibility.go
   - Added `// nolint:unused` directive to unused compatibility functions
   - These functions are intentionally kept for backward compatibility

4. Resolved duplicate function declarations in API helpers
   - Removed duplicate AssertCardResponse, AssertStatsResponse, AssertErrorResponse functions
   - Removed duplicate GetAuthToken, GetCardByID, and related utility functions
   - Used shared implementations from request_helpers.go

5. Fixed imports and variable declarations
   - Added missing imports (database/sql, time)
   - Fixed undefined variables and imported types
   - Removed unused variables and imports

6. Fixed references to renamed functions
   - Renamed GenerateAuthHeader to GenerateAuthHeaderWithService to avoid conflicts
   - Updated card_api_integration_test.go to use GenerateAuthHeaderForUser instead

## Testing
- Tested with `go test -v -tags="integration" ./cmd/server`
- All linting errors related to error handling are now resolved
- Unused functions have been properly annotated

## Next Steps
- The remaining issues in testutils_test package are not part of this task and will be addressed separately in the future
