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

- [ ] **T053:** Enhance Error and Log Messages
  - **Action:** Use more specific error messages for token generation failures and improve logging for refresh token operations.
  - **Files:** `internal/api/auth_handler.go`, `internal/service/auth/jwt_service_impl.go`
  - **Complexity:** Low

- [ ] **T054:** Refactor Large Tests
  - **Action:** Split the large `TestRefreshTokenSuccess` test into smaller focused tests for login and refresh flows.
  - **Files:** `internal/api/auth_handler_test.go`
  - **Complexity:** Medium

- [ ] **T055:** Improve Configuration Documentation
  - **Action:** Add comments explaining the relationship between access and refresh token lifetimes in the configuration.
  - **Files:** `config.yaml.example`
  - **Complexity:** Low

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

- [ ] **T102:** Implement `TaskQueue` struct and methods
  - **Action:** Create the `TaskQueue` struct with a buffered channel for tasks and a logger. Implement `NewTaskQueue` constructor, `Enqueue(task Task) error`, `Close()`, and `GetChannel() <-chan Task` methods. Handle errors for full queues and closed channels.
  - **Depends On:** [T100]
  - **Complexity:** Medium

- [ ] **T103:** Implement `WorkerPool` struct
  - **Action:** Create `WorkerPool` struct with fields for task queue reader, worker count, wait group, shutdown context, cancel function, and logger. Implement constructor.
  - **Depends On:** [T100]
  - **Complexity:** Medium

- [ ] **T104:** Implement worker loop with error handling
  - **Action:** Implement the worker goroutines that consume tasks from the queue. Add proper error handling, context cancellation, and panic recovery. Ensure clean shutdown.
  - **Depends On:** [T102, T103]
  - **Complexity:** High

- [ ] **T105:** Implement `WorkerPool.Start()` and `WorkerPool.Stop()` methods
  - **Action:** Implement methods to start worker goroutines and gracefully stop them, handling proper synchronization via WaitGroup.
  - **Depends On:** [T104]
  - **Complexity:** Medium

- [ ] **T106:** Create `MemoGenerationTask` struct
  - **Action:** Implement the struct with fields for `memoID`, `memoRepo`, `generator`, and `logger`. Add constructor method.
  - **Depends On:** [T100]
  - **Complexity:** Low

- [ ] **T107:** Implement `MemoGenerationTask.Execute()` method
  - **Action:** Implement core flashcard generation logic: Update memo status to 'processing', call generator service, save generated cards to DB, handle errors, and update final memo status. Add structured logging.
  - **Depends On:** [T106]
  - **Complexity:** High

- [ ] **T108:** Create database migration for `tasks` table (if needed)
  - **Action:** Create SQL migration file to define the `tasks` table schema with fields for ID, type, payload, status, error messages, timestamps, etc.
  - **Depends On:** None
  - **Complexity:** Low

- [ ] **T109:** Create task store interface
  - **Action:** Define the `TaskStore` interface with methods for saving, updating, and retrieving tasks from the database.
  - **Depends On:** [T100, T101]
  - **Complexity:** Low

- [ ] **T110:** Implement PostgreSQL task store
  - **Action:** Implement the `TaskStore` interface for PostgreSQL with methods to persist and retrieve tasks.
  - **Depends On:** [T108, T109]
  - **Complexity:** Medium

- [ ] **T111:** Implement recovery mechanism
  - **Action:** Create the `runRecoveryTasks` function to find 'processing' memos on startup and re-enqueue them for processing. Add appropriate logging and error handling.
  - **Depends On:** [T107, T110]
  - **Complexity:** Medium

- [ ] **T112:** Integrate task runner into application startup
  - **Action:** Modify `cmd/server/main.go` to initialize the task queue, worker pool, and recovery mechanism during server startup.
  - **Depends On:** [T105, T111]
  - **Complexity:** Medium

- [ ] **T113:** Add graceful shutdown for task runner
  - **Action:** Update server shutdown logic to stop the worker pool gracefully, ensuring in-progress tasks complete.
  - **Depends On:** [T112]
  - **Complexity:** Low

- [ ] **T114:** Integrate with memo creation endpoint
  - **Action:** Update the memo creation endpoint to save memos with 'pending' status and enqueue generation tasks instead of processing synchronously.
  - **Depends On:** [T107, T112]
  - **Complexity:** Medium

- [ ] **T115:** Add stuck task monitoring
  - **Action:** Implement a background process to identify tasks stuck in 'processing' state for too long and retry them.
  - **Depends On:** [T110, T112]
  - **Complexity:** Medium

- [ ] **T116:** Update configuration system
  - **Action:** Add task runner configuration options (worker count, queue size, retry intervals, etc.) to the application configuration.
  - **Depends On:** [T112]
  - **Complexity:** Low

- [ ] **T117:** Write unit tests for TaskQueue
  - **Action:** Test enqueueing tasks, handling full queues, and closed channels.
  - **Depends On:** [T102]
  - **Complexity:** Medium

- [ ] **T118:** Write unit tests for WorkerPool
  - **Action:** Test starting, stopping, task processing, and panic recovery.
  - **Depends On:** [T105]
  - **Complexity:** Medium

- [ ] **T119:** Write unit tests for MemoGenerationTask
  - **Action:** Test success paths, error handling, and mocked dependencies.
  - **Depends On:** [T107]
  - **Complexity:** Medium

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
