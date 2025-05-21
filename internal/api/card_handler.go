// Package api provides HTTP handlers for the API.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/redact"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/card_review"
)

// CardResponse formats card data for API responses with parsed content
type CardResponse struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	MemoID    string      `json:"memo_id"`
	Content   interface{} `json:"content"` // Varies by card type (question-answer, cloze)
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// CardHandler handles card-related HTTP requests
type CardHandler struct {
	cardReviewService card_review.CardReviewService
	cardService       service.CardService
	logger            *slog.Logger
}

// NewCardHandler creates a new CardHandler with the required services
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

// GetNextReviewCard retrieves the next card due for review for the authenticated user.
func (h *CardHandler) GetNextReviewCard(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract user ID from context (set by auth middleware)
	userID, ok := handleUserIDFromContext(w, r, log)
	if !ok {
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

// EditCardRequest contains updated card content
type EditCardRequest struct {
	Content json.RawMessage `json:"content" validate:"required"` // Structure varies by card type
}

// SubmitAnswerRequest contains the user's response to a flashcard review
type SubmitAnswerRequest struct {
	Outcome string `json:"outcome" validate:"required,oneof=again hard good easy"` // Review quality: again/hard/good/easy
}

// UserCardStatsResponse contains SRS scheduling and review history data
type UserCardStatsResponse struct {
	UserID             string    `json:"user_id"`
	CardID             string    `json:"card_id"`
	Interval           int       `json:"interval"`            // Days between reviews
	EaseFactor         float64   `json:"ease_factor"`         // Controls interval growth rate
	ConsecutiveCorrect int       `json:"consecutive_correct"` // Streak of correct answers
	LastReviewedAt     time.Time `json:"last_reviewed_at"`
	NextReviewAt       time.Time `json:"next_review_at"`
	ReviewCount        int       `json:"review_count"` // Total times reviewed
}

// SubmitAnswer processes a user's card review answer and updates its spaced repetition schedule.
func (h *CardHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract user ID and card ID
	userID, cardID, ok := handleUserIDAndPathUUID(w, r, "id", log)
	if !ok {
		return
	}

	// Parse and validate request
	var req SubmitAnswerRequest
	logFields := []slog.Attr{
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
	}
	if !parseAndValidateRequest(w, r, &req, log, logFields...) {
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

// statsToResponse converts a domain.UserCardStats to a UserCardStatsResponse.
// Handles type conversions like UUIDs to strings.
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

// EditCard updates a card's content after validating user ownership
//
// Handles PUT /api/cards/{id} with content in request body.
// Returns 204 No Content on success.
// Enforces ownership validation to ensure only the card owner can edit it.
func (h *CardHandler) EditCard(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract user ID and card ID
	userID, cardID, ok := handleUserIDAndPathUUID(w, r, "id", log)
	if !ok {
		return
	}

	// Parse and validate request
	var req EditCardRequest
	logFields := []slog.Attr{
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
	}
	if !parseAndValidateRequest(w, r, &req, log, logFields...) {
		return
	}

	// Call service to update card content
	err := h.cardService.UpdateCardContent(r.Context(), userID, cardID, req.Content)
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

// DeleteCard permanently removes a card after validating user ownership
//
// Handles DELETE /api/cards/{id} requests.
// Returns 204 No Content on success.
// Enforces ownership validation and cascades deletion to related records.
func (h *CardHandler) DeleteCard(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract user ID and card ID
	userID, cardID, ok := handleUserIDAndPathUUID(w, r, "id", log)
	if !ok {
		return
	}

	// Call service to delete the card
	err := h.cardService.DeleteCard(r.Context(), userID, cardID)
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

// PostponeCardRequest specifies how long to delay a card's review
type PostponeCardRequest struct {
	Days int `json:"days" validate:"required,min=1"` // Days to extend review date (min 1)
}

// PostponeCard delays a card's next review date by specified number of days
//
// Handles POST /api/cards/{id}/postpone with days parameter in request body.
// Returns updated card stats with new NextReviewAt date.
// Enforces ownership validation and executes in a transaction for atomicity.
func (h *CardHandler) PostponeCard(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract user ID and card ID
	userID, cardID, ok := handleUserIDAndPathUUID(w, r, "id", log)
	if !ok {
		return
	}

	// Parse and validate request
	var req PostponeCardRequest
	logFields := []slog.Attr{
		slog.String("user_id", userID.String()),
		slog.String("card_id", cardID.String()),
	}
	if !parseAndValidateRequest(w, r, &req, log, logFields...) {
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

// cardToResponse converts a domain.Card to a CardResponse.
// Unmarshals JSON content to interface{} or falls back to string representation if that fails.
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
