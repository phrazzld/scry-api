package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestJWTService creates a JWT service with the given parameters for testing.
// This is a helper function to avoid duplication in tests.
func createTestJWTService(
	secret string,
	tokenLifetime time.Duration,
	timeFunc func() time.Time,
	refreshLifetime ...time.Duration,
) JWTService {
	// Set default refresh token lifetime if not provided
	refreshTokenLifetime := tokenLifetime * 7 // Default is 7x access token lifetime
	if len(refreshLifetime) > 0 && refreshLifetime[0] > 0 {
		refreshTokenLifetime = refreshLifetime[0]
	}

	return &hmacJWTService{
		signingKey:           []byte(secret),
		tokenLifetime:        tokenLifetime,
		refreshTokenLifetime: refreshTokenLifetime,
		timeFunc:             timeFunc,
		clockSkew:            0, // No clock skew for tests to make them deterministic
	}
}

func TestGenerateToken(t *testing.T) {
	t.Parallel()

	// Setup
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tokenLifetime := 60 * time.Minute
	secret := "test-secret-that-is-long-enough-for-testing"
	userID := uuid.New()

	// Create service with fixed time function for predictable testing
	svc := createTestJWTService(secret, tokenLifetime, func() time.Time {
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
				svc := createTestJWTService(secret, tokenLifetime, func() time.Time {
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
				genSvc := createTestJWTService(secret, tokenLifetime, func() time.Time {
					return fixedTime
				})
				token, _ := genSvc.GenerateToken(context.Background(), userID)

				// Validate token at a later time (after expiry)
				valSvc := createTestJWTService(secret, tokenLifetime, func() time.Time {
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
				genSvc := createTestJWTService(secret, tokenLifetime, func() time.Time {
					return fixedTime
				})
				token, _ := genSvc.GenerateToken(context.Background(), userID)

				// Validate with different secret
				valSvc := createTestJWTService(wrongSecret, tokenLifetime, func() time.Time {
					return fixedTime
				})
				return valSvc, token
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "malformed token",
			setupFunc: func() (JWTService, string) {
				svc := createTestJWTService(secret, tokenLifetime, func() time.Time {
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

func TestValidateRefreshToken(t *testing.T) {
	t.Parallel()

	// Setup
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	accessTokenLifetime := 60 * time.Minute
	refreshTokenLifetime := 7 * 24 * time.Hour // 7 days
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
			name: "valid refresh token",
			setupFunc: func() (JWTService, string) {
				svc := createTestJWTService(
					secret,
					accessTokenLifetime,
					func() time.Time {
						return fixedTime
					},
					refreshTokenLifetime,
				)
				token, _ := svc.GenerateRefreshToken(context.Background(), userID)
				return svc, token
			},
			wantErr: nil,
		},
		{
			name: "expired refresh token",
			setupFunc: func() (JWTService, string) {
				// Create token at fixed time
				genSvc := createTestJWTService(
					secret,
					accessTokenLifetime,
					func() time.Time {
						return fixedTime
					},
					refreshTokenLifetime,
				)
				token, _ := genSvc.GenerateRefreshToken(context.Background(), userID)

				// Validate token at a later time (after expiry)
				valSvc := createTestJWTService(
					secret,
					accessTokenLifetime,
					func() time.Time {
						return fixedTime.Add(refreshTokenLifetime + time.Hour)
					},
					refreshTokenLifetime,
				)
				return valSvc, token
			},
			wantErr: ErrExpiredRefreshToken,
		},
		{
			name: "invalid signature",
			setupFunc: func() (JWTService, string) {
				// Generate with one secret
				genSvc := createTestJWTService(
					secret,
					accessTokenLifetime,
					func() time.Time {
						return fixedTime
					},
					refreshTokenLifetime,
				)
				token, _ := genSvc.GenerateRefreshToken(context.Background(), userID)

				// Validate with different secret
				valSvc := createTestJWTService(
					wrongSecret,
					accessTokenLifetime,
					func() time.Time {
						return fixedTime
					},
					refreshTokenLifetime,
				)
				return valSvc, token
			},
			wantErr: ErrInvalidRefreshToken,
		},
		{
			name: "malformed token",
			setupFunc: func() (JWTService, string) {
				svc := createTestJWTService(
					secret,
					accessTokenLifetime,
					func() time.Time {
						return fixedTime
					},
					refreshTokenLifetime,
				)
				return svc, "this.is.not.a.valid.jwt.token"
			},
			wantErr: ErrInvalidRefreshToken,
		},
		{
			name: "wrong token type (access token)",
			setupFunc: func() (JWTService, string) {
				svc := createTestJWTService(
					secret,
					accessTokenLifetime,
					func() time.Time {
						return fixedTime
					},
					refreshTokenLifetime,
				)
				// Generate an access token, not a refresh token
				token, _ := svc.GenerateToken(context.Background(), userID)
				return svc, token
			},
			wantErr: ErrWrongTokenType,
		},
	}

	// Run tests
	for _, tt := range tests {
		// Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, token := tt.setupFunc()
			claims, err := svc.ValidateRefreshToken(context.Background(), token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, userID, claims.UserID)
				assert.Equal(t, "refresh", claims.TokenType)
			}
		})
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	t.Parallel()

	// Setup
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	accessTokenLifetime := 60 * time.Minute
	refreshTokenLifetime := 7 * 24 * time.Hour // 7 days
	secret := "test-secret-that-is-long-enough-for-testing"
	userID := uuid.New()

	// Create service with fixed time function for predictable testing
	svc := createTestJWTService(
		secret,
		accessTokenLifetime,
		func() time.Time {
			return fixedTime
		},
		refreshTokenLifetime,
	)

	// Test refresh token generation
	t.Run("generates valid refresh token", func(t *testing.T) {
		t.Parallel()
		// Generate refresh token
		refreshToken, err := svc.GenerateRefreshToken(context.Background(), userID)
		require.NoError(t, err)
		require.NotEmpty(t, refreshToken)

		// Validate refresh token
		claims, err := svc.ValidateRefreshToken(context.Background(), refreshToken)
		require.NoError(t, err)

		// Verify claims
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, userID.String(), claims.Subject)
		assert.Equal(t, "refresh", claims.TokenType)
		assert.Equal(t, fixedTime.Unix(), claims.IssuedAt.Unix())
		assert.Equal(t, fixedTime.Add(refreshTokenLifetime).Unix(), claims.ExpiresAt.Unix())
		assert.NotEmpty(t, claims.ID)
	})

	// Test that refresh token is rejected by access token validator
	t.Run("refresh token is rejected by access token validator", func(t *testing.T) {
		t.Parallel()
		// Generate refresh token
		refreshToken, err := svc.GenerateRefreshToken(context.Background(), userID)
		require.NoError(t, err)
		require.NotEmpty(t, refreshToken)

		// Try to validate as access token (should fail)
		claims, err := svc.ValidateToken(context.Background(), refreshToken)
		assert.ErrorIs(t, err, ErrWrongTokenType)
		assert.Nil(t, claims)
	})
}

func TestNewJWTService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      config.AuthConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: config.AuthConfig{
				JWTSecret:                   "test-secret-that-is-at-least-32-characters-long",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			},
			expectError: false,
		},
		{
			name: "valid configuration with minimum secret length",
			config: config.AuthConfig{
				JWTSecret:                   "exactly-32-chars-long-secret!!!!", // 32 chars
				TokenLifetimeMinutes:        30,
				RefreshTokenLifetimeMinutes: 720,
			},
			expectError: false,
		},
		{
			name: "invalid secret too short",
			config: config.AuthConfig{
				JWTSecret:                   "short",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			},
			expectError: true,
			errorMsg:    "jwt secret must be at least 32 characters",
		},
		{
			name: "secret exactly 31 characters (edge case)",
			config: config.AuthConfig{
				JWTSecret:                   "exactly-31-chars-long-secret!",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			},
			expectError: true,
			errorMsg:    "jwt secret must be at least 32 characters",
		},
		{
			name: "empty secret",
			config: config.AuthConfig{
				JWTSecret:                   "",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 1440,
			},
			expectError: true,
			errorMsg:    "jwt secret must be at least 32 characters",
		},
		{
			name: "zero token lifetime",
			config: config.AuthConfig{
				JWTSecret:                   "test-secret-that-is-at-least-32-characters-long",
				TokenLifetimeMinutes:        0,
				RefreshTokenLifetimeMinutes: 1440,
			},
			expectError: false, // Zero lifetime should be valid (results in tokens that expire immediately)
		},
		{
			name: "zero refresh token lifetime",
			config: config.AuthConfig{
				JWTSecret:                   "test-secret-that-is-at-least-32-characters-long",
				TokenLifetimeMinutes:        60,
				RefreshTokenLifetimeMinutes: 0,
			},
			expectError: false, // Zero lifetime should be valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, err := NewJWTService(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, svc)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)

				// Verify the service is functional by generating a token
				userID := uuid.New()
				token, err := svc.GenerateToken(context.Background(), userID)
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestGenerateRefreshTokenWithExpiry(t *testing.T) {
	t.Parallel()

	// Setup
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	secret := "test-secret-that-is-at-least-32-characters-long"
	userID := uuid.New()

	svc := createTestJWTService(secret, time.Hour, func() time.Time {
		return fixedTime
	})

	tests := []struct {
		name        string
		userID      uuid.UUID
		expiryTime  time.Time
		expectError bool
	}{
		{
			name:        "valid future expiry",
			userID:      userID,
			expiryTime:  fixedTime.Add(24 * time.Hour),
			expectError: false,
		},
		{
			name:        "expiry at current time",
			userID:      userID,
			expiryTime:  fixedTime,
			expectError: false,
		},
		{
			name:        "past expiry time",
			userID:      userID,
			expiryTime:  fixedTime.Add(-1 * time.Hour),
			expectError: false, // Token creation should succeed, but validation should fail
		},
		{
			name:        "far future expiry",
			userID:      userID,
			expiryTime:  fixedTime.Add(365 * 24 * time.Hour),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token, err := svc.GenerateRefreshTokenWithExpiry(context.Background(), tt.userID, tt.expiryTime)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				// Validate the token to check expiry was set correctly
				claims, err := svc.ValidateRefreshToken(context.Background(), token)

				if tt.expiryTime.Before(fixedTime) || tt.expiryTime.Equal(fixedTime) {
					// Token should be expired or invalid
					assert.Error(t, err)
					assert.Nil(t, claims)
				} else {
					// Token should be valid
					assert.NoError(t, err)
					assert.NotNil(t, claims)
					assert.Equal(t, tt.userID, claims.UserID)
					assert.Equal(t, "refresh", claims.TokenType)
					assert.Equal(t, tt.expiryTime.Unix(), claims.ExpiresAt.Unix())
				}
			}
		})
	}
}

func TestNewBcryptVerifier(t *testing.T) {
	t.Parallel()

	verifier := NewBcryptVerifier()
	assert.NotNil(t, verifier)

	// Verify it implements the PasswordVerifier interface
	var _ PasswordVerifier = verifier
}

func TestBcryptVerifier_Compare(t *testing.T) {
	t.Parallel()

	verifier := NewBcryptVerifier()

	// Generate a known bcrypt hash for testing
	password := "testpassword123"
	hashedPassword := "$2a$10$o5ov.BzUkOF7UCMpwsSRduu0/MXC0/WRpk1RbIHF4VBJahzKTewwK" // bcrypt hash of "testpassword123"

	tests := []struct {
		name           string
		hashedPassword string
		password       string
		expectError    bool
	}{
		{
			name:           "valid password verification",
			hashedPassword: hashedPassword,
			password:       password,
			expectError:    false,
		},
		{
			name:           "invalid password",
			hashedPassword: hashedPassword,
			password:       "wrongpassword",
			expectError:    true,
		},
		{
			name:           "empty password",
			hashedPassword: hashedPassword,
			password:       "",
			expectError:    true,
		},
		{
			name:           "empty hash",
			hashedPassword: "",
			password:       password,
			expectError:    true,
		},
		{
			name:           "invalid hash format",
			hashedPassword: "not-a-valid-bcrypt-hash",
			password:       password,
			expectError:    true,
		},
		{
			name:           "password with special characters",
			hashedPassword: "$2a$10$kRlVgQoAxtuVqNXqEujPruRxgRViA1jRu.7AbwQulsnsFPoWlWjyy", // bcrypt hash of "password@#$%^&*()"
			password:       "password@#$%^&*()",
			expectError:    false,
		},
		{
			name:           "long password",
			hashedPassword: "$2a$10$uiDUbhyTdC5Mtt.MHB6KruIUNEuvF6vPVO7lKeBsl.LtQUES/m7.S", // bcrypt hash of "this-is-a-long-password-but-not-too-long-for-bcrypt"
			password:       "this-is-a-long-password-but-not-too-long-for-bcrypt",
			expectError:    false,
		},
		{
			name:           "unicode password",
			hashedPassword: "$2a$10$oX60n1ZLZdmt6VDZAGieTOwCfRwcQzRiinS64ebnAh8FnKebpKSgC", // bcrypt hash of "тест123"
			password:       "тест123",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := verifier.Compare(tt.hashedPassword, tt.password)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJWTValidation_EdgeCases(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	secret := "test-secret-that-is-at-least-32-characters-long"
	userID := uuid.New()

	t.Run("token not yet valid", func(t *testing.T) {
		t.Parallel()

		// Create a service that generates tokens for future use
		genSvc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime.Add(time.Hour) // Token issued 1 hour in the future
		})

		token, err := genSvc.GenerateToken(context.Background(), userID)
		require.NoError(t, err)

		// Validate token at current time (before issuance)
		valSvc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		claims, err := valSvc.ValidateToken(context.Background(), token)
		assert.ErrorIs(t, err, ErrTokenNotYetValid)
		assert.Nil(t, claims)
	})

	t.Run("refresh token not yet valid", func(t *testing.T) {
		t.Parallel()

		// Create a service that generates tokens for future use
		genSvc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime.Add(time.Hour) // Token issued 1 hour in the future
		}, 24*time.Hour)

		token, err := genSvc.GenerateRefreshToken(context.Background(), userID)
		require.NoError(t, err)

		// Validate token at current time (before issuance)
		valSvc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		}, 24*time.Hour)

		claims, err := valSvc.ValidateRefreshToken(context.Background(), token)
		assert.ErrorIs(t, err, ErrInvalidRefreshToken)
		assert.Nil(t, claims)
	})

	t.Run("empty token string", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		claims, err := svc.ValidateToken(context.Background(), "")
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)

		claims, err = svc.ValidateRefreshToken(context.Background(), "")
		assert.ErrorIs(t, err, ErrInvalidRefreshToken)
		assert.Nil(t, claims)
	})

	t.Run("token with only one part", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		claims, err := svc.ValidateToken(context.Background(), "single-part-token")
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("token with invalid base64", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Create a token with invalid base64 in the payload
		invalidToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid-base64-payload.signature"

		claims, err := svc.ValidateToken(context.Background(), invalidToken)
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})
}

