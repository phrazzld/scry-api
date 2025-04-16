# TODO List

## JWT Refresh Token Refactoring

- [x] **T050:** Consolidate Refresh Token Tests
  - **Action:** Remove the redundant `internal/api/refresh_token_test.go` file and ensure all test cases are covered in `auth_handler_test.go`.
  - **Files:** `internal/api/refresh_token_test.go`, `internal/api/auth_handler_test.go`
  - **Complexity:** Low

- [x] **T051:** Extract Token Generation Logic
  - **Action:** Refactor the duplicated token generation code in `Register` and `Login` handlers into a private helper method.
  - **Files:** `internal/api/auth_handler.go`
  - **Complexity:** Low

- [x] **T052:** Improve Time Handling in Auth Handler
  - **Action:** Inject a time source into `AuthHandler` for calculating response `ExpiresAt` instead of using `time.Now()` directly.
  - **Files:** `internal/api/auth_handler.go`
  - **Complexity:** Medium

## Additional Improvements

- [x] **T053:** Enhance Error and Log Messages
  - **Action:** Use more specific error messages for token generation failures and improve logging for refresh token operations.
  - **Files:** `internal/api/auth_handler.go`, `internal/service/auth/jwt_service_impl.go`
  - **Complexity:** Low
  - **Note:** Added more detailed error messages in token generation functions with context on what operation was being performed. Enhanced logging throughout authentication flow with consistent structured logging including token type, expiry information, and operation context. Added success logging at debug level for token generation and validation operations. Improved error specific handling in JWT validation flows to provide more accurate diagnostic information.

- [x] **T054:** Refactor Large Tests
  - **Action:** Split the large `TestRefreshTokenSuccess` test into smaller focused tests for login and refresh flows.
  - **Files:** `internal/api/auth_handler_test.go`
  - **Complexity:** Medium
  - **Note:** Refactored `TestRefreshTokenSuccess` into three separate tests: `TestLoginWithTokenGeneration`, `TestRefreshTokenFlow`, and `TestCompleteAuthFlow`. Created a reusable `setupAuthTestEnvironment` helper function to reduce code duplication. Each test now focuses on a specific aspect of the authentication flow, making tests more maintainable and easier to understand while preserving the same test coverage.

- [x] **T055:** Improve Configuration Documentation
  - **Action:** Add comments explaining the relationship between access and refresh token lifetimes in the configuration.
  - **Files:** `config.yaml.example`
  - **Complexity:** Low
  - **Note:** Added comprehensive documentation to the config.yaml.example file that explains the relationship between access and refresh tokens, their security implications, and recommended values. Created a new 'ACCESS AND REFRESH TOKEN CONFIGURATION' section with detailed explanations of the dual-token approach, security considerations, and practical guidance on token lifetime settings.

## Asynchronous Task Runner Setup

- [x] **T100:** Create core interfaces in `internal/task/task.go`
  - **Action:** Define the `Task` interface with an `Execute(ctx context.Context) error` method. Define `TaskQueueReader` and `TaskQueueWriter` interfaces.
  - **Depends On:** None
  - **Complexity:** Low

- [x] **T101:** Define task status constants
  - **Action:** Define constants for task and memo statuses (e.g., `TaskStatusPending`, `TaskStatusProcessing`, `TaskStatusCompleted`, `TaskStatusFailed`, `MemoStatusPending`, etc.) in the appropriate locations.
  - **Depends On:** None
  - **Complexity:** Low
  - **Note:** Verified that all required status constants are already defined in the codebase: task statuses in `internal/task/task.go`, memo statuses in `internal/domain/memo.go`, and review outcomes in `internal/domain/user_card_stats.go`.

- [x] **T102:** Implement `TaskQueue` struct and methods
  - **Action:** Create the `TaskQueue` struct with a buffered channel for tasks and a logger. Implement `NewTaskQueue` constructor, `Enqueue(task Task) error`, `Close()`, and `GetChannel() <-chan Task` methods. Handle errors for full queues and closed channels.
  - **Depends On:** [T100]
  - **Complexity:** Medium
  - **Note:** Implemented in `internal/task/task_queue.go` with a comprehensive test suite in `internal/task/task_queue_test.go`. The implementation handles all required error cases (full queue, closed queue) and provides proper logging.

- [x] **T103:** Implement `WorkerPool` struct
  - **Action:** Create `WorkerPool` struct with fields for task queue reader, worker count, wait group, shutdown context, cancel function, and logger. Implement constructor.
  - **Depends On:** [T100]
  - **Complexity:** Medium
  - **Note:** Implemented in `internal/task/worker_pool.go` with a test suite in `internal/task/worker_pool_test.go`. The implementation includes a configurable worker count with validation, context for cancellation, and a settable error handler.

