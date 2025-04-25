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

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMemoService is a mock implementation of service.MemoService for testing
type MockMemoService struct {
	CreateMemoAndEnqueueTaskFn func(ctx context.Context, userID uuid.UUID, text string) (*domain.Memo, error)
	UpdateMemoStatusFn         func(ctx context.Context, memoID uuid.UUID, status domain.MemoStatus) error
	GetMemoFn                  func(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error)
}

// CreateMemoAndEnqueueTask implements service.MemoService
func (m *MockMemoService) CreateMemoAndEnqueueTask(
	ctx context.Context,
	userID uuid.UUID,
	text string,
) (*domain.Memo, error) {
	if m.CreateMemoAndEnqueueTaskFn != nil {
		return m.CreateMemoAndEnqueueTaskFn(ctx, userID, text)
	}
	return nil, nil
}

// UpdateMemoStatus implements service.MemoService
func (m *MockMemoService) UpdateMemoStatus(
	ctx context.Context,
	memoID uuid.UUID,
	status domain.MemoStatus,
) error {
	if m.UpdateMemoStatusFn != nil {
		return m.UpdateMemoStatusFn(ctx, memoID, status)
	}
	return nil
}

// GetMemo implements service.MemoService
func (m *MockMemoService) GetMemo(ctx context.Context, memoID uuid.UUID) (*domain.Memo, error) {
	if m.GetMemoFn != nil {
		return m.GetMemoFn(ctx, memoID)
	}
	return nil, nil
}

// TestMemoHandler_CreateMemo tests the CreateMemo handler functionality.
func TestMemoHandler_CreateMemo(t *testing.T) {
	// Setup fixed values for consistent testing
	fixedUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fixedMemoID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	fixedTime := time.Date(2025, time.April, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		setupContext    func(context.Context) context.Context
		requestBody     interface{}
		setupMock       func(*MockMemoService)
		expectedStatus  int
		expectedErrMsg  string
		expectedMemoID  string
		expectedUserID  string
		expectedMemoTxt string
	}{
		{
			name: "successful_memo_creation",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, fixedUserID)
			},
			requestBody: CreateMemoRequest{
				Text: "Test memo content",
			},
			setupMock: func(ms *MockMemoService) {
				ms.CreateMemoAndEnqueueTaskFn = func(ctx context.Context, userID uuid.UUID, text string) (*domain.Memo, error) {
					return &domain.Memo{
						ID:        fixedMemoID,
						UserID:    userID,
						Text:      text,
						Status:    domain.MemoStatusPending,
						CreatedAt: fixedTime,
						UpdatedAt: fixedTime,
					}, nil
				}
			},
			expectedStatus:  http.StatusAccepted,
			expectedMemoID:  fixedMemoID.String(),
			expectedUserID:  fixedUserID.String(),
			expectedMemoTxt: "Test memo content",
		},
		{
			name: "missing_user_id",
			setupContext: func(ctx context.Context) context.Context {
				// No user ID in context
				return ctx
			},
			requestBody: CreateMemoRequest{
				Text: "Test memo content",
			},
			setupMock: func(ms *MockMemoService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusUnauthorized,
			expectedErrMsg: "Authentication required",
		},
		{
			name: "invalid_user_id",
			setupContext: func(ctx context.Context) context.Context {
				// Invalid user ID type
				return context.WithValue(ctx, shared.UserIDContextKey, "not-a-uuid")
			},
			requestBody: CreateMemoRequest{
				Text: "Test memo content",
			},
			setupMock: func(ms *MockMemoService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusUnauthorized,
			expectedErrMsg: "Authentication required",
		},
		{
			name: "nil_user_id",
			setupContext: func(ctx context.Context) context.Context {
				// Nil UUID (zero value)
				return context.WithValue(ctx, shared.UserIDContextKey, uuid.Nil)
			},
			requestBody: CreateMemoRequest{
				Text: "Test memo content",
			},
			setupMock: func(ms *MockMemoService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusUnauthorized,
			expectedErrMsg: "Authentication required",
		},
		{
			name: "invalid_request_format",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, fixedUserID)
			},
			requestBody: `{
				"text": "Invalid JSON
			}`,
			setupMock: func(ms *MockMemoService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "Invalid request format",
		},
		{
			name: "missing_required_text",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, fixedUserID)
			},
			requestBody: CreateMemoRequest{
				// Text field intentionally omitted
				Text: "",
			},
			setupMock: func(ms *MockMemoService) {
				// Mock won't be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "required field",
		},
		{
			name: "domain_validation_error",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, fixedUserID)
			},
			requestBody: CreateMemoRequest{
				Text: "Test memo content",
			},
			setupMock: func(ms *MockMemoService) {
				ms.CreateMemoAndEnqueueTaskFn = func(ctx context.Context, userID uuid.UUID, text string) (*domain.Memo, error) {
					return nil, &domain.ValidationError{
						Field:   "text",
						Message: "contains invalid characters",
					}
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedErrMsg: "Invalid text: contains invalid characters",
		},
		{
			name: "service_error",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, shared.UserIDContextKey, fixedUserID)
			},
			requestBody: CreateMemoRequest{
				Text: "Test memo content",
			},
			setupMock: func(ms *MockMemoService) {
				ms.CreateMemoAndEnqueueTaskFn = func(ctx context.Context, userID uuid.UUID, text string) (*domain.Memo, error) {
					return nil, errors.New("unexpected service error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedErrMsg: "An unexpected error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new mock service
			mockService := &MockMemoService{}

			// Configure the mock
			tt.setupMock(mockService)

			// Create a handler with the mock service
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			handler := NewMemoHandler(mockService, logger)

			// Create request body
			var reqBody []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				// Handle raw JSON string for invalid format tests
				reqBody = []byte(str)
			} else {
				// Handle structured request object
				reqBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/memos", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Apply context setup
			req = req.WithContext(tt.setupContext(req.Context()))

			// Create response recorder
			w := httptest.NewRecorder()

			// Call the handler
			handler.CreateMemo(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var respBody map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &respBody)
			require.NoError(t, err)

			// Check error response
			if tt.expectedErrMsg != "" {
				errorMsg, ok := respBody["error"].(string)
				assert.True(t, ok, "Expected error field in response")
				assert.Contains(t, errorMsg, tt.expectedErrMsg)
			}

			// Check success response
			if tt.expectedMemoID != "" {
				assert.Equal(t, tt.expectedMemoID, respBody["id"])
				assert.Equal(t, tt.expectedUserID, respBody["user_id"])
				assert.Equal(t, tt.expectedMemoTxt, respBody["text"])
				assert.Equal(t, string(domain.MemoStatusPending), respBody["status"])
			}
		})
	}
}

