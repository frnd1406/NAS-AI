package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/database"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// When the user has a registered WebAuthn credential, the password step must
// NOT issue a session; it returns mfa_required + an mfa_token instead.
func TestLoginHandler_MFARequired(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	gin.SetMode(gin.TestMode)

	// User repo (sqlmock) returning a valid user for the password check.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	userRepo := auth_repo.NewUserRepository(&database.DB{DB: db}, logger)

	pwdService := security.NewPasswordService()
	hashedPwd, _ := pwdService.HashPassword("Password123!")
	rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "email_verified", "verified_at", "created_at", "updated_at"}).
		AddRow("user-123", "testuser", "test@example.com", hashedPwd, "user", true, time.Now(), time.Now(), time.Now())
	mock.ExpectQuery("SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at FROM users WHERE email = \\$1").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	// Credential repo (real SQLite) with one credential for user-123.
	testDB, err := database.NewTestDatabase(slog.Default())
	require.NoError(t, err)
	t.Cleanup(func() { _ = testDB.DB.Close() })
	credRepo := auth_repo.NewWebAuthnCredentialRepository(testDB.DB, logger)
	require.NoError(t, credRepo.EnsureTable(context.Background()))
	require.NoError(t, credRepo.Create(context.Background(), "user-123", "YubiKey", "cred-1", []byte(`{"id":"x"}`), 0))

	// Redis + services.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient := &database.RedisClient{Client: redis.NewClient(&redis.Options{Addr: mr.Addr()})}

	cfg := &config.Config{JWTSecret: "test-secret-at-least-32-chars-long-secure"}
	jwtService, _ := security.NewJWTService(cfg, logger)
	webAuthn, err := security.NewWebAuthnService("localhost", "NAS.AI", []string{"http://localhost"}, redisClient, credRepo, logger)
	require.NoError(t, err)

	router := gin.New()
	router.POST("/auth/login", LoginHandler(userRepo, jwtService, pwdService, webAuthn, credRepo, redisClient, logger))

	body, _ := json.Marshal(LoginRequest{Email: "test@example.com", Password: "Password123!"})
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["mfa_required"])
	assert.NotEmpty(t, resp["mfa_token"])

	// No auth cookies must be set at this stage.
	for _, ck := range w.Result().Cookies() {
		assert.NotEqual(t, AccessTokenCookieName, ck.Name, "access token cookie must not be set before 2FA")
	}
}
