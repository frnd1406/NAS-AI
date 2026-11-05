package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/services/operations"
)

type DiagnosticsHandler struct {
	service *operations.DiagnosticsService
}

func NewDiagnosticsHandler(service *operations.DiagnosticsService) *DiagnosticsHandler {
	return &DiagnosticsHandler{service: service}
}

// RunSelfTest triggers the full system diagnostics
func (h *DiagnosticsHandler) RunSelfTest(c *gin.Context) {
	report := h.service.RunFullDiagnosis(c.Request.Context())

	status := http.StatusOK
	// If the system is DOWN (critical components failing), return 503.
	// If DEGRADED (non-critical issues), potentially still 200 or 207?
	// User instruction: "200 (if Healthy) or 503 (if Down)".
	// My service returns "DEGRADED" if e.g. delete failed but write/read worked.
	if report.OverallStatus == "DOWN" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, report)
}
