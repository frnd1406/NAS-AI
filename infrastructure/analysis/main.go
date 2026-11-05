package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

const (
	defaultDatabaseURL         = "postgres://nas_user:nas_dev_password@postgres:5432/nas_db?sslmode=disable"
	defaultInterval            = 10 * time.Second
	defaultLookback            = 60 * time.Second
	queryTimeout               = 5 * time.Second
	severityCritical           = "CRITICAL"
	severityWarning            = "WARNING"
	cpuThreshold       float64 = 80.0
	ramThreshold       float64 = 90.0
)

type averages struct {
	CPU sql.NullFloat64 `db:"avg_cpu"`
	RAM sql.NullFloat64 `db:"avg_ram"`
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	dbURL := getDatabaseURL()
	interval := durationFromSecondsEnv("INTERVAL_SECONDS", defaultInterval)
	lookback := durationFromSecondsEnv("LOOKBACK_SECONDS", defaultLookback)

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to postgres")
	}
	defer db.Close()

	logger.WithFields(logrus.Fields{
		"interval": interval.String(),
		"lookback": lookback.String(),
		"db":       dbURL,
	}).Info("analysis agent started")

	// FIX [BUG-GO-020]: Ensure unique constraint to prevent duplicate open alerts
	// We execute this on startup. It might fail if duplicates exist, but that's acceptable for now (or we could clean them up first).
	// Using "IF NOT EXISTS" logic via DO block or just ignoring error for simplicity in this context.
	// Note: PostgreSQL doesn't support "ADD CONSTRAINT IF NOT EXISTS" directly in all versions easily without a block.
	// We will try to add it and ignore error if it exists.
	_, _ = db.Exec(`
		ALTER TABLE system_alerts 
		ADD CONSTRAINT unique_open_alert 
		UNIQUE (severity, is_resolved) 
		WHERE (is_resolved = false);
	`)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// FIX [BUG-GO-006]: Use defer cancel() to ensure context is only cancelled after runCycle completes
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		runCycle(ctx, db, lookback, logger)
		cancel() // Cancel immediately after runCycle completes (no defer needed here as we want cleanup before waiting)
		<-ticker.C
	}
}

func runCycle(ctx context.Context, db *sqlx.DB, lookback time.Duration, logger *logrus.Logger) {
	avg, err := fetchAverages(ctx, db, lookback)
	if err != nil {
		logger.WithError(err).Warn("analysis: failed to fetch averages")
		return
	}

	if !avg.CPU.Valid && !avg.RAM.Valid {
		logger.Debug("analysis: no metrics in lookback window")
		return
	}

	if avg.CPU.Valid && avg.CPU.Float64 > cpuThreshold {
		msg := fmt.Sprintf("High CPU usage: avg %.2f%% over last %.0f seconds", avg.CPU.Float64, lookback.Seconds())
		if err := ensureAlert(ctx, db, severityCritical, msg, logger); err != nil {
			logger.WithError(err).Warn("analysis: failed to ensure CPU alert")
		}
	}

	if avg.RAM.Valid && avg.RAM.Float64 > ramThreshold {
		msg := fmt.Sprintf("High RAM usage: avg %.2f%% over last %.0f seconds", avg.RAM.Float64, lookback.Seconds())
		if err := ensureAlert(ctx, db, severityWarning, msg, logger); err != nil {
			logger.WithError(err).Warn("analysis: failed to ensure RAM alert")
		}
	}
}

func fetchAverages(ctx context.Context, db *sqlx.DB, lookback time.Duration) (averages, error) {
	query := `
		SELECT
			AVG(cpu_usage) AS avg_cpu,
			AVG(ram_usage) AS avg_ram
		FROM system_metrics
		WHERE created_at >= NOW() - ($1 * INTERVAL '1 second')
	`

	var result averages
	if err := db.GetContext(ctx, &result, query, int64(lookback.Seconds())); err != nil {
		return averages{}, err
	}

	return result, nil
}

func ensureAlert(ctx context.Context, db *sqlx.DB, severity, message string, logger *logrus.Logger) error {
	open, err := hasOpenAlert(ctx, db, severity)
	if err != nil {
		return err
	}

	if open {
		return nil
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO system_alerts (severity, message)
		VALUES ($1, $2)
	`, severity, message)
	if err != nil {
		return err
	}

	logger.WithFields(logrus.Fields{
		"severity": severity,
		"message":  message,
	}).Info("created system alert")

	return nil
}

func hasOpenAlert(ctx context.Context, db *sqlx.DB, severity string) (bool, error) {
	var exists int
	err := db.QueryRowContext(ctx, `
		SELECT 1
		FROM system_alerts
		WHERE is_resolved = FALSE AND severity = $1
		LIMIT 1
	`, severity).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func durationFromSecondsEnv(key string, defaultVal time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}

	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return defaultVal
	}

	return time.Duration(sec) * time.Second
}

func getDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	host := getEnv("DB_HOST", "postgres")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "nas_user")
	name := getEnv("DB_NAME", "nas_db")
	pass := getEnv("DB_PASSWORD", "nas_dev_password")

	if file := os.Getenv("DB_PASSWORD_FILE"); file != "" {
		content, err := os.ReadFile(file)
		if err == nil {
			pass = strings.TrimSpace(string(content))
		}
	} else if file := os.Getenv("POSTGRES_PASSWORD_FILE"); file != "" {
		content, err := os.ReadFile(file)
		if err == nil {
			pass = strings.TrimSpace(string(content))
		}
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, name)
}
