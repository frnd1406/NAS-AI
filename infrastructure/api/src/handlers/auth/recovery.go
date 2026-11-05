package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/nas-ai/api/src/services/security"

	"github.com/sirupsen/logrus"
)

// recoveryLoginRequest carries the one-time MFA pending token plus a recovery
// code, used to complete the second factor when the authenticator is lost.
type recoveryLoginRequest struct {
	MFAToken string `json:"mfa_token" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// WebAuthnLoginRecoveryHandler completes the second factor with a one-time
// recovery code instead of the authenticator (public, rate-limited). It is
// reached only after a successful password step that returned an mfa_token.
// The token is consumed up front, bounding code-guessing to one attempt per
// password step; on success the shared issueSession path issues the session.
func WebAuthnLoginRecoveryHandler(
	recoveryService *security.RecoveryCodeService,
	webAuthnService *security.WebAuthnService,
	userRepo auth_repo.UserRepositoryInterface,
	jwtService security.JWTServiceInterface,
	redis *database.RedisClient,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if webAuthnService == nil || recoveryService == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "webauthn not enabled"})
			return
		}

		var req recoveryLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// Consume the pending token (single-use) and resolve the user.
		userID, err := webAuthnService.ValidateMFAPendingToken(c.Request.Context(), req.MFAToken, true)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired mfa token"})
			return
		}
		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		ok, err := recoveryService.Verify(c.Request.Context(), userID, req.Code)
		if err != nil {
			logger.WithError(err).Error("recovery: verify failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		if !ok {
			logger.WithFields(logrus.Fields{
				"user_id": userID,
				"ip":      c.ClientIP(),
			}).Warn("recovery: invalid or already-used code")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
			return
		}

		_ = issueSession(c, user, jwtService, redis, logger)
	}
}

// GenerateRecoveryCodesHandler creates a fresh set of recovery codes for the
// logged-in user (auth-guarded + CSRF), invalidating any previous set. The
// plaintext codes are returned once and are never stored in the clear.
func GenerateRecoveryCodesHandler(
	recoveryService *security.RecoveryCodeService,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if recoveryService == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "recovery codes not enabled"})
			return
		}
		userID := c.GetString("user_id")
		codes, err := recoveryService.GenerateForUser(c.Request.Context(), userID)
		if err != nil {
			logger.WithError(err).Error("recovery: generate failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate recovery codes"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"codes": codes, "count": len(codes)})
	}
}

// RecoveryCodesStatusHandler reports how many unused recovery codes remain for
// the logged-in user (auth-guarded).
func RecoveryCodesStatusHandler(
	recoveryService *security.RecoveryCodeService,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if recoveryService == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "recovery codes not enabled"})
			return
		}
		userID := c.GetString("user_id")
		remaining, err := recoveryService.RemainingCount(c.Request.Context(), userID)
		if err != nil {
			logger.WithError(err).Error("recovery: status failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"remaining": remaining})
	}
}
