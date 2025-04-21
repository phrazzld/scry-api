package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/phrazzld/scry-api/internal/store"
)

// PostgreSQL error codes
const (
	// uniqueViolationCode is the PostgreSQL error code for unique constraint violations
	uniqueViolationCode = "23505"

	// foreignKeyViolationCode is the PostgreSQL error code for foreign key violations
	foreignKeyViolationCode = "23503"

	// checkViolationCode is the PostgreSQL error code for check constraint violations
	checkViolationCode = "23514"

	// notNullViolationCode is the PostgreSQL error code for not null violations
	notNullViolationCode = "23502"
)

// MapError maps a database error to an appropriate domain error.
// It wraps the original error to preserve context and provide better debugging information.
// This function should be used in all database operations to ensure consistent error handling.
func MapError(err error) error {
	if err == nil {
		return nil
	}

	// Handle common SQL errors
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %v", store.ErrNotFound, err)
	}

	// Handle PostgreSQL-specific errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolationCode:
			return fmt.Errorf("%w: %v", store.ErrDuplicate, err)
		case foreignKeyViolationCode:
			return fmt.Errorf(
				"%w: foreign key violation (%s): %v",
				store.ErrInvalidEntity,
				pgErr.ConstraintName,
				err,
			)
		case checkViolationCode:
			return fmt.Errorf(
				"%w: check constraint violation (%s): %v",
				store.ErrInvalidEntity,
				pgErr.ConstraintName,
				err,
			)
		case notNullViolationCode:
			return fmt.Errorf(
				"%w: not null violation (%s): %v",
				store.ErrInvalidEntity,
				pgErr.ColumnName,
				err,
			)
		}
	}

	// Return the original error for errors that don't have specific mappings
	return err
}

// IsUniqueViolation checks if the given error is a PostgreSQL unique constraint violation.
// This is useful for detecting duplicate records that violate unique constraints.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}

// IsForeignKeyViolation checks if the given error is a PostgreSQL foreign key constraint violation.
// This occurs when an operation would violate referential integrity constraints.
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolationCode
}

// IsCheckConstraintViolation checks if the given error is a PostgreSQL check constraint violation.
// This occurs when an operation would violate a CHECK constraint on a table.
func IsCheckConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == checkViolationCode
}

// IsNotNullViolation checks if the given error is a PostgreSQL not null constraint violation.
// This occurs when an operation attempts to insert or update a NULL value in a column that requires a non-NULL value.
func IsNotNullViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == notNullViolationCode
}

// IsNotFoundError checks if the given error represents a "not found" scenario.
// This handles both sql.ErrNoRows and errors that are or wrap store.ErrNotFound.
func IsNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, store.ErrNotFound)
}

// CheckRowsAffected examines the number of rows affected by a database operation.
// If no rows were affected, it returns store.ErrNotFound.
// This is useful for UPDATE and DELETE operations where the absence of affected rows
// typically indicates that the target record doesn't exist.
func CheckRowsAffected(result sql.Result, entityName string) error {
	if result == nil {
		return fmt.Errorf("nil result provided to CheckRowsAffected")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		if entityName == "" {
			return store.ErrNotFound
		}
		return fmt.Errorf("%w: %s not found", store.ErrNotFound, entityName)
	}

	return nil
}

// MapUniqueViolation maps a PostgreSQL unique violation error to a more specific error.
// If the error is not a unique violation, it returns the original error.
// This is useful for providing more specific error messages for unique constraint violations.
func MapUniqueViolation(
	err error,
	entityName string,
	constraintName string,
	specificError error,
) error {
	if !IsUniqueViolation(err) {
		return err
	}

	// If a specific error is provided, use it
	if specificError != nil {
		return fmt.Errorf("%w: %v", specificError, err)
	}

	// Construct a meaningful error message based on provided information
	var msg string
	if entityName != "" {
		msg = fmt.Sprintf("%s already exists", entityName)
	} else if constraintName != "" {
		msg = fmt.Sprintf("duplicate value for constraint: %s", constraintName)
	} else {
		msg = "duplicate entry"
	}

	return fmt.Errorf("%w: %s: %v", store.ErrDuplicate, msg, err)
}
