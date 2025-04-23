package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CreateAuthComponents creates JWT service and password verifier for testing.
// Returns auth config, jwt service, password verifier, and any error.
func CreateAuthComponents(t *testing.T) (config.AuthConfig, auth.JWTService, auth.PasswordVerifier, error) {
	t.Helper()
	authConfig := config.AuthConfig{
		JWTSecret:                   "test-jwt-secret-thatis32characterslong",
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}
	jwtService, err := auth.NewJWTService(authConfig)
	if err != nil {
		return authConfig, nil, nil, err
	}
	passwordVerifier := auth.NewBcryptVerifier()
	return authConfig, jwtService, passwordVerifier, nil
}

// CreateAPIHandlers creates API handlers and middleware for testing.
// Takes the required dependencies and returns initialized handlers.
func CreateAPIHandlers(
	t *testing.T,
	userStore store.UserStore,
	jwtService auth.JWTService,
	passwordVerifier auth.PasswordVerifier,
	authConfig config.AuthConfig,
	memoService service.MemoService,
) (*api.AuthHandler, *api.MemoHandler, *authmiddleware.AuthMiddleware) {
	t.Helper()
	authHandler := api.NewAuthHandler(userStore, jwtService, passwordVerifier, &authConfig)
	memoHandler := api.NewMemoHandler(memoService)
	authMiddleware := authmiddleware.NewAuthMiddleware(jwtService)
	return authHandler, memoHandler, authMiddleware
}

// CreateTestRouter creates a Chi router with standard middleware applied.
// This provides a consistent router setup for all API tests.
func CreateTestRouter(t *testing.T) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()

	// Apply standard middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	return r
}

// CreateTestServer creates a httptest server with the given handler.
// This is a simple helper to reduce boilerplate in tests.
// Automatically registers cleanup via t.Cleanup() so callers don't need to manually close the server.
func CreateTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

// SetupAuthRoutes configures standard auth routes on the provided router.
// This ensures consistency in how auth routes are set up across tests.
func SetupAuthRoutes(
	t *testing.T,
	r chi.Router,
	authHandler *api.AuthHandler,
	memoHandler *api.MemoHandler,
	authMiddleware *authmiddleware.AuthMiddleware,
) {
	t.Helper()
	r.Route("/api", func(r chi.Router) {
		// Auth endpoints
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			// Memo endpoints
			r.Post("/memos", memoHandler.CreateMemo)
		})
	})
}

// CreatePostgresUserStore creates a PostgresUserStore with default test settings.
// This is a convenience function to create a user store for testing.
func CreatePostgresUserStore(db store.DBTX) *postgres.PostgresUserStore {
	return postgres.NewPostgresUserStore(db, 10) // BCrypt cost = 10 for faster tests
}

//------------------------------------------------------------------------------
// Card Review Test Helpers
//------------------------------------------------------------------------------

// CardOption is a function that configures a Card for testing.
type CardOption func(*domain.Card)

// WithCardID sets a specific ID for the test card.
func WithCardID(id uuid.UUID) CardOption {
	return func(c *domain.Card) {
		c.ID = id
	}
}

// WithCardUserID sets a specific user ID for the test card.
func WithCardUserID(userID uuid.UUID) CardOption {
	return func(c *domain.Card) {
		c.UserID = userID
	}
}

// WithCardMemoID sets a specific memo ID for the test card.
func WithCardMemoID(memoID uuid.UUID) CardOption {
	return func(c *domain.Card) {
		c.MemoID = memoID
	}
}

// WithCardContent sets the content for the test card using a map.
// The map will be marshaled to JSON.
func WithCardContent(content map[string]interface{}) CardOption {
	return func(c *domain.Card) {
		contentBytes, _ := json.Marshal(content)
		c.Content = contentBytes
	}
}

// WithCardCreatedAt sets the creation timestamp for the test card.
func WithCardCreatedAt(createdAt time.Time) CardOption {
	return func(c *domain.Card) {
		c.CreatedAt = createdAt
	}
}

// WithCardUpdatedAt sets the update timestamp for the test card.
func WithCardUpdatedAt(updatedAt time.Time) CardOption {
	return func(c *domain.Card) {
		c.UpdatedAt = updatedAt
	}
}

