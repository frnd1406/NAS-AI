package testutils

import (
	"github.com/gin-gonic/gin"
	authhandlers "github.com/nas-ai/api/src/handlers/auth"
	filehandlers "github.com/nas-ai/api/src/handlers/files"
	"github.com/nas-ai/api/src/middleware/logic"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/nas-ai/api/src/services/content"
)

// SetupTestRouter creates a test router using:
// - REAL security services from TestEnv (JWT, Token, Password, Encryption, Honeyfile)
// - MOCK data services from TestEnv (Storage, AI)
// - Optional overrides for specific mocks if provided
func SetupTestRouter(env *TestEnv) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Request ID middleware
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})

	// Real User Repository from test DB
	userRepo := MakeRealUserRepository(env)

	// Email Service (real config, fake delivery in tests)
	emailService := MakeTestEmailService(env)

	// === AUTH ROUTES (all use REAL services) ===
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authhandlers.RegisterHandler(
			env.Config,
			userRepo,
			env.JWTService,
			env.PasswordService,
			env.TokenService,
			emailService,
			env.RedisClient,
			env.Logger,
		))

		authGroup.POST("/login", authhandlers.LoginHandler(
			userRepo,
			env.JWTService,
			env.PasswordService,
			nil, // WebAuthn disabled in tests
			nil, // no credential repo in tests
			env.RedisClient,
			env.Logger,
		))

		authGroup.POST("/refresh", authhandlers.RefreshHandler(
			env.JWTService,
			env.RedisClient,
			env.Logger,
		))

		authGroup.POST("/logout", authhandlers.LogoutHandler(
			env.JWTService,
			env.RedisClient,
			env.Logger,
		))
	}

	// === PROTECTED ROUTES ===
	protectedGroup := router.Group("/api/v1")
	protectedGroup.Use(logic.AuthMiddleware(env.JWTService, env.TokenService, env.RedisClient, env.Logger))
	{
		// Test endpoint
		protectedGroup.GET("/me", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			userEmail, _ := c.Get("user_email")
			c.JSON(200, gin.H{
				"message":    "protected_resource",
				"user_id":    userID,
				"user_email": userEmail,
			})
		})

		// === FILE ROUTES (mock storage, real security) ===
		storageGroup := protectedGroup.Group("/storage")
		{
			storageGroup.POST("/upload", filehandlers.StorageUploadHandler(
				env.StorageService, // MOCK
				env.PolicyService,  // REAL
				env.HoneyfileSvc,   // REAL
				env.AIService,      // MOCK
				env.Logger,
			))
			storageGroup.GET("/list", filehandlers.StorageListHandler(env.StorageService, env.Logger))
			storageGroup.GET("/download", filehandlers.StorageDownloadHandler(
				env.StorageService, // MOCK
				env.HoneyfileSvc,   // REAL
				env.Logger,
			))
		}

		filesGroup := protectedGroup.Group("/files")
		{
			filesGroup.GET("/content", filehandlers.FileContentHandler(env.StorageService, env.Logger))
		}

		// === ENCRYPTED ROUTES (all REAL encryption) ===
		// Note: Requires an EncryptedStorageService implementation, which depends on real encryption
		// For now, skip encrypted routes in basic test setup
	}

	return router
}

// SetupTestRouterWithMocks allows overriding specific services for unit tests.
// Use this when you need to mock specific behavior.
func SetupTestRouterWithMocks(
	env *TestEnv,
	userRepo auth_repo.UserRepositoryInterface,
	storageService content.StorageService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})

	emailService := MakeTestEmailService(env)

	// Auth routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authhandlers.RegisterHandler(
			env.Config,
			userRepo,
			env.JWTService,
			env.PasswordService,
			env.TokenService,
			emailService,
			env.RedisClient,
			env.Logger,
		))

		authGroup.POST("/login", authhandlers.LoginHandler(
			userRepo,
			env.JWTService,
			env.PasswordService,
			nil, // WebAuthn disabled in tests
			nil, // no credential repo in tests
			env.RedisClient,
			env.Logger,
		))
	}

	// Protected routes with mock storage
	protectedGroup := router.Group("/api/v1")
	protectedGroup.Use(logic.AuthMiddleware(env.JWTService, env.TokenService, env.RedisClient, env.Logger))
	{
		if storageService != nil {
			storageGroup := protectedGroup.Group("/storage")
			{
				storageGroup.POST("/upload", filehandlers.StorageUploadHandler(
					storageService,
					env.PolicyService,
					env.HoneyfileSvc,
					env.AIService,
					env.Logger,
				))
				storageGroup.GET("/list", filehandlers.StorageListHandler(storageService, env.Logger))
				storageGroup.GET("/download", filehandlers.StorageDownloadHandler(
					storageService,
					env.HoneyfileSvc,
					env.Logger,
				))
			}

			filesGroup := protectedGroup.Group("/files")
			{
				filesGroup.GET("/content", filehandlers.FileContentHandler(storageService, env.Logger))
			}
		}
	}

	return router
}
