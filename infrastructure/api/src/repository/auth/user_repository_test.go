package auth_repo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRepoTest(t *testing.T) (*UserRepository, sqlmock.Sqlmock) {
	// Setup DB Mock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	// Create wrapper
	databaseDB := &database.DB{DB: db} // Logger unexported

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	repo := NewUserRepository(databaseDB, logger)
	return repo, mock
}

func TestUserRepository_FindByEmail(t *testing.T) {
	repo, mock := setupRepoTest(t)

	email := "test@example.com"

	// Expectations
	rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "email_verified", "verified_at", "created_at", "updated_at"}).
		AddRow("user-1", "user1", email, "hash", "user", true, time.Now(), time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at FROM users WHERE email = \\$1").
		WithArgs(email).
		WillReturnRows(rows)

	// Execute
	user, err := repo.FindByEmail(context.Background(), email)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, "user1", user.Username)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	repo, mock := setupRepoTest(t)

	email := "missing@example.com"

	// Expectations
	mock.ExpectQuery("SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at FROM users WHERE email = \\$1").
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)

	// Execute
	user, err := repo.FindByEmail(context.Background(), email)

	// Verify
	assert.NoError(t, err) // Should return nil, nil
	assert.Nil(t, user)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_CreateUser(t *testing.T) {
	repo, mock := setupRepoTest(t)

	username := "newuser"
	email := "new@example.com"
	hash := "hashed_password"

	// Expectations
	// CreateUser uses QueryRow and Scan, so we must return rows
	rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "email_verified", "verified_at", "created_at", "updated_at"}).
		AddRow("user-new", username, email, hash, "user", false, nil, time.Now(), time.Now())

	mock.ExpectQuery("INSERT INTO users .* VALUES .* RETURNING .*").
		WithArgs(username, email, hash).
		WillReturnRows(rows)

	// Execute
	user, err := repo.CreateUser(context.Background(), username, email, hash)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, email, user.Email)

	assert.NoError(t, mock.ExpectationsWereMet())
}
