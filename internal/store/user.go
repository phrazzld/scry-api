package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
)

// Common store errors
var (
	// ErrUserNotFound indicates that the requested user does not exist in the store.
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailExists indicates that a user with the given email already exists.
	ErrEmailExists = errors.New("email already exists")
)

// UserStore defines the interface for user data persistence.
type UserStore interface {
	// Create saves a new user to the store.
	// It handles domain validation and password hashing internally.
	// Returns ErrEmailExists if the email is already taken.
	// Returns validation errors from the domain User if data is invalid.
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by their unique ID.
	// Returns ErrUserNotFound if the user does not exist.
	// The returned user contains all fields except the plaintext password.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// GetByEmail retrieves a user by their email address.
	// Returns ErrUserNotFound if the user does not exist.
	// The returned user contains all fields except the plaintext password.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// Update modifies an existing user's details.
	// It handles domain validation and password rehashing if needed.
	// Returns ErrUserNotFound if the user does not exist.
	// Returns ErrEmailExists if updating to an email that already exists.
	// Returns validation errors from the domain User if data is invalid.
	Update(ctx context.Context, user *domain.User) error

	// Delete removes a user from the store by their ID.
	// Returns ErrUserNotFound if the user does not exist.
	// This operation is permanent and cannot be undone.
	Delete(ctx context.Context, id uuid.UUID) error
}
