// Package task manages background job queuing, processing, and lifecycle.
// It provides mechanisms for asynchronous execution of long-running operations
// like generating cards from memos, ensuring they don't block HTTP request
// handling and can recover from application restarts.
package task
