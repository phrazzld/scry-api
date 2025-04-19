package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMemoService is a mock implementation of the MemoService
type MockMemoService struct {
	mock.Mock
}

func (m *MockMemoService) CreateMemoAndEnqueueTask(
	ctx context.Context,
	userID uuid.UUID,
	text string,
) (*domain.Memo, error) {
	args := m.Called(ctx, userID, text)
	memo, _ := args.Get(0).(*domain.Memo)
	return memo, args.Error(1)
}

func (m *MockMemoService) UpdateMemoStatus(
	ctx context.Context,
	memoID uuid.UUID,
	status domain.MemoStatus,
) error {
	args := m.Called(ctx, memoID, status)
	return args.Error(0)
}

func (m *MockMemoService) GetMemo(
	ctx context.Context,
	memoID uuid.UUID,
) (*domain.Memo, error) {
	args := m.Called(ctx, memoID)
	memo, _ := args.Get(0).(*domain.Memo)
	return memo, args.Error(1)
}

func TestMemoHandler_CreateMemo(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		// Setup
		userID := uuid.New()
		memoText := "This is a test memo"
		mockService := &MockMemoService{}

		// Setup mock response
		memo := &domain.Memo{
			ID:        uuid.New(),
			UserID:    userID,
			Text:      memoText,
			Status:    domain.MemoStatusPending,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		mockService.On("CreateMemoAndEnqueueTask", mock.Anything, userID, memoText).Return(memo, nil)

		// Create handler
		handler := NewMemoHandler(mockService)

		// Create request
		requestBody := CreateMemoRequest{
			Text: memoText,
		}
		jsonData, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/memos", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		// Set userID in request context (simulating auth middleware)
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), shared.UserIDContextKey, userID))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.CreateMemo(w, req)

		// Assertions
		assert.Equal(t, http.StatusAccepted, w.Code)

		var response MemoResponse
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, memo.ID.String(), response.ID)
		assert.Equal(t, memo.UserID.String(), response.UserID)
		assert.Equal(t, memo.Text, response.Text)
		assert.Equal(t, string(memo.Status), response.Status)
		assert.NotEmpty(t, response.CreatedAt)
		assert.NotEmpty(t, response.UpdatedAt)

		// Verify mocks
		mockService.AssertExpectations(t)
	})

	t.Run("missing user ID", func(t *testing.T) {
		// Setup
		mockService := &MockMemoService{}
		handler := NewMemoHandler(mockService)

		// Create request with missing user ID in context
		req := httptest.NewRequest("POST", "/api/memos", bytes.NewBufferString(`{"text":"Test memo"}`))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.CreateMemo(w, req)

		// Assertions
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockService.AssertNotCalled(t, "CreateMemoAndEnqueueTask")
	})

	t.Run("invalid request body", func(t *testing.T) {
		// Setup
		userID := uuid.New()
		mockService := &MockMemoService{}
		handler := NewMemoHandler(mockService)

		// Create invalid request (missing text field)
		req := httptest.NewRequest("POST", "/api/memos", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")

		// Set userID in request context
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), shared.UserIDContextKey, userID))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.CreateMemo(w, req)

		// Assertions
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertNotCalled(t, "CreateMemoAndEnqueueTask")
	})

	t.Run("service error", func(t *testing.T) {
		// Setup
		userID := uuid.New()
		memoText := "This is a test memo"
		mockService := &MockMemoService{}

		// Setup mock to return error
		mockError := errors.New("service error")
		mockService.On("CreateMemoAndEnqueueTask", mock.Anything, userID, memoText).Return(nil, mockError)

		// Create handler
		handler := NewMemoHandler(mockService)

		// Create request
		requestBody := CreateMemoRequest{
			Text: memoText,
		}
		jsonData, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/memos", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		// Set userID in request context
		rctx := chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), shared.UserIDContextKey, userID))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.CreateMemo(w, req)

		// Assertions
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}
