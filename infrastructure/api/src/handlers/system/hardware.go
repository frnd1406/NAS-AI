package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/operations"
	"github.com/sirupsen/logrus"
)

// HardwareHandler handles hardware metric requests
type HardwareHandler struct {
	hardwareService *operations.HardwareService
	logger          *logrus.Logger
}

// NewHardwareHandler creates a new hardware handler
func NewHardwareHandler(hardwareService *operations.HardwareService, logger *logrus.Logger) *HardwareHandler {
	return &HardwareHandler{
		hardwareService: hardwareService,
		logger:          logger,
	}
}

// GetStorageInfoHandler returns physical disk usage
// @Summary Get storage info
// @Description Get current usage of physical disks
// @Tags System
// @Produce json
// @Success 200 {array} operations.DiskInfo
// @Router /system/hardware/storage [get]
func (h *HardwareHandler) GetStorageInfoHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		info, err := h.hardwareService.GetStorageInfo()
		if err != nil {
			h.logger.WithError(err).Error("Failed to get storage info")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve storage info"})
			return
		}

		c.JSON(http.StatusOK, info)
	}
}

// GetUPSInfoHandler returns the current UPS (mains power / battery) status.
// @Summary Get UPS status
// @Description Get the live power state of the UPS via NUT (online / on battery / low battery)
// @Tags System
// @Produce json
// @Success 200 {object} operations.UPSInfo
// @Router /system/hardware/ups [get]
func (h *HardwareHandler) GetUPSInfoHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// GetUPSInfo never returns nil; an unreachable UPS yields a graceful
		// {"available": false} snapshot rather than a 500.
		c.JSON(http.StatusOK, h.hardwareService.GetUPSInfo())
	}
}

// GetNetworkInfoHandler returns network interface statistics
// @Summary Get network info
// @Description Get current network interface stats
// @Tags System
// @Produce json
// @Success 200 {array} operations.NetworkInterfaceInfo
// @Router /system/hardware/network [get]
func (h *HardwareHandler) GetNetworkInfoHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		info, err := h.hardwareService.GetNetworkInfo()
		if err != nil {
			h.logger.WithError(err).Error("Failed to get network info")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve network info"})
			return
		}

		c.JSON(http.StatusOK, info)
	}
}
