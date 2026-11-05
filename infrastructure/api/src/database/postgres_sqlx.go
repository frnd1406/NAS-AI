package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/nas-ai/api/src/config"
)

// DBX holds the sqlx database connection pool
// Provides struct scanning and named queries
type DBX struct {
	*sqlx.DB
	logger *slog.Logger
}

// NewPostgresConnectionX creates a new PostgreSQL connection pool using sqlx
// CRITICAL: Fails fast if connection cannot be established
func NewPostgresConnectionX(cfg *config.Config, logger *slog.Logger) (*DBX, error) {
	logger.Info("Connecting to PostgreSQL...",
		slog.String("host", cfg.DatabaseHost),
		slog.String("port", cfg.DatabasePort),
		slog.String("db", cfg.DatabaseName),
	)

	// Open database connection with sqlx
	db, err := sqlx.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// CRITICAL: Fail-fast - Verify connection works
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("CRITICAL: failed to ping database (fail-fast): %w", err)
	}

	logger.Info("✅ PostgreSQL connection established")

	return &DBX{
		DB:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection pool
func (db *DBX) Close() error {
	db.logger.Info("Closing PostgreSQL connection...")
	return db.DB.Close()
}

// HealthCheck verifies the database connection is still alive
func (db *DBX) HealthCheck(ctx context.Context) error {
	if err := db.PingContext(ctx); err != nil {
		db.logger.Error("PostgreSQL health check failed", slog.String("error", err.Error()))
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}

// Transaction helper for executing multiple queries in a transaction
type TxFunc func(*sqlx.Tx) error

// WithTransaction executes a function within a database transaction
// Automatically handles commit/rollback
func (db *DBX) WithTransaction(ctx context.Context, fn TxFunc) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // Re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			db.logger.Error("Failed to rollback transaction",
				slog.String("error", err.Error()),
				slog.String("rollback_error", rbErr.Error()),
			)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
