# CI Failure: Card Tests Password Validation Issues

## Issue Analysis

We still have failing tests after fixing the auth validation tests. The CI is failing with:

1. One auth validation test still failing:
   - `TestAuthValidation_Integration/Registration_-_Password_Too_Short`: Error message mismatch

2. Multiple card-related test failures:
   - `TestCardEditIntegration`
   - `TestCardDeleteIntegration`
   - `TestCardPostponeIntegration`

All card tests are failing with the same error: "password must be at least 12 characters long"

## Root Cause

1. For the auth validation test, the error message from the API doesn't match what the test expects:
   - Test expects: "at least 12 characters"
   - Actual message: "Invalid Password: too short"

2. For the card tests, we've updated the helper functions in `internal/testutils/helpers.go`, but there appear to be additional test files (`cmd/server/card_api_test.go` and `/internal/testdb/db.go`) that have their own user creation logic with short passwords.

## Fix Required

1. Update the auth validation test assertion to match the actual error message:
   - Change assertion from "at least 12 characters" to "too short"

2. Fix card test files:
   - Examine `cmd/server/card_api_test.go` to update any hardcoded short passwords
   - Check if `internal/testdb/db.go` has any user creation functions with short passwords

## Implementation Plan

1. Update the auth validation test error message expectation

2. Find and fix any remaining test files with hard-coded passwords:
   - Check `cmd/server/card_api_test.go`
   - Check `internal/testdb/db.go`
   - Look for any other files that might create test users with passwords

3. Ensure all tests are using the updated longer passwords that meet the 12-character requirement

4. Create a consistency check to detect any future password length issues in tests

This should resolve all the remaining test failures related to password validation.
