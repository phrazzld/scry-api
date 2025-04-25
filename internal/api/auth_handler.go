package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/phrazzld/scry-api/internal/api/shared"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/redact"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
)

// AuthHandler handles authentication-related API requests.
type AuthHandler struct {
	userStore        store.UserStore
	jwtService       auth.JWTService
	passwordVerifier auth.PasswordVerifier
	authConfig       *config.AuthConfig // For accessing token lifetime and other auth settings
	timeFunc         func() time.Time   // Injectable time source for testing
	logger           *slog.Logger       // Added logger field
}

// generateTokenResponse generates access and refresh tokens for a user, along with expiration time.
// Returns the tokens and formatted expiration time, or an error if token generation fails.
func (h *AuthHandler) generateTokenResponse(
	ctx context.Context,
	userID uuid.UUID,
) (accessToken, refreshToken, expiresAt string, err error) {
	// Get logger from context or use default
	log := h.logger.With(slog.String("user_id", userID.String()))

	// Generate access token
	accessToken, err = h.jwtService.GenerateToken(ctx, userID)
	if err != nil {
		log.Error("failed to generate access token",
			slog.String("error", redact.Error(err)),
			slog.String("token_type", "access"),
			slog.Int("lifetime_minutes", h.authConfig.TokenLifetimeMinutes))
		return "", "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err = h.jwtService.GenerateRefreshToken(ctx, userID)
	if err != nil {
		log.Error("failed to generate refresh token",
			slog.String("error", redact.Error(err)),
			slog.String("token_type", "refresh"),
			slog.Int("lifetime_minutes", h.authConfig.RefreshTokenLifetimeMinutes))
		return "", "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Calculate access token expiration time using the injected time source
	expiresAtTime := h.timeFunc().
		Add(time.Duration(h.authConfig.TokenLifetimeMinutes) * time.Minute)

	// Format expiration time in RFC3339 format (standard for JSON API responses)
	expiresAt = expiresAtTime.Format(time.RFC3339)

	// Log successful token generation with appropriate level
	log.Debug("successfully generated token pair",
		slog.String("access_token_expires_at", expiresAt),
		slog.Int("refresh_token_lifetime_minutes", h.authConfig.RefreshTokenLifetimeMinutes))

	return accessToken, refreshToken, expiresAt, nil
}

// NewAuthHandler creates a new AuthHandler with the given dependencies.
func NewAuthHandler(
	userStore store.UserStore,
	jwtService auth.JWTService,
	passwordVerifier auth.PasswordVerifier,
	authConfig *config.AuthConfig,
	logger *slog.Logger,
) *AuthHandler {
	if logger == nil {
		// ALLOW-PANIC: Constructor enforcing required dependency
		panic("logger cannot be nil for AuthHandler")
	}

	return &AuthHandler{
		userStore:        userStore,
		jwtService:       jwtService,
		passwordVerifier: passwordVerifier,
		authConfig:       authConfig,
		timeFunc:         time.Now, // Default to system time
		logger:           logger.With(slog.String("component", "auth_handler")),
	}
}

// WithTimeFunc returns a new AuthHandler with the given time function.
// This is useful for testing with a fixed time source.
// The original handler remains unchanged (immutable pattern).
func (h *AuthHandler) WithTimeFunc(timeFunc func() time.Time) *AuthHandler {
	// Create a new handler that's a copy of the current one
	newHandler := &AuthHandler{
		userStore:        h.userStore,
		jwtService:       h.jwtService,
		passwordVerifier: h.passwordVerifier,
		authConfig:       h.authConfig,
		timeFunc:         timeFunc, // Set the new time function
		logger:           h.logger,
	}
	return newHandler
}

// Register handles the /auth/register endpoint.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	// Parse request
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Validate request
	if err := shared.Validate.Struct(req); err != nil {
		sanitizedError := SanitizeValidationError(err)
		shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, sanitizedError, err)
		return
	}

	// Create user
	user, err := domain.NewUser(req.Email, req.Password)
	if err != nil {
		// Map domain error to appropriate message and status
		statusCode := MapErrorToStatusCode(err)
		safeMessage := GetSafeErrorMessage(err)
		if safeMessage == "An unexpected error occurred" {
			safeMessage = "Invalid user data"
		}
		shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
		return
	}

	// Store user
	if err := h.userStore.Create(r.Context(), user); err != nil {
		statusCode := MapErrorToStatusCode(err)
		safeMessage := GetSafeErrorMessage(err)
		shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
		return
	}

	// Generate tokens
	accessToken, refreshToken, expiresAt, err := h.generateTokenResponse(r.Context(), user.ID)
	if err != nil {
		shared.RespondWithErrorAndLog(w, r, http.StatusInternalServerError,
			"Failed to generate authentication tokens", err)
		return
	}

	// Return success response with both tokens and expiration time
	shared.RespondWithJSON(w, r, http.StatusCreated, AuthResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	})
}

