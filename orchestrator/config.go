package main

import (
	"os"
	"time"
)

// Config holds all orchestrator configuration
type Config struct {
	RegistryPath   string
	APIURL         string
	APIAddr        string
	CheckInterval  time.Duration
	StatusInterval time.Duration
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	return &Config{
		RegistryPath:   getEnv("REGISTRY_PATH", "./data/registry.json"),
		APIURL:         getEnv("API_URL", "http://localhost:8080"),
		APIAddr:        getEnv("API_ADDR", ":9000"),
		CheckInterval:  30 * time.Second,
		StatusInterval: 5 * time.Minute,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