// CreateCardForAPITest creates a Card instance for API testing with default values.
// Options can be passed to customize the card.
func CreateCardForAPITest(t *testing.T, opts ...CardOption) *domain.Card {
	t.Helper()

	now := time.Now().UTC()
	userID := uuid.New()
	memoID := uuid.New()
	cardID := uuid.New()

	// Default test card content
	defaultContent := map[string]interface{}{
		"front": "What is the capital of France?",
		"back":  "Paris",
	}
	contentBytes, err := json.Marshal(defaultContent)
	require.NoError(t, err)

	// Create card with default values
	card := &domain.Card{
		ID:        cardID,
		UserID:    userID,
		MemoID:    memoID,
		Content:   contentBytes,
		CreatedAt: now.Add(-24 * time.Hour),
		UpdatedAt: now.Add(-24 * time.Hour),
	}

	// Apply options
	for _, opt := range opts {
		opt(card)
	}

	return card
}

// StatsOption is a function that configures UserCardStats for testing.
type StatsOption func(*domain.UserCardStats)

// WithStatsUserID sets a specific user ID for the test stats.
func WithStatsUserID(userID uuid.UUID) StatsOption {
	return func(s *domain.UserCardStats) {
		s.UserID = userID
	}
}

// WithStatsCardID sets a specific card ID for the test stats.
func WithStatsCardID(cardID uuid.UUID) StatsOption {
	return func(s *domain.UserCardStats) {
		s.CardID = cardID
	}
}

// WithStatsInterval sets the interval for the test stats.
func WithStatsInterval(interval int) StatsOption {
	return func(s *domain.UserCardStats) {
		s.Interval = interval
	}
}

// WithStatsEaseFactor sets the ease factor for the test stats.
func WithStatsEaseFactor(easeFactor float64) StatsOption {
	return func(s *domain.UserCardStats) {
		s.EaseFactor = easeFactor
	}
}

// WithStatsConsecutiveCorrect sets the consecutive correct count for the test stats.
func WithStatsConsecutiveCorrect(count int) StatsOption {
	return func(s *domain.UserCardStats) {
		s.ConsecutiveCorrect = count
	}
}

// WithStatsLastReviewedAt sets the last reviewed timestamp for the test stats.
func WithStatsLastReviewedAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.LastReviewedAt = timestamp
	}
}

// WithStatsNextReviewAt sets the next review timestamp for the test stats.
func WithStatsNextReviewAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.NextReviewAt = timestamp
	}
}

// WithStatsReviewCount sets the review count for the test stats.
func WithStatsReviewCount(count int) StatsOption {
	return func(s *domain.UserCardStats) {
		s.ReviewCount = count
	}
}

// WithStatsCreatedAt sets the creation timestamp for the test stats.
func WithStatsCreatedAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.CreatedAt = timestamp
	}
}

// WithStatsUpdatedAt sets the update timestamp for the test stats.
func WithStatsUpdatedAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.UpdatedAt = timestamp
	}
}

// CreateStatsForAPITest creates a UserCardStats instance for API testing with default values.
// Options can be passed to customize the stats.
func CreateStatsForAPITest(t *testing.T, opts ...StatsOption) *domain.UserCardStats {
	t.Helper()

	now := time.Now().UTC()
	userID := uuid.New()
	cardID := uuid.New()

	// Create stats with default values
	stats := &domain.UserCardStats{
		UserID:             userID,
		CardID:             cardID,
		Interval:           1,
		EaseFactor:         2.5,
		ConsecutiveCorrect: 1,
		LastReviewedAt:     now,
		NextReviewAt:       now.Add(24 * time.Hour),
		ReviewCount:        1,
		CreatedAt:          now.Add(-24 * time.Hour),
		UpdatedAt:          now,
	}

	// Apply options
	for _, opt := range opts {
		opt(stats)
	}

	return stats
}

//------------------------------------------------------------------------------
// Card Review API Test Helpers
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

	// Create JWT service mock
	jwtMock := &mocks.MockJWTService{}

	if opts.ValidateTokenFn != nil {
		// Custom validation function
		jwtMock.ValidateTokenFn = opts.ValidateTokenFn
	} else {
		// Default validation function
		jwtMock.ValidateTokenFn = func(ctx context.Context, token string) (*auth.Claims, error) {
			return &auth.Claims{
				UserID: opts.UserID,
			}, nil
		}
	}

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Create auth middleware
	authMiddleware := authmiddleware.NewAuthMiddleware(jwtMock)

	// Create card handler
	cardHandler := api.NewCardHandler(cardReviewMock, nil) // nil logger uses default

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

