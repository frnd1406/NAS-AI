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

// WebAuthnCredentialRecord is a stored FIDO2/WebAuthn credential. The verified
// credential itself is kept as opaque JSON (CredentialJSON) so this repository
// stays decoupled from the WebAuthn library's type layout; the service layer
// marshals/unmarshals it.
type WebAuthnCredentialRecord struct {
	ID             string
	UserID         string
	CredentialID   string // base64url of the raw credential ID (for lookups)
	Name           string
	CredentialJSON []byte
	SignCount      int64
	CreatedAt      time.Time
	LastUsedAt     sql.NullTime
}

// WebAuthnCredentialRepository persists WebAuthn credentials.
type WebAuthnCredentialRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// NewWebAuthnCredentialRepository creates a new repository.
func NewWebAuthnCredentialRepository(db *sqlx.DB, logger *logrus.Logger) *WebAuthnCredentialRepository {
	return &WebAuthnCredentialRepository{db: db, logger: logger}
}

// EnsureTable creates the backing table when it does not yet exist.
func (r *WebAuthnCredentialRepository) EnsureTable(ctx context.Context) error {
	// Portable DDL (works on Postgres and the SQLite test DB): timestamps are
	// always supplied by the application, so no DB-side NOW() default is used.
	query := `
        CREATE TABLE IF NOT EXISTS webauthn_credentials (
            id            TEXT PRIMARY KEY,
            user_id       TEXT NOT NULL,
            credential_id TEXT NOT NULL UNIQUE,
            name          TEXT NOT NULL DEFAULT '',
            credential    TEXT NOT NULL,
            sign_count    BIGINT NOT NULL DEFAULT 0,
            created_at    TIMESTAMPTZ NOT NULL,
            last_used_at  TIMESTAMPTZ
        )
    `
	if _, err := r.db.ExecContext(ctx, query); err != nil {
		r.logger.WithError(err).Error("failed to ensure webauthn_credentials table")
		return fmt.Errorf("ensure webauthn_credentials table: %w", err)
	}

	// Index for per-user lookups (idempotent).
	if _, err := r.db.ExecContext(ctx,
		`CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id)`); err != nil {
		r.logger.WithError(err).Warn("failed to ensure webauthn_credentials user_id index")
	}
	return nil
}

// Create stores a new credential for a user.
func (r *WebAuthnCredentialRepository) Create(ctx context.Context, userID, name, credentialID string, credentialJSON []byte, signCount int64) error {
	query := `
        INSERT INTO webauthn_credentials (id, user_id, credential_id, name, credential, sign_count, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `
	_, err := r.db.ExecContext(ctx, query,
		uuid.NewString(), userID, credentialID, name, string(credentialJSON), signCount, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert webauthn credential: %w", err)
	}
	return nil
}

// ListByUserID returns all credentials for a user, newest first.
func (r *WebAuthnCredentialRepository) ListByUserID(ctx context.Context, userID string) ([]WebAuthnCredentialRecord, error) {
	query := `
        SELECT id, user_id, credential_id, name, credential, sign_count, created_at, last_used_at
        FROM webauthn_credentials
        WHERE user_id = $1
        ORDER BY created_at DESC
    `
	rows, err := r.db.QueryxContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query webauthn credentials: %w", err)
	}
	defer rows.Close()

	var out []WebAuthnCredentialRecord
	for rows.Next() {
		var rec WebAuthnCredentialRecord
		var credential string
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.CredentialID, &rec.Name, &credential, &rec.SignCount, &rec.CreatedAt, &rec.LastUsedAt); err != nil {
			return nil, fmt.Errorf("scan webauthn credential: %w", err)
		}
		rec.CredentialJSON = []byte(credential)
		out = append(out, rec)
	}
	return out, rows.Err()
}

// CountByUserID returns how many credentials a user has registered. Used by the
// login flow to decide whether a second factor is required.
func (r *WebAuthnCredentialRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM webauthn_credentials WHERE user_id = $1`, userID); err != nil {
		return 0, fmt.Errorf("count webauthn credentials: %w", err)
	}
	return count, nil
}

// UpdateAfterLogin persists the post-assertion credential state (sign count and
// the refreshed credential JSON, which may carry clone-detection flags).
func (r *WebAuthnCredentialRepository) UpdateAfterLogin(ctx context.Context, credentialID string, credentialJSON []byte, signCount int64) error {
	query := `
        UPDATE webauthn_credentials
        SET credential = $1, sign_count = $2, last_used_at = $3
        WHERE credential_id = $4
    `
	_, err := r.db.ExecContext(ctx, query, string(credentialJSON), signCount, time.Now().UTC(), credentialID)
	if err != nil {
		return fmt.Errorf("update webauthn credential: %w", err)
	}
	return nil
}

// Delete removes a credential owned by the given user.
func (r *WebAuthnCredentialRepository) Delete(ctx context.Context, userID, id string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM webauthn_credentials WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete webauthn credential: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
