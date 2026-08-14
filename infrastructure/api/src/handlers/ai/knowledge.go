package ai

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	
	"github.com/sirupsen/logrus"
	"github.com/nas-ai/api/src/services/intelligence"
)

// ReconcileKnowledgeHandler triggers garbage collection for the AI knowledge index.
// It identifies orphaned embeddings (ghost knowledge) and removes them.
//
// @Summary Reconcile AI knowledge index
// @Description Garbage collect orphaned embeddings from files that no longer exist
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/admin/system/reconcile-knowledge [post]
func ReconcileKnowledgeHandler(
	secureAIFeeder *intelligence.SecureAIFeeder,
	dataPath string,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		if secureAIFeeder == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "AI feeder not configured",
				"message": "SecureAIFeeder is not initialized",
			})
			return
		}

		logger.WithField("request_id", requestID).Info("Starting knowledge index reconciliation")

		// Step 1: Build set of existing file IDs by walking the data directory
		existingFileIDs := make(map[string]bool)
		err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors, continue walking
			}
			if info.IsDir() {
				// Skip hidden directories and trash
				if strings.HasPrefix(info.Name(), ".") || info.Name() == ".trash" {
					return filepath.SkipDir
				}
				return nil
			}
			// Skip hidden files
			if strings.HasPrefix(info.Name(), ".") {
				return nil
			}
			// Add file name as ID
			existingFileIDs[info.Name()] = true
			return nil
		})

		if err != nil {
			logger.WithError(err).Error("Failed to walk data directory")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to scan files",
			})
			return
		}

		logger.WithField("existingFiles", len(existingFileIDs)).Info("Found existing files in storage")

		// Step 2: Call ReconcileIndex to find and delete orphans
		deleted, err := secureAIFeeder.ReconcileIndex(existingFileIDs)
		if err != nil {
			logger.WithError(err).Error("Knowledge reconciliation failed")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Reconciliation failed",
			})
			return
		}

		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"deleted":    deleted,
		}).Info("Knowledge index reconciliation complete")

		c.JSON(http.StatusOK, gin.H{
			"status":         "success",
			"deleted":        deleted,
			"existing_files": len(existingFileIDs),
			"message":        "Knowledge index reconciled successfully",
		})
	}
}
