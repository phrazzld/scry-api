//go:build integration_compat

// Package testutils provides compatibility utilities during the migration to the new testdb structure.
// This file is the main entry point for backwards compatibility functions, but is currently disabled with
// the integration_compat build tag to prevent function redeclarations.

package testutils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/testdb"
	"github.com/phrazzld/scry-api/internal/testutils/api"
	"github.com/phrazzld/scry-api/internal/testutils/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

//
// COMPATIBILITY LAYER
//
// These functions provide a compatibility layer to ease migration to the new
// package structure. They proxy calls to the appropriate sub-packages.
// This allows for gradual migration of tests to use the new package structure
// without breaking existing tests.
//

// IsIntegrationTestEnvironment returns true if the environment is configured
// for running integration tests with a database connection.
// Integration tests should check this and skip if not in an integration test environment.
func IsIntegrationTestEnvironment() bool {
	return testdb.IsIntegrationTestEnvironment()
}

// AssertCloseNoError ensures that the Close() method on the provided closer
// executes without error. It uses assert.NoError to allow subsequent defers
// to run even if this one fails (as opposed to using require.NoError which
// would abort the test immediately).
func AssertCloseNoError(t *testing.T, closer interface{}) {
	t.Helper()

	if closer == nil {
		return
	}

	if db, ok := closer.(*sql.DB); ok {
		testdb.CleanupDB(t, db)
		return
	}

	if c, ok := closer.(io.Closer); ok {
		err := c.Close()
		assert.NoError(t, err, "Deferred Close() failed for %T", closer)
	}
}

// MustInsertUser inserts a user into the database for testing.
// This is the old version that takes a tx parameter.
// It requires a transaction obtained from WithTx to ensure test isolation.
// The function will fail the test if the insert operation fails.
func MustInsertUser(
	ctx context.Context,
	t *testing.T,
	tx *sql.Tx,
	email string,
	bcryptCost ...int,
) uuid.UUID {
	t.Helper()

	// Default bcrypt cost if not provided
	cost := 10
	if len(bcryptCost) > 0 {
		cost = bcryptCost[0]
	}

	// Generate a random UUID for the user
	userID := uuid.New()

	// Hash a default password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpassword123456"), cost)
	require.NoError(t, err, "Failed to hash password")

	// Insert the user directly using SQL
	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO users (id, email, hashed_password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)",
		userID,
		email,
		string(hashedPassword),
		time.Now().UTC(),
		time.Now().UTC(),
	)
	require.NoError(t, err, "Failed to insert test user")

	return userID
}

// MustGetTestDatabaseURL returns the database URL for tests
// This implementation is for backward compatibility
func MustGetTestDatabaseURL() string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// ALLOW-PANIC
		panic("DATABASE_URL environment variable is required for integration tests")
	}
	return dbURL
}

// CreateTestUser creates a new valid user with a random email for testing.
// It does not save the user to the database.
func CreateTestUser(t *testing.T) *domain.User {
	t.Helper()
	email := fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
	user, err := domain.NewUser(email, "TestPassword123456!")
	require.NoError(t, err, "Failed to create test user")
	return user
}

// GetUserByID retrieves a user from the database by ID.
// Returns nil if the user does not exist.
func GetUserByID(ctx context.Context, t *testing.T, db store.DBTX, id uuid.UUID) *domain.User {
	t.Helper()

	// Query for the user
	var user domain.User
	err := db.QueryRowContext(ctx, `
		SELECT id, email, hashed_password, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Email,
		&user.HashedPassword,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		require.NoError(t, err, "Failed to query user by ID")
	}

	return &user
}

// CountUsers counts the number of users in the database matching the given criteria.
func CountUsers(
	ctx context.Context,
	t *testing.T,
	db store.DBTX,
	whereClause string,
	args ...interface{},
) int {
	t.Helper()

	query := "SELECT COUNT(*) FROM users"
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int
	err := db.QueryRowContext(ctx, query, args...).Scan(&count)
	require.NoError(t, err, "Failed to count users")

	return count
}

// ResetTestData truncates all tables in the test database.
// CAUTION: This is potentially destructive and should only be used in test environments.
// This is provided for backward compatibility but NOT RECOMMENDED - use transaction isolation instead.
func ResetTestData(db *sql.DB) error {
	_, err := db.Exec(`
		TRUNCATE TABLE cards, memos, user_card_stats, users, tasks
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		return fmt.Errorf("failed to truncate test tables: %w", err)
	}
	return nil
}

