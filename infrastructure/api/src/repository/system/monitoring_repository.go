package system_repo

import (
		"github.com/nas-ai/api/src/domain/system"
"context"
	"fmt"

	"github.com/nas-ai/api/src/database"

	"github.com/sirupsen/logrus"
)

// MonitoringRepository handles persistence for monitoring samples.
type MonitoringRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

func NewMonitoringRepository(db *database.DB, logger *logrus.Logger) *MonitoringRepository {
	return &MonitoringRepository{db: db, logger: logger}
}

// InsertSample stores a monitoring sample.
func (r *MonitoringRepository) InsertSample(ctx context.Context, sample *system.MonitoringSample) error {
	query := `
		INSERT INTO monitoring_samples (source, cpu_percent, ram_percent)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	if err := r.db.QueryRowContext(ctx, query, sample.Source, sample.CPUPercent, sample.RAMPercent).
		Scan(&sample.ID, &sample.CreatedAt); err != nil {
		r.logger.WithError(err).Error("failed to insert monitoring sample")
		return fmt.Errorf("failed to insert monitoring sample: %w", err)
	}

	return nil
}

// ListRecent returns the latest N samples.
func (r *MonitoringRepository) ListRecent(ctx context.Context, limit int) ([]system.MonitoringSample, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, source, cpu_percent, ram_percent, created_at
		FROM monitoring_samples
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query monitoring samples: %w", err)
	}
	defer rows.Close()

	samples := []system.MonitoringSample{}
	for rows.Next() {
		var s system.MonitoringSample
		if err := rows.Scan(&s.ID, &s.Source, &s.CPUPercent, &s.RAMPercent, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan monitoring sample: %w", err)
		}
		samples = append(samples, s)
	}

	return samples, nil
}
