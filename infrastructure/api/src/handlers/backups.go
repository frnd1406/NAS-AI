package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	
	"github.com/sirupsen/logrus"
	"github.com/nas-ai/api/src/services/operations"
)

func BackupListHandler(backupSvc *operations.BackupService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := backupSvc.ListBackups()
		if err != nil {
			logger.WithError(err).Warn("backup: list failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func BackupCreateHandler(backupSvc *operations.BackupService, cfg *config.Config, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// SECURITY FIX [BUG-GO-010]: Removed dynamic path parameter - uses configured path only
		b, err := backupSvc.CreateBackup()
		if err != nil {
			logger.WithError(err).Warn("backup: create failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create backup"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"backup": b})
	}
}

func BackupRestoreHandler(backupSvc *operations.BackupService, cfg *config.Config, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		userID := c.GetString("user_id")
		id := c.Param("id")

		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "backup id required",
					"request_id": requestID,
				},
			})
			return
		}

		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"user_id":    userID,
			"backup_id":  id,
		}).Warn("CRITICAL: Backup restore initiated by admin")

		// SECURITY: Create emergency pre-restore backup BEFORE destructive operation
		logger.Info("Creating emergency pre-restore backup...")
		// SECURITY FIX [BUG-GO-010]: Removed dynamic path parameter - uses configured path only
		emergencyBackup, err := backupSvc.CreateBackup()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("CRITICAL: Pre-restore safety backup FAILED - aborting restore")

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "backup_failed",
					"message":    "Pre-restore safety backup failed - restore operation aborted for safety",
					"request_id": requestID,
				},
			})
			return
		}

		logger.WithFields(logrus.Fields{
			"request_id":       requestID,
			"emergency_backup": emergencyBackup.ID,
		}).Info("Emergency pre-restore backup created successfully")

		// Proceed with restore
		if err := backupSvc.RestoreBackup(id); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id":       requestID,
				"error":            err.Error(),
				"emergency_backup": emergencyBackup.ID,
			}).Error("backup: restore failed")

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":            "restore_failed",
					"message":         "failed to restore backup",
					"emergency_backup": emergencyBackup.ID,
					"recovery_hint":   "Use emergency backup to recover: POST /api/v1/backups/" + emergencyBackup.ID + "/restore",
					"request_id":      requestID,
				},
			})
			return
		}

		logger.WithFields(logrus.Fields{
			"request_id":       requestID,
			"user_id":          userID,
			"backup_id":        id,
			"emergency_backup": emergencyBackup.ID,
		}).Info("Backup restored successfully")

		c.JSON(http.StatusOK, gin.H{
			"status":           "restored",
			"backup_id":        id,
			"emergency_backup": emergencyBackup.ID,
			"message":          "Backup restored successfully. Emergency pre-restore backup saved as " + emergencyBackup.ID,
		})
	}
}

func BackupDeleteHandler(backupSvc *operations.BackupService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
			return
		}
		if err := backupSvc.DeleteBackup(id); err != nil {
			logger.WithError(err).Warn("backup: delete failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete backup"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "id": id})
	}
}
