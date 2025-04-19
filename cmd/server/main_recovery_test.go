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
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/phrazzld/scry-api/internal/api"
	authmiddleware "github.com/phrazzld/scry-api/internal/api/middleware"
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/generation"
	"github.com/phrazzld/scry-api/internal/mocks"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/service"
	"github.com/phrazzld/scry-api/internal/service/auth"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// RecoveryMockGenerator wraps the standardized mock generator with recovery-specific functionality
type RecoveryMockGenerator struct {
	logger            *slog.Logger
	shouldReturnError bool
	executionDelay    time.Duration        // Add delay to simulate processing time
	executionCount    int                  // Track number of executions
	mu                sync.Mutex           // Protect execution count
	mockGenerator     *mocks.MockGenerator // The underlying mock generator
}

// NewRecoveryMockGenerator creates a new recovery mock generator with specified configuration
func NewRecoveryMockGenerator(logger *slog.Logger, shouldFail bool, delay time.Duration) *RecoveryMockGenerator {
	var mockGen *mocks.MockGenerator

	if shouldFail {
		mockGen = mocks.MockGeneratorThatFails()
	} else {
		// Use default cards with placeholder IDs that will be replaced during task execution
		mockGen = mocks.NewMockGeneratorWithDefaultCards(uuid.Nil, uuid.Nil)
	}

	return &RecoveryMockGenerator{
		logger:            logger,
		shouldReturnError: shouldFail,
		executionDelay:    delay,
		executionCount:    0,
		mockGenerator:     mockGen,
	}
}

// GenerateCards creates test flashcards or returns an error based on configuration
func (g *RecoveryMockGenerator) GenerateCards(
	ctx context.Context,
	memoText string,
	userID uuid.UUID,
) ([]*domain.Card, error) {
	g.mu.Lock()
	g.executionCount++
	count := g.executionCount
	g.mu.Unlock()

	g.logger.Info(
		"Recovery mock generator creating cards",
		"memo_text_length", len(memoText),
		"user_id", userID,
		"should_error", g.shouldReturnError,
		"delay", g.executionDelay,
		"execution_count", count,
	)

	// Simulate work with a delay if configured
	if g.executionDelay > 0 {
		select {
		case <-time.After(g.executionDelay):
			// Delay complete
		case <-ctx.Done():
			g.logger.Warn("Card generation cancelled during delay", "error", ctx.Err())
			return nil, ctx.Err()
		}
	}

	// If configured to return error, return early without calling underlying mock
	if g.shouldReturnError {
		return nil, generation.ErrGenerationFailed
	}

	// Delegate to the underlying standardized mock generator
	return g.mockGenerator.GenerateCards(ctx, memoText, userID)
}

// RecoveryMockCardRepository tracks created cards for verification
type RecoveryMockCardRepository struct {
	logger       *slog.Logger
	mu           sync.Mutex // Protect access to createdCards
	createdCards map[string][]*domain.Card
}

// NewRecoveryMockCardRepository creates a new mock card repository
func NewRecoveryMockCardRepository(logger *slog.Logger) *RecoveryMockCardRepository {
	return &RecoveryMockCardRepository{
		logger:       logger,
		createdCards: make(map[string][]*domain.Card),
	}
}

// CreateMultiple logs card creation for testing and stores them in memory
func (r *RecoveryMockCardRepository) CreateMultiple(
	ctx context.Context,
	cards []*domain.Card,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(cards) == 0 {
		r.logger.Warn("Mock card repository received zero cards to store")
		return nil
	}

	memoID := cards[0].MemoID.String()
	r.logger.Info("Mock card repository storing cards", "count", len(cards), "memo_id", memoID)
	r.createdCards[memoID] = append(r.createdCards[memoID], cards...)
	return nil
}

// GetCreatedCards returns the cards created for a specific memo ID
func (r *RecoveryMockCardRepository) GetCreatedCards(memoID string) []*domain.Card {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Return a copy to avoid race conditions if the caller modifies the slice
	cardsCopy := make([]*domain.Card, len(r.createdCards[memoID]))
	copy(cardsCopy, r.createdCards[memoID])
	return cardsCopy
}

