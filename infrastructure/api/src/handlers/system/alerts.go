package system

import (
		"github.com/nas-ai/api/src/repository/system"
"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sirupsen/logrus"
)

// SystemAlertsListHandler returns all open alerts.
func SystemAlertsListHandler(repo *system_repo.SystemAlertsRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		alerts, err := repo.ListOpen(c.Request.Context())
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("failed to list system alerts")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "failed to load alerts",
					"request_id": requestID,
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items": alerts,
		})
	}
}

// SystemAlertResolveHandler marks an alert as resolved.
func SystemAlertResolveHandler(repo *system_repo.SystemAlertsRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "missing alert id",
					"request_id": requestID,
				},
			})
			return
		}

		updated, err := repo.Resolve(c.Request.Context(), id)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
				"alert_id":   id,
			}).Error("failed to resolve alert")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "failed to resolve alert",
					"request_id": requestID,
				},
			})
			return
		}

		if !updated {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":       "not_found",
					"message":    "alert not found or already resolved",
					"request_id": requestID,
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     "resolved",
			"alert_id":   id,
			"request_id": requestID,
		})
	}
}
