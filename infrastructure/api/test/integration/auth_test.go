package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ============================================================
// REAL Integration Tests - Auth Flow
// Uses REAL handlers from src/handlers/auth
// ============================================================

func TestIntegration_RegisterLoginFlow(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := gin.New()

	// Add request ID middleware (required by handlers)
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})

	// Setup REAL auth endpoints using production handlers
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", testutils.TestableRegisterHandler(env))
		authGroup.POST("/login", testutils.TestableLoginHandler(env))
	}

	// Test 1: Register new user
	t.Run("Register_Success", func(t *testing.T) {
		body := `{"username": "testuser", "email": "test@example.com", "password": "SecurePassword123!", "invite_code": "TEST_INVITE"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp["access_token"])
		assert.NotEmpty(t, resp["csrf_token"])

		// Verify user object is returned
		user, ok := resp["user"].(map[string]interface{})
		require.True(t, ok, "user should be an object")
		assert.Equal(t, "test@example.com", user["email"])
		assert.Equal(t, "testuser", user["username"])
	})

	// Test 2: Duplicate registration
	t.Run("Register_DuplicateEmail", func(t *testing.T) {
		body := `{"username": "testuser2", "email": "test@example.com", "password": "SecurePassword123!", "invite_code": "TEST_INVITE"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		errorObj := resp["error"].(map[string]interface{})
		assert.Equal(t, "email_exists", errorObj["code"])
	})

	// Test 3: Login with registered user
	t.Run("Login_Success", func(t *testing.T) {
		body := `{"email": "test@example.com", "password": "SecurePassword123!"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp["csrf_token"])

		// Verify user object is returned
		user, ok := resp["user"].(map[string]interface{})
		require.True(t, ok, "user should be an object")
		assert.Equal(t, "test@example.com", user["email"])
	})

	// Test 4: Login with wrong password
	t.Run("Login_WrongPassword", func(t *testing.T) {
		body := `{"email": "test@example.com", "password": "WrongPassword!"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		errorObj := resp["error"].(map[string]interface{})
		assert.Equal(t, "invalid_credentials", errorObj["code"])
	})

	// Test 5: Login with non-existent user
	t.Run("Login_UserNotFound", func(t *testing.T) {
		body := `{"email": "nonexistent@example.com", "password": "SomePassword123!"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		errorObj := resp["error"].(map[string]interface{})
		assert.Equal(t, "invalid_credentials", errorObj["code"])
	})
}

func TestIntegration_PasswordValidation(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := gin.New()

	// Add request ID middleware
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})

	// Use REAL register handler
	router.POST("/auth/register", testutils.TestableRegisterHandler(env))

	tests := []struct {
		name       string
		password   string
		wantStatus int
	}{
		{"TooShort", "short", http.StatusBadRequest},
		{"ExactlyMinLength", "Abcdef12", http.StatusCreated}, // 8 chars, Mixed case + digit
		{"StrongPassword", "SecurePassword123!", http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"username":    "testuser_" + tt.name,
				"email":       tt.name + "@example.com",
				"password":    tt.password,
				"invite_code": "TEST_INVITE",
			})

			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestIntegration_InviteCodeValidation(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := gin.New()

	// Add request ID middleware
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})

	// Use REAL register handler
	router.POST("/auth/register", testutils.TestableRegisterHandler(env))

	t.Run("MissingInviteCode", func(t *testing.T) {
		body := `{"username": "testuser", "email": "test@example.com", "password": "SecurePassword123!"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		errorObj := resp["error"].(map[string]interface{})
		assert.Equal(t, "invalid_invite_code", errorObj["code"])
	})

	t.Run("InvalidInviteCode", func(t *testing.T) {
		body := `{"username": "testuser", "email": "test@example.com", "password": "SecurePassword123!", "invite_code": "WRONG_CODE"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		errorObj := resp["error"].(map[string]interface{})
		assert.Equal(t, "invalid_invite_code", errorObj["code"])
	})

	t.Run("ValidInviteCode", func(t *testing.T) {
		body := `{"username": "testuser", "email": "valid@example.com", "password": "SecurePassword123!", "invite_code": "TEST_INVITE"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}
