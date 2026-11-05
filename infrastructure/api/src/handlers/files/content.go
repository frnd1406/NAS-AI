package files

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	storagedrv "github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
	"github.com/sirupsen/logrus"
)

// FileContentHandler returns the raw content of a file for preview
// GET /api/v1/files/content?path=… (absolute under the caller's home, or relative to it)
func FileContentHandler(storage content.StorageService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		storageService, ok := scopedStorage(c, storage)
		if !ok {
			return
		}
		filePath := c.Query("path")
		if filePath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing path parameter"})
			return
		}

		relPath := strings.TrimPrefix(filepath.ToSlash(filePath), "/")
		if root, err := storageService.GetFullPath("."); err == nil {
			absRoot := filepath.Clean(root)
			// GetFullPath(".") may append "."; prefer parent when that happens.
			if strings.HasSuffix(absRoot, string(filepath.Separator)+".") {
				absRoot = filepath.Dir(absRoot)
			}
			absRootSlash := filepath.ToSlash(absRoot)
			fpSlash := filepath.ToSlash(filePath)
			if strings.HasPrefix(fpSlash, absRootSlash+"/") {
				relPath = strings.TrimPrefix(fpSlash, absRootSlash+"/")
			} else if fpSlash == absRootSlash {
				relPath = ""
			}
		}
		relPath = strings.TrimPrefix(relPath, "/")

		// Open file via service
		file, info, contentType, err := storageService.Open(relPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
				return
			}
			if errors.Is(err, storagedrv.ErrPathTraversal) {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
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
		case ".html", ".htm", ".js", ".css", ".xml", ".svg", ".go", ".py":
			// Previews are shown as source. Serving user-uploaded markup with an
			// executable type from the API origin would be stored XSS.
			contentType = "text/plain; charset=utf-8"
		}
		if isActiveContentType(contentType) {
			contentType = "text/plain; charset=utf-8"
		}

		c.Header("Content-Type", contentType)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Security-Policy", "sandbox; default-src 'none'")
		c.Data(http.StatusOK, contentType, contentBytes)
	}
}

// isActiveContentType reports whether a browser would execute or render the
// type as a document (HTML, SVG, XML, scripts) rather than display it inertly.
func isActiveContentType(contentType string) bool {
	ct := strings.ToLower(contentType)
	for _, active := range []string{"html", "xml", "svg", "javascript", "ecmascript"} {
		if strings.Contains(ct, active) {
			return true
		}
	}
	return false
}
