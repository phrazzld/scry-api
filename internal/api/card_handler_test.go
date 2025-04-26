package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
)

// mockCardReviewService is a mock implementation of the CardReviewService interface
type mockCardReviewService struct {
	nextCardFn     func(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
	submitAnswerFn func(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error)
}

// mockCardService is a mock implementation of the CardService interface
type mockCardService struct {
	updateCardContentFn func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error
	deleteCardFn        func(ctx context.Context, userID, cardID uuid.UUID) error
	postponeCardFn      func(ctx context.Context, userID, cardID uuid.UUID, days int) (*domain.UserCardStats, error)
	createCardsFn       func(ctx context.Context, cards []*domain.Card) error
	getCardFn           func(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)
}

func (m *mockCardService) UpdateCardContent(
	ctx context.Context,
	userID, cardID uuid.UUID,
	content json.RawMessage,
) error {
	if m.updateCardContentFn != nil {
		return m.updateCardContentFn(ctx, userID, cardID, content)
	}
	return nil
}

func (m *mockCardService) DeleteCard(ctx context.Context, userID, cardID uuid.UUID) error {
	if m.deleteCardFn != nil {
		return m.deleteCardFn(ctx, userID, cardID)
	}
	return nil
}

func (m *mockCardService) PostponeCard(
	ctx context.Context,
	userID, cardID uuid.UUID,
	days int,
) (*domain.UserCardStats, error) {
	if m.postponeCardFn != nil {
		return m.postponeCardFn(ctx, userID, cardID, days)
	}
	return nil, nil
}

func (m *mockCardService) CreateCards(ctx context.Context, cards []*domain.Card) error {
	if m.createCardsFn != nil {
		return m.createCardsFn(ctx, cards)
	}
	return nil
}

func (m *mockCardService) GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
	if m.getCardFn != nil {
		return m.getCardFn(ctx, cardID)
	}
	return nil, nil
}

func (m *mockCardReviewService) GetNextCard(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.Card, error) {
	return m.nextCardFn(ctx, userID)
}

func (m *mockCardReviewService) SubmitAnswer(
	ctx context.Context,
	userID uuid.UUID,
	cardID uuid.UUID,
	answer card_review.ReviewAnswer,
) (*domain.UserCardStats, error) {
	return m.submitAnswerFn(ctx, userID, cardID, answer)
}

