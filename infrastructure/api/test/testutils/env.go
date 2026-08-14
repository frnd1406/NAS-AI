package testutils

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/database"
	files_repo "github.com/nas-ai/api/src/repository/files"
	"github.com/nas-ai/api/src/services/content"
	"github.com/nas-ai/api/src/services/security"
	"github.com/nas-ai/api/test/testutils/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestEnv holds all dependencies needed for integration tests.
// Real security services + Mock data services (SRP-compliant).
type TestEnv struct {
	// REAL Security Services
	JWTService      security.JWTServiceInterface
	TokenService    security.TokenServiceInterface
	PasswordService security.PasswordServiceInterface
	EncryptionSvc   security.EncryptionServiceInterface
	PolicyService   security.EncryptionPolicyServiceInterface
	HoneyfileSvc    content.HoneyfileServiceInterface
	HoneyfileRepo   files_repo.HoneyfileRepositoryInterface

	// MOCK Data Services (only file I/O and AI)
	StorageService *mocks.MockStorageService
	AIService      *mocks.MockAIAgentService

	// Infrastructure (real with fake backends)
	DB          *database.DBX
	RedisClient *database.RedisClient
	Config      *config.Config
	Logger      *logrus.Logger
	SlogLogger  *slog.Logger

	// Internal: for cleanup
	miniredis *miniredis.Miniredis
	tempDir   string
}

// NewTestEnv creates a fully initialized TestEnv with REAL security services.
func NewTestEnv(t *testing.T) *TestEnv {
	// Setup miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	// Redis client pointing to miniredis
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	redisClient := &database.RedisClient{Client: rdb}

	// Logger (logrus for handlers, slog for database)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Config with test defaults
	cfg := &config.Config{
		JWTSecret:    "test-secret-at-least-32-characters-long",
		ResendAPIKey: "re_test_123",
		EmailFrom:    "test@nas.ai",
		FrontendURL:  "http://localhost:3000",
		InviteCode:   "TEST_INVITE",
	}

	// Create in-memory SQLite database for testing (using sqlx)
	testDB, err := database.NewTestDatabase(slogLogger)
	require.NoError(t, err)
	t.Cleanup(func() { testDB.Close() })

	// Temp directory for encryption vault
	tempDir, err := os.MkdirTemp("", "test-vault-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	vaultPath := filepath.Join(tempDir, "vault.enc")

	// === REAL SECURITY SERVICES ===

	// JWT Service
	jwtSvc, err := security.NewJWTService(cfg, logger)
	require.NoError(t, err)

	// Token Service (revocation)
	tokenSvc := security.NewTokenService(redisClient, logger)

	// Password Service
	passwordSvc := security.NewPasswordService()

	// Encryption Service
	encryptionSvc := security.NewEncryptionService(vaultPath, logger)

	// Encryption Policy Service
	policySvc := security.NewEncryptionPolicyService()

	// Honeyfile Repository (uses test DB with sqlx)
	honeyRepo := files_repo.NewHoneyfileRepository(testDB.DB, logger)

	// Honeyfile Service
	honeyfileSvc := content.NewHoneyfileService(honeyRepo, encryptionSvc, logger)

	// === MOCK DATA SERVICES (only file I/O and AI) ===

	return &TestEnv{
		// Real Security
		JWTService:      jwtSvc,
		TokenService:    tokenSvc,
		PasswordService: passwordSvc,
		EncryptionSvc:   encryptionSvc,
		PolicyService:   policySvc,
		HoneyfileSvc:    honeyfileSvc,
		HoneyfileRepo:   honeyRepo,

		// Mock Data
		StorageService: new(mocks.MockStorageService),
		AIService:      new(mocks.MockAIAgentService),

		// Infrastructure
		DB:          testDB,
		RedisClient: redisClient,
		Config:      cfg,
		Logger:      logger,
		SlogLogger:  slogLogger,
		miniredis:   mr,
		tempDir:     tempDir,
	}
}

// GenerateTestToken creates a real JWT token for testing authenticated endpoints.
func (e *TestEnv) GenerateTestToken(userID, email string) (string, error) {
	return e.JWTService.GenerateAccessToken(userID, email)
}

// ResetMocks clears mock expectations (for sub-tests).
func (e *TestEnv) ResetMocks() {
	e.StorageService = new(mocks.MockStorageService)
	e.AIService = new(mocks.MockAIAgentService)
}
