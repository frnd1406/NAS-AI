package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/domain/auth"
	"github.com/nas-ai/api/src/middleware/logic"
	"github.com/nas-ai/api/src/services/security"

	"github.com/sirupsen/logrus"
)

// issueSession completes a successful authentication: it rotates the CSRF
// session, issues JWT access/refresh cookies and a CSRF token, and writes the
// standard LoginResponse. It is shared by the password-only login path and the
// WebAuthn second-factor finish so both stay identical. On error it writes the
// response itself and returns the error.
func issueSession(
	c *gin.Context,
	user *auth.User,
	jwtService security.JWTServiceInterface,
	redis *database.RedisClient,
	logger *logrus.Logger,
) error {
	requestID := c.GetString("request_id")

	// Rotate session to avoid fixation.
	sessionID, err := logic.EnsureCSRFSession(c)
	if err != nil {
		logger.WithError(err).Error("Failed to create CSRF session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return err
	}

	accessToken, err := jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		logger.WithError(err).Error("Failed to generate access token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return err
	}

	refreshToken, err := jwtService.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		logger.WithError(err).Error("Failed to generate refresh token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return err
	}

	csrfToken, err := logic.GenerateCSRFToken(redis, sessionID)
	if err != nil {
		logger.WithError(err).Error("Failed to generate CSRF token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return err
	}
	logic.SetCSRFCookie(c, sessionID)

	logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"user_id":    user.ID,
		"email":      user.Email,
		"ip":         c.ClientIP(),
	}).Info("User session issued")

	SetAuthCookies(c, accessToken, refreshToken)

	c.JSON(http.StatusOK, LoginResponse{
		User:      user.ToResponse(),
		CSRFToken: csrfToken,
	})
	return nil
}