func TestGetNextReviewCard(t *testing.T) {
	userID := uuid.New()
	cardID := uuid.New()
	memoID := uuid.New()

	// Create sample content for the test card
	cardContent := map[string]interface{}{
		"front": "What is the capital of France?",
		"back":  "Paris",
	}
	contentBytes, _ := json.Marshal(cardContent)

	tests := []struct {
		name           string
		userIDInCtx    uuid.UUID
		serviceResult  *domain.Card
		serviceError   error
		expectedStatus int
		hasBody        bool
	}{
		{
			name:        "Success",
			userIDInCtx: userID,
			serviceResult: &domain.Card{
				ID:        cardID,
				UserID:    userID,
				MemoID:    memoID,
				Content:   contentBytes,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			serviceError:   nil,
			expectedStatus: http.StatusOK,
			hasBody:        true,
		},
		{
			name:           "No Cards Due",
			userIDInCtx:    userID,
			serviceResult:  nil,
			serviceError:   card_review.ErrNoCardsDue,
			expectedStatus: http.StatusNoContent,
			hasBody:        false,
		},
		{
			name:           "Other Error",
			userIDInCtx:    userID,
			serviceResult:  nil,
			serviceError:   errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			hasBody:        true, // Error response has a body
		},
		{
			name:           "Missing User ID",
			userIDInCtx:    uuid.Nil,
			serviceResult:  nil,
			serviceError:   nil,
			expectedStatus: http.StatusUnauthorized,
			hasBody:        true, // Error response has a body
		},
		{
			name:        "Service Returns Card But With Unmarshalable Content",
			userIDInCtx: userID,
			serviceResult: &domain.Card{
				ID:        cardID,
				UserID:    userID,
				MemoID:    memoID,
				Content:   []byte(`{"invalid json`), // Invalid JSON
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			serviceError:   nil,
			expectedStatus: http.StatusOK,
			hasBody:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock service that returns the test case's result
			mockService := &mockCardReviewService{
				nextCardFn: func(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
					return tc.serviceResult, tc.serviceError
				},
			}

			// Create the handler
			testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockCardService := &mockCardService{}
			handler := NewCardHandler(mockService, mockCardService, testLogger)

			// Create a request
			req, err := http.NewRequest("GET", "/cards/next", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Add user ID to context if needed
			if tc.userIDInCtx != uuid.Nil {
				ctx := context.WithValue(req.Context(), shared.UserIDContextKey, tc.userIDInCtx)
				req = req.WithContext(ctx)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.GetNextReviewCard(rr, req)

			// Check status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf(
					"handler returned wrong status code: got %v want %v",
					status,
					tc.expectedStatus,
				)
			}

			// Check body existence
			if tc.hasBody && rr.Body.Len() == 0 {
				t.Errorf("expected response body, but got empty body")
			} else if !tc.hasBody && rr.Body.Len() > 0 {
				t.Errorf("expected empty body, but got response body: %s", rr.Body.String())
			}

			// If success case with body, validate the response structure
			if tc.expectedStatus == http.StatusOK {
				var response CardResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Errorf("failed to decode response body: %v", err)
					return
				}

				// Validate the key fields
				if response.ID != cardID.String() {
					t.Errorf(
						"wrong card ID in response: got %v want %v",
						response.ID,
						cardID.String(),
					)
				}
				if response.UserID != userID.String() {
					t.Errorf(
						"wrong user ID in response: got %v want %v",
						response.UserID,
						userID.String(),
					)
				}
				if response.MemoID != memoID.String() {
					t.Errorf(
						"wrong memo ID in response: got %v want %v",
						response.MemoID,
						memoID.String(),
					)
				}

				// Validate content for valid JSON case
				if tc.name == "Success" {
					content, ok := response.Content.(map[string]interface{})
					if !ok {
						t.Errorf("content was not of expected type: got %T", response.Content)
					} else {
						if content["front"] != "What is the capital of France?" {
							t.Errorf("wrong front content: got %v", content["front"])
						}
						if content["back"] != "Paris" {
							t.Errorf("wrong back content: got %v", content["back"])
						}
					}
				}

				// Check that unmarshalable content is handled gracefully
				if tc.name == "Service Returns Card But With Unmarshalable Content" {
					_, ok := response.Content.(string)
					if !ok {
						t.Errorf(
							"expected string content for invalid JSON, got %T",
							response.Content,
						)
					}
				}
			}
		})
	}
}

