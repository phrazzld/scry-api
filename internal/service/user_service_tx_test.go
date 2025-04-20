package service_test

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockFailingUserStore is a specialized mock that can be configured to fail at specific operations
type MockFailingUserStore struct {
	mock.Mock
	UserStore         store.UserStore
	FailOnCreate      bool
	FailOnUpdate      bool
	FailOnGetByID     bool
	FailOnGetByEmail  bool
	FailOnDelete      bool
	FailAfterCreate   bool // Special flag to fail after a successful create operation
	FailAfterGetByID  bool // Special flag to fail after a successful GetByID operation
	FailAfterUpdateOp bool // Special flag to fail after a successful update operation
}

func (m *MockFailingUserStore) Create(ctx context.Context, user *domain.User) error {
	if m.FailOnCreate {
		return errors.New("simulated create failure")
	}
	err := m.UserStore.Create(ctx, user)
	if err != nil {
		return err
	}
	if m.FailAfterCreate {
		return errors.New("simulated failure after successful create")
	}
	return nil
}

func (m *MockFailingUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.FailOnGetByID {
		return nil, errors.New("simulated GetByID failure")
	}
	user, err := m.UserStore.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.FailAfterGetByID {
		return user, errors.New("simulated failure after successful GetByID")
	}
	return user, nil
}

func (m *MockFailingUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.FailOnGetByEmail {
		return nil, errors.New("simulated GetByEmail failure")
	}
	return m.UserStore.GetByEmail(ctx, email)
}

func (m *MockFailingUserStore) Update(ctx context.Context, user *domain.User) error {
	if m.FailOnUpdate {
		return errors.New("simulated update failure")
	}
	err := m.UserStore.Update(ctx, user)
	if err != nil {
		return err
	}
	if m.FailAfterUpdateOp {
		return errors.New("simulated failure after successful update")
	}
	return nil
}

func (m *MockFailingUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	if m.FailOnDelete {
		return errors.New("simulated delete failure")
	}
	return m.UserStore.Delete(ctx, id)
}

func (m *MockFailingUserStore) WithTx(tx *sql.Tx) store.UserStore {
	// Return a new instance with the transaction set
	return &MockFailingUserStore{
		UserStore:         m.UserStore.WithTx(tx),
		FailOnCreate:      m.FailOnCreate,
		FailOnUpdate:      m.FailOnUpdate,
		FailOnGetByID:     m.FailOnGetByID,
		FailOnGetByEmail:  m.FailOnGetByEmail,
		FailOnDelete:      m.FailOnDelete,
		FailAfterCreate:   m.FailAfterCreate,
		FailAfterGetByID:  m.FailAfterGetByID,
		FailAfterUpdateOp: m.FailAfterUpdateOp,
	}
}

// TestUserService_CreateUser_Atomicity tests that user creation is atomic
func TestUserService_CreateUser_Atomicity(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	testutils.WithTx(t, db, func(tx store.DBTX) {
		ctx := context.Background()
		logger := slog.Default()

		// Setup base user store with transaction
		userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for testing

		t.Run("Transaction_Rollback_On_Failure", func(t *testing.T) {
			// Create a failing repository that first creates the user but then fails
			failingStore := &MockFailingUserStore{
				UserStore:       userStore,
				FailAfterCreate: true, // Fail after successfully creating the user
			}

			// Create service with the failing store
			userService := service.NewUserService(failingStore, db, logger)

			// Attempt to create a user - this should fail after committing to DB but before committing the transaction
			email := "tx-rollback-test@example.com"
			password := "SecurePass123!"
			user, err := userService.CreateUser(ctx, email, password)

			// Verify the operation failed
			assert.Error(t, err, "Operation should fail")
			assert.Nil(t, user, "No user should be returned")

			// Verify the user was NOT actually persisted due to transaction rollback
			var count int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM users WHERE email = $1",
				email,
			).Scan(&count)
			require.NoError(t, err, "Failed to count users")
			assert.Equal(t, 0, count, "No user should exist in the database due to transaction rollback")
		})

		t.Run("Transaction_Commit_On_Success", func(t *testing.T) {
			// Create a succeeding store
			successStore := &MockFailingUserStore{
				UserStore: userStore,
			}

			// Create service with the succeeding store
			userService := service.NewUserService(successStore, db, logger)

			// Create a user - this should succeed
			email := "tx-commit-success@example.com"
			password := "SecurePass123!"
			user, err := userService.CreateUser(ctx, email, password)

			// Verify the operation succeeded
			assert.NoError(t, err, "Operation should succeed")
			assert.NotNil(t, user, "User should be returned")
			assert.Equal(t, email, user.Email, "User email should match")

			// Verify the user was actually persisted
			var count int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM users WHERE id = $1 AND email = $2",
				user.ID, email,
			).Scan(&count)
			require.NoError(t, err, "Failed to count users")
			assert.Equal(t, 1, count, "User should exist in the database")
		})
	})
}

