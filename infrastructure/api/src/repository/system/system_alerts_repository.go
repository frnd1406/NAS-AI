package system_repo

import (
		"github.com/nas-ai/api/src/domain/system"
"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/sirupsen/logrus"
)

// SystemAlertsRepository handles access to system_alerts table.
type SystemAlertsRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

func NewSystemAlertsRepository(db *sqlx.DB, logger *logrus.Logger) *SystemAlertsRepository {
	return &SystemAlertsRepository{
		db:     db,
		logger: logger,
	}
}

// ListOpen returns non-resolved alerts ordered by creation time.
func (r *SystemAlertsRepository) ListOpen(ctx context.Context) ([]system.SystemAlert, error) {
	alerts := []system.SystemAlert{}
	query := `
		SELECT id, severity, message, is_resolved, created_at
		FROM system_alerts
		WHERE is_resolved = FALSE
		ORDER BY created_at DESC
	`

	if err := r.db.SelectContext(ctx, &alerts, query); err != nil {
		r.logger.WithError(err).Error("failed to load open system alerts")
		return nil, fmt.Errorf("failed to load open alerts: %w", err)
	}

	return alerts, nil
}

// Resolve marks an alert as resolved. Returns false if no matching open alert was found.
func (r *SystemAlertsRepository) Resolve(ctx context.Context, id string) (bool, error) {
	query := `
		UPDATE system_alerts
		SET is_resolved = TRUE
		WHERE id = $1 AND is_resolved = FALSE
	`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).Error("failed to resolve alert")
		return false, fmt.Errorf("failed to resolve alert: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to inspect affected rows: %w", err)
	}

	return rows > 0, nil
}

// HasOpenBySeverity checks if an open alert with the given severity exists.
func (r *SystemAlertsRepository) HasOpenBySeverity(ctx context.Context, severity string) (bool, error) {
	query := `
		SELECT 1
		FROM system_alerts
		WHERE is_resolved = FALSE AND severity = $1
		LIMIT 1
	`

	var exists int
	if err := r.db.QueryRowContext(ctx, query, severity).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check open alerts: %w", err)
	}

	return true, nil
}

// Create inserts a new alert.
func (r *SystemAlertsRepository) Create(ctx context.Context, severity, message string) error {
	query := `
		INSERT INTO system_alerts (severity, message)
		VALUES ($1, $2)
	`

	if _, err := r.db.ExecContext(ctx, query, severity, message); err != nil {
		r.logger.WithError(err).Error("failed to create alert")
		return fmt.Errorf("failed to create alert: %w", err)
	}

	return nil
}
