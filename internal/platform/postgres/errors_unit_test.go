package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockResult implements sql.Result for testing
type mockResult struct {
	lastInsertId int64
	rowsAffected int64
	err          error
}

func (m mockResult) LastInsertId() (int64, error) {
	return m.lastInsertId, nil
}

func (m mockResult) RowsAffected() (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.rowsAffected, nil
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedError error
		expectedMsg   string
	}{
		{
			name:          "nil_error",
			err:           nil,
			expectedError: nil,
		},
		{
			name:          "sql_no_rows",
			err:           sql.ErrNoRows,
			expectedError: store.ErrNotFound,
		},
		{
			name: "unique_violation",
			err: &pgconn.PgError{
				Code:           uniqueViolationCode,
				ConstraintName: "users_email_key",
			},
			expectedMsg: "entity already exists",
		},
		{
			name: "foreign_key_violation",
			err: &pgconn.PgError{
				Code: foreignKeyViolationCode,
			},
			expectedMsg: "foreign key violation",
		},
		{
			name: "check_constraint_violation",
			err: &pgconn.PgError{
				Code: checkViolationCode,
			},
			expectedMsg: "validation rule violation",
		},
		{
			name: "not_null_violation",
			err: &pgconn.PgError{
				Code: notNullViolationCode,
			},
			expectedMsg: "not null violation",
		},
		{
			name:          "generic_error",
			err:           errors.New("some other error"),
			expectedError: errors.New("some other error"),
		},
		{
			name: "unknown_pg_code",
			err: &pgconn.PgError{
				Code:    "99999",
				Message: "unknown error",
			},
			expectedError: &pgconn.PgError{
				Code:    "99999",
				Message: "unknown error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapError(tt.err)

			if tt.expectedError == nil && tt.expectedMsg == "" {
				assert.Nil(t, result)
			} else if tt.expectedMsg != "" {
				require.NotNil(t, result)
				assert.Contains(t, result.Error(), tt.expectedMsg)
				// Check that it wraps the appropriate store error
				if errors.Is(result, store.ErrDuplicate) || errors.Is(result, store.ErrInvalidEntity) {
					// Good - it wraps one of the expected errors
				} else {
					t.Errorf("Expected error to wrap store.ErrDuplicate or store.ErrInvalidEntity")
				}
			} else if errors.Is(tt.expectedError, store.ErrNotFound) {
				assert.ErrorIs(t, result, store.ErrNotFound)
			} else {
				assert.Equal(t, tt.expectedError.Error(), result.Error())
			}
		})
	}
}

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
		{
			name: "unique_violation",
			err: &pgconn.PgError{
				Code: uniqueViolationCode,
			},
			expected: true,
		},
		{
			name: "other_violation",
			err: &pgconn.PgError{
				Code: foreignKeyViolationCode,
			},
			expected: false,
		},
		{
			name:     "non_pg_error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "wrapped_unique_violation",
			err: fmt.Errorf("context: %w", &pgconn.PgError{
				Code: uniqueViolationCode,
			}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUniqueViolation(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsForeignKeyViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
		{
			name: "foreign_key_violation",
			err: &pgconn.PgError{
				Code: foreignKeyViolationCode,
			},
			expected: true,
		},
		{
			name: "other_violation",
			err: &pgconn.PgError{
				Code: uniqueViolationCode,
			},
			expected: false,
		},
		{
			name:     "non_pg_error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "wrapped_foreign_key_violation",
			err: fmt.Errorf("context: %w", &pgconn.PgError{
				Code: foreignKeyViolationCode,
			}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsForeignKeyViolation(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCheckConstraintViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
		{
			name: "check_constraint_violation",
			err: &pgconn.PgError{
				Code: checkViolationCode,
			},
			expected: true,
		},
		{
			name: "other_violation",
			err: &pgconn.PgError{
				Code: uniqueViolationCode,
			},
			expected: false,
		},
		{
			name:     "non_pg_error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "wrapped_check_constraint_violation",
			err: fmt.Errorf("context: %w", &pgconn.PgError{
				Code: checkViolationCode,
			}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCheckConstraintViolation(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNotNullViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
		{
			name: "not_null_violation",
			err: &pgconn.PgError{
				Code: notNullViolationCode,
			},
			expected: true,
		},
		{
			name: "other_violation",
			err: &pgconn.PgError{
				Code: uniqueViolationCode,
			},
			expected: false,
		},
		{
			name:     "non_pg_error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name: "wrapped_not_null_violation",
			err: fmt.Errorf("context: %w", &pgconn.PgError{
				Code: notNullViolationCode,
			}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotNullViolation(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "sql_no_rows",
			err:      sql.ErrNoRows,
			expected: true,
		},
		{
			name:     "store_not_found",
			err:      store.ErrNotFound,
			expected: true,
		},
		{
			name:     "wrapped_sql_no_rows",
			err:      fmt.Errorf("wrapped: %w", sql.ErrNoRows),
			expected: true,
		},
		{
			name:     "wrapped_store_not_found",
			err:      fmt.Errorf("wrapped: %w", store.ErrNotFound),
			expected: true,
		},
		{
			name:     "other_error",
			err:      errors.New("other error"),
			expected: false,
		},
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckRowsAffected(t *testing.T) {
	tests := []struct {
		name        string
		result      sql.Result
		entityName  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil_result",
			result:      nil,
			entityName:  "user",
			expectError: true,
			errorMsg:    "invalid result",
		},
		{
			name: "zero_rows_affected_with_entity",
			result: mockResult{
				rowsAffected: 0,
			},
			entityName:  "user",
			expectError: true,
			errorMsg:    "user not found",
		},
		{
			name: "zero_rows_affected_no_entity",
			result: mockResult{
				rowsAffected: 0,
			},
			entityName:  "",
			expectError: true,
			errorMsg:    "",
		},
		{
			name: "one_row_affected",
			result: mockResult{
				rowsAffected: 1,
			},
			entityName:  "user",
			expectError: false,
		},
		{
			name: "multiple_rows_affected",
			result: mockResult{
				rowsAffected: 5,
			},
			entityName:  "user",
			expectError: false,
		},
		{
			name: "error_getting_rows_affected",
			result: mockResult{
				err: errors.New("db error"),
			},
			entityName:  "user",
			expectError: true,
			errorMsg:    "database operation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckRowsAffected(tt.result, tt.entityName)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				// Verify appropriate store errors are returned
				if tt.result == nil {
					assert.ErrorIs(t, err, store.ErrInternal)
				} else if tt.errorMsg == "" {
					assert.ErrorIs(t, err, store.ErrNotFound)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMapUniqueViolation(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		entityName     string
		constraintName string
		specificError  error
		expectedError  error
		checkMsg       string
	}{
		{
			name: "unique_violation_with_specific_error",
			err: &pgconn.PgError{
				Code:           uniqueViolationCode,
				ConstraintName: "users_email_key",
			},
			entityName:     "user",
			constraintName: "users_email_key",
			specificError:  store.ErrEmailExists,
			expectedError:  store.ErrEmailExists,
		},
		{
			name: "unique_violation_with_entity_name",
			err: &pgconn.PgError{
				Code:           uniqueViolationCode,
				ConstraintName: "some_constraint",
			},
			entityName:     "user",
			constraintName: "",
			specificError:  nil,
			checkMsg:       "user already exists",
		},
		{
			name: "unique_violation_with_constraint_name",
			err: &pgconn.PgError{
				Code:           uniqueViolationCode,
				ConstraintName: "some_constraint",
			},
			entityName:     "",
			constraintName: "email_unique",
			specificError:  nil,
			checkMsg:       "duplicate value for: email_unique",
		},
		{
			name: "unique_violation_no_details",
			err: &pgconn.PgError{
				Code:           uniqueViolationCode,
				ConstraintName: "some_constraint",
			},
			entityName:     "",
			constraintName: "",
			specificError:  nil,
			checkMsg:       "duplicate entry",
		},
		{
			name:           "non_unique_violation",
			err:            errors.New("some other error"),
			entityName:     "user",
			constraintName: "constraint",
			specificError:  store.ErrEmailExists,
			checkMsg:       "some other error",
		},
		{
			name:           "nil_error",
			err:            nil,
			entityName:     "user",
			constraintName: "constraint",
			specificError:  store.ErrEmailExists,
			expectedError:  nil,
		},
		{
			name: "pgconn_error_non_unique",
			err: &pgconn.PgError{
				Code:    foreignKeyViolationCode,
				Message: "foreign key violation",
			},
			entityName:     "user",
			constraintName: "constraint",
			specificError:  store.ErrEmailExists,
			checkMsg:       "invalid entity: foreign key violation", // MapError returns this for FK violations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapUniqueViolation(tt.err, tt.entityName, tt.constraintName, tt.specificError)

			if tt.expectedError != nil {
				assert.ErrorIs(t, result, tt.expectedError)
			} else if tt.checkMsg != "" {
				if result != nil {
					assert.Contains(t, result.Error(), tt.checkMsg)
				}
			} else if tt.err == nil {
				assert.Nil(t, result)
			} else {
				// For non-unique violations, it should be sanitized
				assert.NotNil(t, result)
			}
		})
	}
}
