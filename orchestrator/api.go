package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// APIServer provides HTTP API for orchestrator
type APIServer struct {
	orch     *Orchestrator
	registry *ServiceRegistry
	logger   *slog.Logger
}

func NewAPIServer(orch *Orchestrator, registry *ServiceRegistry, logger *slog.Logger) *APIServer {
	return &APIServer{
		orch:     orch,
		registry: registry,
		logger:   logger,
	}
}

// ServiceStatusResponse for /api/services endpoint
type ServiceStatusResponse struct {
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	Healthy         bool      `json:"healthy"`
	LastCheck       time.Time `json:"last_check"`
	LastHealthy     time.Time `json:"last_healthy"`
	ConsecutiveFails int       `json:"consecutive_fails"`
	TotalChecks     int       `json:"total_checks"`
	TotalFailures   int       `json:"total_failures"`
	Uptime          float64   `json:"uptime_percent"`
}

// HandleHealth returns orchestrator health
func (api *APIServer) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0",
	}

	json.NewEncoder(w).Encode(resp)
}

// HandleServices returns all service statuses
func (api *APIServer) HandleServices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	services := make([]ServiceStatusResponse, 0)
	for _, s := range api.orch.GetServiceStatus() {
		services = append(services, ServiceStatusResponse{
			Name:            s.Name,
			URL:             s.URL,
			Healthy:         s.Healthy,
			LastCheck:       s.LastCheck,
			LastHealthy:     s.LastHealthy,
			ConsecutiveFails: s.ConsecutiveFails,
			TotalChecks:     s.TotalChecks,
			TotalFailures:   s.TotalFailures,
			Uptime:          s.Uptime,
		})
	}

	json.NewEncoder(w).Encode(services)
}

// HandleRegistry returns service registry
func (api *APIServer) HandleRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.registry.List())
}

// Start starts the HTTP API server
func (api *APIServer) Start(addr string) error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/health", api.HandleHealth)
	mux.HandleFunc("/api/services", api.HandleServices)
	mux.HandleFunc("/api/registry", api.HandleRegistry)

	// Prometheus metrics
	metrics := NewPrometheusMetrics(api.orch)
	mux.Handle("/metrics", metrics)

	api.logger.Info("API server starting",
		slog.String("addr", addr),
	)

	return http.ListenAndServe(addr, mux)
}