// GetExecutionCount returns the number of times the generator was executed
func (g *RecoveryMockGenerator) GetExecutionCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.executionCount
}

// setupTestLogger creates and configures a logger for testing
func setupTestLogger() *slog.Logger {
	return slog.Default()
}

// setupTestStores initializes database repositories
func setupTestStores(
	t *testing.T,
	dbtx store.DBTX,
	logger *slog.Logger,
) (*postgres.PostgresUserStore, *postgres.PostgresTaskStore, *postgres.PostgresMemoStore) {
	t.Helper()
	userStore := postgres.NewPostgresUserStore(dbtx, 10) // BCrypt cost = 10 for faster tests
	taskStore := postgres.NewPostgresTaskStore(dbtx)
	memoStore := postgres.NewPostgresMemoStore(dbtx, logger)
	return userStore, taskStore, memoStore
}

// setupTestAuthComponents creates auth config and initializes auth services
func setupTestAuthComponents(
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
		return authConfig, nil, nil, fmt.Errorf("failed to create JWT service: %w", err)
	}
	passwordVerifier := auth.NewBcryptVerifier()
	return authConfig, jwtService, passwordVerifier, nil
}

// setupTestTaskProcessing sets up task runner, task factory, memo repository adapter, and memo service
func setupTestTaskProcessing(
	t *testing.T,
	taskStore *postgres.PostgresTaskStore,
	memoStore *postgres.PostgresMemoStore,
	mockGenerator *RecoveryMockGenerator,
	mockCardRepo *RecoveryMockCardRepository,
	taskConfig task.TaskRunnerConfig,
	logger *slog.Logger,
) (*task.TaskRunner, service.MemoService) {
	t.Helper()
	// Configure task runner
	taskRunner := task.NewTaskRunner(taskStore, taskConfig, logger)

	// Create the memo service adapter for task package
	memoServiceAdapter := task.NewMemoServiceAdapter(memoStore)

	// Create the memo generation task factory with the adapter
	memoTaskFactory := task.NewMemoGenerationTaskFactory(
		memoServiceAdapter,
		mockGenerator,
		mockCardRepo,
		logger,
	)

	// Create the memo repository adapter for service package
	memoRepoAdapter := service.NewMemoRepositoryAdapter(memoStore)

	// Create the memo service
	memoService := service.NewMemoService(memoRepoAdapter, taskRunner, memoTaskFactory, logger)

	return taskRunner, memoService
}

// setupTestAPIHandlers creates API handlers and middleware
func setupTestAPIHandlers(
	t *testing.T,
	userStore *postgres.PostgresUserStore,
	jwtService auth.JWTService,
	passwordVerifier auth.PasswordVerifier,
	authConfig config.AuthConfig,
	memoService service.MemoService,
) (*api.AuthHandler, *api.MemoHandler, *authmiddleware.AuthMiddleware) {
	t.Helper()
	authHandler := api.NewAuthHandler(userStore, jwtService, passwordVerifier, &authConfig)
	memoHandler := api.NewMemoHandler(memoService)
	authMiddleware := authmiddleware.NewAuthMiddleware(jwtService)
	return authHandler, memoHandler, authMiddleware
}

// setupRecoveryTestRouter creates router, applies middleware, registers routes for recovery tests
func setupRecoveryTestRouter(
	t *testing.T,
	authHandler *api.AuthHandler,
	memoHandler *api.MemoHandler,
	authMiddleware *authmiddleware.AuthMiddleware,
) *httptest.Server {
	t.Helper()
	// Create router and set up routes
	r := chi.NewRouter()

	// Apply middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
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
	return httptest.NewServer(r)
}

