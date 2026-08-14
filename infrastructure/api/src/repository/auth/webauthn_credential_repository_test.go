package auth_repo

import (
	"context"
	"log/slog"
	"testing"

	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func setupWebAuthnRepo(t *testing.T) *WebAuthnCredentialRepository {
	t.Helper()
	testDB, err := database.NewTestDatabase(slog.Default())
	require.NoError(t, err)
	t.Cleanup(func() { _ = testDB.DB.Close() })

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo := NewWebAuthnCredentialRepository(testDB.DB, logger)
	require.NoError(t, repo.EnsureTable(context.Background())) // idempotent
	return repo
}

func TestWebAuthnCredentialRepository_CRUD(t *testing.T) {
	repo := setupWebAuthnRepo(t)
	ctx := context.Background()
	const userID = "user-1"

	// Empty to start.
	count, err := repo.CountByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Create two credentials.
	require.NoError(t, repo.Create(ctx, userID, "YubiKey 5", "cred-abc", []byte(`{"id":"abc"}`), 0))
	require.NoError(t, repo.Create(ctx, userID, "Backup Key", "cred-def", []byte(`{"id":"def"}`), 0))

	count, err = repo.CountByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// List returns both.
	list, err := repo.ListByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	// Update after login bumps sign count.
	require.NoError(t, repo.UpdateAfterLogin(ctx, "cred-abc", []byte(`{"id":"abc","n":1}`), 5))
	list, err = repo.ListByUserID(ctx, userID)
	require.NoError(t, err)
	var found bool
	for _, rec := range list {
		if rec.CredentialID == "cred-abc" {
			found = true
			require.Equal(t, int64(5), rec.SignCount)
			require.True(t, rec.LastUsedAt.Valid)
		}
	}
	require.True(t, found)

	// Delete one; the other remains.
	target := list[0].ID
	require.NoError(t, repo.Delete(ctx, userID, target))
	count, err = repo.CountByUserID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Deleting a foreign/nonexistent credential yields no rows.
	require.Error(t, repo.Delete(ctx, "other-user", target))
}

func TestWebAuthnCredentialRepository_UniqueCredentialID(t *testing.T) {
	repo := setupWebAuthnRepo(t)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, "user-1", "Key", "dup", []byte(`{}`), 0))
	// Same credential_id must be rejected by the UNIQUE constraint.
	require.Error(t, repo.Create(ctx, "user-2", "Key", "dup", []byte(`{}`), 0))
}
