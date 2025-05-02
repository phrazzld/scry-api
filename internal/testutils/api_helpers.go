//go:build !compatibility && ignore_redeclarations

// This file provides test utilities for APIs.
// It should be used in preference to the compatibility.go file where possible.

package testutils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	"github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/domain/srs"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/service/card_review"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/require"
)

// CreateAuthComponents creates JWT service and password verifier for testing with a pre-configured
// test JWT secret and standard expiration settings.
//
// It returns:
//   - A valid auth configuration with test secrets
//   - A initialized JWT service for creating and validating tokens
//   - A password verifier implementation for testing
//   - Any error encountered during creation
//
// Example:
//
//	authConfig, jwtService, passwordVerifier, err := testutils.CreateAuthComponents(t)
//	require.NoError(t, err)
//	// Use the components in your test
func CreateAuthComponents(
	t *testing.T,
) (config.AuthConfig, auth.JWTService, auth.PasswordVerifier, error) {
	t.Helper()
	authConfig := config.AuthConfig{
		JWTSecret:                   "test-jwt-secret-thatis32characterslong",
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}
	jwtService, err := auth.NewJWTService(authConfig)
	if err != nil {
		return authConfig, nil, nil, err
	}
	passwordVerifier := auth.NewBcryptVerifier()
	return authConfig, jwtService, passwordVerifier, nil
}

// CreateAPIHandlers creates API handlers and middleware for testing purposes.
//
// It initializes:
//   - An auth handler for authentication endpoints
//   - A memo handler for memo-related endpoints
//   - An auth middleware for route protection
//
// The function takes all required dependencies and configures them for testing.
// It uses the default logger (slog.Default()) for simplicity in tests.
//
// Example:
//
//	// Set up stores and services
//	userStore := testutils.CreatePostgresUserStore(db)
//	authConfig, jwtService, passwordVerifier, _ := testutils.CreateAuthComponents(t)
//	memoService := service.NewMemoService(...)
//
//	// Create handlers
//	authHandler, memoHandler, authMiddleware := testutils.CreateAPIHandlers(
//	    t, userStore, jwtService, passwordVerifier, authConfig, memoService)
func CreateAPIHandlers(
	t *testing.T,
	userStore store.UserStore,
	jwtService auth.JWTService,
	passwordVerifier auth.PasswordVerifier,
	authConfig config.AuthConfig,
	memoService service.MemoService,
) (*api.AuthHandler, *api.MemoHandler, *middleware.AuthMiddleware) {
	t.Helper()
	// Use default logger for tests
	logger := slog.Default()
	authHandler := api.NewAuthHandler(userStore, jwtService, passwordVerifier, &authConfig, logger)
	memoHandler := api.NewMemoHandler(memoService, logger)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)
	return authHandler, memoHandler, authMiddleware
}

// CreateTestRouter creates a Chi router with standard middleware applied for testing.
//
// This function provides a consistent router setup for all API tests, applying the
// following middleware:
//   - RequestID: Assigns a unique request ID to each request
//   - RealIP: Sets the request's RemoteAddr to either X-Forwarded-For or X-Real-IP
//   - Recoverer: Recovers from panics and logs them appropriately
//
// Example:
//
//	router := testutils.CreateTestRouter(t)
//	// Add routes to the router
//	router.Get("/health", healthHandler)
func CreateTestRouter(t *testing.T) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()

	// Apply standard middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	return r
}

