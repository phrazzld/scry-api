# CI Failure: Authentication Validation Test Failures

## Issue Analysis

The CI is failing with multiple errors. After fixing the testutils reference issue, there are now failing tests related to authentication validation and password requirements.

Key failing tests:
1. `TestAuthValidation_Integration/Registration_-_Password_Too_Short`
2. `TestAuthValidation_Integration/Login_-_Non-existent_User`
3. `TestAuthValidation_Integration/Login_-_Incorrect_Password`
4. `TestCardEditIntegration` and other card tests fail with: "password must be at least 12 characters long"

## Root Cause

There appears to be a mismatch between the password validation requirements and the test data. The error message "password must be at least 12 characters long" suggests that a minimum password length of 12 characters is enforced, but the tests are using shorter passwords.

This could be due to:
1. A recent change in password policy (increasing minimum length from 8 to 12 characters)
2. A difference between the test environment and CI environment configuration
3. Tests that haven't been updated to match the new password requirements

Specifically, in the auth validation tests:
- The "Password Too Short" test is failing because the error message returned doesn't match what the test expects
- The "Non-existent User" test is failing because it expects a 401 status but is getting a 404
- The "Incorrect Password" test is failing because it expects a 401 status but is getting a 500

## Fix Required

1. Update test helper functions to use compliant passwords (at least 12 characters):
   - In `internal/testutils/helpers.go`, update `CreateTestUser` and any other functions that create test users
   - In `cmd/server/auth_integration_test.go`, ensure all test passwords are at least 12 characters

2. Fix status code expectations in auth tests:
   - For non-existent users, update tests to expect 404 instead of 401
   - For incorrect passwords, investigate why a 500 error is occurring instead of 401

3. Ensure all test validation messages match the actual validation error messages:
   - Update expected error messages to match the actual ones returned from the API

## Implementation Plan

1. First, check the password validation logic in the domain layer
2. Update all test helper functions to use compliant passwords
3. Fix auth validation tests to expect the correct status codes and error messages
4. Run the tests locally to verify the fixes before pushing

This is a higher priority fix because it's affecting multiple tests including the main authentication and card management functionality.
