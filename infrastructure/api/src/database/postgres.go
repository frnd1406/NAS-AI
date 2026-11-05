package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/nas-ai/api/src/config"
	"github.com/sirupsen/logrus"
)

// DB holds the database connection pool
type DB struct {
	*sql.DB
	logger *logrus.Logger
}

// NewPostgresConnection creates a new PostgreSQL connection pool
// CRITICAL: Fails fast if connection cannot be established (Phase 1 principle!)
func NewPostgresConnection(cfg *config.Config, logger *logrus.Logger) (*DB, error) {
	logger.WithFields(logrus.Fields{
		"host": cfg.DatabaseHost,
		"port": cfg.DatabasePort,
		"db":   cfg.DatabaseName,
	}).Info("Connecting to PostgreSQL...")

	// Open database connection
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)

	if d, err := time.ParseDuration(cfg.DBConnMaxLifetime); err == nil {
		db.SetConnMaxLifetime(d)
	} else {
		logger.Warnf("Invalid DBConnMaxLifetime '%s', using default 5m", cfg.DBConnMaxLifetime)
		db.SetConnMaxLifetime(5 * time.Minute)
	}

	if d, err := time.ParseDuration(cfg.DBConnMaxIdleTime); err == nil {
		db.SetConnMaxIdleTime(d)
	} else {
		logger.Warnf("Invalid DBConnMaxIdleTime '%s', using default 10m", cfg.DBConnMaxIdleTime)
		db.SetConnMaxIdleTime(10 * time.Minute)
	}

	// CRITICAL: Fail-fast - Verify connection works
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("CRITICAL: failed to ping database (fail-fast): %w", err)
	}

	logger.Info("✅ PostgreSQL connection established")

	return &DB{
		DB:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() error {
	db.logger.Info("Closing PostgreSQL connection...")
	return db.DB.Close()
}

// HealthCheck verifies the database connection is still alive
func (db *DB) HealthCheck(ctx context.Context) error {
	if err := db.PingContext(ctx); err != nil {
		db.logger.WithError(err).Error("PostgreSQL health check failed")
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}
