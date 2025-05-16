package service

import (
	"context"
	"database/sql"
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

// Common sentinel errors for UserService
var (
	// ErrUserNotFound indicates that the user does not exist
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailExists indicates that the email is already in use
	ErrEmailExists = errors.New("email already exists")
)

// UserServiceError wraps errors from the user service with context.
type UserServiceError struct {
	// Operation is the operation that failed (e.g., "create_user", "update_email")
	Operation string
	// Message is a human-readable description of the error
	Message string
	// Err is the underlying error that caused the failure
	Err error
}

// Error implements the error interface for UserServiceError.
func (e *UserServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("user service %s failed: %s: %v", e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("user service %s failed: %s", e.Operation, e.Message)
}

// Unwrap returns the wrapped error to support errors.Is/errors.As.
func (e *UserServiceError) Unwrap() error {
	return e.Err
}

// NewUserServiceError creates a new UserServiceError.
// It returns known sentinel errors directly without wrapping.
func NewUserServiceError(operation, message string, err error) error {
	if err == nil {
		return nil
	}

	// Check for service-defined sentinel errors
	if errors.Is(err, ErrUserNotFound) {
		return ErrUserNotFound
	}
	if errors.Is(err, ErrEmailExists) {
		return ErrEmailExists
	}

	// Check for store-level sentinel errors that should be mapped to service-level ones
	if errors.Is(err, store.ErrUserNotFound) {
		return ErrUserNotFound
	}
	if errors.Is(err, store.ErrEmailExists) {
		return ErrEmailExists
	}

	// If not a sentinel to be returned directly, wrap it
	return &UserServiceError{
		Operation: operation,
		Message:   message,
		Err:       err,
	}
}

// UserServiceImpl implements the UserService interface
type UserServiceImpl struct {
	userStore store.UserStore
	logger    *slog.Logger
	db        *sql.DB
}

// NewUserService creates a new UserService
// It returns an error if any of the required dependencies are nil.
func NewUserService(userStore store.UserStore, db *sql.DB, logger *slog.Logger) (UserService, error) {
	// Validate dependencies
	if userStore == nil {
		return nil, &UserServiceError{
			Operation: "create_service",
			Message:   "userStore cannot be nil",
			Err:       nil,
		}
	}
	if db == nil {
		return nil, &UserServiceError{
			Operation: "create_service",
			Message:   "db cannot be nil",
			Err:       nil,
		}
	}

	// Use provided logger or create default
	if logger == nil {
		logger = slog.Default()
	}

	return &UserServiceImpl{
		userStore: userStore,
		db:        db,
		logger:    logger.With("component", "user_service"),
	}, nil
}

// GetUser retrieves a user by their ID
func (s *UserServiceImpl) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to retrieve user",
			"error", err,
			"user_id", userID)

		if errors.Is(err, store.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, NewUserServiceError("get_user", "failed to retrieve user", err)
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
			return nil, ErrUserNotFound
		} else {
			s.logger.Error("failed to retrieve user by email",
				"error", err,
				"email", email)
		}
		return nil, NewUserServiceError("get_user_by_email", "failed to retrieve user by email", err)
	}

	s.logger.Debug("retrieved user by email successfully",
		"user_id", user.ID,
		"email", user.Email)

	return user, nil
}

// CreateUser creates a new user with the specified email and password
// Uses a transaction to ensure atomicity of the operation
func (s *UserServiceImpl) CreateUser(
	ctx context.Context,
	email, password string,
) (*domain.User, error) {
	user, err := domain.NewUser(email, password)
	if err != nil {
		s.logger.Error("failed to create user object",
			"error", err,
			"email", email)
		return nil, NewUserServiceError("create_user", "failed to create user object", err)
	}

	// Use a transaction for the user creation
	err = store.RunInTransaction(ctx, s.db, func(ctx context.Context, tx *sql.Tx) error {
		// Get a transaction-aware store
		txStore := s.userStore.WithTx(tx)

		// Create the user within the transaction
		err := txStore.Create(ctx, user)
		if err != nil {
			if errors.Is(err, store.ErrEmailExists) {
				s.logger.Debug("attempted to create user with existing email",
					"email", email)
				return ErrEmailExists
			}

			s.logger.Error("failed to save user to database",
				"error", err,
				"email", email)
			return NewUserServiceError("create_user", "failed to save user to database", err)
		}
		return nil
	})

	if err != nil {
		// Error is already wrapped in the transaction
		return nil, err
	}

	s.logger.Info("user created successfully in transaction",
		"user_id", user.ID,
		"email", user.Email)

	return user, nil
}

