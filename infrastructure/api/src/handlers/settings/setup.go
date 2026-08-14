package settings

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/pathsafe"
	configsvc "github.com/nas-ai/api/src/services/config"
	"github.com/sirupsen/logrus"
)

// SetupConfig represents the initial system configuration
type SetupConfig struct {
	Version           string           `json:"version"`
	SetupComplete     bool             `json:"setupComplete"`
	StoragePath       string           `json:"storagePath"`
	EncryptionEnabled bool             `json:"encryptionEnabled"`
	AIModels          AIModels         `json:"aiModels"`
	NetworkSettings   NetworkSettings  `json:"networkSettings,omitempty"`
	BackupSettings    BackupSettings   `json:"backupSettings,omitempty"`
	SecuritySettings  SecuritySettings `json:"securitySettings,omitempty"`
	StorageMonitor    StorageMonitor   `json:"storageMonitor,omitempty"`
	CreatedAt         time.Time        `json:"createdAt"`
}

type AIModels struct {
	LLM        string   `json:"llm"`
	Embedding  string   `json:"embedding"`
	IndexPaths []string `json:"indexPaths"`
	AutoIndex  bool     `json:"autoIndex"`
}

// NetworkSettings holds network-related configuration
type NetworkSettings struct {
	RateLimitPerMin    int      `json:"rateLimitPerMin"`
	SessionTimeoutMins int      `json:"sessionTimeoutMins"`
	CORSOrigins        []string `json:"corsOrigins"`
}

// BackupSettings holds backup-related configuration
type BackupSettings struct {
	Schedule    string `json:"schedule"`    // Cron expression
	Destination string `json:"destination"` // Backup destination path
	Retention   int    `json:"retention"`   // Days to keep backups
}

// SecuritySettings holds security-related configuration
type SecuritySettings struct {
	TwoFactorEnabled   bool     `json:"twoFactorEnabled"`
	PasswordMinLength  int      `json:"passwordMinLength"`
	SessionTimeoutMins int      `json:"sessionTimeoutMins"`
	AllowedIPs         []string `json:"allowedIPs"`
	BlockedIPs         []string `json:"blockedIPs"`
	MaxLoginAttempts   int      `json:"maxLoginAttempts"`
}

// StorageMonitor holds storage monitoring configuration
type StorageMonitor struct {
	WarningThreshold  int  `json:"warningThreshold"`  // Percentage to warn at
	CriticalThreshold int  `json:"criticalThreshold"` // Percentage to alert at
	AutoCleanup       bool `json:"autoCleanup"`       // Auto-cleanup old files
	CleanupAgeDays    int  `json:"cleanupAgeDays"`    // Days before cleanup
}

// SetupRequest represents the setup wizard request
type SetupRequest struct {
	StoragePath       string   `json:"storagePath"`
	EncryptionEnabled bool     `json:"encryptionEnabled"`
	AIModels          AIModels `json:"aiModels"`
}

const setupConfigPath = "/var/lib/nas/setup.json"

// loadSetupConfig reads the setup configuration from disk
func loadSetupConfig() (*SetupConfig, error) {
	data, err := os.ReadFile(setupConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not configured yet
		}
		return nil, err
	}

	var config SetupConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// saveSetupConfig writes the setup configuration to disk
func saveSetupConfig(config *SetupConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(setupConfigPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(setupConfigPath, data, 0600)
}

// SetupStatusHandler returns the current setup status
func SetupStatusHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		config, err := loadSetupConfig()
		if err != nil {
			logger.WithError(err).Warn("Failed to load setup config")
			c.JSON(http.StatusOK, gin.H{
				"complete":    false,
				"storagePath": "",
			})
			return
		}

		if config == nil {
			c.JSON(http.StatusOK, gin.H{
				"complete":    false,
				"storagePath": "",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"complete":          config.SetupComplete,
			"storagePath":       config.StoragePath,
			"encryptionEnabled": config.EncryptionEnabled,
			"aiModels":          config.AIModels,
			"createdAt":         config.CreatedAt,
		})
	}
}

// SetupHandler processes the initial system setup
func SetupHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SetupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Validate storage path
		storagePath := filepath.Clean(strings.TrimSpace(req.StoragePath))
		if storagePath == "" || storagePath == "." {
			storagePath = "/mnt/data"
		}

		// The path is attacker-controlled, persisted and echoed back by the
		// public /system/setup-status endpoint, so it must be absolute and
		// confined to the configured storage roots before anything touches
		// the filesystem.
		if !filepath.IsAbs(storagePath) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage path"})
			return
		}
		if err := pathsafe.WithinAnyRoot(configsvc.AllowedStorageRootsFromEnv(), storagePath); err != nil {
			logger.WithField("path", storagePath).Warn("Setup rejected: storage path outside allowed roots")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage path"})
			return
		}
		req.StoragePath = storagePath

		// Check if path exists and is writable
		if err := os.MkdirAll(req.StoragePath, 0755); err != nil {
			logger.WithError(err).WithField("path", req.StoragePath).Warn("Storage path not accessible")
			// Don't fail - just warn
		}

		// Set defaults for AI models
		if req.AIModels.LLM == "" {
			req.AIModels.LLM = "qwen2.5:7b"
		}
		if req.AIModels.Embedding == "" {
			req.AIModels.Embedding = "mxbai-embed-large"
		}

		config := &SetupConfig{
			Version:           "2.1.1",
			SetupComplete:     true,
			StoragePath:       req.StoragePath,
			EncryptionEnabled: req.EncryptionEnabled,
			AIModels:          req.AIModels,
			CreatedAt:         time.Now(),
		}

		if err := saveSetupConfig(config); err != nil {
			logger.WithError(err).Error("Failed to save setup config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save configuration"})
			return
		}

		logger.WithFields(logrus.Fields{
			"storagePath":       config.StoragePath,
			"encryptionEnabled": config.EncryptionEnabled,
			"llmModel":          config.AIModels.LLM,
			"embeddingModel":    config.AIModels.Embedding,
		}).Info("System setup completed")

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"config":  config,
		})
	}
}

// AIWarmupHandler triggers AI model preloading
func AIWarmupHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Models []string `json:"models"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Fire and forget - warmup happens async
		go func() {
			for _, model := range req.Models {
				logger.WithField("model", model).Info("AI model warmup requested")
				// Here you would call Ollama to preload the model
				// For now, just log it
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"status": "warmup_started",
			"models": req.Models,
		})
	}
}
