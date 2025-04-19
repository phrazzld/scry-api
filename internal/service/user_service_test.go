package service_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserService_UpdateUserEmail(t *testing.T) {
	// Setup
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	userID := uuid.New()
	email := "user@example.com"
	newEmail := "new@example.com"
	hashedPassword := "hashed_password123"

	t.Run("successful update", func(t *testing.T) {
		// Create a mock UserStore
		mockUserStore := new(mocks.UserStore)

		// Create an existing user object as the store would return it
		existingUser := &domain.User{
			ID:             userID,
			Email:          email,
			HashedPassword: hashedPassword,
			CreatedAt:      time.Now().Add(-24 * time.Hour),
			UpdatedAt:      time.Now().Add(-24 * time.Hour),
		}

		// No need to create an updatedUser variable, we'll use a matcher function instead

		// Setup expectations
		mockUserStore.On("GetByID", mock.Anything, userID).Return(existingUser, nil)

		// Verify that Update is called with the complete user object
		mockUserStore.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.ID == userID &&
				u.Email == newEmail &&
				u.HashedPassword == hashedPassword &&
				u.CreatedAt.Equal(existingUser.CreatedAt)
		})).Return(nil)

		// Create UserService with the mock
		userService := service.NewUserService(mockUserStore, logger)

		// Test the UpdateUserEmail method
		err := userService.UpdateUserEmail(context.Background(), userID, newEmail)

		// Assertions
		require.NoError(t, err)
		mockUserStore.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		// Create a mock UserStore
		mockUserStore := new(mocks.UserStore)

		// Setup expectations
		mockUserStore.On("GetByID", mock.Anything, userID).Return(nil, store.ErrUserNotFound)

		// Create UserService with the mock
		userService := service.NewUserService(mockUserStore, logger)

		// Test the UpdateUserEmail method
		err := userService.UpdateUserEmail(context.Background(), userID, newEmail)

		// Assertions
		require.Error(t, err)
		assert.True(t, errors.Is(err, store.ErrUserNotFound))
		mockUserStore.AssertExpectations(t)
	})

	t.Run("email already exists", func(t *testing.T) {
		// Create a mock UserStore
		mockUserStore := new(mocks.UserStore)

		// Create an existing user object
		existingUser := &domain.User{
			ID:             userID,
			Email:          email,
			HashedPassword: hashedPassword,
			CreatedAt:      time.Now().Add(-24 * time.Hour),
			UpdatedAt:      time.Now().Add(-24 * time.Hour),
		}

		// Setup expectations
		mockUserStore.On("GetByID", mock.Anything, userID).Return(existingUser, nil)
		mockUserStore.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.ID == userID &&
				u.Email == newEmail &&
				u.HashedPassword == hashedPassword
		})).Return(store.ErrEmailExists)

		// Create UserService with the mock
		userService := service.NewUserService(mockUserStore, logger)

		// Test the UpdateUserEmail method
		err := userService.UpdateUserEmail(context.Background(), userID, newEmail)

		// Assertions
		require.Error(t, err)
		assert.True(t, errors.Is(err, store.ErrEmailExists))
		mockUserStore.AssertExpectations(t)
	})
}

func TestUserService_UpdateUserPassword(t *testing.T) {
	// Setup
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	userID := uuid.New()
	email := "user@example.com"
	hashedPassword := "hashed_password123"
	newPassword := "NewPassword123!"

	t.Run("successful update", func(t *testing.T) {
		// Create a mock UserStore
		mockUserStore := new(mocks.UserStore)

		// Create an existing user object
		existingUser := &domain.User{
			ID:             userID,
			Email:          email,
			HashedPassword: hashedPassword,
			CreatedAt:      time.Now().Add(-24 * time.Hour),
			UpdatedAt:      time.Now().Add(-24 * time.Hour),
		}

		// Setup expectations
		mockUserStore.On("GetByID", mock.Anything, userID).Return(existingUser, nil)

		// Verify that Update is called with the complete user object including the new password
		mockUserStore.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.ID == userID &&
				u.Email == email &&
				u.Password == newPassword && // New password is set
				u.HashedPassword == hashedPassword && // Original password hash is preserved
				u.CreatedAt.Equal(existingUser.CreatedAt)
		})).Return(nil)

		// Create UserService with the mock
		userService := service.NewUserService(mockUserStore, logger)

		// Test the UpdateUserPassword method
		err := userService.UpdateUserPassword(context.Background(), userID, newPassword)

		// Assertions
		require.NoError(t, err)
		mockUserStore.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		// Create a mock UserStore
		mockUserStore := new(mocks.UserStore)

		// Setup expectations
		mockUserStore.On("GetByID", mock.Anything, userID).Return(nil, store.ErrUserNotFound)

		// Create UserService with the mock
		userService := service.NewUserService(mockUserStore, logger)

		// Test the UpdateUserPassword method
		err := userService.UpdateUserPassword(context.Background(), userID, newPassword)

		// Assertions
		require.Error(t, err)
		assert.True(t, errors.Is(err, store.ErrUserNotFound))
		mockUserStore.AssertExpectations(t)
	})

	t.Run("invalid password", func(t *testing.T) {
		// Create a mock UserStore
		mockUserStore := new(mocks.UserStore)

		// Create an existing user object
		existingUser := &domain.User{
			ID:             userID,
			Email:          email,
			HashedPassword: hashedPassword,
			CreatedAt:      time.Now().Add(-24 * time.Hour),
			UpdatedAt:      time.Now().Add(-24 * time.Hour),
		}

		// Setup expectations
		mockUserStore.On("GetByID", mock.Anything, userID).Return(existingUser, nil)
		mockUserStore.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.ID == userID &&
				u.Password == "short" // Too short password
		})).Return(store.ErrInvalidEntity)

		// Create UserService with the mock
		userService := service.NewUserService(mockUserStore, logger)

		// Test the UpdateUserPassword method with invalid password
		err := userService.UpdateUserPassword(context.Background(), userID, "short")

		// Assertions
		require.Error(t, err)
		assert.True(t, errors.Is(err, store.ErrInvalidEntity))
		mockUserStore.AssertExpectations(t)
	})
}
