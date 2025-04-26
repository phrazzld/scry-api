package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
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

	// Data fields for simple use cases
	// Card to return from mock service GetNextCard (success case)
	NextCard *domain.Card
	// Stats to return from mock service SubmitAnswer (success case)
	UpdatedStats *domain.UserCardStats
	// Error to return from mock service (error case)
	Error error

	// Override fields for advanced use cases - these take precedence over data fields
	// Function to replace the default GetNextCard behavior
	GetNextCardFn func(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
	// Function to replace the default SubmitAnswer behavior
	SubmitAnswerFn func(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error)

	// Auth customization
	// Function to replace the default JWT validation behavior
	ValidateTokenFn func(ctx context.Context, token string) (*auth.Claims, error)
}

// SetupCardReviewTestServer creates a test server with properly configured
// card review API routes and mocked dependencies.
//
// The mock service is configured according to the following priority:
// 1. Custom behavior functions (GetNextCardFn, SubmitAnswerFn) if provided
// 2. Data fields (NextCard, UpdatedStats, Error) if provided
// 3. Default empty behavior
//
// Automatically registers cleanup via t.Cleanup() so callers don't need to manually close the server.
func SetupCardReviewTestServer(t *testing.T, opts CardReviewServerOptions) *httptest.Server {
	t.Helper()

	// If userID is not provided, generate a new one
	if opts.UserID == uuid.Nil {
		opts.UserID = uuid.New()
	}

	// Create card review service mock with appropriate configuration
	mockOptions := []mocks.MockOption{}

	// Apply data fields if provided and no custom functions
	if opts.Error != nil {
		mockOptions = append(mockOptions, mocks.WithError(opts.Error))
	}
	if opts.NextCard != nil {
		mockOptions = append(mockOptions, mocks.WithNextCard(opts.NextCard))
	}
	if opts.UpdatedStats != nil {
		mockOptions = append(mockOptions, mocks.WithUpdatedStats(opts.UpdatedStats))
	}

	// Create the mock with collected options
	cardReviewMock := mocks.NewMockCardReviewService(mockOptions...)

	// Override with custom functions if provided (these take precedence)
	if opts.GetNextCardFn != nil {
		cardReviewMock.GetNextCardFn = opts.GetNextCardFn
	}
	if opts.SubmitAnswerFn != nil {
		cardReviewMock.SubmitAnswerFn = opts.SubmitAnswerFn
	}

	// Create JWT service - either custom or defaul
	var jwtService auth.JWTService
	if opts.ValidateTokenFn != nil {
		// Custom validation function provided, use a mock
		jwtMock := &mocks.MockJWTService{}
		jwtMock.ValidateTokenFn = opts.ValidateTokenFn
		jwtService = jwtMock
	} else {
		// Use the real JWT service for testing
		jwtService = NewTestJWTService()
	}

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Create auth middleware
	authMiddleware := authmiddleware.NewAuthMiddleware(jwtService)

	// Create logger
	logger := slog.Default()

	// Create mock card service
	cardServiceMock := &mocks.MockCardService{}

	// Create card handler
	cardHandler := api.NewCardHandler(cardReviewMock, cardServiceMock, logger)

	// Set up API routes
	router.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/cards/next", cardHandler.GetNextReviewCard)
			r.Post("/cards/{id}/answer", cardHandler.SubmitAnswer)
		})
	})

	// Create test server and register cleanup
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

//------------------------------------------------------------------------------
// Convenience Constructors for Common Test Scenarios
//------------------------------------------------------------------------------

// SetupCardReviewTestServerWithNextCard creates a test server that returns a specific card
// from the GetNextCard method. This is a convenience wrapper for the common success case.
func SetupCardReviewTestServerWithNextCard(
	t *testing.T,
	userID uuid.UUID,
	card *domain.Card,
) *httptest.Server {
	return SetupCardReviewTestServer(t, CardReviewServerOptions{
		UserID:   userID,
		NextCard: card,
	})
}

// SetupCardReviewTestServerWithError creates a test server that returns a specific error
// from both service methods. This is a convenience wrapper for error test cases.
func SetupCardReviewTestServerWithError(
	t *testing.T,
	userID uuid.UUID,
	err error,
) *httptest.Server {
	return SetupCardReviewTestServer(t, CardReviewServerOptions{
		UserID: userID,
		Error:  err,
	})
}

// SetupCardReviewTestServerWithUpdatedStats creates a test server that returns specific stats
// from the SubmitAnswer method. This is a convenience wrapper for the answer submission success case.
func SetupCardReviewTestServerWithUpdatedStats(
	t *testing.T,
	userID uuid.UUID,
	stats *domain.UserCardStats,
) *httptest.Server {
	return SetupCardReviewTestServer(t, CardReviewServerOptions{
		UserID:       userID,
		UpdatedStats: stats,
	})
}

// SetupCardReviewTestServerWithAuthError creates a test server that returns an authentication error.
// This is a convenience wrapper for testing authentication failure cases.
func SetupCardReviewTestServerWithAuthError(
	t *testing.T,
	userID uuid.UUID,
	authError error,
) *httptest.Server {
	return SetupCardReviewTestServer(t, CardReviewServerOptions{
		UserID: userID,
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

	// Create reques
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

	// Execute reques
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

	// Create reques
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

	// Execute reques
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

// ExecuteSubmitAnswerRequestWithRawID executes a POST /cards/{id}/answer reques
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

	// Execute reques
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

	// Check content if expected card has valid JSON conten
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
