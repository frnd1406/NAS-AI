package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	auth_repo "github.com/nas-ai/api/src/repository/auth"
	files_repo "github.com/nas-ai/api/src/repository/files"
	settings_repo "github.com/nas-ai/api/src/repository/settings"
	system_repo "github.com/nas-ai/api/src/repository/system"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/handlers/ai"
	"github.com/nas-ai/api/src/handlers/auth"
	"github.com/nas-ai/api/src/handlers/files"
	"github.com/nas-ai/api/src/handlers/system"
	"github.com/nas-ai/api/src/middleware/core"
	"github.com/nas-ai/api/src/middleware/logic"

	"github.com/nas-ai/api/src/scheduler"
	"github.com/nas-ai/api/src/services/common"
	servicesConfig "github.com/nas-ai/api/src/services/config"
	"github.com/nas-ai/api/src/services/content"
	"github.com/nas-ai/api/src/services/intelligence"
	"github.com/nas-ai/api/src/services/operations"
	"github.com/nas-ai/api/src/services/security"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// Server holds all dependencies for the API server (Clean Architecture)
type Server struct {
	cfg    *config.Config
	logger *logrus.Logger
	router *gin.Engine
	db     *database.DB
	redis  *database.RedisClient
	dbx    *sqlx.DB

	// Repositories
	userRepo          *auth_repo.UserRepository
	settingsRepo      *settings_repo.SystemSettingsRepository
	systemMetricsRepo *system_repo.SystemMetricsRepository
	systemAlertsRepo  *system_repo.SystemAlertsRepository
	monitoringRepo    *system_repo.MonitoringRepository
	embeddingsRepo    *files_repo.FileEmbeddingsRepository
	honeyfileRepo     *files_repo.HoneyfileRepository
	fileRepo          *files_repo.FileRepository
	webauthnCredRepo  *auth_repo.WebAuthnCredentialRepository
	recoveryCodeRepo  *auth_repo.RecoveryCodeRepository

	// Services
	jwtService              *security.JWTService
	passwordService         *security.PasswordService
	tokenService            *security.TokenService
	webAuthnService         *security.WebAuthnService
	recoveryCodeService     *security.RecoveryCodeService
	emailService            *operations.EmailService
	backupService           *operations.BackupService
	settingsService         *servicesConfig.SettingsService
	encryptionService       *security.EncryptionService
	storageService          *content.StorageManager
	encryptedStorageService *content.EncryptedStorageService
	secureAIFeeder          *intelligence.SecureAIFeeder
	aiHTTPClient            *http.Client
	jobService              *operations.JobService
	honeyfileService        *content.HoneyfileService
	encryptionPolicyService *security.EncryptionPolicyService
	archiveService          *content.ArchiveService
	contentDeliveryService  *content.ContentDeliveryService
	aiAgentService          *intelligence.AIAgentService
	alertService            *operations.AlertService
	benchmarkService        *operations.BenchmarkService
	consistencyService      *operations.ConsistencyService
	diagnosticsService      *operations.DiagnosticsService
	hardwareService         *operations.HardwareService

	// Handlers
	blobStorageHandler *files.BlobStorageHandler
	diagnosticsHandler *system.DiagnosticsHandler
	hardwareHandler    *system.HardwareHandler

	// Route Handlers
	authHandler   *auth.Handler
	filesHandler  *files.Handler
	systemHandler *system.Handler
	aiHandler     *ai.Handler
}

// NewServer creates and initializes all server dependencies
func NewServer(cfg *config.Config, logger *logrus.Logger) (*Server, error) {
	s := &Server{
		cfg:    cfg,
		logger: logger,
	}

	if err := s.initDatabase(); err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	if err := s.initRepositories(); err != nil {
		return nil, fmt.Errorf("repository init failed: %w", err)
	}

	if err := s.initServices(); err != nil {
		return nil, fmt.Errorf("service init failed: %w", err)
	}

	s.initHandlers()
	s.initRouter()
	s.SetupRoutes()
	s.startBackgroundWorkers()

	return s, nil
}

// initDatabase establishes database connections
func (s *Server) initDatabase() error {
	var err error

	s.db, err = database.NewPostgresConnection(s.cfg, s.logger)
	if err != nil {
		return fmt.Errorf("postgres connection failed: %w", err)
	}

	s.redis, err = database.NewRedisConnection(s.cfg, s.logger)
	if err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}

	// Health check
	if err := s.checkDependencies(); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	s.dbx = sqlx.NewDb(s.db.DB, "postgres")
	return nil
}

