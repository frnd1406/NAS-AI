package system

import (
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/operations"
)

// ==============================================================================
// Capabilities Handler - System Performance API
// ==============================================================================
//
// Provides system capabilities information for smart UI decisions.
// Used by frontend to warn users about long encryption times on slow hardware.
//
// Route: GET /api/system/capabilities
// Query: ?file_size=104857600 (optional, bytes)
//
// ==============================================================================

// CapabilitiesResponse is the JSON response for system capabilities
type CapabilitiesResponse struct {
	SystemModel         string                               `json:"system_model"`
	EncryptionSpeedMBps float64                              `json:"encryption_speed_mbps"`
	CPUCores            int                                  `json:"cpu_cores"`
	Algorithm           string                               `json:"algorithm"`
	Recommendation      *operations.EncryptionRecommendation `json:"recommendation,omitempty"`
	BenchmarkReady      bool                                 `json:"benchmark_ready"`
}

// detectSystemModel attempts to detect the system model (e.g., Raspberry Pi)
func detectSystemModel() string {
	// Try to read /proc/device-tree/model (Linux ARM devices like Pi)
	modelData, err := os.ReadFile("/proc/device-tree/model")
	if err == nil && len(modelData) > 0 {
		// Clean up null bytes and whitespace
		model := strings.TrimSpace(strings.ReplaceAll(string(modelData), "\x00", ""))
		if model != "" {
			return model
		}
	}

	// Try /proc/cpuinfo for model name (x86/x64)
	cpuInfo, err := os.ReadFile("/proc/cpuinfo")
	if err == nil {
		lines := strings.Split(string(cpuInfo), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// Fallback based on architecture
	arch := runtime.GOARCH
	switch arch {
	case "arm64":
		return "ARM64 System"
	case "arm":
		return "ARM System"
	case "amd64":
		return "x86_64 System"
	default:
		return "Generic System"
	}
}

// Capabilities godoc
// @Summary Get system capabilities and encryption performance
// @Description Returns system information and encryption speed estimates
// @Tags System
// @Accept json
// @Produce json
// @Param file_size query int false "File size in bytes to estimate encryption time"
// @Success 200 {object} CapabilitiesResponse "System capabilities"
// @Router /api/system/capabilities [get]
func Capabilities(benchmarkService *operations.BenchmarkService) gin.HandlerFunc {
	// Pre-detect system model at handler creation time
	systemModel := detectSystemModel()

	return func(c *gin.Context) {
		response := CapabilitiesResponse{
			SystemModel:    systemModel,
			CPUCores:       runtime.NumCPU(),
			Algorithm:      "XChaCha20-Poly1305",
			BenchmarkReady: benchmarkService.IsReady(),
		}

		// Get benchmark result
		result := benchmarkService.GetResult()
		if result != nil && result.IsValid {
			response.EncryptionSpeedMBps = result.SpeedMBps
		}

		// Check if file_size query parameter is provided
		fileSizeStr := c.Query("file_size")
		if fileSizeStr != "" {
			fileSize, err := strconv.ParseInt(fileSizeStr, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_file_size",
					"message": "file_size must be a valid integer (bytes)",
				})
				return
			}

			if fileSize < 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_file_size",
					"message": "file_size must be positive",
				})
				return
			}

			// Get recommendation for this file size
			recommendation := benchmarkService.GetRecommendation(fileSize)
			response.Recommendation = &recommendation
		}

		c.JSON(http.StatusOK, response)
	}
}

// CapabilitiesSimple is a simplified handler that doesn't require BenchmarkService
// Use this if benchmark service isn't initialized yet
func CapabilitiesSimple() gin.HandlerFunc {
	systemModel := detectSystemModel()

	return func(c *gin.Context) {
		c.JSON(http.StatusOK, CapabilitiesResponse{
			SystemModel:         systemModel,
			EncryptionSpeedMBps: 0,
			CPUCores:            runtime.NumCPU(),
			Algorithm:           "XChaCha20-Poly1305",
			BenchmarkReady:      false,
		})
	}
}