func TestJWTService_ValidationErrorPaths(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	secret := "test-secret-that-is-at-least-32-characters-long"

	t.Run("token with invalid claims structure", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Create a manually crafted token with invalid claims that would trigger
		// the "other validation error" path in ValidateToken
		// This is challenging to do directly, so we'll test with a token that has
		// an unexpected signing method in the header
		tokenWithRS256 := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiIxMjM0NTY3OC05YWJjLWRlZjAtMTIzNC01Njc4OWFiY2RlZjAiLCJ0eXBlIjoiYWNjZXNzIiwic3ViIjoiMTIzNDU2NzgtOWFiYy1kZWYwLTEyMzQtNTY3ODlhYmNkZWYwIiwiaWF0IjoxNjQxMDMwNDAwLCJuYmYiOjE2NDEwMzA0MDAsImV4cCI6MTY0MTAzNDAwMCwianRpIjoiYWJjZGVmZ2gtaWprbC1tbm9wLXFyc3QtdXZ3eHl6MTIzNCJ9.invalid-signature"

		claims, err := svc.ValidateToken(context.Background(), tokenWithRS256)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("refresh token with invalid claims structure", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		}, 24*time.Hour)

		// Test the same with refresh token validation
		tokenWithRS256 := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiIxMjM0NTY3OC05YWJjLWRlZjAtMTIzNC01Njc4OWFiY2RlZjAiLCJ0eXBlIjoicmVmcmVzaCIsInN1YiI6IjEyMzQ1Njc4LTlhYmMtZGVmMC0xMjM0LTU2Nzg5YWJjZGVmMCIsImlhdCI6MTY0MTAzMDQwMCwibmJmIjoxNjQxMDMwNDAwLCJleHAiOjE2NDEwMzQwMDAsImp0aSI6ImFiY2RlZmdoLWlqa2wtbW5vcC1xcnN0LXV2d3h5ejEyMzQifQ.invalid-signature"

		claims, err := svc.ValidateRefreshToken(context.Background(), tokenWithRS256)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidRefreshToken)
		assert.Nil(t, claims)
	})
}

