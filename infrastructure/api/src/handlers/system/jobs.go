package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	
	"github.com/sirupsen/logrus"
	"github.com/nas-ai/api/src/services/operations"
)

// GetJobStatusHandler returns the status and result of an AI job
// @Summary Get AI Job Status
// @Description Returns the current status and result of an AI job
// @Tags AI
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} services.AIJobResult "Job status and result"
// @Failure 404 {object} map[string]interface{} "Job not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/jobs/{id} [get]
func GetJobStatusHandler(jobService *operations.JobService, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")
		requestID := c.GetString("request_id")

		if jobID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "invalid_request",
					"message":    "Job ID is required",
					"request_id": requestID,
				},
			})
			return
		}

		result, err := jobService.GetJobResult(c.Request.Context(), jobID)
		if err != nil {
			logger.WithError(err).WithField("job_id", jobID).Warn("Job not found")
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":       "job_not_found",
					"message":    "Job not found or expired",
					"request_id": requestID,
				},
			})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
