package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/phrazzld/scry-api/internal/config"
	"github.com/phrazzld/scry-api/internal/platform/postgres"
	"github.com/phrazzld/scry-api/internal/store"
	"github.com/phrazzld/scry-api/internal/task"
	"github.com/phrazzld/scry-api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDB is a package-level variable that holds a shared database connection
// for all tests in this package.
var testDB *sql.DB

// TestMain sets up the database and runs all tests once, rather than for each test.
// This improves performance by running migrations only once for all tests.
func TestMain(m *testing.M) {
	// Check if we can run the integration tests with a database
	dbAvailable := false

	// Only try to connect if DATABASE_URL is set
	if testutils.IsIntegrationTestEnvironment() {
		dbURL := os.Getenv("DATABASE_URL")
		var err error
		testDB, err = sql.Open("pgx", dbURL)
		if err == nil {
			// Set connection parameters
			testDB.SetMaxOpenConns(5)
			testDB.SetMaxIdleConns(5)
			testDB.SetConnMaxLifetime(5 * time.Minute)

			// Try to ping the database with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := testDB.PingContext(ctx); err == nil {
				// Database is available, set up schema
				if err := testutils.SetupTestDatabaseSchema(testDB); err == nil {
					// Create migrations config
					cfg := &config.Config{
						Database: config.DatabaseConfig{
							URL: dbURL,
						},
					}

					// Get the project root directory for migrations
					_, thisFile, _, ok := runtime.Caller(0)
					if !ok {
						fmt.Println("Failed to get current file path from runtime.Caller - skipping integration tests")
					} else {
						// Get the directory containing this file (cmd/server)
						thisDir := filepath.Dir(thisFile)

						// Go up two levels: from cmd/server to project root
						projectRoot := filepath.Dir(filepath.Dir(thisDir))

						// Save current working directory
						origWD, err := os.Getwd()
						if err != nil {
							fmt.Printf("Failed to get current working directory: %v - skipping integration tests\n", err)
						} else {
							// Change to project root
							if err := os.Chdir(projectRoot); err != nil {
								fmt.Printf("Failed to change working directory to project root: %v - skipping integration tests\n", err)
							} else {
								// Run migrations to ensure all tables exist including tasks table
								if err := runMigrations(cfg, "up"); err != nil {
									fmt.Printf("Failed to run migrations: %v - skipping integration tests\n", err)
								} else {
									dbAvailable = true
									fmt.Println("Database connection successful - running integration tests")
								}

								// Restore working directory
								if err := os.Chdir(origWD); err != nil {
									fmt.Printf("Warning: Failed to restore working directory: %v\n", err)
								}
							}
						}
					}
				} else {
					fmt.Printf("Database schema setup failed: %v - skipping integration tests\n", err)
					if testDB != nil {
						if err := testDB.Close(); err != nil {
							fmt.Printf("Warning: Failed to close database connection: %v\n", err)
						}
					}
				}
			} else {
				fmt.Printf("Database ping failed: %v - skipping integration tests\n", err)
				if testDB != nil {
					if err := testDB.Close(); err != nil {
						fmt.Printf("Warning: Failed to close database connection: %v\n", err)
					}
				}
			}
		} else {
			fmt.Printf("Database connection failed: %v - skipping integration tests\n", err)
		}
	} else {
		fmt.Println("DATABASE_URL not set - skipping integration tests")
	}

	// If database is not available, set a flag that tests can check
	if !dbAvailable {
		// Make sure testDB is nil so tests can check this
		testDB = nil
	}

	// Run all tests
	exitCode := m.Run()

	// Clean up
	if testDB != nil {
		if err := testDB.Close(); err != nil {
			fmt.Printf("CRITICAL: Failed to close database connection in TestMain: %v\n", err)
		}
	}

	os.Exit(exitCode)
}

// customTestTask implements task.Task for testing
type customTestTask struct {
	id       uuid.UUID
	payload  []byte
	status   task.TaskStatus
	executed bool
	mu       sync.Mutex
}

func newCustomTestTask() *customTestTask {
	return &customTestTask{
		id:       uuid.New(),
		payload:  []byte(`{"test":"data"}`),
		status:   task.TaskStatusPending,
		executed: false,
	}
}

func (t *customTestTask) ID() uuid.UUID {
	return t.id
}

func (t *customTestTask) Type() string {
	return "custom_test_task"
}

func (t *customTestTask) Payload() []byte {
	return t.payload
}

func (t *customTestTask) Status() task.TaskStatus {
	return t.status
}

func (t *customTestTask) Execute(ctx context.Context) error {
	// Mark as executed and simulate processing
	t.mu.Lock()
	t.executed = true
	t.mu.Unlock()

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (t *customTestTask) WasExecuted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.executed
}

// TestTaskRunnerIntegration tests that the task runner can be initialized,
// started, and tasks can be submitted and processed.
func TestTaskRunnerIntegration(t *testing.T) {
	// Don't run in parallel to avoid database table conflicts
	// t.Parallel()

	// Skip if database is not available
	if testDB == nil {
		t.Skip("Skipping integration test - database connection not available")
	}

	// Get the database URL for this test
	dbURL := os.Getenv("DATABASE_URL")

	// First, ensure that all migrations have been run so we have the required tables
	// This is needed because TestMigrationFlow might have run down migrations
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL: dbURL,
		},
	}

	// Get the project root directory for migrations
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get current file path from runtime.Caller")
	}

	// Get the directory containing this file (cmd/server)
	thisDir := filepath.Dir(thisFile)

	// Go up two levels: from cmd/server to project root
	projectRoot := filepath.Dir(filepath.Dir(thisDir))

	// Change to project root for migrations to work
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWD); err != nil {
			t.Logf("Warning: Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change working directory to project root: %v", err)
	}

	// Run migrations to ensure all tables exist
	if err := runMigrations(cfg, "up"); err != nil &&
		!strings.Contains(err.Error(), "no migrations to run") {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Restore working directory
	if err := os.Chdir(origWD); err != nil {
		t.Fatalf("Failed to restore working directory: %v", err)
	}

	testutils.WithTx(t, testDB, func(tx store.DBTX) {
		// Set up the configuration
		cfg := &config.Config{
			Task: config.TaskConfig{
				WorkerCount:         2,
				QueueSize:           10,
				StuckTaskAgeMinutes: 30,
			},
		}

		// Set up the task store
		taskStore := postgres.NewPostgresTaskStore(tx)

		// Set up the task runner
		taskRunner := task.NewTaskRunner(taskStore, task.TaskRunnerConfig{
			WorkerCount:  cfg.Task.WorkerCount,
			QueueSize:    cfg.Task.QueueSize,
			StuckTaskAge: time.Duration(cfg.Task.StuckTaskAgeMinutes) * time.Minute,
		}, slog.Default())

		// Start the task runner
		err := taskRunner.Start()
		require.NoError(t, err, "Failed to start task runner")

		// Create and submit a test task
		testTask := newCustomTestTask()
		err = taskRunner.Submit(context.Background(), testTask)
		require.NoError(t, err, "Failed to submit task")

		// Wait for the task to be processed
		startTime := time.Now()
		for !testTask.WasExecuted() && time.Since(startTime) < 2*time.Second {
			time.Sleep(50 * time.Millisecond)
		}

		// Check that the task was executed
		assert.True(t, testTask.WasExecuted(), "Task should have been executed")

		// Stop the task runner
		taskRunner.Stop()
	})
}