func TestJWTService_AdditionalEdgeCases(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	secret := "test-secret-that-is-at-least-32-characters-long"
	userID := uuid.New()

	t.Run("context cancellation during token generation", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Token generation should still work despite cancelled context
		// (the JWT library doesn't check context during generation)
		token, err := svc.GenerateToken(ctx, userID)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("context cancellation during token validation", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Generate a valid token first
		token, err := svc.GenerateToken(context.Background(), userID)
		require.NoError(t, err)

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Token validation should still work despite cancelled context
		claims, err := svc.ValidateToken(ctx, token)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
	})

	t.Run("token with corrupted signature length", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Create a token with truncated signature
		validToken, err := svc.GenerateToken(context.Background(), userID)
		require.NoError(t, err)

		// Truncate the signature part
		parts := strings.Split(validToken, ".")
		require.Len(t, parts, 3)
		corruptedToken := parts[0] + "." + parts[1] + "." + parts[2][:10] // Truncate signature

		claims, err := svc.ValidateToken(context.Background(), corruptedToken)
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("test logging during successful operations", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Test that successful operations complete (this tests the logging paths)
		token, err := svc.GenerateToken(context.Background(), userID)
		require.NoError(t, err)

		refreshToken, err := svc.GenerateRefreshToken(context.Background(), userID)
		require.NoError(t, err)

		customToken, err := svc.GenerateRefreshTokenWithExpiry(
			context.Background(),
			userID,
			fixedTime.Add(24*time.Hour),
		)
		require.NoError(t, err)

		// Validate all tokens (this tests logging in validation paths)
		claims, err := svc.ValidateToken(context.Background(), token)
		assert.NoError(t, err)
		assert.NotNil(t, claims)

		refreshClaims, err := svc.ValidateRefreshToken(context.Background(), refreshToken)
		assert.NoError(t, err)
		assert.NotNil(t, refreshClaims)

		customClaims, err := svc.ValidateRefreshToken(context.Background(), customToken)
		assert.NoError(t, err)
		assert.NotNil(t, customClaims)
	})
}

