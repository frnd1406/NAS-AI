package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/middleware/logic"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/nas-ai/api/src/services/operations"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// Handler holds dependencies for auth handlers
type Handler struct {
	cfg             *config.Config
	userRepo        *auth_repo.UserRepository
	jwtService      *security.JWTService
	passwordService *security.PasswordService
	tokenService    *security.TokenService
	emailService    *operations.EmailService
	webAuthnService *security.WebAuthnService
	credentialRepo  *auth_repo.WebAuthnCredentialRepository
	recoveryService *security.RecoveryCodeService
	redis           *database.RedisClient
	logger          *logrus.Logger
}

// NewHandler creates a new Auth Handler
func NewHandler(
	cfg *config.Config,
	userRepo *auth_repo.UserRepository,
	jwtService *security.JWTService,
	passwordService *security.PasswordService,
	tokenService *security.TokenService,
	emailService *operations.EmailService,
	webAuthnService *security.WebAuthnService,
	credentialRepo *auth_repo.WebAuthnCredentialRepository,
	recoveryService *security.RecoveryCodeService,
	redis *database.RedisClient,
	logger *logrus.Logger,
) *Handler {
	return &Handler{
		cfg:             cfg,
		userRepo:        userRepo,
		jwtService:      jwtService,
		passwordService: passwordService,
		tokenService:    tokenService,
		emailService:    emailService,
		webAuthnService: webAuthnService,
		credentialRepo:  credentialRepo,
		recoveryService: recoveryService,
		redis:           redis,
		logger:          logger,
	}
}

// RegisterGlobalRoutes registers public auth routes (usually under /auth)
func (h *Handler) RegisterGlobalRoutes(rg *gin.RouterGroup) {
	// Separate buckets so dashboard refresh storms do not lock out login.
	// login: moderate anti-bruteforce; refresh: higher (parallel clients); strict: register/reset.
	loginLimiter := logic.NewRateLimiterWithBurst(20, 8)
	refreshLimiter := logic.NewRateLimiterWithBurst(120, 40)
	strictAuthLimiter := logic.NewRateLimiterWithBurst(10, 5)

	rg.POST("/register",
		strictAuthLimiter.Middleware(),
		RegisterHandler(h.cfg, h.userRepo, h.jwtService, h.passwordService, h.tokenService, h.emailService, h.redis, h.logger),
	)
	rg.POST("/login",
		loginLimiter.Middleware(),
		LoginHandler(h.userRepo, h.jwtService, h.passwordService, h.webAuthnService, h.credentialRepo, h.redis, h.logger),
	)

	// WebAuthn second-factor login (public, rate-limited). Reached only after
	// a successful password step that returned an mfa_token.
	rg.POST("/login/webauthn/begin",
		loginLimiter.Middleware(),
		WebAuthnLoginBeginHandler(h.webAuthnService, h.userRepo, h.logger),
	)
	rg.POST("/login/webauthn/finish",
		loginLimiter.Middleware(),
		WebAuthnLoginFinishHandler(h.webAuthnService, h.userRepo, h.jwtService, h.redis, h.logger),
	)
	// Recovery-code second factor (public, rate-limited): fallback for a lost
	// authenticator. Also reached only after a password step returned an mfa_token.
	rg.POST("/login/recovery",
		loginLimiter.Middleware(),
		WebAuthnLoginRecoveryHandler(h.recoveryService, h.webAuthnService, h.userRepo, h.jwtService, h.redis, h.logger),
	)

	// WebAuthn credential management (auth-guarded + CSRF): enroll and manage
	// keys. CSRFMiddleware skips GET, so listing stays simple.
	authGuard := logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger)
	csrfGuard := logic.CSRFMiddleware(h.redis, h.logger)
	rg.POST("/webauthn/register/begin", authGuard, csrfGuard,
		WebAuthnRegisterBeginHandler(h.webAuthnService, h.userRepo, h.logger),
	)
	rg.POST("/webauthn/register/finish", authGuard, csrfGuard,
		WebAuthnRegisterFinishHandler(h.webAuthnService, h.userRepo, h.logger),
	)
	rg.GET("/webauthn/credentials", authGuard,
		WebAuthnCredentialsListHandler(h.credentialRepo, h.logger),
	)
	rg.DELETE("/webauthn/credentials/:id", authGuard, csrfGuard,
		WebAuthnCredentialDeleteHandler(h.credentialRepo, h.logger),
	)

	// Recovery-code management (auth-guarded). POST (re)generates the set and
	// returns the plaintext codes once; GET reports how many remain. CSRF guards
	// the state-changing POST; GET stays simple.
	rg.POST("/webauthn/recovery-codes", authGuard, csrfGuard,
		GenerateRecoveryCodesHandler(h.recoveryService, h.logger),
	)
	rg.GET("/webauthn/recovery-codes", authGuard,
		RecoveryCodesStatusHandler(h.recoveryService, h.logger),
	)
	// Refresh is rate-limited separately so parallel dashboard probes cannot
	// exhaust the login bucket.
	rg.POST("/refresh",
		refreshLimiter.Middleware(),
		RefreshHandler(h.jwtService, h.redis, h.logger),
	)
	rg.POST("/logout",
		logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
		LogoutHandler(h.jwtService, h.redis, h.logger),
	)

	// Email verification (rate-limited: token guessing)
	rg.POST("/verify-email",
		strictAuthLimiter.Middleware(),
		VerifyEmailHandler(h.userRepo, h.tokenService, h.emailService, h.logger),
	)
	rg.POST("/resend-verification",
		logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
		ResendVerificationHandler(h.userRepo, h.tokenService, h.emailService, h.logger),
	)

	// Password reset (rate-limited: enumeration / reset-token guessing)
	rg.POST("/forgot-password",
		strictAuthLimiter.Middleware(),
		ForgotPasswordHandler(h.userRepo, h.tokenService, h.emailService, h.logger),
	)
	rg.POST("/reset-password",
		strictAuthLimiter.Middleware(),
		ResetPasswordHandler(h.userRepo, h.tokenService, h.passwordService, h.jwtService, h.redis, h.logger),
	)
}

// RegisterV1Routes registers API v1 auth routes (usually under /api/v1/auth)
func (h *Handler) RegisterV1Routes(rg *gin.RouterGroup) {
	rg.GET("/csrf", GetCSRFToken(h.redis, h.logger))
}