// SetupAuthRoutes configures standard auth routes on the provided router.
//
// This function sets up a standard API route structure for testing, including:
//   - Public routes: /api/auth/register and /api/auth/login
//   - Protected routes: /api/memos (requires authentication)
//
// The function ensures consistency in how auth routes are set up across tests
// and simplifies test setup for auth-related functionality.
//
// Example:
//
//	router := testutils.CreateTestRouter(t)
//	authHandler, memoHandler, authMiddleware := testutils.CreateAPIHandlers(...)
//	testutils.SetupAuthRoutes(t, router, authHandler, memoHandler, authMiddleware)
//
//	// Test the routes
//	server := httptest.NewServer(router)
//	defer server.Close()
func SetupAuthRoutes(
	t *testing.T,
	r chi.Router,
	authHandler *api.AuthHandler,
	memoHandler *api.MemoHandler,
	authMiddleware *middleware.AuthMiddleware,
) {
	t.Helper()
	r.Route("/api", func(r chi.Router) {
		// Auth endpoints
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			// Memo endpoints
			r.Post("/memos", memoHandler.CreateMemo)
		})
	})
}

// CreatePostgresUserStore creates a PostgresUserStore with default test settings.
//
// This convenience function initializes a user store with BCrypt cost set to 10
// for faster password hashing during tests. This is lower than production would use
// but significantly speeds up tests that involve user creation or authentication.
//
// Example:
//
//	db := testutils.GetTestDBWithT(t)
//	userStore := testutils.CreatePostgresUserStore(db)
//	// Use userStore in tests
func CreatePostgresUserStore(db store.DBTX) *postgres.PostgresUserStore {
	return postgres.NewPostgresUserStore(db, 10) // BCrypt cost = 10 for faster tests
}

//------------------------------------------------------------------------------
// Card Test Option Types - Used by both API and Domain tests
//------------------------------------------------------------------------------

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

