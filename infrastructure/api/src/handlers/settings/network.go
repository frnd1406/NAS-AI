package settings

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NetworkSettingsRequest represents the network settings request body
type NetworkSettingsRequest struct {
	RateLimitPerMin    int      `json:"rate_limit_per_min"`
	SessionTimeoutMins int      `json:"session_timeout_mins"`
	CORSOrigins        []string `json:"cors_origins"`
}

// NetworkSettingsGetHandler returns current network settings from setup.json
func NetworkSettingsGetHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		config, err := loadSetupConfig()
		if err != nil {
			logger.WithError(err).Warn("Failed to load setup config for network settings")
		}

		// Defaults
		response := NetworkSettingsRequest{
			RateLimitPerMin:    100,
			SessionTimeoutMins: 60,
			CORSOrigins:        []string{},
		}

		// Override with values from config
		if config != nil {
			if config.NetworkSettings.RateLimitPerMin > 0 {
				response.RateLimitPerMin = config.NetworkSettings.RateLimitPerMin
			}
			if config.NetworkSettings.SessionTimeoutMins > 0 {
				response.SessionTimeoutMins = config.NetworkSettings.SessionTimeoutMins
			}
			if len(config.NetworkSettings.CORSOrigins) > 0 {
				response.CORSOrigins = config.NetworkSettings.CORSOrigins
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

// NetworkSettingsSaveHandler saves network settings to setup.json
func NetworkSettingsSaveHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		var req NetworkSettingsRequest
		if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settings format"})
			return
		}

		// Validate
		if req.RateLimitPerMin < 10 || req.RateLimitPerMin > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rate limit must be between 10 and 1000"})
			return
		}
		if req.SessionTimeoutMins < 5 || req.SessionTimeoutMins > 1440 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session timeout must be between 5 and 1440 minutes"})
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

		// Update network settings
		config.NetworkSettings = NetworkSettings{
			RateLimitPerMin:    req.RateLimitPerMin,
			SessionTimeoutMins: req.SessionTimeoutMins,
			CORSOrigins:        req.CORSOrigins,
		}

		// Persist
		if err := saveSetupConfig(config); err != nil {
			logger.WithError(err).Error("Failed to save network settings")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
			return
		}

		logger.WithFields(logrus.Fields{
			"request_id":      requestID,
			"rate_limit":      req.RateLimitPerMin,
			"session_timeout": req.SessionTimeoutMins,
		}).Info("Network settings persisted to setup.json")

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Network settings saved. Restart required for changes to take effect.",
		})
	}
}
