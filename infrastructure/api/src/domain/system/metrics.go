package system

import "time"

// SystemMetric repräsentiert einen Metrikpunkt eines Agents.
type SystemMetric struct {
	ID        string    `json:"id" db:"id"`
	AgentID   string    `json:"agent_id" db:"agent_id"`
	CPUUsage  float64   `json:"cpu_usage" db:"cpu_usage"`
	RAMUsage  float64   `json:"ram_usage" db:"ram_usage"`
	DiskUsage float64   `json:"disk_usage" db:"disk_usage"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
