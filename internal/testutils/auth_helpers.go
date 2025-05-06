//go:build integration || test_without_external_deps

// This file provides test utilities for authentication.
// It should be used in preference to the compatibility.go file where possible.

package testutils

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestUserAuth contains authentication information for a test user.
type TestUserAuth struct {
	UserID    uuid.UUID
	Email     string
	Password  string
	AuthToken string
}

// CreateTestUserWithTx creates a test user in the database within the given transaction
// and returns the created user's ID, email, and password.
func CreateTestUserWithTx(t *testing.T, tx *sql.Tx) *TestUserAuth {
	t.Helper()

	// Create a user store with the transaction
	userStore := postgres.NewPostgresUserStore(tx, bcrypt.MinCost)

	// Generate a unique email for this test to avoid conflicts
	userEmail := "test_" + uuid.New().String() + "@example.com"
	password := "password123456"

	// Create a new user domain object
	user, err := domain.NewUser(userEmail, password)
	require.NoError(t, err, "Failed to create user domain object")

	// Save the user to the database
	err = userStore.Create(context.Background(), user)
	require.NoError(t, err, "Failed to create test user")

	// Generate an auth token for the user
	authToken := auth.GenerateAuthHeaderForTestingT(t, user.ID)

	return &TestUserAuth{
		UserID:    user.ID,
		Email:     userEmail,
		Password:  password,
		AuthToken: authToken,
	}
}

// GenerateAuthHeaderForUser creates an Authorization header with a valid JWT token for a user.
func GenerateAuthHeaderForUser(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	return auth.GenerateAuthHeaderForTestingT(t, userID)
}

// WithAuthenticatedUser runs a test function with a transaction and an authenticated test user.
// It creates a test user in the transaction and passes the auth context to the test function.
//
// NOTE: This function is currently unused and is preserved for compatibility.
// Uses the testdb.WithTx function for transaction management.
// nolint:unused
func WithAuthenticatedUser(
	t *testing.T,
	db *sql.DB,
	fn func(t *testing.T, tx *sql.Tx, auth *TestUserAuth),
) {
	t.Helper()

	// Comment out for now since WithTx is undefined
	// When needed, this function can be restored with proper implementation
	/*
		// Start a transaction for test isolation
		WithTx(t, db, func(t *testing.T, tx *sql.Tx) {
			// Create a test user with authentication
			auth := CreateTestUserWithTx(t, tx)

			// Run the test function with the transaction and auth context
			fn(t, tx, auth)
		})
	*/
}

// AuthenticateRequest adds authorization headers to an HTTP request.
func AuthenticateRequest(req *http.Request, authToken string) {
	req.Header.Set("Authorization", authToken)
}

// CreateTestUserWithAuth creates a test user with the given options and returns the created TestUserAuth.
// This function creates a user in the database and generates an auth token for it.
func CreateTestUserWithAuth(
	t *testing.T,
	tx *sql.Tx,
	email string,
	password string,
) *TestUserAuth {
	t.Helper()

	// Create a user store with the transaction
	userStore := postgres.NewPostgresUserStore(tx, bcrypt.MinCost)

	// Use provided email or generate a unique one
	if email == "" {
		email = "test_" + uuid.New().String() + "@example.com"
	}

	// Use provided password or default
	if password == "" {
		password = "password123456"
	}

	// Create a new user domain object
	user, err := domain.NewUser(email, password)
	require.NoError(t, err, "Failed to create user domain object")

	// Override createdAt for deterministic testing if needed
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	// Save the user to the database
	err = userStore.Create(context.Background(), user)
	require.NoError(t, err, "Failed to create test user")

	// Generate an auth token for the user
	authToken := auth.GenerateAuthHeaderForTestingT(t, user.ID)

	return &TestUserAuth{
		UserID:    user.ID,
		Email:     email,
		Password:  password,
		AuthToken: authToken,
	}
}
