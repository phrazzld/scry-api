package mocks

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/mock"
)

// UserStore is a mock of store.UserStore interface for use with testify/mock
type UserStore struct {
	mock.Mock
}

// Create is a mock implementation of store.UserStore.Create
func (m *UserStore) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// GetByID is a mock implementation of store.UserStore.GetByID
func (m *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if user, ok := args.Get(0).(*domain.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

// GetByEmail is a mock implementation of store.UserStore.GetByEmail
func (m *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if user, ok := args.Get(0).(*domain.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

// Update is a mock implementation of store.UserStore.Update
func (m *UserStore) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// Delete is a mock implementation of store.UserStore.Delete
func (m *UserStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// WithTx is a mock implementation of store.UserStore.WithTx
func (m *UserStore) WithTx(tx *sql.Tx) store.UserStore {
	args := m.Called(tx)
	if ret, ok := args.Get(0).(store.UserStore); ok {
		return ret
	}
	return m
}
