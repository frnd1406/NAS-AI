package system

import (
	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/handlers/files"
	"github.com/nas-ai/api/src/handlers/settings"
	"github.com/nas-ai/api/src/middleware/logic"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	system_repo "github.com/nas-ai/api/src/repository/system"
	servicesConfig "github.com/nas-ai/api/src/services/config"
	"github.com/nas-ai/api/src/services/content"
	"github.com/nas-ai/api/src/services/operations"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// Handler holds dependencies for system handlers
type Handler struct {
	db                 *database.DB
	redis              *database.RedisClient
	cfg                *config.Config
	userRepo           *auth_repo.UserRepository
	systemMetricsRepo  *system_repo.SystemMetricsRepository
	systemAlertsRepo   *system_repo.SystemAlertsRepository
	monitoringRepo     *system_repo.MonitoringRepository
	jobService         *operations.JobService
	benchmarkService   *operations.BenchmarkService
	settingsService    *servicesConfig.SettingsService
	encryptionService  *security.EncryptionService
	honeyfileService   *content.HoneyfileService
	hardwareHandler    *HardwareHandler
	diagnosticsHandler *DiagnosticsHandler
	jwtService         *security.JWTService
	tokenService       *security.TokenService
	logger             *logrus.Logger
}

// NewHandler creates a new System Handler
func NewHandler(
	db *database.DB,
	redis *database.RedisClient,
	cfg *config.Config,
	userRepo *auth_repo.UserRepository,
	systemMetricsRepo *system_repo.SystemMetricsRepository,
	systemAlertsRepo *system_repo.SystemAlertsRepository,
	monitoringRepo *system_repo.MonitoringRepository,
	jobService *operations.JobService,
	benchmarkService *operations.BenchmarkService,
	settingsService *servicesConfig.SettingsService,
	encryptionService *security.EncryptionService,
	honeyfileService *content.HoneyfileService,
	hardwareHandler *HardwareHandler,
	diagnosticsHandler *DiagnosticsHandler,
	jwtService *security.JWTService,
	tokenService *security.TokenService,
	logger *logrus.Logger,
) *Handler {
	return &Handler{
		db:                 db,
		redis:              redis,
		cfg:                cfg,
		userRepo:           userRepo,
		systemMetricsRepo:  systemMetricsRepo,
		systemAlertsRepo:   systemAlertsRepo,
		monitoringRepo:     monitoringRepo,
		jobService:         jobService,
		benchmarkService:   benchmarkService,
		settingsService:    settingsService,
		encryptionService:  encryptionService,
		honeyfileService:   honeyfileService,
		hardwareHandler:    hardwareHandler,
		diagnosticsHandler: diagnosticsHandler,
		jwtService:         jwtService,
		tokenService:       tokenService,
		logger:             logger,
	}
}

// RegisterPublicRoutes registers public system routes
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", Health(h.db, h.redis, h.logger))
	rg.POST("/monitoring/ingest", MonitoringIngestHandler(h.monitoringRepo, h.cfg.MonitoringToken, h.logger))
}

