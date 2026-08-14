package files

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	
	"github.com/sirupsen/logrus"
	"github.com/nas-ai/api/src/services/content"
)

// UploadSession tracks an active chunked upload
type UploadSession struct {
	ID           string
	OriginalName string
	TempPath     string
	TotalSize    int64
	CurrentSize  int64
	Mutex        sync.Mutex
}

// BlobStorageHandler manages chunked encrypted uploads
type BlobStorageHandler struct {
	storage  *content.StorageManager
	sessions map[string]*UploadSession
	mu       sync.RWMutex
	logger   *logrus.Logger
}

func NewBlobStorageHandler(storage *content.StorageManager, logger *logrus.Logger) *BlobStorageHandler {
	return &BlobStorageHandler{
		storage:  storage,
		sessions: make(map[string]*UploadSession),
		logger:   logger,
	}
}

// InitUpload starts a new chunked upload session
// POST /api/v1/vault/upload/init
// Body: { "filename": "foo.txt.enc", "total_size": 12345 }
func (h *BlobStorageHandler) InitUpload(c *gin.Context) {
	var req struct {
		Filename  string `json:"filename" binding:"required"`
		TotalSize int64  `json:"total_size" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate extension (must be .enc)
	if filepath.Ext(req.Filename) != ".enc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only .enc files allowed in vault"})
		return
	}

	uploadID := uuid.New().String()

	// Create temp path
	// We use the storage service's base path to ensure we are within limits
	// But we bypass the public list/save logic for raw access
	// We'll use a hidden .uploads directory
	// Ensure .uploads directory exists
	// We use Mkdir which handles idempotency (MkdirAll)
	if err := h.storage.Mkdir(".uploads"); err != nil {
		h.logger.WithError(err).Error("Failed to ensure .uploads dir")
	}

	uploadsDir, err := h.storage.GetFullPath(".uploads")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage path error"})
		return
	}

	tempPath := filepath.Join(uploadsDir, uploadID+".part")

	session := &UploadSession{
		ID:           uploadID,
		OriginalName: req.Filename,
		TempPath:     tempPath,
		TotalSize:    req.TotalSize,
		CurrentSize:  0,
	}

	h.mu.Lock()
	h.sessions[uploadID] = session
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"upload_id": uploadID,
		"message":   "Upload initialized",
	})
}

// UploadChunk appends a data chunk to the upload
// POST /api/v1/vault/upload/chunk/:id
// Body: Raw binary data
func (h *BlobStorageHandler) UploadChunk(c *gin.Context) {
	uploadID := c.Param("id")

	h.mu.RLock()
	session, exists := h.sessions[uploadID]
	h.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload session not found"})
		return
	}

	session.Mutex.Lock()
	defer session.Mutex.Unlock()

	// Open file in append mode
	f, err := os.OpenFile(session.TempPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		h.logger.WithError(err).Error("Failed to open temp file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Write error"})
		return
	}
	defer f.Close()

	// Copy request body to file
	written, err := io.Copy(f, c.Request.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to write chunk")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Write failed"})
		return
	}

	session.CurrentSize += written

	c.JSON(http.StatusOK, gin.H{
		"uploaded_bytes": written,
		"total_uploaded": session.CurrentSize,
	})
}

// FinalizeUpload moves the completed file to the destination
// POST /api/v1/vault/upload/finalize/:id
// Body: { "path": "encrypted/my-folder" }
func (h *BlobStorageHandler) FinalizeUpload(c *gin.Context) {
	uploadID := c.Param("id")

	var req struct {
		Destination string `json:"path"` // Relative path, e.g. "encrypted"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Destination = "encrypted" // Default
	}

	h.mu.Lock()
	session, exists := h.sessions[uploadID]
	delete(h.sessions, uploadID) // Remove session regardless of outcome
	h.mu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload session not found"})
		return
	}

	// Validate size if needed (optional)
	if session.CurrentSize != session.TotalSize {
		h.logger.Warnf("Upload size mismatch: expected %d, got %d", session.TotalSize, session.CurrentSize)
		// We allow this for now, but log it.
	}

	// Move file using StorageService logic to handle existence/sanitization
	// We need to verify destination directory exists
	if err := h.storage.Mkdir(req.Destination); err != nil {
		h.logger.WithError(err).Error("Failed to ensure dest dir")
		// Continue anyway, maybe it exists
	}

	destDir, err := h.storage.GetFullPath(req.Destination)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path"})
		return
	}

	destPath := filepath.Join(destDir, filepath.Base(session.OriginalName))

	// Move (Rename)
	if err := os.Rename(session.TempPath, destPath); err != nil {
		h.logger.WithError(err).Error("Failed to move finalized file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Finalization failed"})
		return
	}

	h.logger.Infof("Finalized upload: %s", uploadID)

	c.JSON(http.StatusOK, gin.H{
		"status": "completed",
		"path":   filepath.Join(req.Destination, filepath.Base(session.OriginalName)),
	})
}
