package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/logger"
)

// hmacJWTService is an implementation of JWTService using HMAC-SHA signing.
type hmacJWTService struct {
	signingKey    []byte
	tokenLifetime time.Duration
	timeFunc      func() time.Time // Injectable for testing
	clockSkew     time.Duration    // Allowed time difference for validation to handle clock drift
}

// jwtCustomClaims defines the structure of JWT claims we use
type jwtCustomClaims struct {
	UserID uuid.UUID `json:"uid"`
	jwt.RegisteredClaims
}

// Ensure hmacJWTService implements JWTService interface
var _ JWTService = (*hmacJWTService)(nil)

// NewJWTService creates a new JWT service using HMAC-SHA signing.
func NewJWTService(cfg config.AuthConfig) (JWTService, error) {
	// Convert token lifetime from minutes to duration
	lifetime := time.Duration(cfg.TokenLifetimeMinutes) * time.Minute

	// Validate that the secret meets minimum length requirements
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("jwt secret must be at least 32 characters")
	}

	return &hmacJWTService{
		signingKey:    []byte(cfg.JWTSecret),
		tokenLifetime: lifetime,
		timeFunc:      time.Now,
		clockSkew:     2 * time.Minute, // Allow 2 minutes of clock skew to handle minor time drifts
	}, nil
}

// GenerateToken creates a signed JWT token with user claims.
func (s *hmacJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	log := logger.FromContext(ctx)
	now := s.timeFunc()

	// Create the claims with user ID and standard JWT claims
	claims := jwtCustomClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenLifetime)),
			ID:        uuid.New().String(), // Unique token ID
		},
	}

	// Create the token with the claims and sign it with HMAC-SHA256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.signingKey)
	if err != nil {
		log.Error("failed to sign JWT token",
			"error", err,
			"userID", userID)
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims if valid.
func (s *hmacJWTService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	log := logger.FromContext(ctx)

	// Parse and validate the token
	now := s.timeFunc()

	// Configure parser options
	parserOpts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithLeeway(s.clockSkew), // Allow for clock skew when validating time claims
		jwt.WithTimeFunc(func() time.Time {
			return now // Use our injected time function for validation
		}),
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwtCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.signingKey, nil
	}, parserOpts...)

	// Handle parsing errors
	if err != nil {
		// Check for specific JWT validation errors
		if errors.Is(err, jwt.ErrTokenExpired) {
			log.Debug("token validation failed: expired", "error", err)
			return nil, ErrExpiredToken
		} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
			log.Debug("token validation failed: not yet valid", "error", err)
			return nil, ErrTokenNotYetValid
		} else {
			log.Debug("token validation failed: other validation error", "error", err)
		}

		log.Debug("token validation failed", "error", err)
		return nil, ErrInvalidToken
	}

	// Extract claims from valid token
	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
		customClaims := &Claims{
			UserID:    claims.UserID,
			Subject:   claims.Subject,
			IssuedAt:  claims.IssuedAt.Time,
			ExpiresAt: claims.ExpiresAt.Time,
			ID:        claims.ID,
		}
		return customClaims, nil
	}

	log.Debug("token validation failed: invalid claims")
	return nil, ErrInvalidToken
}

// NewTestJWTService creates a JWT service with adjustable time for testing
func NewTestJWTService(secret string, lifetime time.Duration, timeFunc func() time.Time) JWTService {
	return &hmacJWTService{
		signingKey:    []byte(secret),
		tokenLifetime: lifetime,
		timeFunc:      timeFunc,
		clockSkew:     0, // No clock skew for tests to make them deterministic
	}
}