// setupRecoveryTestInstance creates a test instance for recovery testing
// It returns the server, task runner, and any error
func setupRecoveryTestInstance(
	t *testing.T,
	dbtx store.DBTX,
	mockGenerator *RecoveryMockGenerator,
	mockCardRepo *RecoveryMockCardRepository,
	taskConfig task.TaskRunnerConfig,
) (*httptest.Server, *task.TaskRunner, error) {
	t.Helper()

	// Set up logger
	logger := setupTestLogger()

	// Initialize database repositories using the transaction
	userStore, taskStore, memoStore := setupTestStores(t, dbtx, logger)

	// Create authentication components
	authConfig, jwtService, passwordVerifier, err := setupTestAuthComponents(t)
	if err != nil {
		return nil, nil, err
	}

	// Setup task processing components
	taskRunner, memoService := setupTestTaskProcessing(
		t, taskStore, memoStore, mockGenerator, mockCardRepo, taskConfig, logger,
	)

	// Create the API handlers
	authHandler, memoHandler, authMiddleware := setupTestAPIHandlers(
		t, userStore, jwtService, passwordVerifier, authConfig, memoService,
	)

	// Create the test server
	testServer := setupRecoveryTestRouter(t, authHandler, memoHandler, authMiddleware)

	// Note: We intentionally don't start the task runner here
	// The caller will decide when to start it, simulating application startup

	return testServer, taskRunner, nil
}

// Helper function to get task status directly from DB
func getTaskStatusDirectly(
	t *testing.T,
	dbtx store.DBTX,
	taskID uuid.UUID,
) (task.TaskStatus, error) {
	t.Helper()
	var status string

	// Use dbtx directly without casting, using QueryRowContext from the DBTX interface
	err := dbtx.QueryRowContext(
		context.Background(),
		"SELECT status FROM tasks WHERE id = $1",
		taskID,
	).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("task with ID %s not found", taskID)
		}
		return "", fmt.Errorf("failed to query task status for %s: %w", taskID, err)
	}
	return task.TaskStatus(status), nil
}

// Helper function to get task ID for a memo
func getTaskIDForMemo(t *testing.T, dbtx store.DBTX, memoID uuid.UUID) (uuid.UUID, error) {
	t.Helper()
	var taskID uuid.UUID

	// Use dbtx directly without casting, using QueryRowContext from the DBTX interface
	err := dbtx.QueryRowContext(
		context.Background(),
		"SELECT id FROM tasks WHERE payload->>'memo_id' = $1 ORDER BY created_at DESC LIMIT 1",
		memoID.String(),
	).Scan(&taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, fmt.Errorf("no task found for memo ID %s", memoID)
		}
		return uuid.Nil, fmt.Errorf("failed to query task ID for memo %s: %w", memoID, err)
	}
	return taskID, nil
}

// Helper function to get memo status directly from DB
func getMemoStatusDirectly(
	t *testing.T,
	dbtx store.DBTX,
	memoID uuid.UUID,
) (domain.MemoStatus, error) {
	t.Helper()
	var status string

	// Use dbtx directly without casting, using QueryRowContext from the DBTX interface
	err := dbtx.QueryRowContext(
		context.Background(),
		"SELECT status FROM memos WHERE id = $1",
		memoID,
	).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("memo with ID %s not found", memoID)
		}
		return "", fmt.Errorf("failed to query memo status for %s: %w", memoID, err)
	}
	return domain.MemoStatus(status), nil
}

// getDefaultTestTaskConfig returns a default TaskRunnerConfig with common test values
func getDefaultTestTaskConfig() task.TaskRunnerConfig {
	return task.TaskRunnerConfig{
		WorkerCount:  1, // Use 1 worker for more predictable test execution
		QueueSize:    10,
		StuckTaskAge: 30 * time.Minute,
	}
}

// getTestTaskConfigWithWorkers returns a TaskRunnerConfig with a specific worker count
func getTestTaskConfigWithWorkers(workerCount int) task.TaskRunnerConfig {
	config := getDefaultTestTaskConfig()
	config.WorkerCount = workerCount
	return config
}

// waitForCondition polls until the condition function returns true or timeout is reached
func waitForRecoveryCondition(
	t *testing.T,
	timeout time.Duration,
	interval time.Duration,
	condition func() (bool, error),
	message string,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		ok, err := condition()
		if err != nil {
			lastErr = err
			// Log the error but continue polling
			t.Logf("Error checking condition: %v", err)
		} else if ok {
			return // Condition met
		}
		time.Sleep(interval)
	}

	if lastErr != nil {
		t.Fatalf(
			"Timeout waiting for condition: %s (waited %v). Last error: %v",
			message,
			timeout,
			lastErr,
		)
	} else {
		t.Fatalf("Timeout waiting for condition: %s (waited %v)", message, timeout)
	}
}

