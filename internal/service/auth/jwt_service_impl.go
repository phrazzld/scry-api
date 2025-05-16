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
	signingKey           []byte
	tokenLifetime        time.Duration    // Access token lifetime
	refreshTokenLifetime time.Duration    // Refresh token lifetime
	timeFunc             func() time.Time // Injectable for testing
	clockSkew            time.Duration    // Allowed time difference for validation to handle clock drift
}

// jwtCustomClaims defines the structure of JWT claims we use
type jwtCustomClaims struct {
	UserID    uuid.UUID `json:"uid"`
	TokenType string    `json:"type"`
	jwt.RegisteredClaims
}

// Ensure hmacJWTService implements JWTService interface
var _ JWTService = (*hmacJWTService)(nil)

// NewJWTService creates a new JWT service using HMAC-SHA signing.
func NewJWTService(cfg config.AuthConfig) (JWTService, error) {
	// Convert token lifetimes from minutes to duration
	accessTokenLifetime := time.Duration(cfg.TokenLifetimeMinutes) * time.Minute
	refreshTokenLifetime := time.Duration(cfg.RefreshTokenLifetimeMinutes) * time.Minute

	// Validate that the secret meets minimum length requirements
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("jwt secret must be at least 32 characters")
	}

	return &hmacJWTService{
		signingKey:           []byte(cfg.JWTSecret),
		tokenLifetime:        accessTokenLifetime,
		refreshTokenLifetime: refreshTokenLifetime,
		timeFunc:             time.Now,
		clockSkew:            2 * time.Minute, // Allow 2 minutes of clock skew to handle minor time drifts
	}, nil
}

// GenerateToken creates a signed JWT access token with user claims.
func (s *hmacJWTService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	log := logger.FromContext(ctx)
	now := s.timeFunc()

	// Create the claims with user ID, token type, and standard JWT claims
	claims := jwtCustomClaims{
		UserID:    userID,
		TokenType: "access", // Specify this is an access token
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
		log.Error("failed to sign JWT access token",
			"error", err,
			"user_id", userID,
			"token_type", "access",
			"signing_method", jwt.SigningMethodHS256.Name)
		return "", fmt.Errorf("failed to sign access token with HMAC-SHA256: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT access token and returns the claims if valid.
// It verifies the token has type "access" and returns ErrWrongTokenType if not.
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

	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwtCustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method is what we expect
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.signingKey, nil
		},
		parserOpts...)

	// Handle parsing errors
	if err != nil {
		// Check for specific JWT validation errors
		if errors.Is(err, jwt.ErrTokenExpired) {
			log.Debug("access token validation failed: token expired",
				"error", err,
				"token_type", "access")
			return nil, ErrExpiredToken
		} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
			log.Debug("access token validation failed: token not yet valid",
				"error", err,
				"token_type", "access")
			return nil, ErrTokenNotYetValid
		} else if errors.Is(err, jwt.ErrTokenMalformed) {
			log.Debug("access token validation failed: malformed token",
				"error", err,
				"token_type", "access")
			return nil, ErrInvalidToken
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			log.Debug("access token validation failed: invalid signature",
				"error", err,
				"token_type", "access")
			return nil, ErrInvalidToken
		} else {
			log.Debug("access token validation failed: other validation error",
				"error", err,
				"token_type", "access",
				"error_type", fmt.Sprintf("%T", err))
		}

		return nil, ErrInvalidToken
	}

	// Extract claims from valid token
	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
		// Verify this is an access token
		if claims.TokenType != "access" {
			log.Debug("token validation failed: wrong token type",
				"expected", "access",
				"actual", claims.TokenType)
			return nil, ErrWrongTokenType
		}

		customClaims := &Claims{
			UserID:    claims.UserID,
			TokenType: claims.TokenType,
			Subject:   claims.Subject,
			IssuedAt:  claims.IssuedAt.Time,
			ExpiresAt: claims.ExpiresAt.Time,
			ID:        claims.ID,
		}

		// Log successful token validation
		log.Debug("access token validated successfully",
			"user_id", claims.UserID,
			"token_id", claims.ID,
			"expiry", claims.ExpiresAt.Time)

		return customClaims, nil
	}

	log.Debug("token validation failed: invalid claims")
	return nil, ErrInvalidToken
}

