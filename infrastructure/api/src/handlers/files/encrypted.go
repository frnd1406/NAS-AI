package files

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/nas-ai/api/src/services/content"
	"github.com/nas-ai/api/src/services/intelligence"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// EncryptedStorageListHandler lists encrypted files
func EncryptedStorageListHandler(encStorage content.EncryptedStorageServiceInterface, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		path := c.Query("path")

		// Check if vault is unlocked
		if !encStorage.IsEncryptionEnabled() {
			c.JSON(http.StatusLocked, gin.H{
				"error": gin.H{
					"code":    "vault_locked",
					"message": "Vault is locked. Unlock to access encrypted files.",
				},
			})
			return
		}

		items, err := encStorage.ListEncrypted(path)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"path":       path,
				"error":      err.Error(),
			}).Error("encrypted storage: list failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list encrypted files"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items":     items,
			"encrypted": true,
			"basePath":  encStorage.GetEncryptedBasePath(),
		})
	}
}

// EncryptedStorageUploadHandler uploads and encrypts a file
// If aiFeeder is provided, it also triggers AI indexing of the encrypted content
func EncryptedStorageUploadHandler(encStorage content.EncryptedStorageServiceInterface, aiFeeder *intelligence.SecureAIFeeder, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		path := c.PostForm("path")

		// Check if vault is unlocked
		if !encStorage.IsEncryptionEnabled() {
			c.JSON(http.StatusLocked, gin.H{
				"error": gin.H{
					"code":    "vault_locked",
					"message": "Vault is locked. Unlock to upload encrypted files.",
				},
			})
			return
		}

		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}

		src, err := fileHeader.Open()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("encrypted storage: open upload file failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read upload"})
			return
		}
		defer src.Close()

		result, err := encStorage.SaveEncrypted(path, src, fileHeader)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"path":       path,
				"filename":   fileHeader.Filename,
				"error":      err.Error(),
			}).Error("encrypted storage: save failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save encrypted file"})
			return
		}

		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"filename":   fileHeader.Filename,
			"path":       result.Path,
		}).Info("File encrypted and saved successfully")

		// Trigger AI indexing (async, non-blocking)
		// The encrypted file is decrypted in-memory and pushed to the AI agent
		aiIndexed := false
		if aiFeeder != nil {
			go func() {
				// result.Path is the encrypted path (e.g., /media/user/DEMO/geheim.pdf.enc)
				// originalPath is for source citations in AI responses
				originalPath := result.Path
				if err := aiFeeder.FeedEncryptedFile(result.Path, originalPath, result.FileID, result.MimeType); err != nil {
					logger.WithFields(logrus.Fields{
						"fileID": result.FileID,
						"path":   result.Path,
						"error":  err.Error(),
					}).Warn("encrypted storage: AI indexing failed (non-fatal)")
				} else {
					logger.WithField("fileID", result.FileID).Info("encrypted storage: AI indexing complete")
				}
			}()
			aiIndexed = true
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"encrypted":  true,
			"path":       result.Path,
			"ai_indexed": aiIndexed,
		})
	}
}

// EncryptedStorageDownloadHandler downloads and decrypts a file
func EncryptedStorageDownloadHandler(encStorage content.EncryptedStorageServiceInterface, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		path := c.Query("path")

		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
			return
		}

		// Check if vault is unlocked
		if !encStorage.IsEncryptionEnabled() {
			c.JSON(http.StatusLocked, gin.H{
				"error": gin.H{
					"code":    "vault_locked",
					"message": "Vault is locked. Unlock to download encrypted files.",
				},
			})
			return
		}

		reader, info, mimeType, err := encStorage.OpenEncrypted(path)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"path":       path,
				"error":      err.Error(),
			}).Error("encrypted storage: download failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt file"})
			return
		}
		defer reader.Close()

		// Get original filename (strip .enc)
		originalName := filepath.Base(path)

		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", originalName))
		c.Header("X-Encrypted", "true")
		c.DataFromReader(http.StatusOK, info.Size(), mimeType, reader, nil)
	}
}

// EncryptedStorageDeleteHandler deletes an encrypted file
func EncryptedStorageDeleteHandler(encStorage content.EncryptedStorageServiceInterface, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		path := c.Query("path")

		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
			return
		}

		if err := encStorage.DeleteEncrypted(path); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"path":       path,
				"error":      err.Error(),
			}).Error("encrypted storage: delete failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete encrypted file"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "deleted", "encrypted": true})
	}
}

// EncryptedStorageStatusHandler returns encryption status
func EncryptedStorageStatusHandler(encStorage content.EncryptedStorageServiceInterface, encService security.EncryptionServiceInterface) gin.HandlerFunc {
	return func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{
			"enabled":    encStorage.IsEncryptionEnabled(),
			"locked":     !encStorage.IsEncryptionEnabled(),
			"configured": encService.IsConfigured(),
			"basePath":   encStorage.GetEncryptedBasePath(),
		})
	}
}

// EncryptedStoragePreviewHandler decrypts and streams file for preview (images, etc)
func EncryptedStoragePreviewHandler(encStorage content.EncryptedStorageServiceInterface, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		path := c.Query("path")

		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
			return
		}

		if !encStorage.IsEncryptionEnabled() {
			c.JSON(http.StatusLocked, gin.H{
				"error": gin.H{
					"code":    "vault_locked",
					"message": "Vault is locked.",
				},
			})
			return
		}

		reader, info, mimeType, err := encStorage.OpenEncrypted(path)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"path":       path,
				"error":      err.Error(),
			}).Error("encrypted storage: preview failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt for preview"})
			return
		}
		defer reader.Close()

		// Read all data for preview
		data, err := io.ReadAll(reader)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read decrypted data"})
			return
		}

		c.Header("X-Encrypted", "true")
		c.Data(http.StatusOK, mimeType, data)

		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"path":       path,
			"size":       info.Size(),
		}).Debug("Encrypted file previewed")
	}
}