func TestSubmitAnswer(t *testing.T) {
	userID := uuid.New()
	cardID := uuid.New()

	now := time.Now().UTC()
	oneHourLater := now.Add(time.Hour)

	// Create a sample stats object
	sampleStats := &domain.UserCardStats{
		UserID:             userID,
		CardID:             cardID,
		Interval:           1,
		EaseFactor:         2.5,
		ConsecutiveCorrect: 1,
		LastReviewedAt:     now,
		NextReviewAt:       oneHourLater,
		ReviewCount:        1,
		CreatedAt:          now.Add(-time.Hour),
		UpdatedAt:          now,
	}

	tests := []struct {
		name            string
		userIDInCtx     uuid.UUID
		cardIDInPath    string
		requestBody     map[string]string
		serviceResult   *domain.UserCardStats
		serviceError    error
		expectedStatus  int
		expectedErrCode string
	}{
		{
			name:            "Success",
			userIDInCtx:     userID,
			cardIDInPath:    cardID.String(),
			requestBody:     map[string]string{"outcome": "good"},
			serviceResult:   sampleStats,
			serviceError:    nil,
			expectedStatus:  http.StatusOK,
			expectedErrCode: "",
		},
		{
			name:            "Card Not Found",
			userIDInCtx:     userID,
			cardIDInPath:    cardID.String(),
			requestBody:     map[string]string{"outcome": "good"},
			serviceResult:   nil,
			serviceError:    card_review.ErrCardNotFound,
			expectedStatus:  http.StatusNotFound,
			expectedErrCode: "Card not found",
		},
		{
			name:            "Card Not Owned",
			userIDInCtx:     userID,
			cardIDInPath:    cardID.String(),
			requestBody:     map[string]string{"outcome": "good"},
			serviceResult:   nil,
			serviceError:    card_review.ErrCardNotOwned,
			expectedStatus:  http.StatusForbidden,
			expectedErrCode: "You do not own this card",
		},
		{
			name:            "Invalid Answer",
			userIDInCtx:     userID,
			cardIDInPath:    cardID.String(),
			requestBody:     map[string]string{"outcome": "good"},
			serviceResult:   nil,
			serviceError:    card_review.ErrInvalidAnswer,
			expectedStatus:  http.StatusBadRequest,
			expectedErrCode: "Invalid answer",
		},
		{
			name:            "Other Error",
			userIDInCtx:     userID,
			cardIDInPath:    cardID.String(),
			requestBody:     map[string]string{"outcome": "good"},
			serviceResult:   nil,
			serviceError:    errors.New("database error"),
			expectedStatus:  http.StatusInternalServerError,
			expectedErrCode: "Failed to submit answer",
		},
		{
			name:            "Missing User ID",
			userIDInCtx:     uuid.Nil,
			cardIDInPath:    cardID.String(),
			requestBody:     map[string]string{"outcome": "good"},
			serviceResult:   nil,
			serviceError:    nil,
			expectedStatus:  http.StatusUnauthorized,
			expectedErrCode: "Unauthorized operation",
		},
		{
			name:            "Invalid Card ID Format",
			userIDInCtx:     userID,
			cardIDInPath:    "not-a-uuid",
			requestBody:     map[string]string{"outcome": "good"},
			serviceResult:   nil,
			serviceError:    nil,
			expectedStatus:  http.StatusBadRequest,
			expectedErrCode: "Invalid ID",
		},
		{
			name:            "Missing Outcome Field",
			userIDInCtx:     userID,
			cardIDInPath:    cardID.String(),
			requestBody:     map[string]string{},
			serviceResult:   nil,
			serviceError:    nil,
			expectedStatus:  http.StatusBadRequest,
			expectedErrCode: "Invalid Outcome: required field",
		},
		{
			name:            "Invalid Outcome Value",
			userIDInCtx:     userID,
			cardIDInPath:    cardID.String(),
			requestBody:     map[string]string{"outcome": "invalid"},
			serviceResult:   nil,
			serviceError:    nil,
			expectedStatus:  http.StatusBadRequest,
			expectedErrCode: "Invalid Outcome: invalid value",
		},
		{
			name:            "Missing Card ID in Path",
			userIDInCtx:     userID,
			cardIDInPath:    "", // Empty card ID
			requestBody:     map[string]string{"outcome": "good"},
			serviceResult:   nil,
			serviceError:    nil,
			expectedStatus:  http.StatusBadRequest,
			expectedErrCode: "Validation failed",
		},
		{
			name:            "Invalid JSON in Request Body",
			userIDInCtx:     userID,
			cardIDInPath:    cardID.String(),
			requestBody:     nil, // Will send invalid JSON
			serviceResult:   nil,
			serviceError:    nil,
			expectedStatus:  http.StatusBadRequest,
			expectedErrCode: "Validation error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock service that returns the test case's result
			mockService := &mockCardReviewService{
				submitAnswerFn: func(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error) {
					// Only check these in the success case
					if tc.serviceError == nil && tc.userIDInCtx != uuid.Nil &&
						tc.cardIDInPath != "" &&
						tc.cardIDInPath != "not-a-uuid" &&
						tc.requestBody != nil {
						if userID != tc.userIDInCtx {
							t.Errorf(
								"wrong user ID passed to service: got %v want %v",
								userID,
								tc.userIDInCtx,
							)
						}
						expectedCardID, _ := uuid.Parse(tc.cardIDInPath)
						if cardID != expectedCardID {
							t.Errorf(
								"wrong card ID passed to service: got %v want %v",
								cardID,
								expectedCardID,
							)
						}
						if string(answer.Outcome) != tc.requestBody["outcome"] {
							t.Errorf(
								"wrong outcome passed to service: got %v want %v",
								answer.Outcome,
								tc.requestBody["outcome"],
							)
						}
					}
					return tc.serviceResult, tc.serviceError
				},
			}

			// Create the handler
			testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockCardService := &mockCardService{}
			handler := NewCardHandler(mockService, mockCardService, testLogger)

			// Create request body
			var jsonBody []byte
			var req *http.Request
			var err error

			if tc.requestBody != nil {
				jsonBody, _ = json.Marshal(tc.requestBody)
				req, err = http.NewRequest(
					"POST",
					"/cards/"+tc.cardIDInPath+"/answer",
					bytes.NewBuffer(jsonBody),
				)
			} else {
				// Send invalid JSON
				req, err = http.NewRequest("POST", "/cards/"+tc.cardIDInPath+"/answer", bytes.NewBuffer([]byte(`{"outcome": invalid-json`)))
			}

			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Add user ID to context if needed
			if tc.userIDInCtx != uuid.Nil {
				ctx := context.WithValue(req.Context(), shared.UserIDContextKey, tc.userIDInCtx)
				req = req.WithContext(ctx)
			}

			// Create a chi context with URL parameters
			rctx := chi.NewRouteContext()
			if tc.cardIDInPath != "" {
				rctx.URLParams.Add("id", tc.cardIDInPath)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.SubmitAnswer(rr, req)

			// Check status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf(
					"handler returned wrong status code: got %v want %v",
					status,
					tc.expectedStatus,
				)
			}

			// For non-success responses, check the error message
			if tc.expectedStatus != http.StatusOK {
				var errResp shared.ErrorResponse
				if err := json.NewDecoder(rr.Body).Decode(&errResp); err == nil {
					// Only check if the error starts with the expected message
					// (we don't want the test to be too brittle with exact message matching)
					if !strings.HasPrefix(errResp.Error, tc.expectedErrCode) {
						t.Errorf(
							"wrong error message: expected to start with %q, got %q",
							tc.expectedErrCode,
							errResp.Error,
						)
					}
				}
			} else {
				// For success responses, check the response structure
				var response UserCardStatsResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Errorf("failed to decode response body: %v", err)
					return
				}

				// Validate the key fields
				if response.UserID != userID.String() {
					t.Errorf("wrong user ID in response: got %v want %v", response.UserID, userID.String())
				}
				if response.CardID != cardID.String() {
					t.Errorf("wrong card ID in response: got %v want %v", response.CardID, cardID.String())
				}
				if response.Interval != sampleStats.Interval {
					t.Errorf("wrong interval in response: got %v want %v", response.Interval, sampleStats.Interval)
				}
				if response.EaseFactor != sampleStats.EaseFactor {
					t.Errorf("wrong ease factor in response: got %v want %v", response.EaseFactor, sampleStats.EaseFactor)
				}
				if response.ConsecutiveCorrect != sampleStats.ConsecutiveCorrect {
					t.Errorf("wrong consecutive correct in response: got %v want %v", response.ConsecutiveCorrect, sampleStats.ConsecutiveCorrect)
				}
				if !response.NextReviewAt.Equal(sampleStats.NextReviewAt) {
					t.Errorf("wrong next review at in response: got %v want %v", response.NextReviewAt, sampleStats.NextReviewAt)
				}
				if response.ReviewCount != sampleStats.ReviewCount {
					t.Errorf("wrong review count in response: got %v want %v", response.ReviewCount, sampleStats.ReviewCount)
				}
			}
		})
	}
}

