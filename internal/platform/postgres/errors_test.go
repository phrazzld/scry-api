package postgres_test

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
)

// Mock PgError creation helper
func newPgError(code string) *pgconn.PgError {
	return &pgconn.PgError{
		Code:           code,
		Message:        "error message",
		Detail:         "error details",
		Hint:           "error hint",
		Position:       0,
		InternalQuery:  "",
		Where:          "",
		SchemaName:     "public",
		TableName:      "test_table",
		ColumnName:     "test_column",
		DataTypeName:   "",
		ConstraintName: "test_constraint",
		File:           "postgres.go",
		Line:           100,
		Routine:        "test_routine",
	}
}

// MockResult implements sql.Result for testing
type MockResult struct {
	rowsAffected int64
	lastInsertId int64
	err          error
}

func (m MockResult) LastInsertId() (int64, error) {
	return m.lastInsertId, m.err
}

func (m MockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, m.err
}

// TestIsUniqueViolation tests the IsUniqueViolation function
func TestIsUniqueViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-postgres error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "unique violation",
			err:      newPgError("23505"),
			expected: true,
		},
		{
			name:     "foreign key violation",
			err:      newPgError("23503"),
			expected: false,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := postgres.IsUniqueViolation(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsForeignKeyViolation tests the IsForeignKeyViolation function
func TestIsForeignKeyViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-postgres error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "unique violation",
			err:      newPgError("23505"),
			expected: false,
		},
		{
			name:     "foreign key violation",
			err:      newPgError("23503"),
			expected: true,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := postgres.IsForeignKeyViolation(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsCheckConstraintViolation tests the IsCheckConstraintViolation function
func TestIsCheckConstraintViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-postgres error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "check constraint violation",
			err:      newPgError("23514"),
			expected: true,
		},
		{
			name:     "unique violation",
			err:      newPgError("23505"),
			expected: false,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := postgres.IsCheckConstraintViolation(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsNotNullViolation tests the IsNotNullViolation function
func TestIsNotNullViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-postgres error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "not null violation",
			err:      newPgError("23502"),
			expected: true,
		},
		{
			name:     "unique violation",
			err:      newPgError("23505"),
			expected: false,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := postgres.IsNotNullViolation(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsNotFoundError tests the IsNotFoundError function
func TestIsNotFoundError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "sql.ErrNoRows",
			err:      sql.ErrNoRows,
			expected: true,
		},
		{
			name:     "store.ErrNotFound",
			err:      store.ErrNotFound,
			expected: true,
		},
		{
			name:     "wrapped store.ErrNotFound",
			err:      fmt.Errorf("wrapped: %w", store.ErrNotFound),
			expected: true,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := postgres.IsNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCheckRowsAffected tests the CheckRowsAffected function
func TestCheckRowsAffected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		result     sql.Result
		entityName string
		wantErr    bool
		errIs      error
	}{
		{
			name:       "nil result",
			result:     nil,
			entityName: "",
			wantErr:    true,
			errIs:      nil, // Not checking specific error type
		},
		{
			name:       "zero rows affected",
			result:     MockResult{rowsAffected: 0},
			entityName: "",
			wantErr:    true,
			errIs:      store.ErrNotFound,
		},
		{
			name:       "zero rows affected with entity name",
			result:     MockResult{rowsAffected: 0},
			entityName: "User",
			wantErr:    true,
			errIs:      store.ErrNotFound,
		},
		{
			name:       "one row affected",
			result:     MockResult{rowsAffected: 1},
			entityName: "",
			wantErr:    false,
			errIs:      nil,
		},
		{
			name:       "error getting rows affected",
			result:     MockResult{err: errors.New("rows affected error")},
			entityName: "",
			wantErr:    true,
			errIs:      nil, // Not checking specific error type
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := postgres.CheckRowsAffected(tt.result, tt.entityName)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMapError tests the MapError function
func TestMapError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		errIs  error
		errMsg string
	}{
		{
			name:   "nil error",
			err:    nil,
			errIs:  nil,
			errMsg: "",
		},
		{
			name:   "sql.ErrNoRows",
			err:    sql.ErrNoRows,
			errIs:  store.ErrNotFound,
			errMsg: "entity not found",
		},
		{
			name:   "unique violation",
			err:    newPgError("23505"),
			errIs:  store.ErrDuplicate,
			errMsg: "entity already exists",
		},
		{
			name:   "foreign key violation",
			err:    newPgError("23503"),
			errIs:  store.ErrInvalidEntity,
			errMsg: "foreign key violation",
		},
		{
			name:   "check constraint violation",
			err:    newPgError("23514"),
			errIs:  store.ErrInvalidEntity,
			errMsg: "check constraint violation",
		},
		{
			name:   "not null violation",
			err:    newPgError("23502"),
			errIs:  store.ErrInvalidEntity,
			errMsg: "not null violation",
		},
		{
			name:   "other postgres error",
			err:    newPgError("42P01"), // undefined_table
			errIs:  nil,                 // Should return the original error
			errMsg: "",
		},
		{
			name:   "generic error",
			err:    errors.New("generic error"),
			errIs:  nil, // Should return the original error
			errMsg: "",
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := postgres.MapError(tt.err)

			if tt.err == nil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)

			if tt.errIs != nil {
				assert.ErrorIs(t, result, tt.errIs)
				if tt.errMsg != "" {
					assert.Contains(t, result.Error(), tt.errMsg)
				}
			} else {
				// For cases where we expect the original error to be returned
				assert.Equal(t, tt.err.Error(), result.Error())
			}
		})
	}
}

// TestMapUniqueViolation tests the MapUniqueViolation function
func TestMapUniqueViolation(t *testing.T) {
	t.Parallel()

	specificError := errors.New("specific duplicate error")

	tests := []struct {
		name           string
		err            error
		entityName     string
		constraintName string
		specificError  error
		errIs          error
		errContains    string
	}{
		{
			name:           "non-unique violation",
			err:            errors.New("generic error"),
			entityName:     "User",
			constraintName: "",
			specificError:  nil,
			errIs:          nil, // Should return the original error
			errContains:    "generic error",
		},
		{
			name:           "unique violation without specifics",
			err:            newPgError("23505"),
			entityName:     "",
			constraintName: "",
			specificError:  nil,
			errIs:          store.ErrDuplicate,
			errContains:    "duplicate entry",
		},
		{
			name:           "unique violation with entity name",
			err:            newPgError("23505"),
			entityName:     "User",
			constraintName: "",
			specificError:  nil,
			errIs:          store.ErrDuplicate,
			errContains:    "User already exists",
		},
		{
			name:           "unique violation with constraint name",
			err:            newPgError("23505"),
			entityName:     "",
			constraintName: "users_email_key",
			specificError:  nil,
			errIs:          store.ErrDuplicate,
			errContains:    "duplicate value for constraint: users_email_key",
		},
		{
			name:           "unique violation with specific error",
			err:            newPgError("23505"),
			entityName:     "User",
			constraintName: "users_email_key",
			specificError:  specificError,
			errIs:          specificError,
			errContains:    "specific duplicate error",
		},
		{
			name:           "entity name takes precedence over constraint name",
			err:            newPgError("23505"),
			entityName:     "User",
			constraintName: "users_email_key",
			specificError:  nil,
			errIs:          store.ErrDuplicate,
			errContains:    "User already exists",
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := postgres.MapUniqueViolation(
				tt.err,
				tt.entityName,
				tt.constraintName,
				tt.specificError,
			)

			if !postgres.IsUniqueViolation(tt.err) {
				assert.Equal(t, tt.err, result)
				return
			}

			assert.NotNil(t, result)

			if tt.errIs != nil {
				assert.ErrorIs(t, result, tt.errIs)
			}

			if tt.errContains != "" {
				assert.Contains(t, result.Error(), tt.errContains)
			}
		})
	}
}
