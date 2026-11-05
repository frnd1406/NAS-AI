package auth

import (
	"net/http"

	auth_repo "github.com/nas-ai/api/src/repository/auth"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"

	"github.com/nas-ai/api/src/services/operations"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// ForgotPasswordRequest represents the forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest represents the reset password request
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ForgotPasswordHandler godoc
// @Summary Request password reset
// @Description Sends password reset email if account exists (no user enumeration)
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Email address"
// @Success 200 {object} map[string]interface{} "Reset email sent if account exists"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Router /auth/forgot-password [post]
func ForgotPasswordHandler(
	userRepo *auth_repo.UserRepository,
	tokenService *security.TokenService,
	emailService *operations.EmailService,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var req ForgotPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("Invalid forgot password request")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "Invalid request body",
					"request_id": requestID,
				},
			})
			return
		}

		ctx := c.Request.Context()

		// IMPORTANT: Always return 200 OK to prevent user enumeration
		// Find user by email
		user, err := userRepo.FindByEmail(ctx, req.Email)
		if err != nil {
			logger.WithError(err).Error("Failed to find user by email")
		}

		// If user exists, send password reset email
		if user != nil {
			// Generate reset token
			token, err := tokenService.GeneratePasswordResetToken(ctx, user.ID)
			if err != nil {
				logger.WithError(err).Error("Failed to generate password reset token")
			} else {
				// Send password reset email
				if err := emailService.SendPasswordResetEmail(user.Email, user.Username, token); err != nil {
					logger.WithError(err).Error("Failed to send password reset email")
				} else {
					logger.WithFields(logrus.Fields{
						"request_id": requestID,
						"user_id":    user.ID,
						"email":      req.Email,
						"ip":         c.ClientIP(),
					}).Info("Password reset email sent")
				}
			}
		}

		// Audit log (always log, even if user doesn't exist)
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"email":      req.Email,
			"ip":         c.ClientIP(),
			"found":      user != nil,
		}).Info("Password reset requested")

		// Always return success (no user enumeration)
		c.JSON(http.StatusOK, gin.H{
			"message": "If the email exists, a password reset link has been sent",
		})
	}
}

// ResetPasswordHandler godoc
// @Summary Reset password
// @Description Confirms password reset using token and sets new password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset token and new password"
// @Success 200 {object} map[string]interface{} "Password reset successful"
// @Failure 400 {object} map[string]interface{} "Invalid token or weak password"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/reset-password [post]
func ResetPasswordHandler(
	userRepo *auth_repo.UserRepository,
	tokenService *security.TokenService,
	passwordService *security.PasswordService,
	jwtService *security.JWTService,
	redis *database.RedisClient,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var req ResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("Invalid reset password request")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "Invalid request body",
					"request_id": requestID,
				},
			})
			return
		}

		ctx := c.Request.Context()

		// Validate reset token
		userID, err := tokenService.ValidatePasswordResetToken(ctx, req.Token)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("Invalid password reset token")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_token",
					"message":    "Invalid or expired password reset token",
					"request_id": requestID,
				},
			})
			return
		}

		// Validate password strength
		if err := passwordService.ValidatePasswordStrength(req.NewPassword); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "weak_password",
					"message":    err.Error(),
					"request_id": requestID,
				},
			})
			return
		}

		// Hash new password
		newPasswordHash, err := passwordService.HashPassword(req.NewPassword)
		if err != nil {
			logger.WithError(err).Error("Failed to hash password")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to reset password",
					"request_id": requestID,
				},
			})
			return
		}

		// Update password in database
		if err := userRepo.UpdatePassword(ctx, userID, newPasswordHash); err != nil {
			logger.WithError(err).Error("Failed to update password")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to reset password",
					"request_id": requestID,
				},
			})
			return
		}

		// Invalidate all existing tokens for this user (Security)
		if err := tokenService.InvalidateUserTokens(ctx, userID); err != nil {
			logger.WithError(err).Error("Failed to invalidate user tokens after password reset")
			// Non-fatal, but logged
		}

		// Audit log
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"user_id":    userID,
			"ip":         c.ClientIP(),
		}).Info("Password reset successfully")

		c.JSON(http.StatusOK, gin.H{
			"message": "Password reset successfully",
		})
	}
}
