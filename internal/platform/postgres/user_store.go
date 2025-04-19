package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// PostgresUserStore implements the store.UserStore interface
// using a PostgreSQL database as the storage backend.
type PostgresUserStore struct {
	db         store.DBTX
	bcryptCost int // Cost parameter for bcrypt password hashing
}

// DB returns the underlying database connection for testing purposes.
// This method is not part of the store.UserStore interface.
// Note: This now returns a store.DBTX interface, which could be either *sql.DB or *sql.Tx
func (s *PostgresUserStore) DB() store.DBTX {
	return s.db
}

// NewPostgresUserStore creates a new PostgreSQL implementation of the UserStore interface.
// It accepts a database connection or transaction that should be initialized and managed by the caller.
// The bcryptCost parameter determines the computational cost of password hashing.
// If bcryptCost is 0 or invalid, bcrypt.DefaultCost (10) will be used.
func NewPostgresUserStore(db store.DBTX, bcryptCost int) *PostgresUserStore {
	// Validate bcrypt cost and use default if invalid
	if bcryptCost < 4 || bcryptCost > 31 {
		bcryptCost = bcrypt.DefaultCost
	}

	return &PostgresUserStore{
		db:         db,
		bcryptCost: bcryptCost,
	}
}

// Ensure PostgresUserStore implements store.UserStore interface
var _ store.UserStore = (*PostgresUserStore)(nil)

// Create implements store.UserStore.Create
// It creates a new user in the database, handling domain validation and password hashing.
// Returns store.ErrEmailExists if a user with the same email already exists.
func (s *PostgresUserStore) Create(ctx context.Context, user *domain.User) error {
	// Get the logger from context or use default
	log := logger.FromContext(ctx)

	// Validate password format if a plaintext password is provided
	if user.Password != "" {
		if err := domain.ValidatePassword(user.Password); err != nil {
			log.Warn("password validation failed during user create",
				slog.String("error", err.Error()),
				slog.String("email", user.Email))
			return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
		}

		// Hash the password using bcrypt with the configured cost
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), s.bcryptCost)
		if err != nil {
			log.Error("failed to hash password",
				slog.String("error", err.Error()),
				slog.Int("bcrypt_cost", s.bcryptCost))
			return fmt.Errorf("failed to hash password: %w", err)
		}

		// Store the hashed password and clear the plaintext password from memory
		user.HashedPassword = string(hashedPassword)
		user.Password = "" // Clear plaintext password from memory for security
	}

	// Validate user data for persistence
	if err := user.Validate(); err != nil {
		log.Warn("user validation failed during create",
			slog.String("error", err.Error()),
			slog.String("email", user.Email))
		return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
	}

	// Insert the user into the database
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, user.ID, user.Email, user.HashedPassword, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if IsUniqueViolation(err) {
			log.Warn("attempt to create user with existing email",
				slog.String("email", user.Email))
			return store.ErrEmailExists
		}
		// Log other errors
		log.Error("failed to insert user",
			slog.String("error", err.Error()),
			slog.String("email", user.Email))
		return MapError(err)
	}

	log.Info("user created successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email))
	return nil
}

// GetByID implements store.UserStore.GetByID
// It retrieves a user by their unique ID from the database.
// Returns store.ErrUserNotFound if the user does not exist.
func (s *PostgresUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	// Get the logger from context or use default
	log := logger.FromContext(ctx)

	log.Debug("retrieving user by ID", slog.String("user_id", id.String()))

	// Query the user from database
	var user domain.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, hashed_password, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.HashedPassword, &user.CreatedAt, &user.UpdatedAt)

	// Handle the result
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("user not found", slog.String("user_id", id.String()))
			return nil, store.ErrUserNotFound
		}
		log.Error("failed to query user by ID",
			slog.String("user_id", id.String()),
			slog.String("error", err.Error()))
		return nil, MapError(err)
	}

	// Ensure the Password field is empty as it should never be populated from the database
	user.Password = ""

	log.Debug("user retrieved successfully", slog.String("user_id", id.String()))
	return &user, nil
}

