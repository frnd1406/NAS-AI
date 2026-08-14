package settings_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// SystemSettingsRepository persists key/value system settings.
type SystemSettingsRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

const (
	SystemSettingBackupSchedule  = "backup.schedule"
	SystemSettingBackupRetention = "backup.retention"
	SystemSettingBackupPath      = "backup.path"
)

func NewSystemSettingsRepository(db *sqlx.DB, logger *logrus.Logger) *SystemSettingsRepository {
	return &SystemSettingsRepository{db: db, logger: logger}
}

// EnsureTable creates the backing table when it does not yet exist.
func (r *SystemSettingsRepository) EnsureTable(ctx context.Context) error {
	query := `
        CREATE TABLE IF NOT EXISTS system_settings (
            key TEXT PRIMARY KEY,
            value TEXT NOT NULL,
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )
    `

	if _, err := r.db.ExecContext(ctx, query); err != nil {
		r.logger.WithError(err).Error("failed to ensure system_settings table")
		return fmt.Errorf("ensure system_settings table: %w", err)
	}
	return nil
}

// GetAll returns all settings as a simple map.
func (r *SystemSettingsRepository) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryxContext(ctx, "SELECT key, value FROM system_settings")
	if err != nil {
		return nil, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		out[key] = value
	}
	return out, rows.Err()
}

// UpsertMany writes multiple settings atomically.
func (r *SystemSettingsRepository) UpsertMany(ctx context.Context, values map[string]string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	stmt, err := tx.PreparexContext(ctx, `
        INSERT INTO system_settings (key, value, updated_at)
        VALUES ($1, $2, $3)
        ON CONFLICT (key) DO UPDATE
        SET value = EXCLUDED.value,
            updated_at = EXCLUDED.updated_at
    `)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare upsert: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for k, v := range values {
		if _, err := stmt.ExecContext(ctx, k, v, now); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("upsert setting %s: %w", k, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit settings: %w", err)
	}
	return nil
}
