package security

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/nas-ai/api/src/database"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func setupRecoveryService(t *testing.T) *RecoveryCodeService {
	t.Helper()
	testDB, err := database.NewTestDatabase(slog.Default())
	require.NoError(t, err)
	t.Cleanup(func() { _ = testDB.DB.Close() })

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo := auth_repo.NewRecoveryCodeRepository(testDB.DB, logger)
	require.NoError(t, repo.EnsureTable(context.Background()))
	return NewRecoveryCodeService(repo, NewPasswordService(), logger)
}

func TestRecoveryCodeService_GenerateVerifyConsume(t *testing.T) {
	svc := setupRecoveryService(t)
	ctx := context.Background()
	const userID = "user-1"

	codes, err := svc.GenerateForUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, codes, recoveryCodeCount)

	remaining, err := svc.RemainingCount(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, recoveryCodeCount, remaining)

	// A wrong code is rejected.
	ok, err := svc.Verify(ctx, userID, "WRONG-CODE9")
	require.NoError(t, err)
	require.False(t, ok)

	// A valid code works exactly once.
	ok, err = svc.Verify(ctx, userID, codes[0])
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = svc.Verify(ctx, userID, codes[0])
	require.NoError(t, err)
	require.False(t, ok, "a consumed code must not be accepted again")

	remaining, err = svc.RemainingCount(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, recoveryCodeCount-1, remaining)

	// Formatting differences (case, spaces instead of hyphen) still verify.
	variant := strings.ToLower(strings.ReplaceAll(codes[1], "-", " "))
	ok, err = svc.Verify(ctx, userID, variant)
	require.NoError(t, err)
	require.True(t, ok, "normalization should accept differently-formatted input")
}

func TestRecoveryCodeService_RegenerateInvalidatesOld(t *testing.T) {
	svc := setupRecoveryService(t)
	ctx := context.Background()
	const userID = "user-1"

	first, err := svc.GenerateForUser(ctx, userID)
	require.NoError(t, err)

	// Regenerating must invalidate the previous set entirely.
	_, err = svc.GenerateForUser(ctx, userID)
	require.NoError(t, err)

	ok, err := svc.Verify(ctx, userID, first[0])
	require.NoError(t, err)
	require.False(t, ok, "old codes must stop working after regeneration")
}

func TestRecoveryCodeService_EmptyInput(t *testing.T) {
	svc := setupRecoveryService(t)
	ctx := context.Background()
	const userID = "user-1"

	_, err := svc.GenerateForUser(ctx, userID)
	require.NoError(t, err)

	// Blank/whitespace input must never match.
	ok, err := svc.Verify(ctx, userID, "   ")
	require.NoError(t, err)
	require.False(t, ok)
}
