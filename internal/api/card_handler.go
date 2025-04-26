// Package api provides HTTP handlers for the API.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/redact"
	"github.com/phrazzld/scry-api/internal/service"
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
	cardService       service.CardService
	logger            *slog.Logger
}

// NewCardHandler creates a new CardHandler
func NewCardHandler(
	cardReviewService card_review.CardReviewService,
	cardService service.CardService,
	logger *slog.Logger,
) *CardHandler {
	if logger == nil {
		// ALLOW-PANIC: Constructor enforcing required dependency
		panic("logger cannot be nil for CardHandler")
	}

	return &CardHandler{
		cardReviewService: cardReviewService,
		cardService:       cardService,
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
		HandleAPIError(w, r, domain.ErrUnauthorized, "User ID not found or invalid")
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
		HandleAPIError(w, r, err, "Failed to get next review card")
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

// EditCardRequest represents the request body for editing a card's content
type EditCardRequest struct {
	Content json.RawMessage `json:"content" validate:"required"`
}

// SubmitAnswerRequest represents the request body for submitting a card review answer
type SubmitAnswerRequest struct {
	Outcome string `json:"outcome" validate:"required,oneof=again hard good easy"`
}

// UserCardStatsResponse represents the response data for user card statistics
type UserCardStatsResponse struct {
	UserID             string    `json:"user_id"`
	CardID             string    `json:"card_id"`
	Interval           int       `json:"interval"`
	EaseFactor         float64   `json:"ease_factor"`
	ConsecutiveCorrect int       `json:"consecutive_correct"`
	LastReviewedAt     time.Time `json:"last_reviewed_at"`
	NextReviewAt       time.Time `json:"next_review_at"`
	ReviewCount        int       `json:"review_count"`
}

// SubmitAnswer handles POST /cards/{id}/answer requests
// It processes a user's answer to a card review and updates the spaced repetition schedule.
func (h *CardHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract card ID from URL path using chi router
	pathCardID := chi.URLParam(r, "id")
	if pathCardID == "" {
		log.Warn("card ID not found in URL path")
		HandleAPIError(w, r, domain.ErrValidation, "Card ID is required")
		return
	}

	// Parse card ID as UUID
	cardID, err := uuid.Parse(pathCardID)
	if err != nil {
		log.Warn("invalid card ID format", slog.String("card_id", pathCardID))
		HandleAPIError(w, r, domain.ErrInvalidID, "Invalid card ID format")
		return
	}

	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		log.Warn("user ID not found or invalid in request context")
		HandleAPIError(w, r, domain.ErrUnauthorized, "User ID not found or invalid")
		return
	}

	// Parse request body
	var req SubmitAnswerRequest
	if err := shared.DecodeJSON(r, &req); err != nil {
		log.Warn("invalid request format",
			slog.String("error", redact.Error(err)),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		HandleValidationError(w, r, err)
		return
	}

	// Validate request
	if err := shared.Validate.Struct(req); err != nil {
		log.Warn("validation error",
			slog.String("error", redact.Error(err)),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		HandleValidationError(w, r, err)
		return
	}

	// Convert string outcome to domain.ReviewOutcome
	outcome := domain.ReviewOutcome(req.Outcome)

	// Submit answer to service
	stats, err := h.cardReviewService.SubmitAnswer(
		r.Context(),
		userID,
		cardID,
		card_review.ReviewAnswer{Outcome: outcome},
	)

	// Handle errors with our improved error handling
	if err != nil {
		HandleAPIError(w, r, err, "Failed to submit answer")
		return
	}

	// Transform domain object to response
	response := statsToResponse(stats)

	// Return response with 200 OK status
	log.Debug("successfully submitted answer",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
		slog.String("outcome", string(outcome)))
	shared.RespondWithJSON(w, r, http.StatusOK, response)
}

// statsToResponse converts a domain.UserCardStats to a UserCardStatsResponse
func statsToResponse(stats *domain.UserCardStats) UserCardStatsResponse {
	return UserCardStatsResponse{
		UserID:             stats.UserID.String(),
		CardID:             stats.CardID.String(),
		Interval:           stats.Interval,
		EaseFactor:         stats.EaseFactor,
		ConsecutiveCorrect: stats.ConsecutiveCorrect,
		LastReviewedAt:     stats.LastReviewedAt,
		NextReviewAt:       stats.NextReviewAt,
		ReviewCount:        stats.ReviewCount,
	}
}

// EditCard handles PUT /cards/{id} requests
// It updates the content of an existing card after validating user ownership
func (h *CardHandler) EditCard(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract card ID from URL path using chi router
	pathCardID := chi.URLParam(r, "id")
	if pathCardID == "" {
		log.Warn("card ID not found in URL path")
		HandleAPIError(w, r, domain.ErrValidation, "Card ID is required")
		return
	}

	// Parse card ID as UUID
	cardID, err := uuid.Parse(pathCardID)
	if err != nil {
		log.Warn("invalid card ID format", slog.String("card_id", pathCardID))
		HandleAPIError(w, r, domain.ErrInvalidID, "Invalid card ID format")
		return
	}

	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		log.Warn("user ID not found or invalid in request context")
		HandleAPIError(w, r, domain.ErrUnauthorized, "User ID not found or invalid")
		return
	}

	// Parse request body
	var req EditCardRequest
	if err := shared.DecodeJSON(r, &req); err != nil {
		log.Warn("invalid request format",
			slog.String("error", redact.Error(err)),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		HandleValidationError(w, r, err)
		return
	}

	// Validate request
	if err := shared.Validate.Struct(req); err != nil {
		log.Warn("validation error",
			slog.String("error", redact.Error(err)),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		HandleValidationError(w, r, err)
		return
	}

	// Call service to update card content
	err = h.cardService.UpdateCardContent(r.Context(), userID, cardID, req.Content)
	if err != nil {
		log.Error("failed to update card content",
			slog.String("error", redact.Error(err)),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		HandleAPIError(w, r, err, "Failed to update card content")
		return
	}

	// Return success with 204 No Content status
	log.Debug("card content updated successfully",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))
	w.WriteHeader(http.StatusNoContent)
}

// DeleteCard handles DELETE /cards/{id} requests
// It deletes an existing card after validating user ownership
func (h *CardHandler) DeleteCard(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract card ID from URL path using chi router
	pathCardID := chi.URLParam(r, "id")
	if pathCardID == "" {
		log.Warn("card ID not found in URL path")
		HandleAPIError(w, r, domain.ErrValidation, "Card ID is required")
		return
	}

	// Parse card ID as UUID
	cardID, err := uuid.Parse(pathCardID)
	if err != nil {
		log.Warn("invalid card ID format", slog.String("card_id", pathCardID))
		HandleAPIError(w, r, domain.ErrInvalidID, "Invalid card ID format")
		return
	}

	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		log.Warn("user ID not found or invalid in request context")
		HandleAPIError(w, r, domain.ErrUnauthorized, "User ID not found or invalid")
		return
	}

	// Call service to delete the card
	err = h.cardService.DeleteCard(r.Context(), userID, cardID)
	if err != nil {
		log.Error("failed to delete card",
			slog.String("error", redact.Error(err)),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		HandleAPIError(w, r, err, "Failed to delete card")
		return
	}

	// Return success with 204 No Content status
	log.Debug("card deleted successfully",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()))
	w.WriteHeader(http.StatusNoContent)
}

// PostponeCardRequest represents the request body for postponing a card review
type PostponeCardRequest struct {
	Days int `json:"days" validate:"required,min=1"`
}

// PostponeCard handles POST /cards/{id}/postpone requests
// It postpones the next review date of a card by a specified number of days
func (h *CardHandler) PostponeCard(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract card ID from URL path using chi router
	pathCardID := chi.URLParam(r, "id")
	if pathCardID == "" {
		log.Warn("card ID not found in URL path")
		HandleAPIError(w, r, domain.ErrValidation, "Card ID is required")
		return
	}

	// Parse card ID as UUID
	cardID, err := uuid.Parse(pathCardID)
	if err != nil {
		log.Warn("invalid card ID format", slog.String("card_id", pathCardID))
		HandleAPIError(w, r, domain.ErrInvalidID, "Invalid card ID format")
		return
	}

	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		log.Warn("user ID not found or invalid in request context")
		HandleAPIError(w, r, domain.ErrUnauthorized, "User ID not found or invalid")
		return
	}

	// Parse request body
	var req PostponeCardRequest
	if err := shared.DecodeJSON(r, &req); err != nil {
		log.Warn("invalid request format",
			slog.String("error", redact.Error(err)),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()))
		HandleValidationError(w, r, err)
		return
	}

	// Validate request
	if err := shared.Validate.Struct(req); err != nil {
		log.Warn("validation error",
			slog.String("error", redact.Error(err)),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()),
			slog.Int("days", req.Days))
		HandleValidationError(w, r, err)
		return
	}

	// Call service to postpone the card review
	stats, err := h.cardService.PostponeCard(r.Context(), userID, cardID, req.Days)
	if err != nil {
		log.Error("failed to postpone card review",
			slog.String("error", redact.Error(err)),
			slog.String("user_id", userID.String()),
			slog.String("card_id", cardID.String()),
			slog.Int("days", req.Days))
		HandleAPIError(w, r, err, "Failed to postpone card review")
		return
	}

	// Transform domain object to response
	response := statsToResponse(stats)

	// Return response with 200 OK status
	log.Debug("card review successfully postponed",
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
		slog.Int("days", req.Days),
		slog.Time("next_review_at", stats.NextReviewAt))
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
