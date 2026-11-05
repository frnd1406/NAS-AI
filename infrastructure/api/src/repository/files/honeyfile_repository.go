package files_repo

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// Honeyfile represents a trap file for intrusion detection
type Honeyfile struct {
	ID              uuid.UUID    `db:"id"`
	FilePath        string       `db:"file_path"`
	FileType        string       `db:"file_type"` // 'finance', 'it', 'private', 'general'
	TriggerCount    int          `db:"trigger_count"`
	LastTriggeredAt sql.NullTime `db:"last_triggered_at"`
	CreatedAt       time.Time    `db:"created_at"`
	CreatedBy       *uuid.UUID   `db:"created_by"`
}

// HoneyfileEvent represents a forensic log entry
type HoneyfileEvent struct {
	HoneyfileID uuid.UUID      `db:"honeyfile_id"`
	IPAddress   string         `db:"ip_address"`
	UserAgent   string         `db:"user_agent"`
	UserID      *uuid.UUID     `db:"user_id"`
	Action      string         `db:"action"`
	Metadata    sql.NullString `db:"metadata"` // Using JSON string
}

// HoneyfileRepository handles DB operations for honeyfiles
type HoneyfileRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// NewHoneyfileRepository creates a new repository instance
func NewHoneyfileRepository(db *sqlx.DB, logger *logrus.Logger) *HoneyfileRepository {
	return &HoneyfileRepository{db: db, logger: logger}
}

// RecordEvent logs a forensic event for a triggered honeyfile
func (r *HoneyfileRepository) RecordEvent(ctx context.Context, honeyfileID uuid.UUID, event *HoneyfileEvent) error {
	// Query removed (unused)
	// Use explicit struct with db tags for named query or manual arguments
	// Since we don't have a struct defined in this file yet for Event, let's define it or pass args.
	// For safety, let's use positional args with sqlx or standard Exec.

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO honeyfile_events (honeyfile_id, ip_address, user_agent, user_id, action, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, honeyfileID, event.IPAddress, event.UserAgent, event.UserID, event.Action, event.Metadata)

	if err != nil {
		r.logger.WithError(err).Error("Failed to record honeyfile forensic event")
		return fmt.Errorf("record event: %w", err)
	}
	return nil
}

// GetIDByPath resolves a path to a honeyfile ID
func (r *HoneyfileRepository) GetIDByPath(ctx context.Context, rawPath string) (uuid.UUID, error) {
	cleanPath := filepath.Clean(rawPath)
	var id uuid.UUID
	err := r.db.GetContext(ctx, &id, "SELECT id FROM honeyfiles WHERE file_path = $1", cleanPath)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// IncrementTrigger updates the trigger count and timestamp (Overview Statistics)
// Returns the Honeyfile ID for event logging
func (r *HoneyfileRepository) IncrementTrigger(ctx context.Context, rawPath string) (uuid.UUID, error) {
	cleanPath := filepath.Clean(rawPath)

	var id uuid.UUID
	// We do a returning ID to optimize two steps into one if possible,
	// or we select first. Since this is an alarm, correctness > speed.
	// Let's use RETURNING to get the ID for the event log.

	err := r.db.QueryRowContext(ctx, `
		UPDATE honeyfiles 
		SET trigger_count = trigger_count + 1, 
		    last_triggered_at = NOW() 
		WHERE file_path = $1
		RETURNING id
	`, cleanPath).Scan(&id)

	if err != nil {
		return uuid.Nil, fmt.Errorf("increment trigger: %w", err)
	}

	r.logger.WithField("file_path", cleanPath).Warn("🚨 HONEYFILE TRIGGER RECORDED")
	return id, nil
}

// GetAllPaths returns just the paths for cache loading
func (r *HoneyfileRepository) GetAllPaths(ctx context.Context) ([]string, error) {
	var paths []string
	err := r.db.SelectContext(ctx, &paths, "SELECT file_path FROM honeyfiles")
	if err != nil {
		return nil, fmt.Errorf("get all paths: %w", err)
	}
	return paths, nil
}

// EnsureTable creates the honeyfiles table if it doesn't exist
func (r *HoneyfileRepository) EnsureTable(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS honeyfiles (
		id UUID PRIMARY KEY,
		file_path TEXT NOT NULL UNIQUE,
		file_type TEXT NOT NULL,
		trigger_count INT DEFAULT 0,
		last_triggered_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_by UUID
	);
	`
	_, err := r.db.ExecContext(ctx, schema)
	return err
}

// Create adds a new honeyfile
func (r *HoneyfileRepository) Create(ctx context.Context, filePath, fileType string, createdBy *uuid.UUID) (*Honeyfile, error) {
	h := &Honeyfile{
		ID:        uuid.New(),
		FilePath:  filePath,
		FileType:  fileType,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}

	query := `
		INSERT INTO honeyfiles (id, file_path, file_type, created_at, created_by)
		VALUES (:id, :file_path, :file_type, :created_at, :created_by)
	`
	_, err := r.db.NamedExecContext(ctx, query, h)
	if err != nil {
		return nil, err
	}
	return h, nil
}

// ListAll returns all honeyfiles
func (r *HoneyfileRepository) ListAll(ctx context.Context) ([]Honeyfile, error) {
	var files []Honeyfile
	err := r.db.SelectContext(ctx, &files, "SELECT * FROM honeyfiles ORDER BY created_at DESC")
	if err != nil {
		// Return empty list instead of nil if error checks are strict elsewhere, but error is error.
		// If error is sql.ErrNoRows (unlikely with Select), handle it.
		// SelectContext returns slice, so usually safe.
		return nil, err
	}
	// Important: If empty, sqlx SelectContext might return nil slice or empty slice.
	// Initializing files above ensures we return something sensible if Select works but returns 0.
	if files == nil {
		files = []Honeyfile{}
	}
	return files, nil
}

// Delete removes a honeyfile by Path (not ID) as per service usage
func (r *HoneyfileRepository) Delete(ctx context.Context, filePath string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM honeyfiles WHERE file_path = $1", filePath)
	return err
}
