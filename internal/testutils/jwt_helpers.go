package testutils

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/service/auth"
)

// TestJWTConstants provides standard values for JWT testing
const (
	// TestJWTSecret is a dedicated test-only secret for signing JWTs
	// This must never be used in production
	TestJWTSecret = "test-jwt-secret-that-is-32-chars-long"

	// TestTokenLifetime is the default lifetime for test access tokens
	TestTokenLifetime = 15 * time.Minute

	// TestRefreshTokenLifetime is the default lifetime for test refresh tokens
	TestRefreshTokenLifetime = 24 * time.Hour
)

// TestJWTService represents a JWT service implementation for tests
// It implements the auth.JWTService interface with real JWT signing/validation
type TestJWTService struct {
	secret               string
	tokenLifetime        time.Duration
	refreshTokenLifetime time.Duration
	timeFunc             func() time.Time
}

// jwtCustomClaims defines the structure of JWT claims we use in tests
// This matches the structure in the real JWT service implementation
type jwtCustomClaims struct {
	UserID    uuid.UUID `json:"uid"`
	TokenType string    `json:"type"`
	jwt.RegisteredClaims
}

// NewTestJWTService creates a JWT service for testing with the test secret
func NewTestJWTService() auth.JWTService {
	return &TestJWTService{
		secret:               TestJWTSecret,
		tokenLifetime:        TestTokenLifetime,
		refreshTokenLifetime: TestRefreshTokenLifetime,
		timeFunc:             time.Now,
	}
}

// NewTestJWTServiceWithOptions creates a JWT service for testing with custom options
func NewTestJWTServiceWithOptions(
	secret string,
	tokenLifetime time.Duration,
	refreshTokenLifetime time.Duration,
	timeFunc func() time.Time,
) auth.JWTService {
	if secret == "" {
		secret = TestJWTSecret
	}
	if tokenLifetime <= 0 {
		tokenLifetime = TestTokenLifetime
	}
	if refreshTokenLifetime <= 0 {
		refreshTokenLifetime = TestRefreshTokenLifetime
	}
	if timeFunc == nil {
		timeFunc = time.Now
	}

	return &TestJWTService{
		secret:               secret,
		tokenLifetime:        tokenLifetime,
		refreshTokenLifetime: refreshTokenLifetime,
		timeFunc:             timeFunc,
	}
}

// GenerateToken creates a signed JWT access token for the given user ID
func (s *TestJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	now := s.timeFunc()

	// Create the claims with user ID and standard JWT claims
	claims := jwtCustomClaims{
		UserID:    userID,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenLifetime)),
			ID:        uuid.New().String(), // Unique token ID
		},
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign test access token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT access token and returns the claims if valid
func (s *TestJWTService) ValidateToken(ctx context.Context, tokenString string) (*auth.Claims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &jwtCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})

	// Handle parsing errors
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, auth.ErrExpiredToken
		}
		return nil, auth.ErrInvalidToken
	}

	// Extract and verify claims
	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
		// Verify this is an access token
		if claims.TokenType != "access" {
			return nil, auth.ErrWrongTokenType
		}

		// Convert to service claims format
		return &auth.Claims{
			UserID:    claims.UserID,
			TokenType: claims.TokenType,
			Subject:   claims.Subject,
			IssuedAt:  claims.IssuedAt.Time,
			ExpiresAt: claims.ExpiresAt.Time,
			ID:        claims.ID,
		}, nil
	}

	return nil, auth.ErrInvalidToken
}

// GenerateRefreshToken creates a signed JWT refresh token for the given user ID
func (s *TestJWTService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	now := s.timeFunc()

	// Create the claims with user ID and standard JWT claims
	claims := jwtCustomClaims{
		UserID:    userID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenLifetime)),
			ID:        uuid.New().String(), // Unique token ID
		},
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign test refresh token: %w", err)
	}

	return signedToken, nil
}

// ValidateRefreshToken validates a JWT refresh token and returns the claims if valid
func (s *TestJWTService) ValidateRefreshToken(ctx context.Context, tokenString string) (*auth.Claims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &jwtCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})

	// Handle parsing errors
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, auth.ErrExpiredRefreshToken
		}
		return nil, auth.ErrInvalidRefreshToken
	}

	// Extract and verify claims
	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
		// Verify this is a refresh token
		if claims.TokenType != "refresh" {
			return nil, auth.ErrWrongTokenType
		}

		// Convert to service claims format
		return &auth.Claims{
			UserID:    claims.UserID,
			TokenType: claims.TokenType,
			Subject:   claims.Subject,
			IssuedAt:  claims.IssuedAt.Time,
			ExpiresAt: claims.ExpiresAt.Time,
			ID:        claims.ID,
		}, nil
	}

	return nil, auth.ErrInvalidRefreshToken
}

// GenerateTokenWithClaims creates a test access token with custom claims
func GenerateTokenWithClaims(userID uuid.UUID, customClaims map[string]interface{}) (string, error) {
	// Create a JWT service with default settings
	service := NewTestJWTService()

	// Generate a token for the user
	return service.GenerateToken(context.Background(), userID)
}

// GenerateAuthHeader creates an Authorization header value with a valid JWT token
func GenerateAuthHeader(userID uuid.UUID) (string, error) {
	service := NewTestJWTService()
	token, err := service.GenerateToken(context.Background(), userID)
	if err != nil {
		return "", err
	}
	return "Bearer " + token, nil
}

// CreateFixedTimeJWTService creates a JWT service with a fixed time function
// This is useful for deterministic testing where you need to control the exact issued/expiry times
func CreateFixedTimeJWTService(fixedTime time.Time) auth.JWTService {
	return NewTestJWTServiceWithOptions(
		TestJWTSecret,
		TestTokenLifetime,
		TestRefreshTokenLifetime,
		func() time.Time { return fixedTime },
	)
}