// RefreshToken handles the /auth/refresh endpoint.
// It validates a refresh token and issues a new access + refresh token pair.
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest

	// Parse request
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Validate request
	if err := shared.Validate.Struct(req); err != nil {
		sanitizedError := SanitizeValidationError(err)
		shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, sanitizedError, err)
		return
	}

	// Validate refresh token
	claims, err := h.jwtService.ValidateRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		// Map different error types to appropriate status codes and messages
		statusCode := MapErrorToStatusCode(err)
		safeMessage := GetSafeErrorMessage(err)
		shared.RespondWithErrorAndLog(w, r, statusCode, safeMessage, err)
		return
	}

	// Extract user ID from claims
	userID := claims.UserID

	// Log successful refresh token validation
	h.logger.Debug("refresh token validated successfully",
		slog.String("user_id", userID.String()),
		slog.String("token_id", claims.ID))

	// Generate tokens
	accessToken, refreshToken, expiresAt, err := h.generateTokenResponse(r.Context(), userID)
	if err != nil {
		shared.RespondWithErrorAndLog(w, r, http.StatusInternalServerError,
			"Failed to generate new authentication tokens", err)
		return
	}

	// Return success response with new tokens and expiration time
	shared.RespondWithJSON(w, r, http.StatusOK, RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	})
}

// Login handles the /auth/login endpoint.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	// Parse request
	if err := shared.DecodeJSON(r, &req); err != nil {
		shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Validate request
	if err := shared.Validate.Struct(req); err != nil {
		sanitizedError := SanitizeValidationError(err)
		shared.RespondWithErrorAndLog(w, r, http.StatusBadRequest, sanitizedError, err)
		return
	}

	// Get user by email
	user, err := h.userStore.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			// Use generic error message for security (don't reveal if email exists)
			// Elevate to WARN level as repeated auth failures are operationally important
			shared.RespondWithErrorAndLog(
				w,
				r,
				http.StatusUnauthorized,
				"Invalid credentials",
				err,
				shared.WithElevatedLogLevel(),
			)
			return
		}
		shared.RespondWithErrorAndLog(w, r, http.StatusInternalServerError,
			"Failed to authenticate user", err)
		return
	}

	// Verify password using the injected verifier
	if err := h.passwordVerifier.Compare(user.HashedPassword, req.Password); err != nil {
		// Use same generic error message as above for security
		// Elevate to WARN level as repeated auth failures are operationally important
		shared.RespondWithErrorAndLog(
			w,
			r,
			http.StatusUnauthorized,
			"Invalid credentials",
			err,
			shared.WithElevatedLogLevel(),
		)
		return
	}

	// Generate tokens
	accessToken, refreshToken, expiresAt, err := h.generateTokenResponse(r.Context(), user.ID)
	if err != nil {
		shared.RespondWithErrorAndLog(w, r, http.StatusInternalServerError,
			"Failed to generate authentication tokens", err)
		return
	}

	// Return success response with both tokens and expiration time
	shared.RespondWithJSON(w, r, http.StatusOK, AuthResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	})
}
