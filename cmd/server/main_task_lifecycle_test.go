package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCardRepository is a temporary implementation for testing
type MockCardRepository struct {
	logger *slog.Logger
}

// CreateMultiple logs card creation for testing
func (r *MockCardRepository) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	r.logger.Info("Mock card repository storing cards", "count", len(cards))
	return nil
}

// setupTaskLifecycleTestServer sets up a test server with all required dependencies
func setupTaskLifecycleTestServer(
	t *testing.T,
	tx store.DBTX,
	mockGenerator *mocks.MockGenerator,
) (*httptest.Server, *task.TaskRunner, error) {
	t.Helper()

	// Set up logger
	logger := slog.Default()

	// Initialize database repositories/stores
	userStore := postgres.NewPostgresUserStore(tx, 10) // BCrypt cost = 10 for faster tests
	taskStore := postgres.NewPostgresTaskStore(tx)
	memoStore := postgres.NewPostgresMemoStore(tx, logger)

	// Create authentication components
	authConfig := config.AuthConfig{
		JWTSecret:                   "test-jwt-secret-thatis32characterslong",
		TokenLifetimeMinutes:        60,
		RefreshTokenLifetimeMinutes: 1440,
	}
	jwtService, err := auth.NewJWTService(authConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create JWT service: %w", err)
	}
	passwordVerifier := auth.NewBcryptVerifier()

	// Configure task runner
	taskConfig := task.TaskRunnerConfig{
		QueueSize:    10,
		WorkerCount:  2, // Use fewer workers for tests
		StuckTaskAge: 30 * time.Minute,
	}
	taskRunner := task.NewTaskRunner(taskStore, taskConfig, logger)

	// Create the memo generation task factory
	memoTaskFactory := task.NewMemoGenerationTaskFactory(
		memoStore,
		mockGenerator,
		&MockCardRepository{logger: logger}, // Use a real repository in integration tests
		logger,
	)

	// Create the memo service adapter
	memoRepoAdapter := service.NewMemoRepositoryAdapter(memoStore, func(ctx context.Context, memo *domain.Memo) error {
		logger.Info("Creating memo through adapter", "memo_id", memo.ID)
		return nil
	})

	// Create the memo service
	memoService := service.NewMemoService(memoRepoAdapter, taskRunner, memoTaskFactory, logger)

	// Create the API handlers
	authHandler := api.NewAuthHandler(userStore, jwtService, passwordVerifier, &authConfig)
	memoHandler := api.NewMemoHandler(memoService)
	authMiddleware := authmiddleware.NewAuthMiddleware(jwtService)

	// Create router and set up routes
	r := chi.NewRouter()

	// Apply middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Register routes
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

	// Create the test server
	testServer := httptest.NewServer(r)

	// Start the task runner
	err = taskRunner.Start()
	if err != nil {
		testServer.Close()
		return nil, nil, fmt.Errorf("failed to start task runner: %w", err)
	}

	return testServer, taskRunner, nil
}

// waitForCondition polls until the condition function returns true or timeout is reached
func waitForCondition(
	t *testing.T,
	timeout time.Duration,
	interval time.Duration,
	condition func() bool,
	message string,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}

	t.Fatalf("Timeout waiting for condition: %s (waited %v)", message, timeout)
}

