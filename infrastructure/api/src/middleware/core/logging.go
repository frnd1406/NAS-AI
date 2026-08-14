package core

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuditLogger middleware logs all requests with structured logging
// Implements audit trail for security compliance
// Format: JSON structured logging (SECURITY_HANDBOOK.pdf §3.2)
func AuditLogger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get request ID (set by RequestID middleware)
		requestID := c.GetString("request_id")

		// Get user ID (will be set by Auth middleware in Phase 2)
		userID := c.GetString("user_id")
		if userID == "" {
			userID = "anonymous"
		}

		// Log entry
		entry := logger.WithFields(logrus.Fields{
			"request_id":  requestID,
			"timestamp":   start.Format(time.RFC3339),
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"query":       c.Request.URL.RawQuery,
			"status":      c.Writer.Status(),
			"duration_ms": duration.Milliseconds(),
			"ip":          c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
			"user_id":     userID,
			"bytes_sent":  c.Writer.Size(),
		})

		// Log level based on status code
		status := c.Writer.Status()
		switch {
		case status >= 500:
			entry.Error("Server error")
		case status >= 400:
			entry.Warn("Client error")
		case status >= 300:
			entry.Info("Redirect")
		default:
			entry.Info("Request completed")
		}

		// Special audit logging for sensitive operations
		// (Auth endpoints, data modifications, etc.)
		if c.Request.Method != "GET" && c.Request.Method != "OPTIONS" {
			logger.WithFields(logrus.Fields{
				"audit":      true,
				"request_id": requestID,
				"user_id":    userID,
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"ip":         c.ClientIP(),
				"status":     status,
				"timestamp":  start.Format(time.RFC3339),
			}).Info("AUDIT: Data modification request")
		}
	}
}
