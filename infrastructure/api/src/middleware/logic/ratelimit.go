package logic

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"golang.org/x/time/rate"
)

// RateLimiter middleware implements rate limiting per IP address
// Uses token bucket algorithm with TTL-based cleanup
type RateLimiter struct {
	limiters map[string]*limiterEntry
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	ttl      time.Duration
}

// limiterEntry stores limiter with last access time for TTL cleanup
type limiterEntry struct {
	limiter        *rate.Limiter
	lastAccessUnix int64 // Use atomic operations for thread-safe updates
}

// NewRateLimiter creates a new rate limiter with TTL-based cleanup
func NewRateLimiter(cfg *config.Config) *RateLimiter {
	// Convert requests/min to requests/second
	r := rate.Limit(float64(cfg.RateLimitPerMin) / 60.0)

	rl := &RateLimiter{
		limiters: make(map[string]*limiterEntry),
		rate:     r,
		burst:    cfg.RateLimitPerMin, // Allow burst up to limit
		ttl:      10 * time.Minute,    // TTL for inactive IPs
	}

	// Start background cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// getLimiter gets or creates limiter for IP (thread-safe with atomic updates)
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	now := time.Now()

	// Fast path: read lock
	rl.mu.RLock()
	entry, exists := rl.limiters[ip]
	if exists {
		// Atomic update of last access time (thread-safe under RLock)
		atomic.StoreInt64(&entry.lastAccessUnix, now.Unix())
		limiter := entry.limiter
		rl.mu.RUnlock()
		return limiter
	}
	rl.mu.RUnlock()

	// Slow path: write lock (create new limiter)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock (avoid race)
	if entry, exists := rl.limiters[ip]; exists {
		atomic.StoreInt64(&entry.lastAccessUnix, now.Unix())
		return entry.limiter
	}

	// Create new limiter with current timestamp
	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters[ip] = &limiterEntry{
		limiter:        limiter,
		lastAccessUnix: now.Unix(),
	}

	return limiter
}

// cleanupLoop periodically removes expired limiters (TTL-based)
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute) // Cleanup every 2 minutes
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes limiters that haven't been accessed within TTL
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	toDelete := make([]string, 0)

	// Find expired entries using atomic load
	for ip, entry := range rl.limiters {
		lastAccess := time.Unix(atomic.LoadInt64(&entry.lastAccessUnix), 0)
		if now.Sub(lastAccess) > rl.ttl {
			toDelete = append(toDelete, ip)
		}
	}

	// Delete expired entries
	for _, ip := range toDelete {
		delete(rl.limiters, ip)
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			// Rate limit exceeded
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":        "rate_limit_exceeded",
					"message":     "Too many requests. Please try again later.",
					"retry_after": time.Second * 60, // Suggest retry after 1 minute
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
