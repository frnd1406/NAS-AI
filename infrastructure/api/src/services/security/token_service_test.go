package security

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTokenService(t *testing.T) (*TokenService, *miniredis.Miniredis) {
	ensureTCPAllowed(t)

	// Create miniredis instance
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client pointing to miniredis
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	redisClient := &database.RedisClient{
		Client: client,
	}
	return NewTokenService(redisClient, logger), mr
}

func TestTokenService_GenerateVerificationToken(t *testing.T) {
	ts, mr := setupTokenService(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "test-user-123"

	token, err := ts.GenerateVerificationToken(ctx, userID)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Token should be stored in Redis with correct key
	key := "verify:" + token
	storedUserID, err := ts.redis.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, userID, storedUserID)

	// Check TTL is approximately 24 hours
	ttl := mr.TTL(key)
	assert.Greater(t, ttl.Hours(), 23.0)
	assert.Less(t, ttl.Hours(), 25.0)
}

func TestTokenService_ValidateVerificationToken(t *testing.T) {
	ts, mr := setupTokenService(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "test-user-123"

	// Generate token first
	token, err := ts.GenerateVerificationToken(ctx, userID)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   token,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid-token-xyz",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			returnedUserID, err := ts.ValidateVerificationToken(ctx, tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, returnedUserID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, userID, returnedUserID)

				// Token should be deleted after validation (single-use)
				key := "verify:" + tt.token
				_, err := ts.redis.Get(ctx, key).Result()
				assert.Error(t, err) // Should be redis.Nil error
			}
		})
	}
}

func TestTokenService_VerificationTokenExpiry(t *testing.T) {
	ts, mr := setupTokenService(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "test-user-123"

	token, err := ts.GenerateVerificationToken(ctx, userID)
	require.NoError(t, err)

	// Fast-forward time by 24 hours + 1 second
	mr.FastForward(24*time.Hour + 1*time.Second)

	// Token should now be expired
	_, err = ts.ValidateVerificationToken(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired token")
}

func TestTokenService_GeneratePasswordResetToken(t *testing.T) {
	ts, mr := setupTokenService(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "test-user-456"

	token, err := ts.GeneratePasswordResetToken(ctx, userID)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Token should be stored in Redis with correct key
	key := "reset:" + token
	storedUserID, err := ts.redis.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, userID, storedUserID)

	// Check TTL is approximately 1 hour
	ttl := mr.TTL(key)
	assert.Greater(t, ttl.Minutes(), 59.0)
	assert.Less(t, ttl.Minutes(), 61.0)
}

func TestTokenService_ValidatePasswordResetToken(t *testing.T) {
	ts, mr := setupTokenService(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "test-user-456"

	// Generate token first
	token, err := ts.GeneratePasswordResetToken(ctx, userID)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   token,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid-reset-token",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			returnedUserID, err := ts.ValidatePasswordResetToken(ctx, tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, returnedUserID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, userID, returnedUserID)

				// Token should be deleted after validation (single-use)
				key := "reset:" + tt.token
				_, err := ts.redis.Get(ctx, key).Result()
				assert.Error(t, err) // Should be redis.Nil error
			}
		})
	}
}

func TestTokenService_PasswordResetTokenExpiry(t *testing.T) {
	ts, mr := setupTokenService(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "test-user-456"

	token, err := ts.GeneratePasswordResetToken(ctx, userID)
	require.NoError(t, err)

	// Fast-forward time by 1 hour + 1 second
	mr.FastForward(1*time.Hour + 1*time.Second)

	// Token should now be expired
	_, err = ts.ValidatePasswordResetToken(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired token")
}

func TestTokenService_TokenUniqueness(t *testing.T) {
	ts, mr := setupTokenService(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "test-user-789"

	// Generate multiple tokens
	token1, err1 := ts.GenerateVerificationToken(ctx, userID)
	token2, err2 := ts.GenerateVerificationToken(ctx, userID)
	token3, err3 := ts.GeneratePasswordResetToken(ctx, userID)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	// All tokens should be unique
	assert.NotEqual(t, token1, token2)
	assert.NotEqual(t, token1, token3)
	assert.NotEqual(t, token2, token3)
}

func TestTokenService_SingleUseToken(t *testing.T) {
	ts, mr := setupTokenService(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "test-user-single-use"

	// Generate and validate verification token
	token, err := ts.GenerateVerificationToken(ctx, userID)
	require.NoError(t, err)

	// First validation should succeed
	returnedUserID, err := ts.ValidateVerificationToken(ctx, token)
	assert.NoError(t, err)
	assert.Equal(t, userID, returnedUserID)

	// Second validation should fail (token already consumed)
	_, err = ts.ValidateVerificationToken(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired token")
}

func TestTokenService_UserRevocation(t *testing.T) {
	ts, mr := setupTokenService(t)
	defer mr.Close()

	ctx := context.Background()
	userID := "user-revocation-test"

	// 1. Initial state: No revocation
	tokenIssuedAt := time.Now().Unix()
	revoked := ts.IsTokenRevoked(ctx, userID, tokenIssuedAt)
	assert.False(t, revoked, "Token should not be revoked initially")

	// 2. Invalidate tokens
	err := ts.InvalidateUserTokens(ctx, userID)
	require.NoError(t, err)

	// Check Redis state
	key := "user_revocation:" + userID
	revocationTimeStr, err := ts.redis.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.NotEmpty(t, revocationTimeStr)

	// 3. Check token issued BEFORE revocation
	oldTokenIssuedAt := time.Now().Add(-1 * time.Hour).Unix()
	revoked = ts.IsTokenRevoked(ctx, userID, oldTokenIssuedAt)
	assert.True(t, revoked, "Old token should be revoked")

	// 4. Check token issued AFTER revocation
	newTokenIssuedAt := time.Now().Add(1 * time.Hour).Unix()
	revoked = ts.IsTokenRevoked(ctx, userID, newTokenIssuedAt)
	assert.False(t, revoked, "New token should be valid")
}

func ensureTCPAllowed(t *testing.T) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("TCP sockets nicht erlaubt in dieser Umgebung: %v", err)
		return
	}
	ln.Close()
}
