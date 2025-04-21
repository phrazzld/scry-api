package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/card_review"
)

// mockCardReviewService is a mock implementation of the CardReviewService interface
type mockCardReviewService struct {
	nextCardFn     func(ctx context.Context, userID uuid.UUID) (*domain.Card, error)
	submitAnswerFn func(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, answer card_review.ReviewAnswer) (*domain.UserCardStats, error)
}

func (m *mockCardReviewService) GetNextCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
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
			handler := NewCardHandler(mockService, nil)

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
				t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectedStatus)
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
					t.Errorf("wrong card ID in response: got %v want %v", response.ID, cardID.String())
				}
				if response.UserID != userID.String() {
					t.Errorf("wrong user ID in response: got %v want %v", response.UserID, userID.String())
				}
				if response.MemoID != memoID.String() {
					t.Errorf("wrong memo ID in response: got %v want %v", response.MemoID, memoID.String())
				}

				// Validate content
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
		})
	}
}
