package auth

import (
	"crypto/subtle"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/middleware/logic"
	auth_repo "github.com/nas-ai/api/src/repository/auth"

	"github.com/nas-ai/api/src/services/operations"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Username   string `json:"username" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	InviteCode string `json:"invite_code"` // Optional in struct, enforced by logic
}

// RegisterResponse represents the registration response
type RegisterResponse struct {
	User         interface{} `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	CSRFToken    string      `json:"csrf_token"`
	// Dev convenience: returned only in non-production to allow manual verification without email
	VerificationToken string `json:"verification_token,omitempty"`
}

// RegisterHandler godoc
// @Summary Register a new user
// @Description Creates a new user account with email verification
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration request"
// @Success 201 {object} RegisterResponse "User created successfully with tokens"
// @Failure 400 {object} map[string]interface{} "Invalid request or weak password"
// @Failure 409 {object} map[string]interface{} "Email already registered"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/register [post]
func RegisterHandler(
	cfg *config.Config,
	userRepo auth_repo.UserRepositoryInterface,
	jwtService security.JWTServiceInterface,
	passwordService security.PasswordServiceInterface,
	tokenService security.TokenServiceInterface,
	emailService *operations.EmailService,
	redis *database.RedisClient,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("Invalid registration request")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "Invalid request body",
					"request_id": requestID,
				},
			})
			return
		}

		// Security Check: Invite Code (Phase 7 Hardening)
		if cfg.InviteCode != "" {
			if subtle.ConstantTimeCompare([]byte(req.InviteCode), []byte(cfg.InviteCode)) != 1 {
				logger.WithFields(logrus.Fields{
					"request_id":  requestID,
					"ip":          c.ClientIP(),
					"invite_code": req.InviteCode,
				}).Warn("Registration attempt with invalid invite code")

				c.JSON(http.StatusForbidden, gin.H{
					"error": gin.H{
						"code":       "invalid_invite_code",
						"message":    "Invalid or missing invite code",
						"request_id": requestID,
					},
				})
				return
			}
		}

		// Validate username (min 3 chars)
		if len(req.Username) < 3 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_username",
					"message":    "Username must be at least 3 characters",
					"request_id": requestID,
				},
			})
			return
		}

		// Validate email format
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.Email) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_email",
					"message":    "Invalid email format",
					"request_id": requestID,
				},
			})
			return
		}

		// Check if username already exists
		ctx := c.Request.Context()
		existingUserByUsername, err := userRepo.FindByUsername(ctx, req.Username)
		if err != nil {
			logger.WithError(err).Error("Failed to check existing username")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to create user",
					"request_id": requestID,
				},
			})
			return
		}

		if existingUserByUsername != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":       "username_exists",
					"message":    "Username already registered",
					"request_id": requestID,
				},
			})
			return
		}

		// Validate password strength
		if err := passwordService.ValidatePasswordStrength(req.Password); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "weak_password",
					"message":    err.Error(),
					"request_id": requestID,
				},
			})
			return
		}

		// Check if email already exists
		existingUser, err := userRepo.FindByEmail(ctx, req.Email)
		if err != nil {
			logger.WithError(err).Error("Failed to check existing user")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to create user",
					"request_id": requestID,
				},
			})
			return
		}

		if existingUser != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":       "email_exists",
					"message":    "Email already registered",
					"request_id": requestID,
				},
			})
			return
		}

		// Hash password
		passwordHash, err := passwordService.HashPassword(req.Password)
		if err != nil {
			logger.WithError(err).Error("Failed to hash password")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to create user",
					"request_id": requestID,
				},
			})
			return
		}

		// Create user in database
		user, err := userRepo.CreateUser(ctx, req.Username, req.Email, passwordHash)
		if err != nil {
			logger.WithError(err).Error("Failed to create user")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "Failed to create user",
					"request_id": requestID,
				},
			})
			return
		}

		// Start new CSRF session for this user (auto-login after register)
		sessionID, err := logic.EnsureCSRFSession(c)
		if err != nil {
			logger.WithError(err).Error("Failed to create CSRF session")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}

		// Generate JWT tokens
		accessToken, err := jwtService.GenerateAccessToken(user.ID, user.Email)
		if err != nil {
			logger.WithError(err).Error("Failed to generate access token")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}

		refreshToken, err := jwtService.GenerateRefreshToken(user.ID, user.Email)
		if err != nil {
			logger.WithError(err).Error("Failed to generate refresh token")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}

		// Generate CSRF token
		csrfToken, err := logic.GenerateCSRFToken(redis, sessionID)
		if err != nil {
			logger.WithError(err).Error("Failed to generate CSRF token")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		logic.SetCSRFCookie(c, sessionID)

		verificationToken := ""
		// Generate verification token and send email (non-blocking)
		verifyToken, err2 := tokenService.GenerateVerificationToken(ctx, user.ID)
		if err2 != nil {
			logger.WithError(err2).Error("Failed to generate verification token")
			// Don't fail registration, just log the error
		} else {
			if c.GetString("environment") != "production" {
				verificationToken = verifyToken
			}
			// Send verification email asynchronously
			go func() {
				if err := emailService.SendVerificationEmail(user.Email, user.Username, verifyToken); err != nil {
					logger.WithError(err).Error("Failed to send verification email")
				}
			}()
		}

		// Audit log
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"user_id":    user.ID,
			"email":      user.Email,
			"ip":         c.ClientIP(),
		}).Info("User registered successfully")

		c.JSON(http.StatusCreated, RegisterResponse{
			User:              user.ToResponse(),
			AccessToken:       accessToken,
			RefreshToken:      refreshToken,
			CSRFToken:         csrfToken,
			VerificationToken: verificationToken,
		})
	}
}
