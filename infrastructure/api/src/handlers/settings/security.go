package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecuritySettingsRequest for saving security settings
type SecuritySettingsRequest struct {
	TwoFactorEnabled   bool     `json:"twoFactorEnabled"`
	PasswordMinLength  int      `json:"passwordMinLength"`
	SessionTimeoutMins int      `json:"sessionTimeoutMins"`
	AllowedIPs         []string `json:"allowedIPs"`
	BlockedIPs         []string `json:"blockedIPs"`
	MaxLoginAttempts   int      `json:"maxLoginAttempts"`
}

// SecuritySettingsGetHandler returns the current security settings
func SecuritySettingsGetHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		config, err := loadSetupConfig()
		if err != nil {
			logger.WithError(err).Error("Failed to load security settings")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
			return
		}

		if config == nil {
			// Return defaults
			c.JSON(http.StatusOK, SecuritySettingsRequest{
				TwoFactorEnabled:   false,
				PasswordMinLength:  8,
				SessionTimeoutMins: 60,
				AllowedIPs:         []string{},
				BlockedIPs:         []string{},
				MaxLoginAttempts:   5,
			})
			return
		}

		c.JSON(http.StatusOK, SecuritySettingsRequest{
			TwoFactorEnabled:   config.SecuritySettings.TwoFactorEnabled,
			PasswordMinLength:  config.SecuritySettings.PasswordMinLength,
			SessionTimeoutMins: config.SecuritySettings.SessionTimeoutMins,
			AllowedIPs:         config.SecuritySettings.AllowedIPs,
			BlockedIPs:         config.SecuritySettings.BlockedIPs,
			MaxLoginAttempts:   config.SecuritySettings.MaxLoginAttempts,
		})
	}
}

// SecuritySettingsSaveHandler saves security settings to setup.json
func SecuritySettingsSaveHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SecuritySettingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		// Load existing config
		config, err := loadSetupConfig()
		if err != nil {
			logger.WithError(err).Error("Failed to load config for security settings")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load config"})
			return
		}

		if config == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "setup not complete"})
			return
		}

		// Update security settings
		config.SecuritySettings = SecuritySettings{
			TwoFactorEnabled:   req.TwoFactorEnabled,
			PasswordMinLength:  req.PasswordMinLength,
			SessionTimeoutMins: req.SessionTimeoutMins,
			AllowedIPs:         req.AllowedIPs,
			BlockedIPs:         req.BlockedIPs,
			MaxLoginAttempts:   req.MaxLoginAttempts,
		}

		if err := saveSetupConfig(config); err != nil {
			logger.WithError(err).Error("Failed to save security settings")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
			return
		}

		logger.Info("Security settings updated successfully")
		c.JSON(http.StatusOK, gin.H{"status": "saved"})
	}
}
