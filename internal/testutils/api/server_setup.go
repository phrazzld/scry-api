//go:build integration || test_without_external_deps

package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/stretchr/testify/require"
)

// TestServerOptions contains options for creating a test server.
// This provides a standardized way to configure test servers for different types of API tests.
type TestServerOptions struct {
	// Database transaction for test isolation (required)
	Tx *sql.Tx

	// Auth configuration
	JWTService auth.JWTService // Optional, will use default test JWT service if nil

	// Logging configuration
	Logger *slog.Logger // Optional, will use a no-op logger if nil

	// Additional handler configuration
	ConfigureRoutes func(r chi.Router, cardHandler *api.CardHandler, authMiddleware *middleware.AuthMiddleware)
}

// UUIDValidationMiddleware creates a middleware that validates UUIDs in URL parameters.
// This addresses a common pattern across many API endpoints.
func UUIDValidationMiddleware(paramName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract the ID parameter
			idParam := chi.URLParam(r, paramName)

			// Check if ID is a valid UUID
			_, err := uuid.Parse(idParam)
			if err != nil {
				// Return 400 Bad Request for invalid UUID
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Invalid ID format",
				})
				return
			}

			// Continue to the next handler for valid UUIDs
			next.ServeHTTP(w, r)
		})
	}
}

// SetupTestServer creates a standardized test server with optional configuration.
// This is the primary function for setting up test servers in integration tests.
// It automatically registers cleanup via t.Cleanup().
func SetupTestServer(t *testing.T, options TestServerOptions) *httptest.Server {
	t.Helper()

	// Validate required options
	require.NotNil(t, options.Tx, "Transaction is required for test server setup")

	// Set up logger (use no-op logger if not provided)
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	// Set up JWT service (use default test JWT service if not provided)
	jwtService := options.JWTService
	if jwtService == nil {
		testJwt := auth.RequireTestJWTService(t)
		jwtService = testJwt
	}

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Create stores with the transaction
	cardStore := postgres.NewPostgresCardStore(options.Tx, logger)
	statsStore := postgres.NewPostgresUserCardStatsStore(options.Tx, logger)

	// Create SRS service
	srsService, err := srs.NewDefaultService()
	require.NoError(t, err, "Failed to create SRS service")

	// Create repository adapters for service layer
	dbConn := cardStore.DB()
	cardRepo := service.NewCardRepositoryAdapter(cardStore, dbConn)
	statsRepo := service.NewStatsRepositoryAdapter(statsStore)

	// Create card service
	cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, logger)
	require.NoError(t, err, "Failed to create card service")

	// Create card review service
	cardReviewService, err := card_review.NewCardReviewService(cardStore, statsStore, srsService, logger)
	require.NoError(t, err, "Failed to create card review service")

	// Create card handler
	cardHandler := api.NewCardHandler(cardReviewService, cardService, logger)

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Configure routes if custom configuration is provided
	if options.ConfigureRoutes != nil {
		options.ConfigureRoutes(router, cardHandler, authMiddleware)
	} else {
		// Set up default API routes
		router.Route("/api", func(r chi.Router) {
			// Card management endpoints (authenticated)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.Authenticate)

				// Get next card for review
				r.Get("/cards/next", cardHandler.GetNextReviewCard)

				// Card CRUD operations
				r.Route("/cards/{id}", func(sr chi.Router) {
					// UUID validation middleware
					sr.Use(UUIDValidationMiddleware("id"))

					// Card routes
					// sr.Get("/", cardHandler.GetCard) // GetCard method does not exist yet
					sr.Put("/", cardHandler.EditCard)
					sr.Delete("/", cardHandler.DeleteCard)
					sr.Post("/answer", cardHandler.SubmitAnswer)
					sr.Post("/postpone", cardHandler.PostponeCard)
				})
			})
		})
	}

	// Create server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupCardManagementTestServer creates a test server specifically for card management API tests.
