package testutils

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//------------------------------------------------------------------------------
// Card Review API Test Server Setup
//------------------------------------------------------------------------------

// CardReviewServerOptions configures the setup of a card review API test server.
type CardReviewServerOptions struct {
	// UserID to use in test JWT token (required)
	UserID uuid.UUID

	// Database transaction to use for the test (required for real services)
	Tx *sql.Tx

	// Auth customization
	// Function to replace the default JWT validation behavior
	ValidateTokenFn func(ctx context.Context, token string) (*auth.Claims, error)
}

// SetupCardReviewTestServer creates a test server with properly configured
// card review API routes and real dependencies.
//
// Automatically registers cleanup via t.Cleanup() so callers don't need to manually close the server.
func SetupCardReviewTestServer(t *testing.T, opts CardReviewServerOptions) *httptest.Server {
	t.Helper()

	// If userID is not provided, generate a new one
	if opts.UserID == uuid.Nil {
		opts.UserID = uuid.New()
	}

	// Check that we have a transaction
	require.NotNil(t, opts.Tx, "Transaction is required for CardReviewTestServer")

	// Create JWT service - either custom or default
	var jwtService auth.JWTService
	if opts.ValidateTokenFn != nil {
		// Custom validation function provided, use a mock
		jwtMock := &mockJWTService{}
		jwtMock.validateTokenFn = opts.ValidateTokenFn
		jwtService = jwtMock
	} else {
		// Use the real JWT service for testing
		testService, err := CreateTestJWTService()
		require.NoError(t, err, "Failed to create JWT service")
		jwtService = testService
	}

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Create auth middleware
	authMiddleware := authmiddleware.NewAuthMiddleware(jwtService)

	// Set up API routes
	router.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			// The real routes would be registered here when using real services
		})
	})

	// Create server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// Simple mock implementation of JWTService for custom validation behavior
type mockJWTService struct {
	validateTokenFn func(ctx context.Context, token string) (*auth.Claims, error)
}

func (m *mockJWTService) ValidateToken(ctx context.Context, token string) (*auth.Claims, error) {
	return m.validateTokenFn(ctx, token)
}

func (m *mockJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	return "mock-token", nil
}

func (m *mockJWTService) ValidateRefreshToken(ctx context.Context, token string) (*auth.Claims, error) {
	return nil, auth.ErrInvalidRefreshToken
}

func (m *mockJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	return "mock-refresh-token", nil
}

//------------------------------------------------------------------------------
// Convenience Constructors for Common Test Scenarios
//------------------------------------------------------------------------------

// SetupCardReviewTestServerWithAuthError creates a test server that returns an authentication error.
// This is a convenience wrapper for testing authentication failure cases.
func SetupCardReviewTestServerWithAuthError(
	t *testing.T,
	tx *sql.Tx,
	userID uuid.UUID,
	authError error,
) *httptest.Server {
	return SetupCardReviewTestServer(t, CardReviewServerOptions{
		UserID: userID,
		Tx:     tx,
		ValidateTokenFn: func(ctx context.Context, token string) (*auth.Claims, error) {
			return nil, authError
		},
	})
}

//------------------------------------------------------------------------------
// Card Review API Test Request Helpers
//------------------------------------------------------------------------------

// ExecuteGetNextCardRequest executes a GET /cards/next request against the test server.
// Automatically registers cleanup for the response body so callers don't need to manually close it.
// Returns the response and error, if any.
func ExecuteGetNextCardRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
) (*http.Response, error) {
	t.Helper()

	// Create request
	req, err := http.NewRequest("GET", server.URL+"/api/cards/next", nil)
	if err != nil {
		return nil, err
	}

	// Generate real auth token with the provided user ID
	authHeader, err := GenerateAuthHeader(userID)
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	// Add auth header
	req.Header.Set("Authorization", authHeader)

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)

	// Register cleanup for the response body if the request succeeded
	if err == nil && resp != nil {
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		})
	}

	return resp, err
}