func TestJWTService_CompleteCodeCoverage(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	secret := "test-secret-that-is-at-least-32-characters-long"
	userID := uuid.New()

	// Test all functions extensively to try to hit remaining uncovered lines
	t.Run("exhaustive function coverage", func(t *testing.T) {
		t.Parallel()

		// Test with different clock skew settings to potentially hit different paths
		svc := &hmacJWTService{
			signingKey:           []byte(secret),
			tokenLifetime:        time.Hour,
			refreshTokenLifetime: 24 * time.Hour,
			timeFunc:             func() time.Time { return fixedTime },
			clockSkew:            5 * time.Minute, // Different clock skew
		}

		// Generate tokens with this service
		token, err := svc.GenerateToken(context.Background(), userID)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		refreshToken, err := svc.GenerateRefreshToken(context.Background(), userID)
		require.NoError(t, err)
		require.NotEmpty(t, refreshToken)

		// Test with far future expiry
		farFutureToken, err := svc.GenerateRefreshTokenWithExpiry(
			context.Background(),
			userID,
			fixedTime.Add(365*24*time.Hour),
		)
		require.NoError(t, err)
		require.NotEmpty(t, farFutureToken)

		// Validate all tokens to exercise validation paths
		claims, err := svc.ValidateToken(context.Background(), token)
		require.NoError(t, err)
		require.NotNil(t, claims)
		require.Equal(t, userID, claims.UserID)

		refreshClaims, err := svc.ValidateRefreshToken(context.Background(), refreshToken)
		require.NoError(t, err)
		require.NotNil(t, refreshClaims)
		require.Equal(t, userID, refreshClaims.UserID)

		futureClaims, err := svc.ValidateRefreshToken(context.Background(), farFutureToken)
		require.NoError(t, err)
		require.NotNil(t, futureClaims)
		require.Equal(t, userID, futureClaims.UserID)
	})

	t.Run("test with zero clock skew", func(t *testing.T) {
		t.Parallel()

		// Test with zero clock skew to potentially hit different validation paths
		svc := &hmacJWTService{
			signingKey:           []byte(secret),
			tokenLifetime:        30 * time.Minute,
			refreshTokenLifetime: 12 * time.Hour,
			timeFunc:             func() time.Time { return fixedTime },
			clockSkew:            0, // Zero clock skew
		}

		// Generate and validate tokens
		token, err := svc.GenerateToken(context.Background(), userID)
		require.NoError(t, err)

		claims, err := svc.ValidateToken(context.Background(), token)
		require.NoError(t, err)
		require.Equal(t, userID, claims.UserID)

		refreshToken, err := svc.GenerateRefreshToken(context.Background(), userID)
		require.NoError(t, err)

		refreshClaims, err := svc.ValidateRefreshToken(context.Background(), refreshToken)
		require.NoError(t, err)
		require.Equal(t, userID, refreshClaims.UserID)
	})
}

