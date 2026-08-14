package system

import (
	"crypto/subtle"
	"github.com/nas-ai/api/src/domain/system"
	"github.com/nas-ai/api/src/repository/system"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/sirupsen/logrus"
)

type SystemMetricsRequest struct {
	AgentID   string  `json:"agent_id" binding:"required"`
	CPUUsage  float64 `json:"cpu_usage" binding:"required"`
	RAMUsage  float64 `json:"ram_usage" binding:"required"`
	DiskUsage float64 `json:"disk_usage" binding:"required"`
}

// SystemMetricsHandler nimmt Metriken entgegen und schützt per API-Key-Header.
func SystemMetricsHandler(repo *system_repo.SystemMetricsRepository, apiKey string, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		token := c.GetHeader("X-Monitoring-Token")
		if apiKey == "" || subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			logger.WithField("request_id", requestID).Warn("unauthorized system metrics call")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "unauthorized",
					"message":    "invalid or missing monitoring token",
					"request_id": requestID,
				},
			})
			return
		}

		var req SystemMetricsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Warn("invalid system metrics payload")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "invalid payload",
					"request_id": requestID,
				},
			})
			return
		}

		metric := &system.SystemMetric{
			AgentID:   req.AgentID,
			CPUUsage:  req.CPUUsage,
			RAMUsage:  req.RAMUsage,
			DiskUsage: req.DiskUsage,
		}

		if err := repo.Insert(c.Request.Context(), metric); err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("failed to store system metrics")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "failed to store metrics",
					"request_id": requestID,
				},
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":     "ok",
			"request_id": requestID,
		})
	}
}

// SystemMetricsListHandler liefert die neuesten Metriken (öffentlich; read-only).
func SystemMetricsListHandler(repo *system_repo.SystemMetricsRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("request_id")

		limit := 10
		if raw := c.Query("limit"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil {
				limit = n
			}
		}

		items, err := repo.List(c.Request.Context(), limit)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("failed to list system metrics")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":       "internal_error",
					"message":    "failed to load metrics",
					"request_id": requestID,
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items": items,
		})
	}
}

// SystemMetricsLiveHandler returns real-time system stats (CPU, RAM, Disk) using gopsutil.
// Used for the Admin Dashboard "Health Card".
func SystemMetricsLiveHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		// 1. CPU
		cpuPercent, err := cpu.Percent(0, false)
		if err != nil {
			logger.WithError(err).Warn("Failed to get CPU stats")
		}
		cpuVal := 0.0
		if len(cpuPercent) > 0 {
			cpuVal = cpuPercent[0]
		}

		// 2. RAM
		vm, err := mem.VirtualMemory()
		if err != nil {
			logger.WithError(err).Warn("Failed to get RAM stats")
		}
		ramVal := 0.0
		ramTotal := uint64(0)
		if vm != nil {
			ramVal = vm.UsedPercent
			ramTotal = vm.Total
		}

		// 3. Disk (Root)
		diskStat, err := disk.Usage("/")
		if err != nil {
			logger.WithError(err).Warn("Failed to get Disk stats")
		}
		diskVal := 0.0
		diskTotal := uint64(0)
		if diskStat != nil {
			diskVal = diskStat.UsedPercent
			diskTotal = diskStat.Total
		}

		// 4. Get Local IPs for fallback
		localIPs := getLocalIPs()

		c.JSON(http.StatusOK, gin.H{
			"cpu_percent":  math.Round(cpuVal*100) / 100,
			"ram_percent":  math.Round(ramVal*100) / 100,
			"disk_percent": math.Round(diskVal*100) / 100,
			"ram_total":    ramTotal,
			"disk_total":   diskTotal,
			"timestamp":    time.Now(),
			"local_ips":    localIPs,
		})
	}
}

// getLocalIPs returns a list of local IPv4 addresses for fallback connectivity.
// Priority: 1) LOCAL_SERVER_IP environment variable, 2) Non-Docker network IPs
func getLocalIPs() []string {
	var ips []string

	// Check for explicit LOCAL_SERVER_IP first (set in docker-compose)
	if envIP := os.Getenv("LOCAL_SERVER_IP"); envIP != "" {
		return []string{envIP}
	}

	// Auto-detect: get all non-loopback IPv4 addresses
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipStr := ipnet.IP.String()
				// Exclude Docker bridge networks (172.16.0.0/12)
				if !isDockerNetwork(ipStr) {
					ips = append(ips, ipStr)
				}
			}
		}
	}
	return ips
}

// isDockerNetwork checks if an IP is in the Docker default bridge range (172.16.0.0/12)
func isDockerNetwork(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	// Docker typically uses 172.16.0.0/12 (172.16.x.x - 172.31.x.x) and 172.17.x.x
	firstOctet := parsed.To4()[0]
	secondOctet := parsed.To4()[1]
	return firstOctet == 172 && secondOctet >= 16 && secondOctet <= 31
}
