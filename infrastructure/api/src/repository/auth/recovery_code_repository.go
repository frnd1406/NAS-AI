package auth_repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// RecoveryCodeRecord is a single stored recovery code. Only the bcrypt hash of
// the code is persisted; the plaintext is shown to the user once at generation.
type RecoveryCodeRecord struct {
	ID        string
	UserID    string
	CodeHash  string
	UsedAt    sql.NullTime
	CreatedAt time.Time
}

// RecoveryCodeRepository persists one-time recovery (backup) codes for 2FA.
type RecoveryCodeRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// NewRecoveryCodeRepository creates a new repository.
func NewRecoveryCodeRepository(db *sqlx.DB, logger *logrus.Logger) *RecoveryCodeRepository {
	return &RecoveryCodeRepository{db: db, logger: logger}
}

// EnsureTable creates the backing table when it does not yet exist.
func (r *RecoveryCodeRepository) EnsureTable(ctx context.Context) error {
	// Portable DDL (Postgres and the SQLite test DB); timestamps are supplied by
	// the application, matching the webauthn_credentials convention.
	query := `
        CREATE TABLE IF NOT EXISTS recovery_codes (
            id         TEXT PRIMARY KEY,
            user_id    TEXT NOT NULL,
            code_hash  TEXT NOT NULL,
            used_at    TIMESTAMPTZ,
            created_at TIMESTAMPTZ NOT NULL
        )
    `
	if _, err := r.db.ExecContext(ctx, query); err != nil {
		r.logger.WithError(err).Error("failed to ensure recovery_codes table")
		return fmt.Errorf("ensure recovery_codes table: %w", err)
	}

	if _, err := r.db.ExecContext(ctx,
		`CREATE INDEX IF NOT EXISTS idx_recovery_codes_user_id ON recovery_codes(user_id)`); err != nil {
		r.logger.WithError(err).Warn("failed to ensure recovery_codes user_id index")
	}
	return nil
}

// ReplaceForUser atomically deletes any existing codes for the user and inserts
// the given hashes as the new set. Used both at first generation and when the
// user regenerates their codes.
func (r *RecoveryCodeRepository) ReplaceForUser(ctx context.Context, userID string, codeHashes []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin recovery codes tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM recovery_codes WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clear recovery codes: %w", err)
	}

	now := time.Now().UTC()
	for _, hash := range codeHashes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO recovery_codes (id, user_id, code_hash, created_at) VALUES ($1, $2, $3, $4)`,
			uuid.NewString(), userID, hash, now); err != nil {
			return fmt.Errorf("insert recovery code: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit recovery codes: %w", err)
	}
	return nil
}

// ListUnusedByUserID returns the user's still-unused codes, oldest first.
func (r *RecoveryCodeRepository) ListUnusedByUserID(ctx context.Context, userID string) ([]RecoveryCodeRecord, error) {
	query := `
        SELECT id, user_id, code_hash, used_at, created_at
        FROM recovery_codes
        WHERE user_id = $1 AND used_at IS NULL
        ORDER BY created_at ASC
    `
	rows, err := r.db.QueryxContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query recovery codes: %w", err)
	}
	defer rows.Close()

	var out []RecoveryCodeRecord
	for rows.Next() {
		var rec RecoveryCodeRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.CodeHash, &rec.UsedAt, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recovery code: %w", err)
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// MarkUsed consumes a single code by id. It returns true only if this call is
// the one that flipped the row from unused to used, so a concurrent second use
// of the same code returns false rather than granting access twice.
func (r *RecoveryCodeRepository) MarkUsed(ctx context.Context, id string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE recovery_codes SET used_at = $1 WHERE id = $2 AND used_at IS NULL`,
		time.Now().UTC(), id)
	if err != nil {
		return false, fmt.Errorf("mark recovery code used: %w", err)
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// CountUnusedByUserID returns how many unused codes the user has left.
func (r *RecoveryCodeRepository) CountUnusedByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM recovery_codes WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return 0, fmt.Errorf("count recovery codes: %w", err)
	}
	return count, nil
}