func TestJWTService_CoverageGaps(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	secret := "test-secret-that-is-at-least-32-characters-long"

	t.Run("invalid signing method in access token", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Create a token with RSA signing method instead of HMAC
		// This token has the correct structure but wrong signing method
		rsaToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiIxMjM0NTY3OC05YWJjLWRlZjAtMTIzNC01Njc4OWFiY2RlZjAiLCJ0eXBlIjoiYWNjZXNzIiwic3ViIjoiMTIzNDU2NzgtOWFiYy1kZWYwLTEyMzQtNTY3ODlhYmNkZWYwIiwiaWF0IjoxNjQxMDMwNDAwLCJuYmYiOjE2NDEwMzA0MDAsImV4cCI6OTk5OTk5OTk5OSwianRpIjoiYWJjZGVmZ2gtaWprbC1tbm9wLXFyc3QtdXZ3eHl6MTIzNCJ9.signature"

		claims, err := svc.ValidateToken(context.Background(), rsaToken)
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("invalid signing method in refresh token", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		}, 24*time.Hour)

		// Create a refresh token with RSA signing method instead of HMAC
		rsaRefreshToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiIxMjM0NTY3OC05YWJjLWRlZjAtMTIzNC01Njc4OWFiY2RlZjAiLCJ0eXBlIjoicmVmcmVzaCIsInN1YiI6IjEyMzQ1Njc4LTlhYmMtZGVmMC0xMjM0LTU2Nzg5YWJjZGVmMCIsImlhdCI6MTY0MTAzMDQwMCwibmJmIjoxNjQxMDMwNDAwLCJleHAiOjk5OTk5OTk5OTksImp0aSI6ImFiY2RlZmdoLWlqa2wtbW5vcC1xcnN0LXV2d3h5ejEyMzQifQ.signature"

		claims, err := svc.ValidateRefreshToken(context.Background(), rsaRefreshToken)
		assert.ErrorIs(t, err, ErrInvalidRefreshToken)
		assert.Nil(t, claims)
	})

	t.Run("token with invalid JSON in claims", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Create a token with malformed JSON claims (missing closing brace)
		// This should trigger the "other validation error" path
		invalidClaimsToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiIxMjM0NTY3OC05YWJjLWRlZjAtMTIzNC01Njc4OWFiY2RlZjAiLCJ0eXBlIjoiYWNjZXNzIiwic3ViIjoiMTIzNDU2NzgtOWFiYy1kZWYwLTEyMzQtNTY3ODlhYmNkZWYwIiwiaWF0IjoxNjQxMDMwNDAwLCJuYmYiOjE2NDEwMzA0MDAsImV4cCI6OTk5OTk5OTk5OSwianRpIjoiYWJjZGVmZ2gtaWprbC1tbm9wLXFyc3QtdXZ3eHl6MTIzNA.WrongSignatureToTriggerValidationError"

		claims, err := svc.ValidateToken(context.Background(), invalidClaimsToken)
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("refresh token with invalid JSON in claims", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		}, 24*time.Hour)

		// Create a refresh token with malformed JSON claims
		invalidRefreshClaimsToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiIxMjM0NTY3OC05YWJjLWRlZjAtMTIzNC01Njc4OWFiY2RlZjAiLCJ0eXBlIjoicmVmcmVzaCIsInN1YiI6IjEyMzQ1Njc4LTlhYmMtZGVmMC0xMjM0LTU2Nzg5YWJjZGVmMCIsImlhdCI6MTY0MTAzMDQwMCwibmJmIjoxNjQxMDMwNDAwLCJleHAiOjk5OTk5OTk5OTksImp0aSI6ImFiY2RlZmdoLWlqa2wtbW5vcC1xcnN0LXV2d3h5ejEyMzQK.WrongSignatureToTriggerValidationError"

		claims, err := svc.ValidateRefreshToken(context.Background(), invalidRefreshClaimsToken)
		assert.ErrorIs(t, err, ErrInvalidRefreshToken)
		assert.Nil(t, claims)
	})

	t.Run("token with invalid base64 encoding", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Token with invalid base64 that should trigger parsing error
		invalidBase64Token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid_base64_that_cannot_be_decoded!!!.signature"

		claims, err := svc.ValidateToken(context.Background(), invalidBase64Token)
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, claims)
	})

	t.Run("token with invalid claims structure", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		})

		// Create a token with empty claims object that would fail type assertion
		emptyClaimsToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.secret"

		claims, err := svc.ValidateToken(context.Background(), emptyClaimsToken)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("refresh token with invalid claims structure", func(t *testing.T) {
		t.Parallel()

		svc := createTestJWTService(secret, time.Hour, func() time.Time {
			return fixedTime
		}, 24*time.Hour)

		// Create a refresh token with empty claims object
		emptyClaimsRefreshToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.secret"

		claims, err := svc.ValidateRefreshToken(context.Background(), emptyClaimsRefreshToken)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}
