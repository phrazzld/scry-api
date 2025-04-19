package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
)

// UserService provides user-related operations including updates
type UserService interface {
	// GetUser retrieves a user by their ID
	GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)

	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)

	// CreateUser creates a new user with the specified email and password
	CreateUser(ctx context.Context, email, password string) (*domain.User, error)

	// UpdateUserEmail updates a user's email address
	// Note: This uses the pattern of first retrieving the full user, then updating the specific field,
	// and finally passing the complete user object back to the store layer
	UpdateUserEmail(ctx context.Context, userID uuid.UUID, newEmail string) error

	// UpdateUserPassword updates a user's password
	// Following the pattern of getting the full user first, then updating only the specific field
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, newPassword string) error

	// DeleteUser deletes a user by their ID
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}

// UserServiceImpl implements the UserService interface
type UserServiceImpl struct {
	userStore store.UserStore
	logger    *slog.Logger
}

// NewUserService creates a new UserService
func NewUserService(userStore store.UserStore, logger *slog.Logger) UserService {
	return &UserServiceImpl{
		userStore: userStore,
		logger:    logger.With("component", "user_service"),
	}
}

// GetUser retrieves a user by their ID
func (s *UserServiceImpl) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to retrieve user",
			"error", err,
			"user_id", userID)
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	s.logger.Debug("retrieved user successfully",
		"user_id", userID,
		"email", user.Email)

	return user, nil
}

// GetUserByEmail retrieves a user by their email address
func (s *UserServiceImpl) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := s.userStore.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			s.logger.Debug("user not found by email",
				"email", email)
		} else {
			s.logger.Error("failed to retrieve user by email",
				"error", err,
				"email", email)
		}
		return nil, fmt.Errorf("failed to retrieve user by email: %w", err)
	}

	s.logger.Debug("retrieved user by email successfully",
		"user_id", user.ID,
		"email", user.Email)

	return user, nil
}

// CreateUser creates a new user with the specified email and password
func (s *UserServiceImpl) CreateUser(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := domain.NewUser(email, password)
	if err != nil {
		s.logger.Error("failed to create user object",
			"error", err,
			"email", email)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	err = s.userStore.Create(ctx, user)
	if err != nil {
		if errors.Is(err, store.ErrEmailExists) {
			s.logger.Debug("attempted to create user with existing email",
				"email", email)
		} else {
			s.logger.Error("failed to save user to database",
				"error", err,
				"email", email)
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("user created successfully",
		"user_id", user.ID,
		"email", user.Email)

	return user, nil
}

// UpdateUserEmail updates a user's email address
// Following the pattern of getting the complete user first, then updating the specific field
func (s *UserServiceImpl) UpdateUserEmail(ctx context.Context, userID uuid.UUID, newEmail string) error {
	// First, retrieve the current user to get the complete user object
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to retrieve user for email update",
			"error", err,
			"user_id", userID)
		return fmt.Errorf("failed to retrieve user for update: %w", err)
	}

	// Update only the email field
	user.Email = newEmail

	// Save the complete user object back to the store
	// Note: UserStore.Update now requires a complete user object including HashedPassword
	err = s.userStore.Update(ctx, user)
	if err != nil {
		if errors.Is(err, store.ErrEmailExists) {
			s.logger.Debug("attempted to update to an existing email",
				"user_id", userID,
				"new_email", newEmail)
		} else {
			s.logger.Error("failed to update user email",
				"error", err,
				"user_id", userID,
				"new_email", newEmail)
		}
		return fmt.Errorf("failed to update user email: %w", err)
	}

	s.logger.Info("user email updated successfully",
		"user_id", userID,
		"new_email", newEmail)

	return nil
}

// UpdateUserPassword updates a user's password
// Following the pattern of getting the complete user first, then updating only the specific field
func (s *UserServiceImpl) UpdateUserPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	// First, retrieve the current user to get the complete user object
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to retrieve user for password update",
			"error", err,
			"user_id", userID)
		return fmt.Errorf("failed to retrieve user for password update: %w", err)
	}

	// Set the new password (UserStore.Update will handle the hashing)
	user.Password = newPassword

	// Save the complete user object back to the store
	// Note: UserStore.Update now requires a complete user object including HashedPassword
	err = s.userStore.Update(ctx, user)
	if err != nil {
		s.logger.Error("failed to update user password",
			"error", err,
			"user_id", userID)
		return fmt.Errorf("failed to update user password: %w", err)
	}

	s.logger.Info("user password updated successfully",
		"user_id", userID)

	return nil
}

// DeleteUser deletes a user by their ID
func (s *UserServiceImpl) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	err := s.userStore.Delete(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			s.logger.Debug("attempted to delete non-existent user",
				"user_id", userID)
		} else {
			s.logger.Error("failed to delete user",
				"error", err,
				"user_id", userID)
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.logger.Info("user deleted successfully",
		"user_id", userID)

	return nil
}
