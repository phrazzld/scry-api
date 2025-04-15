package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/phrazzld/scry-api/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// PostgreSQL error codes
const uniqueViolationCode = "23505" // PostgreSQL unique violation error code

// PostgresUserStore implements the store.UserStore interface
// using a PostgreSQL database as the storage backend.
type PostgresUserStore struct {
	db *sql.DB
}

// DB returns the underlying database connection for testing purposes.
// This method is not part of the store.UserStore interface.
func (s *PostgresUserStore) DB() *sql.DB {
	return s.db
}

// NewPostgresUserStore creates a new PostgreSQL implementation of the UserStore interface.
// It accepts a database connection that should be initialized and managed by the caller.
func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
	return &PostgresUserStore{
		db: db,
	}
}

// isUniqueViolation checks if the given error is a PostgreSQL unique constraint violation.
// This is used to detect when an operation fails due to a unique constraint,
// such as duplicate email addresses.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return true
	}
	return false
}

// Ensure PostgresUserStore implements store.UserStore interface
var _ store.UserStore = (*PostgresUserStore)(nil)

// Create implements store.UserStore.Create
// It creates a new user in the database, handling domain validation and password hashing.
// Returns store.ErrEmailExists if a user with the same email already exists.
func (s *PostgresUserStore) Create(ctx context.Context, user *domain.User) error {
	// Get the logger from context or use default
	log := logger.FromContext(ctx)

	// First, validate the user data
	if err := user.Validate(); err != nil {
		log.Warn("user validation failed during create",
			slog.String("error", err.Error()),
			slog.String("email", user.Email))
		return err
	}

	// Hash the password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to hash password",
			slog.String("error", err.Error()))
		return err
	}

	// Store the hashed password and clear the plaintext password from memory
	user.HashedPassword = string(hashedPassword)
	user.Password = "" // Clear plaintext password from memory for security

	// Start a transaction to ensure data consistency
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction",
			slog.String("error", err.Error()))
		return err
	}
	// Defer a rollback in case anything fails
	defer func() {
		// If error occurs, attempt to rollback
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error("failed to rollback transaction",
					slog.String("rollback_error", rbErr.Error()),
					slog.String("original_error", err.Error()))
			}
		}
	}()

	// Insert the user into the database
	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (id, email, hashed_password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, user.ID, user.Email, user.HashedPassword, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if isUniqueViolation(err) {
			log.Warn("attempt to create user with existing email",
				slog.String("email", user.Email))
			return store.ErrEmailExists
		}
		// Log other errors
		log.Error("failed to insert user",
			slog.String("error", err.Error()),
			slog.String("email", user.Email))
		return err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		log.Error("failed to commit transaction",
			slog.String("error", err.Error()),
			slog.String("email", user.Email))
		return err
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
		return nil, err
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
		return nil, err
	}

	// Ensure the Password field is empty as it should never be populated from the database
	user.Password = ""

	log.Debug("user retrieved successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email))
	return &user, nil
}

// Update implements store.UserStore.Update
func (s *PostgresUserStore) Update(ctx context.Context, user *domain.User) error {
	// Placeholder implementation - will be fully implemented in a separate task
	return nil
}

// Delete implements store.UserStore.Delete
func (s *PostgresUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	// Placeholder implementation - will be fully implemented in a separate task
	return nil
}
