//go:build (integration || test_without_external_deps) && exported_core_functions

package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/logger"
	"github.com/stretchr/testify/require"
)

// Backwards compatibility functions
// ================================

// loadConfig is a backwards compatibility function for tests.
// It simply calls loadAppConfig to match the original signature.
func loadConfig() (*config.Config, error) {
	return loadAppConfig()
}

// IsIntegrationTestEnvironment is a backwards compatibility helper.
// It checks if DATABASE_URL is set.
func IsIntegrationTestEnvironment() bool {
	return os.Getenv("DATABASE_URL") != ""
}

// Test Infrastructure for cmd/server
// ==================================

// MockDB is a minimal mock database for testing server components without actual DB
type MockDB struct {
	shouldFailPing  bool
	shouldFailClose bool
}

func (m *MockDB) PingContext(ctx context.Context) error {
	if m.shouldFailPing {
		return sql.ErrConnDone
	}
	return nil
}

func (m *MockDB) Close() error {
	if m.shouldFailClose {
		return sql.ErrConnDone
	}
	return nil
}

func (m *MockDB) Prepare(query string) (*sql.Stmt, error) {
	return nil, sql.ErrNoRows
}

func (m *MockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, sql.ErrNoRows
}

func (m *MockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, sql.ErrNoRows
}

func (m *MockDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

// NewMockDB creates a mock database for testing
func NewMockDB() *MockDB {
	return &MockDB{}
}

// NewFailingMockDB creates a mock database that fails operations
func NewFailingMockDB() *MockDB {
	return &MockDB{
		shouldFailPing:  true,
		shouldFailClose: true,
	}
}

// CreateTestConfig creates a minimal valid configuration for testing
func CreateTestConfig(t *testing.T) *config.Config {
	t.Helper()

	return &config.Config{
		Server: config.ServerConfig{
			Port:     8080,
			LogLevel: "info",
		},
		Database: config.DatabaseConfig{
			URL: "postgres://testuser:testpass@localhost:5432/testdb",
		},
		Auth: config.AuthConfig{
			JWTSecret:                   "test-jwt-secret-key-32-chars-123", // Now 32 chars
			TokenLifetimeMinutes:        60,
			RefreshTokenLifetimeMinutes: 1440,
		},
		LLM: config.LLMConfig{
			GeminiAPIKey:       "test-gemini-api-key",
			ModelName:          "gemini-1.5-flash",
			PromptTemplatePath: "../../prompts/flashcard_template.txt",
		},
		Task: config.TaskConfig{
			WorkerCount:         1,
			QueueSize:           10,
			StuckTaskAgeMinutes: 5,
		},
	}
}

// CreateMinimalTestConfig creates config with only required fields for testing
func CreateMinimalTestConfig(t *testing.T) *config.Config {
	t.Helper()

	return &config.Config{
		Server: config.ServerConfig{
			Port:     8080,
			LogLevel: "info",
		},
		Database: config.DatabaseConfig{
			URL: "test://database",
		},
		Auth: config.AuthConfig{
			JWTSecret:                   "minimal-test-secret-key-32-char",
			TokenLifetimeMinutes:        60,
			RefreshTokenLifetimeMinutes: 1440,
		},
		LLM: config.LLMConfig{
			GeminiAPIKey:       "test-key",
			ModelName:          "test-model",
			PromptTemplatePath: "test.txt",
		},
		Task: config.TaskConfig{
			WorkerCount:         1,
			QueueSize:           10,
			StuckTaskAgeMinutes: 5,
		},
	}
}

// CreateTestLogger creates a test logger for server testing
func CreateTestLogger(t *testing.T) (*slog.Logger, *logger.TestLogBuffer) {
	t.Helper()
	return logger.GetTestLogger(t)
}

// ServerTestCase represents a test case for server testing
type ServerTestCase struct {
	Name          string
	Config        *config.Config
	ExpectError   bool
	ErrorContains string
	SkipReason    string
	PreTest       func(t *testing.T)
	PostTest      func(t *testing.T)
}

// RunServerTestCases runs a table of server test cases
func RunServerTestCases(t *testing.T, testCases []ServerTestCase, testFunc func(t *testing.T, tc ServerTestCase)) {
	t.Helper()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.SkipReason != "" {
				t.Skip(tc.SkipReason)
			}

			if tc.PreTest != nil {
				tc.PreTest(t)
			}

			testFunc(t, tc)

			if tc.PostTest != nil {
				tc.PostTest(t)
			}
		})
	}
}

// AssertConfigurationValid validates that a config is non-nil and has required fields
func AssertConfigurationValid(t *testing.T, cfg *config.Config) {
	t.Helper()

	require.NotNil(t, cfg, "config should not be nil")
	require.NotEmpty(t, cfg.Database.URL, "database URL should not be empty")
	require.NotEmpty(t, cfg.Auth.JWTSecret, "JWT secret should not be empty")
	require.Greater(t, cfg.Server.Port, 0, "server port should be positive")
	require.NotEmpty(t, cfg.Server.LogLevel, "log level should not be empty")
}

// AssertLoggerValid validates that a logger is functional
func AssertLoggerValid(t *testing.T, logger *slog.Logger) {
	t.Helper()

	require.NotNil(t, logger, "logger should not be nil")

	// Test that logger doesn't panic on various log levels
	require.NotPanics(t, func() {
		logger.Debug("test debug message")
		logger.Info("test info message")
		logger.Warn("test warn message")
		logger.Error("test error message")
	}, "logger should not panic on log calls")
}

// MockHTTPHandler creates a simple mock HTTP handler for testing
func MockHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// WaitForCondition waits for a condition to be true or times out
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Condition not met within timeout: %s", message)
}

// SetupTestEnvironment sets up the required environment variables for configuration loading
func SetupTestEnvironment(t *testing.T) {
	t.Helper()

	// Set up all required environment variables for configuration validation
	t.Setenv("SCRY_SERVER_PORT", "8080")
	t.Setenv("SCRY_SERVER_LOG_LEVEL", "info")
	t.Setenv("SCRY_DATABASE_URL", "postgres://testuser:testpass@localhost:5432/testdb")
	t.Setenv("SCRY_AUTH_JWT_SECRET", "test-jwt-secret-key-32-chars-123")
	t.Setenv("SCRY_AUTH_TOKEN_LIFETIME_MINUTES", "60")
	t.Setenv("SCRY_AUTH_REFRESH_TOKEN_LIFETIME_MINUTES", "1440")
	t.Setenv("SCRY_LLM_GEMINI_API_KEY", "test-gemini-api-key")
	t.Setenv("SCRY_LLM_MODEL_NAME", "gemini-1.5-flash")
	t.Setenv("SCRY_LLM_PROMPT_TEMPLATE_PATH", "../../prompts/flashcard_template.txt")
	t.Setenv("SCRY_TASK_WORKER_COUNT", "1")
	t.Setenv("SCRY_TASK_QUEUE_SIZE", "10")
	t.Setenv("SCRY_TASK_STUCK_TASK_AGE_MINUTES", "5")
}
