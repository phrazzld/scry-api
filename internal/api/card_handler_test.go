//go:build test || integration || test_without_external_deps

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/card_review"
)

// MockCardReviewService implements card_review.CardReviewService for testing
type MockCardReviewService struct {
	GetNextCardFn  func(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
	SubmitAnswerFn func(ctx context.Context, userID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error)
}

func (m *MockCardReviewService) GetNextCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	if m.GetNextCardFn != nil {
		return m.GetNextCardFn(ctx, userID)
	}
	return nil, nil
}

func (m *MockCardReviewService) SubmitAnswer(
	ctx context.Context,
	userID, cardID uuid.UUID,
	answer card_review.ReviewAnswer,
) (*domain.UserCardStats, error) {
	if m.SubmitAnswerFn != nil {
		return m.SubmitAnswerFn(ctx, userID, cardID, answer)
	}
	return nil, nil
}

// MockCardService implements service.CardService for testing
type MockCardService struct {
	CreateCardsFn       func(ctx context.Context, cards []*domain.Card) error
	GetCardFn           func(ctx context.Context, cardID uuid.UUID) (*domain.Card, error)
	UpdateCardContentFn func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error
	DeleteCardFn        func(ctx context.Context, userID, cardID uuid.UUID) error
	PostponeCardFn      func(ctx context.Context, userID, cardID uuid.UUID, days int) (*domain.UserCardStats, error)
}

func (m *MockCardService) CreateCards(ctx context.Context, cards []*domain.Card) error {
	if m.CreateCardsFn != nil {
		return m.CreateCardsFn(ctx, cards)
	}
	return nil
}

func (m *MockCardService) GetCard(ctx context.Context, cardID uuid.UUID) (*domain.Card, error) {
	if m.GetCardFn != nil {
		return m.GetCardFn(ctx, cardID)
	}
	return nil, nil
}

func (m *MockCardService) UpdateCardContent(
	ctx context.Context,
	userID, cardID uuid.UUID,
	content json.RawMessage,
) error {
	if m.UpdateCardContentFn != nil {
		return m.UpdateCardContentFn(ctx, userID, cardID, content)
	}
	return nil
}

func (m *MockCardService) DeleteCard(ctx context.Context, userID, cardID uuid.UUID) error {
	if m.DeleteCardFn != nil {
		return m.DeleteCardFn(ctx, userID, cardID)
	}
	return nil
}

func (m *MockCardService) PostponeCard(
	ctx context.Context,
	userID, cardID uuid.UUID,
	days int,
) (*domain.UserCardStats, error) {
	if m.PostponeCardFn != nil {
		return m.PostponeCardFn(ctx, userID, cardID, days)
	}
	return nil, nil
}

// Test fixtures
var (
	fixedCardTime  = time.Date(2025, time.May, 1, 10, 0, 0, 0, time.UTC)
	testCardUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testCardID     = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testMemoID     = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)

func TestNewCardHandler(t *testing.T) {
	tests := []struct {
		name              string
		cardReviewService card_review.CardReviewService
		cardService       service.CardService
		logger            *slog.Logger
		wantPanic         bool
	}{
		{
			name:              "successful creation",
			cardReviewService: &MockCardReviewService{},
			cardService:       &MockCardService{},
			logger:            slog.Default(),
			wantPanic:         false,
		},
		{
			name:              "nil logger panics",
			cardReviewService: &MockCardReviewService{},
			cardService:       &MockCardService{},
			logger:            nil,
			wantPanic:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() {
					NewCardHandler(tt.cardReviewService, tt.cardService, tt.logger)
				})
				return
			}

			handler := NewCardHandler(tt.cardReviewService, tt.cardService, tt.logger)
			assert.NotNil(t, handler)
			assert.Equal(t, tt.cardReviewService, handler.cardReviewService)
			assert.Equal(t, tt.cardService, handler.cardService)
			assert.NotNil(t, handler.logger)
		})
	}
}

