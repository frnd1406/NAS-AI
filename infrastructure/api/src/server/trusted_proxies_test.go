package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// With no trusted proxies, a spoofed X-Forwarded-For must not change ClientIP.
func TestClientIPIgnoresForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	var got string
	r.GET("/", func(c *gin.Context) { got = c.ClientIP() })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	r.ServeHTTP(httptest.NewRecorder(), req)

	if got != "192.0.2.10" {
		t.Fatalf("ClientIP = %q, want 192.0.2.10", got)
	}
}
