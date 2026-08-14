package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// ServiceRegistryEntry represents a service in the registry
type ServiceRegistryEntry struct {
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	RegisteredAt time.Time         `json:"registered_at"`
	Tags         []string          `json:"tags,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ServiceRegistry manages service discovery
type ServiceRegistry struct {
	filepath string
	services map[string]*ServiceRegistryEntry
	mu       sync.RWMutex
	logger   *slog.Logger
}

func NewServiceRegistry(filepath string, logger *slog.Logger) (*ServiceRegistry, error) {
	sr := &ServiceRegistry{
		filepath: filepath,
		services: make(map[string]*ServiceRegistryEntry),
		logger:   logger,
	}

	// Load existing registry
	if err := sr.Load(); err != nil {
		// If file doesn't exist, start fresh
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return sr, nil
}

// Register adds or updates a service in the registry
func (sr *ServiceRegistry) Register(name, url string, tags []string, metadata map[string]string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	entry := &ServiceRegistryEntry{
		Name:         name,
		URL:          url,
		RegisteredAt: time.Now(),
		Tags:         tags,
		Metadata:     metadata,
	}

	sr.services[name] = entry

	if err := sr.save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	sr.logger.Info("Service registered",
		slog.String("service", name),
		slog.String("url", url),
	)

	return nil
}

// Deregister removes a service from the registry
func (sr *ServiceRegistry) Deregister(name string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	delete(sr.services, name)

	if err := sr.save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	sr.logger.Info("Service deregistered",
		slog.String("service", name),
	)

	return nil
}

// Get retrieves a service entry by name
func (sr *ServiceRegistry) Get(name string) (*ServiceRegistryEntry, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	entry, exists := sr.services[name]
	return entry, exists
}

// List returns all registered services
func (sr *ServiceRegistry) List() []*ServiceRegistryEntry {
	entries := make([]*ServiceRegistryEntry, 0, len(sr.services))
	for _, entry := range sr.services {
		// Deep copy the entry to prevent race conditions
		newEntry := &ServiceRegistryEntry{
			Name:         entry.Name,
			URL:          entry.URL,
			RegisteredAt: entry.RegisteredAt,
		}
		if entry.Tags != nil {
			newEntry.Tags = make([]string, len(entry.Tags))
			copy(newEntry.Tags, entry.Tags)
		}
		if entry.Metadata != nil {
			newEntry.Metadata = make(map[string]string)
			for k, v := range entry.Metadata {
				newEntry.Metadata[k] = v
			}
		}
		entries = append(entries, newEntry)
	}

	return entries
}

// FindByTag returns services with the specified tag
func (sr *ServiceRegistry) FindByTag(tag string) []*ServiceRegistryEntry {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	var entries []*ServiceRegistryEntry
	for _, entry := range sr.services {
		for _, t := range entry.Tags {
			if t == tag {
				// Deep copy the entry to prevent race conditions
				newEntry := &ServiceRegistryEntry{
					Name:         entry.Name,
					URL:          entry.URL,
					RegisteredAt: entry.RegisteredAt,
				}
				if entry.Tags != nil {
					newEntry.Tags = make([]string, len(entry.Tags))
					copy(newEntry.Tags, entry.Tags)
				}
				if entry.Metadata != nil {
					newEntry.Metadata = make(map[string]string)
					for k, v := range entry.Metadata {
						newEntry.Metadata[k] = v
					}
				}
				entries = append(entries, newEntry)
				break
			}
		}
	}

	return entries
}

// Load reads the registry from disk
func (sr *ServiceRegistry) Load() error {
	data, err := os.ReadFile(sr.filepath)
	if err != nil {
		return err
	}

	var entries []*ServiceRegistryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	sr.services = make(map[string]*ServiceRegistryEntry)
	for _, entry := range entries {
		sr.services[entry.Name] = entry
	}

	sr.logger.Info("Registry loaded",
		slog.Int("services", len(sr.services)),
	)

	return nil
}

// save writes the registry to disk using atomic write pattern
// This prevents data corruption on crash/power loss by:
// 1. Writing to a temporary file
// 2. Syncing to disk (fsync)
// 3. Atomically renaming temp -> target (POSIX guarantees atomicity)
func (sr *ServiceRegistry) save() error {
	entries := make([]*ServiceRegistryEntry, 0, len(sr.services))
	for _, entry := range sr.services {
		entries = append(entries, entry)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Atomic Write Pattern: temp file -> fsync -> rename
	tempPath := sr.filepath + ".tmp"

	// Step 1: Write to temp file
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp registry: %w", err)
	}

	// Step 2: Fsync to ensure data is on disk before rename
	tempFile, err := os.Open(tempPath)
	if err != nil {
		os.Remove(tempPath) // Cleanup on error
		return fmt.Errorf("failed to open temp file for sync: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	tempFile.Close()

	// Step 3: Atomic rename (guaranteed by POSIX)
	// Either old file exists or new file exists, never corrupted state
	if err := os.Rename(tempPath, sr.filepath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp to registry: %w", err)
	}

	return nil
}
