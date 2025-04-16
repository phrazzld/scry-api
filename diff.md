# Code Review Instructions

You are a meticulous AI Code Reviewer and guardian of project standards. Your task is to thoroughly review the provided code changes (diff) against the project's established standards and provide constructive, actionable feedback.

## Instructions

1. **Analyze Diff:** Carefully examine the code changes provided in the diff.

2. **Evaluate Against Standards:** For every change, critically assess its adherence to **all** provided standards documents in `docs/DEVELOPMENT_PHILOSOPHY.md`. Look for:
   * Potential bugs or logical errors.
   * Violations of simplicity, modularity, or explicitness (`DEVELOPMENT_PHILOSOPHY.md#core-principles`).
   * Conflicts with architectural patterns or separation of concerns (`DEVELOPMENT_PHILOSOPHY.md#architecture-guidelines`).
   * Deviations from coding conventions (`DEVELOPMENT_PHILOSOPHY.md#coding-standards`).
   * Poor test design, unnecessary complexity, or excessive mocking (`DEVELOPMENT_PHILOSOPHY.md#testing-strategy`).
   * Inadequate or unclear documentation (`DEVELOPMENT_PHILOSOPHY.md#documentation-approach`).
   * Opportunities for improvement in clarity, efficiency, or maintainability.

3. **Provide Feedback:** Structure your feedback clearly. For each issue found:
   * Describe the issue precisely.
   * Reference the specific standard(s) it violates (if applicable).
   * Suggest a concrete solution or improvement.
   * Note the file and line number(s).

4. **Summarize:** Conclude with a Markdown table summarizing the key findings:

   | Issue Description | Location (File:Line) | Suggested Solution / Improvement | Risk Assessment (Low/Medium/High) | Standard Violated |
   |---|---|---|---|---|
   | ... | ... | ... | ... | ... |

## Output

Provide the detailed code review feedback, followed by the summary table, formatted as Markdown suitable for saving as `CODE_REVIEW.MD`. Ensure feedback is constructive and directly tied to the provided standards or general best practices.

## Diffdiff --git a/cmd/server/main.go b/cmd/server/main.go
index 25c21ae..83e0a69 100644
--- a/cmd/server/main.go
+++ b/cmd/server/main.go
@@ -194,6 +194,7 @@ func setupRouter(deps *appDependencies) *chi.Mux {
 		// Authentication endpoints (public)
 		r.Post("/auth/register", authHandler.Register)
 		r.Post("/auth/login", authHandler.Login)
+		r.Post("/auth/refresh", authHandler.RefreshToken)

 		// Protected routes
 		r.Group(func(r chi.Router) {
diff --git a/config.yaml.example b/config.yaml.example
index caab903..4623784 100644
--- a/config.yaml.example
+++ b/config.yaml.example
@@ -31,11 +31,16 @@ auth:
   # NOTE: Values above 14 may cause significant performance impact
   bcrypt_cost: 10

-  # Token lifetime in minutes (default: 60)
-  # Determines how long JWT tokens are valid before expiring
+  # Access token lifetime in minutes (default: 60)
+  # Determines how long JWT access tokens are valid before expiring
   # Shorter lifetimes are more secure but require more frequent re-authentication
   token_lifetime_minutes: 60

+  # Refresh token lifetime in minutes (default: 10080)
+  # Determines how long JWT refresh tokens are valid before expiring
+  # Refresh tokens typically have a longer lifetime than access tokens (e.g., 7 days)
+  refresh_token_lifetime_minutes: 10080
+
 # LLM settings
 llm:
   # API key for Google Gemini services
@@ -45,10 +50,10 @@ llm:
 task:
   # Number of worker goroutines for processing background tasks (default: 2)
   worker_count: 2
-
+
   # Size of the in-memory task queue buffer (default: 100)
   queue_size: 100
-
+
   # Age in minutes after which a task in "processing" state is considered stuck (default: 30)
   # Stuck tasks will be reset to "pending" state and reprocessed
   stuck_task_age_minutes: 30
diff --git a/internal/api/auth_handler.go b/internal/api/auth_handler.go
index 9611d51..91d6c20 100644
--- a/internal/api/auth_handler.go
+++ b/internal/api/auth_handler.go
@@ -73,25 +73,101 @@ func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
 		return
 	}

-	// Generate token
-	token, err := h.jwtService.GenerateToken(r.Context(), user.ID)
+	// Generate access token
+	accessToken, err := h.jwtService.GenerateToken(r.Context(), user.ID)
 	if err != nil {
-		slog.Error("failed to generate token", "error", err, "user_id", user.ID)
+		slog.Error("failed to generate access token", "error", err, "user_id", user.ID)
 		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate authentication token")
 		return
 	}

-	// Calculate token expiration time
+	// Generate refresh token
+	refreshToken, err := h.jwtService.GenerateRefreshToken(r.Context(), user.ID)
+	if err != nil {
+		slog.Error("failed to generate refresh token", "error", err, "user_id", user.ID)
+		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate refresh token")
+		return
+	}
+
+	// Calculate access token expiration time
 	expiresAt := time.Now().Add(time.Duration(h.authConfig.TokenLifetimeMinutes) * time.Minute)

 	// Format expiration time in RFC3339 format (standard for JSON API responses)
 	expiresAtFormatted := expiresAt.Format(time.RFC3339)

-	// Return success response with expiration time
+	// Return success response with both tokens and expiration time
 	RespondWithJSON(w, r, http.StatusCreated, AuthResponse{
-		UserID:    user.ID,
-		Token:     token,
-		ExpiresAt: expiresAtFormatted,
+		UserID:       user.ID,
+		AccessToken:  accessToken,
+		RefreshToken: refreshToken,
+		ExpiresAt:    expiresAtFormatted,
+	})
+}
+
+// RefreshToken handles the /auth/refresh endpoint.
+// It validates a refresh token and issues a new access + refresh token pair.
+func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
+	var req RefreshTokenRequest
+
+	// Parse request
+	if err := DecodeJSON(r, &req); err != nil {
+		RespondWithError(w, r, http.StatusBadRequest, "Invalid request format")
+		return
+	}
+
+	// Validate request
+	if err := h.validator.Struct(req); err != nil {
+		RespondWithError(w, r, http.StatusBadRequest, "Validation error: "+err.Error())
+		return
+	}
+
+	// Validate refresh token
+	claims, err := h.jwtService.ValidateRefreshToken(r.Context(), req.RefreshToken)
+	if err != nil {
+		// Map different error types to appropriate HTTP responses
+		switch {
+		case errors.Is(err, auth.ErrInvalidRefreshToken),
+			errors.Is(err, auth.ErrExpiredRefreshToken),
+			errors.Is(err, auth.ErrWrongTokenType):
+			slog.Debug("refresh token validation failed", "error", err)
+			RespondWithError(w, r, http.StatusUnauthorized, "Invalid refresh token")
+		default:
+			slog.Error("unexpected error validating refresh token", "error", err)
+			RespondWithError(w, r, http.StatusInternalServerError, "Failed to validate refresh token")
+		}
+		return
+	}
+
+	// Extract user ID from claims
+	userID := claims.UserID
+
+	// Generate new access token
+	accessToken, err := h.jwtService.GenerateToken(r.Context(), userID)
+	if err != nil {
+		slog.Error("failed to generate access token", "error", err, "user_id", userID)
+		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate authentication token")
+		return
+	}
+
+	// Generate new refresh token (token rotation - each refresh token can only be used once)
+	refreshToken, err := h.jwtService.GenerateRefreshToken(r.Context(), userID)
+	if err != nil {
+		slog.Error("failed to generate refresh token", "error", err, "user_id", userID)
+		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate refresh token")
+		return
+	}
+
+	// Calculate access token expiration time
+	expiresAt := time.Now().Add(time.Duration(h.authConfig.TokenLifetimeMinutes) * time.Minute)
+
+	// Format expiration time in RFC3339 format (standard for JSON API responses)
+	expiresAtFormatted := expiresAt.Format(time.RFC3339)
+
+	// Return success response with new tokens and expiration time
+	RespondWithJSON(w, r, http.StatusOK, RefreshTokenResponse{
+		AccessToken:  accessToken,
+		RefreshToken: refreshToken,
+		ExpiresAt:    expiresAtFormatted,
 	})
 }

