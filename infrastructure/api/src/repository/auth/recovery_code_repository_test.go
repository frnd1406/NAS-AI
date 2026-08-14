package auth_repo

import (
	"context"
	"log/slog"
	"testing"

	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func setupRecoveryRepo(t *testing.T) *RecoveryCodeRepository {
	t.Helper()
	testDB, err := database.NewTestDatabase(slog.Default())
	require.NoError(t, err)
	t.Cleanup(func() { _ = testDB.DB.Close() })

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo := NewRecoveryCodeRepository(testDB.DB, logger)
	require.NoError(t, repo.EnsureTable(context.Background())) // idempotent
	return repo
}

func TestRecoveryCodeRepository_ReplaceListConsume(t *testing.T) {
	repo := setupRecoveryRepo(t)
	ctx := context.Background()
	const userID = "user-1"

	// Empty to start.
	count, err := repo.CountUnusedByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Store three hashes.
	require.NoError(t, repo.ReplaceForUser(ctx, userID, []string{"h1", "h2", "h3"}))
	count, err = repo.CountUnusedByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	list, err := repo.ListUnusedByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// Consume one.
	first := list[0].ID
	ok, err := repo.MarkUsed(ctx, first)
	require.NoError(t, err)
	require.True(t, ok)

	// Consuming the same code again is a no-op (single-use).
	ok, err = repo.MarkUsed(ctx, first)
	require.NoError(t, err)
	require.False(t, ok)

	count, err = repo.CountUnusedByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestRecoveryCodeRepository_ReplaceClearsOldPerUser(t *testing.T) {
	repo := setupRecoveryRepo(t)
	ctx := context.Background()
	const userID = "user-1"

	require.NoError(t, repo.ReplaceForUser(ctx, userID, []string{"a", "b"}))
	// Regenerate: the previous set is dropped and replaced.
	require.NoError(t, repo.ReplaceForUser(ctx, userID, []string{"c", "d", "e"}))
	count, err := repo.CountUnusedByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	// A different user's codes are untouched by another user's regeneration.
	require.NoError(t, repo.ReplaceForUser(ctx, "user-2", []string{"x"}))
	require.NoError(t, repo.ReplaceForUser(ctx, userID, []string{"f"}))
	c2, err := repo.CountUnusedByUserID(ctx, "user-2")
	require.NoError(t, err)
	require.Equal(t, 1, c2)
}