// TestUserService_UpdateUserEmail_Atomicity tests the atomicity of email updates
func TestUserService_UpdateUserEmail_Atomicity(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	testutils.WithTx(t, db, func(tx store.DBTX) {
		ctx := context.Background()
		logger := slog.Default()

		// Setup base user store with transaction
		userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for testing

		// Create a test user directly
		initialEmail := "update-email-tx-test@example.com"
		userID := testutils.MustInsertUser(ctx, t, tx, initialEmail)

		t.Run("Transaction_Rollback_On_GetByID_Failure", func(t *testing.T) {
			// Create a failing store that fails on GetByID
			failingStore := &MockFailingUserStore{
				UserStore:     userStore,
				FailOnGetByID: true, // Fail during GetByID
			}

			// Create service with the failing store
			userService := service.NewUserService(failingStore, db, logger)

			// Attempt to update email - this should fail during GetByID
			newEmail := "new-email-getbyid-fail@example.com"
			err := userService.UpdateUserEmail(ctx, userID, newEmail)

			// Verify the operation failed
			assert.Error(t, err, "Operation should fail")
			assert.Contains(t, err.Error(), "simulated GetByID failure", "Error should be from our mock")

			// Verify the email was NOT changed due to transaction rollback
			var email string
			err = tx.QueryRowContext(ctx,
				"SELECT email FROM users WHERE id = $1",
				userID,
			).Scan(&email)
			require.NoError(t, err, "Failed to get user email")
			assert.Equal(t, initialEmail, email,
				"Email should remain unchanged due to transaction rollback")
		})

		t.Run("Transaction_Rollback_On_Update_Failure", func(t *testing.T) {
			// Create a failing store that fails on Update
			failingStore := &MockFailingUserStore{
				UserStore:    userStore,
				FailOnUpdate: true, // Fail during Update
			}

			// Create service with the failing store
			userService := service.NewUserService(failingStore, db, logger)

			// Attempt to update email - this should fail during Update
			newEmail := "new-email-update-fail@example.com"
			err := userService.UpdateUserEmail(ctx, userID, newEmail)

			// Verify the operation failed
			assert.Error(t, err, "Operation should fail")
			assert.Contains(t, err.Error(), "simulated update failure", "Error should be from our mock")

			// Verify the email was NOT changed due to transaction rollback
			var email string
			err = tx.QueryRowContext(ctx,
				"SELECT email FROM users WHERE id = $1",
				userID,
			).Scan(&email)
			require.NoError(t, err, "Failed to get user email")
			assert.Equal(t, initialEmail, email,
				"Email should remain unchanged due to transaction rollback")
		})

		t.Run("Transaction_Commit_On_Success", func(t *testing.T) {
			// Create a succeeding store
			successStore := &MockFailingUserStore{
				UserStore: userStore,
			}

			// Create service with the succeeding store
			userService := service.NewUserService(successStore, db, logger)

			// Update the email - this should succeed
			newEmail := "new-email-success@example.com"
			err := userService.UpdateUserEmail(ctx, userID, newEmail)

			// Verify the operation succeeded
			assert.NoError(t, err, "Operation should succeed")

			// Verify the email was actually updated
			var email string
			err = tx.QueryRowContext(ctx,
				"SELECT email FROM users WHERE id = $1",
				userID,
			).Scan(&email)
			require.NoError(t, err, "Failed to get user email")
			assert.Equal(t, newEmail, email, "Email should be updated to the new value")
		})
	})
}

