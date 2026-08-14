package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/middleware/logic"
	"github.com/sirupsen/logrus"
)

// GetCSRFToken returns a CSRF token for the authenticated user
// @Summary Get CSRF Token
// @Description Bootstrap CSRF cookie + token for SPA flows.
// @Tags CSRF
// @Produce json
// @Success 200 {object} map[string]string "CSRF token generated successfully"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/csrf [get]
func GetCSRFToken(redis *database.RedisClient, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		sessionID, err := logic.EnsureCSRFSession(c)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("Failed to ensure CSRF session")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "csrf_generation_failed",
					"message":    "Failed to generate CSRF token",
					"request_id": requestID,
				},
			})
			return
		}

		token, err := logic.GenerateCSRFToken(redis, sessionID)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"session":    sessionID,
				"error":      err.Error(),
			}).Error("Failed to generate CSRF token")

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "csrf_generation_failed",
					"message":    "Failed to generate CSRF token",
					"request_id": requestID,
				},
			})
			return
		}

		logic.SetCSRFCookie(c, sessionID)

		c.JSON(http.StatusOK, gin.H{
			"csrf_token": token,
		})
	}
}
