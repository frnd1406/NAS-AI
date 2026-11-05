package system

import (
	"errors"
	"net/http"
	"strings"

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

const (
	// The endpoint is public (the frontend reports errors before login), so the
	// body and every logged field are bounded to keep it from flooding the log.
	maxFrontendLogBody    = 16 << 10
	maxFrontendLogMessage = 2000
	maxFrontendLogURL     = 512
)

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return strings.ToValidUTF8(s[:max], "") + "...[truncated]"
}

// FrontendLogHandler receives critical errors from the frontend
func FrontendLogHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFrontendLogBody)

		var entry FrontendLogEntry
		if err := c.ShouldBindJSON(&entry); err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "log entry too large"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid log format"})
			return
		}
		entry.Message = truncate(entry.Message, maxFrontendLogMessage)
		entry.URL = truncate(entry.URL, maxFrontendLogURL)

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
