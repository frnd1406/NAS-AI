package system

import "time"

// MonitoringSample stores a monitoring datapoint from the MonitoringAgent.
type MonitoringSample struct {
	ID         string    `json:"id" db:"id"`
	Source     string    `json:"source" db:"source"`
	CPUPercent float64   `json:"cpu_percent" db:"cpu_percent"`
	RAMPercent float64   `json:"ram_percent" db:"ram_percent"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