- [x] **T104:** Implement worker loop with error handling
  - **Action:** Implement the worker goroutines that consume tasks from the queue. Add proper error handling, context cancellation, and panic recovery. Ensure clean shutdown.
  - **Depends On:** [T102, T103]
  - **Complexity:** High
  - **Note:** Implemented in `internal/task/worker_pool.go` with `runWorker` and `processTask` methods. Includes panic recovery, context cancellation, and proper error handling with comprehensive test coverage. Worker loop separates concurrency control from task execution logic for improved clarity and maintainability.

- [x] **T105:** Implement `WorkerPool.Start()` and `WorkerPool.Stop()` methods
  - **Action:** Implement methods to start worker goroutines and gracefully stop them, handling proper synchronization via WaitGroup.
  - **Depends On:** [T104]
  - **Complexity:** Medium
  - **Note:** Implemented in `internal/task/worker_pool.go` alongside task T104. The `Start()` method launches worker goroutines and logs the start, while `Stop()` cancels the context, waits for all workers to finish via `WaitGroup`, and logs the completion. Both methods have comprehensive test coverage.

- [x] **T106:** Create `MemoGenerationTask` struct
  - **Action:** Implement the struct with fields for `memoID`, `memoRepo`, `generator`, and `logger`. Add constructor method.
  - **Depends On:** [T100]
  - **Complexity:** Low
  - **Note:** Implemented in `internal/task/memo_generation_task.go` with fields for memoID, repositories, generator, and logger. Added constructor with validation and implemented Task interface methods (ID, Type, Status, Payload). Full test coverage provided in `internal/task/memo_generation_task_test.go`.

- [x] **T107:** Implement `MemoGenerationTask.Execute()` method
  - **Action:** Implement core flashcard generation logic: Update memo status to 'processing', call generator service, save generated cards to DB, handle errors, and update final memo status. Add structured logging.
  - **Depends On:** [T106]
  - **Complexity:** High
  - **Note:** Implemented in `internal/task/memo_generation_task.go` with comprehensive error handling and proper status transitions. The implementation follows the complete lifecycle for memo generation: retrieve memo, update status to processing, generate cards, save to database, and finalize with appropriate status based on outcome. Includes structured logging throughout and handles edge cases like context cancellation, empty card generation results, and partial failures. Test coverage in `internal/task/memo_generation_task_test.go` includes happy path and various error cases.

- [x] **T108:** Create database migration for `tasks` table (if needed)
  - **Action:** Create SQL migration file to define the `tasks` table schema with fields for ID, type, payload, status, error messages, timestamps, etc.
  - **Depends On:** None
  - **Complexity:** Low
  - **Note:** The migration file already exists at `internal/platform/postgres/migrations/20250415000005_create_tasks_table.sql` with a complete schema including id, type, payload, status, error_message, and timestamps. The PostgreSQL task store implementation is also complete in `internal/platform/postgres/task_store.go` with comprehensive tests. The existing implementation fully meets the requirements of the TaskStore interface defined in `internal/task/task.go`.

- [x] **T109:** Create task store interface
  - **Action:** Define the `TaskStore` interface with methods for saving, updating, and retrieving tasks from the database.
  - **Depends On:** [T100, T101]
  - **Complexity:** Low
  - **Note:** The TaskStore interface is already defined in `internal/task/task.go` (lines 57-72) with methods for saving tasks, updating task status, and retrieving tasks by status. The interface is well-documented and includes all necessary methods for the task runner system.

- [x] **T110:** Implement PostgreSQL task store
  - **Action:** Implement the `TaskStore` interface for PostgreSQL with methods to persist and retrieve tasks.
  - **Depends On:** [T108, T109]
  - **Complexity:** Medium
  - **Note:** The PostgreSQL task store implementation already exists in `internal/platform/postgres/task_store.go` with a comprehensive test suite in `internal/platform/postgres/task_store_test.go`. The implementation includes methods for saving tasks, updating their status, and retrieving tasks by status with optional filtering by age. The code is well-structured with error handling and proper logging.

- [x] **T111:** Implement recovery mechanism
  - **Action:** Create the `runRecoveryTasks` function to find 'processing' memos on startup and re-enqueue them for processing. Add appropriate logging and error handling.
  - **Depends On:** [T107, T110]
  - **Complexity:** Medium
  - **Note:** Recovery mechanism is already implemented in `internal/task/runner.go` through the `Recover()` method (lines 126-184), which finds tasks in both "pending" and "processing" states, updates their status as needed, and requeues them for execution. This is accompanied by a comprehensive test suite in `internal/task/runner_test.go`. Additionally, a stuck task monitoring mechanism is implemented in `stuckTaskMonitor()` method to handle tasks that have been in the processing state for too long.

