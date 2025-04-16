package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
)

// AuthHandler handles authentication-related API requests.
type AuthHandler struct {
	userStore        store.UserStore
	jwtService       auth.JWTService
	passwordVerifier auth.PasswordVerifier
	validator        *validator.Validate
	authConfig       *config.AuthConfig // For accessing token lifetime and other auth settings
}

// generateTokenResponse generates access and refresh tokens for a user, along with expiration time.
// Returns the tokens and formatted expiration time, or an error if token generation fails.
func (h *AuthHandler) generateTokenResponse(
	ctx context.Context,
	userID uuid.UUID,
) (accessToken, refreshToken, expiresAt string, err error) {
	// Generate access token
	accessToken, err = h.jwtService.GenerateToken(ctx, userID)
	if err != nil {
		slog.Error("failed to generate access token", "error", err, "user_id", userID)
		return "", "", "", err
	}

	// Generate refresh token
	refreshToken, err = h.jwtService.GenerateRefreshToken(ctx, userID)
	if err != nil {
		slog.Error("failed to generate refresh token", "error", err, "user_id", userID)
		return "", "", "", err
	}

	// Calculate access token expiration time
	expiresAtTime := time.Now().Add(time.Duration(h.authConfig.TokenLifetimeMinutes) * time.Minute)

	// Format expiration time in RFC3339 format (standard for JSON API responses)
	expiresAt = expiresAtTime.Format(time.RFC3339)

	return accessToken, refreshToken, expiresAt, nil
}

// NewAuthHandler creates a new AuthHandler with the given dependencies.
func NewAuthHandler(
	userStore store.UserStore,
	jwtService auth.JWTService,
	passwordVerifier auth.PasswordVerifier,
	authConfig *config.AuthConfig,
) *AuthHandler {
	return &AuthHandler{
		userStore:        userStore,
		jwtService:       jwtService,
		passwordVerifier: passwordVerifier,
		validator:        validator.New(),
		authConfig:       authConfig,
	}
}

// Register handles the /auth/register endpoint.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	// Parse request
	if err := DecodeJSON(r, &req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// Create user
	user, err := domain.NewUser(req.Email, req.Password)
	if err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid user data: "+err.Error())
		return
	}

	// Store user
	if err := h.userStore.Create(r.Context(), user); err != nil {
		if errors.Is(err, store.ErrEmailExists) {
			RespondWithError(w, r, http.StatusConflict, "Email already exists")
			return
		}
		slog.Error("failed to create user", "error", err, "email", req.Email)
		RespondWithError(w, r, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate tokens
	accessToken, refreshToken, expiresAt, err := h.generateTokenResponse(r.Context(), user.ID)
	if err != nil {
		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}

	// Return success response with both tokens and expiration time
	RespondWithJSON(w, r, http.StatusCreated, AuthResponse{
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
	if err := DecodeJSON(r, &req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// Validate refresh token
	claims, err := h.jwtService.ValidateRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		// Map different error types to appropriate HTTP responses
		switch {
		case errors.Is(err, auth.ErrInvalidRefreshToken),
			errors.Is(err, auth.ErrExpiredRefreshToken),
			errors.Is(err, auth.ErrWrongTokenType):
			slog.Debug("refresh token validation failed", "error", err)
			RespondWithError(w, r, http.StatusUnauthorized, "Invalid refresh token")
		default:
			slog.Error("unexpected error validating refresh token", "error", err)
			RespondWithError(w, r, http.StatusInternalServerError, "Failed to validate refresh token")
		}
		return
	}

	// Extract user ID from claims
	userID := claims.UserID

	// Generate tokens
	accessToken, refreshToken, expiresAt, err := h.generateTokenResponse(r.Context(), userID)
	if err != nil {
		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}

	// Return success response with new tokens and expiration time
	RespondWithJSON(w, r, http.StatusOK, RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	})
}

// Login handles the /auth/login endpoint.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	// Parse request
	if err := DecodeJSON(r, &req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		RespondWithError(w, r, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// Get user by email
	user, err := h.userStore.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			RespondWithError(w, r, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		slog.Error("failed to get user by email", "error", err, "email", req.Email)
		RespondWithError(w, r, http.StatusInternalServerError, "Failed to authenticate user")
		return
	}

	// Verify password using the injected verifier
	if err := h.passwordVerifier.Compare(user.HashedPassword, req.Password); err != nil {
		RespondWithError(w, r, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate tokens
	accessToken, refreshToken, expiresAt, err := h.generateTokenResponse(r.Context(), user.ID)
	if err != nil {
		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}

	// Return success response with both tokens and expiration time
	RespondWithJSON(w, r, http.StatusOK, AuthResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	})
}
