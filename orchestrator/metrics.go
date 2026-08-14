package main

import (
	"fmt"
	"net/http"
	"strings"
)

// PrometheusMetrics exposes orchestrator metrics in Prometheus format
type PrometheusMetrics struct {
	orch *Orchestrator
}

func NewPrometheusMetrics(orch *Orchestrator) *PrometheusMetrics {
	return &PrometheusMetrics{orch: orch}
}

// ServeHTTP implements http.Handler for /metrics endpoint
func (pm *PrometheusMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	var sb strings.Builder

    // HOLEN DER DATEN THREAD-SAFE
    // Wir nutzen die existierende Methode GetServiceStatus, die eine sichere Kopie liefert.
    services := pm.orch.GetServiceStatus()

	// Health status metrics
	sb.WriteString("# HELP orchestrator_service_healthy Whether service is healthy (1) or not (0)\n")
	sb.WriteString("# TYPE orchestrator_service_healthy gauge\n")
	for _, service := range services {
		healthyValue := 0
		if service.Healthy {
			healthyValue = 1
		}
		sb.WriteString(fmt.Sprintf("orchestrator_service_healthy{service=\"%s\",url=\"%s\"} %d\n",
			service.Name, service.URL, healthyValue))
	}

	// Uptime percentage
	sb.WriteString("# HELP orchestrator_service_uptime_percent Service uptime percentage\n")
	sb.WriteString("# TYPE orchestrator_service_uptime_percent gauge\n")
	for _, service := range services {
		sb.WriteString(fmt.Sprintf("orchestrator_service_uptime_percent{service=\"%s\"} %.2f\n",
			service.Name, service.Uptime))
	}

	// Total checks
	sb.WriteString("# HELP orchestrator_service_checks_total Total number of health checks performed\n")
	sb.WriteString("# TYPE orchestrator_service_checks_total counter\n")
	for _, service := range services {
		sb.WriteString(fmt.Sprintf("orchestrator_service_checks_total{service=\"%s\"} %d\n",
			service.Name, service.TotalChecks))
	}

	// Total failures
	sb.WriteString("# HELP orchestrator_service_failures_total Total number of failed health checks\n")
	sb.WriteString("# TYPE orchestrator_service_failures_total counter\n")
	for _, service := range services {
		sb.WriteString(fmt.Sprintf("orchestrator_service_failures_total{service=\"%s\"} %d\n",
			service.Name, service.TotalFailures))
	}

	// Consecutive failures
	sb.WriteString("# HELP orchestrator_service_consecutive_failures Current consecutive failures\n")
	sb.WriteString("# TYPE orchestrator_service_consecutive_failures gauge\n")
	for _, service := range services {
		sb.WriteString(fmt.Sprintf("orchestrator_service_consecutive_failures{service=\"%s\"} %d\n",
			service.Name, service.ConsecutiveFails))
	}

	// Last check timestamp
	sb.WriteString("# HELP orchestrator_service_last_check_timestamp Unix timestamp of last health check\n")
	sb.WriteString("# TYPE orchestrator_service_last_check_timestamp gauge\n")
	for _, service := range services {
		sb.WriteString(fmt.Sprintf("orchestrator_service_last_check_timestamp{service=\"%s\"} %d\n",
			service.Name, service.LastCheck.Unix()))
	}

	// Orchestrator info
	sb.WriteString("# HELP orchestrator_info Orchestrator version info\n")
	sb.WriteString("# TYPE orchestrator_info gauge\n")
	sb.WriteString("orchestrator_info{version=\"1.0.0\"} 1\n")

	w.Write([]byte(sb.String()))
}