// MustCreateCardForTest creates a Card instance for testing or fails the test if there's an error.
// This is a convenience wrapper around CreateCardForAPITest.
func MustCreateCardForTest(t *testing.T, opts ...CardOption) *domain.Card {
	t.Helper()
	card := CreateCardForAPITest(t, opts...)
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

// MustCreateStatsForTest creates a UserCardStats instance for testing or fails the test if there's an error.
// This is a convenience wrapper around CreateStatsForAPITest.
func MustCreateStatsForTest(t *testing.T, opts ...StatsOption) *domain.UserCardStats {
	t.Helper()
	stats := CreateStatsForAPITest(t, opts...)
	return stats
}

//------------------------------------------------------------------------------
// Memo Test Options
//------------------------------------------------------------------------------

// MemoOption is a function that configures a Memo for testing.
type MemoOption func(*domain.Memo)

// WithMemoID sets a specific ID for the test memo.
func WithMemoID(id uuid.UUID) MemoOption {
	return func(m *domain.Memo) {
		m.ID = id
	}
}

// WithMemoUserID sets a specific user ID for the test memo.
func WithMemoUserID(userID uuid.UUID) MemoOption {
	return func(m *domain.Memo) {
		m.UserID = userID
	}
}

// WithMemoText sets the text content for the test memo.
func WithMemoText(text string) MemoOption {
	return func(m *domain.Memo) {
		m.Text = text
	}
}

// WithMemoCreatedAt sets the creation timestamp for the test memo.
func WithMemoCreatedAt(createdAt time.Time) MemoOption {
	return func(m *domain.Memo) {
		m.CreatedAt = createdAt
	}
}

// WithMemoUpdatedAt sets the update timestamp for the test memo.
func WithMemoUpdatedAt(updatedAt time.Time) MemoOption {
	return func(m *domain.Memo) {
		m.UpdatedAt = updatedAt
	}
}

// CreateMemoForTest creates a Memo instance for testing with default values.
// Options can be passed to customize the memo.
// If t is nil, error checking is skipped.
func CreateMemoForTest(t *testing.T, opts ...MemoOption) *domain.Memo {
	if t != nil {
		t.Helper()
	}

	now := time.Now().UTC()
	userID := uuid.New()

	// Create memo with default values
	memo := &domain.Memo{
		ID:        uuid.New(),
		UserID:    userID,
		Text:      fmt.Sprintf("Test memo content %s", uuid.New().String()[:8]),
		Status:    domain.MemoStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Apply options
	for _, opt := range opts {
		opt(memo)
	}

	return memo
}

// MustCreateMemoForTest creates a Memo instance for testing or fails the test if there's an error.
// This is a convenience wrapper around CreateMemoForTest.
func MustCreateMemoForTest(t *testing.T, opts ...MemoOption) *domain.Memo {
	t.Helper()
	memo := CreateMemoForTest(t, opts...)
	return memo
}

// Create additional test utility functions for card management API tests

// CreatePostgresCardStore creates a PostgresCardStore with a logger for testing
func CreatePostgresCardStore(db store.DBTX) *postgres.PostgresCardStore {
	return postgres.NewPostgresCardStore(db, slog.Default())
}

// CreatePostgresUserCardStatsStore creates a PostgresUserCardStatsStore with a logger for testing
func CreatePostgresUserCardStatsStore(db store.DBTX) *postgres.PostgresUserCardStatsStore {
	return postgres.NewPostgresUserCardStatsStore(db, slog.Default())
}

// CreateSRSService creates an SRS service with default parameters for testing
func CreateSRSService() (srs.Service, error) {
	return srs.NewDefaultService()
}

// CreateCardRepositoryAdapter creates a card repository adapter for testing
func CreateCardRepositoryAdapter(cardStore store.CardStore, db store.DBTX) service.CardRepository {
	sqlDB, ok := db.(*sql.DB)
	if !ok {
		// If it's not a *sql.DB, it must be our TxDB wrapper with a transaction
		// For testing, just pass nil as the DB since we're using transactions
		return service.NewCardRepositoryAdapter(cardStore, nil)
	}
	return service.NewCardRepositoryAdapter(cardStore, sqlDB)
}

// CreateStatsRepositoryAdapter creates a stats repository adapter for testing
func CreateStatsRepositoryAdapter(statsStore store.UserCardStatsStore) service.StatsRepository {
	return service.NewStatsRepositoryAdapter(statsStore)
}

// CreateCardService creates a card service for testing
func CreateCardService(
	cardRepo service.CardRepository,
	statsRepo service.StatsRepository,
	srsService srs.Service,
) (service.CardService, error) {
	return service.NewCardService(cardRepo, statsRepo, srsService, slog.Default())
}

// CreateCardReviewService creates a card review service for testing
func CreateCardReviewService(
	cardStore store.CardStore,
	statsStore store.UserCardStatsStore,
	srsService srs.Service,
) (card_review.CardReviewService, error) {
	return card_review.NewCardReviewService(
		cardStore,
		statsStore,
		srsService,
		slog.Default(),
	)
}

// CreateAuthHandler creates an auth handler for testing
func CreateAuthHandler(
	userStore store.UserStore,
	jwtService auth.JWTService,
	passwordVerifier auth.PasswordVerifier,
	authConfig config.AuthConfig,
) *api.AuthHandler {
	return api.NewAuthHandler(
		userStore,
		jwtService,
		passwordVerifier,
		&authConfig,
		slog.Default(),
	)
}

// CreateCardHandler creates a card handler for testing
func CreateCardHandler(
	cardReviewService card_review.CardReviewService,
	cardService service.CardService,
) *api.CardHandler {
	return api.NewCardHandler(
		cardReviewService,
		cardService,
		slog.Default(),
	)
}

// CreateAuthMiddleware creates an auth middleware for testing
func CreateAuthMiddleware(jwtService auth.JWTService) *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware(jwtService)
}

// This function is already declared in helpers.go
// // AssertRollbackNoError is a utility function to cleanly roll back transactions.
// // This is a helper for WithTx and other transaction-related functions.
// func AssertRollbackNoError(t *testing.T, tx *sql.Tx) {
// 	t.Helper()
//
// 	if tx == nil {
// 		return
// 	}
//
// 	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
// 		t.Logf("Warning: failed to roll back transaction: %v", err)
// 	}
// }
