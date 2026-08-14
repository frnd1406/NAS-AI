package system

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HealthChecker is implemented by dependencies that can be probed.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// getDiskStats returns disk usage for the given path
func getDiskStats(path string) (total, used, free uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}
	total = stat.Blocks * uint64(stat.Bsize)
	free = stat.Bavail * uint64(stat.Bsize)
	used = total - free
	return
}

// countFilesAndFolders counts files and folders in path
func countFilesAndFolders(root string) (files, folders int) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			folders++
		} else {
			files++
		}
		return nil
	})
	return
}

// probeDependencies checks the health of core dependencies (db, redis) and
// returns a per-dependency status map plus an overall healthy flag. It is
// shared by the public and detailed health handlers so both agree on status.
func probeDependencies(ctx context.Context, db, redis HealthChecker, logger *logrus.Logger) (gin.H, bool) {
	dependencies := gin.H{}
	healthy := true

	if db == nil {
		logger.Error("PostgreSQL health check skipped: dependency not provided")
		dependencies["database"] = "unhealthy"
		healthy = false
	} else if err := db.HealthCheck(ctx); err != nil {
		logger.WithError(err).Error("PostgreSQL health check failed")
		dependencies["database"] = "unhealthy"
		healthy = false
	} else {
		dependencies["database"] = "ok"
	}

	if redis == nil {
		logger.Error("Redis health check skipped: dependency not provided")
		dependencies["redis"] = "unhealthy"
		healthy = false
	} else if err := redis.HealthCheck(ctx); err != nil {
		logger.WithError(err).Error("Redis health check failed")
		dependencies["redis"] = "unhealthy"
		healthy = false
	} else {
		dependencies["redis"] = "ok"
	}

	return dependencies, healthy
}

// Health godoc
// @Summary Public health check endpoint
// @Description Returns a minimal liveness status. To avoid handing recon
// @Description information to an in-network attacker, no version, service name,
// @Description dependency detail, disk usage or file counts are disclosed here.
// @Description Use the authenticated /system/health/detailed endpoint for diagnostics.
// @Tags System
// @Produce json
// @Success 200 {object} map[string]string "Service is healthy"
// @Failure 503 {object} map[string]string "Service is degraded"
// @Router /health [get]
func Health(db HealthChecker, redis HealthChecker, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		// Probe dependencies to determine liveness, but do NOT leak which one
		// is failing or any other internal detail in the public response.
		_, healthy := probeDependencies(ctx, db, redis, logger)

		if !healthy {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// HealthDetailed godoc
// @Summary Detailed health check endpoint (authenticated)
// @Description Returns full diagnostics: dependency status, service metadata,
// @Description disk usage and file/folder counts. Protected behind auth + CSRF
// @Description so this recon-sensitive information is never exposed anonymously.
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{} "Detailed health information"
// @Failure 503 {object} map[string]interface{} "Dependency unavailable"
// @Router /system/health/detailed [get]
func HealthDetailed(db HealthChecker, redis HealthChecker, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		dependencies, healthy := probeDependencies(ctx, db, redis, logger)

		// Get disk stats for /mnt/data
		diskTotal, diskUsed, diskFree := getDiskStats("/mnt/data")
		fileCount, folderCount := countFilesAndFolders("/mnt/data")

		status := gin.H{
			"status":       "ok",
			"timestamp":    time.Now().Format(time.RFC3339),
			"service":      "nas-api",
			"version":      "1.0.0-phase1",
			"dependencies": dependencies,
			"disk_total":   diskTotal,
			"disk_used":    diskUsed,
			"disk_free":    diskFree,
			"file_count":   fileCount,
			"folder_count": folderCount,
		}

		if !healthy {
			status["status"] = "degraded"
			c.JSON(http.StatusServiceUnavailable, status)
			return
		}

		c.JSON(http.StatusOK, status)
	}
}
