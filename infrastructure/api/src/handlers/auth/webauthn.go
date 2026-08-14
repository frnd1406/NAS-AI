package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/nas-ai/api/src/services/security"

	"github.com/sirupsen/logrus"
)

// registerFinishRequest carries the friendly name plus the raw attestation.
// The attestation JSON is re-read from the request body by the WebAuthn parser,
// so only the name is bound here (from a query/form value).

// WebAuthnRegisterBeginHandler starts enrolling a new authenticator for the
// currently logged-in user (auth-guarded).
func WebAuthnRegisterBeginHandler(
	webAuthnService *security.WebAuthnService,
	userRepo auth_repo.UserRepositoryInterface,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if webAuthnService == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "webauthn not enabled"})
			return
		}
		userID := c.GetString("user_id")
		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		options, err := webAuthnService.BeginRegistration(c.Request.Context(), user)
		if err != nil {
			logger.WithError(err).Error("webauthn: begin registration failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin registration"})
			return
		}
		c.JSON(http.StatusOK, options)
	}
}

// WebAuthnRegisterFinishHandler verifies the attestation and stores the
// credential (auth-guarded). The optional ?name= query names the key.
func WebAuthnRegisterFinishHandler(
	webAuthnService *security.WebAuthnService,
	userRepo auth_repo.UserRepositoryInterface,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if webAuthnService == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "webauthn not enabled"})
			return
		}
		userID := c.GetString("user_id")
		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		name := c.Query("name")
		if name == "" {
			name = "Security Key"
		}

		if err := webAuthnService.FinishRegistration(c.Request.Context(), user, name, c.Request.Body); err != nil {
			logger.WithError(err).Warn("webauthn: finish registration failed")
			c.JSON(http.StatusBadRequest, gin.H{"error": "registration failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// mfaTokenRequest is the JSON body carrying the one-time MFA pending token.
type mfaTokenRequest struct {
	MFAToken string `json:"mfa_token" binding:"required"`
}

// WebAuthnLoginBeginHandler starts the assertion ceremony after the password
// step. It expects the mfa_token issued by LoginHandler (public, rate-limited).
func WebAuthnLoginBeginHandler(
	webAuthnService *security.WebAuthnService,
	userRepo auth_repo.UserRepositoryInterface,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if webAuthnService == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "webauthn not enabled"})
			return
		}
		var req mfaTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// Peek (do not consume) the token to resolve the user.
		userID, err := webAuthnService.ValidateMFAPendingToken(c.Request.Context(), req.MFAToken, false)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired mfa token"})
			return
		}
		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		options, err := webAuthnService.BeginLogin(c.Request.Context(), req.MFAToken, user)
		if err != nil {
			logger.WithError(err).Error("webauthn: begin login failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin login"})
			return
		}
		c.JSON(http.StatusOK, options)
	}
}

// WebAuthnLoginFinishHandler verifies the assertion and, on success, issues the
// real session — reusing the shared issueSession path (public, rate-limited).
// The mfa_token is passed via the X-MFA-Token header so the request body can
// carry the raw WebAuthn assertion JSON for the parser.
func WebAuthnLoginFinishHandler(
	webAuthnService *security.WebAuthnService,
	userRepo auth_repo.UserRepositoryInterface,
	jwtService security.JWTServiceInterface,
	redis *database.RedisClient,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if webAuthnService == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "webauthn not enabled"})
			return
		}
		mfaToken := c.GetHeader("X-MFA-Token")
		if mfaToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing mfa token"})
			return
		}

		// Consume the token (single-use) and resolve the user.
		userID, err := webAuthnService.ValidateMFAPendingToken(c.Request.Context(), mfaToken, true)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired mfa token"})
			return
		}
		user, err := userRepo.FindByID(c.Request.Context(), userID)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if err := webAuthnService.FinishLogin(c.Request.Context(), mfaToken, user, c.Request.Body); err != nil {
			logger.WithError(err).Warn("webauthn: finish login failed")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
			return
		}

		_ = issueSession(c, user, jwtService, redis, logger)
	}
}

// WebAuthnCredentialsListHandler lists the user's registered authenticators
// (auth-guarded).
func WebAuthnCredentialsListHandler(
	credentialRepo *auth_repo.WebAuthnCredentialRepository,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		records, err := credentialRepo.ListByUserID(c.Request.Context(), userID)
		if err != nil {
			logger.WithError(err).Error("webauthn: list credentials failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list credentials"})
			return
		}

		type credentialView struct {
			ID         string  `json:"id"`
			Name       string  `json:"name"`
			CreatedAt  string  `json:"created_at"`
			LastUsedAt *string `json:"last_used_at"`
		}
		out := make([]credentialView, 0, len(records))
		for _, r := range records {
			view := credentialView{ID: r.ID, Name: r.Name, CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
			if r.LastUsedAt.Valid {
				s := r.LastUsedAt.Time.Format("2006-01-02T15:04:05Z07:00")
				view.LastUsedAt = &s
			}
			out = append(out, view)
		}
		c.JSON(http.StatusOK, gin.H{"credentials": out})
	}
}

// WebAuthnCredentialDeleteHandler removes one of the user's authenticators
// (auth-guarded).
func WebAuthnCredentialDeleteHandler(
	credentialRepo *auth_repo.WebAuthnCredentialRepository,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		id := c.Param("id")
		if err := credentialRepo.Delete(c.Request.Context(), userID, id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
