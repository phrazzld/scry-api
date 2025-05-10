//go:build integration || test_without_external_deps

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test server setup for integration tests with transaction isolation
func setupTestServer(t *testing.T, tx *sql.Tx) *httptest.Server {
	t.Helper()

	// Create UserStore using the provided transaction
	userStore := postgres.NewPostgresUserStore(tx, 10)

	// Create JWT Service
	authConfig := config.AuthConfig{
		JWTSecret:            "thisisatestjwtsecretthatis32charslong",
		TokenLifetimeMinutes: 60,
	}
	jwtService, err := auth.NewJWTService(authConfig)
	require.NoError(t, err)

	// Create router
	r := chi.NewRouter()

	// Apply standard middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Create the password verifier
	passwordVerifier := auth.NewBcryptVerifier()

	// Create API handlers with test config
	testAuthConfig := &config.AuthConfig{
		TokenLifetimeMinutes: 60, // 1 hour for tests
	}
	logger := slog.Default()
	authHandler := api.NewAuthHandler(
		userStore,
		jwtService,
		passwordVerifier,
		testAuthConfig,
		logger,
	)
	authMiddleware := authmiddleware.NewAuthMiddleware(jwtService)

	// Create a test handler for protected routes
	profileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authmiddleware.GetUserID(r)
		if !ok {
			shared.RespondWithError(w, r, http.StatusUnauthorized, "User ID not found in context")
			return
		}
		shared.RespondWithJSON(w, r, http.StatusOK, map[string]interface{}{
			"user_id": userID,
			"message": "Profile data",
		})
	})

	// Register routes
	r.Route("/api", func(r chi.Router) {
		// Authentication endpoints (public)
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.RefreshToken)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/user/profile", profileHandler)
		})
	})

	// Create test server
	testServer := httptest.NewServer(r)
	t.Cleanup(func() {
		testServer.Close()
	})

	return testServer
}

