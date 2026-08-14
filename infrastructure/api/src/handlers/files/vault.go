package files

import (
	"archive/zip"
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
	
	"github.com/sirupsen/logrus"
	"github.com/nas-ai/api/src/services/security"
)

// VaultStatusRequest is empty, just for documentation
type VaultSetupRequest struct {
	MasterPassword string `json:"masterPassword" binding:"required,min=8"`
}

type VaultUnlockRequest struct {
	MasterPassword string `json:"masterPassword" binding:"required"`
}

type VaultConfigRequest struct {
	VaultPath string `json:"vaultPath" binding:"required"`
}

// VaultStatusHandler returns the current vault status
// @Summary Get vault status
// @Description Returns whether the vault is locked, configured, and the vault path
// @Tags vault
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/vault/status [get]
func VaultStatusHandler(encSvc *security.EncryptionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, encSvc.GetStatus())
	}
}

// VaultSetupHandler initializes the vault with a master password
// @Summary Setup vault
// @Description Initialize the vault with a master password (first-time setup only)
// @Tags vault
// @Accept json
// @Produce json
// @Param request body VaultSetupRequest true "Setup request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/v1/vault/setup [post]
func VaultSetupHandler(encSvc *security.EncryptionService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req VaultSetupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: masterPassword required (min 8 characters)"})
			return
		}

		if err := encSvc.Setup(req.MasterPassword); err != nil {
			if err == security.ErrVaultAlreadySetup {
				c.JSON(http.StatusConflict, gin.H{"error": "vault is already configured"})
				return
			}
			logger.WithError(err).Error("vault setup failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to setup vault"})
			return
		}

		logger.Info("vault setup completed successfully")
		c.JSON(http.StatusOK, gin.H{
			"message": "vault setup completed",
			"status":  encSvc.GetStatus(),
		})
	}
}

// VaultUnlockHandler unlocks the vault with the master password
// @Summary Unlock vault
// @Description Unlock the vault using the master password
// @Tags vault
// @Accept json
// @Produce json
// @Param request body VaultUnlockRequest true "Unlock request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 423 {object} map[string]string
// @Router /api/v1/vault/unlock [post]
func VaultUnlockHandler(encSvc *security.EncryptionService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req VaultUnlockRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: masterPassword required"})
			return
		}

		if err := encSvc.Unlock(req.MasterPassword); err != nil {
			switch err {
			case security.ErrVaultNotSetup:
				c.JSON(http.StatusPreconditionFailed, gin.H{"error": "vault is not configured"})
			case security.ErrAlreadyUnlocked:
				c.JSON(http.StatusOK, gin.H{"message": "vault is already unlocked", "status": encSvc.GetStatus()})
			case security.ErrInvalidPassword:
				logger.Warn("vault unlock: invalid password attempt")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid master password"})
			default:
				logger.WithError(err).Error("vault unlock failed")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlock vault"})
			}
			return
		}

		logger.Info("vault unlocked successfully")
		c.JSON(http.StatusOK, gin.H{
			"message": "vault unlocked",
			"status":  encSvc.GetStatus(),
		})
	}
}

// VaultLockHandler locks the vault
// @Summary Lock vault
// @Description Lock the vault and wipe encryption keys from memory
// @Tags vault
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 423 {object} map[string]string
// @Router /api/v1/vault/lock [post]
func VaultLockHandler(encSvc *security.EncryptionService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := encSvc.Lock(); err != nil {
			if err == security.ErrAlreadyLocked {
				c.JSON(http.StatusOK, gin.H{"message": "vault is already locked", "status": encSvc.GetStatus()})
				return
			}
			logger.WithError(err).Error("vault lock failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to lock vault"})
			return
		}

		logger.Info("vault locked successfully")
		c.JSON(http.StatusOK, gin.H{
			"message": "vault locked",
			"status":  encSvc.GetStatus(),
		})
	}
}

