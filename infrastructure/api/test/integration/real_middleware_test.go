package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nas-ai/api/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// Integration Tests using REAL AuthMiddleware
// These tests verify the actual middleware behavior with real JWT validation
// ============================================================

func TestRealAuthMiddleware_ValidToken(t *testing.T) {
	// Setup with real services backed by miniredis
	env := testutils.NewTestEnv(t)
	router := testutils.SetupTestRouter(env)

	// Generate a real JWT token using env's JWTService
	token, err := env.GenerateTestToken("user-123", "test@example.com")
	require.NoError(t, err)

	// Test protected endpoint with valid token
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-123")
	assert.Contains(t, w.Body.String(), "test@example.com")
}

func TestRealAuthMiddleware_NoToken(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := testutils.SetupTestRouter(env)

	// Test protected endpoint without token
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing authorization token")
}

func TestRealAuthMiddleware_InvalidToken(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := testutils.SetupTestRouter(env)

	// Test with invalid token
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid or expired token")
}

func TestRealAuthMiddleware_CookieAuth(t *testing.T) {
	env := testutils.NewTestEnv(t)
	router := testutils.SetupTestRouter(env)

	// Generate real token using env's JWTService
	token, err := env.GenerateTestToken("cookie-user", "cookie@example.com")
	require.NoError(t, err)

	// Test with cookie instead of header
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cookie-user")
}
