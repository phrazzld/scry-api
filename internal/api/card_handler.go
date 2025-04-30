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

// CardResponse represents the response data for a card.
// It contains all the card details in a format suitable for API responses,
// with UUIDs converted to strings and content parsed into a generic interface.
type CardResponse struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	MemoID string `json:"memo_id"`

	// Content varies in structure depending on the card type (e.g., question-answer, cloze)
	Content interface{} `json:"content"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

// EditCardRequest represents the request body for editing a card's content.
// It is used for the PUT /cards/{id} endpoint to update a card's content.
type EditCardRequest struct {
	// New JSON content for the card. Structure depends on card type (e.g., question, answer, hints)
	Content json.RawMessage `json:"content" validate:"required"`
}

// SubmitAnswerRequest represents the request body for submitting a card review answer.
// It is used for the POST /cards/{id}/answer endpoint to record the result of a user
// reviewing a flashcard and update its spaced repetition scheduling.
type SubmitAnswerRequest struct {
	// Must be one of: "again" (failed), "hard" (difficult), "good" (correct), or "easy" (very easy)
	// Maps to SRS algorithm difficulty levels and affects interval calculations
	Outcome string `json:"outcome" validate:"required,oneof=again hard good easy"`
}

// UserCardStatsResponse represents the response data for user card statistics.
// It contains the spaced repetition algorithm parameters and scheduling information
// for a specific user-card pair.
type UserCardStatsResponse struct {
	UserID string `json:"user_id"`
	CardID string `json:"card_id"`

	// Current review interval in days (time between reviews when answered correctly)
	Interval int `json:"interval"`

	// Affects how quickly intervals grow based on answer quality
	EaseFactor float64 `json:"ease_factor"`

	// Number of times the card has been answered correctly in a row
	ConsecutiveCorrect int `json:"consecutive_correct"`

	LastReviewedAt time.Time `json:"last_reviewed_at"`
	NextReviewAt   time.Time `json:"next_review_at"`

	// Total number of times this card has been reviewed
	ReviewCount int `json:"review_count"`
}

// SubmitAnswer handles POST /cards/{id}/answer requests
// It processes a user's answer to a card review and updates the spaced repetition schedule.
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
// This helper function transforms the internal domain model to the API response format,
// ensuring proper type conversions (e.g., UUIDs to strings).
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

// EditCard handles PUT /cards/{id} requests.
// It updates the content of an existing card after validating user ownership.
//
// HTTP Request:
//   - Method: PUT
//   - Path: /api/cards/{id}
//   - Path Parameters:
//   - id: UUID of the card to edit
//   - Headers:
//   - Authorization: Bearer <JWT token>
//   - Body: JSON object with "content" field (EditCardRequest)
//
// HTTP Response:
//   - 204 No Content: Card updated successfully
//   - 400 Bad Request: Invalid request body, invalid JSON, or card ID format
//   - 401 Unauthorized: Missing or invalid JWT token
//   - 403 Forbidden: User is not the owner of the card
//   - 404 Not Found: Card not found
//   - 500 Internal Server Error: Server error
//
// The handler extracts the card ID from the URL, the user ID from the JWT token,
// validates the request body, and calls the CardService.UpdateCardContent method.
// It performs ownership validation to ensure that only the card's owner can edit it.
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

// DeleteCard handles DELETE /cards/{id} requests.
// It deletes an existing card after validating user ownership.
//
// HTTP Request:
//   - Method: DELETE
//   - Path: /api/cards/{id}
//   - Path Parameters:
//   - id: UUID of the card to delete
//   - Headers:
//   - Authorization: Bearer <JWT token>
//
// HTTP Response:
//   - 204 No Content: Card deleted successfully
//   - 400 Bad Request: Invalid card ID format
//   - 401 Unauthorized: Missing or invalid JWT token
//   - 403 Forbidden: User is not the owner of the card
//   - 404 Not Found: Card not found
//   - 500 Internal Server Error: Server error
//
// The handler extracts the card ID from the URL, the user ID from the JWT token,
// and calls the CardService.DeleteCard method. It performs ownership validation
// to ensure that only the card's owner can delete it. The deletion is permanent
// and cascades to associated user_card_stats entries through database constraints.
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

// PostponeCardRequest represents the request body for postponing a card review.
// It is used for the POST /cards/{id}/postpone endpoint to delay a card's next review date.
type PostponeCardRequest struct {
	// Number of days to extend the next review date. Must be at least 1.
	Days int `json:"days" validate:"required,min=1"`
}

// PostponeCard handles POST /cards/{id}/postpone requests.
// It postpones the next review date of a card by a specified number of days.
//
// HTTP Request:
//   - Method: POST
//   - Path: /api/cards/{id}/postpone
//   - Path Parameters:
//   - id: UUID of the card to postpone
//   - Headers:
//   - Authorization: Bearer <JWT token>
//   - Body: JSON object with "days" field (PostponeCardRequest)
//
// HTTP Response:
//   - 200 OK: Card postponed successfully, with updated UserCardStatsResponse in body
//   - 400 Bad Request: Invalid request body, days < 1, or invalid card ID format
//   - 401 Unauthorized: Missing or invalid JWT token
//   - 403 Forbidden: User is not the owner of the card
//   - 404 Not Found: Card not found or stats not found
//   - 500 Internal Server Error: Server error
//
// The handler extracts the card ID from the URL, the user ID from the JWT token,
// validates the request body, and calls the CardService.PostponeCard method.
// It performs ownership validation to ensure that only the card's owner can postpone it.
// The operation is executed in a transaction to ensure atomicity and prevent race conditions.
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
// Transforms internal model to API format, converting UUIDs to strings and
// unmarshaling JSON content to interface{}. Falls back to string representation
// if unmarshaling fails.
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
