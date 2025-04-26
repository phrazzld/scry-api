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
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestPostponeCard(t *testing.T) {
	// Setup common test variables
	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now().UTC()

	// Sample stats for the response
	sampleStats := &domain.UserCardStats{
		UserID:             userID,
		CardID:             cardID,
		Interval:           1,
		EaseFactor:         2.5,
		ConsecutiveCorrect: 1,
		LastReviewedAt:     now.Add(-24 * time.Hour),
		NextReviewAt:       now.AddDate(0, 0, 7), // Postponed to 7 days from now
		ReviewCount:        1,
		CreatedAt:          now.Add(-7 * 24 * time.Hour),
		UpdatedAt:          now,
	}

	tests := []struct {
		name                string
		requestCardID       string
		requestUserID       uuid.UUID
		requestBody         []byte
		mockServiceFn       func(ctx context.Context, userId, cardId uuid.UUID, days int) (*domain.UserCardStats, error)
		expectedStatusCode  int
		expectedErrContains string
	}{
		{
			name:          "Success",
			requestCardID: cardID.String(),
			requestUserID: userID,
			requestBody:   []byte(`{"days": 7}`),
			mockServiceFn: func(ctx context.Context, userId, cardId uuid.UUID, days int) (*domain.UserCardStats, error) {
				// Verify input parameters
				if userId != userID {
					t.Errorf("expected userID %s, got %s", userID, userId)
				}
				if cardId != cardID {
					t.Errorf("expected cardID %s, got %s", cardID, cardId)
				}
				if days != 7 {
					t.Errorf("expected days 7, got %d", days)
				}
				return sampleStats, nil
			},
			expectedStatusCode:  http.StatusOK,
			expectedErrContains: "",
		},
		{
			name:                "Invalid Card ID",
			requestCardID:       "not-a-uuid",
			requestUserID:       userID,
			requestBody:         []byte(`{"days": 7}`),
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrContains: "Invalid ID",
		},
		{
			name:          "Card Not Found",
			requestCardID: cardID.String(),
			requestUserID: userID,
			requestBody:   []byte(`{"days": 7}`),
			mockServiceFn: func(ctx context.Context, userId, cardId uuid.UUID, days int) (*domain.UserCardStats, error) {
				return nil, service.NewCardServiceError("postpone_card", "card not found", store.ErrCardNotFound)
			},
			expectedStatusCode:  http.StatusNotFound,
			expectedErrContains: "not found",
		},
		{
			name:          "Stats Not Found",
			requestCardID: cardID.String(),
			requestUserID: userID,
			requestBody:   []byte(`{"days": 7}`),
			mockServiceFn: func(ctx context.Context, userId, cardId uuid.UUID, days int) (*domain.UserCardStats, error) {
				return nil, service.NewCardServiceError(
					"postpone_card",
					"user card statistics not found",
					service.ErrStatsNotFound,
				)
			},
			expectedStatusCode:  http.StatusNotFound,
			expectedErrContains: "not found",
		},
		{
			name:          "Not Owned By User",
			requestCardID: cardID.String(),
			requestUserID: userID,
			requestBody:   []byte(`{"days": 7}`),
			mockServiceFn: func(ctx context.Context, userId, cardId uuid.UUID, days int) (*domain.UserCardStats, error) {
				return nil, service.NewCardServiceError(
					"postpone_card",
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
			name:                "Missing Days Field",
			requestCardID:       cardID.String(),
			requestUserID:       userID,
			requestBody:         []byte(`{}`),
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrContains: "Days",
		},
		{
			name:                "Invalid Days Value (Zero)",
			requestCardID:       cardID.String(),
			requestUserID:       userID,
			requestBody:         []byte(`{"days": 0}`),
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrContains: "Days",
		},
		{
			name:                "Invalid Days Value (Negative)",
			requestCardID:       cardID.String(),
			requestUserID:       userID,
			requestBody:         []byte(`{"days": -1}`),
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusBadRequest,
			expectedErrContains: "Days",
		},
		{
			name:          "Internal Server Error",
			requestCardID: cardID.String(),
			requestUserID: userID,
			requestBody:   []byte(`{"days": 7}`),
			mockServiceFn: func(ctx context.Context, userId, cardId uuid.UUID, days int) (*domain.UserCardStats, error) {
				return nil, errors.New("unexpected error")
			},
			expectedStatusCode:  http.StatusInternalServerError,
			expectedErrContains: "Failed to postpone",
		},
		{
			name:                "Missing User ID",
			requestCardID:       cardID.String(),
			requestUserID:       uuid.Nil,
			requestBody:         []byte(`{"days": 7}`),
			mockServiceFn:       nil,
			expectedStatusCode:  http.StatusUnauthorized,
			expectedErrContains: "Unauthorized",
		},
		{
			name:                "Missing Card ID in Path",
			requestCardID:       "", // Empty card ID
			requestUserID:       userID,
			requestBody:         []byte(`{"days": 7}`),
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
				postponeCardFn: tt.mockServiceFn,
			}

			// Create handler
			testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			handler := NewCardHandler(mockCardReviewService, mockCardService, testLogger)

			// Create request
			req, err := http.NewRequest(
				http.MethodPost,
				"/cards/"+tt.requestCardID+"/postpone",
				bytes.NewBuffer(tt.requestBody),
			)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

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
			handler.PostponeCard(rr, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			// For error responses, check the error message
			if tt.expectedStatusCode != http.StatusOK && tt.expectedErrContains != "" {
				var errResp shared.ErrorResponse
				if err := json.NewDecoder(rr.Body).Decode(&errResp); err == nil {
					assert.Contains(t, errResp.Error, tt.expectedErrContains)
				}
			}

			// For success case, validate response structure
			if tt.expectedStatusCode == http.StatusOK {
				var response UserCardStatsResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Errorf("failed to decode response body: %v", err)
					return
				}

				// Validate key fields
				assert.Equal(t, userID.String(), response.UserID)
				assert.Equal(t, cardID.String(), response.CardID)
				assert.Equal(t, sampleStats.Interval, response.Interval)
				assert.Equal(t, sampleStats.EaseFactor, response.EaseFactor)
				assert.Equal(t, sampleStats.ConsecutiveCorrect, response.ConsecutiveCorrect)
				assert.Equal(t, sampleStats.NextReviewAt, response.NextReviewAt)
				assert.Equal(t, sampleStats.ReviewCount, response.ReviewCount)
			}
		})
	}
}
