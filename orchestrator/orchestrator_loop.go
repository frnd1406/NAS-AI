package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// HealthResponse represents the API health check response
type HealthResponse struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// ServiceStatus tracks service health over time
type ServiceStatus struct {
	mu               sync.RWMutex // CONCURRENCY FIX: Protects all fields from concurrent access
	Name             string
	URL              string
	Healthy          bool
	LastCheck        time.Time
	LastHealthy      time.Time
	ConsecutiveFails int
	TotalChecks      int
	TotalFailures    int
	Uptime           float64
}

// Snapshot returns a thread-safe copy of the ServiceStatus
func (s *ServiceStatus) Snapshot() ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return ServiceStatus{
		Name:             s.Name,
		URL:              s.URL,
		Healthy:          s.Healthy,
		LastCheck:        s.LastCheck,
		LastHealthy:      s.LastHealthy,
		ConsecutiveFails: s.ConsecutiveFails,
		TotalChecks:      s.TotalChecks,
		TotalFailures:    s.TotalFailures,
		Uptime:           s.Uptime,
	}
}

// Orchestrator manages health checks for all services
type Orchestrator struct {
	mu       sync.RWMutex // CONCURRENCY FIX: Protects services map from concurrent access
	services map[string]*ServiceStatus
	logger   *slog.Logger
	client   *http.Client
}

func NewOrchestrator(logger *slog.Logger) *Orchestrator {
	return &Orchestrator{
		services: make(map[string]*ServiceStatus),
		logger:   logger,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// RegisterService adds a service to monitor
func (o *Orchestrator) RegisterService(name, url string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.services[name] = &ServiceStatus{
		Name:        name,
		URL:         url,
		Healthy:     false,
		LastCheck:   time.Time{},
		LastHealthy: time.Time{},
	}
	o.logger.Info("Service registered",
		slog.String("service", name),
		slog.String("url", url),
	)
}

// CheckHealth performs health check on a service
func (o *Orchestrator) CheckHealth(ctx context.Context, service *ServiceStatus) error {
	req, err := http.NewRequestWithContext(ctx, "GET", service.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if health.Status != "ok" {
		return fmt.Errorf("service reports unhealthy status: %s", health.Status)
	}

	return nil
}

// HealthCheckLoop runs continuous health checks
func (o *Orchestrator) HealthCheckLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Read services count with lock
	o.mu.RLock()
	serviceCount := len(o.services)
	o.mu.RUnlock()

	o.logger.Info("Health check loop started",
		slog.Duration("interval", interval),
		slog.Int("services", serviceCount),
	)

	// Initial check
	o.checkAllServices(ctx)

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("Health check loop stopped")
			return
		case <-ticker.C:
			o.checkAllServices(ctx)
		}
	}
}

func (o *Orchestrator) checkAllServices(ctx context.Context) {
	// CONCURRENCY FIX: Create a copy of services slice to minimize lock time
	// and prevent "concurrent map iteration and map write" panic
	o.mu.RLock()
	servicesCopy := make([]*ServiceStatus, 0, len(o.services))
	for _, service := range o.services {
		servicesCopy = append(servicesCopy, service)
	}
	o.mu.RUnlock()

	// PERFORMANCE FIX: Check all services in parallel using WaitGroup
	// This reduces worst-case check time from N*5s (sequential) to 5s (parallel)
	var wg sync.WaitGroup
	for _, service := range servicesCopy {
		wg.Add(1)
		go func(s *ServiceStatus) {
			defer wg.Done()
			o.checkService(ctx, s)
		}(service)
	}
	wg.Wait()

	o.logSummary()
}

func (o *Orchestrator) checkService(ctx context.Context, service *ServiceStatus) {
	// Perform HTTP request WITHOUT lock to avoid blocking readers during network call
	err := o.CheckHealth(ctx, service)

	// CONCURRENCY FIX: Lock for writing results to prevent race conditions
	service.mu.Lock()
	defer service.mu.Unlock()

	service.TotalChecks++
	service.LastCheck = time.Now()

	if err != nil {
		service.Healthy = false
		service.ConsecutiveFails++
		service.TotalFailures++

		o.logger.Error("Health check failed",
			slog.String("service", service.Name),
			slog.String("url", service.URL),
			slog.String("error", err.Error()),
			slog.Int("consecutive_fails", service.ConsecutiveFails),
		)
	} else {
		wasUnhealthy := !service.Healthy
		service.Healthy = true
		service.LastHealthy = time.Now()
		service.ConsecutiveFails = 0

		if wasUnhealthy {
			o.logger.Info("Service recovered",
				slog.String("service", service.Name),
				slog.String("url", service.URL),
			)
		} else {
			o.logger.Debug("Health check passed",
				slog.String("service", service.Name),
			)
		}
	}

	// Calculate uptime percentage
	if service.TotalChecks > 0 {
		successChecks := service.TotalChecks - service.TotalFailures
		service.Uptime = (float64(successChecks) / float64(service.TotalChecks)) * 100
	}
}

