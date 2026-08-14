package security

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) *database.RedisClient {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return &database.RedisClient{Client: rdb}
}

func newTestWebAuthnService(t *testing.T) *WebAuthnService {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	svc, err := NewWebAuthnService("localhost", "NAS.AI", []string{"http://localhost"}, newTestRedis(t), nil, logger)
	require.NoError(t, err)
	require.NotNil(t, svc)
	return svc
}

// Disabled when no RP ID is configured.
func TestNewWebAuthnService_DisabledWhenNoRPID(t *testing.T) {
	logger := logrus.New()
	svc, err := NewWebAuthnService("", "NAS.AI", nil, nil, nil, logger)
	require.NoError(t, err)
	require.Nil(t, svc)
}

func TestMFAPendingToken_PeekThenConsume(t *testing.T) {
	svc := newTestWebAuthnService(t)
	ctx := context.Background()

	token, err := svc.GenerateMFAPendingToken(ctx, "user-42")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Peek (begin step) does not consume: two peeks both succeed.
	uid, err := svc.ValidateMFAPendingToken(ctx, token, false)
	require.NoError(t, err)
	require.Equal(t, "user-42", uid)

	uid, err = svc.ValidateMFAPendingToken(ctx, token, false)
	require.NoError(t, err)
	require.Equal(t, "user-42", uid)

	// Consume (finish step) invalidates the token.
	uid, err = svc.ValidateMFAPendingToken(ctx, token, true)
	require.NoError(t, err)
	require.Equal(t, "user-42", uid)

	_, err = svc.ValidateMFAPendingToken(ctx, token, false)
	require.Error(t, err)
}

func TestMFAPendingToken_InvalidToken(t *testing.T) {
	svc := newTestWebAuthnService(t)
	_, err := svc.ValidateMFAPendingToken(context.Background(), "does-not-exist", false)
	require.Error(t, err)
}
