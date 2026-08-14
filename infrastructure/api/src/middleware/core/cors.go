package core

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"github.com/sirupsen/logrus"
)

func CORS(cfg *config.Config, logger *logrus.Logger) gin.HandlerFunc {
	// CORSOrigins is already a []string slice from config
	allowedOrigins := cfg.CORSOrigins

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Strict Origin Validation - only allow whitelisted origins
		if origin != "" && isOriginAllowed(origin, allowedOrigins) {
			reqHeaders := c.Request.Header.Get("Access-Control-Request-Headers")
			allowHeaders := "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, x-csrf-token, Authorization, accept, origin, Cache-Control, X-Requested-With"
			if reqHeaders != "" {
				allowHeaders = allowHeaders + ", " + reqHeaders
			}
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", allowHeaders)
			c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
			c.Header("Access-Control-Max-Age", "86400")
		} else if origin != "" {
			// Log rejected origins for security monitoring
			logger.WithFields(logrus.Fields{
				"origin":     origin,
				"ip":         c.ClientIP(),
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"request_id": c.GetString("request_id"),
			}).Warn("CORS: Rejected origin not in whitelist")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// isOriginAllowed checks if the origin is in the whitelist
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	// Empty whitelist = deny all (fail-safe)
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}

	return false
}
