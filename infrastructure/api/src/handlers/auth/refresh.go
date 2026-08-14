package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"

	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// RefreshRequest represents the refresh token request (for backward compatibility)
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents the refresh token response
// Note: Access token is now also set as HttpOnly cookie
type RefreshResponse struct {
	AccessToken string `json:"access_token,omitempty"` // Kept for backward compat, will be empty in future
	Success     bool   `json:"success"`
}

// RefreshHandler godoc
// @Summary Refresh access token
// @Description Gets new access token using valid refresh token (from cookie or body)
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshRequest false "Refresh token (optional if using cookies)"
// @Success 200 {object} RefreshResponse "New access token issued"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid or expired refresh token"
// @Router /auth/refresh [post]
func RefreshHandler(
	jwtService security.JWTServiceInterface,
	redis *database.RedisClient,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		// Try to get refresh token from cookie first (new secure method)
		refreshToken := GetRefreshToken(c)

		// Fallback to JSON body for backward compatibility
		if refreshToken == "" {
			var req RefreshRequest
			// Don't fail if no body - we might have cookie
			_ = c.ShouldBindJSON(&req)
			refreshToken = req.RefreshToken
		}

		if refreshToken == "" {
			logger.WithField("request_id", requestID).Warn("No refresh token provided")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "Missing refresh token",
					"request_id": requestID,
				},
			})
			return
		}

		// Validate refresh token
		claims, err := jwtService.ValidateToken(refreshToken)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("Invalid refresh token")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "invalid_token",
					"message":    "Invalid or expired refresh token",
					"request_id": requestID,
				},
			})
			return
		}

		// Check if refresh token is blacklisted
		ctx := context.Background()
		blacklisted, err := redis.Get(ctx, "blacklist:"+refreshToken).Result()
		if err == nil && blacklisted == "1" {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"user_id":    claims.UserID,
			}).Warn("Blacklisted refresh token used")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "token_revoked",
					"message":    "Refresh token has been revoked",
					"request_id": requestID,
				},
			})
			return
		}

		// Verify token type is refresh token
		if claims.TokenType != security.RefreshToken {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"user_id":    claims.UserID,
				"token_type": claims.TokenType,
			}).Warn("Wrong token type used for refresh")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "invalid_token_type",
					"message":    "Invalid token type",
					"request_id": requestID,
				},
			})
			return
		}

		// Generate new access token
		accessToken, err := jwtService.GenerateAccessToken(claims.UserID, claims.Email)
		if err != nil {
			logger.WithError(err).Error("Failed to generate new access token")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Token refresh failed",
					"request_id": requestID,
				},
			})
			return
		}

		// Set new access token as HttpOnly cookie
		SetAccessTokenCookie(c, accessToken)

		// Audit log
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"user_id":    claims.UserID,
			"ip":         c.ClientIP(),
		}).Info("Access token refreshed successfully")

		c.JSON(http.StatusOK, RefreshResponse{
			Success: true,
		})
	}
}
