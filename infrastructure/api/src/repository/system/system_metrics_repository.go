package system_repo

import (
		"github.com/nas-ai/api/src/domain/system"
"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/sirupsen/logrus"
)

// SystemMetricsRepository speichert Agent-Metriken via sqlx.
type SystemMetricsRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

func NewSystemMetricsRepository(db *sqlx.DB, logger *logrus.Logger) *SystemMetricsRepository {
	return &SystemMetricsRepository{
		db:     db,
		logger: logger,
	}
}

func (r *SystemMetricsRepository) Insert(ctx context.Context, metric *system.SystemMetric) error {
	query := `
		INSERT INTO system_metrics (agent_id, cpu_usage, ram_usage, disk_usage)
		VALUES (:agent_id, :cpu_usage, :ram_usage, :disk_usage)
		RETURNING id, created_at
	`

	rows, err := r.db.NamedQueryContext(ctx, query, map[string]interface{}{
		"agent_id":   metric.AgentID,
		"cpu_usage":  metric.CPUUsage,
		"ram_usage":  metric.RAMUsage,
		"disk_usage": metric.DiskUsage,
	})
	if err != nil {
		r.logger.WithError(err).Error("failed to insert system metric")
		return fmt.Errorf("failed to insert system metric: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&metric.ID, &metric.CreatedAt); err != nil {
			return fmt.Errorf("failed to scan inserted system metric: %w", err)
		}
	}

	return nil
}

func (r *SystemMetricsRepository) List(ctx context.Context, limit int) ([]system.SystemMetric, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	var items []system.SystemMetric
	query := `
		SELECT id, agent_id, cpu_usage, ram_usage, disk_usage, created_at
		FROM system_metrics
		ORDER BY created_at DESC
		LIMIT $1
	`
	if err := r.db.SelectContext(ctx, &items, query, limit); err != nil {
		r.logger.WithError(err).Error("failed to list system metrics")
		return nil, fmt.Errorf("failed to list system metrics: %w", err)
	}
	return items, nil
}
