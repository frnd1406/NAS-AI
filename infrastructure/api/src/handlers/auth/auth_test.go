package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
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
	"github.com/nas-ai/api/src/services/operations"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*gin.Engine, sqlmock.Sqlmock, *auth_repo.UserRepository, *security.JWTService, *security.TokenService, *config.Config) {
	// Setup Logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Setup Mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	// Create database.DB wrapper
	databaseDB := &database.DB{DB: db} // Logger is unexported

	userRepo := auth_repo.NewUserRepository(databaseDB, logger)

	// Setup Redis (MiniRedis)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	redisClient := &database.RedisClient{Client: rdb}

	// Setup Services
	cfg := &config.Config{
		JWTSecret:    "test-secret-at-least-32-chars-long-secure",
		ResendAPIKey: "re_123456789",
		EmailFrom:    "test@example.com",
		FrontendURL:  "http://localhost:3000",
		InviteCode:   "SECRET_CODE",
	}
	jwtService, _ := security.NewJWTService(cfg, logger)
	tokenService := security.NewTokenService(redisClient, logger)

	pwdService := security.NewPasswordService()
	emailService := operations.NewEmailService(cfg, logger)

	// Setup Router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register Handlers
	loginHandler := LoginHandler(userRepo, jwtService, pwdService, nil, nil, redisClient, logger)
	registerHandler := RegisterHandler(cfg, userRepo, jwtService, pwdService, tokenService, emailService, redisClient, logger)

	api := router.Group("/auth")
	{
		api.POST("/login", loginHandler)
		api.POST("/register", registerHandler)
	}

	return router, mock, userRepo, jwtService, tokenService, cfg
}

func TestLoginHandler_Success(t *testing.T) {
	router, mock, _, _, _, _ := setupTest(t)
	pwdService := security.NewPasswordService()

	// Hash password
	hashedPwd, _ := pwdService.HashPassword("Password123!")

	// Expect FindByEmail query
	rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "email_verified", "verified_at", "created_at", "updated_at"}).
		AddRow("user-123", "testuser", "test@example.com", hashedPwd, "user", true, time.Now(), time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at FROM users WHERE email = \\$1").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	// Perform Login Request
	loginReq := LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}
	body, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user")

	// Verify cookies
	cookies := w.Result().Cookies()
	var accessTokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "access_token" {
			accessTokenCookie = c
			break
		}
	}
	assert.NotNil(t, accessTokenCookie, "access_token cookie should be set")
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	router, mock, _, _, _, _ := setupTest(t)
	pwdService := security.NewPasswordService()

	hashedPwd, _ := pwdService.HashPassword("Password123!")

	// Expect FindByEmail query - User Found
	rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "email_verified", "verified_at", "created_at", "updated_at"}).
		AddRow("user-123", "testuser", "test@example.com", hashedPwd, "user", true, time.Now(), time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at FROM users WHERE email = \\$1").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	// Perform Login Request with WRONG password
	loginReq := LoginRequest{
		Email:    "test@example.com",
		Password: "WrongPassword!",
	}
	body, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginHandler_UserNotFound(t *testing.T) {
	router, mock, _, _, _, _ := setupTest(t)

	// Expect FindByEmail query - SQL ErrNoRows
	mock.ExpectQuery("SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at FROM users WHERE email = \\$1").
		WithArgs("nonexistent@example.com").
		WillReturnError(sql.ErrNoRows) // Now specifically returning ErrNoRows

	// Perform Login Request
	loginReq := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "Password123!",
	}
	body, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRegisterHandler_Success(t *testing.T) {
	router, mock, _, _, _, cfg := setupTest(t)

	// Expectations
	// 1. Find by Username (should return nil aka no error, no user)
	// mock expects regex, so escape $1
	mock.ExpectQuery("SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at FROM users WHERE username = \\$1").
		WithArgs("newuser").
		WillReturnError(sql.ErrNoRows)

	// 2. Find by Email
	mock.ExpectQuery("SELECT id, username, email, password_hash, role, email_verified, verified_at, created_at, updated_at FROM users WHERE email = \\$1").
		WithArgs("new@example.com").
		WillReturnError(sql.ErrNoRows)

	// 3. Create User (INSERT)
	rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "email_verified", "verified_at", "created_at", "updated_at"}).
		AddRow("user-new", "newuser", "new@example.com", "hashed_pwd", "user", false, nil, time.Now(), time.Now())

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("newuser", "new@example.com", sqlmock.AnyArg()). // 3 args: username, email, passwordHash
		WillReturnRows(rows)

	// Perform Register Request
	regReq := RegisterRequest{
		Username:   "newuser",
		Email:      "new@example.com",
		Password:   "Password123!",
		InviteCode: cfg.InviteCode,
	}
	body, _ := json.Marshal(regReq)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")
}
