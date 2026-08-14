package system

import (
		"github.com/nas-ai/api/src/repository/system"
"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/sirupsen/logrus"
)

type AlertCreateRequest struct {
	Severity string `json:"severity" binding:"required"`
	Message  string `json:"message" binding:"required"`
}

// SystemAlertCreateHandler creates a new alert entry.
func SystemAlertCreateHandler(repo *system_repo.SystemAlertsRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var req AlertCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "invalid payload",
					"request_id": requestID,
				},
			})
			return
		}

		severity := strings.ToUpper(strings.TrimSpace(req.Severity))
		message := strings.TrimSpace(req.Message)

		if severity == "" || message == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "severity and message are required",
					"request_id": requestID,
				},
			})
			return
		}

		if err := repo.Create(c.Request.Context(), severity, message); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("failed to create alert")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "failed to create alert",
					"request_id": requestID,
				},
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":     "created",
			"request_id": requestID,
		})
	}
}
