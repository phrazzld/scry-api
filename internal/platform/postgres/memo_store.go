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
	"github.com/phrazzld/scry-api/internal/store"
)

// PostgresMemoStore implements the MemoRepository interface for PostgreSQL
type PostgresMemoStore struct {
	db     store.DBTX
	logger *slog.Logger
}

// NewPostgresMemoStore creates a new PostgreSQL implementation of the MemoRepository
func NewPostgresMemoStore(db store.DBTX, logger *slog.Logger) *PostgresMemoStore {
	return &PostgresMemoStore{
		db:     db,
		logger: logger.With("component", "memo_store"),
	}
}

// Create saves a new memo to the database
func (s *PostgresMemoStore) Create(ctx context.Context, memo *domain.Memo) error {
	query := `
		INSERT INTO memos (id, user_id, text, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.db.ExecContext(
		ctx,
		query,
		memo.ID,
		memo.UserID,
		memo.Text,
		memo.Status,
		memo.CreatedAt,
		memo.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("failed to create memo",
			"error", err,
			"memo_id", memo.ID,
			"user_id", memo.UserID)
		return fmt.Errorf("failed to create memo: %w", err)
	}

	return nil
}

// GetByID retrieves a memo by its unique ID
func (s *PostgresMemoStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	query := `
		SELECT id, user_id, text, status, created_at, updated_at
		FROM memos
		WHERE id = $1
	`

	var memo domain.Memo
	var status string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&memo.ID,
		&memo.UserID,
		&memo.Text,
		&status,
		&memo.CreatedAt,
		&memo.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Info("memo not found", "memo_id", id)
			return nil, fmt.Errorf("memo not found: %w", err)
		}
		s.logger.Error("failed to get memo by ID",
			"error", err,
			"memo_id", id)
		return nil, fmt.Errorf("failed to get memo: %w", err)
	}

	memo.Status = domain.MemoStatus(status)
	return &memo, nil
}

// Update saves changes to an existing memo
func (s *PostgresMemoStore) Update(ctx context.Context, memo *domain.Memo) error {
	query := `
		UPDATE memos
		SET text = $1, status = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := s.db.ExecContext(
		ctx,
		query,
		memo.Text,
		memo.Status,
		time.Now().UTC(), // Always update the updated_at time to now
		memo.ID,
	)

	if err != nil {
		s.logger.Error("failed to update memo",
			"error", err,
			"memo_id", memo.ID,
			"status", memo.Status)
		return fmt.Errorf("failed to update memo: %w", err)
	}

	return nil
}
