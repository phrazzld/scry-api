//go:build integration

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test server setup for integration tests
func setupTestServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()

	// Create UserStore using the provided database connection
	userStore := postgres.NewPostgresUserStore(db, 10)

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

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/user/profile", profileHandler)
		})
	})

	// Create test server
	testServer := httptest.NewServer(r)

	return testServer
}

func TestAuthIntegration(t *testing.T) {
	// Skip test if database is not available (testDB is set in TestMain)
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	// Start test server with the shared database connection
	testServer := setupTestServer(t, testDB)
	defer testServer.Close()

	// Test data
	email := "integration-test@example.com"
	password := "securepassword1234"

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
			t.Errorf("Failed to close register response body: %v", err)
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
			t.Errorf("Failed to close login response body: %v", err)
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
			t.Errorf("Failed to close protected route response body: %v", err)
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
			t.Errorf("Failed to close invalid token response body: %v", err)
		}
	}()

	// Check invalid token response
	assert.Equal(t, http.StatusUnauthorized, invalidTokenResp.StatusCode)
}
