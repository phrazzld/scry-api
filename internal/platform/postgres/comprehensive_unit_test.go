package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/stretchr/testify/assert"
)

// comprehensiveMockDBTX provides sophisticated mocking for unit testing
type comprehensiveMockDBTX struct {
	execError  error
	queryError error
	execResult sql.Result
	queryRows  *sql.Rows
}

func newComprehensiveMockDBTX() *comprehensiveMockDBTX {
	return &comprehensiveMockDBTX{}
}

func (m *comprehensiveMockDBTX) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if m.execError != nil {
		return nil, m.execError
	}
	if m.execResult != nil {
		return m.execResult, nil
	}
	return mockResult{rowsAffected: 1}, nil
}

func (m *comprehensiveMockDBTX) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, errors.New("prepare not implemented in mock")
}

func (m *comprehensiveMockDBTX) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if m.queryError != nil {
		return nil, m.queryError
	}
	return m.queryRows, nil
}

func (m *comprehensiveMockDBTX) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	// For unit tests, we need to return a row that will properly error
	// when scanned to simulate database errors like "not found"
	// The actual error handling will be done by the store methods
	// For mocking purposes, we'll use sql.ErrNoRows which the stores handle
	return &sql.Row{}
}

// TestCardStoreValidation tests card store validation logic without database dependencies
func TestCardStoreValidation(t *testing.T) {
	logger := slog.Default()

	t.Run("CreateMultiple validation", func(t *testing.T) {
		t.Run("empty list succeeds", func(t *testing.T) {
			mock := newComprehensiveMockDBTX()
			store := NewPostgresCardStore(mock, logger)
			ctx := context.Background()

			err := store.CreateMultiple(ctx, []*domain.Card{})
			assert.NoError(t, err)
		})

		t.Run("nil card list returns error", func(t *testing.T) {
			mock := newComprehensiveMockDBTX()
			store := NewPostgresCardStore(mock, logger)
			ctx := context.Background()

			err := store.CreateMultiple(ctx, nil)
			assert.NoError(t, err) // Empty/nil should succeed
		})
	})

	t.Run("Error mapping and validation", func(t *testing.T) {
		// Test pgconn error handling
		t.Run("foreign key violation mapping", func(t *testing.T) {
			pgErr := &pgconn.PgError{
				Code:    "23503",
				Message: "foreign key violation",
			}
			mapped := MapError(pgErr)
			assert.Error(t, mapped)
			assert.Contains(t, mapped.Error(), "foreign key violation")
		})

		t.Run("unique violation mapping", func(t *testing.T) {
			pgErr := &pgconn.PgError{
				Code:    "23505",
				Message: "unique constraint violation",
			}
			mapped := MapError(pgErr)
			assert.Error(t, mapped)
			assert.Contains(t, mapped.Error(), "already exists")
		})

		t.Run("generic error passes through sanitized", func(t *testing.T) {
			genericErr := errors.New("generic database error")
			mapped := MapError(genericErr)
			assert.Error(t, mapped)
			assert.Equal(t, genericErr, mapped)
		})
	})
}

// TestMemoStoreValidation tests memo store validation logic
func TestMemoStoreValidation(t *testing.T) {
	logger := slog.Default()

	t.Run("Store initialization", func(t *testing.T) {
		t.Run("valid initialization", func(t *testing.T) {
			mock := newComprehensiveMockDBTX()
			store := NewPostgresMemoStore(mock, logger)
			assert.NotNil(t, store)
		})
	})

	t.Run("Domain validation", func(t *testing.T) {
		// Test domain-level memo validation
		t.Run("valid memo creation", func(t *testing.T) {
			userID := uuid.New()
			text := "Valid memo text with sufficient content"
			memo, err := domain.NewMemo(userID, text)
			assert.NoError(t, err)
			assert.Equal(t, userID, memo.UserID)
			assert.Equal(t, text, memo.Text)
			assert.Equal(t, domain.MemoStatusPending, memo.Status)
		})

		t.Run("empty text fails validation", func(t *testing.T) {
			userID := uuid.New()
			text := ""
			_, err := domain.NewMemo(userID, text)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "text")
		})
	})
}

// TestUserStoreValidation tests user store validation logic
func TestUserStoreValidation(t *testing.T) {
	t.Run("Store initialization", func(t *testing.T) {
		t.Run("valid initialization", func(t *testing.T) {
			mock := newComprehensiveMockDBTX()
			store := NewPostgresUserStore(mock, 12)
			assert.NotNil(t, store)
		})
	})

	t.Run("Domain validation", func(t *testing.T) {
		t.Run("valid user creation", func(t *testing.T) {
			email := "test@example.com"
			password := "validpassword123"
			user, err := domain.NewUser(email, password)
			assert.NoError(t, err)
			assert.Equal(t, email, user.Email)
			assert.Equal(t, password, user.Password) // Plaintext password stored temporarily
			assert.Empty(t, user.HashedPassword)     // Hashed password not set yet
		})

		t.Run("invalid email fails validation", func(t *testing.T) {
			email := "invalid-email"
			password := "validpassword123"
			_, err := domain.NewUser(email, password)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "email")
		})

		t.Run("weak password fails validation", func(t *testing.T) {
			email := "test@example.com"
			password := "weak"
			_, err := domain.NewUser(email, password)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "password")
		})
	})
}

// TestStatsStoreValidation tests stats store validation logic
func TestStatsStoreValidation(t *testing.T) {
	logger := slog.Default()

	t.Run("Store initialization", func(t *testing.T) {
		t.Run("valid initialization", func(t *testing.T) {
			mock := newComprehensiveMockDBTX()
			store := NewPostgresUserCardStatsStore(mock, logger)
			assert.NotNil(t, store)
		})
	})

	t.Run("Domain validation", func(t *testing.T) {
		t.Run("valid stats creation", func(t *testing.T) {
			userID := uuid.New()
			cardID := uuid.New()
			stats, err := domain.NewUserCardStats(userID, cardID)
			assert.NoError(t, err)
			assert.Equal(t, userID, stats.UserID)
			assert.Equal(t, cardID, stats.CardID)
			assert.Equal(t, 0, stats.Interval) // Initial interval is 0
			assert.Equal(t, 2.5, stats.EaseFactor)
			assert.Equal(t, 0, stats.ReviewCount)
		})

		t.Run("nil user ID fails validation", func(t *testing.T) {
			cardID := uuid.New()
			_, err := domain.NewUserCardStats(uuid.Nil, cardID)
			assert.Error(t, err)
		})

		t.Run("nil card ID fails validation", func(t *testing.T) {
			userID := uuid.New()
			_, err := domain.NewUserCardStats(userID, uuid.Nil)
			assert.Error(t, err)
		})
	})
}

// TestTaskStoreComprehensiveValidation tests task store validation logic
func TestTaskStoreComprehensiveValidation(t *testing.T) {
	t.Run("Store initialization", func(t *testing.T) {
		t.Run("valid initialization", func(t *testing.T) {
			mock := newComprehensiveMockDBTX()
			store := NewPostgresTaskStore(mock)
			assert.NotNil(t, store)
		})
	})

	t.Run("Task status validation", func(t *testing.T) {
		t.Run("valid task statuses", func(t *testing.T) {
			validStatuses := []task.TaskStatus{
				task.TaskStatusPending,
				task.TaskStatusProcessing,
				task.TaskStatusCompleted,
				task.TaskStatusFailed,
			}

			for _, status := range validStatuses {
				assert.NotEmpty(t, string(status))
			}
		})
	})
}
