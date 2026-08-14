package core

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID middleware generates a unique ID for each request
// Used for distributed tracing and log correlation
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request already has an ID (from upstream proxy)
		requestID := c.GetHeader("X-Request-ID")

		if requestID == "" {
			// Generate new UUID
			requestID = uuid.New().String()
		}

		// Store in context for use by other middleware/handlers
		c.Set("request_id", requestID)

		// Set response header
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