// TestMemoHandler_HelperFunctions tests the helper functions in the memo handler.
func TestMemoHandler_HelperFunctions(t *testing.T) {
	t.Run("memoToDTOResponse", func(t *testing.T) {
		// Create a test memo
		userID := uuid.New()
		memoID := uuid.New()
		now := time.Now().UTC()
		memo := &domain.Memo{
			ID:        memoID,
			UserID:    userID,
			Text:      "Test memo content",
			Status:    domain.MemoStatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Convert to response
		response := memoToDTOResponse(memo)

		// Verify correct conversion
		assert.Equal(t, memoID.String(), response.ID)
		assert.Equal(t, userID.String(), response.UserID)
		assert.Equal(t, "Test memo content", response.Text)
		assert.Equal(t, string(domain.MemoStatusPending), response.Status)
		assert.Equal(t, now, response.CreatedAt)
		assert.Equal(t, now, response.UpdatedAt)
	})
}

// TestMemoHandler_NewMemoHandler tests the constructor function.
func TestMemoHandler_NewMemoHandler(t *testing.T) {
	mockService := &MockMemoService{}

	t.Run("with_logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
		handler := NewMemoHandler(mockService, logger)

		assert.NotNil(t, handler)
		assert.Equal(t, mockService, handler.memoService)
		// Validator now uses shared.Validate singleton
		assert.NotNil(t, handler.logger)
	})

	t.Run("without_logger", func(t *testing.T) {
		// Test for panic with nil logger
		assert.Panics(t, func() {
			NewMemoHandler(mockService, nil)
		})

	})
}
