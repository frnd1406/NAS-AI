package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
)

// TokenService handles verification and reset tokens
type TokenService struct {
	redis  *database.RedisClient
	logger *logrus.Logger
}

// NewTokenService creates a new token service
func NewTokenService(redis *database.RedisClient, logger *logrus.Logger) *TokenService {
	return &TokenService{
		redis:  redis,
		logger: logger,
	}
}

// GenerateVerificationToken generates a 32-byte random token for email verification
func (s *TokenService) GenerateVerificationToken(ctx context.Context, userID string) (string, error) {
	token, err := s.generateRandomToken()
	if err != nil {
		return "", err
	}

	// FIX [BUG-GO-016]: Add Redis timeout to prevent hanging on slow Redis
	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Store in Redis with 24-hour expiry
	key := "verify:" + token
	if err := s.redis.Set(redisCtx, key, userID, 24*time.Hour).Err(); err != nil {
		s.logger.WithError(err).Error("Failed to store verification token")
		return "", fmt.Errorf("failed to store verification token: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"token":   token[:8] + "...", // Log only first 8 chars
	}).Debug("Verification token generated")

	return token, nil
}

// ValidateVerificationToken validates and consumes a verification token
func (s *TokenService) ValidateVerificationToken(ctx context.Context, token string) (string, error) {
	// FIX [BUG-GO-016]: Add Redis timeout to prevent hanging on slow Redis
	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	key := "verify:" + token
	userID, err := s.redis.Get(redisCtx, key).Result()
	if err != nil {
		s.logger.WithError(err).Debug("Verification token not found or expired")
		return "", fmt.Errorf("invalid or expired token")
	}

	// Delete token (single-use) - use fresh timeout context
	delCtx, delCancel := context.WithTimeout(ctx, 2*time.Second)
	defer delCancel()
	if err := s.redis.Del(delCtx, key).Err(); err != nil {
		s.logger.WithError(err).Warn("Failed to delete verification token")
	}

	s.logger.WithField("user_id", userID).Info("Verification token validated")
	return userID, nil
}

// GeneratePasswordResetToken generates a 32-byte random token for password reset
func (s *TokenService) GeneratePasswordResetToken(ctx context.Context, userID string) (string, error) {
	token, err := s.generateRandomToken()
	if err != nil {
		return "", err
	}

	// FIX [BUG-GO-016]: Add Redis timeout to prevent hanging on slow Redis
	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Store in Redis with 1-hour expiry
	key := "reset:" + token
	if err := s.redis.Set(redisCtx, key, userID, 1*time.Hour).Err(); err != nil {
		s.logger.WithError(err).Error("Failed to store password reset token")
		return "", fmt.Errorf("failed to store password reset token: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"token":   token[:8] + "...", // Log only first 8 chars
	}).Debug("Password reset token generated")

	return token, nil
}

// ValidatePasswordResetToken validates and consumes a password reset token
func (s *TokenService) ValidatePasswordResetToken(ctx context.Context, token string) (string, error) {
	// FIX [BUG-GO-016]: Add Redis timeout to prevent hanging on slow Redis
	redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	key := "reset:" + token
	userID, err := s.redis.Get(redisCtx, key).Result()
	if err != nil {
		s.logger.WithError(err).Debug("Password reset token not found or expired")
		return "", fmt.Errorf("invalid or expired token")
	}

	// Delete token (single-use) - use fresh timeout context
	delCtx, delCancel := context.WithTimeout(ctx, 2*time.Second)
	defer delCancel()
	if err := s.redis.Del(delCtx, key).Err(); err != nil {
		s.logger.WithError(err).Warn("Failed to delete password reset token")
	}

	s.logger.WithField("user_id", userID).Info("Password reset token validated")
	return userID, nil
}

// generateRandomToken generates a 32-byte random token
func (s *TokenService) generateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// InvalidateUserTokens invalidates all current tokens for a user by setting a revocation timestamp
// This is used after password resets or security events
func (s *TokenService) InvalidateUserTokens(ctx context.Context, userID string) error {
	// Set timestamp key "user_revocation:{userID}" to current time
	key := fmt.Sprintf("user_revocation:%s", userID)
	now := time.Now().Unix()

	// Use a long TTL (e.g. 7 days - max refresh token life) or effectively infinite
	// If a user doesn't login for 7 days, older tokens would expire anyway
	err := s.redis.Set(ctx, key, now, 7*24*time.Hour).Err()
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to set revocation timestamp")
		return fmt.Errorf("failed to invalidate tokens: %w", err)
	}

	s.logger.WithField("user_id", userID).Info("All user tokens invalidated (MinIAT updated)")
	return nil
}

// IsTokenRevoked checks if a token has been implicitly revoked via MinIAT policy
// Returns true (revoked) if Token IssuedAt < User Revocation Timestamp
func (s *TokenService) IsTokenRevoked(ctx context.Context, userID string, tokenIssuedAtUnix int64) bool {
	// 1. Get user revocation timestamp
	key := fmt.Sprintf("user_revocation:%s", userID)
	revocationTimestamp, err := s.redis.Get(ctx, key).Int64()
	if err != nil {
		// Key doesn't exist (no revocation) or Redis error
		// Fail open for Redis errors (allow access) to prevent lockout if Redis is unstable
		// Security trade-off: Availability > Strict Revocation in case of Redis failure
		return false
	}

	// 2. Compare with token IssuedAt
	// If Token IssuedAt < Revocation Timestamp, token is invalid
	if tokenIssuedAtUnix < revocationTimestamp {
		s.logger.WithFields(logrus.Fields{
			"user_id":      userID,
			"token_iat":    tokenIssuedAtUnix,
			"revocation_t": revocationTimestamp,
		}).Warn("Token rejected by MinIAT policy (Password Reset via Redis)")
		return true
	}

	return false
}