// initRepositories initializes all data access layers
func (s *Server) initRepositories() error {
	var err error

	s.settingsRepo = settings_repo.NewSystemSettingsRepository(s.dbx, s.logger)
	if err = s.settingsRepo.EnsureTable(context.Background()); err != nil {
		return fmt.Errorf("settings table init failed: %w", err)
	}
	s.applyPersistedBackupSettings()

	s.userRepo = auth_repo.NewUserRepository(s.db, s.logger)
	s.systemMetricsRepo = system_repo.NewSystemMetricsRepository(s.dbx, s.logger)
	s.systemAlertsRepo = system_repo.NewSystemAlertsRepository(s.dbx, s.logger)
	s.monitoringRepo = system_repo.NewMonitoringRepository(s.db, s.logger)
	s.embeddingsRepo = files_repo.NewFileEmbeddingsRepository(s.dbx, s.logger)
	s.honeyfileRepo = files_repo.NewHoneyfileRepository(s.dbx, s.logger)

	if err = s.honeyfileRepo.EnsureTable(context.Background()); err != nil {
		return fmt.Errorf("honeyfiles table init failed: %w", err)
	}

	s.fileRepo = files_repo.NewFileRepository(s.dbx, s.logger)

	s.webauthnCredRepo = auth_repo.NewWebAuthnCredentialRepository(s.dbx, s.logger)
	if err = s.webauthnCredRepo.EnsureTable(context.Background()); err != nil {
		return fmt.Errorf("webauthn_credentials table init failed: %w", err)
	}

	s.recoveryCodeRepo = auth_repo.NewRecoveryCodeRepository(s.dbx, s.logger)
	if err = s.recoveryCodeRepo.EnsureTable(context.Background()); err != nil {
		return fmt.Errorf("recovery_codes table init failed: %w", err)
	}

	return nil
}

// initServices initializes all business logic services
func (s *Server) initServices() error {
	var err error

	// Auth Services
	s.jwtService, err = security.NewJWTService(s.cfg, s.logger)
	if err != nil {
		return fmt.Errorf("JWT service init failed: %w", err)
	}
	s.passwordService = security.NewPasswordService()
	s.tokenService = security.NewTokenService(s.redis, s.logger)
	s.emailService = operations.NewEmailService(s.cfg, s.logger)

	// WebAuthn (YubiKey second factor). Nil when RP ID is unresolved => disabled.
	s.webAuthnService, err = security.NewWebAuthnService(
		s.cfg.WebAuthnRPID,
		s.cfg.WebAuthnRPDisplayName,
		s.cfg.WebAuthnRPOrigins,
		s.redis,
		s.webauthnCredRepo,
		s.logger,
	)
	if err != nil {
		return fmt.Errorf("webauthn service init failed: %w", err)
	}
	if s.webAuthnService != nil {
		s.logger.WithField("rp_id", s.cfg.WebAuthnRPID).Info("WebAuthn (2FA) enabled")
	}

	// Recovery codes (2FA backup codes). Always available so a user who loses
	// their authenticator is not locked out.
	s.recoveryCodeService = security.NewRecoveryCodeService(s.recoveryCodeRepo, s.passwordService, s.logger)

	// Backup Service
	s.backupService, err = operations.NewBackupService(s.cfg.FilesStorageRoot, s.cfg.BackupStoragePath, s.cfg.AllowedStorageRoots, s.logger)
	if err != nil {
		return fmt.Errorf("backup service init failed: %w", err)
	}

	// Settings Service
	onRestartScheduler := func() error {
		return scheduler.RestartScheduler()
	}
	s.settingsService = servicesConfig.NewSettingsService(s.cfg, s.settingsRepo, s.backupService, onRestartScheduler, s.logger)
	s.logger.Info("SettingsService initialized")

	// Encryption Service (Zero-Knowledge)
	vaultPath := "/tmp/nas-vault-demo"
	if s.cfg.Environment == "production" {
		if _, err := os.Stat("/var/lib/nas/vault"); err == nil {
			vaultPath = "/var/lib/nas/vault"
			s.logger.Warn("⚠️  Vault persistence enabled: Keys survive restarts (security trade-off)")
		}
	}
	s.encryptionService = security.NewEncryptionService(vaultPath, s.logger)
	s.logger.WithField("vaultPath", vaultPath).Info("Encryption service initialized")

	// Storage Services
	localStore, err := storage.NewLocalStore(s.cfg.FilesStorageRoot)
	if err != nil {
		return fmt.Errorf("local store init failed: %w", err)
	}
	s.storageService = content.NewStorageManager(localStore, s.encryptionService, s.fileRepo, s.logger)

	// Encrypted Storage (optional) — separate physical root from plain files
	encryptedStoragePath := s.cfg.VaultStorageRoot
	s.encryptedStorageService, err = content.NewEncryptedStorageService(
		s.storageService,
		s.encryptionService,
		encryptedStoragePath,
		s.logger,
	)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to initialize encrypted storage service (non-fatal)")
	} else {
		s.logger.WithField("path", encryptedStoragePath).Info("Encrypted storage service initialized")
	}

	// AI Services
	if s.encryptionService != nil {
		s.secureAIFeeder = intelligence.NewSecureAIFeeder(
			s.encryptionService,
			s.cfg.AIServiceURL,
			s.cfg.InternalAPISecret,
			s.logger,
		)
		s.logger.Info("SecureAIFeeder initialized")
	}

	s.aiHTTPClient = common.NewSecureHTTPClient(s.cfg.InternalAPISecret, 15*time.Second)
	s.jobService = operations.NewJobService(s.redis, s.logger)
	if err := s.jobService.EnsureConsumerGroup(context.Background()); err != nil {
		s.logger.WithError(err).Warn("Failed to ensure AI job consumer group (non-fatal)")
	}
	s.logger.Info("JobService initialized")

	// Security Services
	s.honeyfileService = content.NewHoneyfileService(s.honeyfileRepo, s.encryptionService, s.logger)
	s.logger.Info("HoneyfileService initialized")

	s.encryptionPolicyService = security.NewEncryptionPolicyService()
	s.logger.Info("EncryptionPolicyService initialized")

	// Utility Services
	s.archiveService = content.NewArchiveService(s.logger)
	s.logger.Info("ArchiveService initialized")

	s.contentDeliveryService = content.NewContentDeliveryService(s.storageService, s.encryptionService, s.logger)
	s.logger.Info("ContentDeliveryService initialized")

	s.aiAgentService = intelligence.NewAIAgentService(s.logger, s.honeyfileService, s.cfg.InternalAPISecret)
	s.logger.Info("AIAgentService initialized")

	s.alertService = operations.NewAlertService(s.emailService, s.cfg, s.logger)
	s.logger.Info("AlertService initialized")

	s.benchmarkService = operations.NewBenchmarkService(s.logger)

	// Consistency Service
	s.consistencyService = operations.NewConsistencyService(
		s.dbx,
		s.embeddingsRepo,
		s.cfg.FilesStorageRoot,
		time.Duration(s.cfg.ConsistencyCheckIntervalMin)*time.Minute,
		s.logger,
	)

	// Diagnostics Service
	s.diagnosticsService = operations.NewDiagnosticsService(s.storageService, s.encryptionService, s.dbx)
	s.logger.Info("DiagnosticsService initialized")

	// Hardware Service
	s.hardwareService = operations.NewHardwareService(s.logger)
	s.logger.Info("HardwareService initialized")

	// Initial reconciliation
	s.logger.Info("Running initial consistency reconciliation...")
	if err := s.consistencyService.RunReconciliation(context.Background()); err != nil {
		s.logger.WithError(err).Warn("Initial reconciliation failed (non-fatal)")
	}

	return nil
}

