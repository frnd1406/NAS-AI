package testutils

import (
	"github.com/gin-gonic/gin"
	authhandlers "github.com/nas-ai/api/src/handlers/auth"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/nas-ai/api/src/services/operations"
)

// ============================================================
// HELPERS
// ============================================================

// MakeRealUserRepository creates a real UserRepository connected to TestEnv DB
// Uses sqlx-based repository for compatibility with DBX
func MakeRealUserRepository(env *TestEnv) auth_repo.UserRepositoryInterface {
	return auth_repo.NewUserRepositoryX(env.DB, env.SlogLogger)
}

// MakeTestEmailService creates a real EmailService configured for testing (won't send)
func MakeTestEmailService(env *TestEnv) *operations.EmailService {
	return operations.NewEmailService(env.Config, env.Logger)
}

// ============================================================
// REAL HANDLER WRAPPERS (Deprecated usage - prefer SetupTestRouter)
// These import and call the REAL production handlers
// ============================================================

func TestableRegisterHandler(env *TestEnv) gin.HandlerFunc {
	// Create REAL repository (uses in-memory SQLite for tests)
	userRepo := MakeRealUserRepository(env)
	emailService := MakeTestEmailService(env)

	// Call the REAL handler constructor using TestEnv's real services
	return authhandlers.RegisterHandler(
		env.Config,
		userRepo,
		env.JWTService,
		env.PasswordService,
		env.TokenService,
		emailService,
		env.RedisClient,
		env.Logger,
	)
}

func TestableLoginHandler(env *TestEnv) gin.HandlerFunc {
	userRepo := MakeRealUserRepository(env)

	return authhandlers.LoginHandler(
		userRepo,
		env.JWTService,
		env.PasswordService,
		nil, // WebAuthn disabled in tests
		nil, // no credential repo in tests
		env.RedisClient,
		env.Logger,
	)
}

func TestableRefreshHandler(env *TestEnv) gin.HandlerFunc {
	return authhandlers.RefreshHandler(
		env.JWTService,
		env.RedisClient,
		env.Logger,
	)
}

func TestableLogoutHandler(env *TestEnv) gin.HandlerFunc {
	return authhandlers.LogoutHandler(
		env.JWTService,
		env.RedisClient,
		env.Logger,
	)
}
