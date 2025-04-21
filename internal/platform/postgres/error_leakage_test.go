package postgres_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
)

// TestMapErrorNoLeakage tests that MapError does not leak internal details
func TestMapErrorNoLeakage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		shouldMap bool // whether we expect the error to be mapped to a domain error
	}{
		{
			name:      "nil error",
			err:       nil,
			shouldMap: false,
		},
		{
			name:      "unique violation",
			err:       newPgError("23505"),
			shouldMap: true,
		},
		{
			name:      "foreign key violation",
			err:       newPgError("23503"),
			shouldMap: true,
		},
		{
			name:      "not null violation",
			err:       newPgError("23502"),
			shouldMap: true,
		},
		{
			name:      "check constraint violation",
			err:       newPgError("23514"),
			shouldMap: true,
		},
		{
			name:      "generic database error",
			err:       errors.New("database error"),
			shouldMap: false, // passes through unchanged
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

			if tt.shouldMap {
				// For mapped errors, verify they've been properly translated
				// to domain errors and don't leak details
				assert.NotEqual(t, tt.err, result, "Error should have been mapped")

				// Verify proper error wrapping
				var pgErr *pgconn.PgError
				assert.False(t, errors.As(result, &pgErr),
					"PostgreSQL error details should not be accessible in mapped error")

				// Check common domain errors are used
				isDomainErr := errors.Is(result, store.ErrNotFound) ||
					errors.Is(result, store.ErrDuplicate) ||
					errors.Is(result, store.ErrInvalidEntity)
				assert.True(t, isDomainErr, "Error should be mapped to a standard domain error")
			}

			// Check that no sensitive details are leaked
			AssertNoErrorLeakage(t, result)
		})
	}
}

// TestErrorsConsistency creates a set of tests that verify all error mapping
// functions in the package maintain consistency in error messages and types
func TestErrorsConsistency(t *testing.T) {
	t.Parallel()

	// Test that CheckRowsAffected doesn't leak implementation details
	t.Run("CheckRowsAffected", func(t *testing.T) {
		t.Parallel()

		err := postgres.CheckRowsAffected(MockResult{rowsAffected: 0}, "User")
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrNotFound)
		AssertNoErrorLeakage(t, err)
	})

	// Test that MapUniqueViolation doesn't leak implementation details
	t.Run("MapUniqueViolation", func(t *testing.T) {
		t.Parallel()

		// Test with various parameters
		testCases := []struct {
			entityName     string
			constraintName string
		}{
			{"User", ""},
			{"", "users_email_key"},
			{"User", "users_email_key"},
			{"", ""},
		}

		for _, tc := range testCases {
			name := tc.entityName
			if name == "" {
				name = "no entity"
			}
			if tc.constraintName != "" {
				name += " with " + tc.constraintName
			}

			t.Run(name, func(t *testing.T) {
				t.Parallel()

				err := postgres.MapUniqueViolation(
					newPgError("23505"),
					tc.entityName,
					tc.constraintName,
					nil,
				)

				assert.Error(t, err)
				assert.ErrorIs(t, err, store.ErrDuplicate)
				AssertNoErrorLeakage(t, err)
			})
		}
	})
}
