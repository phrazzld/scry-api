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
	// UserID to use in test JWT token
	UserID uuid.UUID
	// Card to return from mock service GetNextCard
	NextCard *domain.Card
	// Stats to return from mock service SubmitAnswer
	UpdatedStats *domain.UserCardStats
	// Error to return from mock service
	Error error
	// Function to replace the default GetNextCard behavior
	GetNextCardFn func(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
	// Function to replace the default SubmitAnswer behavior
	SubmitAnswerFn func(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error)
	// Function to replace the default JWT validation behavior
	ValidateTokenFn func(ctx context.Context, token string) (*auth.Claims, error)
}

// SetupCardReviewTestServer creates a test server with properly configured
// card review API routes and mocked dependencies.
// Automatically registers cleanup via t.Cleanup() so callers don't need to manually close the server.
func SetupCardReviewTestServer(t *testing.T, opts CardReviewServerOptions) *httptest.Server {
	t.Helper()

	// If userID is not provided, generate a new one
	if opts.UserID == uuid.Nil {
		opts.UserID = uuid.New()
	}

	// Create card review service mock
	var cardReviewMock *mocks.MockCardReviewService

	if opts.GetNextCardFn != nil || opts.SubmitAnswerFn != nil {
		// Custom functions provided
		cardReviewMock = &mocks.MockCardReviewService{
			GetNextCardFn:  opts.GetNextCardFn,
			SubmitAnswerFn: opts.SubmitAnswerFn,
			NextCard:       opts.NextCard,
			UpdatedStats:   opts.UpdatedStats,
			Err:            opts.Error,
		}
	} else if opts.Error != nil {
		// Error case
		cardReviewMock = mocks.NewMockCardReviewService(
			mocks.WithError(opts.Error),
		)
	} else if opts.NextCard != nil {
		// Success case for GetNextCard
		cardReviewMock = mocks.NewMockCardReviewService(
			mocks.WithNextCard(opts.NextCard),
		)
	} else if opts.UpdatedStats != nil {
		// Success case for SubmitAnswer
		cardReviewMock = mocks.NewMockCardReviewService(
			mocks.WithUpdatedStats(opts.UpdatedStats),
		)
	} else {
		// Default case
		cardReviewMock = mocks.NewMockCardReviewService()
	}

	// Create JWT service
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

	// Create card handler
	cardHandler := api.NewCardHandler(cardReviewMock, logger)

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
// Card Review API Test Request Helpers
//------------------------------------------------------------------------------

// ExecuteGetNextCardRequest executes a GET /cards/next request against the test server.
// Returns the response and error, if any.
func ExecuteGetNextCardRequest(t *testing.T, server *httptest.Server, userID uuid.UUID) (*http.Response, error) {
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
	return client.Do(req)
}

// ExecuteSubmitAnswerRequest executes a POST /cards/{id}/answer request against the test server.
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
	return client.Do(req)
}

// ExecuteSubmitAnswerRequestWithRawID executes a POST /cards/{id}/answer request
// with a raw ID string (can be invalid for testing error cases).
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
	return client.Do(req)
}

// AssertCardResponse checks that a response contains a valid card with the expected values.
// Automatically registers cleanup for the response body.
func AssertCardResponse(t *testing.T, resp *http.Response, expectedCard *domain.Card) {
	t.Helper()

	// Register cleanup for the response body
	CleanupResponseBody(t, resp)

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
// Automatically registers cleanup for the response body.
func AssertStatsResponse(t *testing.T, resp *http.Response, expectedStats *domain.UserCardStats) {
	t.Helper()

	// Register cleanup for the response body
	CleanupResponseBody(t, resp)

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
