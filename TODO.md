# TODO

## JWT Refresh Token Implementation

- [x] **T029:** Add RefreshTokenLifetimeMinutes to AuthConfig
    - **Action:** Add the `RefreshTokenLifetimeMinutes` field to the `AuthConfig` struct in `/internal/config/config.go` to configure refresh token lifetime. Include appropriate `mapstructure` and `validate` tags.
    - **Depends On:** None
    - **Files:** `/internal/config/config.go`
    - **Complexity:** Low

- [x] **T030:** Define Refresh Token Errors
    - **Action:** Add specific error variables for refresh token operations in `/internal/service/auth/errors.go`: `ErrInvalidRefreshToken`, `ErrExpiredRefreshToken`, and `ErrWrongTokenType`.
    - **Depends On:** None
    - **Files:** `/internal/service/auth/errors.go`
    - **Complexity:** Low

- [ ] **T031:** Add TokenType to Claims Struct
    - **Action:** Add a `TokenType` string field to the `Claims` struct in `/internal/service/auth/jwt_service.go` to differentiate between access and refresh tokens.
    - **Depends On:** None
    - **Files:** `/internal/service/auth/jwt_service.go`
    - **Complexity:** Low

- [ ] **T032:** Extend JWTService Interface
    - **Action:** Add new method signatures to the `JWTService` interface in `/internal/service/auth/jwt_service.go`: `GenerateRefreshToken` and `ValidateRefreshToken`.
    - **Depends On:** [T031]
    - **Files:** `/internal/service/auth/jwt_service.go`
    - **Complexity:** Low

- [ ] **T033:** Update JWT Service Struct
    - **Action:** Add the `refreshTokenLifetime` field to the `hmacJWTService` struct and add the `TokenType` field to the `jwtCustomClaims` struct in `/internal/service/auth/jwt_service_impl.go`.
    - **Depends On:** [T031]
    - **Files:** `/internal/service/auth/jwt_service_impl.go`
    - **Complexity:** Low

- [ ] **T034:** Update NewJWTService Constructor
    - **Action:** Modify the `NewJWTService` function in `/internal/service/auth/jwt_service_impl.go` to calculate both token lifetimes from config and store them in the struct.
    - **Depends On:** [T029, T033]
    - **Files:** `/internal/service/auth/jwt_service_impl.go`
    - **Complexity:** Medium

- [ ] **T035:** Update GenerateToken Method
    - **Action:** Modify the `GenerateToken` method in `/internal/service/auth/jwt_service_impl.go` to include the token type "access" in the claims.
    - **Depends On:** [T033, T034]
    - **Files:** `/internal/service/auth/jwt_service_impl.go`
    - **Complexity:** Low

- [ ] **T036:** Implement GenerateRefreshToken Method
    - **Action:** Implement the `GenerateRefreshToken` method in `/internal/service/auth/jwt_service_impl.go` to create refresh tokens with longer lifetime and "refresh" token type.
    - **Depends On:** [T032, T033, T034]
    - **Files:** `/internal/service/auth/jwt_service_impl.go`
    - **Complexity:** Medium

- [ ] **T037:** Update ValidateToken Method
    - **Action:** Modify the `ValidateToken` method in `/internal/service/auth/jwt_service_impl.go` to verify the token has type "access" and return `ErrWrongTokenType` if not.
    - **Depends On:** [T030, T033, T034]
    - **Files:** `/internal/service/auth/jwt_service_impl.go`
    - **Complexity:** Medium

- [ ] **T038:** Implement ValidateRefreshToken Method
    - **Action:** Implement the `ValidateRefreshToken` method in `/internal/service/auth/jwt_service_impl.go` following the implementation plan. Verify type "refresh" and handle appropriate errors.
    - **Depends On:** [T030, T032, T033, T034]
    - **Files:** `/internal/service/auth/jwt_service_impl.go`
    - **Complexity:** High

