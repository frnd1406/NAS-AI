package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHealthEndpoint(t *testing.T) {
	// Setup test server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected /health, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "ok",
			"services": map[string]string{},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Test request
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", result["status"])
	}
}

func TestMetricsEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			t.Errorf("Expected /metrics, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# HELP orchestrator_up\norchestrator_up 1\n"))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Clear env vars to test defaults
	os.Unsetenv("REGISTRY_PATH")
	os.Unsetenv("API_URL")
	os.Unsetenv("API_ADDR")

	cfg := LoadConfig()

	if cfg.RegistryPath == "" {
		t.Error("RegistryPath should have a default value")
	}

	if cfg.APIURL == "" {
		t.Error("APIURL should have a default value")
	}

	if cfg.APIAddr == "" {
		t.Error("APIAddr should have a default value")
	}
}

func TestConfigFromEnv(t *testing.T) {
	os.Setenv("REGISTRY_PATH", "/custom/path.json")
	os.Setenv("API_URL", "http://custom:8080")
	os.Setenv("API_ADDR", ":9999")
	defer func() {
		os.Unsetenv("REGISTRY_PATH")
		os.Unsetenv("API_URL")
		os.Unsetenv("API_ADDR")
	}()

	cfg := LoadConfig()

	if cfg.RegistryPath != "/custom/path.json" {
		t.Errorf("Expected /custom/path.json, got %s", cfg.RegistryPath)
	}

	if cfg.APIURL != "http://custom:8080" {
		t.Errorf("Expected http://custom:8080, got %s", cfg.APIURL)
	}

	if cfg.APIAddr != ":9999" {
		t.Errorf("Expected :9999, got %s", cfg.APIAddr)
	}
}

func TestMetricsThreadSafety(t *testing.T) {
	m := NewMetrics()

	// Simulate concurrent access
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			m.RecordHealthCheck("test-service", true, 100*time.Millisecond)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		go func() {
			_ = m.GetPrometheusMetrics()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestRegistryJSON(t *testing.T) {
	testJSON := `{
		"services": [
			{"name": "api", "url": "http://localhost:8080/health"}
		]
	}`

	var registry Registry
	if err := json.NewDecoder(strings.NewReader(testJSON)).Decode(&registry); err != nil {
		t.Fatalf("Failed to decode registry: %v", err)
	}

	if len(registry.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(registry.Services))
	}

	if registry.Services[0].Name != "api" {
		t.Errorf("Expected service name 'api', got %s", registry.Services[0].Name)
	}
}
