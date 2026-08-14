package content

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Security constants
const MaxUploadSize = 100 * 1024 * 1024 // 100 MB

// AllowedMimeTypes defines the whitelist of permitted file types
var AllowedMimeTypes = map[string]bool{
	// Images
	"image/jpeg":    true,
	"image/jpg":     true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,

	// Documents
	"application/pdf": true,
	"text/plain":      true,
	"text/csv":        true,
	"text/markdown":   true,

	// Archives
	"application/zip":              true,
	"application/x-zip-compressed": true,
	"application/gzip":             true,
	"application/x-gzip":           true,
	"application/x-tar":            true,

	// Video
	"video/mp4":  true,
	"video/mpeg": true,
	"video/webm": true,

	// Audio
	"audio/mpeg": true,
	"audio/mp3":  true,
	"audio/wav":  true,
	"audio/ogg":  true,
}

// Magic number signatures for common file types
var magicNumbers = map[string][]byte{
	"image/jpeg":      {0xFF, 0xD8, 0xFF},
	"image/png":       {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
	"image/gif":       {0x47, 0x49, 0x46, 0x38},
	"application/pdf": {0x25, 0x50, 0x44, 0x46},
	"application/zip": {0x50, 0x4B, 0x03, 0x04},
	"video/mp4":       {0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}, // ftyp box
}

func ValidateFileSize(size int64) error {
	if size > MaxUploadSize {
		return fmt.Errorf("%w: file size %d bytes exceeds maximum of %d bytes", ErrFileTooLarge, size, MaxUploadSize)
	}
	return nil
}

// ValidateFileType checks magic numbers and extensions
func ValidateFileType(file multipart.File, filename string) (string, error) {
	// Read header
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read header: %w", err)
	}
	// Reset
	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("reset seek: %w", err)
	}

	// Detect MIME
	detectedType := http.DetectContentType(buffer[:n])
	if idx := strings.Index(detectedType, ";"); idx != -1 {
		detectedType = strings.TrimSpace(detectedType[:idx])
	}

	// Exception for Encrypted files
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".enc" {
		return "application/octet-stream", nil
	}

	// Magic Number check for octet-stream
	if detectedType == "application/octet-stream" {
		for mimeType, magic := range magicNumbers {
			if len(buffer) >= len(magic) && bytes.Equal(buffer[:len(magic)], magic) {
				detectedType = mimeType
				break
			}
		}
	}

	if !AllowedMimeTypes[detectedType] {
		// Log detailed error in caller, here just return error
		return "", fmt.Errorf("%w: %s (detected as %s)", ErrInvalidFileType, filename, detectedType)
	}

	// Dangerous Extension Blocklist
	dangerousExtensions := []string{
		".exe", ".bat", ".cmd", ".com", ".pif", ".scr", ".vbs", ".js", ".jar",
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".app", ".deb", ".rpm",
		".php", ".jsp", ".asp", ".aspx", ".cgi", ".pl", ".py", ".rb",
	}

	for _, dangerous := range dangerousExtensions {
		if ext == dangerous {
			return "", fmt.Errorf("%w: executable or script file extension not allowed (%s)", ErrInvalidFileType, ext)
		}
	}

	return detectedType, nil
}

// LogValidationFailure helper
func LogValidationFailure(logger *logrus.Logger, filename, detectedType string, err error) {
	logger.WithFields(logrus.Fields{
		"filename":      filename,
		"detected_type": detectedType,
		"error":         err.Error(),
	}).Warn("File validation failed")
}