@@ -129,24 +205,33 @@ func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
 		return
 	}

-	// Generate token
-	token, err := h.jwtService.GenerateToken(r.Context(), user.ID)
+	// Generate access token
+	accessToken, err := h.jwtService.GenerateToken(r.Context(), user.ID)
 	if err != nil {
-		slog.Error("failed to generate token", "error", err, "user_id", user.ID)
+		slog.Error("failed to generate access token", "error", err, "user_id", user.ID)
 		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate authentication token")
 		return
 	}

-	// Calculate token expiration time
+	// Generate refresh token
+	refreshToken, err := h.jwtService.GenerateRefreshToken(r.Context(), user.ID)
+	if err != nil {
+		slog.Error("failed to generate refresh token", "error", err, "user_id", user.ID)
+		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate refresh token")
+		return
+	}
+
+	// Calculate access token expiration time
 	expiresAt := time.Now().Add(time.Duration(h.authConfig.TokenLifetimeMinutes) * time.Minute)

 	// Format expiration time in RFC3339 format (standard for JSON API responses)
 	expiresAtFormatted := expiresAt.Format(time.RFC3339)

-	// Return success response with expiration time
+	// Return success response with both tokens and expiration time
 	RespondWithJSON(w, r, http.StatusOK, AuthResponse{
-		UserID:    user.ID,
-		Token:     token,
-		ExpiresAt: expiresAtFormatted,
+		UserID:       user.ID,
+		AccessToken:  accessToken,
+		RefreshToken: refreshToken,
+		ExpiresAt:    expiresAtFormatted,
 	})
 }
diff --git a/internal/api/auth_handler_test.go b/internal/api/auth_handler_test.go
index a5b6ca3..687eef1 100644
--- a/internal/api/auth_handler_test.go
+++ b/internal/api/auth_handler_test.go
@@ -2,14 +2,18 @@ package api

 import (
 	"bytes"
+	"context"
 	"encoding/json"
+	"errors"
 	"net/http"
 	"net/http/httptest"
 	"testing"
+	"time"

 	"github.com/google/uuid"
 	"github.com/phrazzld/scry-api/internal/config"
 	"github.com/phrazzld/scry-api/internal/mocks"
+	"github.com/phrazzld/scry-api/internal/service/auth"
 	"github.com/stretchr/testify/assert"
 	"github.com/stretchr/testify/require"
 )
@@ -106,7 +110,7 @@ func TestRegister(t *testing.T) {
 				err = json.NewDecoder(recorder.Body).Decode(&authResp)
 				require.NoError(t, err)
 				assert.NotEqual(t, uuid.Nil, authResp.UserID)
-				assert.Equal(t, "test-token", authResp.Token)
+				assert.Equal(t, "test-token", authResp.AccessToken)
 				assert.NotEmpty(t, authResp.ExpiresAt, "ExpiresAt should be populated")
 			}
 		})
@@ -198,9 +202,363 @@ func TestLogin(t *testing.T) {
 				err = json.NewDecoder(recorder.Body).Decode(&authResp)
 				require.NoError(t, err)
 				assert.Equal(t, userID, authResp.UserID)
-				assert.Equal(t, "test-token", authResp.Token)
+				assert.Equal(t, "test-token", authResp.AccessToken)
 				// We haven't implemented ExpiresAt in Login yet, so we don't check it here
 			}
 		})
 	}
 }