func TestCardHandler_GetNextReviewCard(t *testing.T) {
	tests := []struct {
		name             string
		setupContext     func(context.Context) context.Context
		setupMock        func(*MockCardReviewService)
		expectedStatus   int
		expectedErrMsg   string
		validateResponse func(t *testing.T, body []byte)
	}{
		{
			name: "successful_card_retrieval",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			setupMock: func(mockService *MockCardReviewService) {
				mockService.GetNextCardFn = func(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
					return &domain.Card{
						ID:        testCardID,
						UserID:    userID,
						MemoID:    testMemoID,
						Content:   json.RawMessage(`{"question": "What is 2+2?", "answer": "4"}`),
						CreatedAt: fixedCardTime,
						UpdatedAt: fixedCardTime,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var resp CardResponse
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				assert.Equal(t, testCardID.String(), resp.ID)
				assert.Equal(t, testCardUserID.String(), resp.UserID)
				assert.Equal(t, testMemoID.String(), resp.MemoID)
				assert.NotNil(t, resp.Content)
			},
		},
		{
			name: "no_cards_due",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			setupMock: func(mockService *MockCardReviewService) {
				mockService.GetNextCardFn = func(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
					return nil, card_review.ErrNoCardsDue
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "missing_user_id",
			setupContext: func(ctx context.Context) context.Context {
				return ctx // No user ID in context
			},
			setupMock: func(mockService *MockCardReviewService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusUnauthorized,
			expectedErrMsg: "Unauthorized operation",
		},
		{
			name: "service_error",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			setupMock: func(mockService *MockCardReviewService) {
				mockService.GetNextCardFn = func(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
					return nil, errors.New("database connection failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedErrMsg: "Failed to get next review card",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCardReviewService := &MockCardReviewService{}
			mockCardService := &MockCardService{}
			tt.setupMock(mockCardReviewService)

			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewCardHandler(mockCardReviewService, mockCardService, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/cards/next", nil)
			req = req.WithContext(tt.setupContext(req.Context()))

			w := httptest.NewRecorder()
			handler.GetNextReviewCard(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedErrMsg != "" {
				var respBody map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &respBody)
				require.NoError(t, err)
				errorMsg, ok := respBody["error"].(string)
				assert.True(t, ok, "Expected error field in response")
				assert.Contains(t, errorMsg, tt.expectedErrMsg)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestCardHandler_SubmitAnswer(t *testing.T) {
	tests := []struct {
		name             string
		setupContext     func(context.Context) context.Context
		cardID           string
		requestBody      interface{}
		setupMock        func(*MockCardReviewService)
		expectedStatus   int
		expectedErrMsg   string
		validateResponse func(t *testing.T, body []byte)
	}{
		{
			name: "successful_answer_submission",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			requestBody: SubmitAnswerRequest{
				Outcome: "good",
			},
			setupMock: func(mockService *MockCardReviewService) {
				mockService.SubmitAnswerFn = func(ctx context.Context, userID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error) {
					return &domain.UserCardStats{
						UserID:             userID,
						CardID:             cardID,
						Interval:           3,
						EaseFactor:         2.5,
						ConsecutiveCorrect: 1,
						LastReviewedAt:     fixedCardTime,
						NextReviewAt:       fixedCardTime.AddDate(0, 0, 3),
						ReviewCount:        1,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var resp UserCardStatsResponse
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				assert.Equal(t, testCardUserID.String(), resp.UserID)
				assert.Equal(t, testCardID.String(), resp.CardID)
				assert.Equal(t, 3, resp.Interval)
				assert.Equal(t, 2.5, resp.EaseFactor)
			},
		},
		{
			name: "missing_user_id",
			setupContext: func(ctx context.Context) context.Context {
				return ctx // No user ID in context
			},
			cardID: testCardID.String(),
			requestBody: SubmitAnswerRequest{
				Outcome: "good",
			},
			setupMock: func(mockService *MockCardReviewService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusUnauthorized,
			expectedErrMsg: "Unauthorized operation",
		},
		{
			name: "invalid_card_id",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: "invalid-uuid",
			requestBody: SubmitAnswerRequest{
				Outcome: "good",
			},
			setupMock: func(mockService *MockCardReviewService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "invalid format",
		},
		{
			name: "invalid_request_format",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			requestBody: `{
				"outcome": "invalid JSON
			}`,
			setupMock: func(mockService *MockCardReviewService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "Validation error",
		},
		{
			name: "invalid_outcome_value",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			requestBody: SubmitAnswerRequest{
				Outcome: "invalid",
			},
			setupMock: func(mockService *MockCardReviewService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "Invalid Outcome",
		},
		{
			name: "service_error",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			requestBody: SubmitAnswerRequest{
				Outcome: "good",
			},
			setupMock: func(mockService *MockCardReviewService) {
				mockService.SubmitAnswerFn = func(ctx context.Context, userID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error) {
					return nil, errors.New("database connection failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedErrMsg: "Failed to submit answer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCardReviewService := &MockCardReviewService{}
			mockCardService := &MockCardService{}
			tt.setupMock(mockCardReviewService)

			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewCardHandler(mockCardReviewService, mockCardService, logger)

			// Create request body
			var reqBody []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/cards/"+tt.cardID+"/answer", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tt.setupContext(req.Context()))

			// Set up chi router context for path parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.cardID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.SubmitAnswer(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedErrMsg != "" {
				var respBody map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &respBody)
				require.NoError(t, err)
				errorMsg, ok := respBody["error"].(string)
				assert.True(t, ok, "Expected error field in response")
				assert.Contains(t, errorMsg, tt.expectedErrMsg)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestCardHandler_EditCard(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(context.Context) context.Context
		cardID         string
		requestBody    interface{}
		setupMock      func(*MockCardService)
		expectedStatus int
		expectedErrMsg string
	}{
		{
			name: "successful_card_edit",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			requestBody: EditCardRequest{
				Content: json.RawMessage(`{"question": "What is 3+3?", "answer": "6"}`),
			},
			setupMock: func(mockService *MockCardService) {
				mockService.UpdateCardContentFn = func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "missing_user_id",
			setupContext: func(ctx context.Context) context.Context {
				return ctx // No user ID in context
			},
			cardID: testCardID.String(),
			requestBody: EditCardRequest{
				Content: json.RawMessage(`{"question": "What is 3+3?", "answer": "6"}`),
			},
			setupMock: func(mockService *MockCardService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusUnauthorized,
			expectedErrMsg: "Unauthorized operation",
		},
		{
			name: "invalid_card_id",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: "invalid-uuid",
			requestBody: EditCardRequest{
				Content: json.RawMessage(`{"question": "What is 3+3?", "answer": "6"}`),
			},
			setupMock: func(mockService *MockCardService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "invalid format",
		},
		{
			name: "missing_content",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID:      testCardID.String(),
			requestBody: `{}`, // Empty JSON object without required content field
			setupMock: func(mockService *MockCardService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "required",
		},
		{
			name: "service_error",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			requestBody: EditCardRequest{
				Content: json.RawMessage(`{"question": "What is 3+3?", "answer": "6"}`),
			},
			setupMock: func(mockService *MockCardService) {
				mockService.UpdateCardContentFn = func(ctx context.Context, userID, cardID uuid.UUID, content json.RawMessage) error {
					return errors.New("database connection failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedErrMsg: "Failed to update card content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCardReviewService := &MockCardReviewService{}
			mockCardService := &MockCardService{}
			tt.setupMock(mockCardService)

			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewCardHandler(mockCardReviewService, mockCardService, logger)

			// Create request body
			var reqBody []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/cards/"+tt.cardID, bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tt.setupContext(req.Context()))

			// Set up chi router context for path parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.cardID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.EditCard(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedErrMsg != "" {
				var respBody map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &respBody)
				require.NoError(t, err)
				errorMsg, ok := respBody["error"].(string)
				assert.True(t, ok, "Expected error field in response")
				assert.Contains(t, errorMsg, tt.expectedErrMsg)
			}
		})
	}
}

func TestCardHandler_DeleteCard(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(context.Context) context.Context
		cardID         string
		setupMock      func(*MockCardService)
		expectedStatus int
		expectedErrMsg string
	}{
		{
			name: "successful_card_deletion",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			setupMock: func(mockService *MockCardService) {
				mockService.DeleteCardFn = func(ctx context.Context, userID, cardID uuid.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "missing_user_id",
			setupContext: func(ctx context.Context) context.Context {
				return ctx // No user ID in context
			},
			cardID: testCardID.String(),
			setupMock: func(mockService *MockCardService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusUnauthorized,
			expectedErrMsg: "Unauthorized operation",
		},
		{
			name: "invalid_card_id",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: "invalid-uuid",
			setupMock: func(mockService *MockCardService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "invalid format",
		},
		{
			name: "service_error",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			setupMock: func(mockService *MockCardService) {
				mockService.DeleteCardFn = func(ctx context.Context, userID, cardID uuid.UUID) error {
					return errors.New("database connection failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedErrMsg: "Failed to delete card",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCardReviewService := &MockCardReviewService{}
			mockCardService := &MockCardService{}
			tt.setupMock(mockCardService)

			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewCardHandler(mockCardReviewService, mockCardService, logger)

			req := httptest.NewRequest(http.MethodDelete, "/api/cards/"+tt.cardID, nil)
			req = req.WithContext(tt.setupContext(req.Context()))

			// Set up chi router context for path parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.cardID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.DeleteCard(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedErrMsg != "" {
				var respBody map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &respBody)
				require.NoError(t, err)
				errorMsg, ok := respBody["error"].(string)
				assert.True(t, ok, "Expected error field in response")
				assert.Contains(t, errorMsg, tt.expectedErrMsg)
			}
		})
	}
}

func TestCardHandler_PostponeCard(t *testing.T) {
	tests := []struct {
		name             string
		setupContext     func(context.Context) context.Context
		cardID           string
		requestBody      interface{}
		setupMock        func(*MockCardService)
		expectedStatus   int
		expectedErrMsg   string
		validateResponse func(t *testing.T, body []byte)
	}{
		{
			name: "successful_card_postponement",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			requestBody: PostponeCardRequest{
				Days: 5,
			},
			setupMock: func(mockService *MockCardService) {
				mockService.PostponeCardFn = func(ctx context.Context, userID, cardID uuid.UUID, days int) (*domain.UserCardStats, error) {
					return &domain.UserCardStats{
						UserID:             userID,
						CardID:             cardID,
						Interval:           7,
						EaseFactor:         2.5,
						ConsecutiveCorrect: 2,
						LastReviewedAt:     fixedCardTime,
						NextReviewAt:       fixedCardTime.AddDate(0, 0, days),
						ReviewCount:        2,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var resp UserCardStatsResponse
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				assert.Equal(t, testCardUserID.String(), resp.UserID)
				assert.Equal(t, testCardID.String(), resp.CardID)
				assert.Equal(t, 7, resp.Interval)
				assert.Equal(t, 2.5, resp.EaseFactor)
			},
		},
		{
			name: "missing_user_id",
			setupContext: func(ctx context.Context) context.Context {
				return ctx // No user ID in context
			},
			cardID: testCardID.String(),
			requestBody: PostponeCardRequest{
				Days: 5,
			},
			setupMock: func(mockService *MockCardService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusUnauthorized,
			expectedErrMsg: "Unauthorized operation",
		},
		{
			name: "invalid_card_id",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: "invalid-uuid",
			requestBody: PostponeCardRequest{
				Days: 5,
			},
			setupMock: func(mockService *MockCardService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "invalid format",
		},
		{
			name: "invalid_days_value",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			requestBody: PostponeCardRequest{
				Days: 0, // Invalid: must be at least 1
			},
			setupMock: func(mockService *MockCardService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "Invalid Days",
		},
		{
			name: "service_error",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, testCardUserID)
			},
			cardID: testCardID.String(),
			requestBody: PostponeCardRequest{
				Days: 5,
			},
			setupMock: func(mockService *MockCardService) {
				mockService.PostponeCardFn = func(ctx context.Context, userID, cardID uuid.UUID, days int) (*domain.UserCardStats, error) {
					return nil, errors.New("database connection failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedErrMsg: "Failed to postpone card review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCardReviewService := &MockCardReviewService{}
			mockCardService := &MockCardService{}
			tt.setupMock(mockCardService)

			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewCardHandler(mockCardReviewService, mockCardService, logger)

			// Create request body
			reqBody, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/cards/"+tt.cardID+"/postpone", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tt.setupContext(req.Context()))

			// Set up chi router context for path parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.cardID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.PostponeCard(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedErrMsg != "" {
				var respBody map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &respBody)
				require.NoError(t, err)
				errorMsg, ok := respBody["error"].(string)
				assert.True(t, ok, "Expected error field in response")
				assert.Contains(t, errorMsg, tt.expectedErrMsg)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, w.Body.Bytes())
			}
		})
	}
}

// Helper function tests
func TestCardHandler_HelperFunctions(t *testing.T) {
	t.Run("cardToResponse", func(t *testing.T) {
		card := &domain.Card{
			ID:        testCardID,
			UserID:    testCardUserID,
			MemoID:    testMemoID,
			Content:   json.RawMessage(`{"question": "What is 2+2?", "answer": "4"}`),
			CreatedAt: fixedCardTime,
			UpdatedAt: fixedCardTime,
		}

		response := cardToResponse(card)

		assert.Equal(t, testCardID.String(), response.ID)
		assert.Equal(t, testCardUserID.String(), response.UserID)
		assert.Equal(t, testMemoID.String(), response.MemoID)
		assert.NotNil(t, response.Content)
		assert.Equal(t, fixedCardTime, response.CreatedAt)
		assert.Equal(t, fixedCardTime, response.UpdatedAt)

		// Verify content was properly unmarshaled
		contentMap, ok := response.Content.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "What is 2+2?", contentMap["question"])
		assert.Equal(t, "4", contentMap["answer"])
	})

	t.Run("cardToResponse_invalid_json", func(t *testing.T) {
		card := &domain.Card{
			ID:        testCardID,
			UserID:    testCardUserID,
			MemoID:    testMemoID,
			Content:   json.RawMessage(`invalid json`),
			CreatedAt: fixedCardTime,
			UpdatedAt: fixedCardTime,
		}

		response := cardToResponse(card)

		// Should fall back to string representation
		assert.Equal(t, "invalid json", response.Content)
	})

	t.Run("statsToResponse", func(t *testing.T) {
		stats := &domain.UserCardStats{
			UserID:             testCardUserID,
			CardID:             testCardID,
			Interval:           3,
			EaseFactor:         2.5,
			ConsecutiveCorrect: 1,
			LastReviewedAt:     fixedCardTime,
			NextReviewAt:       fixedCardTime.AddDate(0, 0, 3),
			ReviewCount:        1,
		}

		response := statsToResponse(stats)

		assert.Equal(t, testCardUserID.String(), response.UserID)
		assert.Equal(t, testCardID.String(), response.CardID)
		assert.Equal(t, 3, response.Interval)
		assert.Equal(t, 2.5, response.EaseFactor)
		assert.Equal(t, 1, response.ConsecutiveCorrect)
		assert.Equal(t, fixedCardTime, response.LastReviewedAt)
		assert.Equal(t, fixedCardTime.AddDate(0, 0, 3), response.NextReviewAt)
		assert.Equal(t, 1, response.ReviewCount)
	})
}
