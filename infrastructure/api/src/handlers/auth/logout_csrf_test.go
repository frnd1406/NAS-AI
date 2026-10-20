package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/middleware/logic"
	"github.com/nas-ai/api/src/services/security"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// logoutFixture wires the real /auth routes (RegisterGlobalRoutes) so the test
// covers the actual middleware chain of POST /auth/logout.
func logoutFixture(t *testing.T) (router *gin.Engine, redisClient *database.RedisClient, accessToken string) {
	t.Helper()

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	redisClient = &database.RedisClient{Client: redis.NewClient(&redis.Options{Addr: mr.Addr()})}

	cfg := &config.Config{JWTSecret: "test-secret-at-least-32-chars-long-secure"}
	jwtService, err := security.NewJWTService(cfg, logger)
	require.NoError(t, err)
	tokenService := security.NewTokenService(redisClient, logger)

	accessToken, err = jwtService.GenerateAccessToken("user-123", "test@example.com")
	require.NoError(t, err)

	h := NewHandler(cfg, nil, jwtService, nil, tokenService, nil, nil, nil, nil, redisClient, logger)

	gin.SetMode(gin.TestMode)
	router = gin.New()
	h.RegisterGlobalRoutes(router.Group("/auth"))
	return router, redisClient, accessToken
}

func logoutRequest(accessToken, csrfSession, csrfHeader string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: accessToken})
	if csrfSession != "" {
		req.AddCookie(&http.Cookie{Name: "csrf_session", Value: csrfSession})
	}
	if csrfHeader != "" {
		req.Header.Set("X-CSRF-Token", csrfHeader)
	}
	return req
}

func TestLogout_RequiresCSRFToken(t *testing.T) {
	router, redisClient, accessToken := logoutFixture(t)

	csrf, err := logic.GenerateCSRFToken(redisClient, "sess-1")
	require.NoError(t, err)

	t.Run("missing header is rejected and token is not revoked", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, logoutRequest(accessToken, "sess-1", ""))
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "csrf_token_missing")
		assert.Error(t, redisClient.Get(t.Context(), "blacklist:"+accessToken).Err(), "token must not be blacklisted")
	})

	t.Run("wrong token is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, logoutRequest(accessToken, "sess-1", "not-the-token"))
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "csrf_validation_failed")
	})

	t.Run("missing csrf session cookie is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, logoutRequest(accessToken, "", csrf))
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("valid token logs out and blacklists the access token", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, logoutRequest(accessToken, "sess-1", csrf))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, redisClient.Get(t.Context(), "blacklist:"+accessToken).Err(), "token must be blacklisted")
	})
}