- [x] **T112:** Integrate task runner into application startup
  - **Action:** Modify `cmd/server/main.go` to initialize the task queue, worker pool, and recovery mechanism during server startup.
  - **Depends On:** [T105, T111]
  - **Complexity:** Medium
  - **Note:** Implemented in `cmd/server/main.go` with the `setupTaskRunner` function (lines 378-399). The function configures a new TaskRunner with appropriate worker count, queue size, and stuck task monitoring settings from the application config. The task runner is properly initialized during server startup in the `startServer` function, where it calls `setupTaskRunner`, starts the task runner, and sets up a proper deferred stop to ensure graceful shutdown.

- [x] **T113:** Add graceful shutdown for task runner
  - **Action:** Update server shutdown logic to stop the worker pool gracefully, ensuring in-progress tasks complete.
  - **Depends On:** [T112]
  - **Complexity:** Low
  - **Note:** Implemented in `cmd/server/main.go` with a deferred call to `taskRunner.Stop()` in the `startServer` function (line 271). This ensures that when the server receives a shutdown signal, the task runner is properly stopped, allowing in-progress tasks to complete before the application exits. The TaskRunner.Stop() method itself is implemented in `internal/task/runner.go` to handle proper cancellation and wait for all workers to finish.

- [ ] **T114:** Integrate with memo creation endpoint
  - **Action:** Update the memo creation endpoint to save memos with 'pending' status and enqueue generation tasks instead of processing synchronously.
  - **Depends On:** [T107, T112]
  - **Complexity:** Medium

- [x] **T115:** Add stuck task monitoring
  - **Action:** Implement a background process to identify tasks stuck in 'processing' state for too long and retry them.
  - **Depends On:** [T110, T112]
  - **Complexity:** Medium
  - **Note:** Stuck task monitoring is already implemented in `internal/task/runner.go` through the `stuckTaskMonitor()` method (lines 250-305), which runs as a goroutine and periodically checks for tasks that have been in the "processing" state for too long, resets their status to "pending", and requeues them for processing. The implementation includes configurable parameters for stuck task age and check interval, proper error handling, and comprehensive logging. Test coverage is provided in `internal/task/runner_test.go` through the `TestTaskRunner_StuckTasks` test function.

- [x] **T116:** Update configuration system
  - **Action:** Add task runner configuration options (worker count, queue size, retry intervals, etc.) to the application configuration.
  - **Depends On:** [T112]
  - **Complexity:** Low
  - **Note:** Task runner configuration options are already implemented in `internal/config/config.go` with the TaskConfig struct (lines 93-107), including WorkerCount, QueueSize, and StuckTaskAgeMinutes fields. The example configuration in `config.yaml.example` includes documentation and default values for these options. These configuration values are properly used in the `setupTaskRunner` function in `cmd/server/main.go` to initialize the task runner with the configured settings.

- [x] **T117:** Write unit tests for TaskQueue
  - **Action:** Test enqueueing tasks, handling full queues, and closed channels.
  - **Depends On:** [T102]
  - **Complexity:** Medium
  - **Note:** Complete test suite already exists in `internal/task/task_queue_test.go` covering task enqueueing, queue full conditions, closing the queue, and channel operations. Tests include both individual operations and concurrent access patterns.

- [x] **T118:** Write unit tests for WorkerPool
  - **Action:** Test starting, stopping, task processing, and panic recovery.
  - **Depends On:** [T105]
  - **Complexity:** Medium
  - **Note:** Comprehensive test suite implemented in `internal/task/worker_pool_test.go` testing worker pool creation, starting and stopping workers, task processing success and failure scenarios, panic recovery, and context cancellation handling.

- [x] **T119:** Write unit tests for MemoGenerationTask
  - **Action:** Test success paths, error handling, and mocked dependencies.
  - **Depends On:** [T107]
  - **Complexity:** Medium
  - **Note:** Extensive tests implemented in `internal/task/memo_generation_task_test.go` covering constructor validation, payload serialization, and the Execute method with various scenarios including success path, error handling for each step of the process, context cancellation, and edge cases.

- [ ] **T120:** Create integration tests for task lifecycle
  - **Action:** Create end-to-end tests for task submission, processing, and completion.
  - **Depends On:** [T114]
  - **Complexity:** High

- [ ] **T121:** Create integration tests for recovery mechanism
  - **Action:** Test the system's ability to recover 'processing' tasks after restart.
  - **Depends On:** [T111]
  - **Complexity:** Medium

## [!] CLARIFICATIONS NEEDED / ASSUMPTIONS

- [ ] **Assumption:** The `generator.Service` interface and its implementation will be created concurrently or before implementing `MemoGenerationTask`.
  - **Context:** `MemoGenerationTask` depends on a `generator.Service` to create flashcards from memos.

- [ ] **Assumption:** Memo, card, and user card stats repositories will be available when implementing the memo generation task.
  - **Context:** The task needs to interact with these repositories to update statuses and save generated cards.

- [ ] **Assumption:** Task persistence might require adjustment based on whether tasks need to survive application restarts or if recovery can work directly from memo status.
  - **Context:** The plan shows both approaches: recovering from memo statuses and potentially having a tasks table.