// GetByEmail implements store.UserStore.GetByEmail
// It retrieves a user by their email address from the database.
// Returns store.ErrUserNotFound if the user does not exist.
// The email matching is case-insensitive.
func (s *PostgresUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Get the logger from context or use default
	log := logger.FromContext(ctx)

	log.Debug("retrieving user by email",
		slog.String("email", email))

	// Query the user from database with case-insensitive email matching
	var user domain.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, hashed_password, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`, email).Scan(&user.ID, &user.Email, &user.HashedPassword, &user.CreatedAt, &user.UpdatedAt)

	// Handle the result
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("user not found", slog.String("email", email))
			return nil, store.ErrUserNotFound
		}
		log.Error("failed to query user by email",
			slog.String("email", email),
			slog.String("error", err.Error()))
		return nil, MapError(err)
	}

	// Ensure the Password field is empty as it should never be populated from the database
	user.Password = ""

	log.Debug("user retrieved successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email))
	return &user, nil
}

// Update implements store.UserStore.Update
// It modifies an existing user's details in the database.
// Returns store.ErrUserNotFound if the user does not exist.
// Returns store.ErrEmailExists if updating to an email that already exists.
// Returns validation errors from the domain User if data is invalid.
func (s *PostgresUserStore) Update(ctx context.Context, user *domain.User) error {
	// Get the logger from context or use default
	log := logger.FromContext(ctx)

	log.Debug("updating user", slog.String("user_id", user.ID.String()))

	// Update the timestamp
	user.UpdatedAt = time.Now().UTC()

	// Determine password hash to store
	var hashedPasswordToStore string

	if user.Password != "" {
		// Validate the new password format
		if err := domain.ValidatePassword(user.Password); err != nil {
			log.Warn("password validation failed during user update",
				slog.String("error", err.Error()),
				slog.String("user_id", user.ID.String()))
			return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
		}

		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), s.bcryptCost)
		if err != nil {
			log.Error("failed to hash password",
				slog.String("error", err.Error()),
				slog.String("user_id", user.ID.String()),
				slog.Int("bcrypt_cost", s.bcryptCost))
			return fmt.Errorf("failed to hash password: %w", err)
		}
		hashedPasswordToStore = string(hashedPassword)
		user.Password = "" // Clear plaintext password for security
		user.HashedPassword = hashedPasswordToStore
	} else if user.HashedPassword != "" {
		// If no new password is provided but user already has a hashed password, use that
		hashedPasswordToStore = user.HashedPassword
		log.Debug("using provided hashed password", slog.String("user_id", user.ID.String()))
	} else {
		// Only fetch the existing password hash if we don't have a new password or existing hash
		log.Debug("fetching existing hashed password", slog.String("user_id", user.ID.String()))
		err := s.db.QueryRowContext(ctx, `
			SELECT hashed_password FROM users WHERE id = $1
		`, user.ID).Scan(&hashedPasswordToStore)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Debug("user not found", slog.String("user_id", user.ID.String()))
				return store.ErrUserNotFound
			}
			log.Error("failed to fetch existing user",
				slog.String("error", err.Error()),
				slog.String("user_id", user.ID.String()))
			return MapError(err)
		}
		// Store the fetched hashed password in the user object
		user.HashedPassword = hashedPasswordToStore
	}

	// Validate the user for persistence after any modifications
	if err := user.Validate(); err != nil {
		log.Warn("user validation failed during update",
			slog.String("error", err.Error()),
			slog.String("user_id", user.ID.String()))
		return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
	}

	// Execute the update statement
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET email = $1, hashed_password = $2, updated_at = $3
		WHERE id = $4
	`, user.Email, hashedPasswordToStore, user.UpdatedAt, user.ID)

	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if IsUniqueViolation(err) {
			log.Warn("email already exists",
				slog.String("email", user.Email),
				slog.String("user_id", user.ID.String()))
			return store.ErrEmailExists
		}
		// Log other errors
		log.Error("failed to update user",
			slog.String("error", err.Error()),
			slog.String("user_id", user.ID.String()))
		return MapError(err)
	}

	// Check if a row was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected",
			slog.String("error", err.Error()),
			slog.String("user_id", user.ID.String()))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// If no rows were affected, the user didn't exist
	if rowsAffected == 0 {
		log.Debug("user not found for update", slog.String("user_id", user.ID.String()))
		return store.ErrUserNotFound
	}

	log.Info("user updated successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email))
	return nil
}

// Delete implements store.UserStore.Delete
// It removes a user from the database by their ID.
// Returns store.ErrUserNotFound if the user does not exist.
func (s *PostgresUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	// Get the logger from context or use default
	log := logger.FromContext(ctx)

	log.Debug("deleting user by ID", slog.String("user_id", id.String()))

	// Execute the DELETE statement
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM users
		WHERE id = $1
	`, id)

	// Handle execution errors
	if err != nil {
		log.Error("failed to execute delete statement",
			slog.String("user_id", id.String()),
			slog.String("error", err.Error()))
		return MapError(err)
	}

	// Check if a row was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected",
			slog.String("user_id", id.String()),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// If no rows were affected, the user didn't exist
	if rowsAffected == 0 {
		log.Debug("user not found for deletion", slog.String("user_id", id.String()))
		return store.ErrUserNotFound
	}

	log.Info("user deleted successfully", slog.String("user_id", id.String()))
	return nil
}