// TestTaskRecovery_Success tests the successful recovery of a task stuck in 'processing' state
func TestTaskRecovery_Success(t *testing.T) {
	// Skip if no test database available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	testutils.WithTx(t, testDB, func(dbtx store.DBTX) {
		tx := dbtx.(*sql.Tx) // Cast for direct DB operations
		logger := slog.Default()

		// --- Setup Phase ---
		t.Log("Setting up initial application instance...")

		// Create mocks for the first instance
		mockGenerator1 := NewRecoveryMockGenerator(logger, false, 0)
		mockCardRepo1 := NewRecoveryMockCardRepository(logger)
		taskConfig1 := getDefaultTestTaskConfig()

		// Setup first app instance (doesn't need a running server)
		_, _, err := setupRecoveryTestInstance(t, tx, mockGenerator1, mockCardRepo1, taskConfig1)
		require.NoError(t, err, "Failed to set up first app instance")

		// Create a test user directly in DB
		userID := uuid.New()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		_, err = tx.Exec("INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)",
			userID, "recovery-test@example.com", string(hashedPassword))
		require.NoError(t, err, "Failed to create test user")

		// Create a memo directly in DB
		memoID := uuid.New()
		memoText := "Memo for recovery test"

		// Create a memo with specific ID and status
		memo := &domain.Memo{
			ID:        memoID,
			UserID:    userID,
			Text:      memoText,
			Status:    domain.MemoStatusPending,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		memoStore := postgres.NewPostgresMemoStore(tx, logger)
		err = memoStore.Create(context.Background(), memo)
		require.NoError(t, err, "Failed to create test memo")

		// Create a task to process this memo
		taskID := uuid.New()
		taskPayload := map[string]string{
			"memo_id": memoID.String(),
			"user_id": userID.String(),
		}
		payloadBytes, err := json.Marshal(taskPayload)
		require.NoError(t, err, "Failed to marshal task payload")

		// Insert task with 'processing' status (simulating a task interrupted by shutdown)
		now := time.Now().UTC()
		_, err = tx.Exec(
			"INSERT INTO tasks (id, type, payload, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
			taskID,
			task.TaskTypeMemoGeneration,
			payloadBytes,
			string(task.TaskStatusProcessing),
			now,
			now,
		)
		require.NoError(t, err, "Failed to insert task with processing status")

		// Update memo status to match task's state
		_, err = tx.Exec(
			"UPDATE memos SET status = $1 WHERE id = $2",
			string(domain.MemoStatusProcessing), memoID,
		)
		require.NoError(t, err, "Failed to update memo status to processing")

		// Verify initial state
		initialTaskStatus, err := getTaskStatusDirectly(t, tx, taskID)
		require.NoError(t, err, "Failed to get initial task status")
		assert.Equal(
			t,
			task.TaskStatusProcessing,
			initialTaskStatus,
			"Task should start in processing state",
		)

		initialMemoStatus, err := getMemoStatusDirectly(t, tx, memoID)
		require.NoError(t, err, "Failed to get initial memo status")
		assert.Equal(
			t,
			domain.MemoStatusProcessing,
			initialMemoStatus,
			"Memo should start in processing state",
		)

		// --- Recovery Phase ---
		t.Log("Setting up second application instance to trigger recovery...")

		// Create mocks for the second instance
		mockGenerator2 := NewRecoveryMockGenerator(logger, false, 200*time.Millisecond)
		mockCardRepo2 := NewRecoveryMockCardRepository(logger)
		taskConfig2 := getTestTaskConfigWithWorkers(2)

		// Setup second app instance
		_, taskRunner2, err := setupRecoveryTestInstance(
			t,
			tx,
			mockGenerator2,
			mockCardRepo2,
			taskConfig2,
		)
		require.NoError(t, err, "Failed to set up second app instance")

		// Start the runner - this triggers recovery
		t.Log("Starting second task runner - triggering recovery process...")
		err = taskRunner2.Start()
		require.NoError(t, err, "Failed to start the second task runner")
		defer taskRunner2.Stop()

		// --- Verification Phase ---
		t.Log("Verifying task completion after recovery...")

		// Wait for the task to be completed
		waitForRecoveryCondition(t, 15*time.Second, 200*time.Millisecond, func() (bool, error) {
			status, err := getTaskStatusDirectly(t, tx, taskID)
			if err != nil {
				return false, err
			}
			t.Logf("Current task status: %s", status)
			return status == task.TaskStatusCompleted, nil
		}, "task to complete after recovery")

		// Wait for the memo status to be updated to completed
		waitForRecoveryCondition(t, 8*time.Second, 200*time.Millisecond, func() (bool, error) {
			status, err := getMemoStatusDirectly(t, tx, memoID)
			if err != nil {
				return false, err
			}
			t.Logf("Current memo status: %s", status)
			return status == domain.MemoStatusCompleted, nil
		}, "memo to complete after recovery")

		// Verify execution count of the generator
		assert.Equal(
			t,
			1,
			mockGenerator2.GetExecutionCount(),
			"Generator should be executed exactly once",
		)

		// Verify cards were created
		createdCards := mockCardRepo2.GetCreatedCards(memoID.String())
		assert.NotEmpty(t, createdCards, "Cards should have been created after recovery")
		assert.Len(t, createdCards, 2, "Expected 2 cards to be created")
	})
}

