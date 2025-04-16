package store

import (
	"context"
	"database/sql"
)

// DBTX is an interface that abstracts the database access layer.
// It is implemented by both *sql.DB and *sql.Tx, allowing our code
// to work with either a database connection or a transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
