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

// CardResponse represents the response data for a card.
// It contains all the card details in a format suitable for API responses,
// with UUIDs converted to strings and content parsed into a generic interface.
type CardResponse struct {
	// ID is the unique identifier for the card
	ID string `json:"id"`

	// UserID is the identifier of the user who owns this card
	UserID string `json:"user_id"`

	// MemoID is the identifier of the memo from which this card was generated
	MemoID string `json:"memo_id"`

	// Content contains the card's actual content, which can vary in structure
	// depending on the card type (e.g., question-answer, cloze, etc.)
	Content interface{} `json:"content"`

	// CreatedAt is the timestamp when the card was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when the card was last updated
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

// EditCardRequest represents the request body for editing a card's content.
// It is used for the PUT /cards/{id} endpoint to update a card's content.
type EditCardRequest struct {
	// Content is the new JSON content for the card.
	// It must be valid JSON and is validated with the "required" tag.
	// The structure depends on the card type but typically includes fields like
	// question, answer, hints, etc.
	Content json.RawMessage `json:"content" validate:"required"`
}

// SubmitAnswerRequest represents the request body for submitting a card review answer.
// It is used for the POST /cards/{id}/answer endpoint to record the result of a user
// reviewing a flashcard and update its spaced repetition scheduling.
type SubmitAnswerRequest struct {
	// Outcome is the result of the card review, affecting how the next review date is calculated.
	// It must be one of: "again" (failed), "hard" (difficult), "good" (correct), or "easy" (very easy).
	// These values map to the SRS algorithm's difficulty levels and affect interval calculations.
	Outcome string `json:"outcome" validate:"required,oneof=again hard good easy"`
}

// UserCardStatsResponse represents the response data for user card statistics.
// It contains the spaced repetition algorithm parameters and scheduling information
// for a specific user-card pair.
type UserCardStatsResponse struct {
	// UserID is the unique identifier of the user who owns these stats
	UserID string `json:"user_id"`

	// CardID is the unique identifier of the card these stats are for
	CardID string `json:"card_id"`

	// Interval is the current review interval in days
	// (time between reviews when answered correctly)
	Interval int `json:"interval"`

	// EaseFactor is the card's current ease factor, which affects
	// how quickly intervals grow based on answer quality
	EaseFactor float64 `json:"ease_factor"`

	// ConsecutiveCorrect is the number of times the card has been
	// answered correctly in a row
	ConsecutiveCorrect int `json:"consecutive_correct"`

	// LastReviewedAt is the timestamp of when the card was last reviewed
	LastReviewedAt time.Time `json:"last_reviewed_at"`

	// NextReviewAt is the timestamp of when the card is next due for review
	NextReviewAt time.Time `json:"next_review_at"`

	// ReviewCount is the total number of times this card has been reviewed
	ReviewCount int `json:"review_count"`
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

// statsToResponse converts a domain.UserCardStats to a UserCardStatsResponse.
// This helper function transforms the internal domain model to the API response format,
// ensuring proper type conversions (e.g., UUIDs to strings) and field mapping.
//
// Parameters:
//   - stats: The domain model user card statistics to convert
//
// Returns:
//   - UserCardStatsResponse: The transformed API response object
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

// PostponeCardRequest represents the request body for postponing a card review.
// It is used for the POST /cards/{id}/postpone endpoint to delay a card's next review date.
type PostponeCardRequest struct {
	// Days is the number of days to postpone the card's next review.
	// It must be at least 1 day, as validated by the "min=1" tag.
	// The next review date will be extended by this many days from its current value.
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

// cardToResponse converts a domain.Card to a CardResponse.
// This helper function transforms the internal domain model to the API response format,
// ensuring proper type conversions (e.g., UUIDs to strings) and unmarshaling the
// JSON content field into a more usable interface{} rather than raw bytes.
//
// Parameters:
//   - card: The domain model card to convert
//
// Returns:
//   - CardResponse: The transformed API response object
//
// If the JSON content cannot be unmarshaled, it falls back to representing
// the content as a string of the raw bytes.
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