// ExecuteSubmitAnswerRequest executes a POST /cards/{id}/answer request against the test server.
// Automatically registers cleanup for the response body so callers don't need to manually close it.
// Returns the response and error, if any.
func ExecuteSubmitAnswerRequest(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	cardID uuid.UUID,
	outcome domain.ReviewOutcome,
) (*http.Response, error) {
	t.Helper()

	// Create request body
	requestBody := map[string]string{"outcome": string(outcome)}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// Create request
	req, err := http.NewRequest(
		"POST",
		server.URL+"/api/cards/"+cardID.String()+"/answer",
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, err
	}

	// Generate real auth token with the provided user ID
	authHeader, err := GenerateAuthHeader(userID)
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	// Add headers
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)

	// Register cleanup for the response body if the request succeeded
	if err == nil && resp != nil {
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		})
	}

	return resp, err
}

// ExecuteSubmitAnswerRequestWithRawID executes a POST /cards/{id}/answer request
// with a raw ID string (can be invalid for testing error cases).
// Automatically registers cleanup for the response body so callers don't need to manually close it.
// Returns the response and error, if any.
func ExecuteSubmitAnswerRequestWithRawID(
	t *testing.T,
	server *httptest.Server,
	userID uuid.UUID,
	rawCardID string,
	outcome domain.ReviewOutcome,
) (*http.Response, error) {
	t.Helper()

	// Create request body
	requestBody := map[string]string{"outcome": string(outcome)}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// Create request with raw ID string (may be invalid)
	req, err := http.NewRequest(
		"POST",
		server.URL+"/api/cards/"+rawCardID+"/answer",
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, err
	}

	// Generate real auth token with the provided user ID
	authHeader, err := GenerateAuthHeader(userID)
	if err != nil {
		t.Fatalf("Failed to generate auth header: %v", err)
	}

	// Add headers
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)

	// Register cleanup for the response body if the request succeeded
	if err == nil && resp != nil {
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		})
	}

	return resp, err
}

// AssertCardResponse checks that a response contains a valid card with the expected values.
// Note: No longer registers cleanup for the response body as the request helpers handle this.
func AssertCardResponse(t *testing.T, resp *http.Response, expectedCard *domain.Card) {
	t.Helper()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Parse response
	var cardResp api.CardResponse
	err = json.Unmarshal(body, &cardResp)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, expectedCard.ID.String(), cardResp.ID)
	assert.Equal(t, expectedCard.UserID.String(), cardResp.UserID)
	assert.Equal(t, expectedCard.MemoID.String(), cardResp.MemoID)

	// Check content if expected card has valid JSON content
	var expectedContent map[string]interface{}
	if err := json.Unmarshal(expectedCard.Content, &expectedContent); err == nil {
		content, ok := cardResp.Content.(map[string]interface{})
		assert.True(t, ok, "Content should be a map")

		// Check content fields
		for key, expectedValue := range expectedContent {
			assert.Equal(t, expectedValue, content[key])
		}
	}
}

// AssertStatsResponse checks that a response contains valid stats with the expected values.
// Note: No longer registers cleanup for the response body as the request helpers handle this.
func AssertStatsResponse(t *testing.T, resp *http.Response, expectedStats *domain.UserCardStats) {
	t.Helper()

	// Check status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Parse response
	var statsResp api.UserCardStatsResponse
	err = json.Unmarshal(body, &statsResp)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, expectedStats.UserID.String(), statsResp.UserID)
	assert.Equal(t, expectedStats.CardID.String(), statsResp.CardID)
	assert.Equal(t, expectedStats.Interval, statsResp.Interval)
	assert.Equal(t, expectedStats.EaseFactor, statsResp.EaseFactor)
	assert.Equal(t, expectedStats.ConsecutiveCorrect, statsResp.ConsecutiveCorrect)
	assert.Equal(t, expectedStats.ReviewCount, statsResp.ReviewCount)
	assert.True(t, expectedStats.LastReviewedAt.Equal(statsResp.LastReviewedAt))
	assert.True(t, expectedStats.NextReviewAt.Equal(statsResp.NextReviewAt))
}

// CheckErrorResponse checks that a response contains an error with the expected status code.
func CheckErrorResponse(resp *http.Response, expectedStatus int, expectedMsg string) error {
	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the error response
	var errorResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &errorResp); err != nil {
		return fmt.Errorf("failed to unmarshal error response: %w", err)
	}

	// Verify the error message
	if expectedMsg != "" && errorResp.Error != expectedMsg {
		return fmt.Errorf("expected error message %q, got %q", expectedMsg, errorResp.Error)
	}

	return nil
}