func TestAuthIntegration(t *testing.T) {
	// Skip test if database is not available
	if db.ShouldSkipDatabaseTest() {
		t.Skip("DATABASE_URL or SCRY_TEST_DB_URL not set - skipping integration test")
	}

	// Get a test database connection
	testDB := testdb.GetTestDBWithT(t)

	// Use transaction isolation
	testdb.WithTx(t, testDB, func(t *testing.T, tx *sql.Tx) {
		// Start test server with transaction isolation
		testServer := setupTestServer(t, tx)

		// Test data
		email := "integration-test@example.com"
		password := "securepassword123456"

		// Register user
		registerPayload := map[string]interface{}{
			"email":    email,
			"password": password,
		}
		registerBody, _ := json.Marshal(registerPayload)

		registerResp, err := http.Post(
			testServer.URL+"/api/auth/register",
			"application/json",
			bytes.NewBuffer(registerBody),
		)
		require.NoError(t, err)
		defer func() {
			if err := registerResp.Body.Close(); err != nil {
				t.Logf("Warning: Failed to close register response body: %v", err)
			}
		}()

		// Check register response
		assert.Equal(t, http.StatusCreated, registerResp.StatusCode)

		var registerData map[string]interface{}
		err = json.NewDecoder(registerResp.Body).Decode(&registerData)
		require.NoError(t, err)
		assert.NotEmpty(t, registerData["user_id"])
		assert.NotEmpty(t, registerData["token"])

		// Login with registered user
		loginPayload := map[string]interface{}{
			"email":    email,
			"password": password,
		}
		loginBody, _ := json.Marshal(loginPayload)

		loginResp, err := http.Post(
			testServer.URL+"/api/auth/login",
			"application/json",
			bytes.NewBuffer(loginBody),
		)
		require.NoError(t, err)
		defer func() {
			if err := loginResp.Body.Close(); err != nil {
				t.Logf("Warning: Failed to close login response body: %v", err)
			}
		}()

		// Check login response
		assert.Equal(t, http.StatusOK, loginResp.StatusCode)

		var loginData map[string]interface{}
		err = json.NewDecoder(loginResp.Body).Decode(&loginData)
		require.NoError(t, err)
		assert.NotEmpty(t, loginData["user_id"])
		assert.NotEmpty(t, loginData["token"])
		token := loginData["token"].(string)

		// Test protected route
		req, err := http.NewRequest("GET", testServer.URL+"/api/user/profile", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		protectedResp, err := client.Do(req)
		require.NoError(t, err)
		defer func() {
			if err := protectedResp.Body.Close(); err != nil {
				t.Logf("Warning: Failed to close protected route response body: %v", err)
			}
		}()

		// Check protected route response
		assert.Equal(t, http.StatusOK, protectedResp.StatusCode)

		// Test invalid token
		req, err = http.NewRequest("GET", testServer.URL+"/api/user/profile", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid-token")

		invalidTokenResp, err := client.Do(req)
		require.NoError(t, err)
		defer func() {
			if err := invalidTokenResp.Body.Close(); err != nil {
				t.Logf("Warning: Failed to close invalid token response body: %v", err)
			}
		}()

		// Check invalid token response
		assert.Equal(t, http.StatusUnauthorized, invalidTokenResp.StatusCode)
	})
}

// TestAuthValidation_Integration tests validation errors in the auth endpoints
func TestAuthValidation_Integration(t *testing.T) {
	// Skip test if database is not available
	if db.ShouldSkipDatabaseTest() {
		t.Skip("DATABASE_URL or SCRY_TEST_DB_URL not set - skipping integration test")
	}

	// Get a test database connection
	testDB := testdb.GetTestDBWithT(t)

	// Use transaction isolation
	testdb.WithTx(t, testDB, func(t *testing.T, tx *sql.Tx) {
		// Start test server with transaction isolation
		testServer := setupTestServer(t, tx)

		// Test cases for registration validation
		t.Run("Registration - Invalid Email Format", func(t *testing.T) {
			payload := map[string]interface{}{
				"email":    "invalid-email",
				"password": "securepassword123456",
			}

			reqBody, err := json.Marshal(payload)
			require.NoError(t, err)

			resp, err := http.Post(
				testServer.URL+"/api/auth/register",
				"application/json",
				bytes.NewBuffer(reqBody),
			)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: Failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Verify error response contains email validation error
			var errResp shared.ErrorResponse
			err = json.NewDecoder(resp.Body).Decode(&errResp)
			require.NoError(t, err)
			assert.Contains(t, errResp.Error, "email")
		})

		t.Run("Registration - Password Too Short", func(t *testing.T) {
			payload := map[string]interface{}{
				"email":    "valid@example.com",
				"password": "short",
			}

			reqBody, err := json.Marshal(payload)
			require.NoError(t, err)

			resp, err := http.Post(
				testServer.URL+"/api/auth/register",
				"application/json",
				bytes.NewBuffer(reqBody),
			)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: Failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Verify error response contains password validation error
			var errResp shared.ErrorResponse
			err = json.NewDecoder(resp.Body).Decode(&errResp)
			require.NoError(t, err)
			assert.Contains(t, errResp.Error, "too short")
		})

		t.Run("Registration - Missing Fields", func(t *testing.T) {
			payload := map[string]interface{}{}

			reqBody, err := json.Marshal(payload)
			require.NoError(t, err)

			resp, err := http.Post(
				testServer.URL+"/api/auth/register",
				"application/json",
				bytes.NewBuffer(reqBody),
			)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: Failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("Registration - Email Already Exists", func(t *testing.T) {
			// Create a test user with a known email
			userEmail := "duplicate@example.com"
			password := "securepassword123456"

			// Create the user using plain HTTP request
			createPayload := map[string]interface{}{
				"email":    userEmail,
				"password": password,
			}
			createBody, err := json.Marshal(createPayload)
			require.NoError(t, err)

			_, err = http.Post(
				testServer.URL+"/api/auth/register",
				"application/json",
				bytes.NewBuffer(createBody),
			)
			require.NoError(t, err)

			// Try to register with the same email
			duplicateResp, err := http.Post(
				testServer.URL+"/api/auth/register",
				"application/json",
				bytes.NewBuffer(createBody), // Same payload creates a duplicate
			)
			require.NoError(t, err)
			defer func() {
				if err := duplicateResp.Body.Close(); err != nil {
					t.Logf("Warning: Failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusConflict, duplicateResp.StatusCode)
		})

		// Test cases for login validation
		t.Run("Login - Non-existent User", func(t *testing.T) {
			payload := map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": "securepassword123456",
			}

			reqBody, err := json.Marshal(payload)
			require.NoError(t, err)

			resp, err := http.Post(
				testServer.URL+"/api/auth/login",
				"application/json",
				bytes.NewBuffer(reqBody),
			)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: Failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("Login - Incorrect Password", func(t *testing.T) {
			// Create a test user first
			userEmail := "test-user@example.com"
			password := "securepassword123456"

			// Create the user
			createPayload := map[string]interface{}{
				"email":    userEmail,
				"password": password,
			}
			createBody, err := json.Marshal(createPayload)
			require.NoError(t, err)

			_, err = http.Post(
				testServer.URL+"/api/auth/register",
				"application/json",
				bytes.NewBuffer(createBody),
			)
			require.NoError(t, err)

			// Try to login with incorrect password
			loginPayload := map[string]interface{}{
				"email":    userEmail,
				"password": "wrong-password123456",
			}
			loginBody, err := json.Marshal(loginPayload)
			require.NoError(t, err)

			resp, err := http.Post(
				testServer.URL+"/api/auth/login",
				"application/json",
				bytes.NewBuffer(loginBody),
			)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: Failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		})

		t.Run("Login - Invalid JSON", func(t *testing.T) {
			resp, err := http.Post(
				testServer.URL+"/api/auth/login",
				"application/json",
				bytes.NewBuffer([]byte(`{"email": "test@example.com", "password": "test"`)), // Missing closing brace
			)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Warning: Failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	})
}