// UpdateUserEmail updates a user's email address
// Following the pattern of getting the complete user first, then updating the specific field
// Uses a transaction to ensure atomicity of the operation
func (s *UserServiceImpl) UpdateUserEmail(
	ctx context.Context,
	userID uuid.UUID,
	newEmail string,
) error {
	return store.RunInTransaction(ctx, s.db, func(ctx context.Context, tx *sql.Tx) error {
		// Get a transaction-aware store
		txStore := s.userStore.WithTx(tx)

		// First, retrieve the current user to get the complete user object
		user, err := txStore.GetByID(ctx, userID)
		if err != nil {
			s.logger.Error("failed to retrieve user for email update",
				"error", err,
				"user_id", userID)

			if errors.Is(err, store.ErrUserNotFound) {
				return ErrUserNotFound
			}
			return NewUserServiceError("update_email", "failed to retrieve user for update", err)
		}

		// Update only the email field
		user.Email = newEmail

		// Save the complete user object back to the store
		// Note: UserStore.Update now requires a complete user object including HashedPassword
		err = txStore.Update(ctx, user)
		if err != nil {
			if errors.Is(err, store.ErrEmailExists) {
				s.logger.Debug("attempted to update to an existing email",
					"user_id", userID,
					"new_email", newEmail)
				return ErrEmailExists
			} else {
				s.logger.Error("failed to update user email",
					"error", err,
					"user_id", userID,
					"new_email", newEmail)
			}
			return NewUserServiceError("update_email", "failed to update user email", err)
		}

		s.logger.Info("user email updated successfully in transaction",
			"user_id", userID,
			"new_email", newEmail)

		return nil
	})
}

// UpdateUserPassword updates a user's password
// Following the pattern of getting the complete user first, then updating only the specific field
// Uses a transaction to ensure atomicity of the operation
func (s *UserServiceImpl) UpdateUserPassword(
	ctx context.Context,
	userID uuid.UUID,
	newPassword string,
) error {
	return store.RunInTransaction(ctx, s.db, func(ctx context.Context, tx *sql.Tx) error {
		// Get a transaction-aware store
		txStore := s.userStore.WithTx(tx)

		// First, retrieve the current user to get the complete user object
		user, err := txStore.GetByID(ctx, userID)
		if err != nil {
			s.logger.Error("failed to retrieve user for password update",
				"error", err,
				"user_id", userID)

			if errors.Is(err, store.ErrUserNotFound) {
				return ErrUserNotFound
			}
			return NewUserServiceError("update_password", "failed to retrieve user for password update", err)
		}

		// Set the new password (UserStore.Update will handle the hashing)
		user.Password = newPassword

		// Save the complete user object back to the store
		// Note: UserStore.Update now requires a complete user object including HashedPassword
		err = txStore.Update(ctx, user)
		if err != nil {
			s.logger.Error("failed to update user password",
				"error", err,
				"user_id", userID)
			return NewUserServiceError("update_password", "failed to update user password", err)
		}

		s.logger.Info("user password updated successfully in transaction",
			"user_id", userID)

		return nil
	})
}

// DeleteUser deletes a user by their ID
// Uses a transaction to ensure atomicity of the operation
func (s *UserServiceImpl) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return store.RunInTransaction(ctx, s.db, func(ctx context.Context, tx *sql.Tx) error {
		// Get a transaction-aware store
		txStore := s.userStore.WithTx(tx)

		// Delete the user within the transaction
		err := txStore.Delete(ctx, userID)
		if err != nil {
			if errors.Is(err, store.ErrUserNotFound) {
				s.logger.Debug("attempted to delete non-existent user",
					"user_id", userID)
				return ErrUserNotFound
			} else {
				s.logger.Error("failed to delete user",
					"error", err,
					"user_id", userID)
			}
			return NewUserServiceError("delete_user", "failed to delete user", err)
		}

		s.logger.Info("user deleted successfully in transaction",
			"user_id", userID)

		return nil
	})
}
