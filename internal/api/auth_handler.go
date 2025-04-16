package api

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"

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

	// Generate token
	token, err := h.jwtService.GenerateToken(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to generate token", "error", err, "user_id", user.ID)
		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}

	// Calculate token expiration time
	expiresAt := time.Now().Add(time.Duration(h.authConfig.TokenLifetimeMinutes) * time.Minute)

	// Format expiration time in RFC3339 format (standard for JSON API responses)
	expiresAtFormatted := expiresAt.Format(time.RFC3339)

	// Return success response with expiration time
	RespondWithJSON(w, r, http.StatusCreated, AuthResponse{
		UserID:      user.ID,
		AccessToken: token,
		ExpiresAt:   expiresAtFormatted,
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

	// Generate token
	token, err := h.jwtService.GenerateToken(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to generate token", "error", err, "user_id", user.ID)
		RespondWithError(w, r, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}

	// Calculate token expiration time
	expiresAt := time.Now().Add(time.Duration(h.authConfig.TokenLifetimeMinutes) * time.Minute)

	// Format expiration time in RFC3339 format (standard for JSON API responses)
	expiresAtFormatted := expiresAt.Format(time.RFC3339)

	// Return success response with expiration time
	RespondWithJSON(w, r, http.StatusOK, AuthResponse{
		UserID:      user.ID,
		AccessToken: token,
		ExpiresAt:   expiresAtFormatted,
	})
}