- [ ] **T039:** Update NewTestJWTService Function
    - **Action:** Modify the `NewTestJWTService` function in `/internal/service/auth/jwt_service_impl.go` to accept and set the refresh token lifetime for testing.
    - **Depends On:** [T033, T034]
    - **Files:** `/internal/service/auth/jwt_service_impl.go`
    - **Complexity:** Low

- [ ] **T040:** Update AuthResponse Model
    - **Action:** Modify the `AuthResponse` struct in `/internal/api/models.go`: rename `Token` to `AccessToken` and add a new `RefreshToken` field.
    - **Depends On:** None
    - **Files:** `/internal/api/models.go`
    - **Complexity:** Low

- [ ] **T041:** Add Refresh Token Request/Response Models
    - **Action:** Add new structs to `/internal/api/models.go`: `RefreshTokenRequest` with a refresh token field and `RefreshTokenResponse` with access token, refresh token, and expiry fields.
    - **Depends On:** None
    - **Files:** `/internal/api/models.go`
    - **Complexity:** Low

- [ ] **T042:** Update Register Handler
    - **Action:** Modify the `Register` method in `/internal/api/auth_handler.go` to generate both token types and return them in the updated `AuthResponse`.
    - **Depends On:** [T036, T040]
    - **Files:** `/internal/api/auth_handler.go`
    - **Complexity:** Medium

- [ ] **T043:** Update Login Handler
    - **Action:** Modify the `Login` method in `/internal/api/auth_handler.go` to generate both token types and return them in the updated `AuthResponse`.
    - **Depends On:** [T036, T040]
    - **Files:** `/internal/api/auth_handler.go`
    - **Complexity:** Medium

- [ ] **T044:** Implement RefreshToken Handler
    - **Action:** Implement the new `RefreshToken` method in `/internal/api/auth_handler.go` to validate refresh tokens, generate new token pairs, and implement token rotation.
    - **Depends On:** [T035, T036, T038, T041]
    - **Files:** `/internal/api/auth_handler.go`
    - **Complexity:** High

- [ ] **T045:** Register Refresh Token Endpoint
    - **Action:** Add the `POST /auth/refresh` route to the router setup in `/cmd/server/main.go`, mapping to the new `authHandler.RefreshToken` method.
    - **Depends On:** [T044]
    - **Files:** `/cmd/server/main.go`
    - **Complexity:** Low

- [ ] **T046:** Write JWT Service Unit Tests - GenerateRefreshToken
    - **Action:** Add unit tests for the `GenerateRefreshToken` method in `/internal/service/auth/jwt_service_impl_test.go` to verify token generation and claims.
    - **Depends On:** [T036, T039]
    - **Files:** `/internal/service/auth/jwt_service_impl_test.go`
    - **Complexity:** Medium

- [ ] **T047:** Write JWT Service Unit Tests - ValidateRefreshToken
    - **Action:** Add unit tests for the `ValidateRefreshToken` method in `/internal/service/auth/jwt_service_impl_test.go` with cases for valid, expired, invalid signature, and wrong token type.
    - **Depends On:** [T037, T038, T039]
    - **Files:** `/internal/service/auth/jwt_service_impl_test.go`
    - **Complexity:** Medium

- [ ] **T048:** Write Integration Tests for Success Case
    - **Action:** Add an integration test to `/internal/api/auth_handler_test.go` for successful token refresh flow, from login through refresh to verification of new tokens.
    - **Depends On:** [T042, T043, T044, T045]
    - **Files:** `/internal/api/auth_handler_test.go`
    - **Complexity:** Medium

- [ ] **T049:** Write Integration Tests for Failure Cases
    - **Action:** Add integration tests to `/internal/api/auth_handler_test.go` for unsuccessful refresh scenarios: using access token, invalid token, and missing token.
    - **Depends On:** [T042, T043, T044, T045]
    - **Files:** `/internal/api/auth_handler_test.go`
    - **Complexity:** Medium
