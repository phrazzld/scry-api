//go:build integration

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
)

// SetupCardReviewTestServerWithNextCard creates a test server that returns the specified card
// when a request is made to /api/cards/next.
func SetupCardReviewTestServerWithNextCard(t *testing.T, userID uuid.UUID, card *domain.Card) *httptest.Server {
	t.Helper()

	// Create a handler that directly responds with the card
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For GET /cards/next endpoint
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/cards/next") {
			// If card is nil, we should return a "no cards due" error
			if card == nil {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Return the card as a CardResponse
			cardResp := api.CardResponse{
				ID:        card.ID.String(),
				UserID:    card.UserID.String(),
				MemoID:    card.MemoID.String(),
				CreatedAt: card.CreatedAt,
				UpdatedAt: card.UpdatedAt,
			}

			// Unmarshal content
			var content interface{}
			if err := json.Unmarshal(card.Content, &content); err != nil {
				content = string(card.Content)
			}
			cardResp.Content = content

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cardResp)
			return
		}

		// Default 404 for other routes
		http.NotFound(w, r)
	})

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Set up API routes
	router.Route("/api", func(r chi.Router) {
		r.Get("/cards/next", handler)

		// Add specific handlers for paths with UUID parameters
		r.Route("/cards/{id}", func(sr chi.Router) {
			sr.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Extract the ID parameter
					idParam := chi.URLParam(r, "id")

					// Check if ID is a valid UUID
					_, err := uuid.Parse(idParam)
					if err != nil {
						// Return 400 Bad Request for invalid UUID
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						json.NewEncoder(w).Encode(shared.ErrorResponse{
							Error: "Invalid ID",
						})
						return
					}

					// Continue to the next handler for valid UUIDs
					next.ServeHTTP(w, r)
				})
			})

			// Add routes that will be matched after the UUID validation middleware
			sr.Post("/answer", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// For validation tests, return 400 Bad Request with appropriate error
				var outcome struct {
					Outcome string `json:"outcome"`
				}

				// Try to decode the request body
				err := json.NewDecoder(r.Body).Decode(&outcome)
				if err != nil {
					// Invalid JSON format
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Validation error: invalid JSON format",
					})
					return
				}

				// Check if outcome is empty (required field validation)
				if outcome.Outcome == "" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Outcome: required field",
					})
					return
				}

				// Check if outcome is valid (enum validation)
				validOutcomes := []string{"again", "hard", "good", "easy"}
				isValid := false
				for _, valid := range validOutcomes {
					if outcome.Outcome == valid {
						isValid = true
						break
					}
				}

				if !isValid {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Outcome: invalid value",
					})
					return
				}

				// For valid cases, just return 404 Not Found (since this is just a test stub)
				http.NotFound(w, r)
			}))
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
// when a request is made to /api/cards/next or /api/cards/{id}/answer.
func SetupCardReviewTestServerWithError(t *testing.T, userID uuid.UUID, err error) *httptest.Server {
	t.Helper()

	// Create a handler that directly responds with the expected status code and error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Map error to status code
		var statusCode int
		var errorMsg string

		switch {
		case errors.Is(err, card_review.ErrNoCardsDue):
			statusCode = http.StatusNoContent
			w.WriteHeader(statusCode)
			return
		case errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrExpiredToken):
			statusCode = http.StatusUnauthorized
			errorMsg = "Invalid token"
		case errors.Is(err, card_review.ErrCardNotFound):
			statusCode = http.StatusNotFound
			errorMsg = "Card not found"
		case errors.Is(err, card_review.ErrCardNotOwned):
			statusCode = http.StatusForbidden
			errorMsg = "You do not own this card"
		case errors.Is(err, card_review.ErrInvalidAnswer):
			statusCode = http.StatusBadRequest
			errorMsg = "Invalid answer"
		default:
			statusCode = http.StatusInternalServerError
			if r.URL.Path == "/api/cards/next" {
				errorMsg = "Failed to get next review card"
			} else {
				errorMsg = "Failed to submit answer"
			}
		}

		// Return JSON error response
		errResp := shared.ErrorResponse{
			Error: errorMsg,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(errResp)
	})

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Set up API routes
	router.Route("/api", func(r chi.Router) {
		r.Get("/cards/next", handler)

		// Add a specific handler for invalid UUIDs to properly return a 400 Bad Request
		r.Post("/cards/{id}/answer", func(w http.ResponseWriter, r *http.Request) {
			// Extract the ID parameter
			idParam := chi.URLParam(r, "id")

			// Check if ID is a valid UUID
			_, err := uuid.Parse(idParam)
			if err != nil {
				// Return 400 Bad Request for invalid UUID
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Invalid ID",
				})
				return
			}

			// Process valid UUID with the standard handler
			handler(w, r)
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

	// Create a handler that directly responds with unauthorized status and the specified error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 401 Unauthorized with the auth error
		statusCode := http.StatusUnauthorized

		// Create an error message in the expected format
		var errorMsg string
		if errors.Is(authError, auth.ErrInvalidToken) {
			errorMsg = "Invalid token"
		} else {
			errorMsg = authError.Error()
		}

		// Return JSON error response
		errResp := shared.ErrorResponse{
			Error: errorMsg,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(errResp)
	})

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Set up API routes without auth middleware (since we're simulating auth failure)
	router.Route("/api", func(r chi.Router) {
		r.Get("/cards/next", handler)
		r.Post("/cards/{id}/answer", handler)
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

	// Create a handler that directly responds with the stats
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For POST /cards/{id}/answer endpoint
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/cards/") && strings.HasSuffix(r.URL.Path, "/answer") {
			// Return the stats as a UserCardStatsResponse
			statsResp := api.UserCardStatsResponse{
				UserID:             stats.UserID.String(),
				CardID:             stats.CardID.String(),
				Interval:           stats.Interval,
				EaseFactor:         stats.EaseFactor,
				ConsecutiveCorrect: stats.ConsecutiveCorrect,
				ReviewCount:        stats.ReviewCount,
				LastReviewedAt:     stats.LastReviewedAt,
				NextReviewAt:       stats.NextReviewAt,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(statsResp)
			return
		}

		// Default 404 for other routes
		http.NotFound(w, r)
	})

	// Create router with standard middleware
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Set up API routes
	router.Route("/api", func(r chi.Router) {
		// Add a specific handler for invalid UUIDs to properly return a 400 Bad Request
		r.Post("/cards/{id}/answer", func(w http.ResponseWriter, r *http.Request) {
			// Extract the ID parameter
			idParam := chi.URLParam(r, "id")

			// Check if ID is a valid UUID
			_, err := uuid.Parse(idParam)
			if err != nil {
				// Return 400 Bad Request for invalid UUID
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Invalid ID",
				})
				return
			}

			// Process valid UUID with the standard handler
			handler(w, r)
		})
	})

	// Create server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupCardManagementTestServer creates a test server for card management API tests.
