//go:build exported_core_functions

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/phrazzld/scry-api/internal/config"
)

// setupAppDatabase establishes a connection to the database and configures connection pools.
// Returns the database connection if successful, or an error if the connection fails.
func setupAppDatabase(cfg *config.Config, logger *slog.Logger) (*sql.DB, error) {
	// Open database connection
	db, err := sql.Open("pgx", cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool with reasonable defaults
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established")
	return db, nil
}
