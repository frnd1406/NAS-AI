package files

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/content"
	"github.com/sirupsen/logrus"
)

// FileContentHandler returns the raw content of a file for preview
// GET /api/v1/files/content?path=/mnt/data/...
// FileContentHandler returns the raw content of a file for preview
// GET /api/v1/files/content?path=/mnt/data/...
func FileContentHandler(storageService content.StorageService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		filePath := c.Query("path")
		if filePath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing path parameter"})
			return
		}

		// Use StorageService to open the file (handles path security logic internally if implemented,
		// but checking prefix here is still good defensive practice if storageService assumes relative paths)
		// Assuming storageService.Open takes relative path or handles it.
		// Existing StorageService Open takes relative path usually.
		// filePath from query might be absolute or relative.
		// Logic: If /mnt/data is root, strip it.
		relPath := strings.TrimPrefix(filePath, "/mnt/data/")
		relPath = strings.TrimPrefix(relPath, "/")

		// Open file via service
		file, info, contentType, err := storageService.Open(relPath)
		if err != nil {
			if os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
				return
			}
			logger.WithError(err).Error("Failed to open file via storage service")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access file"})
			return
		}
		defer file.Close()

		// Ensure it's a file, not a directory
		if info.IsDir() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path is a directory, not a file"})
			return
		}

		// Security: Limit file size to prevent abuse (max 10MB for preview)
		const maxPreviewSize = 10 * 1024 * 1024 // 10MB
		if info.Size() > maxPreviewSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":     "file too large for preview",
				"max_size":  maxPreviewSize,
				"file_size": info.Size(),
			})
			return
		}

		// Read file content
		contentBytes, err := io.ReadAll(file)
		if err != nil {
			logger.WithError(err).Error("Failed to read file content")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
			return
		}

		// If contentType from storage is empty or generic, detect it
		if contentType == "" || contentType == "application/octet-stream" {
			contentType = http.DetectContentType(contentBytes)
		}

		// Override for common text formats (re-implementing original logic)
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".txt", ".log":
			contentType = "text/plain; charset=utf-8"
		case ".json":
			contentType = "application/json; charset=utf-8"
		case ".md":
			contentType = "text/markdown; charset=utf-8"
		case ".html":
			contentType = "text/html; charset=utf-8"
		case ".js":
			contentType = "application/javascript; charset=utf-8"
		case ".css":
			contentType = "text/css; charset=utf-8"
		case ".xml":
			contentType = "application/xml; charset=utf-8"
		case ".go":
			contentType = "text/plain; charset=utf-8"
		case ".py":
			contentType = "text/plain; charset=utf-8"
		}

		c.Header("Content-Type", contentType)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(http.StatusOK, contentType, contentBytes)
	}
}
