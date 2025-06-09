package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserStore provides a mock implementation of store.UserStore for testing
type MockUserStore struct {
	mock.Mock
}

func (m *MockUserStore) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserStore) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserStore) WithTx(tx *sql.Tx) store.UserStore {
	args := m.Called(tx)
	return args.Get(0).(store.UserStore)
}

// Helper function to create a test user
func createTestUser(userID uuid.UUID, email string) *domain.User {
	user, _ := domain.NewUser(email, "hashedpass123")
	user.ID = userID
	return user
}

func TestUserService_Constructor(t *testing.T) {
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()

	t.Run("successful_construction", func(t *testing.T) {
		service, err := NewUserService(mockStore, db, logger)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("nil_store", func(t *testing.T) {
		service, err := NewUserService(nil, db, logger)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "userStore")
	})

	t.Run("nil_db", func(t *testing.T) {
		service, err := NewUserService(mockStore, nil, logger)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "db")
	})

	t.Run("nil_logger_uses_default", func(t *testing.T) {
		service, err := NewUserService(mockStore, db, nil)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})
}

func TestUserService_GetUser(t *testing.T) {
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()

	service, err := NewUserService(mockStore, db, logger)
	require.NoError(t, err)

	userID := uuid.New()
	ctx := context.Background()

	t.Run("successful_get", func(t *testing.T) {
		expectedUser := createTestUser(userID, "test@example.com")
		mockStore.On("GetByID", ctx, userID).Return(expectedUser, nil).Once()

		user, err := service.GetUser(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockStore.AssertExpectations(t)
	})

	t.Run("user_not_found", func(t *testing.T) {
		mockStore.On("GetByID", ctx, userID).Return(nil, store.ErrUserNotFound).Once()

		user, err := service.GetUser(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		// The service should map store errors appropriately
		assert.Contains(t, err.Error(), "user not found")
		mockStore.AssertExpectations(t)
	})

	t.Run("store_error", func(t *testing.T) {
		storeErr := errors.New("database connection failed")
		mockStore.On("GetByID", ctx, userID).Return(nil, storeErr).Once()

		user, err := service.GetUser(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		// Should be wrapped in a service error
		assert.Contains(t, err.Error(), "get_user")
		mockStore.AssertExpectations(t)
	})
}

func TestUserService_GetUserByEmail(t *testing.T) {
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()

	service, err := NewUserService(mockStore, db, logger)
	require.NoError(t, err)

	email := "test@example.com"
	ctx := context.Background()

	t.Run("successful_get", func(t *testing.T) {
		expectedUser := createTestUser(uuid.New(), email)
		mockStore.On("GetByEmail", ctx, email).Return(expectedUser, nil).Once()

		user, err := service.GetUserByEmail(ctx, email)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockStore.AssertExpectations(t)
	})

	t.Run("user_not_found", func(t *testing.T) {
		mockStore.On("GetByEmail", ctx, email).Return(nil, store.ErrUserNotFound).Once()

		user, err := service.GetUserByEmail(ctx, email)
		assert.Error(t, err)
		assert.Nil(t, user)
		// The service should map store errors appropriately
		assert.Contains(t, err.Error(), "user not found")
		mockStore.AssertExpectations(t)
	})

	t.Run("store_error", func(t *testing.T) {
		storeErr := errors.New("database timeout")
		mockStore.On("GetByEmail", ctx, email).Return(nil, storeErr).Once()

		user, err := service.GetUserByEmail(ctx, email)
		assert.Error(t, err)
		assert.Nil(t, user)
		// Should be wrapped in a service error
		assert.Contains(t, err.Error(), "get_user_by_email")
		mockStore.AssertExpectations(t)
	})
}

func TestUserService_CreateUser(t *testing.T) {
	// Note: Full transaction testing is done in user_service_tx_test.go
	// This focuses on validation and error handling that can be unit tested
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()

	service, err := NewUserService(mockStore, db, logger)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("invalid_email", func(t *testing.T) {
		user, err := service.CreateUser(ctx, "invalid-email", "password123456")
		assert.Error(t, err)
		assert.Nil(t, user)
		// Should fail with validation error related to email format
		assert.Contains(t, err.Error(), "create_user")
	})

	t.Run("invalid_password", func(t *testing.T) {
		user, err := service.CreateUser(ctx, "test@example.com", "short")
		assert.Error(t, err)
		assert.Nil(t, user)
		// Should fail with password validation error
		assert.Contains(t, err.Error(), "password must be at least")
	})
}

func TestUserServiceError_Unwrap(t *testing.T) {
	// Test the Unwrap method for error chaining
	innerErr := errors.New("database error")
	serviceErr := NewUserServiceError("test_operation", "test failed", innerErr)

	// Test error wrapping functionality
	assert.Error(t, serviceErr)
	assert.Contains(t, serviceErr.Error(), "test_operation")

	// Test errors.Is works with unwrapping
	assert.True(t, errors.Is(serviceErr, innerErr))

	// Test with nil inner error (returns nil)
	serviceErrNil := NewUserServiceError("test_operation", "test message", nil)
	assert.Nil(t, serviceErrNil)
}

func TestUserServiceError_Comprehensive(t *testing.T) {
	t.Run("error_creation_and_formatting", func(t *testing.T) {
		innerErr := errors.New("connection timeout")
		serviceErr := NewUserServiceError("get_user", "failed to retrieve user", innerErr)

		// Test Error method
		errorMsg := serviceErr.Error()
		assert.Contains(t, errorMsg, "get_user")
		assert.Contains(t, errorMsg, "connection timeout")
	})

	t.Run("error_with_nil_cause", func(t *testing.T) {
		serviceErr := NewUserServiceError("validation", "validation failed", nil)
		// Should return nil when cause is nil
		assert.Nil(t, serviceErr)
	})

	t.Run("error_chaining_compatibility", func(t *testing.T) {
		// Test that error works with Go's error handling
		innerErr := errors.New("database connection failed")
		serviceErr := NewUserServiceError("create_user", "failed to create", innerErr)

		// Should work with errors.Is
		assert.True(t, errors.Is(serviceErr, innerErr))

		// Should work with errors.As for the wrapped error
		assert.Error(t, serviceErr)
	})
}

func TestUserService_UpdateUserEmail(t *testing.T) {
	// Test error handling paths that can be unit tested
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock that will fail transactions
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()

	service, err := NewUserService(mockStore, db, logger)
	require.NoError(t, err)

	userID := uuid.New()
	ctx := context.Background()

	t.Run("transaction_failure", func(t *testing.T) {
		// Test transaction failure path
		mock.ExpectBegin().WillReturnError(errors.New("connection failed"))

		err := service.UpdateUserEmail(ctx, userID, "valid@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestUserService_UpdateUserPassword(t *testing.T) {
	// Test error handling paths that can be unit tested
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock that will fail transactions
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()

	service, err := NewUserService(mockStore, db, logger)
	require.NoError(t, err)

	userID := uuid.New()
	ctx := context.Background()

	t.Run("transaction_failure", func(t *testing.T) {
		// Test transaction failure path
		mock.ExpectBegin().WillReturnError(errors.New("connection failed"))

		err := service.UpdateUserPassword(ctx, userID, "ValidPassword123!")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	// Test error handling paths that can be unit tested
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock that will fail transactions
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()

	service, err := NewUserService(mockStore, db, logger)
	require.NoError(t, err)

	userID := uuid.New()
	ctx := context.Background()

	t.Run("transaction_failure", func(t *testing.T) {
		// Test transaction failure path
		mock.ExpectBegin().WillReturnError(errors.New("connection failed"))

		err := service.DeleteUser(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// Test additional error mapping and validation scenarios
func TestUserService_ErrorMapping(t *testing.T) {
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()
	service, err := NewUserService(mockStore, db, logger)
	require.NoError(t, err)

	ctx := context.Background()
	userID := uuid.New()
	email := "test@example.com"

	t.Run("GetUser_store_error_mapping", func(t *testing.T) {
		// Test that store.ErrUserNotFound is mapped to ErrUserNotFound
		mockStore.On("GetByID", ctx, userID).Return(nil, store.ErrUserNotFound).Once()

		user, err := service.GetUser(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, ErrUserNotFound))
		mockStore.AssertExpectations(t)
	})

	t.Run("GetUser_generic_store_error_wrapping", func(t *testing.T) {
		genericErr := errors.New("database connection timeout")
		mockStore.On("GetByID", ctx, userID).Return(nil, genericErr).Once()

		user, err := service.GetUser(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, user)

		// Should be wrapped in UserServiceError
		var userSvcErr *UserServiceError
		assert.True(t, errors.As(err, &userSvcErr))
		assert.Equal(t, "get_user", userSvcErr.Operation)
		assert.True(t, errors.Is(err, genericErr))
		mockStore.AssertExpectations(t)
	})

	t.Run("GetUserByEmail_store_error_mapping", func(t *testing.T) {
		// Test that store.ErrUserNotFound is mapped to ErrUserNotFound
		mockStore.On("GetByEmail", ctx, email).Return(nil, store.ErrUserNotFound).Once()

		user, err := service.GetUserByEmail(ctx, email)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, ErrUserNotFound))
		mockStore.AssertExpectations(t)
	})

	t.Run("GetUserByEmail_generic_store_error_wrapping", func(t *testing.T) {
		genericErr := errors.New("connection pool exhausted")
		mockStore.On("GetByEmail", ctx, email).Return(nil, genericErr).Once()

		user, err := service.GetUserByEmail(ctx, email)
		assert.Error(t, err)
		assert.Nil(t, user)

		// Should be wrapped in UserServiceError
		var userSvcErr *UserServiceError
		assert.True(t, errors.As(err, &userSvcErr))
		assert.Equal(t, "get_user_by_email", userSvcErr.Operation)
		assert.True(t, errors.Is(err, genericErr))
		mockStore.AssertExpectations(t)
	})
}

// Test CreateUser validation scenarios - simplified to avoid transaction complexity
func TestUserService_CreateUser_ValidationEdgeCases(t *testing.T) {
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()
	service, err := NewUserService(mockStore, db, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Only test the most basic validation cases that fail before transactions
	tests := []struct {
		name          string
		email         string
		password      string
		expectError   bool
		errorContains string
	}{
		{
			name:          "empty_email",
			email:         "",
			password:      "ValidPassword123!",
			expectError:   true,
			errorContains: "email",
		},
		{
			name:          "password_too_short",
			email:         "valid@example.com",
			password:      "short",
			expectError:   true,
			errorContains: "password must be at least",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.CreateUser(ctx, tt.email, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, user)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}
		})
	}
}

// Test NewUserServiceError edge cases
func TestNewUserServiceError_EdgeCases(t *testing.T) {
	t.Run("nil_error_returns_nil", func(t *testing.T) {
		err := NewUserServiceError("test_op", "test message", nil)
		assert.Nil(t, err)
	})

	t.Run("store_ErrUserNotFound_returns_sentinel", func(t *testing.T) {
		err := NewUserServiceError("test_op", "test message", store.ErrUserNotFound)
		assert.Equal(t, ErrUserNotFound, err)
		assert.True(t, errors.Is(err, ErrUserNotFound))
	})

	t.Run("store_ErrEmailExists_returns_sentinel", func(t *testing.T) {
		err := NewUserServiceError("test_op", "test message", store.ErrEmailExists)
		assert.Equal(t, ErrEmailExists, err)
		assert.True(t, errors.Is(err, ErrEmailExists))
	})

	t.Run("service_ErrUserNotFound_returns_sentinel", func(t *testing.T) {
		err := NewUserServiceError("test_op", "test message", ErrUserNotFound)
		assert.Equal(t, ErrUserNotFound, err)
		assert.True(t, errors.Is(err, ErrUserNotFound))
	})

	t.Run("service_ErrEmailExists_returns_sentinel", func(t *testing.T) {
		err := NewUserServiceError("test_op", "test message", ErrEmailExists)
		assert.Equal(t, ErrEmailExists, err)
		assert.True(t, errors.Is(err, ErrEmailExists))
	})

	t.Run("generic_error_gets_wrapped", func(t *testing.T) {
		originalErr := errors.New("some database error")
		err := NewUserServiceError("test_op", "test message", originalErr)

		var userSvcErr *UserServiceError
		assert.True(t, errors.As(err, &userSvcErr))
		assert.Equal(t, "test_op", userSvcErr.Operation)
		assert.Equal(t, "test message", userSvcErr.Message)
		assert.True(t, errors.Is(err, originalErr))
	})
}

// Test UserServiceError methods
func TestUserServiceError_Methods(t *testing.T) {
	t.Run("Error_method_with_underlying_error", func(t *testing.T) {
		innerErr := errors.New("database error")
		userErr := &UserServiceError{
			Operation: "create",
			Message:   "failed to create",
			Err:       innerErr,
		}

		expected := "user service create failed: failed to create: database error"
		assert.Equal(t, expected, userErr.Error())
	})

	t.Run("Error_method_without_underlying_error", func(t *testing.T) {
		userErr := &UserServiceError{
			Operation: "validate",
			Message:   "validation failed",
			Err:       nil,
		}

		expected := "user service validate failed: validation failed"
		assert.Equal(t, expected, userErr.Error())
	})

	t.Run("Unwrap_method", func(t *testing.T) {
		innerErr := errors.New("wrapped error")
		userErr := &UserServiceError{
			Operation: "test",
			Message:   "test message",
			Err:       innerErr,
		}

		assert.Equal(t, innerErr, userErr.Unwrap())

		// Test with nil error
		userErr.Err = nil
		assert.Nil(t, userErr.Unwrap())
	})
}

// Test service behavior with edge case inputs
func TestUserService_EdgeCaseInputs(t *testing.T) {
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := slog.Default()
	service, err := NewUserService(mockStore, db, logger)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("GetUser_zero_uuid", func(t *testing.T) {
		mockStore.On("GetByID", ctx, uuid.UUID{}).Return(nil, store.ErrUserNotFound).Once()

		user, err := service.GetUser(ctx, uuid.UUID{})
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, ErrUserNotFound))
		mockStore.AssertExpectations(t)
	})

	t.Run("GetUserByEmail_empty_email", func(t *testing.T) {
		mockStore.On("GetByEmail", ctx, "").Return(nil, store.ErrUserNotFound).Once()

		user, err := service.GetUserByEmail(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, errors.Is(err, ErrUserNotFound))
		mockStore.AssertExpectations(t)
	})

	t.Run("GetUserByEmail_very_long_email", func(t *testing.T) {
		// Test with extremely long email
		longEmail := "very" + strings.Repeat("long", 100) + "@example.com"
		mockStore.On("GetByEmail", ctx, longEmail).Return(nil, store.ErrUserNotFound).Once()

		user, err := service.GetUserByEmail(ctx, longEmail)
		assert.Error(t, err)
		assert.Nil(t, user)
		mockStore.AssertExpectations(t)
	})
}

// Test logger handling with nil logger
func TestUserService_LoggerHandling(t *testing.T) {
	mockStore := new(MockUserStore)

	// Create a mock database connection using sqlmock
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	t.Run("service_with_nil_logger", func(t *testing.T) {
		// Create service with nil logger - should use default
		service, err := NewUserService(mockStore, db, nil)
		require.NoError(t, err)

		userID := uuid.New()
		ctx := context.Background()

		// Test that operations don't panic with nil logger
		expectedUser := createTestUser(userID, "test@example.com")
		mockStore.On("GetByID", ctx, userID).Return(expectedUser, nil).Once()

		user, err := service.GetUser(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		mockStore.AssertExpectations(t)
	})
}
