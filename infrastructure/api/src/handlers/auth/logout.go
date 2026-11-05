package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/services/security"

	"github.com/sirupsen/logrus"
)

// LogoutHandler godoc
// @Summary Logout user
// @Description Invalidates access token by blacklisting it
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Logged out successfully"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/logout [post]
func LogoutHandler(
	jwtService security.JWTServiceInterface,
	redis *database.RedisClient,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		userID := c.GetString("user_id")

		// Get access token from cookie or header (using helper for compatibility)
		tokenString := GetAccessToken(c)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "Missing authorization token",
					"request_id": requestID,
				},
			})
			return
		}

		// Extract claims to get expiry time
		claims, err := jwtService.ExtractClaims(tokenString)
		if err != nil {
			logger.WithError(err).Warn("Failed to extract token claims for logout")
			// Still allow logout even if we can't parse the token
		}

		// Add token to blacklist in Redis
		// TTL = time until token expires (so we don't store it forever)
		ctx := context.Background()
		var ttl time.Duration
		if claims != nil && claims.ExpiresAt != nil {
			ttl = time.Until(claims.ExpiresAt.Time)
			if ttl < 0 {
				ttl = 1 * time.Hour // Already expired, but blacklist for 1 hour anyway
			}
		} else {
			ttl = 1 * time.Hour // Default TTL if we can't determine expiry
		}

		if err := redis.Set(ctx, "blacklist:"+tokenString, "1", ttl).Err(); err != nil {
			logger.WithError(err).Error("Failed to blacklist token")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Logout failed",
					"request_id": requestID,
				},
			})
			return
		}

		// Audit log
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"user_id":    userID,
			"ip":         c.ClientIP(),
		}).Info("User logged out successfully")

		// Clear auth cookies (for cookie-based auth)
		ClearAuthCookies(c)

		c.JSON(http.StatusOK, gin.H{
			"message": "Logged out successfully",
		})
	}
}