// ExecuteGetNextCardRequest executes a GET /cards/next request against the test server.
// Returns the response and error, if any.
func ExecuteGetNextCardRequest(t *testing.T, server *httptest.Server) (*http.Response, error) {
	t.Helper()

	// Create request
	req, err := http.NewRequest("GET", server.URL+"/api/cards/next", nil)
	if err != nil {
		return nil, err
	}

	// Add auth header
	req.Header.Set("Authorization", "Bearer test-token")

	// Execute request
	client := &http.Client{}
	return client.Do(req)
}

// ExecuteSubmitAnswerRequest executes a POST /cards/{id}/answer request against the test server.
// Returns the response and error, if any.
func ExecuteSubmitAnswerRequest(
	t *testing.T,
	server *httptest.Server,
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

	// Add headers
	req.Header.Set("Authorization", "Bearer test-token")
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

	// Add headers
	req.Header.Set("Authorization", "Bearer test-token")
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

// CleanupResponseBody registers a cleanup function to close the response body
// to prevent resource leaks. Should be used in tests when receiving an HTTP response.
func CleanupResponseBody(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp != nil && resp.Body != nil {
		t.Cleanup(func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Warning: failed to close response body: %v", err)
			}
		})
	}
}

// AssertErrorResponse checks that a response contains an error with the expected status code and message.
// Automatically registers cleanup for the response body.
func AssertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedErrorMsgPart string) {
	t.Helper()

	// Register cleanup for the response body
	CleanupResponseBody(t, resp)

	// Check status code
	assert.Equal(
		t,
		expectedStatus,
		resp.StatusCode,
		"Expected status code %d but got %d",
		expectedStatus,
		resp.StatusCode,
	)

	// For 204 No Content, body should be empty
	if expectedStatus == http.StatusNoContent {
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")
		assert.Empty(t, body, "Expected empty body for 204 No Content")
		return
	}

	// Read body for other status codes
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Parse error response
	var errResp shared.ErrorResponse
	err = json.Unmarshal(body, &errResp)
	require.NoError(t, err, "Failed to unmarshal error response: %s", string(body))

	// Verify error message
	assert.Contains(t, errResp.Error, expectedErrorMsgPart,
		"Error message should contain '%s' but got '%s'", expectedErrorMsgPart, errResp.Error)
}

// ExecuteInvalidJSONRequest sends a request with an invalid JSON body to test error handling.
func ExecuteInvalidJSONRequest(t *testing.T, server *httptest.Server, method, path string) (*http.Response, error) {
	t.Helper()

	// Create request with invalid JSON body
	req, err := http.NewRequest(
		method,
		server.URL+path,
		bytes.NewBuffer([]byte(`{"invalid_json": true,`)), // Malformed JSON (missing closing bracket)
	)
	require.NoError(t, err, "Failed to create request with invalid JSON")

	// Add headers
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	return client.Do(req)
}

// ExecuteEmptyBodyRequest sends a request with an empty body to test validation.
func ExecuteEmptyBodyRequest(t *testing.T, server *httptest.Server, method, path string) (*http.Response, error) {
	t.Helper()

	// Create request with empty body
	req, err := http.NewRequest(
		method,
		server.URL+path,
		bytes.NewBuffer([]byte(`{}`)), // Empty JSON object
	)
	require.NoError(t, err, "Failed to create request with empty body")

	// Add headers
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	return client.Do(req)
}

// ExecuteCustomBodyRequest sends a request with a custom JSON body for testing.
func ExecuteCustomBodyRequest(
	t *testing.T,
	server *httptest.Server,
	method, path string,
	body interface{},
) (*http.Response, error) {
	t.Helper()

	// Marshal the body to JSON
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err, "Failed to marshal request body")

	// Create request
	req, err := http.NewRequest(
		method,
		server.URL+path,
		bytes.NewBuffer(bodyBytes),
	)
	require.NoError(t, err, "Failed to create request")

	// Add headers
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	return client.Do(req)
}