// TestTaskRecovery_Failure tests recovery where the task fails after being recovered
func TestTaskRecovery_Failure(t *testing.T) {
	// Skip if no test database available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	testutils.WithTx(t, testDB, func(dbtx store.DBTX) {
		tx := dbtx.(*sql.Tx)
		logger := slog.Default()

		// --- Setup Phase ---
		t.Log("Setting up test environment for recovery failure test...")

		// Create a test user
		userID := uuid.New()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		_, err := tx.Exec("INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)",
			userID, "recovery-failure@example.com", string(hashedPassword))
		require.NoError(t, err, "Failed to create test user")

		// Create a memo
		memoID := uuid.New()
		memoText := "Memo for recovery failure test"

		// Create a memo with specific ID and status
		memo := &domain.Memo{
			ID:        memoID,
			UserID:    userID,
			Text:      memoText,
			Status:    domain.MemoStatusPending,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		memoStore := postgres.NewPostgresMemoStore(tx, logger)
		err = memoStore.Create(context.Background(), memo)
		require.NoError(t, err, "Failed to create test memo")

		// Create a task that's in 'processing' status
		taskID := uuid.New()
		taskPayload := map[string]string{
			"memo_id": memoID.String(),
			"user_id": userID.String(),
		}
		payloadBytes, err := json.Marshal(taskPayload)
		require.NoError(t, err, "Failed to marshal task payload")

		now := time.Now().UTC()
		_, err = tx.Exec(
			"INSERT INTO tasks (id, type, payload, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
			taskID,
			task.TaskTypeMemoGeneration,
			payloadBytes,
			string(task.TaskStatusProcessing),
			now,
			now,
		)
		require.NoError(t, err, "Failed to insert task with processing status")

		// Update memo status to match the task
		_, err = tx.Exec(
			"UPDATE memos SET status = $1 WHERE id = $2",
			string(domain.MemoStatusProcessing), memoID,
		)
		require.NoError(t, err, "Failed to update memo status to processing")

		// --- Recovery Phase ---
		t.Log("Setting up application instance to trigger recovery with error...")

		// Create mocks configured to fail
		mockGenerator := NewRecoveryMockGenerator(logger, true, 100*time.Millisecond)
		mockCardRepo := NewRecoveryMockCardRepository(logger)
		taskConfig := getDefaultTestTaskConfig()

		// Setup app instance
		_, taskRunner, err := setupRecoveryTestInstance(
			t,
			tx,
			mockGenerator,
			mockCardRepo,
			taskConfig,
		)
		require.NoError(t, err, "Failed to set up app instance")

		// Start the runner to trigger recovery
		t.Log("Starting task runner - triggering recovery process...")
		err = taskRunner.Start()
		require.NoError(t, err, "Failed to start task runner")
		defer taskRunner.Stop()

		// --- Verification Phase ---
		t.Log("Verifying task failure after recovery...")

		// Wait for the task to be marked as failed
		waitForRecoveryCondition(t, 15*time.Second, 200*time.Millisecond, func() (bool, error) {
			status, err := getTaskStatusDirectly(t, tx, taskID)
			if err != nil {
				return false, err
			}
			t.Logf("Current task status: %s", status)
			return status == task.TaskStatusFailed, nil
		}, "task to fail after recovery")

		// Wait for the memo status to be updated to failed
		waitForRecoveryCondition(t, 8*time.Second, 200*time.Millisecond, func() (bool, error) {
			status, err := getMemoStatusDirectly(t, tx, memoID)
			if err != nil {
				return false, err
			}
			t.Logf("Current memo status: %s", status)
			return status == domain.MemoStatusFailed, nil
		}, "memo to fail after recovery")

		// Verify execution count
		assert.Equal(
			t,
			1,
			mockGenerator.GetExecutionCount(),
			"Generator should be executed exactly once",
		)

		// Verify no cards were created
		createdCards := mockCardRepo.GetCreatedCards(memoID.String())
		assert.Empty(t, createdCards, "No cards should be created when generator fails")

		// Verify task has an error message
		var errorMsg string
		err = tx.QueryRow("SELECT error_message FROM tasks WHERE id = $1", taskID).Scan(&errorMsg)
		require.NoError(t, err, "Failed to query error message")
		assert.Contains(t, errorMsg, "generation failed", "Task should have error message set")
	})
}

