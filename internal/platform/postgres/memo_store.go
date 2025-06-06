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
)

// Compile-time check to ensure PostgresMemoStore implements store.MemoStore
var _ store.MemoStore = (*PostgresMemoStore)(nil)

// PostgresMemoStore implements the store.MemoStore interface
// using a PostgreSQL database as the storage backend.
type PostgresMemoStore struct {
	db     store.DBTX
	logger *slog.Logger
}

// NewPostgresMemoStore creates a new PostgreSQL implementation of the MemoStore interface.
// It accepts a database connection or transaction that should be initialized and managed by the caller.
// If logger is nil, a default logger will be used.
func NewPostgresMemoStore(db store.DBTX, logger *slog.Logger) *PostgresMemoStore {
	// Validate inputs
	if db == nil {
		// ALLOW-PANIC: Constructor enforcing required dependency
		panic("db cannot be nil")
	}

	// Use provided logger or create default
	if logger == nil {
		logger = slog.Default()
	}

	return &PostgresMemoStore{
		db:     db,
		logger: logger.With(slog.String("component", "memo_store")),
	}
}

// Ensure PostgresMemoStore implements store.MemoStore interface
var _ store.MemoStore = (*PostgresMemoStore)(nil)

// Create implements store.MemoStore.Create
// It saves a new memo to the database, handling domain validation.
// Returns validation errors from the domain Memo if data is invalid.
// Returns store.ErrInvalidEntity if the user ID doesn't exist (foreign key violation).
func (s *PostgresMemoStore) Create(ctx context.Context, memo *domain.Memo) error {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	// Validate memo data
	if err := memo.Validate(); err != nil {
		log.Warn("memo validation failed during create",
			slog.String("error", err.Error()),
			slog.String("memo_id", memo.ID.String()))
		return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
	}

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
		// Check for foreign key violation
		if IsForeignKeyViolation(err) {
			log.Warn("foreign key violation during memo creation",
				slog.String("error", err.Error()),
				slog.String("memo_id", memo.ID.String()),
				slog.String("user_id", memo.UserID.String()))
			return fmt.Errorf("%w: user with ID %s not found",
				store.ErrInvalidEntity, memo.UserID)
		}

		// Log the error
		log.Error("failed to create memo",
			slog.String("error", err.Error()),
			slog.String("memo_id", memo.ID.String()),
			slog.String("user_id", memo.UserID.String()))

		// Use the error mapping helper with proper context
		return fmt.Errorf("failed to create memo: %w", MapError(err))
	}

	log.Debug("memo created successfully",
		slog.String("memo_id", memo.ID.String()),
		slog.String("user_id", memo.UserID.String()),
		slog.String("status", string(memo.Status)))
	return nil
}

// GetByID implements store.MemoStore.GetByID
// It retrieves a memo by its unique ID.
// Returns store.ErrMemoNotFound if the memo does not exist.
func (s *PostgresMemoStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("retrieving memo by ID", slog.String("memo_id", id.String()))

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
		if IsNotFoundError(err) {
			log.Debug("memo not found", slog.String("memo_id", id.String()))
			return nil, store.ErrMemoNotFound
		}
		log.Error("failed to get memo by ID",
			slog.String("error", err.Error()),
			slog.String("memo_id", id.String()))
		return nil, fmt.Errorf("failed to get memo by ID: %w", MapError(err))
	}

	memo.Status = domain.MemoStatus(status)

	log.Debug("memo retrieved successfully",
		slog.String("memo_id", id.String()),
		slog.String("status", string(memo.Status)))
	return &memo, nil
}

