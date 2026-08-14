package logic

import (
	"net/http"
	"strings"

	"github.com/nas-ai/api/src/domain/auth"
	auth_repo "github.com/nas-ai/api/src/repository/auth"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"

	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// Cookie name for access token (must match handlers/cookies.go)
const accessTokenCookieName = "access_token"

// getAccessTokenFromRequest extracts access token from cookie first, falls back to Authorization header
// This enables gradual migration from header-based to cookie-based auth
func getAccessTokenFromRequest(c *gin.Context) string {
	// Try cookie first (new secure method)
	if token, err := c.Cookie(accessTokenCookieName); err == nil && token != "" {
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

// AuthMiddleware validates JWT tokens and checks blacklist
// Supports both cookie-based (new) and header-based (legacy) token retrieval
func AuthMiddleware(jwtService security.JWTServiceInterface, tokenService security.TokenServiceInterface, redis *database.RedisClient, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		// Get access token from cookie first, fallback to Authorization header
		tokenString := getAccessTokenFromRequest(c)
		if tokenString == "" {
			logger.WithField("request_id", requestID).Warn("No access token provided")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "Missing authorization token",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		// Validate JWT token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("Invalid JWT token")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "Invalid or expired token",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		// Check if token is blacklisted (logout revocation)
		ctx := c.Request.Context()
		blacklisted, err := redis.Get(ctx, "blacklist:"+tokenString).Result()
		if err == nil && blacklisted == "1" {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"user_id":    claims.UserID,
			}).Warn("Blacklisted token used")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "Token has been revoked",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		// Check if user tokens have been revoked via MinIAT (Password Reset)
		if tokenService.IsTokenRevoked(ctx, claims.UserID, claims.IssuedAt.Time.Unix()) {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"user_id":    claims.UserID,
			}).Warn("Token revoked by password reset (MinIAT)")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "Session expired due to security change",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		// Store user info in context for handlers
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)

		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"user_id":    claims.UserID,
		}).Debug("User authenticated successfully")

		c.Next()
	}
}

// RequireRole creates middleware that enforces role-based access control
// IMPORTANT: Must be used AFTER AuthMiddleware
func RequireRole(userRepo auth_repo.UserRepositoryInterface, requiredRole auth.UserRole, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		userID := c.GetString("user_id")

		// Check if user_id exists in context (should be set by AuthMiddleware)
		if userID == "" {
			logger.WithField("request_id", requestID).Error("RBAC: user_id not found in context - AuthMiddleware not executed?")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "Authentication required",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		// Fetch user from database to check role
		ctx := c.Request.Context()
		user, err := userRepo.FindByID(ctx, userID)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"user_id":    userID,
				"error":      err.Error(),
			}).Error("RBAC: failed to fetch user")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to verify permissions",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		if user == nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"user_id":    userID,
			}).Warn("RBAC: user not found in database")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "User not found",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		// Check if user has required role
		if user.Role != requiredRole {
			logger.WithFields(logrus.Fields{
				"request_id":    requestID,
				"user_id":       userID,
				"user_role":     user.Role,
				"required_role": requiredRole,
			}).Warn("RBAC: access denied - insufficient permissions")
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":       "forbidden",
					"message":    "Insufficient permissions - admin access required",
					"request_id": requestID,
				},
			})
			c.Abort()
			return
		}

		// Store user role in context for downstream handlers
		c.Set("user_role", string(user.Role))

		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"user_id":    userID,
			"user_role":  user.Role,
		}).Debug("RBAC: access granted")

		c.Next()
	}
}

// AdminOnly is a convenience function for requiring admin role
func AdminOnly(userRepo auth_repo.UserRepositoryInterface, logger *logrus.Logger) gin.HandlerFunc {
	return RequireRole(userRepo, auth.RoleAdmin, logger)
}