// TestTaskRecovery_API tests end-to-end recovery using the API
func TestTaskRecovery_API(t *testing.T) {
	// Skip if no test database available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	testutils.WithTx(t, testDB, func(dbtx store.DBTX) {
		tx := dbtx.(*sql.Tx)
		logger := slog.Default()

		// --- Setup Phase ---
		t.Log("Setting up initial API instance...")

		// Create mocks for the first instance
		mockGenerator1 := NewRecoveryMockGenerator(logger, false, 0)
		mockCardRepo1 := NewRecoveryMockCardRepository(logger)
		taskConfig1 := getDefaultTestTaskConfig()

		// Setup first app instance
		server1, _, err := setupRecoveryTestInstance(
			t,
			tx,
			mockGenerator1,
			mockCardRepo1,
			taskConfig1,
		)
		require.NoError(t, err, "Failed to set up first app instance")
		defer server1.Close()

		// Register a test user via API
		userEmail := "api-recovery-test@example.com"
		userPassword := "securepassword123"
		registerPayload := map[string]interface{}{
			"email":    userEmail,
			"password": userPassword,
		}
		registerBody, err := json.Marshal(registerPayload)
		require.NoError(t, err, "Failed to marshal register payload")

		// Send registration request
		registerResp, err := http.Post(
			server1.URL+"/api/auth/register",
			"application/json",
			bytes.NewBuffer(registerBody),
		)
		require.NoError(t, err, "Failed to send register request")
		defer func() {
			if err := registerResp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		require.Equal(t, http.StatusCreated, registerResp.StatusCode, "Register should succeed")

		// Parse registration response
		var registerData map[string]interface{}
		err = json.NewDecoder(registerResp.Body).Decode(&registerData)
		require.NoError(t, err, "Failed to decode register response")
		require.NotEmpty(t, registerData["user_id"], "User ID should be returned")
		require.NotEmpty(t, registerData["token"], "Token should be returned")

		// Get user ID and token from response
		token := registerData["token"].(string)

		// Log the userID for debugging
		t.Logf("Created user with ID: %s", registerData["user_id"].(string))

		// Create a memo but DO NOT start the task runner
		// This simulates server shutdown before task processing
		memoText := "API test memo for recovery mechanism"
		memoPayload := map[string]interface{}{
			"text": memoText,
		}
		memoBody, err := json.Marshal(memoPayload)
		require.NoError(t, err, "Failed to marshal memo payload")

		// Create HTTP request for memo creation
		memoReq, err := http.NewRequest(
			"POST",
			server1.URL+"/api/memos",
			bytes.NewBuffer(memoBody),
		)
		require.NoError(t, err, "Failed to create memo request")
		memoReq.Header.Set("Content-Type", "application/json")
		memoReq.Header.Set("Authorization", "Bearer "+token)

		// Send memo creation request
		client := &http.Client{}
		memoResp, err := client.Do(memoReq)
		require.NoError(t, err, "Failed to send memo request")
		defer func() {
			if err := memoResp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Verify response
		require.Equal(
			t,
			http.StatusAccepted,
			memoResp.StatusCode,
			"Memo creation should be accepted",
		)

		// Parse memo response
		var memoData api.MemoResponse
		err = json.NewDecoder(memoResp.Body).Decode(&memoData)
		require.NoError(t, err, "Failed to decode memo response")

		memoID := memoData.ID
		require.NotEmpty(t, memoID, "Memo ID should be returned")
		assert.Equal(
			t,
			string(domain.MemoStatusPending),
			memoData.Status,
			"Initial status should be pending",
		)

		// Simulate server shutdown by closing the resources
		server1.Close()

		// But before the second instance, manually update task status to 'processing'
		// to simulate a crash during processing
		// Simulate crash during processing: Manually update task and memo status to 'processing'
		// before starting the recovery instance. This simulates a situation where the server crashed
		// or was shut down while tasks were in the middle of being processed, leaving them in an
		// inconsistent state that should be recovered by the task recovery mechanism.
		taskID, err := getTaskIDForMemo(t, tx, uuid.MustParse(memoID))
		require.NoError(t, err, "Failed to get task ID for memo")

		_, err = tx.Exec(
			"UPDATE tasks SET status = $1 WHERE id = $2",
			string(task.TaskStatusProcessing), taskID,
		)
		require.NoError(t, err, "Failed to update task status to processing")

		_, err = tx.Exec(
			"UPDATE memos SET status = $1 WHERE id = $2",
			string(domain.MemoStatusProcessing), memoID,
		)
		require.NoError(t, err, "Failed to update memo status to processing")

		// --- Recovery Phase ---
		t.Log("Setting up second API instance to trigger recovery...")

		// Create mocks for the second instance
		mockGenerator2 := NewRecoveryMockGenerator(logger, false, 200*time.Millisecond)
		mockCardRepo2 := NewRecoveryMockCardRepository(logger)
		taskConfig2 := getTestTaskConfigWithWorkers(2)

		// Setup second app instance
		server2, taskRunner2, err := setupRecoveryTestInstance(
			t,
			tx,
			mockGenerator2,
			mockCardRepo2,
			taskConfig2,
		)
		require.NoError(t, err, "Failed to set up second app instance")
		defer server2.Close()

		// Start the runner to trigger recovery
		t.Log("Starting second task runner - triggering recovery process...")
		err = taskRunner2.Start()
		require.NoError(t, err, "Failed to start the second task runner")
		defer taskRunner2.Stop()

		// --- Verification Phase ---
		t.Log("Verifying task completion after recovery...")

		// Wait for the task to be completed
		waitForRecoveryCondition(t, 15*time.Second, 200*time.Millisecond, func() (bool, error) {
			status, err := getTaskStatusDirectly(t, tx, taskID)
			if err != nil {
				return false, err
			}
			t.Logf("Current task status: %s", status)
			return status == task.TaskStatusCompleted, nil
		}, "task to complete after recovery")

		// Wait for the memo status to be updated to completed
		waitForRecoveryCondition(t, 8*time.Second, 200*time.Millisecond, func() (bool, error) {
			status, err := getMemoStatusDirectly(t, tx, uuid.MustParse(memoID))
			if err != nil {
				return false, err
			}
			t.Logf("Current memo status: %s", status)
			return status == domain.MemoStatusCompleted, nil
		}, "memo to complete after recovery")

		// Verify execution count
		assert.Equal(
			t,
			1,
			mockGenerator2.GetExecutionCount(),
			"Generator should be executed exactly once",
		)

		// Verify cards were created
		createdCards := mockCardRepo2.GetCreatedCards(memoID)
		assert.NotEmpty(t, createdCards, "Cards should have been created after recovery")
		assert.Len(t, createdCards, 2, "Expected 2 cards to be created")

		// Verify card count in database
		var cardCount int
		err = tx.QueryRow("SELECT COUNT(*) FROM cards WHERE memo_id = $1", memoID).Scan(&cardCount)
		require.NoError(t, err, "Failed to count cards in database")
		assert.Equal(t, 2, cardCount, "Two cards should exist in the database")
	})
}
