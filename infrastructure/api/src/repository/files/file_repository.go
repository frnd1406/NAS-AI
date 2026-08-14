package files_repo

import (
		"github.com/nas-ai/api/src/domain/files"
"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/sirupsen/logrus"
)

type FileRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

func NewFileRepository(db *sqlx.DB, logger *logrus.Logger) *FileRepository {
	return &FileRepository{
		db:     db,
		logger: logger,
	}
}

// Save creates or updates a file record
func (r *FileRepository) Save(ctx context.Context, file *files.File) error {
	query := `
		INSERT INTO files (
			id, owner_id, filename, mime_type, storage_path, 
			size_bytes, checksum, encryption_status, encryption_metadata, 
			created_at, updated_at, deleted_at
		) VALUES (
			:id, :owner_id, :filename, :mime_type, :storage_path, 
			:size_bytes, :checksum, :encryption_status, :encryption_metadata, 
			:created_at, :updated_at, :deleted_at
		)
		ON CONFLICT (id) DO UPDATE SET
			filename = :filename,
			mime_type = :mime_type,
			storage_path = :storage_path,
			size_bytes = :size_bytes,
			checksum = :checksum,
			encryption_status = :encryption_status,
			encryption_metadata = :encryption_metadata,
			updated_at = :updated_at,
			deleted_at = :deleted_at
	`
	_, err := r.db.NamedExecContext(ctx, query, file)
	if err != nil {
		r.logger.WithError(err).WithField("file_id", file.ID).Error("Failed to save file metadata")
		return fmt.Errorf("save file: %w", err)
	}
	return nil
}

// GetByID retrieves a file by ID
func (r *FileRepository) GetByID(ctx context.Context, id string) (*files.File, error) {
	var file files.File
	query := `SELECT * FROM files WHERE id = $1`
	err := r.db.GetContext(ctx, &file, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &file, nil
}

// DeleteSoft marks a file as deleted
func (r *FileRepository) DeleteSoft(ctx context.Context, id string) error {
	query := `UPDATE files SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteHard permanently removes a file record
func (r *FileRepository) DeleteHard(ctx context.Context, id string) error {
	query := `DELETE FROM files WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
