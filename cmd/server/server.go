//go:build exported_core_functions

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// startHTTPServer starts the HTTP server with graceful shutdown support.
// It takes a context that can be used to signal cancellation and the router.
// Returns an error if the server fails to start or encounters problems.
func (app *application) startHTTPServer(ctx context.Context, router http.Handler) error {
	// Configure and create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.config.Server.Port),
		Handler: router,
	}

	// Create a context for graceful shutdown
	serverCtx, cancelServer := context.WithCancel(ctx)
	defer cancelServer()

	// Set up graceful shutdown with signal handling
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine to allow for graceful shutdown
	go func() {
		app.logger.Info("Starting server", "port", app.config.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.logger.Error("Server failed", "error", err)
			cancelServer() // Signal the server context to cancel
		}
	}()

	// Wait for shutdown signal or context cancellation
	select {
	case <-shutdownCh:
		app.logger.Info("Shutting down server...")
	case <-serverCtx.Done():
		app.logger.Info("Server context canceled, shutting down...")
	}

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		app.logger.Error("Server shutdown failed", "error", err)
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// Run application cleanup
	app.cleanup()

	app.logger.Info("Server shutdown completed")
	return nil
}
