package testutils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
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
func CreateTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
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