// initHandlers initializes HTTP handlers
func (s *Server) initHandlers() {
	s.blobStorageHandler = files.NewBlobStorageHandler(s.storageService, s.logger)
	s.logger.Info("BlobStorageHandler initialized")

	s.diagnosticsHandler = system.NewDiagnosticsHandler(s.diagnosticsService)
	s.logger.Info("DiagnosticsHandler initialized")

	s.hardwareHandler = system.NewHardwareHandler(s.hardwareService, s.logger)
	s.logger.Info("HardwareHandler initialized")

	// Initialize new module handlers
	s.authHandler = auth.NewHandler(
		s.cfg,
		s.userRepo,
		s.jwtService,
		s.passwordService,
		s.tokenService,
		s.emailService,
		s.webAuthnService,
		s.webauthnCredRepo,
		s.recoveryCodeService,
		s.redis,
		s.logger,
	)

	s.filesHandler = files.NewHandler(
		s.storageService,
		s.encryptionService,
		s.encryptionPolicyService,
		s.honeyfileService,
		s.aiAgentService,
		s.archiveService,
		s.contentDeliveryService,
		s.encryptedStorageService,
		s.secureAIFeeder,
		s.blobStorageHandler,
		s.jwtService,
		s.tokenService,
		s.redis,
		s.logger,
	)

	s.systemHandler = system.NewHandler(
		s.db,
		s.redis,
		s.cfg,
		s.userRepo,
		s.systemMetricsRepo,
		s.systemAlertsRepo,
		s.monitoringRepo,
		s.jobService,
		s.benchmarkService,
		s.settingsService,
		s.encryptionService,
		s.honeyfileService,
		s.hardwareHandler,
		s.diagnosticsHandler,
		s.jwtService,
		s.tokenService,
		s.logger,
	)

	s.aiHandler = ai.NewHandler(
		s.db,
		s.cfg,
		s.aiHTTPClient,
		s.jobService,
		s.secureAIFeeder,
		s.userRepo,
		s.jwtService,
		s.tokenService,
		s.redis,
		s.logger,
	)
}

