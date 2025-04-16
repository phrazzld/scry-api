# TODO List

## JWT Refresh Token Refactoring

- [x] **T050:** Consolidate Refresh Token Tests
  - **Action:** Remove the redundant `internal/api/refresh_token_test.go` file and ensure all test cases are covered in `auth_handler_test.go`.
  - **Files:** `internal/api/refresh_token_test.go`, `internal/api/auth_handler_test.go`
  - **Complexity:** Low

- [ ] **T051:** Extract Token Generation Logic
  - **Action:** Refactor the duplicated token generation code in `Register` and `Login` handlers into a private helper method.
  - **Files:** `internal/api/auth_handler.go`
  - **Complexity:** Low

- [ ] **T052:** Improve Time Handling in Auth Handler
  - **Action:** Inject a time source into `AuthHandler` for calculating response `ExpiresAt` instead of using `time.Now()` directly.
  - **Files:** `internal/api/auth_handler.go`
  - **Complexity:** Medium

## Additional Improvements

- [ ] **T053:** Enhance Error and Log Messages
  - **Action:** Use more specific error messages for token generation failures and improve logging for refresh token operations.
  - **Files:** `internal/api/auth_handler.go`, `internal/service/auth/jwt_service_impl.go`
  - **Complexity:** Low

- [ ] **T054:** Refactor Large Tests
  - **Action:** Split the large `TestRefreshTokenSuccess` test into smaller focused tests for login and refresh flows.
  - **Files:** `internal/api/auth_handler_test.go`
  - **Complexity:** Medium

- [ ] **T055:** Improve Configuration Documentation
  - **Action:** Add comments explaining the relationship between access and refresh token lifetimes in the configuration.
  - **Files:** `config.yaml.example`
  - **Complexity:** Low