// TestMemoTaskLifecycleSuccess tests the successful path of task submission, processing, and completion
func TestMemoTaskLifecycleSuccess(t *testing.T) {
	// Skip if no test database available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	testutils.WithTx(t, testDB, func(dbtx store.DBTX) {
		// Cast to *sql.Tx for direct DB access
		tx := dbtx.(*sql.Tx)
		// Create mock generator configured for success
		mockGenerator := mocks.NewMockGeneratorWithDefaultCards(uuid.Nil, uuid.Nil)

		// Set up test server
		testServer, taskRunner, err := setupTaskLifecycleTestServer(t, tx, mockGenerator)
		require.NoError(t, err, "Failed to set up test server")
		defer func() {
			taskRunner.Stop()
			testServer.Close()
		}()

		// Create a test user
		userEmail := "task-test@example.com"
		userPassword := "securepassword1234"
		var userID string
		var token string

		// Step 1: Register a new user
		t.Log("Registering test user")
		registerPayload := map[string]interface{}{
			"email":    userEmail,
			"password": userPassword,
		}
		registerBody, err := json.Marshal(registerPayload)
		require.NoError(t, err)

		registerResp, err := http.Post(
			testServer.URL+"/api/auth/register",
			"application/json",
			bytes.NewBuffer(registerBody),
		)
		require.NoError(t, err)
		defer func() {
			if err := registerResp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		require.Equal(t, http.StatusCreated, registerResp.StatusCode, "Registration should succeed")

		var registerData map[string]interface{}
		err = json.NewDecoder(registerResp.Body).Decode(&registerData)
		require.NoError(t, err)
		require.NotEmpty(t, registerData["user_id"], "User ID should be returned")
		require.NotEmpty(t, registerData["token"], "Token should be returned")

		userID = registerData["user_id"].(string)
		token = registerData["token"].(string)

		// Step 2: Create a memo via API
		t.Log("Creating memo via API")
		memoText := "Test memo for task lifecycle integration test"
		memoPayload := map[string]interface{}{
			"text": memoText,
		}
		memoBody, err := json.Marshal(memoPayload)
		require.NoError(t, err)

		memoReq, err := http.NewRequest(
			"POST",
			testServer.URL+"/api/memos",
			bytes.NewBuffer(memoBody),
		)
		require.NoError(t, err)
		memoReq.Header.Set("Content-Type", "application/json")
		memoReq.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		memoResp, err := client.Do(memoReq)
		require.NoError(t, err)
		defer func() {
			if err := memoResp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Verify the immediate API response
		require.Equal(t, http.StatusAccepted, memoResp.StatusCode, "Memo creation should be accepted")

		var memoData api.MemoResponse
		err = json.NewDecoder(memoResp.Body).Decode(&memoData)
		require.NoError(t, err)

		require.NotEmpty(t, memoData.ID, "Memo ID should be returned")
		assert.Equal(t, userID, memoData.UserID, "User ID should match")
		assert.Equal(t, memoText, memoData.Text, "Memo text should match")
		assert.Equal(t, string(domain.MemoStatusPending), memoData.Status, "Initial status should be pending")

		memoID := memoData.ID

		// Step 3: Poll for memo status to change to processing and then completed
		t.Log("Waiting for memo processing to complete")

		// Helper function to get current memo state
		getMemoStatus := func() (domain.MemoStatus, error) {
			var status string
			err := tx.QueryRow(
				"SELECT status FROM memos WHERE id = $1",
				memoID,
			).Scan(&status)
			if err != nil {
				return "", err
			}
			return domain.MemoStatus(status), nil
		}

		// Helper function to check for task status
		getTaskStatus := func(memoID string) (task.TaskStatus, error) {
			var status string
			// Note: This query assumes that the task payload contains the memo_id
			// The actual query may need to be adjusted based on how tasks are stored
			err := tx.QueryRow(
				"SELECT status FROM tasks WHERE payload::json->>'memo_id' = $1",
				memoID,
			).Scan(&status)
			if err != nil {
				return "", err
			}
			return task.TaskStatus(status), nil
		}

		// Helper function to count cards for the memo
		countCards := func(memoID string) (int, error) {
			var count int
			err := tx.QueryRow(
				"SELECT COUNT(*) FROM cards WHERE memo_id = $1",
				memoID,
			).Scan(&count)
			if err != nil {
				return 0, err
			}
			return count, nil
		}

		// Wait for memo to transition to processing
		waitForCondition(t, 5*time.Second, 100*time.Millisecond, func() bool {
			status, err := getMemoStatus()
			if err != nil {
				t.Logf("Error getting memo status: %v", err)
				return false
			}
			t.Logf("Current memo status: %s", status)
			return status == domain.MemoStatusProcessing
		}, "memo to transition to processing status")

		// Wait for memo to transition to completed
		waitForCondition(t, 10*time.Second, 200*time.Millisecond, func() bool {
			status, err := getMemoStatus()
			if err != nil {
				t.Logf("Error getting memo status: %v", err)
				return false
			}
			t.Logf("Current memo status: %s", status)
			return status == domain.MemoStatusCompleted
		}, "memo to transition to completed status")

		// Verify task completed
		taskStatus, err := getTaskStatus(memoID)
		require.NoError(t, err, "Should be able to fetch task status")
		assert.Equal(t, task.TaskStatusCompleted, taskStatus, "Task should be completed")

		// Verify cards were created
		cardCount, err := countCards(memoID)
		require.NoError(t, err, "Should be able to count cards")
		assert.Equal(t, 2, cardCount, "Two cards should have been created")

		// Additional verification: Fetch and check card content
		var cardFront string
		err = tx.QueryRow(
			"SELECT content->>'front' FROM cards WHERE memo_id = $1 LIMIT 1",
			memoID,
		).Scan(&cardFront)
		require.NoError(t, err, "Should be able to fetch card content")
		assert.Contains(t, cardFront, "architecture", "Card front should contain expected text")
	})
}

