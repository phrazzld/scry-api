package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	t.Parallel()

	// Setup
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tokenLifetime := 60 * time.Minute
	secret := "test-secret-that-is-long-enough-for-testing"
	userID := uuid.New()

	// Create service with fixed time function for predictable testing
	svc := NewTestJWTService(secret, tokenLifetime, func() time.Time {
		return fixedTime
	})

	// Test token generation
	t.Run("generates valid token", func(t *testing.T) {
		t.Parallel()
		// Generate token
		token, err := svc.GenerateToken(context.Background(), userID)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		// Validate token
		claims, err := svc.ValidateToken(context.Background(), token)
		require.NoError(t, err)

		// Verify claims
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, userID.String(), claims.Subject)
		// Compare Unix timestamps to avoid timezone issues
		assert.Equal(t, fixedTime.Unix(), claims.IssuedAt.Unix())
		assert.Equal(t, fixedTime.Add(tokenLifetime).Unix(), claims.ExpiresAt.Unix())
		assert.NotEmpty(t, claims.ID)
	})
}

func TestValidateToken(t *testing.T) {
	t.Parallel()

	// Setup
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tokenLifetime := 60 * time.Minute
	secret := "test-secret-that-is-long-enough-for-testing"
	wrongSecret := "wrong-secret-that-is-long-enough-for-testing"
	userID := uuid.New()

	// Test cases
	tests := []struct {
		name      string
		setupFunc func() (JWTService, string)
		wantErr   error
	}{
		{
			name: "valid token",
			setupFunc: func() (JWTService, string) {
				svc := NewTestJWTService(secret, tokenLifetime, func() time.Time {
					return fixedTime
				})
				token, _ := svc.GenerateToken(context.Background(), userID)
				return svc, token
			},
			wantErr: nil,
		},
		{
			name: "expired token",
			setupFunc: func() (JWTService, string) {
				// Create token at fixed time
				genSvc := NewTestJWTService(secret, tokenLifetime, func() time.Time {
					return fixedTime
				})
				token, _ := genSvc.GenerateToken(context.Background(), userID)

				// Validate token at a later time (after expiry)
				valSvc := NewTestJWTService(secret, tokenLifetime, func() time.Time {
					return fixedTime.Add(tokenLifetime + time.Hour)
				})
				return valSvc, token
			},
			wantErr: ErrExpiredToken,
		},
		{
			name: "invalid signature",
			setupFunc: func() (JWTService, string) {
				// Generate with one secret
				genSvc := NewTestJWTService(secret, tokenLifetime, func() time.Time {
					return fixedTime
				})
				token, _ := genSvc.GenerateToken(context.Background(), userID)

				// Validate with different secret
				valSvc := NewTestJWTService(wrongSecret, tokenLifetime, func() time.Time {
					return fixedTime
				})
				return valSvc, token
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "malformed token",
			setupFunc: func() (JWTService, string) {
				svc := NewTestJWTService(secret, tokenLifetime, func() time.Time {
					return fixedTime
				})
				return svc, "this.is.not.a.valid.jwt.token"
			},
			wantErr: ErrInvalidToken,
		},
	}

	// Run tests
	for _, tt := range tests {
		// Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, token := tt.setupFunc()
			claims, err := svc.ValidateToken(context.Background(), token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, userID, claims.UserID)
			}
		})
	}
}
