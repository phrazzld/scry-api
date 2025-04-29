//go:build integration

package main

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/require"
)

// APIServerOptions configures the setup of an API test server.
type APIServerOptions struct {
	// Services
	CardService       service.CardService
	CardReviewService card_review.CardReviewService
	JWTService        auth.JWTService

	// Logger
	Logger *slog.Logger
}

// SetupAPITestServer creates a test server with properly configured API endpoints.
// It configures all API routes with appropriate middleware and handlers.
func SetupAPITestServer(t *testing.T, tx *sql.Tx, opts APIServerOptions) http.Handler {
	t.Helper()

	// Create a logger if not provided
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	// Create JWT service if not provided
	jwtService := opts.JWTService
	if jwtService == nil {
		var err error
		jwtService, err = testutils.CreateTestJWTService()
		require.NoError(t, err, "Failed to create JWT service")
	}

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)
	router.Use(middleware.NewTraceMiddleware(logger))

	// Create stores if services are not provided
	if opts.CardService == nil || opts.CardReviewService == nil {
		// Create stores with the transaction
		userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for test speed
		cardStore := postgres.NewPostgresCardStore(tx, logger)
		statsStore := postgres.NewPostgresUserCardStatsStore(tx, logger)

		// Create SRS service
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err, "Failed to create SRS service")

		// Create repository adapters
		dbConn := cardStore.DB()
		cardRepo := service.NewCardRepositoryAdapter(cardStore, dbConn)
		statsRepo := service.NewStatsRepositoryAdapter(statsStore)

		// Create card service if not provided
		if opts.CardService == nil {
			cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, logger)
			require.NoError(t, err, "Failed to create card service")
			opts.CardService = cardService
		}

		// Create card review service if not provided
		if opts.CardReviewService == nil {
			cardReviewService, err := card_review.NewCardReviewService(
				cardStore,
				statsStore,
				srsService,
				logger,
			)
			require.NoError(t, err, "Failed to create card review service")
			opts.CardReviewService = cardReviewService
		}

		// Create auth configs for the auth handler
		authConfig := &config.AuthConfig{
			JWTSecret:                   "testsecrettestsecrettestsecrettestsecret", // 32+ chars
			BCryptCost:                  4,                                          // Use minimum cost for tests
			TokenLifetimeMinutes:        60,
			RefreshTokenLifetimeMinutes: 1440,
		}
		passwordVerifier := auth.NewBcryptVerifier()
		authHandler := api.NewAuthHandler(
			userStore,
			jwtService,
			passwordVerifier,
			authConfig,
			logger,
		)

		// Set up auth routes (registration, login, refresh)
		router.Route("/api", func(r chi.Router) {
			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.RefreshToken)
		})
	}

	// Create card handler with services
	cardHandler := api.NewCardHandler(opts.CardReviewService, opts.CardService, logger)

	// Set up card API routes
	router.Route("/api", func(r chi.Router) {
		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// Card review endpoints
			r.Get("/cards/next", cardHandler.GetNextReviewCard)
			r.Post("/cards/{id}/answer", cardHandler.SubmitAnswer)

			// Card management endpoints
			r.Put("/cards/{id}", cardHandler.EditCard)
			r.Delete("/cards/{id}", cardHandler.DeleteCard)
			r.Post("/cards/{id}/postpone", cardHandler.PostponeCard)
		})
	})

	return router
}
