package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/service"
)

// CreateMemoRequest represents the request body for creating a new memo
type CreateMemoRequest struct {
	Text string `json:"text" validate:"required,min=1"`
}

// MemoResponse represents the response data for a memo
type MemoResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Text      string    `json:"text"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MemoHandler handles memo-related HTTP requests
type MemoHandler struct {
	memoService service.MemoService
	logger      *slog.Logger
}

// NewMemoHandler creates a new MemoHandler
func NewMemoHandler(memoService service.MemoService, logger *slog.Logger) *MemoHandler {
	if logger == nil {
		// ALLOW-PANIC: Constructor enforcing required dependency
		panic("logger cannot be nil for MemoHandler")
	}

	return &MemoHandler{
		memoService: memoService,
		logger:      logger.With(slog.String("component", "memo_handler")),
	}
}

// CreateMemo handles POST /api/memos requests
func (h *MemoHandler) CreateMemo(w http.ResponseWriter, r *http.Request) {
	// Get logger from context or use default
	log := logger.FromContextOrDefault(r.Context(), h.logger)

	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		log.Warn("user ID not found or invalid in request context")
		shared.RespondWithError(w, r, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Parse request body
	var req CreateMemoRequest
	if err := shared.DecodeJSON(r, &req); err != nil {
		HandleValidationError(w, r, err)
		return
	}

	// Validate request
	if err := shared.Validate.Struct(req); err != nil {
		HandleValidationError(w, r, err)
		return
	}

	// Create memo and enqueue task
	memo, err := h.memoService.CreateMemoAndEnqueueTask(r.Context(), userID, req.Text)
	if err != nil {
		// Map error to appropriate status code and get sanitized message
		statusCode := MapErrorToStatusCode(err)
		safeMessage := GetSafeErrorMessage(err)

		// Log the full error details but only send sanitized message to client
		shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
		return
	}

	// Transform domain object to response
	response := memoToDTOResponse(memo)

	// Return response with 202 Accepted status (since processing happens asynchronously)
	shared.RespondWithJSON(w, r, http.StatusAccepted, response)
}

// memoToDTOResponse converts a domain.Memo to a MemoResponse
func memoToDTOResponse(memo *domain.Memo) MemoResponse {
	return MemoResponse{
		ID:        memo.ID.String(),
		UserID:    memo.UserID.String(),
		Text:      memo.Text,
		Status:    string(memo.Status),
		CreatedAt: memo.CreatedAt,
		UpdatedAt: memo.UpdatedAt,
	}
}
