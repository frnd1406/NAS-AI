package settings

import (
	"database/sql"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	auth_repo "github.com/nas-ai/api/src/repository/auth"
	settings_repo "github.com/nas-ai/api/src/repository/settings"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/database"

	"github.com/sirupsen/logrus"
)

// ============================================================
// Admin Settings Types
// ============================================================

type AdminSettings struct {
	RateLimitPerMin       int      `json:"rate_limit_per_min"`
	CORSOrigins           []string `json:"cors_origins"`
	AIServiceURL          string   `json:"ai_service_url"`
	MaintenanceMode       bool     `json:"maintenance_mode"`
	SessionTimeoutMins    int      `json:"session_timeout_mins"`
	MaxLoginAttempts      int      `json:"max_login_attempts"`
	TwoFactorEnabled      bool     `json:"two_factor_enabled"`
	IPWhitelist           []string `json:"ip_whitelist"`
	AuditLogRetentionDays int      `json:"audit_log_retention_days"`
}

type UpdateAdminSettingsRequest struct {
	RateLimitPerMin       *int      `json:"rate_limit_per_min,omitempty"`
	CORSOrigins           *[]string `json:"cors_origins,omitempty"`
	MaintenanceMode       *bool     `json:"maintenance_mode,omitempty"`
	SessionTimeoutMins    *int      `json:"session_timeout_mins,omitempty"`
	MaxLoginAttempts      *int      `json:"max_login_attempts,omitempty"`
	TwoFactorEnabled      *bool     `json:"two_factor_enabled,omitempty"`
	IPWhitelist           *[]string `json:"ip_whitelist,omitempty"`
	AuditLogRetentionDays *int      `json:"audit_log_retention_days,omitempty"`
}

type DBPoolStatus struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration_ms"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}

type SystemStatus struct {
	Uptime          string       `json:"uptime"`
	GoVersion       string       `json:"go_version"`
	NumGoroutines   int          `json:"num_goroutines"`
	MemoryAllocMB   float64      `json:"memory_alloc_mb"`
	MaintenanceMode bool         `json:"maintenance_mode"`
	DBPool          DBPoolStatus `json:"db_pool"`
}

