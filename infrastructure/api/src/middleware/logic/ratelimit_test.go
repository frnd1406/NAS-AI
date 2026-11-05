package logic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_AllowedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create rate limiter with 10 requests per minute
	cfg := &config.Config{
		RateLimitPerMin: 10,
	}
	rl := NewRateLimiter(cfg)

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request should succeed
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_ExceedLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create rate limiter with very low limit (2 requests per minute)
	cfg := &config.Config{
		RateLimitPerMin: 2,
	}
	rl := NewRateLimiter(cfg)

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Burst allows initial requests
	// First 2 requests should succeed (burst = 2)
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:1234" // Same IP
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// 3rd request should be rate limited
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:1234" // Same IP
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "rate_limit_exceeded")
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RateLimitPerMin: 2,
	}
	rl := NewRateLimiter(cfg)

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Exhaust limit for IP1
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// IP1 should now be rate limited
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.100:1234"
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusTooManyRequests, w1.Code)

	// IP2 should still be allowed (separate limiter)
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.200:5678"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestRateLimiter_BurstAllowance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 60 requests per minute = burst of 60
	cfg := &config.Config{
		RateLimitPerMin: 60,
	}
	rl := NewRateLimiter(cfg)

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Should allow burst of 60 requests immediately
	successCount := 0
	for i := 0; i < 60; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusOK {
			successCount++
		}
	}

	// All 60 burst requests should succeed
	assert.Equal(t, 60, successCount)

	// 61st request should be rate limited
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimiter_LimiterCreation(t *testing.T) {
	cfg := &config.Config{
		RateLimitPerMin: 100,
	}
	rl := NewRateLimiter(cfg)

	// Test getLimiter creates new limiters for new IPs
	limiter1 := rl.getLimiter("192.168.1.1")
	limiter2 := rl.getLimiter("192.168.1.2")

	assert.NotNil(t, limiter1)
	assert.NotNil(t, limiter2)
	assert.False(t, limiter1 == limiter2, "Different IPs should have different limiters")

	// Same IP should return same limiter
	limiter1Again := rl.getLimiter("192.168.1.1")
	assert.Equal(t, limiter1, limiter1Again, "Same IP should return same limiter")
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RateLimitPerMin: 100,
	}
	rl := NewRateLimiter(cfg)

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Simulate concurrent requests from same IP
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:1234"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// No panics = success (testing concurrent map access safety)
	assert.True(t, true)
}