func (o *Orchestrator) logSummary() {
	o.mu.RLock()
	defer o.mu.RUnlock()

	healthy := 0
	total := len(o.services)

	for _, service := range o.services {
		// CONCURRENCY FIX: Read lock for accessing service.Healthy
		service.mu.RLock()
		isHealthy := service.Healthy
		service.mu.RUnlock()

		if isHealthy {
			healthy++
		}
	}

	o.logger.Info("Health check summary",
		slog.Int("healthy", healthy),
		slog.Int("total", total),
		slog.Int("unhealthy", total-healthy),
	)
}

// GetServiceStatus returns current status of all services as deep copies
func (o *Orchestrator) GetServiceStatus() map[string]ServiceStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// CONCURRENCY FIX: Return deep copies (values not pointers) to prevent concurrent modification
	statusCopy := make(map[string]ServiceStatus, len(o.services))
	for k, v := range o.services {
		statusCopy[k] = v.Snapshot()
	}
	return statusCopy
}

// PrintStatus prints current status to stdout
func (o *Orchestrator) PrintStatus() {
	// CONCURRENCY FIX: Copy services map to avoid concurrent iteration
	o.mu.RLock()
	servicesCopy := make([]*ServiceStatus, 0, len(o.services))
	for _, service := range o.services {
		servicesCopy = append(servicesCopy, service)
	}
	o.mu.RUnlock()

	fmt.Println("\n=== Orchestrator Status ===")
	fmt.Printf("Time: %s\n\n", time.Now().Format(time.RFC3339))

	for _, service := range servicesCopy {
		// CONCURRENCY FIX: Read lock for accessing all service fields
		service.mu.RLock()
		status := "UNHEALTHY"
		if service.Healthy {
			status = "HEALTHY"
		}

		fmt.Printf("Service: %s\n", service.Name)
		fmt.Printf("  URL:              %s\n", service.URL)
		fmt.Printf("  Status:           %s\n", status)
		fmt.Printf("  Last Check:       %s\n", service.LastCheck.Format(time.RFC3339))
		fmt.Printf("  Last Healthy:     %s\n", service.LastHealthy.Format(time.RFC3339))
		fmt.Printf("  Consecutive Fails: %d\n", service.ConsecutiveFails)
		fmt.Printf("  Total Checks:     %d\n", service.TotalChecks)
		fmt.Printf("  Total Failures:   %d\n", service.TotalFailures)
		fmt.Printf("  Uptime:           %.2f%%\n\n", service.Uptime)
		service.mu.RUnlock()
	}
}

func main() {
	// Load configuration
	cfg := LoadConfig()

	// Setup logger
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	logger.Info("Orchestrator starting",
		slog.String("version", "1.0.0"),
	)

	// Create orchestrator
	orch := NewOrchestrator(logger)

	// Create service registry
	os.MkdirAll("./data", 0755)

	registry, err := NewServiceRegistry(cfg.RegistryPath, logger)
	if err != nil {
		logger.Error("Failed to create registry", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Register default service
	registry.Register("nas-api", cfg.APIURL+"/health", []string{"core", "api"}, map[string]string{
		"type":     "backend",
		"language": "go",
	})

	// Load all services from registry into orchestrator
	for _, entry := range registry.List() {
		orch.RegisterService(entry.Name, entry.URL)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP API server
	apiServer := NewAPIServer(orch, registry, logger)
	go func() {
		if err := apiServer.Start(cfg.APIAddr); err != nil {
			logger.Error("API server failed", slog.String("error", err.Error()))
		}
	}()

	// Start health check loop in goroutine
	go orch.HealthCheckLoop(ctx, cfg.CheckInterval)

	// Status printer goroutine
	go func() {
		statusTicker := time.NewTicker(cfg.StatusInterval)
		defer statusTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-statusTicker.C:
				orch.PrintStatus()
			}
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Info("Received shutdown signal",
		slog.String("signal", sig.String()),
	)

	// Cancel context to stop all goroutines
	cancel()

	// Print final status
	orch.PrintStatus()

	logger.Info("Orchestrator stopped")
}
