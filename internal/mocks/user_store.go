package mocks

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
)

// MockUserStore implements store.UserStore for testing
type MockUserStore struct {
	// Function fields for customizable behavior
	CreateFn     func(ctx context.Context, user *domain.User) error
	GetByEmailFn func(ctx context.Context, email string) (*domain.User, error)
	GetByIDFn    func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateFn     func(ctx context.Context, user *domain.User) error
	DeleteFn     func(ctx context.Context, id uuid.UUID) error

	// Data for default implementation
	Users           map[string]*domain.User
	LastUserID      uuid.UUID
	CreateError     error
	GetByEmailError error
}

// NewMockUserStore creates a new mock store with initialized defaults
func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		Users: make(map[string]*domain.User),
	}
}

// Create implements the UserStore interface
func (m *MockUserStore) Create(ctx context.Context, user *domain.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, user)
	}

	if m.CreateError != nil {
		return m.CreateError
	}

	if _, exists := m.Users[user.Email]; exists {
		return store.ErrEmailExists
	}

	m.Users[user.Email] = user
	m.LastUserID = user.ID
	return nil
}

// GetByEmail implements the UserStore interface
func (m *MockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}

	if m.GetByEmailError != nil {
		return nil, m.GetByEmailError
	}

	user, exists := m.Users[email]
	if !exists {
		return nil, store.ErrUserNotFound
	}

	return user, nil
}

// GetByID implements the UserStore interface
func (m *MockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}

	// Default implementation searches through Users map
	for _, user := range m.Users {
		if user.ID == id {
			return user, nil
		}
	}

	return nil, store.ErrUserNotFound
}

// Update implements the UserStore interface
func (m *MockUserStore) Update(ctx context.Context, user *domain.User) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, user)
	}

	// Default simple implementation - just replace user with same ID
	for email, existingUser := range m.Users {
		if existingUser.ID == user.ID {
			// If email changed, check it's not taken
			if email != user.Email {
				if _, exists := m.Users[user.Email]; exists {
					return store.ErrEmailExists
				}
				// Remove old entry
				delete(m.Users, email)
			}

			// Store updated user
			m.Users[user.Email] = user
			return nil
		}
	}

	return store.ErrUserNotFound
}

// Delete implements the UserStore interface
func (m *MockUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}

	// Default implementation searches through Users map
	for email, user := range m.Users {
		if user.ID == id {
			delete(m.Users, email)
			return nil
		}
	}

	return store.ErrUserNotFound
}

// WithTx implements the UserStore interface for transaction support
func (m *MockUserStore) WithTx(tx *sql.Tx) store.UserStore {
	// For mock purposes, just return the same mock
	// In a real implementation, this would create a new store with the transaction
	return m
}

// MockLoginUserStore is a specialized mock for login tests
type MockLoginUserStore struct {
	GetByEmailFn    func(ctx context.Context, email string) (*domain.User, error)
	GetByEmailError error
	UserID          uuid.UUID
	UserEmail       string
	HashedPassword  string
}

// NewLoginMockUserStore creates a specialized mock for login testing
func NewLoginMockUserStore(userID uuid.UUID, email, hashedPassword string) *MockLoginUserStore {
	return &MockLoginUserStore{
		UserID:         userID,
		UserEmail:      email,
		HashedPassword: hashedPassword,
	}
}

// Create - placeholder implementation for UserStore interface
func (m *MockLoginUserStore) Create(ctx context.Context, user *domain.User) error {
	return nil
}

// GetByEmail implements the UserStore interface with login-specific behavior
func (m *MockLoginUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}

	if m.GetByEmailError != nil {
		return nil, m.GetByEmailError
	}

	if email != m.UserEmail {
		return nil, store.ErrUserNotFound
	}

	return &domain.User{
		ID:             m.UserID,
		Email:          m.UserEmail,
		HashedPassword: m.HashedPassword,
	}, nil
}

// GetByID - placeholder implementation for UserStore interface
func (m *MockLoginUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, nil
}

// Update - placeholder implementation for UserStore interface
func (m *MockLoginUserStore) Update(ctx context.Context, user *domain.User) error {
	return nil
}

// Delete - placeholder implementation for UserStore interface
func (m *MockLoginUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// WithTx implements the UserStore interface for transaction support
func (m *MockLoginUserStore) WithTx(tx *sql.Tx) store.UserStore {
	// For mock purposes, just return the same mock
	return m
}
