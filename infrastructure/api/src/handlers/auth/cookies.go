package auth

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Cookie configuration constants
const (
	// Cookie names
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"

	// Cookie expiration times (must match JWT expiry in jwt_service.go)
	AccessTokenMaxAge  = 15 * 60       // 15 minutes in seconds
	RefreshTokenMaxAge = 7 * 24 * 3600 // 7 days in seconds
)

// CookieConfig holds the configuration for auth cookies
type CookieConfig struct {
	Domain   string
	Secure   bool
	SameSite http.SameSite
}

// getCookieConfig returns the cookie configuration based on environment
func getCookieConfig(c *gin.Context) CookieConfig {
	env := c.GetString("environment")
	isProduction := env == "production"

	// Determine domain from request or environment
	domain := os.Getenv("COOKIE_DOMAIN")
	if domain == "" {
		// Extract domain from Host header for dynamic configuration
		host := c.Request.Host
		// Remove port if present
		if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
			host = host[:colonIdx]
		}
		// For production with a real domain, prefix with dot for subdomain support
		if isProduction && !isLocalhost(host) {
			domain = "." + getBaseDomain(host)
		}
		// For localhost/development, leave empty (browser will use request origin)
	}

	return CookieConfig{
		Domain:   domain,
		Secure:   isProduction || isSecureRequest(c),
		SameSite: http.SameSiteStrictMode,
	}
}

// isSecureRequest reports whether the request reached us over HTTPS, either
// directly (TLS terminated here) or via a TLS-terminating reverse proxy that
// sets X-Forwarded-Proto: https. Without this, cookies would drop the Secure
// flag when TLS is terminated at the proxy.
func isSecureRequest(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	proto := c.GetHeader("X-Forwarded-Proto")
	// May be a comma-separated list (proxy chain); the first entry is the client.
	if idx := strings.IndexByte(proto, ','); idx != -1 {
		proto = proto[:idx]
	}
	return strings.EqualFold(strings.TrimSpace(proto), "https")
}

// isLocalhost checks if the host is a localhost variant
func isLocalhost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || strings.HasPrefix(host, "192.168.")
}

// getBaseDomain extracts the base domain (e.g., "example.com" from "api.example.com")
func getBaseDomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return host
}

// SetAuthCookies sets the access and refresh tokens as HttpOnly cookies
func SetAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	cfg := getCookieConfig(c)

	// Access Token Cookie
	// Path: "/" - available for all API endpoints
	c.SetSameSite(cfg.SameSite)
	c.SetCookie(
		AccessTokenCookieName,
		accessToken,
		AccessTokenMaxAge,
		"/",
		cfg.Domain,
		cfg.Secure,
		true, // HttpOnly - NOT accessible via JavaScript
	)

	// Refresh Token Cookie
	// Path: "/auth" - only sent to auth endpoints (minimizes exposure)
	c.SetCookie(
		RefreshTokenCookieName,
		refreshToken,
		RefreshTokenMaxAge,
		"/auth",
		cfg.Domain,
		cfg.Secure,
		true, // HttpOnly
	)
}

// SetAccessTokenCookie sets only the access token cookie (used during refresh)
func SetAccessTokenCookie(c *gin.Context, accessToken string) {
	cfg := getCookieConfig(c)

	c.SetSameSite(cfg.SameSite)
	c.SetCookie(
		AccessTokenCookieName,
		accessToken,
		AccessTokenMaxAge,
		"/",
		cfg.Domain,
		cfg.Secure,
		true, // HttpOnly
	)
}

// ClearAuthCookies removes both auth cookies (used during logout)
func ClearAuthCookies(c *gin.Context) {
	cfg := getCookieConfig(c)

	// Clear access token cookie
	c.SetSameSite(cfg.SameSite)
	c.SetCookie(
		AccessTokenCookieName,
		"",
		-1, // Negative MaxAge = delete cookie
		"/",
		cfg.Domain,
		cfg.Secure,
		true,
	)

	// Clear refresh token cookie
	c.SetCookie(
		RefreshTokenCookieName,
		"",
		-1,
		"/auth",
		cfg.Domain,
		cfg.Secure,
		true,
	)
}

// GetAccessToken extracts access token from cookie, falling back to Authorization header
// This enables backward compatibility during migration
func GetAccessToken(c *gin.Context) string {
	// Try cookie first (new secure method)
	if token, err := c.Cookie(AccessTokenCookieName); err == nil && token != "" {
		return token
	}

	// Fallback to Authorization header (legacy/backward compat)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	return ""
}

// GetRefreshToken extracts refresh token from cookie, falling back to JSON body
func GetRefreshToken(c *gin.Context) string {
	// Try cookie first
	if token, err := c.Cookie(RefreshTokenCookieName); err == nil && token != "" {
		return token
	}
	return ""
}
