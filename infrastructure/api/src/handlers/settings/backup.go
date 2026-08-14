package settings

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// BackupSettingsConfigRequest represents the backup settings for setup.json persistence
// Named differently from BackupSettingsRequest in settings.go to avoid conflict
type BackupSettingsConfigRequest struct {
	Schedule    string `json:"backup_schedule"`
	Destination string `json:"backup_destination"`
	Retention   int    `json:"backup_retention_days"`
}

// BackupSettingsGetHandler returns current backup settings from setup.json
func BackupSettingsGetHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		config, err := loadSetupConfig()
		if err != nil {
			logger.WithError(err).Warn("Failed to load setup config for backup settings")
		}

		// Defaults
		response := BackupSettingsConfigRequest{
			Schedule:    "0 3 * * *", // Daily at 3am
			Destination: "/mnt/backups",
			Retention:   30,
		}

		// Override with values from config
		if config != nil {
			if config.BackupSettings.Schedule != "" {
				response.Schedule = config.BackupSettings.Schedule
			}
			if config.BackupSettings.Destination != "" {
				response.Destination = config.BackupSettings.Destination
			}
			if config.BackupSettings.Retention > 0 {
				response.Retention = config.BackupSettings.Retention
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

// BackupSettingsSaveHandler saves backup settings to setup.json
func BackupSettingsSaveHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var req BackupSettingsConfigRequest
		if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settings format"})
			return
		}

		// Validate retention
		if req.Retention < 1 || req.Retention > 365 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "retention must be between 1 and 365 days"})
			return
		}

		// Load existing config
		config, err := loadSetupConfig()
		if err != nil {
			logger.WithError(err).Warn("Failed to load setup config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load config"})
			return
		}

		if config == nil {
			config = &SetupConfig{
				Version:       "2.1",
				SetupComplete: true,
				StoragePath:   "/mnt/data",
			}
		}

		// Update backup settings
		config.BackupSettings = BackupSettings{
			Schedule:    req.Schedule,
			Destination: req.Destination,
			Retention:   req.Retention,
		}

		// Persist
		if err := saveSetupConfig(config); err != nil {
			logger.WithError(err).Error("Failed to save backup settings")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
			return
		}

		logger.WithFields(logrus.Fields{
			"request_id":  requestID,
			"schedule":    req.Schedule,
			"destination": req.Destination,
			"retention":   req.Retention,
		}).Info("Backup settings persisted to setup.json")

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Backup settings saved",
		})
	}
}
