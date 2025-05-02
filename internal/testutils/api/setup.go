// Package api provides testing utilities for API testing,
// including server setup, request handling, and authentication helpers.
package api

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
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
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// APIServerOptions configures the setup of an API test server.
type APIServerOptions struct {
	// Services
	CardService       service.CardService
	CardReviewService card_review.CardReviewService
	JWTService        auth.JWTService

	// Stores
	UserStore  store.UserStore
	CardStore  store.CardStore
	StatsStore store.UserCardStatsStore
	MemoStore  store.MemoStore

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
		// Create a test JWT service directly
		authConfig := &config.AuthConfig{
			JWTSecret:                   "testsecrettestsecrettestsecrettestsecret",
			TokenLifetimeMinutes:        60,
			RefreshTokenLifetimeMinutes: 1440,
		}
		jwtService, err = auth.NewJWTService(*authConfig)
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
	var userStore store.UserStore
	var cardStore store.CardStore
	var statsStore store.UserCardStatsStore
	var cardService service.CardService
	var cardReviewService card_review.CardReviewService

	// Use provided stores or create new ones
	if opts.UserStore != nil {
		userStore = opts.UserStore
	} else {
		userStore = postgres.NewPostgresUserStore(tx, 4) // Low cost for test speed
	}

	if opts.CardStore != nil {
		cardStore = opts.CardStore
	} else {
		cardStore = postgres.NewPostgresCardStore(tx, logger)
	}

	if opts.StatsStore != nil {
		statsStore = opts.StatsStore
	} else {
		statsStore = postgres.NewPostgresUserCardStatsStore(tx, logger)
	}

	// MemoStore isn't used in our current setup
	// Removed for simplicity

	// Create services if not provided
	if opts.CardService == nil || opts.CardReviewService == nil {
		// Create SRS service
		srsService, err := srs.NewDefaultService()
		require.NoError(t, err, "Failed to create SRS service")

		// Create repository adapters
		dbConn := cardStore.DB()
		cardRepo := service.NewCardRepositoryAdapter(cardStore, dbConn)
		statsRepo := service.NewStatsRepositoryAdapter(statsStore)

		// Create card service if not provided
		if opts.CardService == nil {
			cardService, err = service.NewCardService(cardRepo, statsRepo, srsService, logger)
			require.NoError(t, err, "Failed to create card service")
		} else {
			cardService = opts.CardService
		}

		// Create card review service if not provided
		if opts.CardReviewService == nil {
			cardReviewService, err = card_review.NewCardReviewService(
				cardStore,
				statsStore,
				srsService,
				logger,
			)
			require.NoError(t, err, "Failed to create card review service")
		} else {
			cardReviewService = opts.CardReviewService
		}
	} else {
		cardService = opts.CardService
		cardReviewService = opts.CardReviewService
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

	// Create card and memo handlers
	cardHandler := api.NewCardHandler(cardReviewService, cardService, logger)

	// Create the memo service with minimal requirements (no task runner or event emitter for testing)
	memoService, err := service.NewMemoService(nil, nil, nil, logger)
	require.NoError(t, err, "Failed to create memo service")
	memoHandler := api.NewMemoHandler(memoService, logger)

	// Set up routes
	router.Route("/api", func(r chi.Router) {
		// Auth endpoints
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.RefreshToken)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// Memo endpoints
			r.Post("/memos", memoHandler.CreateMemo)

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

// CreateTestServer creates a httptest.Server with the given handler.
// This is a convenience wrapper around httptest.NewServer.
func CreateTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// SetupCardManagementTestServer sets up a test server with the card management endpoints.
func SetupCardManagementTestServer(t *testing.T, tx *sql.Tx) *httptest.Server {
	t.Helper()

	// Create the router
	router, err := SetupCardManagementTestRouter(t, tx)
	require.NoError(t, err, "Failed to set up test router")

	// Create and return a test server
	return httptest.NewServer(router)
}

// SetupCardManagementTestRouter creates a router with card management endpoints for testing.
func SetupCardManagementTestRouter(t *testing.T, tx *sql.Tx) (http.Handler, error) {
	t.Helper()

	// Create test logger that writes to discarded output
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	// Create stores using the transaction
	userStore := postgres.NewPostgresUserStore(tx, bcrypt.MinCost)
	cardStore := postgres.NewPostgresCardStore(tx, testLogger)
	userCardStatsStore := postgres.NewPostgresUserCardStatsStore(tx, testLogger)

	// Create SRS service with default parameters
	srsService, err := srs.NewDefaultService()
	if err != nil {
		return nil, err
	}

	// Create card repository adapter that implements service.CardRepository
	// For testing, we can pass nil as the second parameter since we're using transactions directly
	cardRepo := service.NewCardRepositoryAdapter(cardStore, nil)

	// Create stats repository adapter that implements service.StatsRepository
	statsRepo := service.NewStatsRepositoryAdapter(userCardStatsStore)

	// Create card service
	cardService, err := service.NewCardService(cardRepo, statsRepo, srsService, testLogger)
	if err != nil {
		return nil, err
	}

	// Create card review service
	cardReviewService, err := card_review.NewCardReviewService(
		cardStore,
		userCardStatsStore,
		srsService,
		testLogger,
	)
	if err != nil {
		return nil, err
	}

	// Create JWT service for authentication
	authConfig := &config.AuthConfig{
		JWTSecret:                   "testsecrettestsecrettestsecrettestsecret",
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}
	var jwtService auth.JWTService
	jwtService, err = auth.NewJWTService(*authConfig)
	if err != nil {
		return nil, err
	}

	// Create password verifier for auth
	passwordVerifier := auth.NewBcryptVerifier()

	// Create router with standard middleware
	router := chi.NewRouter()

	// Add standard middleware
	router.Use(middleware.NewTraceMiddleware(testLogger))

	// Create a test auth config
	testAuthConfig := &config.AuthConfig{
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
		testAuthConfig,
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

// CreateTestUser creates a test user in the database within the given transaction
// and returns the created user's ID.
func CreateTestUser(t *testing.T, tx *sql.Tx) uuid.UUID {
	t.Helper()

	// Create a user store with the transaction
	userStore := postgres.NewPostgresUserStore(tx, bcrypt.MinCost)

	// Generate a unique email for this test to avoid conflicts
	userEmail := "test_" + uuid.New().String() + "@example.com"

	// Hash the password with minimal cost for testing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err, "Failed to hash password")

	// Create a new user
	now := time.Now().UTC()
	user := &domain.User{
		ID:        uuid.New(),
		Email:     userEmail,
		Password:  string(hashedPassword),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save the user
	err = userStore.Create(context.Background(), user)
	require.NoError(t, err, "Failed to create test user")

	return user.ID
}

// GetAuthToken generates an authentication token for testing.
func GetAuthToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()

	// Create a JWT service with a test secret
	authConfig := &config.AuthConfig{
		JWTSecret:                   "testsecrettestsecrettestsecrettestsecret",
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}
	jwtService, err := auth.NewJWTService(*authConfig)
	require.NoError(t, err, "Failed to create JWT service")

	// Generate a token
	token, err := jwtService.GenerateToken(context.Background(), userID)
	require.NoError(t, err, "Failed to generate auth token")

	return token
}
