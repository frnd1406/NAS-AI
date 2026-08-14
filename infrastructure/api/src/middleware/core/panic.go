package core

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PanicRecovery middleware catches panics and returns 500 Internal Server Error
// This prevents the entire server from crashing on handler panics
func PanicRecovery(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := debug.Stack()

				// Log the panic (sanitized - no sensitive data!)
				logger.WithFields(logrus.Fields{
					"error":      fmt.Sprintf("%v", err),
					"stack":      string(stack),
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"ip":         c.ClientIP(),
					"request_id": c.GetString("request_id"), // Set by RequestID middleware
				}).Error("PANIC RECOVERED")

				// Return 500 to client (don't expose stack trace!)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "internal_server_error",
						"message": "An unexpected error occurred",
						// Include request ID for debugging
						"request_id": c.GetString("request_id"),
					},
				})

				// Abort further processing
				c.Abort()
			}
		}()

		c.Next()
	}
}