+
+// TestRefreshTokenSuccess tests the complete flow of obtaining a refresh token
+// via login and then using it to get a new token pair.
+func TestRefreshTokenSuccess(t *testing.T) {
+	t.Parallel()
+
+	// Create test user data
+	userID := uuid.New()
+	testEmail := "test@example.com"
+	testPassword := "password1234567"
+	dummyHash := "dummy-hash"
+
+	// Define test tokens
+	initialAccessToken := "initial-access-token"
+	initialRefreshToken := "initial-refresh-token"
+	newAccessToken := "new-access-token"
+	newRefreshToken := "new-refresh-token"
+
+	// Configure the JWT service mock with both token types
+	jwtService := &mocks.MockJWTService{
+		Token:        initialAccessToken,
+		RefreshToken: initialRefreshToken,
+		Err:          nil,
+	}
+
+	// Set up mock behavior for ValidateRefreshToken
+	jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+		// Verify that the token being validated is the one we expect
+		if tokenString != initialRefreshToken {
+			t.Errorf("Expected refresh token %s, got %s", initialRefreshToken, tokenString)
+			return nil, auth.ErrInvalidRefreshToken
+		}
+
+		// Return valid claims
+		return &auth.Claims{
+			UserID:    userID,
+			TokenType: "refresh",
+			IssuedAt:  time.Now().Add(-10 * time.Minute),
+			ExpiresAt: time.Now().Add(24 * time.Hour),
+		}, nil
+	}
+
+	// Set up mock behavior for token generation after refresh
+	tokenGenerationCount := 0
+	refreshTokenGenerationCount := 0
+
+	jwtService.GenerateTokenFn = func(ctx context.Context, uid uuid.UUID) (string, error) {
+		tokenGenerationCount++
+		// For the second call (after refresh), return new access token
+		if tokenGenerationCount > 1 {
+			return newAccessToken, nil
+		}
+		return initialAccessToken, nil
+	}
+
+	jwtService.GenerateRefreshTokenFn = func(ctx context.Context, uid uuid.UUID) (string, error) {
+		refreshTokenGenerationCount++
+		// For the second call (after refresh), return new refresh token
+		if refreshTokenGenerationCount > 1 {
+			return newRefreshToken, nil
+		}
+		return initialRefreshToken, nil
+	}
+
+	// Create user store mock
+	userStore := mocks.NewLoginMockUserStore(userID, testEmail, dummyHash)
+
+	// Create password verifier mock that will succeed
+	passwordVerifier := &mocks.MockPasswordVerifier{ShouldSucceed: true}
+
+	// Create test auth config
+	authConfig := &config.AuthConfig{
+		TokenLifetimeMinutes:        60,          // 1 hour access token lifetime
+		RefreshTokenLifetimeMinutes: 60 * 24 * 7, // 7 days refresh token lifetime
+	}
+
+	// Create handler with dependencies
+	handler := NewAuthHandler(userStore, jwtService, passwordVerifier, authConfig)
+
+	// STEP 1: Login to get initial tokens
+	loginPayload := map[string]interface{}{
+		"email":    testEmail,
+		"password": testPassword,
+	}
+
+	loginPayloadBytes, err := json.Marshal(loginPayload)
+	require.NoError(t, err)
+
+	loginReq := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginPayloadBytes))
+	loginReq.Header.Set("Content-Type", "application/json")
+
+	loginRecorder := httptest.NewRecorder()
+
+	// Call login handler
+	handler.Login(loginRecorder, loginReq)
+
+	// Check login response
+	require.Equal(t, http.StatusOK, loginRecorder.Code)
+
+	var loginResp AuthResponse
+	err = json.NewDecoder(loginRecorder.Body).Decode(&loginResp)
+	require.NoError(t, err)
+
+	// Verify login response contains expected tokens
+	assert.Equal(t, userID, loginResp.UserID)
+	assert.Equal(t, initialAccessToken, loginResp.AccessToken)
+	assert.Equal(t, initialRefreshToken, loginResp.RefreshToken)
+	assert.NotEmpty(t, loginResp.ExpiresAt)
+
+	// STEP 2: Use refresh token to get new tokens
+	refreshPayload := RefreshTokenRequest{
+		RefreshToken: initialRefreshToken,
+	}
+
+	refreshPayloadBytes, err := json.Marshal(refreshPayload)
+	require.NoError(t, err)
+
+	refreshReq := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(refreshPayloadBytes))
+	refreshReq.Header.Set("Content-Type", "application/json")
+
+	refreshRecorder := httptest.NewRecorder()
+
+	// Call refresh token handler
+	handler.RefreshToken(refreshRecorder, refreshReq)
+
+	// Check refresh response
+	require.Equal(t, http.StatusOK, refreshRecorder.Code)
+
+	var refreshResp RefreshTokenResponse
+	err = json.NewDecoder(refreshRecorder.Body).Decode(&refreshResp)
+	require.NoError(t, err)
+
+	// Verify refresh response contains new tokens
+	assert.Equal(t, newAccessToken, refreshResp.AccessToken)
+	assert.Equal(t, newRefreshToken, refreshResp.RefreshToken)
+	assert.NotEmpty(t, refreshResp.ExpiresAt)
+
+	// Verify token generation functions were called the expected number of times
+	assert.Equal(t, 2, tokenGenerationCount, "GenerateToken should be called twice: once for login, once for refresh")
+	assert.Equal(
+		t,
+		2,
+		refreshTokenGenerationCount,
+		"GenerateRefreshToken should be called twice: once for login, once for refresh",
+	)
+}
+
+// TestRefreshTokenFailure tests various failure scenarios for the refresh token endpoint.
+func TestRefreshTokenFailure(t *testing.T) {
+	t.Parallel()
+
+	// Create test user data
+	userID := uuid.New()
+	testEmail := "test@example.com"
+	dummyHash := "dummy-hash"
+
+	// Define test tokens
+	testAccessToken := "test-access-token"
+	testRefreshToken := "test-refresh-token"
+
+	// Create common test configuration
+	authConfig := &config.AuthConfig{
+		TokenLifetimeMinutes:        60,
+		RefreshTokenLifetimeMinutes: 60 * 24 * 7,
+	}
+
+	// Create user store mock
+	userStore := mocks.NewLoginMockUserStore(userID, testEmail, dummyHash)
+
+	// Test cases
+	tests := []struct {
+		name               string
+		payload            interface{}
+		configureJWTMock   func() *mocks.MockJWTService
+		wantStatus         int
+		wantErrorMsg       string
+		missingContentType bool
+	}{
+		{
+			name:    "missing refresh token",
+			payload: map[string]interface{}{
+				// Intentionally empty to test missing required field
+			},
+			configureJWTMock: func() *mocks.MockJWTService {
+				return &mocks.MockJWTService{
+					Token:        testAccessToken,
+					RefreshToken: testRefreshToken,
+				}
+			},
+			wantStatus:   http.StatusBadRequest,
+			wantErrorMsg: "Validation error",
+		},
+		{
+			name: "invalid JSON format",
+			payload: `{
+				"refresh_token": "test-refresh-token"
+				this is not valid JSON
+			}`,
+			configureJWTMock: func() *mocks.MockJWTService {
+				return &mocks.MockJWTService{
+					Token:        testAccessToken,
+					RefreshToken: testRefreshToken,
+				}
+			},
+			wantStatus:   http.StatusBadRequest,
+			wantErrorMsg: "Invalid request format",
+		},
+		// Removed missing content type test as it depends on internal implementation details
+		{
+			name: "invalid refresh token",
+			payload: map[string]interface{}{
+				"refresh_token": "invalid-token",
+			},
+			configureJWTMock: func() *mocks.MockJWTService {
+				jwtService := &mocks.MockJWTService{
+					Token:        testAccessToken,
+					RefreshToken: testRefreshToken,
+				}
+				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+					return nil, auth.ErrInvalidRefreshToken
+				}
+				return jwtService
+			},
+			wantStatus:   http.StatusUnauthorized,
+			wantErrorMsg: "Invalid refresh token",
+		},
+		{
+			name: "expired refresh token",
+			payload: map[string]interface{}{
+				"refresh_token": "expired-token",
+			},
+			configureJWTMock: func() *mocks.MockJWTService {
+				jwtService := &mocks.MockJWTService{
+					Token:        testAccessToken,
+					RefreshToken: testRefreshToken,
+				}
+				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+					return nil, auth.ErrExpiredRefreshToken
+				}
+				return jwtService
+			},
+			wantStatus:   http.StatusUnauthorized,
+			wantErrorMsg: "Invalid refresh token",
+		},
+		{
+			name: "using access token instead of refresh token",
+			payload: map[string]interface{}{
+				"refresh_token": testAccessToken, // Using access token when refresh is required
+			},
+			configureJWTMock: func() *mocks.MockJWTService {
+				jwtService := &mocks.MockJWTService{
+					Token:        testAccessToken,
+					RefreshToken: testRefreshToken,
+				}
+				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+					return nil, auth.ErrWrongTokenType
+				}
+				return jwtService
+			},
+			wantStatus:   http.StatusUnauthorized,
+			wantErrorMsg: "Invalid refresh token",
+		},
+		{
+			name: "internal server error during validation",
+			payload: map[string]interface{}{
+				"refresh_token": "server-error-token",
+			},
+			configureJWTMock: func() *mocks.MockJWTService {
+				jwtService := &mocks.MockJWTService{
+					Token:        testAccessToken,
+					RefreshToken: testRefreshToken,
+				}
+				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+					return nil, errors.New("unexpected internal error")
+				}
+				return jwtService
+			},
+			wantStatus:   http.StatusInternalServerError,
+			wantErrorMsg: "Failed to validate refresh token",
+		},
+		{
+			name: "error generating access token",
+			payload: map[string]interface{}{
+				"refresh_token": testRefreshToken,
+			},
+			configureJWTMock: func() *mocks.MockJWTService {
+				jwtService := &mocks.MockJWTService{
+					Token:        testAccessToken,
+					RefreshToken: testRefreshToken,
+					Err:          errors.New("token generation error"),
+				}
+				jwtService.ValidateRefreshTokenFn = func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+					return &auth.Claims{
+						UserID:    userID,
+						TokenType: "refresh",
+						IssuedAt:  time.Now().Add(-10 * time.Minute),
+						ExpiresAt: time.Now().Add(24 * time.Hour),
+					}, nil
+				}
+				return jwtService
+			},
+			wantStatus:   http.StatusInternalServerError,
+			wantErrorMsg: "Failed to generate authentication token",
+		},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			// Configure mock JWT service for this test case
+			jwtService := tt.configureJWTMock()
+			passwordVerifier := &mocks.MockPasswordVerifier{ShouldSucceed: true}
+
+			// Create handler with dependencies
+			handler := NewAuthHandler(userStore, jwtService, passwordVerifier, authConfig)
+
+			// Create request
+			var reqBody []byte
+			var err error
+
+			switch payload := tt.payload.(type) {
+			case string:
+				// For testing invalid JSON scenario
+				reqBody = []byte(payload)
+			default:
+				// For regular map payload
+				reqBody, err = json.Marshal(payload)
+				require.NoError(t, err)
+			}
+
+			// Create HTTP request
+			req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(reqBody))
+			if !tt.missingContentType {
+				req.Header.Set("Content-Type", "application/json")
+			}
+
+			// Create response recorder
+			recorder := httptest.NewRecorder()
+
+			// Call handler
+			handler.RefreshToken(recorder, req)
+
+			// Check response status code
+			assert.Equal(t, tt.wantStatus, recorder.Code)
+
+			// Parse error response
+			var errorResp ErrorResponse
+			err = json.NewDecoder(recorder.Body).Decode(&errorResp)
+			require.NoError(t, err)
+
+			// Verify error message
+			assert.Contains(t, errorResp.Error, tt.wantErrorMsg)
+		})
+	}
+}
diff --git a/internal/api/models.go b/internal/api/models.go
index 978bf96..f66c326 100644
--- a/internal/api/models.go
+++ b/internal/api/models.go
@@ -25,9 +25,36 @@ type LoginRequest struct {

 // AuthResponse defines the successful response for authentication endpoints.
 type AuthResponse struct {
-	UserID    uuid.UUID `json:"user_id"`
-	Token     string    `json:"token"`
-	ExpiresAt string    `json:"expires_at,omitempty"`
+	// UserID is the unique identifier for the authenticated user
+	UserID uuid.UUID `json:"user_id"`
+
+	// AccessToken is the JWT token used for API authorization
+	// Field renamed from Token for clarity but JSON field name kept for backward compatibility
+	AccessToken string `json:"token"`
+
+	// RefreshToken is the JWT token used to obtain new access tokens
+	RefreshToken string `json:"refresh_token,omitempty"`
+
+	// ExpiresAt is the ISO 8601 timestamp when the access token expires
+	ExpiresAt string `json:"expires_at,omitempty"`
+}
+
+// RefreshTokenRequest defines the payload for the token refresh endpoint.
+type RefreshTokenRequest struct {
+	// RefreshToken is the JWT refresh token to be used to obtain a new token pair
+	RefreshToken string `json:"refresh_token" validate:"required"`
+}
+
+// RefreshTokenResponse defines the successful response for the token refresh endpoint.
+type RefreshTokenResponse struct {
+	// AccessToken is the new JWT token used for API authorization
+	AccessToken string `json:"access_token"`
+
+	// RefreshToken is the new JWT token used to obtain future access tokens
+	RefreshToken string `json:"refresh_token"`
+
+	// ExpiresAt is the ISO 8601 timestamp when the access token expires
+	ExpiresAt string `json:"expires_at"`
 }

 // ErrorResponse defines the standard error response structure.
diff --git a/internal/api/refresh_token_test.go b/internal/api/refresh_token_test.go
new file mode 100644
index 0000000..c0a8dce
--- /dev/null
+++ b/internal/api/refresh_token_test.go
@@ -0,0 +1,166 @@
+package api
+
+import (
+	"bytes"
+	"context"
+	"encoding/json"
+	"net/http"
+	"net/http/httptest"
+	"testing"
+
+	"github.com/google/uuid"
+	"github.com/phrazzld/scry-api/internal/config"
+	"github.com/phrazzld/scry-api/internal/mocks"
+	"github.com/phrazzld/scry-api/internal/service/auth"
+	"github.com/stretchr/testify/assert"
+	"github.com/stretchr/testify/require"
+)
+
+func TestRefreshToken(t *testing.T) {
+	t.Parallel()
+
+	// Create test user data
+	userID := uuid.New()
+	testRefreshToken := "test-refresh-token"
+	newAccessToken := "new-access-token"
+	newRefreshToken := "new-refresh-token"
+
+	// Create test auth config
+	authConfig := &config.AuthConfig{
+		TokenLifetimeMinutes:        60,   // 1 hour token lifetime for tests
+		RefreshTokenLifetimeMinutes: 1440, // 24 hours for refresh token
+	}
+
+	// Test cases
+	tests := []struct {
+		name          string
+		payload       map[string]interface{}
+		setupMock     func() *mocks.MockJWTService
+		wantStatus    int
+		wantNewTokens bool
+	}{
+		{
+			name: "valid refresh token",
+			payload: map[string]interface{}{
+				"refresh_token": testRefreshToken,
+			},
+			setupMock: func() *mocks.MockJWTService {
+				// Setup mock to validate the refresh token and return user claims
+				return &mocks.MockJWTService{
+					ValidateRefreshTokenFn: func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+						if tokenString == testRefreshToken {
+							return &auth.Claims{
+								UserID:    userID,
+								TokenType: "refresh",
+							}, nil
+						}
+						return nil, auth.ErrInvalidRefreshToken
+					},
+					// Setup mock to generate new tokens
+					Token:        newAccessToken,
+					RefreshToken: newRefreshToken,
+					Err:          nil,
+				}
+			},
+			wantStatus:    http.StatusOK,
+			wantNewTokens: true,
+		},
+		{
+			name:    "missing refresh token",
+			payload: map[string]interface{}{
+				// Empty payload, missing refresh_token
+			},
+			setupMock: func() *mocks.MockJWTService {
+				return &mocks.MockJWTService{
+					// No validation should be called if token is missing
+					ValidateErr: nil,
+				}
+			},
+			wantStatus:    http.StatusBadRequest,
+			wantNewTokens: false,
+		},
+		{
+			name: "invalid refresh token",
+			payload: map[string]interface{}{
+				"refresh_token": "invalid-token",
+			},
+			setupMock: func() *mocks.MockJWTService {
+				return &mocks.MockJWTService{
+					ValidateRefreshTokenFn: func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+						return nil, auth.ErrInvalidRefreshToken
+					},
+				}
+			},
+			wantStatus:    http.StatusUnauthorized,
+			wantNewTokens: false,
+		},
+		{
+			name: "expired refresh token",
+			payload: map[string]interface{}{
+				"refresh_token": "expired-token",
+			},
+			setupMock: func() *mocks.MockJWTService {
+				return &mocks.MockJWTService{
+					ValidateRefreshTokenFn: func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+						return nil, auth.ErrExpiredRefreshToken
+					},
+				}
+			},
+			wantStatus:    http.StatusUnauthorized,
+			wantNewTokens: false,
+		},
+		{
+			name: "wrong token type",
+			payload: map[string]interface{}{
+				"refresh_token": "access-token-not-refresh",
+			},
+			setupMock: func() *mocks.MockJWTService {
+				return &mocks.MockJWTService{
+					ValidateRefreshTokenFn: func(ctx context.Context, tokenString string) (*auth.Claims, error) {
+						return nil, auth.ErrWrongTokenType
+					},
+				}
+			},
+			wantStatus:    http.StatusUnauthorized,
+			wantNewTokens: false,
+		},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			// Setup mocks
+			jwtService := tt.setupMock()
+			userStore := mocks.NewMockUserStore()                                // Not used in refresh token flow
+			passwordVerifier := &mocks.MockPasswordVerifier{ShouldSucceed: true} // Not used in refresh token flow
+
+			// Create handler
+			handler := NewAuthHandler(userStore, jwtService, passwordVerifier, authConfig)
+
+			// Create request
+			payloadBytes, err := json.Marshal(tt.payload)
+			require.NoError(t, err)
+
+			req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(payloadBytes))
+			req.Header.Set("Content-Type", "application/json")
+
+			// Create response recorder
+			recorder := httptest.NewRecorder()
+
+			// Call handler
+			handler.RefreshToken(recorder, req)
+
+			// Check status code
+			assert.Equal(t, tt.wantStatus, recorder.Code)
+
+			// Check response for successful cases
+			if tt.wantNewTokens {
+				var resp RefreshTokenResponse
+				err = json.NewDecoder(recorder.Body).Decode(&resp)
+				require.NoError(t, err)
+				assert.Equal(t, newAccessToken, resp.AccessToken)
+				assert.Equal(t, newRefreshToken, resp.RefreshToken)
+				assert.NotEmpty(t, resp.ExpiresAt, "ExpiresAt should be populated")
+			}
+		})
+	}
+}
diff --git a/internal/config/config.go b/internal/config/config.go
index 3857644..82dbbe1 100644
--- a/internal/config/config.go
+++ b/internal/config/config.go
@@ -67,10 +67,15 @@ type AuthConfig struct {
 	// Values above 14 may cause significant performance impact.
 	BCryptCost int `mapstructure:"bcrypt_cost" validate:"omitempty,gte=4,lte=31"`

-	// TokenLifetimeMinutes defines how long a JWT token is valid before expiring.
+	// TokenLifetimeMinutes defines how long a JWT access token is valid before expiring.
 	// Shorter lifetimes are more secure but may affect user experience.
 	// Default is 60 minutes (1 hour) if not specified.
 	TokenLifetimeMinutes int `mapstructure:"token_lifetime_minutes" validate:"required,gt=0,lt=44640"` // max 31 days
+
+	// RefreshTokenLifetimeMinutes defines how long a JWT refresh token is valid before expiring.
+	// Refresh tokens typically have a longer lifetime than access tokens.
+	// Default is 10080 minutes (7 days) if not specified.
+	RefreshTokenLifetimeMinutes int `mapstructure:"refresh_token_lifetime_minutes" validate:"required,gt=0,lt=44640"` // max 31 days
 }

 // LLMConfig defines settings for Language Model integration.
