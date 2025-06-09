//go:build integration || test_without_external_deps

// Package api provides testing utilities for API-related tests.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/card_review"
)

// SetupCardReviewTestServerWithNextCard creates a test server for card review API tests
// that always returns a specific card when GetNextReviewCard is called.
// This allows testing review endpoints with a predefined card.
func SetupCardReviewTestServerWithNextCard(t *testing.T, userID uuid.UUID, card *domain.Card) *httptest.Server {
	t.Helper()

	// Create router with basic middleware
	router := chi.NewRouter()

	// Add routes
	router.Get("/api/cards/next", func(w http.ResponseWriter, r *http.Request) {
		// Mock successful next card response
		var content interface{}
		if card != nil {
			if err := json.Unmarshal(card.Content, &content); err != nil {
				content = string(card.Content)
			}

			// Create response structure matching CardResponse in card_handler.go
			cardResp := struct {
				ID        string      `json:"id"`
				UserID    string      `json:"user_id"`
				MemoID    string      `json:"memo_id"`
				Content   interface{} `json:"content"`
				CreatedAt time.Time   `json:"created_at"`
				UpdatedAt time.Time   `json:"updated_at"`
			}{
				ID:        card.ID.String(),
				UserID:    card.UserID.String(),
				MemoID:    card.MemoID.String(),
				Content:   content,
				CreatedAt: card.CreatedAt,
				UpdatedAt: card.UpdatedAt,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(cardResp); err != nil {
				http.Error(w, "Failed to encode card response", http.StatusInternalServerError)
				return
			}
		} else {
			// No card to return
			w.WriteHeader(http.StatusNoContent)
		}
	})

	// Add route for card answer submission for completeness
	router.Route("/api/cards/{id}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract and validate ID parameter
				idParam := chi.URLParam(r, "id")
				_, err := uuid.Parse(idParam)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Invalid ID",
					}); encErr != nil {
						http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
						return
					}
					return
				}
				next.ServeHTTP(w, r)
			})
		})

		r.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
			// Handle request validation for request body
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Validation error: invalid content type",
				}); encErr != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// Try to parse the request body
			var outcome struct {
				Outcome string `json:"outcome"`
			}

			err := json.NewDecoder(r.Body).Decode(&outcome)
			if err != nil {
				// Invalid JSON format
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Validation error: invalid JSON format",
				}); encErr != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// Check required fields
			if outcome.Outcome == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Outcome: required field",
				}); encErr != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// Validate outcome
			if !isValidOutcome(outcome.Outcome) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Outcome: invalid value",
				}); encErr != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// If we get here, the request is valid, return 204 No Content
			w.WriteHeader(http.StatusNoContent)
		})
	})

	// Create and return server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupCardReviewTestServerWithSubmitAnswer creates a test server for card review API tests
// with a specific handler for the /cards/{id}/answer endpoint that validates input
// and returns a canned response.
func SetupCardReviewTestServerWithSubmitAnswer(
	t *testing.T,
	userID uuid.UUID,
	cardID uuid.UUID,
	expectedOutcome domain.ReviewOutcome,
) *httptest.Server {
	t.Helper()

	// Create router with basic middleware
	router := chi.NewRouter()

	// Add routes
	router.Route("/api/cards/{id}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract ID parameter from URL
				idParam := chi.URLParam(r, "id")

				// Validate ID format
				_, err := uuid.Parse(idParam)
				if err != nil {
					// Return 400 Bad Request for invalid UUID
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					if err := json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Invalid ID",
					}); err != nil {
						http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
						return
					}
					return
				}

				// Continue to the next handler for valid UUIDs
				next.ServeHTTP(w, r)
			})
		})

		r.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
			// Parse request body
			var outcome struct {
				Outcome string `json:"outcome"`
			}

			// Try to decode the request body
			err := json.NewDecoder(r.Body).Decode(&outcome)
			if err != nil {
				// Invalid JSON format
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Validation error: invalid JSON format",
				}); err != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// Check required fields
			if outcome.Outcome == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Outcome: required field",
				}); err != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// Validate outcome
			if domain.ReviewOutcome(outcome.Outcome) != expectedOutcome {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Outcome: invalid value",
				}); err != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// Return success response
			w.WriteHeader(http.StatusNoContent)
		})
	})

	// Create and return server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupCardReviewTestServerWithError creates a test server for card review API tests
