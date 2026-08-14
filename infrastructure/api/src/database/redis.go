package database

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/nas-ai/api/src/config"
	"github.com/sirupsen/logrus"
)

// RedisClient wraps redis.Client with logging
type RedisClient struct {
	*redis.Client
	logger *logrus.Logger
}

// NewRedisConnection creates a new Redis connection
// CRITICAL: Fails fast if connection cannot be established (Phase 1 principle!)
func NewRedisConnection(cfg *config.Config, logger *logrus.Logger) (*RedisClient, error) {
	logger.WithFields(logrus.Fields{
		"host": cfg.RedisHost,
		"port": cfg.RedisPort,
	}).Info("Connecting to Redis...")

	// Build client options
	opts := &redis.Options{
		Addr:         cfg.RedisURL,
		Password:     "", // No password for dev
		DB:           0,  // Default DB
		DialTimeout:  10 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	// Enable TLS to protect Redis traffic from in-network sniffing.
	// RedisTLSSkipVerify allows self-signed certs on trusted internal networks.
	if cfg.RedisTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.RedisTLSSkipVerify, //nolint:gosec // opt-in for internal self-signed certs
		}
		logger.WithField("skip_verify", cfg.RedisTLSSkipVerify).Info("Redis: TLS enabled")
	}

	// Create Redis client
	client := redis.NewClient(opts)

	// CRITICAL: Fail-fast - Verify connection works
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("CRITICAL: failed to ping Redis (fail-fast): %w", err)
	}

	logger.Info("✅ Redis connection established")

	return &RedisClient{
		Client: client,
		logger: logger,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	r.logger.Info("Closing Redis connection...")
	return r.Client.Close()
}

// HealthCheck verifies the Redis connection is still alive
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	if err := r.Client.Ping(ctx).Err(); err != nil {
		r.logger.WithError(err).Error("Redis health check failed")
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}
