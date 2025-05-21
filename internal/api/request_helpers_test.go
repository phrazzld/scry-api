package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/phrazzld/scry-api/internal/api/shared"
)

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func() context.Context
		expectedUserID uuid.UUID
		expectedOK     bool
	}{
		{
			name: "valid user ID in context",
			setupContext: func() context.Context {
				userID := uuid.New()
				return context.WithValue(context.Background(), shared.UserIDContextKey, userID)
			},
			expectedOK: true,
		},
		{
			name: "missing user ID in context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedUserID: uuid.Nil,
			expectedOK:     false,
		},
		{
			name: "nil user ID in context",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), shared.UserIDContextKey, uuid.Nil)
			},
			expectedUserID: uuid.Nil,
			expectedOK:     false,
		},
		{
			name: "wrong type in context",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), shared.UserIDContextKey, "not-a-uuid")
			},
			expectedUserID: uuid.Nil,
			expectedOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(ctx)

			userID, ok := getUserIDFromContext(req)

			assert.Equal(t, tt.expectedOK, ok)
			if tt.expectedOK {
				assert.NotEqual(t, uuid.Nil, userID)
			} else {
				assert.Equal(t, tt.expectedUserID, userID)
			}
		})
	}
}

func TestGetPathUUID(t *testing.T) {
	validUUID := uuid.New()

	tests := []struct {
		name        string
		setupRouter func() *chi.Mux
		path        string
		paramName   string
		expectError bool
		expectedID  uuid.UUID
	}{
		{
			name: "valid UUID parameter",
			setupRouter: func() *chi.Mux {
				r := chi.NewRouter()
				r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
					// Handler implementation not needed for this test
				})
				return r
			},
			path:        "/test/" + validUUID.String(),
			paramName:   "id",
			expectError: false,
			expectedID:  validUUID,
		},
		{
			name: "missing parameter",
			setupRouter: func() *chi.Mux {
				r := chi.NewRouter()
				r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
					// Handler implementation not needed for this test
				})
				return r
			},
			path:        "/test",
			paramName:   "id",
			expectError: true,
			expectedID:  uuid.Nil,
		},
		{
			name: "invalid UUID format",
			setupRouter: func() *chi.Mux {
				r := chi.NewRouter()
				r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
					// Handler implementation not needed for this test
				})
				return r
			},
			path:        "/test/invalid-uuid",
			paramName:   "id",
			expectError: true,
			expectedID:  uuid.Nil,
		},
		{
			name: "empty UUID parameter",
			setupRouter: func() *chi.Mux {
				r := chi.NewRouter()
				r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
					// Handler implementation not needed for this test
				})
				return r
			},
			path:        "/test/",
			paramName:   "id",
			expectError: true,
			expectedID:  uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := tt.setupRouter()

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			// Create a temporary handler to capture the request with chi context
			var capturedReq *http.Request
			router.Get("/test/*", func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
			})
			router.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
			})

			// Route the request to set up chi context
			router.ServeHTTP(rr, req)

			if capturedReq == nil {
				// If no route matched, create a manual context for the missing parameter test
				capturedReq = req
			}

			id, err := getPathUUID(capturedReq, tt.paramName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedID, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestHandleUserIDFromContext(t *testing.T) {
	validUserID := uuid.New()

	tests := []struct {
		name           string
		setupContext   func() context.Context
		expectedStatus int
		expectedOK     bool
		expectedUserID uuid.UUID
	}{
		{
			name: "valid user ID",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), shared.UserIDContextKey, validUserID)
			},
			expectedOK:     true,
			expectedUserID: validUserID,
		},
		{
			name: "missing user ID",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedOK:     false,
			expectedUserID: uuid.Nil,
		},
		{
			name: "nil user ID",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), shared.UserIDContextKey, uuid.Nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedOK:     false,
			expectedUserID: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(tt.setupContext())
			rr := httptest.NewRecorder()

			userID, ok := handleUserIDFromContext(rr, req, nil)

			assert.Equal(t, tt.expectedOK, ok)
			if tt.expectedOK {
				assert.Equal(t, tt.expectedUserID, userID)
			} else {
				assert.Equal(t, tt.expectedUserID, userID)
				assert.Equal(t, tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestHandleUserIDAndPathUUID(t *testing.T) {
	validUserID := uuid.New()
	validPathUUID := uuid.New()

	tests := []struct {
		name           string
		setupContext   func() context.Context
		setupRouter    func() *chi.Mux
		path           string
		paramName      string
		expectedStatus int
		expectedOK     bool
		expectedUserID uuid.UUID
		expectedPathID uuid.UUID
	}{
		{
			name: "valid user ID and path UUID",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), shared.UserIDContextKey, validUserID)
			},
			setupRouter: func() *chi.Mux {
				r := chi.NewRouter()
				r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {})
				return r
			},
			path:           "/test/" + validPathUUID.String(),
			paramName:      "id",
			expectedOK:     true,
			expectedUserID: validUserID,
			expectedPathID: validPathUUID,
		},
		{
			name: "missing user ID",
			setupContext: func() context.Context {
				return context.Background()
			},
			setupRouter: func() *chi.Mux {
				r := chi.NewRouter()
				r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {})
				return r
			},
			path:           "/test/" + validPathUUID.String(),
			paramName:      "id",
			expectedStatus: http.StatusUnauthorized,
			expectedOK:     false,
			expectedUserID: uuid.Nil,
			expectedPathID: uuid.Nil,
		},
		{
			name: "valid user ID but invalid path UUID",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), shared.UserIDContextKey, validUserID)
			},
			setupRouter: func() *chi.Mux {
				r := chi.NewRouter()
				r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {})
				return r
			},
			path:           "/test/invalid-uuid",
			paramName:      "id",
			expectedStatus: http.StatusBadRequest,
			expectedOK:     false,
			expectedUserID: uuid.Nil,
			expectedPathID: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := tt.setupRouter()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req = req.WithContext(tt.setupContext())
			rr := httptest.NewRecorder()

			// Route the request to set up chi context
			var capturedReq *http.Request
			router.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
				capturedReq = r
			})
			router.ServeHTTP(rr, req)

			if capturedReq != nil {
				req = capturedReq
			}

			userID, pathID, ok := handleUserIDAndPathUUID(rr, req, tt.paramName, nil)

			assert.Equal(t, tt.expectedOK, ok)
			if tt.expectedOK {
				assert.Equal(t, tt.expectedUserID, userID)
				assert.Equal(t, tt.expectedPathID, pathID)
			} else {
				assert.Equal(t, tt.expectedUserID, userID)
				assert.Equal(t, tt.expectedPathID, pathID)
				assert.Equal(t, tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestParseAndValidateRequest(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedOK     bool
		expectedStatus int
	}{
		{
			name: "valid request",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "ValidPassword123",
			},
			expectedOK: true,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedOK:     false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - invalid email",
			requestBody: RegisterRequest{
				Email:    "invalid-email",
				Password: "ValidPassword123",
			},
			expectedOK:     false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - short password",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "short",
			},
			expectedOK:     false,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - missing email",
			requestBody: RegisterRequest{
				Password: "ValidPassword123",
			},
			expectedOK:     false,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if str, ok := tt.requestBody.(string); ok {
				body.WriteString(str)
			} else {
				_ = json.NewEncoder(&body).Encode(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/", &body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			var parsedReq RegisterRequest
			ok := parseAndValidateRequest(rr, req, &parsedReq, nil)

			assert.Equal(t, tt.expectedOK, ok)
			if !tt.expectedOK {
				assert.Equal(t, tt.expectedStatus, rr.Code)
			}
		})
	}
}