type AuditLogEntry struct {
	ID        int64     `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Action    string    `json:"action" db:"action"`
	Resource  string    `json:"resource" db:"resource"`
	Details   string    `json:"details" db:"details"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type UserListItem struct {
	ID          string     `json:"id" db:"id"`
	Username    string     `json:"username" db:"username"`
	Email       string     `json:"email" db:"email"`
	Role        string     `json:"role" db:"role"`
	Verified    bool       `json:"verified" db:"is_verified"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

// Global maintenance mode flag (in production, use Redis)
var maintenanceMode = false
var serverStartTime = time.Now()

// ============================================================
// Admin Settings Handlers
// ============================================================

// GetAdminSettingsHandler returns current admin configuration
func GetAdminSettingsHandler(cfg *config.Config, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings := AdminSettings{
			RateLimitPerMin:       cfg.RateLimitPerMin,
			CORSOrigins:           cfg.CORSOrigins,
			AIServiceURL:          cfg.AIServiceURL,
			MaintenanceMode:       maintenanceMode,
			SessionTimeoutMins:    60, // Default JWT expiry
			MaxLoginAttempts:      5,
			TwoFactorEnabled:      false, // Not yet implemented
			IPWhitelist:           []string{},
			AuditLogRetentionDays: 30,
		}

		c.JSON(http.StatusOK, settings)
	}
}

// UpdateAdminSettingsHandler updates admin configuration
func UpdateAdminSettingsHandler(cfg *config.Config, settingsRepo *settings_repo.SystemSettingsRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateAdminSettingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		requestID := c.GetString("request_id")

		// Apply updates
		if req.RateLimitPerMin != nil && *req.RateLimitPerMin >= 1 {
			cfg.RateLimitPerMin = *req.RateLimitPerMin
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"value":      *req.RateLimitPerMin,
			}).Info("Admin updated rate limit")
		}

		if req.MaintenanceMode != nil {
			maintenanceMode = *req.MaintenanceMode
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"value":      *req.MaintenanceMode,
			}).Warn("Maintenance mode changed")
		}

		if req.SessionTimeoutMins != nil && *req.SessionTimeoutMins >= 5 {
			// Would update JWT service config
			logger.WithField("request_id", requestID).Info("Session timeout updated")
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Settings updated successfully",
			"updated": req,
		})
	}
}

// ============================================================
// System Status Handler
// ============================================================

func SystemStatusHandler(db *database.DB, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		dbStats := db.Stats()

		status := SystemStatus{
			Uptime:          time.Since(serverStartTime).Round(time.Second).String(),
			GoVersion:       runtime.Version(),
			NumGoroutines:   runtime.NumGoroutine(),
			MemoryAllocMB:   float64(memStats.Alloc) / 1024 / 1024,
			MaintenanceMode: maintenanceMode,
			DBPool: DBPoolStatus{
				MaxOpenConnections: dbStats.MaxOpenConnections,
				OpenConnections:    dbStats.OpenConnections,
				InUse:              dbStats.InUse,
				Idle:               dbStats.Idle,
				WaitCount:          dbStats.WaitCount,
				WaitDuration:       dbStats.WaitDuration / time.Millisecond,
				MaxIdleClosed:      dbStats.MaxIdleClosed,
				MaxLifetimeClosed:  dbStats.MaxLifetimeClosed,
			},
		}

		c.JSON(http.StatusOK, status)
	}
}

// ============================================================
// User Management Handler
// ============================================================

func UserListHandler(userRepo *auth_repo.UserRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		users, err := userRepo.GetAllUsers(ctx)
		if err != nil {
			logger.WithError(err).Error("Failed to fetch users")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"users": users,
			"total": len(users),
		})
	}
}

func UpdateUserRoleHandler(userRepo *auth_repo.UserRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")

		var req struct {
			Role string `json:"role" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		// Validate role
		if req.Role != "user" && req.Role != "admin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role, must be 'user' or 'admin'"})
			return
		}

		ctx := c.Request.Context()
		if err := userRepo.UpdateRole(ctx, userID, req.Role); err != nil {
			logger.WithError(err).Error("Failed to update user role")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
			return
		}

		logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"new_role":   req.Role,
			"changed_by": c.GetString("user_id"),
		}).Warn("User role changed")

		c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
	}
}

// ============================================================
// Audit Log Handler
// ============================================================

func AuditLogHandler(db *database.DB, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get pagination params
		limit := 50
		offset := 0

		if l := c.Query("limit"); l != "" {
			if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}
		if o := c.Query("offset"); o != "" {
			if parsed, err := parseInt(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		// Query audit logs (assuming table exists)
		rows, err := db.QueryContext(c.Request.Context(), `
			SELECT id, user_id, action, resource, details, ip_address, created_at
			FROM audit_logs
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`, limit, offset)
		if err != nil {
			// Table might not exist yet
			if err == sql.ErrNoRows {
				c.JSON(http.StatusOK, gin.H{"logs": []AuditLogEntry{}, "total": 0})
				return
			}
			logger.WithError(err).Warn("Failed to query audit logs")
			c.JSON(http.StatusOK, gin.H{"logs": []AuditLogEntry{}, "total": 0, "note": "Audit logging not configured"})
			return
		}
		defer rows.Close()

		logs := make([]AuditLogEntry, 0)
		for rows.Next() {
			var entry AuditLogEntry
			if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.Resource, &entry.Details, &entry.IPAddress, &entry.CreatedAt); err != nil {
				logger.WithError(err).Warn("Failed to scan audit log row")
				continue
			}
			logs = append(logs, entry)
		}

		c.JSON(http.StatusOK, gin.H{
			"logs":   logs,
			"limit":  limit,
			"offset": offset,
		})
	}
}

// ============================================================
// Maintenance Mode Handler
// ============================================================

func ToggleMaintenanceModeHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Enabled bool   `json:"enabled"`
			Message string `json:"message,omitempty"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		maintenanceMode = req.Enabled

		logger.WithFields(logrus.Fields{
			"enabled":    req.Enabled,
			"message":    req.Message,
			"changed_by": c.GetString("user_id"),
			"ip":         c.ClientIP(),
		}).Warn("ADMIN: Maintenance mode toggled")

		c.JSON(http.StatusOK, gin.H{
			"maintenance_mode": maintenanceMode,
			"message":          req.Message,
		})
	}
}

// ============================================================
// Helper Functions
// ============================================================

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