// TestMemoTaskLifecycleFailure tests the error handling path of task processing
func TestMemoTaskLifecycleFailure(t *testing.T) {
	// Skip if no test database available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	testutils.WithTx(t, testDB, func(dbtx store.DBTX) {
		// Cast to *sql.Tx for direct DB access
		tx := dbtx.(*sql.Tx)
		// Create mock generator configured to return an error
		mockGenerator := mocks.MockGeneratorThatFails()

		// Set up test server
		testServer, taskRunner, err := setupTaskLifecycleTestServer(t, tx, mockGenerator)
		require.NoError(t, err, "Failed to set up test server")
		defer func() {
			taskRunner.Stop()
			testServer.Close()
		}()

		// Create a test user
		userEmail := "task-failure-test@example.com"
		userPassword := "securepassword1234"
		var userID string
		var token string

		// Step 1: Register a new user
		t.Log("Registering test user")
		registerPayload := map[string]interface{}{
			"email":    userEmail,
			"password": userPassword,
		}
		registerBody, err := json.Marshal(registerPayload)
		require.NoError(t, err)

		registerResp, err := http.Post(
			testServer.URL+"/api/auth/register",
			"application/json",
			bytes.NewBuffer(registerBody),
		)
		require.NoError(t, err)
		defer func() {
			if err := registerResp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		require.Equal(t, http.StatusCreated, registerResp.StatusCode, "Registration should succeed")

		var registerData map[string]interface{}
		err = json.NewDecoder(registerResp.Body).Decode(&registerData)
		require.NoError(t, err)
		require.NotEmpty(t, registerData["user_id"], "User ID should be returned")
		require.NotEmpty(t, registerData["token"], "Token should be returned")

		userID = registerData["user_id"].(string)
		token = registerData["token"].(string)

		// Step 2: Create a memo via API
		t.Log("Creating memo via API (should fail during processing)")
		memoText := "Test memo that will fail during processing"
		memoPayload := map[string]interface{}{
			"text": memoText,
		}
		memoBody, err := json.Marshal(memoPayload)
		require.NoError(t, err)

		memoReq, err := http.NewRequest(
			"POST",
			testServer.URL+"/api/memos",
			bytes.NewBuffer(memoBody),
		)
		require.NoError(t, err)
		memoReq.Header.Set("Content-Type", "application/json")
		memoReq.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		memoResp, err := client.Do(memoReq)
		require.NoError(t, err)
		defer func() {
			if err := memoResp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Verify immediate API response (should still be accepted since failure happens async)
		require.Equal(t, http.StatusAccepted, memoResp.StatusCode, "Memo creation should be accepted")

		var memoData api.MemoResponse
		err = json.NewDecoder(memoResp.Body).Decode(&memoData)
		require.NoError(t, err)

		require.NotEmpty(t, memoData.ID, "Memo ID should be returned")
		assert.Equal(t, userID, memoData.UserID, "User ID should match")
		assert.Equal(t, memoText, memoData.Text, "Memo text should match")
		assert.Equal(t, string(domain.MemoStatusPending), memoData.Status, "Initial status should be pending")

		memoID := memoData.ID

		// Step 3: Poll for memo status to change to processing and then failed
		t.Log("Waiting for memo processing to fail")

		// Helper function to get current memo state
		getMemoStatus := func() (domain.MemoStatus, error) {
			var status string
			err := tx.QueryRow(
				"SELECT status FROM memos WHERE id = $1",
				memoID,
			).Scan(&status)
			if err != nil {
				return "", err
			}
			return domain.MemoStatus(status), nil
		}

		// Helper function to check for task status
		getTaskStatus := func(memoID string) (task.TaskStatus, error) {
			var status string
			err := tx.QueryRow(
				"SELECT status FROM tasks WHERE payload::json->>'memo_id' = $1",
				memoID,
			).Scan(&status)
			if err != nil {
				return "", err
			}
			return task.TaskStatus(status), nil
		}

		// Helper function to count cards for the memo
		countCards := func(memoID string) (int, error) {
			var count int
			err := tx.QueryRow(
				"SELECT COUNT(*) FROM cards WHERE memo_id = $1",
				memoID,
			).Scan(&count)
			if err != nil {
				return 0, err
			}
			return count, nil
		}

		// Wait for memo to transition to processing
		waitForCondition(t, 5*time.Second, 100*time.Millisecond, func() bool {
			status, err := getMemoStatus()
			if err != nil {
				t.Logf("Error getting memo status: %v", err)
				return false
			}
			t.Logf("Current memo status: %s", status)
			return status == domain.MemoStatusProcessing
		}, "memo to transition to processing status")

		// Wait for memo to transition to failed
		waitForCondition(t, 10*time.Second, 200*time.Millisecond, func() bool {
			status, err := getMemoStatus()
			if err != nil {
				t.Logf("Error getting memo status: %v", err)
				return false
			}
			t.Logf("Current memo status: %s", status)
			return status == domain.MemoStatusFailed
		}, "memo to transition to failed status")

		// Verify task failed
		taskStatus, err := getTaskStatus(memoID)
		require.NoError(t, err, "Should be able to fetch task status")
		assert.Equal(t, task.TaskStatusFailed, taskStatus, "Task should be failed")

		// Verify no cards were created
		cardCount, err := countCards(memoID)
		require.NoError(t, err, "Should be able to count cards")
		assert.Equal(t, 0, cardCount, "No cards should have been created")

		// Verify error message in task (optional, if your implementation stores it)
		var errorMsg string
		err = tx.QueryRow(
			"SELECT error_message FROM tasks WHERE payload::json->>'memo_id' = $1",
			memoID,
		).Scan(&errorMsg)
		if err == nil { // Only check if the query succeeded (some implementations might not store the error)
			assert.Contains(t, errorMsg, "generation failed", "Task should contain the error message")
		}
	})
}
