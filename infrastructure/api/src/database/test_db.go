package database

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// NewTestDatabase creates an in-memory SQLite database for testing using sqlx.
// This allows testing real SQL queries and repositories without needing a full Postgres instance.
func NewTestDatabase(logger *slog.Logger) (*DBX, error) {
	// Open in-memory SQLite database with sqlx
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	// Create users table schema (matching Postgres schema, SQLite syntax)
	usersSchema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		email_verified BOOLEAN NOT NULL DEFAULT FALSE,
		verified_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	// Create honeyfiles table schema
	honeyfilesSchema := `
	CREATE TABLE IF NOT EXISTS honeyfiles (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		file_path TEXT NOT NULL UNIQUE,
		file_type TEXT NOT NULL,
		created_by TEXT,
		trigger_count INTEGER NOT NULL DEFAULT 0,
		last_triggered_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	// Create honeyfile_events table schema
	honeyfileEventsSchema := `
	CREATE TABLE IF NOT EXISTS honeyfile_events (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		honeyfile_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		source_ip TEXT,
		user_agent TEXT,
		request_path TEXT,
		metadata TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (honeyfile_id) REFERENCES honeyfiles(id)
	);`

	// Create webauthn_credentials table schema (MFA second factor)
	webauthnSchema := `
	CREATE TABLE IF NOT EXISTS webauthn_credentials (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		credential_id TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL DEFAULT '',
		credential TEXT NOT NULL,
		sign_count BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL,
		last_used_at TIMESTAMP
	);`

	// Create recovery_codes table schema (MFA backup codes). Declared with
	// TIMESTAMP (not Postgres' TIMESTAMPTZ) so the SQLite driver parses the
	// columns back into time.Time; production uses the repository's EnsureTable.
	recoveryCodesSchema := `
	CREATE TABLE IF NOT EXISTS recovery_codes (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		code_hash TEXT NOT NULL,
		used_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL
	);`

	// Execute all schema migrations
	for _, schema := range []string{usersSchema, honeyfilesSchema, honeyfileEventsSchema, webauthnSchema, recoveryCodesSchema} {
		if _, err := db.Exec(schema); err != nil {
			db.Close()
			return nil, err
		}
	}

	logger.Debug("✅ Test database (SQLite in-memory with sqlx) initialized")

	return &DBX{
		DB:     db,
		logger: logger,
	}, nil
}