func TestNewCardHandler(t *testing.T) {
	mockReviewService := &mockCardReviewService{}
	mockCardService := &mockCardService{}

	// Test with valid logger
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewCardHandler(mockReviewService, mockCardService, testLogger)

	if handler == nil {
		t.Fatal("expected handler to be created")
	}

	// Test with nil logger should panic
	assert.Panics(t, func() {
		NewCardHandler(mockReviewService, mockCardService, nil)
	})

	if handler.cardReviewService == nil {
		t.Error("expected cardReviewService to be set")
	}

	if handler.cardService == nil {
		t.Error("expected cardService to be set")
	}

	// Validator now uses shared.Validate
	if handler.logger == nil {
		t.Error("expected default logger to be set")
	}
}

func TestEditCard(t *testing.T) {
	// Setup common test variables
	userID := uuid.New()
	cardID := uuid.New()

	tests := []struct {
		name                string
		requestCardID       string
		requestUserID       uuid.UUID
		requestBody         []byte
		mockServiceFn       func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error
		expectedStatusCode  int
		expectedErrContains string // Partial error message for checking
	}{
		{
			name:          "Success",
			requestCardID: cardID.String(),
			requestUserID: userID,
			requestBody:   []byte(`{"content": {"front": "Question", "back": "Answer"}}`),
			mockServiceFn: func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error {
				return nil
			},
			expectedStatusCode:  http.StatusNoContent,
			expectedErrContains: "",
		},
		{
			name:                "Invalid Card ID",
			requestCardID:       "not-a-uuid",
			requestUserID:       userID,
			requestBody:         []byte(`{"content": {"front": "Question", "back": "Answer"}}`),
			mockServiceFn:       nil, // No mock behavior needed
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrContains: "Invalid ID",
		},
		{
			name:          "Card Not Found",
			requestCardID: cardID.String(),
			requestUserID: userID,
			requestBody:   []byte(`{"content": {"front": "Question", "back": "Answer"}}`),
			mockServiceFn: func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error {
				return service.NewCardServiceError("update_card_content", "card not found", store.ErrCardNotFound)
			},
			expectedStatusCode:  http.StatusNotFound,
			expectedErrContains: "not found",
		},
		{
			name:          "Not Owned By User",
			requestCardID: cardID.String(),
			requestUserID: userID,
			requestBody:   []byte(`{"content": {"front": "Question", "back": "Answer"}}`),
			mockServiceFn: func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error {
				return service.NewCardServiceError(
					"update_card_content",
					"card is owned by another user",
					service.ErrNotOwned,
				)
			},
			expectedStatusCode:  http.StatusForbidden,
			expectedErrContains: "do not own",
		},
		{
			name:                "Invalid Request Body",
			requestCardID:       cardID.String(),
			requestUserID:       userID,
			requestBody:         []byte(`{invalid json`),
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrContains: "Validation error",
		},
		{
			name:                "Missing Content Field",
			requestCardID:       cardID.String(),
			requestUserID:       userID,
			requestBody:         []byte(`{}`),
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrContains: "Content",
		},
		{
			name:          "Internal Server Error",
			requestCardID: cardID.String(),
			requestUserID: userID,
			requestBody:   []byte(`{"content": {"front": "Question", "back": "Answer"}}`),
			mockServiceFn: func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error {
				return errors.New("unexpected error")
			},
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrContains: "Failed to update",
		},
		{
			name:                "Missing User ID",
			requestCardID:       cardID.String(),
			requestUserID:       uuid.Nil,
			requestBody:         []byte(`{"content": {"front": "Question", "back": "Answer"}}`),
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrContains: "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockCardReviewService := &mockCardReviewService{}
			mockCardService := &mockCardService{
				updateCardContentFn: tt.mockServiceFn,
			}

			// Create handler
			testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			handler := NewCardHandler(mockCardReviewService, mockCardService, testLogger)

			// Create request
			req, err := http.NewRequest(http.MethodPut, "/cards/"+tt.requestCardID, bytes.NewBuffer(tt.requestBody))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Use chi router to get URL parameters
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.requestCardID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add user ID to context if needed
			if tt.requestUserID != uuid.Nil {
				req = req.WithContext(context.WithValue(req.Context(), shared.UserIDContextKey, tt.requestUserID))
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.EditCard(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			// For error responses, check the error message
			if tt.expectedStatusCode != http.StatusNoContent && tt.expectedErrContains != "" {
				var errResp shared.ErrorResponse
				if err := json.NewDecoder(rr.Body).Decode(&errResp); err == nil {
					assert.Contains(t, errResp.Error, tt.expectedErrContains)
				}
			}
		})
	}
}

func TestDeleteCard(t *testing.T) {
	// Setup common test variables
	userID := uuid.New()
	cardID := uuid.New()

	tests := []struct {
		name                string
		requestCardID       string
		requestUserID       uuid.UUID
		mockServiceFn       func(ctx context.Context, userID, cardID uuid.UUID) error
		expectedStatusCode  int
		expectedErrContains string // Partial error message for checking
	}{
		{
			name:          "Success",
			requestCardID: cardID.String(),
			requestUserID: userID,
			mockServiceFn: func(ctx context.Context, userID, cardID uuid.UUID) error {
				return nil
			},
			expectedStatusCode:  http.StatusNoContent,
			expectedErrContains: "",
		},
		{
			name:                "Invalid Card ID",
			requestCardID:       "not-a-uuid",
			requestUserID:       userID,
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrContains: "Invalid ID",
		},
		{
			name:          "Card Not Found",
			requestCardID: cardID.String(),
			requestUserID: userID,
			mockServiceFn: func(ctx context.Context, userID, cardID uuid.UUID) error {
				return service.NewCardServiceError("delete_card", "card not found", store.ErrCardNotFound)
			},
			expectedStatusCode:  http.StatusNotFound,
			expectedErrContains: "not found",
		},
		{
			name:          "Not Owned By User",
			requestCardID: cardID.String(),
			requestUserID: userID,
			mockServiceFn: func(ctx context.Context, userID, cardID uuid.UUID) error {
				return service.NewCardServiceError(
					"delete_card",
					"card is owned by another user",
					service.ErrNotOwned,
				)
			},
			expectedStatusCode:  http.StatusForbidden,
			expectedErrContains: "do not own",
		},
		{
			name:          "Internal Server Error",
			requestCardID: cardID.String(),
			requestUserID: userID,
			mockServiceFn: func(ctx context.Context, userID, cardID uuid.UUID) error {
				return errors.New("unexpected error")
			},
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrContains: "Failed to delete",
		},
		{
			name:                "Missing User ID",
			requestCardID:       cardID.String(),
			requestUserID:       uuid.Nil,
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrContains: "Unauthorized",
		},
		{
			name:                "Missing Card ID in Path",
			requestCardID:       "", // Empty card ID
			requestUserID:       userID,
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrContains: "Validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockCardReviewService := &mockCardReviewService{}
			mockCardService := &mockCardService{
				deleteCardFn: tt.mockServiceFn,
			}

			// Create handler
			testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			handler := NewCardHandler(mockCardReviewService, mockCardService, testLogger)

			// Create request
			req, err := http.NewRequest(http.MethodDelete, "/cards/"+tt.requestCardID, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Use chi router to get URL parameters
			rctx := chi.NewRouteContext()
			if tt.requestCardID != "" {
				rctx.URLParams.Add("id", tt.requestCardID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add user ID to context if needed
			if tt.requestUserID != uuid.Nil {
				req = req.WithContext(context.WithValue(req.Context(), shared.UserIDContextKey, tt.requestUserID))
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.DeleteCard(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			// For error responses, check the error message
			if tt.expectedStatusCode != http.StatusNoContent && tt.expectedErrContains != "" {
				var errResp shared.ErrorResponse
				if err := json.NewDecoder(rr.Body).Decode(&errResp); err == nil {
					assert.Contains(t, errResp.Error, tt.expectedErrContains)
				}
			}
		})
	}
}

// TestPostponeCard has been moved to card_handler_postpone_test.go
