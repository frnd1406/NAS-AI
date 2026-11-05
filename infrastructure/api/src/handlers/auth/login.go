package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/nas-ai/api/src/services/security"

	"github.com/sirupsen/logrus"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
// Note: Access and Refresh tokens are now sent as HttpOnly cookies, not in JSON body
type LoginResponse struct {
	User      interface{} `json:"user"`
	CSRFToken string      `json:"csrf_token"`
}

// LoginHandler godoc
// @Summary Login user
// @Description Authenticate user and return JWT tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse "Login successful with tokens"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/login [post]
func LoginHandler(
	userRepo auth_repo.UserRepositoryInterface,
	jwtService security.JWTServiceInterface,
	passwordService security.PasswordServiceInterface,
	webAuthnService *security.WebAuthnService,
	credentialRepo *auth_repo.WebAuthnCredentialRepository,
	redis *database.RedisClient,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("Invalid login request")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "Invalid request body",
					"request_id": requestID,
				},
			})
			return
		}

		// Find user by email
		ctx := c.Request.Context()
		user, err := userRepo.FindByEmail(ctx, req.Email)
		if err != nil {
			logger.WithError(err).Error("Failed to find user")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Login failed",
					"request_id": requestID,
				},
			})
			return
		}

		// User not found or invalid password - same error message (security)
		if user == nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"email":      req.Email,
				"ip":         c.ClientIP(),
			}).Warn("Login failed: user not found")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "invalid_credentials",
					"message":    "Invalid email or password",
					"request_id": requestID,
				},
			})
			return
		}

		// Verify password
		if err := passwordService.ComparePassword(user.PasswordHash, req.Password); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"user_id":    user.ID,
				"email":      req.Email,
				"ip":         c.ClientIP(),
			}).Warn("Login failed: invalid password")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "invalid_credentials",
					"message":    "Invalid email or password",
					"request_id": requestID,
				},
			})
			return
		}

		// Second factor: if WebAuthn is enabled and the user has registered at
		// least one authenticator, do NOT issue a session yet. Return an MFA
		// pending token; the client then completes the WebAuthn assertion.
		if webAuthnService != nil && credentialRepo != nil {
			count, err := credentialRepo.CountByUserID(ctx, user.ID)
			if err != nil {
				logger.WithError(err).Error("Failed to check webauthn credentials")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
				return
			}
			if count > 0 {
				mfaToken, err := webAuthnService.GenerateMFAPendingToken(ctx, user.ID)
				if err != nil {
					logger.WithError(err).Error("Failed to create MFA pending token")
					c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
					return
				}
				logger.WithFields(logrus.Fields{
					"request_id": requestID,
					"user_id":    user.ID,
				}).Info("Password OK, second factor required")
				c.JSON(http.StatusOK, gin.H{
					"mfa_required": true,
					"mfa_token":    mfaToken,
					"methods":      []string{"webauthn", "recovery_code"},
				})
				return
			}
		}

		// No second factor: issue the session directly.
		_ = issueSession(c, user, jwtService, redis, logger)
	}
}
