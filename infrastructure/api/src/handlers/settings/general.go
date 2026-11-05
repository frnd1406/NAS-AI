package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/config"
)

type BackupSettingsRequest struct {
	Schedule  string `json:"schedule" binding:"required"`
	Retention int    `json:"retention" binding:"required"`
	Path      string `json:"path" binding:"required"`
}

type ValidatePathRequest struct {
	Path string `json:"path" binding:"required"`
}

type BackupSettingsResponse struct {
	Schedule  string `json:"schedule"`
	Retention int    `json:"retention"`
	Path      string `json:"path"`
}

type SettingsResponse struct {
	Backup BackupSettingsResponse `json:"backup"`
}

// SystemSettingsHandler returns the current system configuration
// GET /system/settings
func SystemSettingsHandler(settingsSvc *config.SettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		backup := settingsSvc.GetBackupSettings()

		c.JSON(http.StatusOK, SettingsResponse{
			Backup: BackupSettingsResponse{
				Schedule:  backup.Schedule,
				Retention: backup.Retention,
				Path:      backup.Path,
			},
		})
	}
}

// ValidatePathHandler checks if a path is valid for usage
// POST /system/validate-path
func ValidatePathHandler(settingsSvc *config.SettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ValidatePathRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		result := settingsSvc.ValidatePath(req.Path)
		c.JSON(http.StatusOK, result)
	}
}

// UpdateBackupSettingsHandler updates the backup configuration
// PUT /system/settings/backup
func UpdateBackupSettingsHandler(settingsSvc *config.SettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BackupSettingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		dto := config.BackupSettingsDTO{
			Schedule:  req.Schedule,
			Retention: req.Retention,
			Path:      req.Path,
		}

		if err := settingsSvc.UpdateBackupSettings(c.Request.Context(), dto); err != nil {
			// Do not echo the internal error (may contain paths/DB detail);
			// the service logs the specifics.
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup settings"})
			return
		}

		// Return updated settings
		backup := settingsSvc.GetBackupSettings()
		c.JSON(http.StatusOK, SettingsResponse{
			Backup: BackupSettingsResponse{
				Schedule:  backup.Schedule,
				Retention: backup.Retention,
				Path:      backup.Path,
			},
		})
	}
}
