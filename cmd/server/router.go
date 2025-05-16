//go:build exported_core_functions

package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/phrazzld/scry-api/internal/api"
	apiMiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
)

// setupRouter creates and configures the application router with all routes and middleware.
// It accepts the application dependencies to create handlers and register routes.
// Returns the configured router.
func (app *application) setupRouter() http.Handler {
	// Create a router
	r := chi.NewRouter()

	// Apply standard middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(
		apiMiddleware.NewTraceMiddleware(app.logger),
	) // Add trace IDs for improved error handling

	// Create API handlers using the application's services
	authHandler := api.NewAuthHandler(
		app.userStore,
		app.jwtService,
		app.passwordVerifier,
		&app.config.Auth,
		app.logger,
	)
	authMiddleware := apiMiddleware.NewAuthMiddleware(app.jwtService)

	// Use memo service from application dependencies
	memoHandler := api.NewMemoHandler(app.memoService, app.logger)

	// Use card service directly from application dependencies
	cardHandler := api.NewCardHandler(app.cardReviewService, app.cardService, app.logger)

	// Register routes
	r.Route("/api", func(r chi.Router) {
		// Authentication endpoints (public)
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

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			app.logger.Error("Failed to write health check response", "error", err)
		}
	})

	return r
}