// TestUserService_UpdateUserPassword_Atomicity tests the atomicity of password updates
func TestUserService_UpdateUserPassword_Atomicity(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	testutils.WithTx(t, db, func(tx store.DBTX) {
		ctx := context.Background()
		logger := slog.Default()

		// Setup base user store with transaction
		userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for testing

		// Create a test user directly
		email := "update-password-tx-test@example.com"
		userID := testutils.MustInsertUser(ctx, t, tx, email)

		// Get the initial hashed password to verify it doesn't change on rollback
		var initialHash string
		err = tx.QueryRowContext(ctx,
			"SELECT password_hash FROM users WHERE id = $1",
			userID,
		).Scan(&initialHash)
		require.NoError(t, err, "Failed to get initial password hash")

		t.Run("Transaction_Rollback_On_Failure", func(t *testing.T) {
			// Create a failing store that fails after Get but before Update
			failingStore := &MockFailingUserStore{
				UserStore:        userStore,
				FailAfterGetByID: true, // Fail after successful GetByID
			}

			// Create service with the failing store
			userService := service.NewUserService(failingStore, db, logger)

			// Attempt to update password - this should fail after Get but before Update
			newPassword := "NewSecurePass456!"
			err := userService.UpdateUserPassword(ctx, userID, newPassword)

			// Verify the operation failed
			assert.Error(t, err, "Operation should fail")
			assert.Contains(t, err.Error(), "simulated failure after successful GetByID",
				"Error should be from our mock")

			// Verify the password hash was NOT changed due to transaction rollback
			var currentHash string
			err = tx.QueryRowContext(ctx,
				"SELECT password_hash FROM users WHERE id = $1",
				userID,
			).Scan(&currentHash)
			require.NoError(t, err, "Failed to get current password hash")
			assert.Equal(t, initialHash, currentHash,
				"Password hash should remain unchanged due to transaction rollback")
		})

		t.Run("Transaction_Commit_On_Success", func(t *testing.T) {
			// Create a succeeding store
			successStore := &MockFailingUserStore{
				UserStore: userStore,
			}

			// Create service with the succeeding store
			userService := service.NewUserService(successStore, db, logger)

			// Update the password - this should succeed
			newPassword := "SuccessPassword789!"
			err := userService.UpdateUserPassword(ctx, userID, newPassword)

			// Verify the operation succeeded
			assert.NoError(t, err, "Operation should succeed")

			// Verify the password hash was actually updated (it should be different)
			var newHash string
			err = tx.QueryRowContext(ctx,
				"SELECT password_hash FROM users WHERE id = $1",
				userID,
			).Scan(&newHash)
			require.NoError(t, err, "Failed to get new password hash")
			assert.NotEqual(t, initialHash, newHash,
				"Password hash should be updated to a new value")
		})
	})
}

// TestUserService_DeleteUser_Atomicity tests the atomicity of user deletion
func TestUserService_DeleteUser_Atomicity(t *testing.T) {
	// Skip if not in integration test environment
	if !testutils.IsIntegrationTestEnvironment() {
		t.Skip("Skipping integration test - requires DATABASE_URL environment variable")
	}

	// Get a database connection
	db, err := testutils.GetTestDB()
	require.NoError(t, err, "Failed to connect to test database")
	defer testutils.AssertCloseNoError(t, db)

	testutils.WithTx(t, db, func(tx store.DBTX) {
		ctx := context.Background()
		logger := slog.Default()

		// Setup base user store with transaction
		userStore := postgres.NewPostgresUserStore(tx, 4) // Low cost for testing

		// Create a test user for the delete failure test
		emailFail := "delete-fail-tx-test@example.com"
		userIDFail := testutils.MustInsertUser(ctx, t, tx, emailFail)

		// Create a test user for the delete success test
		emailSuccess := "delete-success-tx-test@example.com"
		userIDSuccess := testutils.MustInsertUser(ctx, t, tx, emailSuccess)

		t.Run("Transaction_Rollback_On_Delete_Failure", func(t *testing.T) {
			// Create a failing store that fails on Delete
			failingStore := &MockFailingUserStore{
				UserStore:    userStore,
				FailOnDelete: true, // Fail during Delete
			}

			// Create service with the failing store
			userService := service.NewUserService(failingStore, db, logger)

			// Attempt to delete the user - this should fail
			err := userService.DeleteUser(ctx, userIDFail)

			// Verify the operation failed
			assert.Error(t, err, "Operation should fail")
			assert.Contains(t, err.Error(), "simulated delete failure", "Error should be from our mock")

			// Verify the user was NOT deleted due to transaction rollback
			var count int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM users WHERE id = $1",
				userIDFail,
			).Scan(&count)
			require.NoError(t, err, "Failed to count users")
			assert.Equal(t, 1, count, "User should still exist due to transaction rollback")
		})

		t.Run("Transaction_Commit_On_Success", func(t *testing.T) {
			// Create a succeeding store
			successStore := &MockFailingUserStore{
				UserStore: userStore,
			}

			// Create service with the succeeding store
			userService := service.NewUserService(successStore, db, logger)

			// Delete the user - this should succeed
			err := userService.DeleteUser(ctx, userIDSuccess)

			// Verify the operation succeeded
			assert.NoError(t, err, "Operation should succeed")

			// Verify the user was actually deleted
			var count int
			err = tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM users WHERE id = $1",
				userIDSuccess,
			).Scan(&count)
			require.NoError(t, err, "Failed to count users")
			assert.Equal(t, 0, count, "User should be deleted")
		})
	})
}
