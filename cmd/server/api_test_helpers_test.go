//go:build integration

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// createTestUser creates a test user in the database within the given transaction
// and returns the created user's ID.
func createTestUser(t *testing.T, tx *sql.Tx) uuid.UUID {
	t.Helper()

	// Create a user store with the transaction
	userStore := postgres.NewPostgresUserStore(tx, bcrypt.MinCost)

	// Generate a unique email for this test to avoid conflicts
	userEmail := "test_" + uuid.New().String() + "@example.com"

	// Hash the password with minimal cost for testing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err, "Failed to hash password")

	// Create a new user
	user := &domain.User{
		ID:        uuid.New(),
		Email:     userEmail,
		Password:  string(hashedPassword),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Save the user
	err = userStore.Create(context.Background(), user)
	require.NoError(t, err, "Failed to create test user")

	return user.ID
}

// createTestCard creates a test card in the database within the given transaction
// and returns the created card.
func createTestCard(t *testing.T, tx *sql.Tx, userID uuid.UUID) *domain.Card {
	t.Helper()

	// Create a test logger that writes to discarded output
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	// Create a card store with the transaction
	cardStore := postgres.NewPostgresCardStore(tx, testLogger)

	// Create a memo for the card
	memoStore := postgres.NewPostgresMemoStore(tx, testLogger)
	memo := &domain.Memo{
		ID:        uuid.New(),
		UserID:    userID,
		Text:      "Test memo content",
		Status:    domain.MemoStatusCompleted,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := memoStore.Create(context.Background(), memo)
	require.NoError(t, err, "Failed to create test memo")

	// Create a card
	cardID := uuid.New()
	card := &domain.Card{
		ID:        cardID,
		UserID:    userID,
		MemoID:    memo.ID,
		Content:   json.RawMessage(`{"question": "Test question", "answer": "Test answer"}`),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Create a user_card_stats entry for this card
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, testLogger)
	stats, err := domain.NewUserCardStats(userID, cardID)
	require.NoError(t, err, "Failed to create user card stats object")
	err = statsStore.Create(context.Background(), stats)
	require.NoError(t, err, "Failed to create test user card stats")

	// Save the card
	err = cardStore.CreateMultiple(context.Background(), []*domain.Card{card})
	require.NoError(t, err, "Failed to create test card")

	return card
}

// getCardByID retrieves a card by its ID from the database within the given transaction.
func getCardByID(tx *sql.Tx, cardID uuid.UUID) (*domain.Card, error) {
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cardStore := postgres.NewPostgresCardStore(tx, testLogger)
	return cardStore.GetByID(context.Background(), cardID)
}

// getAuthToken generates an authentication token for testing.
func getAuthToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()

	// Create a JWT service with a test secret
	jwtService := testutils.NewTestJWTService()

	// Generate a token
	token, err := jwtService.GenerateToken(context.Background(), userID)
	require.NoError(t, err, "Failed to generate auth token")

	return token
}

// setupCardManagementTestServer sets up a test server with the card management endpoints.
func setupCardManagementTestServer(t *testing.T, tx *sql.Tx) *httptest.Server {
	t.Helper()

	// Create the router
	router, err := setupCardManagementTestRouter(t, tx)
	require.NoError(t, err, "Failed to set up test router")

	// Create and return a test server
	return httptest.NewServer(router)
}

// getUserCardStats retrieves user card statistics for a given card and user.
func getUserCardStats(t *testing.T, tx *sql.Tx, userID, cardID uuid.UUID) *domain.UserCardStats {
	t.Helper()

	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, testLogger)
	stats, err := statsStore.Get(context.Background(), userID, cardID)

	if err != nil && errors.Is(err, store.ErrUserCardStatsNotFound) {
		return nil
	}

	require.NoError(t, err, "Failed to get user card stats")
	return stats
}

// setupCardManagementTestRouter creates a router with card management endpoints for testing.
func setupCardManagementTestRouter(t *testing.T, tx *sql.Tx) (http.Handler, error) {
	t.Helper()

	// Create test logger that writes to discarded output
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	// Create stores using the transaction
	userStore := postgres.NewPostgresUserStore(tx, bcrypt.MinCost)
	cardStore := postgres.NewPostgresCardStore(tx, testLogger)
	statsStore := postgres.NewPostgresUserCardStatsStore(tx, testLogger)

	// Create SRS service with default parameters
	srsService, err := srs.NewDefaultService()
	if err != nil {
		return nil, err
	}

	// Create card repository adapter that implements service.CardRepository
	// For testing, we can pass nil as the second parameter since we're using transactions directly
	cardRepo := service.NewCardRepositoryAdapter(cardStore, nil)

	// Create stats repository adapter that implements service.StatsRepository
	statsRepo := service.NewStatsRepositoryAdapter(statsStore)

	// Create card service
	cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, testLogger)
	if err != nil {
		return nil, err
	}

	// Create card review service
	cardReviewService, err := card_review.NewCardReviewService(
		cardStore,
		statsStore,
		srsService,
		testLogger,
	)
	if err != nil {
		return nil, err
	}

	// Create JWT service for authentication
	jwtService := testutils.NewTestJWTService()

	// Create password verifier for auth
	passwordVerifier := auth.NewBcryptVerifier()

	// Create router with standard middleware
	router := chi.NewRouter()

	// Add standard middleware
	router.Use(middleware.NewTraceMiddleware(testLogger))

	// Create a test auth config
	authConfig := &config.AuthConfig{
		JWTSecret:                   "testsecrettestsecrettestsecrettestsecret", // 32+ chars
		BCryptCost:                  4,                                          // Use minimum cost for tests
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}

	// Create handlers
	authHandler := api.NewAuthHandler(
		userStore,
		jwtService,
		passwordVerifier,
		authConfig,
		testLogger,
	)
	cardHandler := api.NewCardHandler(cardReviewService, cardService, testLogger)

	// Configure authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Set up routes
	router.Route("/api", func(r chi.Router) {
		// Auth endpoints
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.RefreshToken)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// Card review endpoints
			r.Get("/cards/next", cardHandler.GetNextReviewCard)
			r.Post("/cards/{id}/answer", cardHandler.SubmitAnswer)

			// Card management endpoints (our test targets)
			r.Put("/cards/{id}", cardHandler.EditCard)
			r.Delete("/cards/{id}", cardHandler.DeleteCard)
			r.Post("/cards/{id}/postpone", cardHandler.PostponeCard)
		})
	})

	return router, nil
}
