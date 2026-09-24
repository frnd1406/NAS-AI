package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
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

func TestRegistryConcurrentAccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry, err := NewServiceRegistry(filepath.Join(t.TempDir(), "registry.json"), logger)
	if err != nil {
		t.Fatalf("NewServiceRegistry: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("svc-%d", i)
			if err := registry.Register(name, "http://localhost/health", []string{"core"}, nil); err != nil {
				t.Errorf("Register: %v", err)
			}
		}(i)
		go func() {
			defer wg.Done()
			_ = registry.List()
			_ = registry.FindByTag("core")
		}()
	}
	wg.Wait()

	if got := len(registry.List()); got != 10 {
		t.Errorf("Expected 10 services, got %d", got)
	}
}

func TestRegistryPersistence(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	path := filepath.Join(t.TempDir(), "registry.json")

	registry, err := NewServiceRegistry(path, logger)
	if err != nil {
		t.Fatalf("NewServiceRegistry: %v", err)
	}
	if err := registry.Register("api", "http://localhost:8080/health", nil, nil); err != nil {
		t.Fatalf("Register: %v", err)
	}

	reloaded, err := NewServiceRegistry(path, logger)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	entry, ok := reloaded.Get("api")
	if !ok {
		t.Fatal("Expected service 'api' after reload")
	}
	if entry.URL != "http://localhost:8080/health" {
		t.Errorf("Expected URL http://localhost:8080/health, got %s", entry.URL)
	}
}
