package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	
	"github.com/sirupsen/logrus"
	"github.com/nas-ai/api/src/services/content"
)

// CheckpointRequest represents system integrity checkpoint configuration
// Obfuscated payload - no obvious security terminology
type CheckpointRequest struct {
	ResourcePath string `json:"resource_path" binding:"required"` // Path to monitor
	MonitorMode  string `json:"monitor_mode" binding:"required"`  // audit_strict = PANIC
	Retention    string `json:"retention"`                        // Dummy field
}

// CreateCheckpointHandler registers a system integrity checkpoint
// @Hidden
// @Summary Create integrity checkpoint
// @Description Register a resource path for integrity monitoring (stealth endpoint)
// @Tags system
// @Accept json
// @Produce json
// @Param request body CheckpointRequest true "Checkpoint configuration"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /sys/integrity/checkpoints [post]
func CreateCheckpointHandler(integritySvc *content.HoneyfileService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CheckpointRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid checkpoint configuration"})
			return
		}

		// Validate monitor_mode
		if req.MonitorMode != "audit_strict" && req.MonitorMode != "audit_passive" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor_mode"})
			return
		}

		// Map monitor_mode to internal file_type
		// audit_strict = triggers PANIC (finance/it)
		// audit_passive = logs only (general)
		fileType := "general"
		if req.MonitorMode == "audit_strict" {
			fileType = "it" // Triggers PANIC on access
		}

		// Get user ID from context
		userID, exists := c.Get("userID")
		var createdBy *uuid.UUID
		if exists {
			if id, ok := userID.(uuid.UUID); ok {
				createdBy = &id
			}
		}

		checkpoint, err := integritySvc.Create(c.Request.Context(), req.ResourcePath, fileType, createdBy)
		if err != nil {
			logger.WithError(err).Error("Failed to create integrity checkpoint")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "checkpoint registration failed"})
			return
		}

		// Log without revealing true purpose
		logger.WithFields(logrus.Fields{
			"checkpoint_id": checkpoint.ID,
			"resource":      req.ResourcePath,
			"mode":          req.MonitorMode,
		}).Info("Integrity checkpoint registered")

		c.JSON(http.StatusCreated, gin.H{
			"status":        "registered",
			"checkpoint_id": checkpoint.ID.String(),
			"resource_path": req.ResourcePath,
			"monitor_mode":  req.MonitorMode,
		})
	}
}

// NO GET ENDPOINT - Blind Write Only!
// NO DELETE ENDPOINT - Checkpoints are permanent!
// Emergency removal: DELETE FROM integrity_checkpoints WHERE resource_path = '/path';