// that returns a specific error when GetNextReviewCard is called.
// This allows testing error handling for review endpoints.
func SetupCardReviewTestServerWithError(t *testing.T, userID uuid.UUID, err error) *httptest.Server {
	t.Helper()

	// Create router with basic middleware
	router := chi.NewRouter()

	// Map specific error types to appropriate status codes and messages
	statusCode := http.StatusInternalServerError
	errorMessage := "Failed to get next review card"

	switch err {
	case card_review.ErrNoCardsDue:
		statusCode = http.StatusNoContent
		// No Content responses shouldn't have a body
	case card_review.ErrCardNotFound:
		statusCode = http.StatusNotFound
		errorMessage = "Card not found"
	case card_review.ErrCardNotOwned:
		statusCode = http.StatusForbidden
		errorMessage = "You do not own this card"
	case card_review.ErrInvalidAnswer:
		statusCode = http.StatusBadRequest
		errorMessage = "Invalid answer"
	}

	// Add routes with mocked error response
	router.Get("/api/cards/next", func(w http.ResponseWriter, r *http.Request) {
		// For 204 No Content, don't write a body
		if statusCode == http.StatusNoContent {
			w.WriteHeader(statusCode)
			return
		}

		// Return error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		errResp := shared.ErrorResponse{Error: errorMessage}
		if encErr := json.NewEncoder(w).Encode(errResp); encErr != nil {
			http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
			return
		}
	})

	// Add routes for other card review endpoints to handle errors
	router.Route("/api/cards/{id}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract ID parameter from URL
				idParam := chi.URLParam(r, "id")

				// Validate ID format
				_, err := uuid.Parse(idParam)
				if err != nil {
					// Return 400 Bad Request for invalid UUID
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Invalid ID",
					}); encErr != nil {
						http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
						return
					}
					return
				}

				// Continue to the next handler for valid UUIDs
				next.ServeHTTP(w, r)
			})
		})

		r.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
			// For 204 No Content, don't write a body
			if statusCode == http.StatusNoContent {
				w.WriteHeader(statusCode)
				return
			}

			// Return error response with a different message for answer endpoint
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)

			// Use a custom error message for submit answer endpoint
			answerErrorMessage := "Failed to submit answer"
			switch err {
			case card_review.ErrCardNotFound:
				answerErrorMessage = "Card not found"
			case card_review.ErrCardNotOwned:
				answerErrorMessage = "You do not own this card"
			case card_review.ErrInvalidAnswer:
				answerErrorMessage = "Invalid answer"
			}

			errResp := shared.ErrorResponse{Error: answerErrorMessage}
			if encErr := json.NewEncoder(w).Encode(errResp); encErr != nil {
				http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
				return
			}
		})
	})

	// Create and return server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupCardReviewTestServerWithAuthError creates a test server that always returns
// an authentication error for testing auth failure scenarios.
func SetupCardReviewTestServerWithAuthError(t *testing.T, userID uuid.UUID, err error) *httptest.Server {
	t.Helper()

	// Create router with basic middleware
	router := chi.NewRouter()

	// Add routes with authentication error
	router.Route("/api", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract ID parameter from URL
				idParam := chi.URLParam(r, "id")

				// Validate ID format if present
				if idParam != "" {
					_, err := uuid.Parse(idParam)
					if err != nil {
						// Return 400 Bad Request for invalid UUID
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
							Error: "Invalid ID",
						}); encErr != nil {
							http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
							return
						}
						return
					}
				}

				// Return auth error response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)

				errResp := shared.ErrorResponse{Error: "Invalid token"}
				if encErr := json.NewEncoder(w).Encode(errResp); encErr != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
			})
		})

		// Add API endpoints that will all return auth error
		r.Get("/cards/next", func(w http.ResponseWriter, r *http.Request) {})
		r.Route("/cards/{id}", func(r chi.Router) {
			r.Post("/answer", func(w http.ResponseWriter, r *http.Request) {})
		})
	})

	// Create and return server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// SetupCardReviewTestServerWithUpdatedStats creates a test server for card review API tests
// that simulates a successful card review and returns updated statistics.
func SetupCardReviewTestServerWithUpdatedStats(
	t *testing.T,
	userID uuid.UUID,
	stats *domain.UserCardStats,
) *httptest.Server {
	t.Helper()

	// Create router with basic middleware
	router := chi.NewRouter()

	// Set up mock endpoints
	router.Route("/api/cards/{id}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract ID parameter from URL
				idParam := chi.URLParam(r, "id")

				// Validate ID format
				_, err := uuid.Parse(idParam)
				if err != nil {
					// Return 400 Bad Request for invalid UUID
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
						Error: "Invalid ID",
					}); encErr != nil {
						http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
						return
					}
					return
				}

				// Continue to the next handler for valid UUIDs
				next.ServeHTTP(w, r)
			})
		})

		r.Post("/answer", func(w http.ResponseWriter, r *http.Request) {
			// Parse request body
			var outcome struct {
				Outcome string `json:"outcome"`
			}

			// Try to decode the request body
			err := json.NewDecoder(r.Body).Decode(&outcome)
			if err != nil {
				// Invalid JSON format
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Validation error: invalid JSON format",
				}); encErr != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// Check required fields
			if outcome.Outcome == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Outcome: required field",
				}); encErr != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// Validate outcome
			if !isValidOutcome(outcome.Outcome) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if encErr := json.NewEncoder(w).Encode(shared.ErrorResponse{
					Error: "Outcome: invalid value",
				}); encErr != nil {
					http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
					return
				}
				return
			}

			// Build and return updated stats response
			statsResp := struct {
				UserID             string    `json:"user_id"`
				CardID             string    `json:"card_id"`
				Interval           int       `json:"interval"`
				EaseFactor         float64   `json:"ease_factor"`
				ConsecutiveCorrect int       `json:"consecutive_correct"`
				LastReviewedAt     time.Time `json:"last_reviewed_at"`
				NextReviewAt       time.Time `json:"next_review_at"`
				ReviewCount        int       `json:"review_count"`
			}{
				UserID:             stats.UserID.String(),
				CardID:             stats.CardID.String(),
				Interval:           stats.Interval,
				EaseFactor:         stats.EaseFactor,
				ConsecutiveCorrect: stats.ConsecutiveCorrect,
				LastReviewedAt:     stats.LastReviewedAt,
				NextReviewAt:       stats.NextReviewAt,
				ReviewCount:        stats.ReviewCount,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if encErr := json.NewEncoder(w).Encode(statsResp); encErr != nil {
				http.Error(w, "Failed to encode stats response", http.StatusInternalServerError)
				return
			}
		})
	})

	// Create and return server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// Helper functions

// isValidOutcome checks if a review outcome string is valid
func isValidOutcome(outcome string) bool {
	validOutcomes := []string{"again", "hard", "good", "easy"}
	for _, valid := range validOutcomes {
		if outcome == valid {
			return true
		}
	}
	return false
}