// VaultConfigGetHandler returns the current vault configuration
// @Summary Get vault config
// @Description Returns the vault path configuration
// @Tags vault
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/vault/config [get]
func VaultConfigGetHandler(encSvc *security.EncryptionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"vaultPath":  encSvc.GetVaultPath(),
			"configured": encSvc.IsConfigured(),
		})
	}
}

// VaultConfigUpdateHandler updates the vault path (only when locked and not configured)
// @Summary Update vault config
// @Description Update the vault path (only when vault is locked)
// @Tags vault
// @Accept json
// @Produce json
// @Param request body VaultConfigRequest true "Config request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/v1/vault/config [put]
func VaultConfigUpdateHandler(encSvc *security.EncryptionService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req VaultConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: vaultPath required"})
			return
		}

		if err := encSvc.SetVaultPath(req.VaultPath); err != nil {
			logger.WithError(err).Warn("vault path update failed")
			c.JSON(http.StatusConflict, gin.H{"error": "failed to update vault path"})
			return
		}

		logger.WithField("vaultPath", req.VaultPath).Info("vault path updated")
		c.JSON(http.StatusOK, gin.H{
			"message":   "vault path updated",
			"vaultPath": encSvc.GetVaultPath(),
		})
	}
}

// VaultPanicHandler - EMERGENCY: Destroys all encryption keys immediately
// This is the "red button" for security emergencies.
// @Summary Emergency key destruction
// @Description DANGER: Immediately destroys all encryption keys from memory. Use only in security emergencies.
// @Tags vault
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/vault/panic [post]
func VaultPanicHandler(encSvc *security.EncryptionService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Warn("PANIC: Emergency key destruction triggered!")

		// 1. Force lock and wipe all keys (multi-pass wipe + GC)
		_ = encSvc.Lock() // Ignore error - we want to destroy keys regardless

		// 2. Log the security event
		logger.WithFields(logrus.Fields{
			"event":  "PANIC_KEY_DESTRUCTION",
			"status": "KEYS_DESTROYED",
		}).Warn("All encryption keys have been destroyed")

		// 3. Respond with status
		c.JSON(http.StatusOK, gin.H{
			"status":  "SYSTEM_LOCKED",
			"keys":    "DESTROYED",
			"message": "Emergency lockdown complete. All keys wiped from memory.",
		})
	}
}

// VaultExportConfigHandler exports vault configuration files for backup
// @Summary Export vault config
// @Description Downloads salt.bin and config.json as a ZIP file for backup. ⚠️ Store securely!
// @Tags vault
// @Produce application/zip
// @Success 200 {file} binary "vault_backup.zip"
// @Failure 412 {object} map[string]string "Vault not configured"
// @Failure 500 {object} map[string]string
// @Router /api/v1/vault/export-config [get]
func VaultExportConfigHandler(encSvc *security.EncryptionService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		files, err := encSvc.GetVaultConfigFiles()
		if err != nil {
			if err == security.ErrVaultNotSetup {
				c.JSON(http.StatusPreconditionFailed, gin.H{"error": "vault is not configured"})
				return
			}
			logger.WithError(err).Error("failed to get vault config files")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read vault files"})
			return
		}

		// Create ZIP in memory
		var buf bytes.Buffer
		zipWriter := zip.NewWriter(&buf)

		for _, file := range files {
			w, err := zipWriter.Create(file.Filename)
			if err != nil {
				logger.WithError(err).Error("failed to create zip entry")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create backup"})
				return
			}
			if _, err := w.Write(file.Content); err != nil {
				logger.WithError(err).Error("failed to write zip entry")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write backup"})
				return
			}
		}

		if err := zipWriter.Close(); err != nil {
			logger.WithError(err).Error("failed to close zip")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to finalize backup"})
			return
		}

		logger.Info("vault config exported for backup")

		// Set headers for file download
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", "attachment; filename=vault_backup.zip")
		c.Header("X-Vault-Warning", "⚠️ Store this file securely! Without it, your password is useless.")
		c.Data(http.StatusOK, "application/zip", buf.Bytes())
	}
}