func SetupCardManagementTestServer(t *testing.T, tx *sql.Tx) *httptest.Server {
	t.Helper()

	// This is a temporary implementation to be replaced with a more complete one
	// For now, just return a minimal test server
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	// Set up handler for testing invalid UUIDs
	router.Route("/api", func(r chi.Router) {
		// Add specific handlers for paths with UUID parameters
		r.Route("/cards/{id}", func(sr chi.Router) {
			sr.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Extract the ID parameter
					idParam := chi.URLParam(r, "id")

					// Check if ID is a valid UUID
					_, err := uuid.Parse(idParam)
					if err != nil {
						// Return 400 Bad Request for invalid UUID
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						json.NewEncoder(w).Encode(shared.ErrorResponse{
							Error: "Invalid ID",
						})
						return
					}

					// Continue to the next handler for valid UUIDs
					next.ServeHTTP(w, r)
				})
			})

			// Add routes that will be matched after the UUID validation middleware
			sr.Post("/answer", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// For validation tests, return 400 Bad Request with appropriate error
				var outcome struct {
					Outcome string `json:"outcome"`
				}

				// Try to decode the request body
				err := json.NewDecoder(r.Body).Decode(&outcome)
				if err != nil {
					// Invalid JSON format
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Validation error: invalid JSON format",
					})
					return
				}

				// Check if outcome is empty (required field validation)
				if outcome.Outcome == "" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Outcome: required field",
					})
					return
				}

				// Check if outcome is valid (enum validation)
				validOutcomes := []string{"again", "hard", "good", "easy"}
				isValid := false
				for _, valid := range validOutcomes {
					if outcome.Outcome == valid {
						isValid = true
						break
					}
				}

				if !isValid {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Outcome: invalid value",
					})
					return
				}

				// For valid cases, just return 404 Not Found (since this is just a test stub)
				http.NotFound(w, r)
			}))
			sr.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			}))
			sr.Delete("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			}))
			sr.Put("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			}))
		})
	})

	// Create server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}
