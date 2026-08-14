package logic

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/security"
)

// VaultGuard blocks requests when the vault is locked
// This middleware should be applied to routes that require encryption access
func VaultGuard(encSvc *security.EncryptionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !encSvc.IsUnlocked() {
			c.JSON(http.StatusLocked, gin.H{
				"error":   "🔒 Vault ist gesperrt",
				"message": "Der Vault muss entsperrt werden, um auf verschlüsselte Dateien zuzugreifen. Bitte gib dein Master-Passwort ein.",
				"code":    "VAULT_LOCKED",
				"action":  "Gehe zu Einstellungen → Vault und entsperre mit deinem Master-Passwort.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// VaultConfigured ensures the vault is configured before proceeding
// Used for routes that require the vault to be set up first
func VaultConfigured(encSvc *security.EncryptionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !encSvc.IsConfigured() {
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"error":   "🔐 Vault nicht eingerichtet",
				"message": "Der Vault muss zuerst eingerichtet werden, bevor verschlüsselte Dateien genutzt werden können. Lege ein Master-Passwort fest, um zu starten.",
				"code":    "VAULT_NOT_CONFIGURED",
				"action":  "Gehe zu Einstellungen → Vault und erstelle ein Master-Passwort.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
