package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
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
	validator   *validator.Validate
}

// NewMemoHandler creates a new MemoHandler
func NewMemoHandler(memoService service.MemoService) *MemoHandler {
	return &MemoHandler{
		memoService: memoService,
		validator:   validator.New(),
	}
}

// CreateMemo handles POST /api/memos requests
func (h *MemoHandler) CreateMemo(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(shared.UserIDContextKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		shared.RespondWithError(w, r, http.StatusUnauthorized, "User ID not found or invalid")
		return
	}

	// Parse request body
	var req CreateMemoRequest
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.RespondWithError(w, r, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		shared.RespondWithError(w, r, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// Create memo and enqueue task
	memo, err := h.memoService.CreateMemoAndEnqueueTask(r.Context(), userID, req.Text)
	if err != nil {
		slog.Error("Failed to create memo", "error", err, "user_id", userID)
		// TODO(api-error-handling): Add more specific error handling based on error types:
		// 1. Create an errors.go file in the api package with:
		//    - A function to map domain errors to HTTP status codes (e.g., domain.ErrInvalidMemo -> 400)
		//    - A function to map service errors to HTTP status codes (e.g., service.ErrPermissionDenied -> 403)
		//    - A function to extract user-safe error messages
		// 2. Implement error handling middleware that uses error wrapping (errors.Is, errors.As)
		// 3. Update shared.RespondWithError to accept error types and handle mapping internally
		// 4. Replace all direct status code assignments with the new mapping function
		// 5. Add tests verifying correct status code mapping for each error type
		shared.RespondWithError(w, r, http.StatusInternalServerError, "Failed to create memo")
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