// initRouter creates and configures the Gin router
func (s *Server) initRouter() {
	if s.cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.New()

	// Middleware chain (Onion Principle)
	rateLimiter := logic.NewRateLimiter(s.cfg)
	s.router.Use(
		core.PanicRecovery(s.logger),
		core.RequestID(),
		core.GinSecureHeaders(),
		core.CORS(s.cfg, s.logger),
		rateLimiter.Middleware(),
		core.AuditLogger(s.logger),
	)

	// OPTIONS preflight
	s.router.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	// Environment context
	s.router.Use(func(c *gin.Context) {
		c.Set("environment", s.cfg.Environment)
		c.Next()
	})
}

// startBackgroundWorkers starts all background goroutines
func (s *Server) startBackgroundWorkers() {
	// Alert ticker
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.alertService.RunSystemChecks()
		}
	}()

	// Startup benchmark
	go func() {
		if err := s.benchmarkService.RunStartupBenchmark(); err != nil {
			s.logger.WithError(err).Warn("Startup benchmark failed (non-fatal)")
		}
	}()

	// Backup scheduler
	go func() {
		if err := scheduler.StartBackupScheduler(s.backupService, s.cfg); err != nil {
			s.logger.WithError(err).Error("Failed to start backup scheduler")
		}
	}()

	// Consistency worker
	go s.consistencyService.Start(context.Background())
}

// Run starts the HTTPS server and waits for shutdown signal.
// The server is TLS-only: plaintext HTTP is never served, so an attacker
// sniffing the wire (e.g. Wireshark) only sees encrypted traffic.
func (s *Server) Run() error {
	secureHandler := core.SecureHeaders(s.router)

	bindAddr := s.cfg.BindAddr
	if bindAddr == "" {
		bindAddr = "0.0.0.0"
	}

	srv := &http.Server{
		Addr:           bindAddr + ":" + s.cfg.Port,
		Handler:        secureHandler,
		ReadTimeout:    600 * time.Second,
		WriteTimeout:   600 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      newBaseTLSConfig(),
	}

	certFile, keyFile, err := s.resolveTLSMaterial(srv)
	if err != nil {
		return err
	}

	// Announce over mDNS so the desktop client can discover this server on the
	// LAN. Best-effort: never blocks startup. Unregistered on shutdown.
	if mdnsStop, mErr := startMDNSAdvertise(s.cfg.Port, s.logger); mErr != nil {
		s.logger.WithError(mErr).Warn("mDNS advertising unavailable")
	} else {
		defer mdnsStop()
	}

	// Start server
	go func() {
		s.logger.WithFields(logrus.Fields{
			"addr":   srv.Addr,
			"scheme": "https",
		}).Info("Server listening (TLS)")
		// When an ephemeral in-memory cert was injected into TLSConfig,
		// certFile/keyFile are empty and ListenAndServeTLS uses that cert.
		if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")
	s.consistencyService.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.WithError(err).Error("Server forced to shutdown")
		return err
	}

	s.logger.Info("Server exited")
	return nil
}

// Close cleans up all resources
func (s *Server) Close() {
	if s.db != nil {
		s.db.Close()
	}
	if s.redis != nil {
		s.redis.Close()
	}
}

// checkDependencies verifies database connectivity
func (s *Server) checkDependencies() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres unreachable: %w", err)
	}

	if err := s.redis.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis unreachable: %w", err)
	}

	return nil
}

// applyPersistedBackupSettings loads settings from database
func (s *Server) applyPersistedBackupSettings() {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	settings, err := s.settingsRepo.GetAll(context.Background())
	if err != nil {
		s.logger.WithError(err).Warn("Failed to load persisted settings; continuing with defaults")
		return
	}

	if schedule, ok := settings[settings_repo.SystemSettingBackupSchedule]; ok {
		sc := strings.TrimSpace(schedule)
		if sc != "" {
			if _, err := parser.Parse(sc); err != nil {
				s.logger.WithError(err).Warn("Ignoring invalid persisted backup schedule")
			} else {
				s.cfg.BackupSchedule = sc
			}
		}
	}

	if retentionStr, ok := settings[settings_repo.SystemSettingBackupRetention]; ok {
		if n, err := strconv.Atoi(retentionStr); err == nil && n > 0 {
			s.cfg.BackupRetentionCount = n
		} else if err != nil {
			s.logger.WithError(err).Warn("Ignoring invalid persisted backup retention")
		}
	}

	if path, ok := settings[settings_repo.SystemSettingBackupPath]; ok {
		p := filepath.Clean(strings.TrimSpace(path))
		if p != "" && p != "." && p != string(os.PathSeparator) {
			s.cfg.BackupStoragePath = p
		} else {
			s.logger.Warn("Ignoring invalid persisted backup path")
		}
	}
}
