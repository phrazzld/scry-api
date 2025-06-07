package api

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  RegisterRequest
		valid    bool
		jsonData string
	}{
		{
			name: "valid register request",
			request: RegisterRequest{
				Email:    "test@example.com",
				Password: "ValidPassword123",
			},
			valid:    true,
			jsonData: `{"email":"test@example.com","password":"ValidPassword123"}`,
		},
		{
			name: "valid register request with complex password",
			request: RegisterRequest{
				Email:    "user@domain.co.uk",
				Password: "Complex!Password@123#$%",
			},
			valid:    true,
			jsonData: `{"email":"user@domain.co.uk","password":"Complex!Password@123#$%"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonBytes, err := json.Marshal(tt.request)
			require.NoError(t, err)
			assert.JSONEq(t, tt.jsonData, string(jsonBytes))

			// Test JSON unmarshaling
			var parsed RegisterRequest
			err = json.Unmarshal([]byte(tt.jsonData), &parsed)
			require.NoError(t, err)
			assert.Equal(t, tt.request.Email, parsed.Email)
			assert.Equal(t, tt.request.Password, parsed.Password)
		})
	}
}

func TestLoginRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  LoginRequest
		jsonData string
	}{
		{
			name: "valid login request",
			request: LoginRequest{
				Email:    "test@example.com",
				Password: "password",
			},
			jsonData: `{"email":"test@example.com","password":"password"}`,
		},
		{
			name: "login request with special characters",
			request: LoginRequest{
				Email:    "user+tag@example.com",
				Password: "password!@#$%^&*()",
			},
			jsonData: `{"email":"user+tag@example.com","password":"password!@#$%^&*()"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonBytes, err := json.Marshal(tt.request)
			require.NoError(t, err)
			assert.JSONEq(t, tt.jsonData, string(jsonBytes))

			// Test JSON unmarshaling
			var parsed LoginRequest
			err = json.Unmarshal([]byte(tt.jsonData), &parsed)
			require.NoError(t, err)
			assert.Equal(t, tt.request.Email, parsed.Email)
			assert.Equal(t, tt.request.Password, parsed.Password)
		})
	}
}

func TestAuthResponse(t *testing.T) {
	userID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name     string
		response AuthResponse
		jsonData string
	}{
		{
			name: "complete auth response",
			response: AuthResponse{
				UserID:       userID,
				AccessToken:  "access-token-value",
				RefreshToken: "refresh-token-value",
				ExpiresAt:    "2024-01-15T13:00:00Z",
			},
			jsonData: `{
				"user_id":"123e4567-e89b-12d3-a456-426614174000",
				"token":"access-token-value",
				"refresh_token":"refresh-token-value",
				"expires_at":"2024-01-15T13:00:00Z"
			}`,
		},
		{
			name: "auth response without optional fields",
			response: AuthResponse{
				UserID:      userID,
				AccessToken: "access-token-value",
			},
			jsonData: `{
				"user_id":"123e4567-e89b-12d3-a456-426614174000",
				"token":"access-token-value"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonBytes, err := json.Marshal(tt.response)
			require.NoError(t, err)
			assert.JSONEq(t, tt.jsonData, string(jsonBytes))

			// Test JSON unmarshaling
			var parsed AuthResponse
			err = json.Unmarshal([]byte(tt.jsonData), &parsed)
			require.NoError(t, err)
			assert.Equal(t, tt.response.UserID, parsed.UserID)
			assert.Equal(t, tt.response.AccessToken, parsed.AccessToken)
			assert.Equal(t, tt.response.RefreshToken, parsed.RefreshToken)
			assert.Equal(t, tt.response.ExpiresAt, parsed.ExpiresAt)
		})
	}
}

func TestRefreshTokenRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  RefreshTokenRequest
		jsonData string
	}{
		{
			name: "valid refresh token request",
			request: RefreshTokenRequest{
				RefreshToken: "refresh-token-value",
			},
			jsonData: `{"refresh_token":"refresh-token-value"}`,
		},
		{
			name: "refresh token request with long token",
			request: RefreshTokenRequest{
				RefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			},
			jsonData: `{"refresh_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonBytes, err := json.Marshal(tt.request)
			require.NoError(t, err)
			assert.JSONEq(t, tt.jsonData, string(jsonBytes))

			// Test JSON unmarshaling
			var parsed RefreshTokenRequest
			err = json.Unmarshal([]byte(tt.jsonData), &parsed)
			require.NoError(t, err)
			assert.Equal(t, tt.request.RefreshToken, parsed.RefreshToken)
		})
	}
}

func TestRefreshTokenResponse(t *testing.T) {
	tests := []struct {
		name     string
		response RefreshTokenResponse
		jsonData string
	}{
		{
			name: "complete refresh token response",
			response: RefreshTokenResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresAt:    "2024-01-15T14:00:00Z",
			},
			jsonData: `{
				"access_token":"new-access-token",
				"refresh_token":"new-refresh-token",
				"expires_at":"2024-01-15T14:00:00Z"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonBytes, err := json.Marshal(tt.response)
			require.NoError(t, err)
			assert.JSONEq(t, tt.jsonData, string(jsonBytes))

			// Test JSON unmarshaling
			var parsed RefreshTokenResponse
			err = json.Unmarshal([]byte(tt.jsonData), &parsed)
			require.NoError(t, err)
			assert.Equal(t, tt.response.AccessToken, parsed.AccessToken)
			assert.Equal(t, tt.response.RefreshToken, parsed.RefreshToken)
			assert.Equal(t, tt.response.ExpiresAt, parsed.ExpiresAt)
		})
	}
}

func TestJSONFieldMapping(t *testing.T) {
	// Test that AccessToken maps to "token" in JSON for backward compatibility
	resp := AuthResponse{
		UserID:      uuid.New(),
		AccessToken: "test-token",
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	// Verify the JSON contains "token" not "access_token"
	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"token":"test-token"`)
	assert.NotContains(t, jsonStr, `"access_token"`)

	// Test that RefreshToken shows up as "refresh_token" when provided
	resp.RefreshToken = "test-refresh"
	jsonBytes, err = json.Marshal(resp)
	require.NoError(t, err)

	jsonStr = string(jsonBytes)
	assert.Contains(t, jsonStr, `"refresh_token":"test-refresh"`)
}

func TestOmitEmptyFields(t *testing.T) {
	// Test that empty optional fields are omitted from JSON
	resp := AuthResponse{
		UserID:      uuid.New(),
		AccessToken: "test-token",
		// RefreshToken and ExpiresAt are empty and should be omitted
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.NotContains(t, jsonStr, "refresh_token")
	assert.NotContains(t, jsonStr, "expires_at")
}