// SetupTestDatabaseSchema runs database migrations to set up the test database.
// This maintains the original function signature for backward compatibility.
func SetupTestDatabaseSchema(db *sql.DB) error {
	// We need an implementation that doesn't use testing.T features
	// Get project root - similar to the implementation in testdb
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find go.mod by traversing up directories
	projectRoot := ""
	for {
		if _, err := os.Stat(fmt.Sprintf("%s/go.mod", dir)); err == nil {
			projectRoot = dir
			break
		}

		parent := fmt.Sprintf("%s/..", dir)
		if parent == dir {
			return fmt.Errorf("could not find project root (go.mod file)")
		}
		dir = parent
	}

	// Set up migrations directory
	migrationsDir := fmt.Sprintf("%s/internal/platform/postgres/migrations", projectRoot)
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at %s: %w", migrationsDir, err)
	}

	// Run migrations
	if err := testdb.ApplyMigrations(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// WithTx executes a test function within a transaction, automatically rolling back
// after the test completes. This is a compatibility function that forwards to testdb.WithTx.
func WithTx(t *testing.T, db *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	testdb.WithTx(t, db, fn)
}

// GetTestDBWithT returns a database connection for testing.
// Compatibility function that delegates to db.GetTestDBWithT.
func GetTestDBWithT(t *testing.T) *sql.DB {
	t.Helper()
	return db.GetTestDBWithT(t)
}

// GetTestDB is the original version that returns an error rather than using t.Helper
// Compatibility function that delegates to db.GetTestDB.
func GetTestDB() (*sql.DB, error) {
	return db.GetTestDB()
}

// SetupTestDatabaseSchemaDB initializes the database schema using project migrations.
// Compatibility function that delegates to db.SetupTestDatabaseSchema.
func SetupTestDatabaseSchemaDB(dbConn *sql.DB) error {
	return db.SetupTestDatabaseSchema(dbConn)
}

// WithTxDB runs a test function with transaction-based isolation.
// Compatibility function that delegates to db.WithTx.
func WithTxDB(t *testing.T, dbConn *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
	t.Helper()
	db.WithTx(t, dbConn, fn)
}

// CleanupDB properly closes a database connection and logs any errors.
// Compatibility function that delegates to db.CleanupDB.
func CleanupDB(t *testing.T, dbConn *sql.DB) {
	t.Helper()
	db.CleanupDB(t, dbConn)
}

// ResetTestData truncates all test tables to ensure test isolation.
// Compatibility function that delegates to db.ResetTestData.
func ResetTestData(dbConn *sql.DB) error {
	return db.ResetTestData(dbConn)
}

// AssertRollbackNoError attempts to roll back a transaction and logs an error if it fails.
// Compatibility function that delegates to db.AssertRollbackNoError.
func AssertRollbackNoError(t *testing.T, tx *sql.Tx) {
	t.Helper()
	db.AssertRollbackNoError(t, tx)
}

// CreateTestUser creates a test user in the database within the given transaction
// Compatibility function that delegates to api.CreateTestUser.
func CreateTestUser(t *testing.T, tx *sql.Tx) uuid.UUID {
	t.Helper()
	return api.CreateTestUser(t, tx)
}

// CreateTestCard creates a test card in the database within the given transaction
// Compatibility function that delegates to api.CreateTestCard.
func CreateTestCard(t *testing.T, tx *sql.Tx, userID uuid.UUID) *domain.Card {
	t.Helper()
	return api.CreateTestCard(t, tx, userID)
}

// GetCardByID retrieves a card by its ID from the database within the given transaction.
// Compatibility function that delegates to api.GetCardByID.
func GetCardByID(tx *sql.Tx, cardID uuid.UUID) (*domain.Card, error) {
	return api.GetCardByID(tx, cardID)
}

// GetAuthToken generates an authentication token for testing.
// Compatibility function that delegates to api.GetAuthToken.
func GetAuthToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	return api.GetAuthToken(t, userID)
}

// GetUserCardStats retrieves user card statistics for a given card and user.
// Compatibility function that delegates to api.GetUserCardStats.
func GetUserCardStats(t *testing.T, tx *sql.Tx, userID, cardID uuid.UUID) *domain.UserCardStats {
	t.Helper()
	return api.GetUserCardStats(t, tx, userID, cardID)
}

