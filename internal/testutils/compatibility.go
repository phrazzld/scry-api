//go:build compatibility

// This file provides a compatibility layer to ease migration to the new
// package structure. It should only be used during the migration period
// and will be removed once all tests are updated to use the new structure.

package testutils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/testutils/api"
	"github.com/phrazzld/scry-api/internal/testutils/db"
	"github.com/stretchr/testify/require"
)

//
// COMPATIBILITY LAYER
//
// These functions provide a compatibility layer to ease migration to the new
// package structure. They proxy calls to the appropriate sub-packages.
// This allows for gradual migration of tests to use the new package structure
// without breaking existing tests.
//

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

// SetupTestDatabaseSchema initializes the database schema using project migrations.
// Compatibility function that delegates to db.SetupTestDatabaseSchema.
func SetupTestDatabaseSchema(dbConn *sql.DB) error {
	return db.SetupTestDatabaseSchema(dbConn)
}

// WithTx runs a test function with transaction-based isolation.
// Compatibility function that delegates to db.WithTx.
func WithTx(t *testing.T, dbConn *sql.DB, fn func(t *testing.T, tx *sql.Tx)) {
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