// RegisterV1Routes registers API v1 system routes
func (h *Handler) RegisterV1Routes(rg *gin.RouterGroup) {
	// Public (Token based)
	rg.POST("/system/metrics", SystemMetricsHandler(h.systemMetricsRepo, h.cfg.MonitoringToken, h.logger))

	// Public (Hardware Stats - Runtime)
	rg.GET("/system/hardware/storage", h.hardwareHandler.GetStorageInfoHandler())
	rg.GET("/system/hardware/network", h.hardwareHandler.GetNetworkInfoHandler())
	rg.GET("/system/hardware/ups", h.hardwareHandler.GetUPSInfoHandler())

	// Public (Frontend Logging)
	rg.POST("/system/logs/frontend", FrontendLogHandler(h.logger))

	// Public (needed pre-login by setup wizard / capability probe)
	rg.GET("/system/capabilities", Capabilities(h.benchmarkService))
	rg.GET("/system/setup-status", settings.SetupStatusHandler(h.logger))

	// Protected: metrics history/live (leaks internal host IPs), alerts and job
	// status must not be reachable anonymously.
	protected := rg.Group("")
	protected.Use(
		logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
		logic.CSRFMiddleware(h.redis, h.logger),
	)
	{
		protected.GET("/system/metrics", SystemMetricsListHandler(h.systemMetricsRepo, h.logger))
		protected.GET("/system/metrics/live", SystemMetricsLiveHandler(h.logger))
		protected.GET("/system/alerts", SystemAlertsListHandler(h.systemAlertsRepo, h.logger))
		protected.GET("/jobs/:id", GetJobStatusHandler(h.jobService, h.logger)) // Generic job status

		protected.POST("/system/alerts", SystemAlertCreateHandler(h.systemAlertsRepo, h.logger))
		protected.POST("/system/alerts/:id/resolve", SystemAlertResolveHandler(h.systemAlertsRepo, h.logger))
	}

	// Protected System Settings & Vault Management
	settingsV1 := rg.Group("/system")
	settingsV1.Use(
		logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
		logic.CSRFMiddleware(h.redis, h.logger),
	)
	{
		// Detailed health (recon-sensitive) is authenticated; the public
		// /health endpoint only reveals a minimal liveness status.
		settingsV1.GET("/health/detailed", HealthDetailed(h.db, h.redis, h.logger))

		settingsV1.GET("/settings", settings.SystemSettingsHandler(h.settingsService))
		settingsV1.PUT("/settings/backup", settings.UpdateBackupSettingsHandler(h.settingsService))
		settingsV1.POST("/validate-path", settings.ValidatePathHandler(h.settingsService))

		// Vault management
		settingsV1.POST("/vault/setup", files.VaultSetupHandler(h.encryptionService, h.logger))
		settingsV1.POST("/vault/unlock", files.VaultUnlockHandler(h.encryptionService, h.logger))
		settingsV1.POST("/vault/lock", files.VaultLockHandler(h.encryptionService, h.logger))
		settingsV1.POST("/vault/panic", files.VaultPanicHandler(h.encryptionService, h.logger))
		settingsV1.PUT("/vault/config", files.VaultConfigUpdateHandler(h.encryptionService, h.logger))
		settingsV1.GET("/vault/export-config", files.VaultExportConfigHandler(h.encryptionService, h.logger))

		// Setup wizard
		settingsV1.POST("/setup", settings.SetupHandler(h.logger))
	}

	// Integrity Checkpoints (Admin Only)
	sysV1 := rg.Group("/sys")
	sysV1.Use(
		logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
		logic.CSRFMiddleware(h.redis, h.logger),
		logic.AdminOnly(h.userRepo, h.logger),
	)
	{
		sysV1.POST("/integrity/checkpoints", CreateCheckpointHandler(h.honeyfileService, h.logger))
	}
}

// RegisterSettingsRoutes registers additional settings routes (Network, Backup, Security, Storage Settings)
func (h *Handler) RegisterSettingsRoutes(rg *gin.RouterGroup) {
	// SECURITY: these endpoints expose and modify sensitive configuration
	// (IP allow/block lists, backup destinations, storage paths, etc.). They
	// must require authentication + CSRF, not be reachable anonymously.
	sg := rg.Group("")
	sg.Use(
		logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
		logic.CSRFMiddleware(h.redis, h.logger),
	)

	// Network Settings
	sg.GET("/network/settings", settings.NetworkSettingsGetHandler(h.logger))
	sg.PUT("/network/settings", settings.NetworkSettingsSaveHandler(h.logger))

	// Backup Settings
	sg.GET("/backup/settings", settings.BackupSettingsGetHandler(h.logger))
	sg.PUT("/backup/settings", settings.BackupSettingsSaveHandler(h.logger))

	// Security Settings
	sg.GET("/security/settings", settings.SecuritySettingsGetHandler(h.logger))
	sg.PUT("/security/settings", settings.SecuritySettingsSaveHandler(h.logger))

	// Storage Settings
	sg.GET("/storage/settings", settings.StorageSettingsGetHandler(h.logger))
	sg.PUT("/storage/settings", settings.StorageSettingsSaveHandler(h.logger))
}

// RegisterDiagnosticsRoutes registers diagnostics routes
func (h *Handler) RegisterDiagnosticsRoutes(rg *gin.RouterGroup) {
	rg.GET("/system/diagnostics",
		logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
		logic.AdminOnly(h.userRepo, h.logger),
		h.diagnosticsHandler.RunSelfTest,
	)
}