// RunInTransaction is an alias for WithTx to maintain compatibility with existing code.
// This is used in card_service_tx_test.go
func RunInTransaction(t *testing.T, db *sql.DB, ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	t.Helper()

	// Begin a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Make sure the transaction is rolled back when done
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	// Run the function with the transaction
	if err := fn(ctx, tx); err != nil {
		return err
	}

	// If we got here, no error occurred in the function, so commit
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateTestJWTService creates a real JWT service for testing with a pre-configured secret and expiration.
func CreateTestJWTService() (auth.JWTService, error) {
	// Create minimal auth config with values valid for testing
	authConfig := config.AuthConfig{
		JWTSecret:                   "test-jwt-secret-that-is-32-chars-long", // At least 32 chars
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}

	return auth.NewJWTService(authConfig)
}

// GenerateAuthHeader creates an Authorization header value with a valid JWT token for testing.
func GenerateAuthHeader(userID uuid.UUID) (string, error) {
	jwtService, err := CreateTestJWTService()
	if err != nil {
		return "", fmt.Errorf("failed to create test JWT service: %w", err)
	}

	token, err := jwtService.GenerateToken(context.Background(), userID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return "Bearer " + token, nil
}

// CardOption is a function that configures a Card for testing.
type CardOption func(*domain.Card)

// WithCardID sets a specific ID for the test card.
func WithCardID(id uuid.UUID) CardOption {
	return func(c *domain.Card) {
		c.ID = id
	}
}

// WithCardUserID sets a specific user ID for the test card.
func WithCardUserID(userID uuid.UUID) CardOption {
	return func(c *domain.Card) {
		c.UserID = userID
	}
}

// WithCardMemoID sets a specific memo ID for the test card.
func WithCardMemoID(memoID uuid.UUID) CardOption {
	return func(c *domain.Card) {
		c.MemoID = memoID
	}
}

// WithCardContent sets the content for the test card using a map.
// The map will be marshaled to JSON.
func WithCardContent(content map[string]interface{}) CardOption {
	return func(c *domain.Card) {
		contentBytes, _ := json.Marshal(content)
		c.Content = contentBytes
	}
}

// WithRawCardContent sets raw JSON content for the test card.
// This allows direct setting of pre-marshaled JSON.
func WithRawCardContent(content json.RawMessage) CardOption {
	return func(c *domain.Card) {
		c.Content = content
	}
}

// WithCardCreatedAt sets the creation timestamp for the test card.
func WithCardCreatedAt(createdAt time.Time) CardOption {
	return func(c *domain.Card) {
		c.CreatedAt = createdAt
	}
}

// WithCardUpdatedAt sets the update timestamp for the test card.
func WithCardUpdatedAt(updatedAt time.Time) CardOption {
	return func(c *domain.Card) {
		c.UpdatedAt = updatedAt
	}
}

// CreateCardForAPITest creates a Card instance for API testing with default values.
// Options can be passed to customize the card.
// If t is nil, error checking for JSON marshaling is skipped.
func CreateCardForAPITest(t *testing.T, opts ...CardOption) *domain.Card {
	if t != nil {
		t.Helper()
	}

	now := time.Now().UTC()
	userID := uuid.New()
	memoID := uuid.New()
	cardID := uuid.New()

	// Default test card content
	defaultContent := map[string]interface{}{
		"front": "What is the capital of France?",
		"back":  "Paris",
	}
	contentBytes, err := json.Marshal(defaultContent)
	if t != nil {
		require.NoError(t, err)
	}

	// Create card with default values
	card := &domain.Card{
		ID:        cardID,
		UserID:    userID,
		MemoID:    memoID,
		Content:   contentBytes,
		CreatedAt: now.Add(-24 * time.Hour),
		UpdatedAt: now.Add(-24 * time.Hour),
	}

	// Apply options
	for _, opt := range opts {
		opt(card)
	}

	return card
}

// StatsOption is a function that configures UserCardStats for testing.
type StatsOption func(*domain.UserCardStats)

// WithStatsUserID sets a specific user ID for the test stats.
func WithStatsUserID(userID uuid.UUID) StatsOption {
	return func(s *domain.UserCardStats) {
		s.UserID = userID
	}
}

// WithStatsCardID sets a specific card ID for the test stats.
func WithStatsCardID(cardID uuid.UUID) StatsOption {
	return func(s *domain.UserCardStats) {
		s.CardID = cardID
	}
}

// WithStatsInterval sets the interval for the test stats.
func WithStatsInterval(interval int) StatsOption {
	return func(s *domain.UserCardStats) {
		s.Interval = interval
	}
}

// WithStatsEaseFactor sets the ease factor for the test stats.
func WithStatsEaseFactor(easeFactor float64) StatsOption {
	return func(s *domain.UserCardStats) {
		s.EaseFactor = easeFactor
	}
}

// WithStatsConsecutiveCorrect sets the consecutive correct count for the test stats.
func WithStatsConsecutiveCorrect(count int) StatsOption {
	return func(s *domain.UserCardStats) {
		s.ConsecutiveCorrect = count
	}
}

// WithStatsLastReviewedAt sets the last reviewed timestamp for the test stats.
func WithStatsLastReviewedAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.LastReviewedAt = timestamp
	}
}

