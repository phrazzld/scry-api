package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
)

// PostgreSQL error codes
//
//nolint:unused
const uniqueViolationCode = "23505" // PostgreSQL unique violation error code

// PostgresUserStore implements the store.UserStore interface
// using a PostgreSQL database as the storage backend.
type PostgresUserStore struct {
	db *sql.DB
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
//
//nolint:unused
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
func (s *PostgresUserStore) Create(ctx context.Context, user *domain.User) error {
	// Placeholder implementation - will be fully implemented in a separate task
	return nil
}

// GetByID implements store.UserStore.GetByID
func (s *PostgresUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	// Placeholder implementation - will be fully implemented in a separate task
	return nil, nil
}

// GetByEmail implements store.UserStore.GetByEmail
func (s *PostgresUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Placeholder implementation - will be fully implemented in a separate task
	return nil, nil
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