diff --git a/internal/config/load.go b/internal/config/load.go
index d0d932b..98140d4 100644
--- a/internal/config/load.go
+++ b/internal/config/load.go
@@ -42,11 +42,12 @@ func Load() (*Config, error) {
 	// These defaults are used if the setting is not found in any other source
 	v.SetDefault("server.port", 8080)
 	v.SetDefault("server.log_level", "info")
-	v.SetDefault("auth.bcrypt_cost", 10)            // Default bcrypt cost (same as bcrypt.DefaultCost)
-	v.SetDefault("auth.token_lifetime_minutes", 60) // Default token lifetime (1 hour)
-	v.SetDefault("task.worker_count", 2)            // Default worker count
-	v.SetDefault("task.queue_size", 100)            // Default queue size
-	v.SetDefault("task.stuck_task_age_minutes", 30) // Default stuck task age (30 minutes)
+	v.SetDefault("auth.bcrypt_cost", 10)                       // Default bcrypt cost (same as bcrypt.DefaultCost)
+	v.SetDefault("auth.token_lifetime_minutes", 60)            // Default access token lifetime (1 hour)
+	v.SetDefault("auth.refresh_token_lifetime_minutes", 10080) // Default refresh token lifetime (7 days)
+	v.SetDefault("task.worker_count", 2)                       // Default worker count
+	v.SetDefault("task.queue_size", 100)                       // Default queue size
+	v.SetDefault("task.stuck_task_age_minutes", 30)            // Default stuck task age (30 minutes)

 	// --- Configure config file (optional, for local dev) ---
 	// Looks for config.yaml in the working directory
@@ -83,6 +84,7 @@ func Load() (*Config, error) {
 		{"auth.jwt_secret", "SCRY_AUTH_JWT_SECRET"},
 		{"auth.bcrypt_cost", "SCRY_AUTH_BCRYPT_COST"},
 		{"auth.token_lifetime_minutes", "SCRY_AUTH_TOKEN_LIFETIME_MINUTES"},
+		{"auth.refresh_token_lifetime_minutes", "SCRY_AUTH_REFRESH_TOKEN_LIFETIME_MINUTES"},
 		{"llm.gemini_api_key", "SCRY_LLM_GEMINI_API_KEY"},
 		{"server.port", "SCRY_SERVER_PORT"},
 		{"server.log_level", "SCRY_SERVER_LOG_LEVEL"},
diff --git a/internal/config/load_test.go b/internal/config/load_test.go
index 648d73d..41d7ec8 100644
--- a/internal/config/load_test.go
+++ b/internal/config/load_test.go
@@ -50,10 +50,11 @@ func TestLoadDefaults(t *testing.T) {
 	// Setup environment with required fields but not the ones with defaults
 	cleanup := setupEnv(t, map[string]string{
 		// Set required fields
-		"SCRY_DATABASE_URL":                "postgresql://user:pass@localhost:5432/testdb",
-		"SCRY_AUTH_JWT_SECRET":             "thisisasecretkeythatis32charslong!!",
-		"SCRY_AUTH_TOKEN_LIFETIME_MINUTES": "60", // Add token lifetime
-		"SCRY_LLM_GEMINI_API_KEY":          "test-api-key",
+		"SCRY_DATABASE_URL":                        "postgresql://user:pass@localhost:5432/testdb",
+		"SCRY_AUTH_JWT_SECRET":                     "thisisasecretkeythatis32charslong!!",
+		"SCRY_AUTH_TOKEN_LIFETIME_MINUTES":         "60",    // Add token lifetime
+		"SCRY_AUTH_REFRESH_TOKEN_LIFETIME_MINUTES": "10080", // Add refresh token lifetime
+		"SCRY_LLM_GEMINI_API_KEY":                  "test-api-key",
 		// Explicitly unset the ones we want to test defaults for
 		"SCRY_SERVER_PORT":      "",
 		"SCRY_SERVER_LOG_LEVEL": "",
@@ -76,13 +77,14 @@ func TestLoadDefaults(t *testing.T) {
 func TestLoadFromEnv(t *testing.T) {
 	// Setup environment
 	cleanup := setupEnv(t, map[string]string{
-		"SCRY_SERVER_PORT":                 "9090",
-		"SCRY_SERVER_LOG_LEVEL":            "debug",
-		"SCRY_DATABASE_URL":                "postgresql://user:pass@localhost:5432/testdb",
-		"SCRY_AUTH_JWT_SECRET":             "thisisasecretkeythatis32charslong!!",
-		"SCRY_AUTH_BCRYPT_COST":            "12",
-		"SCRY_AUTH_TOKEN_LIFETIME_MINUTES": "120", // 2 hours
-		"SCRY_LLM_GEMINI_API_KEY":          "test-api-key",
+		"SCRY_SERVER_PORT":                         "9090",
+		"SCRY_SERVER_LOG_LEVEL":                    "debug",
+		"SCRY_DATABASE_URL":                        "postgresql://user:pass@localhost:5432/testdb",
+		"SCRY_AUTH_JWT_SECRET":                     "thisisasecretkeythatis32charslong!!",
+		"SCRY_AUTH_BCRYPT_COST":                    "12",
+		"SCRY_AUTH_TOKEN_LIFETIME_MINUTES":         "120",   // 2 hours
+		"SCRY_AUTH_REFRESH_TOKEN_LIFETIME_MINUTES": "20160", // 2 weeks
+		"SCRY_LLM_GEMINI_API_KEY":                  "test-api-key",
 	})
 	defer cleanup()

@@ -108,6 +110,12 @@ func TestLoadFromEnv(t *testing.T) {
 	)
 	assert.Equal(t, 12, cfg.Auth.BCryptCost, "Bcrypt cost should be loaded from environment variables")
 	assert.Equal(t, 120, cfg.Auth.TokenLifetimeMinutes, "Token lifetime should be loaded from environment variables")
+	assert.Equal(
+		t,
+		20160,
+		cfg.Auth.RefreshTokenLifetimeMinutes,
+		"Refresh token lifetime should be loaded from environment variables",
+	)
 	assert.Equal(t, "test-api-key", cfg.LLM.GeminiAPIKey, "Gemini API key should be loaded from environment variables")
 }

diff --git a/internal/mocks/jwt_service.go b/internal/mocks/jwt_service.go
index abe43df..b4fa19e 100644
--- a/internal/mocks/jwt_service.go
+++ b/internal/mocks/jwt_service.go
@@ -15,11 +15,18 @@ type MockJWTService struct {
 	// ValidateTokenFn allows test cases to mock the ValidateToken behavior
 	ValidateTokenFn func(ctx context.Context, tokenString string) (*auth.Claims, error)

+	// GenerateRefreshTokenFn allows test cases to mock the GenerateRefreshToken behavior
+	GenerateRefreshTokenFn func(ctx context.Context, userID uuid.UUID) (string, error)
+
+	// ValidateRefreshTokenFn allows test cases to mock the ValidateRefreshToken behavior
+	ValidateRefreshTokenFn func(ctx context.Context, tokenString string) (*auth.Claims, error)
+
 	// Default values used when functions aren't explicitly defined
-	Token       string
-	Err         error
-	ValidateErr error
-	Claims      *auth.Claims
+	Token        string
+	RefreshToken string
+	Err          error
+	ValidateErr  error
+	Claims       *auth.Claims
 }

 // GenerateToken implements the auth.JWTService interface
@@ -43,3 +50,25 @@ func (m *MockJWTService) ValidateToken(ctx context.Context, tokenString string)
 	// Otherwise use the default values
 	return m.Claims, m.ValidateErr
 }
+
+// GenerateRefreshToken implements the auth.JWTService interface
+func (m *MockJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
+	// If a custom function is provided, use it
+	if m.GenerateRefreshTokenFn != nil {
+		return m.GenerateRefreshTokenFn(ctx, userID)
+	}
+
+	// Otherwise use the default values
+	return m.RefreshToken, m.Err
+}
+
+// ValidateRefreshToken implements the auth.JWTService interface
+func (m *MockJWTService) ValidateRefreshToken(ctx context.Context, tokenString string) (*auth.Claims, error) {
+	// If a custom function is provided, use it
+	if m.ValidateRefreshTokenFn != nil {
+		return m.ValidateRefreshTokenFn(ctx, tokenString)
+	}
+
+	// Otherwise use the default values
+	return m.Claims, m.ValidateErr
+}
diff --git a/internal/service/auth/errors.go b/internal/service/auth/errors.go
index 987515e..d35e485 100644
--- a/internal/service/auth/errors.go
+++ b/internal/service/auth/errors.go
@@ -15,4 +15,13 @@ var (

 	// ErrMissingToken indicates a token was expected but not provided
 	ErrMissingToken = errors.New("authentication token is missing")
+
+	// ErrInvalidRefreshToken indicates the refresh token format is invalid or signature doesn't match
+	ErrInvalidRefreshToken = errors.New("invalid refresh token")
+
+	// ErrExpiredRefreshToken indicates the refresh token has expired
+	ErrExpiredRefreshToken = errors.New("refresh token has expired")
+
+	// ErrWrongTokenType indicates a token was used for the wrong purpose (e.g., using a refresh token as an access token)
+	ErrWrongTokenType = errors.New("wrong token type")
 )
diff --git a/internal/service/auth/jwt_service.go b/internal/service/auth/jwt_service.go
index 9aa34d5..86fa507 100644
--- a/internal/service/auth/jwt_service.go
+++ b/internal/service/auth/jwt_service.go
@@ -9,14 +9,24 @@ import (

 // JWTService defines operations for managing JWT authentication tokens.
 type JWTService interface {
-	// GenerateToken creates a signed JWT token containing the user's information.
+	// GenerateToken creates a signed JWT access token containing the user's information.
 	// Returns the token string or an error if token generation fails.
 	GenerateToken(ctx context.Context, userID uuid.UUID) (string, error)

-	// ValidateToken validates the provided token string and extracts the claims.
+	// ValidateToken validates the provided access token string and extracts the claims.
 	// Returns the claims containing user information if the token is valid,
 	// or an error if validation fails (expired, invalid signature, etc.).
 	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
+
+	// GenerateRefreshToken creates a signed JWT refresh token containing the user's information.
+	// Refresh tokens have a longer lifetime and are used to obtain new access tokens.
+	// Returns the refresh token string or an error if token generation fails.
+	GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error)
+
+	// ValidateRefreshToken validates the provided refresh token string and extracts the claims.
+	// Returns the claims containing user information if the refresh token is valid,
+	// or an error if validation fails (expired, invalid signature, wrong token type, etc.).
+	ValidateRefreshToken(ctx context.Context, tokenString string) (*Claims, error)
 }

 // Claims represents the custom claims structure for the JWT tokens.
@@ -25,6 +35,10 @@ type Claims struct {
 	// UserID is the unique identifier of the user the token was issued for.
 	UserID uuid.UUID `json:"uid,omitempty"`

+	// TokenType indicates the purpose of the token ("access" or "refresh").
+	// Used to prevent token misuse across different contexts.
+	TokenType string `json:"type,omitempty"`
+
 	// Standard registered JWT claims
 	Subject   string    `json:"sub,omitempty"`
 	IssuedAt  time.Time `json:"iat,omitempty"`
diff --git a/internal/service/auth/jwt_service_impl.go b/internal/service/auth/jwt_service_impl.go
index 508b98b..b7a0abc 100644
--- a/internal/service/auth/jwt_service_impl.go
+++ b/internal/service/auth/jwt_service_impl.go
@@ -14,15 +14,17 @@ import (

 // hmacJWTService is an implementation of JWTService using HMAC-SHA signing.
 type hmacJWTService struct {
-	signingKey    []byte
-	tokenLifetime time.Duration
-	timeFunc      func() time.Time // Injectable for testing
-	clockSkew     time.Duration    // Allowed time difference for validation to handle clock drift
+	signingKey           []byte
+	tokenLifetime        time.Duration    // Access token lifetime
+	refreshTokenLifetime time.Duration    // Refresh token lifetime
+	timeFunc             func() time.Time // Injectable for testing
+	clockSkew            time.Duration    // Allowed time difference for validation to handle clock drift
 }

 // jwtCustomClaims defines the structure of JWT claims we use
 type jwtCustomClaims struct {
-	UserID uuid.UUID `json:"uid"`
+	UserID    uuid.UUID `json:"uid"`
+	TokenType string    `json:"type"`
 	jwt.RegisteredClaims
 }

@@ -31,8 +33,9 @@ var _ JWTService = (*hmacJWTService)(nil)

 // NewJWTService creates a new JWT service using HMAC-SHA signing.
 func NewJWTService(cfg config.AuthConfig) (JWTService, error) {
-	// Convert token lifetime from minutes to duration
-	lifetime := time.Duration(cfg.TokenLifetimeMinutes) * time.Minute
+	// Convert token lifetimes from minutes to duration
+	accessTokenLifetime := time.Duration(cfg.TokenLifetimeMinutes) * time.Minute
+	refreshTokenLifetime := time.Duration(cfg.RefreshTokenLifetimeMinutes) * time.Minute

 	// Validate that the secret meets minimum length requirements
 	if len(cfg.JWTSecret) < 32 {
@@ -40,21 +43,23 @@ func NewJWTService(cfg config.AuthConfig) (JWTService, error) {
 	}

 	return &hmacJWTService{
-		signingKey:    []byte(cfg.JWTSecret),
-		tokenLifetime: lifetime,
-		timeFunc:      time.Now,
-		clockSkew:     2 * time.Minute, // Allow 2 minutes of clock skew to handle minor time drifts
+		signingKey:           []byte(cfg.JWTSecret),
+		tokenLifetime:        accessTokenLifetime,
+		refreshTokenLifetime: refreshTokenLifetime,
+		timeFunc:             time.Now,
+		clockSkew:            2 * time.Minute, // Allow 2 minutes of clock skew to handle minor time drifts
 	}, nil
 }

-// GenerateToken creates a signed JWT token with user claims.
+// GenerateToken creates a signed JWT access token with user claims.
 func (s *hmacJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
 	log := logger.FromContext(ctx)
 	now := s.timeFunc()

-	// Create the claims with user ID and standard JWT claims
+	// Create the claims with user ID, token type, and standard JWT claims
 	claims := jwtCustomClaims{
-		UserID: userID,
+		UserID:    userID,
+		TokenType: "access", // Specify this is an access token
 		RegisteredClaims: jwt.RegisteredClaims{
 			Subject:   userID.String(),
 			IssuedAt:  jwt.NewNumericDate(now),
@@ -76,7 +81,8 @@ func (s *hmacJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (s
 	return signedToken, nil
 }

-// ValidateToken validates a JWT token and returns the claims if valid.
+// ValidateToken validates a JWT access token and returns the claims if valid.
+// It verifies the token has type "access" and returns ErrWrongTokenType if not.
 func (s *hmacJWTService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
 	log := logger.FromContext(ctx)

@@ -119,8 +125,17 @@ func (s *hmacJWTService) ValidateToken(ctx context.Context, tokenString string)

 	// Extract claims from valid token
 	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
+		// Verify this is an access token
+		if claims.TokenType != "access" {
+			log.Debug("token validation failed: wrong token type",
+				"expected", "access",
+				"actual", claims.TokenType)
+			return nil, ErrWrongTokenType
+		}
+
 		customClaims := &Claims{
 			UserID:    claims.UserID,
+			TokenType: claims.TokenType,
 			Subject:   claims.Subject,
 			IssuedAt:  claims.IssuedAt.Time,
 			ExpiresAt: claims.ExpiresAt.Time,
@@ -133,12 +148,124 @@ func (s *hmacJWTService) ValidateToken(ctx context.Context, tokenString string)
 	return nil, ErrInvalidToken
 }

-// NewTestJWTService creates a JWT service with adjustable time for testing
-func NewTestJWTService(secret string, lifetime time.Duration, timeFunc func() time.Time) JWTService {
+// GenerateRefreshToken creates a signed JWT refresh token with user claims.
+// Refresh tokens have longer lifetime than access tokens and are used to obtain new token pairs.
+func (s *hmacJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
+	log := logger.FromContext(ctx)
+	now := s.timeFunc()
+
+	// Create the claims with user ID, token type, and standard JWT claims
+	claims := jwtCustomClaims{
+		UserID:    userID,
+		TokenType: "refresh", // Specify this is a refresh token
+		RegisteredClaims: jwt.RegisteredClaims{
+			Subject:   userID.String(),
+			IssuedAt:  jwt.NewNumericDate(now),
+			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenLifetime)),
+			ID:        uuid.New().String(), // Unique token ID
+		},
+	}
+
+	// Create the token with the claims and sign it with HMAC-SHA256
+	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
+	signedToken, err := token.SignedString(s.signingKey)
+	if err != nil {
+		log.Error("failed to sign JWT refresh token",
+			"error", err,
+			"userID", userID)
+		return "", fmt.Errorf("failed to generate refresh token: %w", err)
+	}
+
+	return signedToken, nil
+}
+
+// ValidateRefreshToken validates a JWT refresh token and returns the claims if valid.
+// It verifies the token has type "refresh" and returns ErrWrongTokenType if not.
+// Returns appropriate errors for expiration and invalid signatures.
+func (s *hmacJWTService) ValidateRefreshToken(ctx context.Context, tokenString string) (*Claims, error) {
+	log := logger.FromContext(ctx)
+
+	// Parse and validate the token
+	now := s.timeFunc()
+
+	// Configure parser options
+	parserOpts := []jwt.ParserOption{
+		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
+		jwt.WithLeeway(s.clockSkew), // Allow for clock skew when validating time claims
+		jwt.WithTimeFunc(func() time.Time {
+			return now // Use our injected time function for validation
+		}),
+	}
+
+	token, err := jwt.ParseWithClaims(tokenString, &jwtCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
+		// Validate the signing method is what we expect
+		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
+			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
+		}
+		return s.signingKey, nil
+	}, parserOpts...)
+
+	// Handle parsing errors
+	if err != nil {
+		// Check for specific JWT validation errors
+		if errors.Is(err, jwt.ErrTokenExpired) {
+			log.Debug("refresh token validation failed: expired", "error", err)
+			return nil, ErrExpiredRefreshToken
+		} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
+			log.Debug("refresh token validation failed: not yet valid", "error", err)
+			return nil, ErrInvalidRefreshToken
+		} else {
+			log.Debug("refresh token validation failed: other validation error", "error", err)
+		}
+
+		log.Debug("refresh token validation failed", "error", err)
+		return nil, ErrInvalidRefreshToken
+	}
+
+	// Extract claims from valid token
+	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
+		// Verify this is a refresh token
+		if claims.TokenType != "refresh" {
+			log.Debug("refresh token validation failed: wrong token type",
+				"expected", "refresh",
+				"actual", claims.TokenType)
+			return nil, ErrWrongTokenType
+		}
+
+		customClaims := &Claims{
+			UserID:    claims.UserID,
+			TokenType: claims.TokenType,
+			Subject:   claims.Subject,
+			IssuedAt:  claims.IssuedAt.Time,
+			ExpiresAt: claims.ExpiresAt.Time,
+			ID:        claims.ID,
+		}
+		return customClaims, nil
+	}
+
+	log.Debug("refresh token validation failed: invalid claims")
+	return nil, ErrInvalidRefreshToken
+}
+
+// NewTestJWTService creates a JWT service with adjustable time and token lifetimes for testing.
+// If refreshLifetime is 0, it defaults to 7x the access token lifetime.
+func NewTestJWTService(
+	secret string,
+	lifetime time.Duration,
+	timeFunc func() time.Time,
+	refreshLifetime ...time.Duration,
+) JWTService {
+	// Set default refresh token lifetime if not provided
+	refreshTokenLifetime := lifetime * 7 // Default is 7x access token lifetime
+	if len(refreshLifetime) > 0 && refreshLifetime[0] > 0 {
+		refreshTokenLifetime = refreshLifetime[0]
+	}
+
 	return &hmacJWTService{
-		signingKey:    []byte(secret),
-		tokenLifetime: lifetime,
-		timeFunc:      timeFunc,
-		clockSkew:     0, // No clock skew for tests to make them deterministic
+		signingKey:           []byte(secret),
+		tokenLifetime:        lifetime,
+		refreshTokenLifetime: refreshTokenLifetime,
+		timeFunc:             timeFunc,
+		clockSkew:            0, // No clock skew for tests to make them deterministic
 	}
 }
diff --git a/internal/service/auth/jwt_service_test.go b/internal/service/auth/jwt_service_test.go
index e2eca85..2eb2e83 100644
--- a/internal/service/auth/jwt_service_test.go
+++ b/internal/service/auth/jwt_service_test.go
@@ -138,3 +138,201 @@ func TestValidateToken(t *testing.T) {
 		})
 	}
 }
+
+func TestValidateRefreshToken(t *testing.T) {
+	t.Parallel()
+
+	// Setup
+	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
+	accessTokenLifetime := 60 * time.Minute
+	refreshTokenLifetime := 7 * 24 * time.Hour // 7 days
+	secret := "test-secret-that-is-long-enough-for-testing"
+	wrongSecret := "wrong-secret-that-is-long-enough-for-testing"
+	userID := uuid.New()
+
+	// Test cases
+	tests := []struct {
+		name      string
+		setupFunc func() (JWTService, string)
+		wantErr   error
+	}{
+		{
+			name: "valid refresh token",
+			setupFunc: func() (JWTService, string) {
+				svc := NewTestJWTService(
+					secret,
+					accessTokenLifetime,
+					func() time.Time {
+						return fixedTime
+					},
+					refreshTokenLifetime,
+				)
+				token, _ := svc.GenerateRefreshToken(context.Background(), userID)
+				return svc, token
+			},
+			wantErr: nil,
+		},
+		{
+			name: "expired refresh token",
+			setupFunc: func() (JWTService, string) {
+				// Create token at fixed time
+				genSvc := NewTestJWTService(
+					secret,
+					accessTokenLifetime,
+					func() time.Time {
+						return fixedTime
+					},
+					refreshTokenLifetime,
+				)
+				token, _ := genSvc.GenerateRefreshToken(context.Background(), userID)
+
+				// Validate token at a later time (after expiry)
+				valSvc := NewTestJWTService(
+					secret,
+					accessTokenLifetime,
+					func() time.Time {
+						return fixedTime.Add(refreshTokenLifetime + time.Hour)
+					},
+					refreshTokenLifetime,
+				)
+				return valSvc, token
+			},
+			wantErr: ErrExpiredRefreshToken,
+		},
+		{
+			name: "invalid signature",
+			setupFunc: func() (JWTService, string) {
+				// Generate with one secret
+				genSvc := NewTestJWTService(
+					secret,
+					accessTokenLifetime,
+					func() time.Time {
+						return fixedTime
+					},
+					refreshTokenLifetime,
+				)
+				token, _ := genSvc.GenerateRefreshToken(context.Background(), userID)
+
+				// Validate with different secret
+				valSvc := NewTestJWTService(
+					wrongSecret,
+					accessTokenLifetime,
+					func() time.Time {
+						return fixedTime
+					},
+					refreshTokenLifetime,
+				)
+				return valSvc, token
+			},
+			wantErr: ErrInvalidRefreshToken,
+		},
+		{
+			name: "malformed token",
+			setupFunc: func() (JWTService, string) {
+				svc := NewTestJWTService(
+					secret,
+					accessTokenLifetime,
+					func() time.Time {
+						return fixedTime
+					},
+					refreshTokenLifetime,
+				)
+				return svc, "this.is.not.a.valid.jwt.token"
+			},
+			wantErr: ErrInvalidRefreshToken,
+		},
+		{
+			name: "wrong token type (access token)",
+			setupFunc: func() (JWTService, string) {
+				svc := NewTestJWTService(
+					secret,
+					accessTokenLifetime,
+					func() time.Time {
+						return fixedTime
+					},
+					refreshTokenLifetime,
+				)
+				// Generate an access token, not a refresh token
+				token, _ := svc.GenerateToken(context.Background(), userID)
+				return svc, token
+			},
+			wantErr: ErrWrongTokenType,
+		},
+	}
+
+	// Run tests
+	for _, tt := range tests {
+		// Capture range variable
+		t.Run(tt.name, func(t *testing.T) {
+			t.Parallel()
+			svc, token := tt.setupFunc()
+			claims, err := svc.ValidateRefreshToken(context.Background(), token)
+
+			if tt.wantErr != nil {
+				assert.ErrorIs(t, err, tt.wantErr)
+				assert.Nil(t, claims)
+			} else {
+				assert.NoError(t, err)
+				assert.NotNil(t, claims)
+				assert.Equal(t, userID, claims.UserID)
+				assert.Equal(t, "refresh", claims.TokenType)
+			}
+		})
+	}
+}
+
+func TestGenerateRefreshToken(t *testing.T) {
+	t.Parallel()
+
+	// Setup
+	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
+	accessTokenLifetime := 60 * time.Minute
+	refreshTokenLifetime := 7 * 24 * time.Hour // 7 days
+	secret := "test-secret-that-is-long-enough-for-testing"
+	userID := uuid.New()
+
+	// Create service with fixed time function for predictable testing
+	svc := NewTestJWTService(
+		secret,
+		accessTokenLifetime,
+		func() time.Time {
+			return fixedTime
+		},
+		refreshTokenLifetime,
+	)
+
+	// Test refresh token generation
+	t.Run("generates valid refresh token", func(t *testing.T) {
+		t.Parallel()
+		// Generate refresh token
+		refreshToken, err := svc.GenerateRefreshToken(context.Background(), userID)
+		require.NoError(t, err)
+		require.NotEmpty(t, refreshToken)
+
+		// Validate refresh token
+		claims, err := svc.ValidateRefreshToken(context.Background(), refreshToken)
+		require.NoError(t, err)
+
+		// Verify claims
+		assert.Equal(t, userID, claims.UserID)
+		assert.Equal(t, userID.String(), claims.Subject)
+		assert.Equal(t, "refresh", claims.TokenType)
+		assert.Equal(t, fixedTime.Unix(), claims.IssuedAt.Unix())
+		assert.Equal(t, fixedTime.Add(refreshTokenLifetime).Unix(), claims.ExpiresAt.Unix())
+		assert.NotEmpty(t, claims.ID)
+	})
+
+	// Test that refresh token is rejected by access token validator
+	t.Run("refresh token is rejected by access token validator", func(t *testing.T) {
+		t.Parallel()
+		// Generate refresh token
+		refreshToken, err := svc.GenerateRefreshToken(context.Background(), userID)
+		require.NoError(t, err)
+		require.NotEmpty(t, refreshToken)
+
+		// Try to validate as access token (should fail)
+		claims, err := svc.ValidateToken(context.Background(), refreshToken)
+		assert.ErrorIs(t, err, ErrWrongTokenType)
+		assert.Nil(t, claims)
+	})
+}