// WithStatsNextReviewAt sets the next review timestamp for the test stats.
func WithStatsNextReviewAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.NextReviewAt = timestamp
	}
}

// WithStatsReviewCount sets the review count for the test stats.
func WithStatsReviewCount(count int) StatsOption {
	return func(s *domain.UserCardStats) {
		s.ReviewCount = count
	}
}

// WithStatsCreatedAt sets the creation timestamp for the test stats.
func WithStatsCreatedAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.CreatedAt = timestamp
	}
}

// WithStatsUpdatedAt sets the update timestamp for the test stats.
func WithStatsUpdatedAt(timestamp time.Time) StatsOption {
	return func(s *domain.UserCardStats) {
		s.UpdatedAt = timestamp
	}
}

// CreateStatsForAPITest creates a UserCardStats instance for API testing with default values.
// Options can be passed to customize the stats.
// If t is nil, no test helper functionality is used.
func CreateStatsForAPITest(t *testing.T, opts ...StatsOption) *domain.UserCardStats {
	if t != nil {
		t.Helper()
	}

	now := time.Now().UTC()
	userID := uuid.New()
	cardID := uuid.New()

	// Create stats with default values
	stats := &domain.UserCardStats{
		UserID:             userID,
		CardID:             cardID,
		Interval:           1,
		EaseFactor:         2.5,
		ConsecutiveCorrect: 1,
		LastReviewedAt:     now,
		NextReviewAt:       now.Add(24 * time.Hour),
		ReviewCount:        1,
		CreatedAt:          now.Add(-24 * time.Hour),
		UpdatedAt:          now,
	}

	// Apply options
	for _, opt := range opts {
		opt(stats)
	}

	return stats
}

// SetupCardReviewTestServerWithNextCard is a compatibility helper that delegates to api.SetupCardReviewTestServerWithNextCard
func SetupCardReviewTestServerWithNextCard(t *testing.T, userID uuid.UUID, card *domain.Card) *httptest.Server {
	t.Helper()
	return api.SetupCardReviewTestServerWithNextCard(t, userID, card)
}

// SetupCardReviewTestServerWithError is a compatibility helper that delegates to api.SetupCardReviewTestServerWithError
func SetupCardReviewTestServerWithError(t *testing.T, userID uuid.UUID, err error) *httptest.Server {
	t.Helper()
	return api.SetupCardReviewTestServerWithError(t, userID, err)
}

// SetupCardReviewTestServerWithAuthError is a compatibility helper that delegates to api.SetupCardReviewTestServerWithAuthError
func SetupCardReviewTestServerWithAuthError(t *testing.T, userID uuid.UUID, err error) *httptest.Server {
	t.Helper()
	return api.SetupCardReviewTestServerWithAuthError(t, userID, err)
}

// SetupCardReviewTestServerWithUpdatedStats is a compatibility helper that delegates to api.SetupCardReviewTestServerWithUpdatedStats
func SetupCardReviewTestServerWithUpdatedStats(
	t *testing.T,
	userID uuid.UUID,
	stats *domain.UserCardStats,
) *httptest.Server {
	t.Helper()
	return api.SetupCardReviewTestServerWithUpdatedStats(t, userID, stats)
}

// AssertErrorResponse is a compatibility helper that delegates to api.AssertErrorResponse
func AssertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedError string) {
	t.Helper()
	api.AssertErrorResponse(t, resp, expectedStatus, expectedError)
}

// AssertValidationError is a compatibility helper that delegates to api.AssertValidationError
func AssertValidationError(t *testing.T, resp *http.Response, field string, msgPart string) {
	t.Helper()
	api.AssertValidationError(t, resp, field, msgPart)
}

// GenerateRefreshTokenWithExpiry generates a refresh token with a custom expiration time.
// This is useful for testing token expiration scenarios.
func GenerateRefreshTokenWithExpiry(t *testing.T, userID uuid.UUID, expiry time.Time) (string, error) {
	t.Helper()

	jwtService, err := CreateTestJWTService()
	if err != nil {
		return "", fmt.Errorf("failed to create test JWT service: %w", err)
	}

	token, err := jwtService.GenerateRefreshTokenWithExpiry(context.Background(), userID, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token with custom expiry: %w", err)
	}

	return token, nil
}
