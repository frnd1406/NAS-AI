package auth

import (
		"github.com/nas-ai/api/src/repository/auth"
"net/http"

	"github.com/gin-gonic/gin"

	
	"github.com/sirupsen/logrus"
	"github.com/nas-ai/api/src/services/operations"
	"github.com/nas-ai/api/src/services/security"
)

// VerifyEmailRequest represents the email verification request
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// VerifyEmailHandler godoc
// @Summary Verify email address
// @Description Verifies user's email address using token from verification email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body VerifyEmailRequest true "Verification token"
// @Success 200 {object} map[string]interface{} "Email verified successfully"
// @Failure 400 {object} map[string]interface{} "Invalid or expired token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/verify-email [post]
func VerifyEmailHandler(
	userRepo *auth_repo.UserRepository,
	tokenService *security.TokenService,
	emailService *operations.EmailService,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var req VerifyEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("Invalid verification request")
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

		// Validate token and get user ID
		userID, err := tokenService.ValidateVerificationToken(ctx, req.Token)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("Invalid verification token")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_token",
					"message":    "Invalid or expired verification token",
					"request_id": requestID,
				},
			})
			return
		}

		// Mark user email as verified
		if err := userRepo.VerifyEmail(ctx, userID); err != nil {
			logger.WithError(err).Error("Failed to verify user email")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to verify email",
					"request_id": requestID,
				},
			})
			return
		}

		// Get user details to send welcome email
		user, err := userRepo.FindByID(ctx, userID)
		if err == nil && user != nil {
			// Send welcome email (non-blocking - ignore errors)
			go emailService.SendWelcomeEmail(user.Email, user.Username)
		}

		// Audit log
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"user_id":    userID,
			"ip":         c.ClientIP(),
		}).Info("User email verified successfully")

		c.JSON(http.StatusOK, gin.H{
			"message": "Email verified successfully",
		})
	}
}

// ResendVerificationHandler godoc
// @Summary Resend verification email
// @Description Resends the verification email to authenticated user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Verification email sent"
// @Failure 400 {object} map[string]interface{} "Email already verified"
// @Failure 401 {object} map[string]interface{} "Not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/resend-verification [post]
func ResendVerificationHandler(
	userRepo *auth_repo.UserRepository,
	tokenService *security.TokenService,
	emailService *operations.EmailService,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		userID := c.GetString("user_id")

		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "User not authenticated",
					"request_id": requestID,
				},
			})
			return
		}

		ctx := c.Request.Context()

		// Get user
		user, err := userRepo.FindByID(ctx, userID)
		if err != nil {
			logger.WithError(err).Error("Failed to find user")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to resend verification",
					"request_id": requestID,
				},
			})
			return
		}

		if user == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":       "user_not_found",
					"message":    "User not found",
					"request_id": requestID,
				},
			})
			return
		}

		// Check if already verified
		if user.EmailVerified {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "already_verified",
					"message":    "Email is already verified",
					"request_id": requestID,
				},
			})
			return
		}

		// Generate new verification token
		token, err := tokenService.GenerateVerificationToken(ctx, user.ID)
		if err != nil {
			logger.WithError(err).Error("Failed to generate verification token")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to resend verification",
					"request_id": requestID,
				},
			})
			return
		}

		// Send verification email
		if err := emailService.SendVerificationEmail(user.Email, user.Username, token); err != nil {
			logger.WithError(err).Error("Failed to send verification email")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to send verification email",
					"request_id": requestID,
				},
			})
			return
		}

		// Audit log
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"user_id":    userID,
			"ip":         c.ClientIP(),
		}).Info("Verification email resent")

		c.JSON(http.StatusOK, gin.H{
			"message": "Verification email sent",
		})
	}
}
