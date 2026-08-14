package server

import (
	"github.com/nas-ai/api/src/handlers"
	"github.com/nas-ai/api/src/handlers/settings"
	"github.com/nas-ai/api/src/handlers/system"
	"github.com/nas-ai/api/src/middleware/logic"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRoutes configures all HTTP routes (SRP: Routing Only)
func (s *Server) SetupRoutes() {
	// === PUBLIC ROUTES (no auth, but rate-limited) ===
	s.systemHandler.RegisterPublicRoutes(s.router.Group("/"))

	// Swagger documentation (only in development)
	if s.cfg.Environment != "production" {
		s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Register Module Routes
	s.setupAuthRoutes()
	s.setupAPIRoutes()
	s.setupV1Routes()

	// Legacy routes that haven't been moved yet
	s.setupBackupRoutes()
	s.setupAdminRoutes()
	// setupSystemRoutes was moved to SystemHandler.RegisterV1Routes (integrity check)
}

// setupAuthRoutes configures authentication endpoints
func (s *Server) setupAuthRoutes() {
	authGroup := s.router.Group("/auth")
	s.authHandler.RegisterGlobalRoutes(authGroup)
}

// setupAPIRoutes configures protected API endpoints
func (s *Server) setupAPIRoutes() {
	apiGroup := s.router.Group("/api")
	apiGroup.Use(logic.AuthMiddleware(s.jwtService, s.tokenService, s.redis, s.logger))
	apiGroup.Use(logic.CSRFMiddleware(s.redis, s.logger))
	{
		apiGroup.GET("/profile", handlers.ProfileHandler(s.userRepo, s.logger))
		apiGroup.GET("/monitoring", system.MonitoringListHandler(s.monitoringRepo, s.logger))
	}
}

// setupV1Routes configures API v1 endpoints
func (s *Server) setupV1Routes() {
	v1 := s.router.Group("/api/v1")

	// Auth Module
	s.authHandler.RegisterV1Routes(v1)

	// System Module (includes metrics, alerts, capabilities, settings, vault management)
	s.systemHandler.RegisterV1Routes(v1)
	s.systemHandler.RegisterSettingsRoutes(v1)
	s.systemHandler.RegisterDiagnosticsRoutes(v1)

	// AI Module (includes search, query, ask, ai settings)
	s.aiHandler.RegisterV1Routes(v1)
	s.aiHandler.RegisterAdminRoutes(v1) // Reconcile knowledge

	// Files Module (includes files, storage, encrypted, vault uploads)
	s.filesHandler.RegisterV1Routes(v1)
}

// setupBackupRoutes configures backup endpoints
func (s *Server) setupBackupRoutes() {
	backupV1 := s.router.Group("/api/v1/backups")
	backupV1.Use(
		logic.AuthMiddleware(s.jwtService, s.tokenService, s.redis, s.logger),
		logic.CSRFMiddleware(s.redis, s.logger),
	)
	{
		backupV1.GET("", handlers.BackupListHandler(s.backupService, s.logger))
		backupV1.POST("", handlers.BackupCreateHandler(s.backupService, s.cfg, s.logger))

		backupV1.POST("/:id/restore",
			logic.AdminOnly(s.userRepo, s.logger),
			handlers.BackupRestoreHandler(s.backupService, s.cfg, s.logger),
		)
		backupV1.DELETE("/:id",
			logic.AdminOnly(s.userRepo, s.logger),
			handlers.BackupDeleteHandler(s.backupService, s.logger),
		)
	}
}

// setupAdminRoutes configures admin-only endpoints
func (s *Server) setupAdminRoutes() {
	adminV1 := s.router.Group("/api/v1/admin")
	adminV1.Use(
		logic.AuthMiddleware(s.jwtService, s.tokenService, s.redis, s.logger),
		logic.CSRFMiddleware(s.redis, s.logger),
		logic.AdminOnly(s.userRepo, s.logger),
	)
	{
		adminV1.GET("/settings", settings.GetAdminSettingsHandler(s.cfg, s.logger))
		adminV1.PUT("/settings", settings.UpdateAdminSettingsHandler(s.cfg, s.settingsRepo, s.logger))
		adminV1.GET("/status", settings.SystemStatusHandler(s.db, s.logger))
		adminV1.GET("/users", settings.UserListHandler(s.userRepo, s.logger))
		adminV1.PUT("/users/:id/role", settings.UpdateUserRoleHandler(s.userRepo, s.logger))
		adminV1.POST("/maintenance", settings.ToggleMaintenanceModeHandler(s.logger))
		adminV1.GET("/audit-logs", settings.AuditLogHandler(s.db, s.logger))
		// Reconcile knowledge was moved to AI Handler
	}
}
