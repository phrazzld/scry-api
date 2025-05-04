//go:build integration

// This file provides helpers specifically for integration tests
// It should be built only when the integration tag is enabled

package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/testutils/api"
)

// CreateTestJWTService creates a real JWT service for testing with a pre-configured secret and expiration.
// NOTE: This function is kept for backward compatibility. New code should use auth.DefaultJWTConfig() instead.
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
// NOTE: This function is kept for backward compatibility. New code should use api.GenerateAuthHeader instead.
func GenerateAuthHeader(userID uuid.UUID) (string, error) {
	return api.GenerateAuthHeader(userID)
}

// These functions are for backward compatibility only.
// New code should use the corresponding functions in the api package.

// CardOption is a function that configures a Card for testing.
type CardOption = api.CardOption

// WithCardID sets a specific ID for the test card.
var WithCardID = api.WithCardID

// WithCardUserID sets a specific user ID for the test card.
var WithCardUserID = api.WithCardUserID

// WithCardMemoID sets a specific memo ID for the test card.
var WithCardMemoID = api.WithCardMemoID

// WithCardContent sets the content for the test card using a map.
var WithCardContent = api.WithCardContent

// WithRawCardContent sets raw JSON content for the test card.
var WithRawCardContent = api.WithRawCardContent

// WithCardCreatedAt sets the creation timestamp for the test card.
var WithCardCreatedAt = api.WithCardCreatedAt

// WithCardUpdatedAt sets the update timestamp for the test card.
var WithCardUpdatedAt = api.WithCardUpdatedAt

// CreateCardForAPITest creates a Card instance for API testing with default values.
var CreateCardForAPITest = api.CreateCardForAPITest

// StatsOption is a function that configures UserCardStats for testing.
type StatsOption = api.StatsOption

// WithStatsUserID sets a specific user ID for the test stats.
var WithStatsUserID = api.WithStatsUserID

// WithStatsCardID sets a specific card ID for the test stats.
var WithStatsCardID = api.WithStatsCardID

// WithStatsInterval sets the interval for the test stats.
var WithStatsInterval = api.WithStatsInterval

// WithStatsEaseFactor sets the ease factor for the test stats.
var WithStatsEaseFactor = api.WithStatsEaseFactor

// WithStatsConsecutiveCorrect sets the consecutive correct count for the test stats.
var WithStatsConsecutiveCorrect = api.WithStatsConsecutiveCorrect

// WithStatsLastReviewedAt sets the last reviewed timestamp for the test stats.
var WithStatsLastReviewedAt = api.WithStatsLastReviewedAt

// WithStatsNextReviewAt sets the next review timestamp for the test stats.
var WithStatsNextReviewAt = api.WithStatsNextReviewAt

// WithStatsReviewCount sets the review count for the test stats.
var WithStatsReviewCount = api.WithStatsReviewCount

// WithStatsCreatedAt sets the creation timestamp for the test stats.
var WithStatsCreatedAt = api.WithStatsCreatedAt

// WithStatsUpdatedAt sets the update timestamp for the test stats.
var WithStatsUpdatedAt = api.WithStatsUpdatedAt

// CreateStatsForAPITest creates a UserCardStats instance for API testing with default values.
var CreateStatsForAPITest = api.CreateStatsForAPITest

// Forward declarations for functions that have been moved to the api package
// These are kept for backward compatibility only and should not be used in new code.

// SetupCardReviewTestServerWithNextCard is a compatibility function that delegates to api.SetupCardReviewTestServerWithNextCard
var SetupCardReviewTestServerWithNextCard = api.SetupCardReviewTestServerWithNextCard

// SetupCardReviewTestServerWithError is a compatibility function that delegates to api.SetupCardReviewTestServerWithError
var SetupCardReviewTestServerWithError = api.SetupCardReviewTestServerWithError

// SetupCardReviewTestServerWithAuthError is a compatibility function that delegates to api.SetupCardReviewTestServerWithAuthError
var SetupCardReviewTestServerWithAuthError = api.SetupCardReviewTestServerWithAuthError

// SetupCardReviewTestServerWithUpdatedStats is a compatibility function that delegates to api.SetupCardReviewTestServerWithUpdatedStats
var SetupCardReviewTestServerWithUpdatedStats = api.SetupCardReviewTestServerWithUpdatedStats

// AssertErrorResponse is a compatibility function that delegates to api.AssertErrorResponse
var AssertErrorResponse = api.AssertErrorResponse

// AssertValidationError is a compatibility function that delegates to api.AssertValidationError
var AssertValidationError = api.AssertValidationError

// GenerateRefreshTokenWithExpiry generates a refresh token with a custom expiration time.
// This function is unique here and not duplicated in the api package.
func GenerateRefreshTokenWithExpiry(t *testing.T, userID uuid.UUID, expiry time.Time) (string, error) {
	jwtService, err := CreateTestJWTService()
	if err != nil {
		return "", err
	}

	token, err := jwtService.GenerateRefreshTokenWithExpiry(context.Background(), userID, expiry)
	if err != nil {
		return "", err
	}

	return token, nil
}