// GenerateRefreshToken creates a signed JWT refresh token with user claims.
// Refresh tokens have longer lifetime than access tokens and are used to obtain new token pairs.
func (s *hmacJWTService) GenerateRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
) (string, error) {
	log := logger.FromContext(ctx)
	now := s.timeFunc()

	// Create the claims with user ID, token type, and standard JWT claims
	claims := jwtCustomClaims{
		UserID:    userID,
		TokenType: "refresh", // Specify this is a refresh token
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenLifetime)),
			ID:        uuid.New().String(), // Unique token ID
		},
	}

	// Create the token with the claims and sign it with HMAC-SHA256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.signingKey)
	if err != nil {
		log.Error("failed to sign JWT refresh token",
			"error", err,
			"user_id", userID,
			"token_type", "refresh",
			"signing_method", jwt.SigningMethodHS256.Name)
		return "", fmt.Errorf("failed to sign refresh token with HMAC-SHA256: %w", err)
	}

	return signedToken, nil
}

// ValidateRefreshToken validates a JWT refresh token and returns the claims if valid.
// It verifies the token has type "refresh" and returns ErrWrongTokenType if not.
// Returns appropriate errors for expiration and invalid signatures.
func (s *hmacJWTService) ValidateRefreshToken(
	ctx context.Context,
	tokenString string,
) (*Claims, error) {
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

	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwtCustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method is what we expect
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.signingKey, nil
		},
		parserOpts...)

	// Handle parsing errors
	if err != nil {
		// Check for specific JWT validation errors
		if errors.Is(err, jwt.ErrTokenExpired) {
			log.Debug("refresh token validation failed: token expired",
				"error", err,
				"token_type", "refresh")
			return nil, ErrExpiredRefreshToken
		} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
			log.Debug("refresh token validation failed: token not yet valid",
				"error", err,
				"token_type", "refresh")
			return nil, ErrInvalidRefreshToken
		} else if errors.Is(err, jwt.ErrTokenMalformed) {
			log.Debug("refresh token validation failed: malformed token",
				"error", err,
				"token_type", "refresh")
			return nil, ErrInvalidRefreshToken
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			log.Debug("refresh token validation failed: invalid signature",
				"error", err,
				"token_type", "refresh")
			return nil, ErrInvalidRefreshToken
		} else {
			log.Debug("refresh token validation failed: other validation error",
				"error", err,
				"token_type", "refresh",
				"error_type", fmt.Sprintf("%T", err))
		}

		return nil, ErrInvalidRefreshToken
	}

	// Extract claims from valid token
	if claims, ok := token.Claims.(*jwtCustomClaims); ok && token.Valid {
		// Verify this is a refresh token
		if claims.TokenType != "refresh" {
			log.Debug("refresh token validation failed: wrong token type",
				"expected", "refresh",
				"actual", claims.TokenType)
			return nil, ErrWrongTokenType
		}

		customClaims := &Claims{
			UserID:    claims.UserID,
			TokenType: claims.TokenType,
			Subject:   claims.Subject,
			IssuedAt:  claims.IssuedAt.Time,
			ExpiresAt: claims.ExpiresAt.Time,
			ID:        claims.ID,
		}

		// Log successful token validation
		log.Debug("refresh token validated successfully",
			"user_id", claims.UserID,
			"token_id", claims.ID,
			"expiry", claims.ExpiresAt.Time)

		return customClaims, nil
	}

	log.Debug("refresh token validation failed: invalid claims")
	return nil, ErrInvalidRefreshToken
}

// GenerateRefreshTokenWithExpiry is a testing helper function that generates a refresh token
// with a custom expiration time. This is used primarily for testing expiration scenarios.
func (s *hmacJWTService) GenerateRefreshTokenWithExpiry(
	ctx context.Context,
	userID uuid.UUID,
	expiryTime time.Time,
) (string, error) {
	log := logger.FromContext(ctx)
	now := s.timeFunc()

	// Create the claims with user ID and custom expiry
	claims := jwtCustomClaims{
		UserID:    userID,
		TokenType: "refresh", // Specify this is a refresh token
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiryTime),
			ID:        uuid.New().String(), // Unique token ID
		},
	}

	// Create the token with the claims and sign it with HMAC-SHA256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.signingKey)
	if err != nil {
		log.Error("failed to sign JWT refresh token with custom expiry",
			"error", err,
			"user_id", userID,
			"token_type", "refresh",
			"custom_expiry", expiryTime)
		return "", fmt.Errorf("failed to sign refresh token with HMAC-SHA256: %w", err)
	}

	return signedToken, nil
}

// For testing utilities, see test_helpers.go
