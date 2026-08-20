package files

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	
	"github.com/sirupsen/logrus"
	"github.com/nas-ai/api/src/services/content"
)

// StorageUploadZipHandler handles the upload and extraction of ZIP files
func StorageUploadZipHandler(storageService *content.StorageManager, archiveService *content.ArchiveService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		scoped, ok := scopedStorage(c, storageService)
		if !ok {
			return
		}
		storageService = scoped.(*content.StorageManager)

		// Initialize logger with context
		requestID := c.GetString("RequestId")
		log := logger.WithFields(logrus.Fields{
			"component":  "ZipUpload",
			"request_id": requestID,
			"user_id":    c.GetString("user_id"),
		})

		// Get target path from query
		targetPath := c.PostForm("path")
		if targetPath == "" {
			targetPath = "/"
		}

		// Handle file upload
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}
		defer file.Close()

		// Check file extension
		if filepath.Ext(header.Filename) != ".zip" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only .zip files are allowed"})
			return
		}

		// Read file into memory (we have a limit on request body size in nginx/gin usually)
		// For huge files, we might want to stream, but for security scanning, memory is safer if size is capped.
		// Assuming MaxRequestBodySize is handled by middleware.
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, file)
		if err != nil {
			log.Error("Failed to read uploaded file: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}

		zipData := buf.Bytes()

		// Get secure absolute path for storage (using injected storageService)
		absDestPath, err := storageService.GetFullPath(targetPath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target path"})
			return
		}

		// Perform Secure Unzip using ArchiveService
		// We pass the bytes via a Reader
		result, err := archiveService.UnzipSecure(c.Request.Context(), bytes.NewReader(zipData), int64(len(zipData)), absDestPath)
		if err != nil {
			if strings.Contains(err.Error(), "SECURITY") {
				log.WithField("security_alert", true).Error(err)
				c.JSON(http.StatusForbidden, gin.H{"error": "Security check failed"})
			} else {
				log.Error("Extraction failed: ", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Extraction failed"})
			}
			return
		}

		// Success Response
		c.JSON(http.StatusOK, gin.H{
			"message":     "Zip extracted successfully",
			"files_count": result.FileCount,
			"total_bytes": result.TotalBytes,
			"target_path": targetPath,
		})
	}
}
