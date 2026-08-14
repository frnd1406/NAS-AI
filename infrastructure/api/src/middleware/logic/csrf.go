package logic

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
)

const csrfSessionCookie = "csrf_session"

// CSRFMiddleware validates CSRF tokens for state-changing requests
func CSRFMiddleware(redis *database.RedisClient, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		// Only validate CSRF for POST, PUT, DELETE, PATCH
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Extract CSRF token from header
		csrfToken := c.GetHeader("X-CSRF-Token")
		if csrfToken == "" {
			logger.WithField("request_id", requestID).Warn("Missing CSRF token")
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":       "csrf_token_missing",
					"message":    "CSRF token is required",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		sessionID, err := c.Cookie(csrfSessionCookie)
		if err != nil || sessionID == "" {
			logger.WithField("request_id", requestID).Warn("Missing CSRF session cookie")
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":       "csrf_validation_failed",
					"message":    "Invalid CSRF token",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		// FIX [BUG-GO-017]: Use request context with timeout instead of Background
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		key := "csrf:" + sessionID
		storedToken, err := redis.Get(ctx, key).Result()

		if err != nil || subtle.ConstantTimeCompare([]byte(storedToken), []byte(csrfToken)) != 1 {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"session":    sessionID,
			}).Warn("Invalid CSRF token")
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":       "csrf_validation_failed",
					"message":    "Invalid CSRF token",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"session":    sessionID,
		}).Debug("CSRF token validated successfully")

		c.Next()
	}
}

// GenerateCSRFToken generates and stores a token for a session.
func GenerateCSRFToken(redis *database.RedisClient, sessionID string) (string, error) {
	// Generate 32-byte random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// FIX [BUG-GO-017]: Add timeout to prevent hanging on slow Redis
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Store in Redis with 24-hour expiry
	key := "csrf:" + sessionID
	if err := redis.Set(ctx, key, token, 24*time.Hour).Err(); err != nil {
		return "", err
	}

	return token, nil
}

// SetCSRFCookie sets the session cookie for CSRF tokens.
func SetCSRFCookie(c *gin.Context, sessionID string) {
	c.SetCookie(csrfSessionCookie, sessionID, 24*3600, "/", "", true, true)
}

// EnsureCSRFSession ensures a session ID exists (cookie) and returns it.
func EnsureCSRFSession(c *gin.Context) (string, error) {
	if sessionID, err := c.Cookie(csrfSessionCookie); err == nil && sessionID != "" {
		return sessionID, nil
	}
	// create new session id
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
