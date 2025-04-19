package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
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
	// The caller MUST provide a complete user object including HashedPassword.
	// If a new plain text Password is provided, it will be hashed and the HashedPassword will be updated.
	// Returns ErrUserNotFound if the user does not exist.
	// Returns ErrEmailExists if updating to an email that already exists.
	// Returns validation errors from the domain User if data is invalid.
	Update(ctx context.Context, user *domain.User) error

	// Delete removes a user from the store by their ID.
	// Returns ErrUserNotFound if the user does not exist.
	// This operation is permanent and cannot be undone.
	Delete(ctx context.Context, id uuid.UUID) error

	// WithTx returns a new UserStore instance that uses the provided transaction.
	// This allows for multiple operations to be executed within a single transaction.
	// The transaction should be created and managed by the caller (typically a service).
	WithTx(tx *sql.Tx) UserStore
}
