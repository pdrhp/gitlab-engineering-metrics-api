package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Config defines the configuration needed for database connection
type Config interface {
	GetDSN() string
	GetMaxOpenConns() int
	GetMaxIdleConns() int
	GetConnMaxLifetime() time.Duration
}

// New creates a new database connection with the given configuration
func New(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.GetMaxOpenConns())
	db.SetMaxIdleConns(cfg.GetMaxIdleConns())
	db.SetConnMaxLifetime(cfg.GetConnMaxLifetime())

	// Test connection with initial ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
