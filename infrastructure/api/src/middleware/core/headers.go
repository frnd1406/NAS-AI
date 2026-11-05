package core

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// apiCSP is a restrictive Content-Security-Policy for a JSON API: no
// resources are allowed to load and the responses cannot be framed.
const apiCSP = "default-src 'none'; frame-ancestors 'none'"

// SecureHeaders (http.Handler style) sets strict security headers.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", apiCSP)
		// Minimize recon: never advertise the server software/framework.
		w.Header().Del("X-Powered-By")
		w.Header().Del("Server")

		next.ServeHTTP(w, r)
	})
}

// GinSecureHeaders adapts SecureHeaders for gin.
func GinSecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", apiCSP)
		// Minimize recon: never advertise the server software/framework.
		h.Del("X-Powered-By")
		h.Del("Server")
		c.Next()
	}
}