// UpdateStatus implements store.MemoStore.UpdateStatus
// It updates the status of an existing memo.
// Returns store.ErrMemoNotFound if the memo does not exist.
// Returns validation errors if the status is invalid.
func (s *PostgresMemoStore) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.MemoStatus,
) error {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	log.Debug("updating memo status",
		slog.String("memo_id", id.String()),
		slog.String("status", string(status)))

	// Create a temp memo to validate the status
	tempMemo := &domain.Memo{
		ID:        id,
		UserID:    uuid.New(), // This field is required but not used for status validation
		Text:      "temp",     // This field is required but not used for status validation
		Status:    status,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := tempMemo.Validate(); err != nil {
		log.Warn("memo validation failed during status update",
			slog.String("error", err.Error()),
			slog.String("memo_id", id.String()),
			slog.String("status", string(status)))
		return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
	}

	updatedAt := time.Now().UTC()

	query := `
		UPDATE memos
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := s.db.ExecContext(
		ctx,
		query,
		status,
		updatedAt,
		id,
	)

	if err != nil {
		log.Error("failed to update memo status",
			slog.String("error", err.Error()),
			slog.String("memo_id", id.String()),
			slog.String("status", string(status)))
		return fmt.Errorf("failed to update memo status: %w", MapError(err))
	}

	// Check if a row was actually updated using the helper
	err = CheckRowsAffected(result, "memo")
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Convert to specific error
			return store.ErrMemoNotFound
		}
		return fmt.Errorf("failed to update memo status: %w", err)
	}

	log.Debug("memo status updated successfully",
		slog.String("memo_id", id.String()),
		slog.String("status", string(status)))
	return nil
}

// Update implements store.MemoStore.Update
// It saves changes to an existing memo.
// Returns store.ErrMemoNotFound if the memo does not exist.
// Returns validation errors if the memo data is invalid.
func (s *PostgresMemoStore) Update(ctx context.Context, memo *domain.Memo) error {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	// Validate memo data
	if err := memo.Validate(); err != nil {
		log.Warn("memo validation failed during update",
			slog.String("error", err.Error()),
			slog.String("memo_id", memo.ID.String()))
		return fmt.Errorf("%w: %v", store.ErrInvalidEntity, err)
	}

	query := `
		UPDATE memos
		SET text = $1, status = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := s.db.ExecContext(
		ctx,
		query,
		memo.Text,
		memo.Status,
		memo.UpdatedAt,
		memo.ID,
	)

	if err != nil {
		log.Error("failed to update memo",
			slog.String("error", err.Error()),
			slog.String("memo_id", memo.ID.String()),
			slog.String("status", string(memo.Status)))
		return fmt.Errorf("failed to update memo: %w", MapError(err))
	}

	// Check if a row was actually updated using the helper
	err = CheckRowsAffected(result, "memo")
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Convert to specific error
			return store.ErrMemoNotFound
		}
		return fmt.Errorf("failed to update memo: %w", err)
	}

	log.Debug("memo updated successfully",
		slog.String("memo_id", memo.ID.String()),
		slog.String("status", string(memo.Status)))
	return nil
}

// FindMemosByStatus implements store.MemoStore.FindMemosByStatus
// It retrieves all memos with the specified status.
// Returns an empty slice if no memos match the criteria.
func (s *PostgresMemoStore) FindMemosByStatus(
	ctx context.Context,
	status domain.MemoStatus,
	limit, offset int,
) ([]*domain.Memo, error) {
	// Get the logger from context or use default
	log := logger.FromContextOrDefault(ctx, s.logger)

	// Validate limit and offset
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	log.Debug("finding memos by status",
		slog.String("status", string(status)),
		slog.Int("limit", limit),
		slog.Int("offset", offset))

	query := `
		SELECT id, user_id, text, status, created_at, updated_at
		FROM memos
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		log.Error("failed to query memos by status",
			slog.String("error", err.Error()),
			slog.String("status", string(status)))
		return nil, fmt.Errorf("failed to query memos by status: %w", MapError(err))
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Error("failed to close rows", slog.String("error", err.Error()))
		}
	}()

	var memos []*domain.Memo
	for rows.Next() {
		var memo domain.Memo
		var statusStr string

		err := rows.Scan(
			&memo.ID,
			&memo.UserID,
			&memo.Text,
			&statusStr,
			&memo.CreatedAt,
			&memo.UpdatedAt,
		)
		if err != nil {
			log.Error("failed to scan memo row",
				slog.String("error", err.Error()))
			return nil, fmt.Errorf("failed to scan memo row: %w", err)
		}

		memo.Status = domain.MemoStatus(statusStr)
		memos = append(memos, &memo)
	}

	if err := rows.Err(); err != nil {
		log.Error("error after scanning rows",
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("error after scanning rows: %w", err)
	}

	// Return empty slice instead of nil if no memos found
	if memos == nil {
		memos = []*domain.Memo{}
	}

	log.Debug("found memos by status",
		slog.String("status", string(status)),
		slog.Int("count", len(memos)))
	return memos, nil
}

// WithTx implements store.MemoStore.WithTx
// It returns a new MemoStore instance that uses the provided transaction.
// This allows for multiple operations to be executed within a single transaction.
func (s *PostgresMemoStore) WithTx(tx *sql.Tx) store.MemoStore {
	return &PostgresMemoStore{
		db:     tx,
		logger: s.logger,
	}
}
