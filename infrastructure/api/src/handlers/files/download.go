package files

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nas-ai/api/src/services/content"
	"github.com/sirupsen/logrus"
)

// ==============================================================================
// Smart Download Handler - Phase 4: Hybrid Streaming
// ==============================================================================
//
// This handler implements intelligent file delivery based on encryption status:
//
// NONE (Performance Path):
//   - Uses Nginx X-Accel-Redirect for zero-copy file serving
//   - Go doesn't read the file - Nginx handles it directly
//   - Supports Range requests natively via Nginx
//
// USER (Secure Path):
//   - Streams file through DecryptStream to HTTP response
//   - Supports partial Range requests for video seeking
//   - Uses chunked AEAD decryption (64KB chunks)
//
// ==============================================================================

// DownloadRequest represents parameters for file download
type DownloadRequest struct {
	Path               string // Relative path from storage root
	EncryptionPassword string // Required for USER-encrypted files
	Inline             bool   // If true, display inline instead of download
}

// SmartDownloadHandler handles file downloads with encryption-aware streaming.
// Automatically detects encrypted files and routes to appropriate handler.
//
// Query Parameters:
//   - path: Required. Relative path to file
//   - password: Required for encrypted files. Decryption password
//   - inline: Optional. If "true", use inline Content-Disposition
//   - mode: Optional. "raw" or "auto" (default: "auto")
//   - raw: Stream ciphertext as-is (for offline decryption)
//   - auto: Decrypt if vault unlocked, return 423 if locked
//
// Headers:
//   - Range: Optional. Byte range for partial content (video seeking)
func SmartDownloadHandler(
	storage *content.StorageManager,
	honeySvc *content.HoneyfileService,
	deliverySvc *content.ContentDeliveryService,
	logger *logrus.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")
		path := c.Query("path")

		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
			return
		}

		// Parse optional parameters
		password := c.Query("password")
		// Security Enhancement: Checking header for password (preferred)
		if password == "" {
			password = c.GetHeader("X-Encryption-Password")
		}

		inline := c.Query("inline") == "true"
		mode := c.Query("mode")
		if mode == "" {
			mode = "auto" // Default: auto-decrypt if possible
		}

		// Get full filesystem path
		fullPath, err := storage.GetFullPath(path)
		if err != nil {
			handleStorageError(c, err, logger, requestID)
			return
		}

		// ==== SECURITY: Honeyfile Check ====
		if honeySvc != nil {
			meta := content.RequestMetadata{
				IPAddress: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Action:    "download",
			}
			if honeySvc.CheckAndTrigger(c.Request.Context(), fullPath, meta) {
				logger.WithFields(logrus.Fields{
					"request_id": requestID,
					"path":       path,
					"ip":         meta.IPAddress,
				}).Error("🔒 HONEYFILE TRIGGERED - ACCESS DENIED")
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
		}

		// Delegate to ContentDeliveryService
		result, err := deliverySvc.GetStream(c.Request.Context(), path, c.Request.Header.Get("Range"), password, mode, nil)
		if err != nil {
			if err.Error() == "VAULT_LOCKED" {
				c.JSON(http.StatusLocked, gin.H{
					"error":             "Vault is locked. Cannot decrypt file without password.",
					"encryption_status": "USER",
					"hint":              "Provide ?password=... or unlock vault first",
					"alternative":       "Use ?mode=raw to download encrypted file",
				})
				return
			}
			if err.Error() == "PASSWORD_REQUIRED" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":             "password required for encrypted file",
					"encryption_status": "USER",
				})
				return
			}
			if os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
				return
			}
			// Use common error handler for other errors
			handleStorageError(c, err, logger, requestID)
			return
		}

		// Clean up stream on return (if not nil)
		if result.Stream != nil {
			defer result.Stream.Close()
		}

		// Handle X-Accel-Redirect (Zero Copy)
		if result.XAccelRedirect != "" {
			c.Header("X-Accel-Redirect", result.XAccelRedirect)
			c.Header("X-Accel-Buffering", result.XAccelBuffering)
			c.Header("Content-Type", result.ContentType)
			c.Status(http.StatusOK)
			return
		}

		// Set Standard Headers
		c.Header("Content-Type", result.ContentType)

		disposition := "attachment"
		if inline {
			disposition = "inline"
		}
		// Extract filename from path for content disposition
		filename := filepath.Base(result.XAccelRedirect) // Fallback if XAccelRedirect is empty?
		// Wait, result doesn't have filename directly, but we can get it from path
		_, filename = filepath.Split(path)
		// Strip .enc for display if user mode decryption happened? The service handles content type detection on decoded name.
		// Detailed filename handling might be needed in result struct or just use path base.

		// Note: The service doesn't return the display filename in struct yet.
		// Use simple logic for now:
		if strings.HasSuffix(strings.ToLower(filename), ".enc") && mode != "raw" {
			filename = filename[:len(filename)-4]
		}

		c.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, filename))

		if result.ETag != "" {
			c.Header("ETag", result.ETag)
		}

		if result.ContentRange != "" {
			c.Header("Content-Range", result.ContentRange)
			c.Header("Content-Length", strconv.FormatInt(result.ContentLength, 10))
			c.Header("Accept-Ranges", "bytes")
		} else if result.ContentLength > 0 {
			// Only set Content-Length if known (might be chunked otherwise)
			c.Header("Content-Length", strconv.FormatInt(result.ContentLength, 10))
			c.Header("Accept-Ranges", "bytes")
		}

		c.Status(result.StatusCode)

		// Stream content
		if result.Stream != nil {
			if _, err := io.Copy(c.Writer, result.Stream); err != nil {
				logger.WithError(err).Warn("Stream interrupted")
			}
		}
	}
}
