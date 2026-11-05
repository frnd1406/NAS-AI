package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nas-ai/api/src/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupJWTService(t *testing.T) *JWTService {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-minimum-32-characters-long-for-security",
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	svc, err := NewJWTService(cfg, logger)
	require.NoError(t, err)
	return svc
}

func TestNewJWTService_InvalidSecret(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "too-short",
	}
	logger := logrus.New()

	svc, err := NewJWTService(cfg, logger)
	assert.Nil(t, svc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least")
}

func TestJWTService_GenerateAccessToken(t *testing.T) {
	js := setupJWTService(t)

	userID := "test-user-123"
	email := "test@example.com"

	token, err := js.GenerateAccessToken(userID, email)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token structure (JWT has 3 parts separated by dots)
	assert.Contains(t, token, ".")
}

func TestJWTService_GenerateRefreshToken(t *testing.T) {
	js := setupJWTService(t)

	userID := "test-user-123"
	email := "test@example.com"

	token, err := js.GenerateRefreshToken(userID, email)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Contains(t, token, ".")
}

func TestJWTService_ValidateToken(t *testing.T) {
	js := setupJWTService(t)

	userID := "test-user-123"
	email := "test@example.com"

	// Generate a valid access token
	accessToken, err := js.GenerateAccessToken(userID, email)
	require.NoError(t, err)

	// Generate a valid refresh token
	refreshToken, err := js.GenerateRefreshToken(userID, email)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid access token",
			token:   accessToken,
			wantErr: false,
		},
		{
			name:    "valid refresh token",
			token:   refreshToken,
			wantErr: false,
		},
		{
			name:    "invalid token format",
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not-a-jwt-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := js.ValidateToken(tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				if claims != nil {
					assert.Equal(t, userID, claims.UserID)
					assert.Equal(t, email, claims.Email)
				}
			}
		})
	}
}

func TestJWTService_TokenExpiration(t *testing.T) {
	js := setupJWTService(t)

	userID := "test-user-123"
	email := "test@example.com"

	// Generate token
	token, err := js.GenerateAccessToken(userID, email)
	require.NoError(t, err)

	// Validate immediately - should work
	claims, err := js.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Check expiration time is in the future
	assert.True(t, claims.ExpiresAt.After(time.Now()))

	// Access token should expire in 15 minutes
	expectedExpiry := time.Now().Add(15 * time.Minute)
	timeDiff := claims.ExpiresAt.Sub(expectedExpiry)
	assert.Less(t, timeDiff.Abs().Seconds(), 2.0, "Expiry should be ~15 minutes from now")
}

func TestJWTService_RefreshTokenExpiration(t *testing.T) {
	js := setupJWTService(t)

	userID := "test-user-123"
	email := "test@example.com"

	// Generate refresh token
	token, err := js.GenerateRefreshToken(userID, email)
	require.NoError(t, err)

	// Validate immediately
	claims, err := js.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Refresh token should expire in 7 days
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	timeDiff := claims.ExpiresAt.Sub(expectedExpiry)
	assert.Less(t, timeDiff.Abs().Seconds(), 2.0, "Expiry should be ~7 days from now")
}

func TestJWTService_ClaimsContent(t *testing.T) {
	js := setupJWTService(t)

	userID := "test-user-456"
	email := "another@example.com"

	token, err := js.GenerateAccessToken(userID, email)
	require.NoError(t, err)

	claims, err := js.ValidateToken(token)
	require.NoError(t, err)

	// Verify all claim fields
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.NotZero(t, claims.IssuedAt)
	assert.NotZero(t, claims.ExpiresAt)
}

func TestJWTService_ExtractClaims(t *testing.T) {
	js := setupJWTService(t)

	userID := "test-user-claims"
	email := "claims@example.com"

	// 1. Valid token
	token, err := js.GenerateAccessToken(userID, email)
	require.NoError(t, err)

	claims, err := js.ExtractClaims(token)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)

	// 2. Expired token (should still extract claims)
	// Create an expired token manually
	expiredClaims := TokenClaims{
		UserID:    userID,
		Email:     email,
		TokenType: AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, _ := expiredToken.SignedString(js.secret)

	claims, err = js.ExtractClaims(expiredTokenString)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)

	// 3. Invalid token (garbage)
	_, err = js.ExtractClaims("garbage.token.string")
	assert.Error(t, err)
}
