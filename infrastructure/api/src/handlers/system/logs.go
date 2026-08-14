package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type FrontendLogEntry struct {
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
	URL     string                 `json:"url"`
	Time    string                 `json:"time"`
}

// FrontendLogHandler receives critical errors from the frontend
func FrontendLogHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var entry FrontendLogEntry
		if err := c.ShouldBindJSON(&entry); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid log format"})
			return
		}

		// Log using backend logger, tagged as "frontend"
		logField := logger.WithFields(logrus.Fields{
			"source": "frontend",
			"url":    entry.URL,
			"ctx":    entry.Context,
		})

		switch entry.Level {
		case "error":
			logField.Error(entry.Message)
		case "warn":
			logField.Warn(entry.Message)
		case "info":
			logField.Info(entry.Message)
		default:
			logField.Info(entry.Message)
		}

		c.Status(http.StatusOK)
	}
}
