package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

type metricPayload struct {
	AgentID   string  `json:"agent_id"`
	CPUUsage  float64 `json:"cpu_usage"`
	RAMUsage  float64 `json:"ram_usage"`
	DiskUsage float64 `json:"disk_usage"`
}

func main() {
	apiURL := getEnv("API_URL", "http://api:8080/api/v1/system/metrics")
	apiKey := getMonitoringToken()
	agentID := getEnv("AGENT_ID", "monitoring-agent")
	interval := getInterval()

	if apiKey == "" {
		log.Fatal("MONITORING_TOKEN ist erforderlich")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starte Monitoring-Agent. Ziel: %s, Intervall: %s", apiURL, interval)

	for {
		// FIX [BUG-GO-011]: Add retry logic (3 attempts) to handle transient failures
		for attempt := 1; attempt <= 3; attempt++ {
			if err := sendMetrics(client, apiURL, apiKey, agentID); err != nil {
				if attempt < 3 {
					log.Printf("Sende-Fehler (Versuch %d/3): %v, Wiederholung in 2s...", attempt, err)
					time.Sleep(2 * time.Second)
					continue
				}
				log.Printf("Sende-Fehler nach 3 Versuchen: %v", err)
			} else {
				break
			}
		}

		<-ticker.C
	}
}

func sendMetrics(client *http.Client, apiURL, apiKey, agentID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cpuUsage, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil || len(cpuUsage) == 0 {
		return errOrFallback("CPU", err)
	}

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return errOrFallback("RAM", err)
	}

	diskUsage, err := disk.UsageWithContext(ctx, "/")
	if err != nil {
		return errOrFallback("Disk", err)
	}

	payload := metricPayload{
		AgentID:   agentID,
		CPUUsage:  round(cpuUsage[0]),
		RAMUsage:  round(vm.UsedPercent),
		DiskUsage: round(diskUsage.UsedPercent),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Monitoring-Token", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("API antwortete mit Status %d", resp.StatusCode)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getInterval() time.Duration {
	raw := os.Getenv("INTERVAL_SECONDS")
	if raw == "" {
		return 10 * time.Second
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return 10 * time.Second
	}
	return time.Duration(sec) * time.Second
}

func round(v float64) float64 {
	return float64(int(v*100)) / 100
}

func errOrFallback(label string, err error) error {
	if err != nil {
		return err
	}
	return fmt.Errorf("%s Messung fehlgeschlagen", label)
}

func getMonitoringToken() string {
	if file := os.Getenv("MONITORING_TOKEN_FILE"); file != "" {
		content, err := os.ReadFile(file)
		if err == nil {
			return strings.TrimSpace(string(content))
		}
	}
	return os.Getenv("MONITORING_TOKEN")
}