// This is a convenience wrapper around SetupTestServer with pre-configured routes for card management.
func SetupCardManagementTestServer(t *testing.T, tx *sql.Tx) *httptest.Server {
	t.Helper()

	return SetupTestServer(t, TestServerOptions{
		Tx: tx,
		ConfigureRoutes: func(r chi.Router, cardHandler *api.CardHandler, authMiddleware *middleware.AuthMiddleware) {
			r.Route("/api", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(authMiddleware.Authenticate)

					// Card management endpoints
					r.Get("/cards/next", cardHandler.GetNextReviewCard)
					// CreateCard method does not exist yet, comment out for now
					// r.Post("/cards", cardHandler.CreateCard)

					r.Route("/cards/{id}", func(sr chi.Router) {
						// UUID validation middleware
						sr.Use(UUIDValidationMiddleware("id"))

						// Card routes
						// sr.Get("/", cardHandler.GetCard) // GetCard method does not exist yet
						sr.Put("/", cardHandler.EditCard)
						sr.Delete("/", cardHandler.DeleteCard)
						sr.Post("/answer", cardHandler.SubmitAnswer)
						sr.Post("/postpone", cardHandler.PostponeCard)
					})
				})
			})
		},
	})
}

// SetupCardReviewTestServer creates a test server specifically for card review API tests.
// This is a convenience wrapper around SetupTestServer with pre-configured routes for card review.
func SetupCardReviewTestServer(t *testing.T, tx *sql.Tx) *httptest.Server {
	t.Helper()

	return SetupTestServer(t, TestServerOptions{
		Tx: tx,
		ConfigureRoutes: func(r chi.Router, cardHandler *api.CardHandler, authMiddleware *middleware.AuthMiddleware) {
			r.Route("/api", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(authMiddleware.Authenticate)

					// Card review endpoints
					r.Get("/cards/next", cardHandler.GetNextReviewCard)

					r.Route("/cards/{id}", func(sr chi.Router) {
						// UUID validation middleware
						sr.Use(UUIDValidationMiddleware("id"))

						// Answer route
						sr.Post("/answer", cardHandler.SubmitAnswer)
					})
				})
			})
		},
	})
}

// SetupAuthTestServer creates a test server specifically for authentication API tests.
// This is a convenience wrapper around SetupTestServer with pre-configured routes for authentication.
func SetupAuthTestServer(t *testing.T, tx *sql.Tx) *httptest.Server {
	t.Helper()

	// Create logger
	logger := slog.New(slog.NewTextHandler(nil, nil))

	// Create stores
	userStore := postgres.NewPostgresUserStore(tx, 4) // Lower cost for tests

	// Create JWT service
	jwtService := auth.RequireTestJWTService(t)

	// Create password verifier
	passwordVerifier := auth.NewBcryptVerifier()

	// Create config
	jwtConfig := auth.DefaultJWTConfig()

	// Create auth handler
	authHandler := api.NewAuthHandler(
		userStore,
		jwtService,
		passwordVerifier,
		&jwtConfig,
		logger,
	)

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Create router
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Set up routes
	router.Route("/api", func(r chi.Router) {
		// Public auth endpoints
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.RefreshToken)

		// Protected routes for testing auth
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/user/profile", func(w http.ResponseWriter, r *http.Request) {
				userID, ok := middleware.GetUserID(r)
				if !ok {
					shared.RespondWithError(w, r, http.StatusUnauthorized, "User ID not found in context")
					return
				}
				shared.RespondWithJSON(w, r, http.StatusOK, map[string]interface{}{
					"user_id": userID,
					"message": "Profile data",
				})
			})
		})
	})

	// Create server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupAPITestServer is a backward-compatible alias for SetupTestServer.
// This function exists to ease the transition from old code to the new standardized approach.
// New code should use SetupTestServer directly.
func SetupAPITestServer(t *testing.T, tx *sql.Tx, options TestServerOptions) *httptest.Server {
	t.Helper()
	options.Tx = tx // Ensure tx is set correctly
	return SetupTestServer(t, options)
}
