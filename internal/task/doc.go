// Package task manages background job queuing, processing, and lifecycle.
// It provides mechanisms for asynchronous execution of long-running operations
// like generating cards from memos, ensuring they don't block HTTP request
// handling and can recover from application restarts.
//
// The task package implements an asynchronous processing system for background
// operations in the Scry application. It separates time-consuming processes
// from the synchronous HTTP request/response flow, improving application
// responsiveness and resilience.
//
// Key components:
//
// 1. Task Queue:
//   - In-memory queue for background job storage and processing
//   - Ensure tasks are preserved during application lifecycle
//
// 2. Worker Pool:
//   - Concurrent execution of background tasks using goroutines
//   - Controlled parallelism with configurable worker count
//
// 3. Task Types:
//   - GenerateCardsTask: Processes memos to generate flashcards using LLM/AI
//   - (Future) Other task types for background operations
//
// 4. Resilience Features:
//   - Recovery mechanism for incomplete tasks after application restart
//   - Retry logic for transient failures
//   - Dead-letter queue for persistently failing tasks
//
// 5. Monitoring:
//   - Task status tracking (pending, processing, completed, failed)
//   - Error logging and collection for failed tasks
//
// The task package depends on domain entities and application services but
// isolates the complexity of concurrent processing, providing a simple interface
// for enqueueing jobs and retrieving their results.
package task
