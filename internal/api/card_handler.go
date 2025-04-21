// Package api provides HTTP handlers for the API.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/service/card_review"
)

// CardResponse represents the response data for a card
type CardResponse struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	MemoID    string      `json:"memo_id"`
	Content   interface{} `json:"content"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// CardHandler handles card-related HTTP requests
type CardHandler struct {
	cardReviewService card_review.CardReviewService
	validator         *validator.Validate
	logger            *slog.Logger
}

// NewCardHandler creates a new CardHandler
func NewCardHandler(cardReviewService card_review.CardReviewService, logger *slog.Logger) *CardHandler {
	// Use provided logger or create default
	if logger == nil {
		logger = slog.Default()
	}

	return &CardHandler{
		cardReviewService: cardReviewService,
		validator:         validator.New(),
		logger:            logger.With(slog.String("component", "card_handler")),
	}
}

// GetNextReviewCard handles GET /cards/next requests
// It retrieves the next card due for review for the authenticated user.
func (h *CardHandler) GetNextReviewCard(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		log.Warn("user ID not found or invalid in request context")
		shared.RespondWithError(w, r, http.StatusUnauthorized, "User ID not found or invalid")
		return
	}

	log.Debug("getting next review card", slog.String("user_id", userID.String()))

	// Get next card from service
	card, err := h.cardReviewService.GetNextCard(r.Context(), userID)

	// Special case: no cards due for review
	if errors.Is(err, card_review.ErrNoCardsDue) {
		log.Debug("no cards due for review", slog.String("user_id", userID.String()))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Handle other errors
	if err != nil {
		log.Error("failed to get next review card",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		shared.RespondWithError(w, r, http.StatusInternalServerError, "Failed to get next review card")
		return
	}

	// Transform domain object to response
	response := cardToResponse(card)

	// Return response with 200 OK status
	log.Debug("successfully retrieved next review card",
		slog.String("user_id", userID.String()),
		slog.String("card_id", card.ID.String()))
	shared.RespondWithJSON(w, r, http.StatusOK, response)
}

// cardToResponse converts a domain.Card to a CardResponse
func cardToResponse(card *domain.Card) CardResponse {
	var content interface{}
	if err := json.Unmarshal(card.Content, &content); err != nil {
		// In case we can't unmarshal, return raw bytes as a string representation
		content = string(card.Content)
	}

	return CardResponse{
		ID:        card.ID.String(),
		UserID:    card.UserID.String(),
		MemoID:    card.MemoID.String(),
		Content:   content,
		CreatedAt: card.CreatedAt,
		UpdatedAt: card.UpdatedAt,
	}
}
