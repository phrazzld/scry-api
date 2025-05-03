//go:build integration

package api

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/stretchr/testify/require"
)

// SetupCardReviewTestServerWithNextCard creates a test server that returns the specified card
// when a request is made to /api/cards/next.
func SetupCardReviewTestServerWithNextCard(t *testing.T, userID uuid.UUID, card *domain.Card) *httptest.Server {
	t.Helper()

	// Create mock card review service that returns the specified card
	mockService := &card_review.MockCardReviewService{
		GetNextCardToReviewFunc: func(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
	}

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Create JWT service for auth - we can use the real one for tests
	jwtService, err := auth.NewJWTService(auth.DefaultJWTConfig())
	require.NoError(t, err, "Failed to create JWT service")

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Create card handler with mock service
	cardHandler := api.NewCardHandler(mockService, nil, slog.Default())

	// Set up API routes
	router.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/cards/next", cardHandler.GetNextCardToReview)
		})
	})

	// Create server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupCardReviewTestServerWithError creates a test server that returns the specified error
// when a request is made to /api/cards/next.
func SetupCardReviewTestServerWithError(t *testing.T, userID uuid.UUID, err error) *httptest.Server {
	t.Helper()

	// Create mock card review service that returns the specified error
	mockService := &card_review.MockCardReviewService{
		GetNextCardToReviewFunc: func(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
			return nil, err
		},
		SubmitAnswerFunc: func(ctx context.Context, userID, cardID uuid.UUID, answer string) (*domain.UserCardStats, error) {
			return nil, err
		},
	}

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Create JWT service for auth - we can use the real one for tests
	jwtService, err := auth.NewJWTService(auth.DefaultJWTConfig())
	require.NoError(t, err, "Failed to create JWT service")

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Create card handler with mock service
	cardHandler := api.NewCardHandler(mockService, nil, slog.Default())

	// Set up API routes
	router.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/cards/next", cardHandler.GetNextCardToReview)
			r.Post("/cards/{id}/answer", cardHandler.SubmitCardAnswer)
			r.Post("/cards/{id}/postpone", cardHandler.PostponeCard)
		})
	})

	// Create server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupCardReviewTestServerWithAuthError creates a test server that returns the specified auth error
// when validating JWT tokens.
func SetupCardReviewTestServerWithAuthError(t *testing.T, userID uuid.UUID, authError error) *httptest.Server {
	t.Helper()

	// Create mock JWT service that returns the specified error
	mockJWTService := &auth.MockJWTService{
		ValidateTokenFunc: func(ctx context.Context, token string) (*auth.Claims, error) {
			return nil, authError
		},
	}

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Create auth middleware with mock JWT service
	authMiddleware := middleware.NewAuthMiddleware(mockJWTService)

	// Create mock card review service
	mockService := &card_review.MockCardReviewService{}

	// Create card handler with mock service
	cardHandler := api.NewCardHandler(mockService, nil, slog.Default())

	// Set up API routes
	router.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/cards/next", cardHandler.GetNextCardToReview)
			r.Post("/cards/{id}/answer", cardHandler.SubmitCardAnswer)
		})
	})

	// Create server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupCardReviewTestServerWithUpdatedStats creates a test server that returns the specified stats
// when a review answer is submitted.
func SetupCardReviewTestServerWithUpdatedStats(
	t *testing.T,
	userID uuid.UUID,
	stats *domain.UserCardStats,
) *httptest.Server {
	t.Helper()

	// Create mock card review service that returns the specified stats
	mockService := &card_review.MockCardReviewService{
		SubmitAnswerFunc: func(ctx context.Context, userID, cardID uuid.UUID, answer string) (*domain.UserCardStats, error) {
			return stats, nil
		},
	}

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Create JWT service for auth - we can use the real one for tests
	jwtService, err := auth.NewJWTService(auth.DefaultJWTConfig())
	require.NoError(t, err, "Failed to create JWT service")

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Create card handler with mock service
	cardHandler := api.NewCardHandler(mockService, nil, slog.Default())

	// Set up API routes
	router.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Post("/cards/{id}/answer", cardHandler.SubmitCardAnswer)
		})
	})

	// Create server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}
