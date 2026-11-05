package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nas-ai/api/test/testutils"
	"github.com/stretchr/testify/assert"
)

// TestLogin_Success tests login with REAL security services (JWT, Password).
// This is a true integration test - no mocks for auth logic.
func TestLogin_Success_WithRealServices(t *testing.T) {
	// 1. Setup TestEnv with REAL security services
	env := testutils.NewTestEnv(t)

	// 2. First, register a user (so we have someone to login as)
	router := testutils.SetupTestRouter(env)

	// Register user
	registerPayload := map[string]string{
		"username":    "testuser",
		"email":       "test@example.com",
		"password":    "SecurePassword123!",
		"invite_code": env.Config.InviteCode,
	}
	jsonData, _ := json.Marshal(registerPayload)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Registration should succeed (status might be OK or Created)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated,
		"Registration should succeed, got: %d, body: %s", w.Code, w.Body.String())

	// 3. Now login with the registered user
	loginPayload := map[string]string{
		"email":    "test@example.com",
		"password": "SecurePassword123!",
	}
	jsonData, _ = json.Marshal(loginPayload)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 4. Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check user in response
	userData, ok := response["user"].(map[string]interface{})
	assert.True(t, ok, "User field should be present")
	assert.Equal(t, "test@example.com", userData["email"])

	// Check Cookies for tokens
	cookies := w.Result().Cookies()
	var accessTokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "access_token" {
			accessTokenCookie = c
			break
		}
	}
	assert.NotNil(t, accessTokenCookie, "Access token cookie should be set")
	assert.NotEmpty(t, accessTokenCookie.Value, "Access token should not be empty")
}

// TestLogin_Failure_WrongPassword tests login failure with wrong password
func TestLogin_Failure_WrongPassword(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := testutils.SetupTestRouter(env)

	// First register a user
	registerPayload := map[string]string{
		"username":    "wrongpwuser",
		"email":       "wrongpw@example.com",
		"password":    "CorrectPassword123!",
		"invite_code": env.Config.InviteCode,
	}
	jsonData, _ := json.Marshal(registerPayload)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Now try to login with wrong password
	loginPayload := map[string]string{
		"email":    "wrongpw@example.com",
		"password": "WrongPassword456!",
	}
	jsonData, _ = json.Marshal(loginPayload)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestLogin_Failure_UserNotFound tests login with non-existent user
func TestLogin_Failure_UserNotFound(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := testutils.SetupTestRouter(env)

	loginPayload := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "AnyPassword123!",
	}
	jsonData, _ := json.Marshal(loginPayload)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestProtectedEndpoint_WithRealToken tests accessing protected endpoint with real JWT
func TestProtectedEndpoint_WithRealToken(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := testutils.SetupTestRouter(env)

	// Generate a real JWT token
	token, err := env.GenerateTestToken("user-123", "realuser@example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Access protected endpoint
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "user-123", response["user_id"])
	assert.Equal(t, "realuser@example.com", response["user_email"])
}

// TestProtectedEndpoint_NoToken tests accessing protected endpoint without token
func TestProtectedEndpoint_NoToken(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := testutils.SetupTestRouter(env)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
