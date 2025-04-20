package mocks

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"
)

// MockDB is a mock implementation of *sql.DB for testing
type MockDB struct {
	BeginTxFn func(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// NewMockDB creates a new instance of MockDB with default implementations
func NewMockDB() *MockDB {
	return &MockDB{
		BeginTxFn: func(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
			// This returns nil for testing purposes
			// Tests using this mock should set their own BeginTxFn if needed
			return nil, nil
		},
	}
}

// BeginTx implements the BeginTx method of *sql.DB for the mock
func (m *MockDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return m.BeginTxFn(ctx, opts)
}

// The following methods are required to implement the database/sql.DB interface

func (m *MockDB) Begin() (*sql.Tx, error) {
	return m.BeginTx(context.Background(), nil)
}

func (m *MockDB) Close() error {
	return nil
}

func (m *MockDB) Conn(ctx context.Context) (*sql.Conn, error) {
	return nil, nil
}

func (m *MockDB) Driver() driver.Driver {
	return nil
}

func (m *MockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *MockDB) Ping() error {
	return nil
}

func (m *MockDB) PingContext(ctx context.Context) error {
	return nil
}

func (m *MockDB) Prepare(query string) (*sql.Stmt, error) {
	return nil, nil
}

func (m *MockDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, nil
}

func (m *MockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *MockDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *MockDB) SetConnMaxIdleTime(d time.Duration) {}
func (m *MockDB) SetConnMaxLifetime(d time.Duration) {}
func (m *MockDB) SetMaxIdleConns(n int)              {}
func (m *MockDB) SetMaxOpenConns(n int)              {}
func (m *MockDB) Stats() sql.DBStats                 { return sql.DBStats{} }
