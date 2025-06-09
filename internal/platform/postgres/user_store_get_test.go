//go:build integration

package postgres_test

// This file contains tests for the GetByID and GetByEmail methods of PostgresUserStore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestPostgresUserStore_GetByID tests the GetByID method
func TestPostgresUserStore_GetByID(t *testing.T) {
	t.Parallel() // Enable parallel testing

	db := testdb.GetTestDBWithT(t)
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create a new user store
		userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)

		// Test Case 1: Successful retrieval of an existing user
		t.Run("Successful retrieval", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Insert a test user
			email := fmt.Sprintf("get-by-id-%s@example.com", uuid.New().String()[:8])
			userID := insertTestUser(ctx, t, tx, email)

			// Call the GetByID method
			user, err := userStore.GetByID(ctx, userID)

			// Verify the result
			require.NoError(t, err, "User retrieval should succeed")
			require.NotNil(t, user, "User should not be nil")
			assert.Equal(t, userID, user.ID, "User ID should match")
			assert.Equal(t, email, user.Email, "User email should match")
			assert.NotEmpty(t, user.HashedPassword, "Hashed password should not be empty")
			assert.Empty(t, user.Password, "Password should be empty")
			assert.False(t, user.CreatedAt.IsZero(), "CreatedAt should not be zero")
			assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt should not be zero")
		})

		// Test Case 2: Attempt to retrieve a non-existent user
		t.Run("Non-existent user", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Generate a random user ID that doesn't exist
			nonExistentID := uuid.New()

			// Call the GetByID method
			user, err := userStore.GetByID(ctx, nonExistentID)

			// Verify the result
			assert.ErrorIs(t, err, store.ErrUserNotFound, "Should return ErrUserNotFound")
			assert.Nil(t, user, "User should be nil for non-existent ID")
		})
	})
}

// TestPostgresUserStore_GetByEmail tests the GetByEmail method
func TestPostgresUserStore_GetByEmail(t *testing.T) {
	t.Parallel() // Enable parallel testing

	db := testdb.GetTestDBWithT(t)
	testdb.WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
		// Create a new user store
		userStore := postgres.NewPostgresUserStore(tx, bcrypt.DefaultCost)

		// Test Case 1: Successful retrieval of an existing user
		t.Run("Successful retrieval", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Insert a test user
			email := fmt.Sprintf("get-by-email-%s@example.com", uuid.New().String()[:8])
			userID := insertTestUser(ctx, t, tx, email)

			// Call the GetByEmail method
			user, err := userStore.GetByEmail(ctx, email)

			// Verify the result
			require.NoError(t, err, "User retrieval should succeed")
			require.NotNil(t, user, "User should not be nil")
			assert.Equal(t, userID, user.ID, "User ID should match")
			assert.Equal(t, email, user.Email, "User email should match")
			assert.NotEmpty(t, user.HashedPassword, "Hashed password should not be empty")
			assert.Empty(t, user.Password, "Password should be empty")
		})

		// Test Case 2: Case-insensitive email matching
		t.Run("Case-insensitive matching", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Insert a test user with mixed-case email
			email := fmt.Sprintf("Case-Sensitive-%s@Example.com", uuid.New().String()[:8])
			userID := insertTestUser(ctx, t, tx, email)

			// Call GetByEmail with lowercase email
			lowerCaseEmail := strings.ToLower(email)
			user, err := userStore.GetByEmail(ctx, lowerCaseEmail)

			// Verify the result
			require.NoError(t, err, "User retrieval should succeed with case-insensitive matching")
			require.NotNil(t, user, "User should not be nil")
			assert.Equal(t, userID, user.ID, "User ID should match")
			assert.Equal(t, email, user.Email, "User email should match the original case")

			// Call GetByEmail with uppercase email
			upperCaseEmail := strings.ToUpper(email)
			user, err = userStore.GetByEmail(ctx, upperCaseEmail)

			// Verify the result
			require.NoError(t, err, "User retrieval should succeed with case-insensitive matching")
			require.NotNil(t, user, "User should not be nil")
			assert.Equal(t, userID, user.ID, "User ID should match")
		})

		// Test Case 3: Attempt to retrieve a non-existent user
		t.Run("Non-existent user", func(t *testing.T) {
			t.Parallel() // Enable parallel subtests

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Generate a random email that doesn't exist
			nonExistentEmail := fmt.Sprintf("non-existent-%s@example.com", uuid.New().String())

			// Call the GetByEmail method
			user, err := userStore.GetByEmail(ctx, nonExistentEmail)

			// Verify the result
			assert.ErrorIs(t, err, store.ErrUserNotFound, "Should return ErrUserNotFound")
			assert.Nil(t, user, "User should be nil for non-existent email")
		})
	})
}
