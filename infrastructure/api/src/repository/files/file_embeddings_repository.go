package files_repo

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// FileEmbeddingEntry represents a file embedding record for consistency checks
type FileEmbeddingEntry struct {
	ID       string  `db:"id"`
	FileID   string  `db:"file_id"`
	FilePath *string `db:"file_path"` // Nullable - extracted from metadata JSONB
}

// FileEmbeddingsRepository provides access to file_embeddings table
// Used by ConsistencyService to detect and remove orphaned vectors
type FileEmbeddingsRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// NewFileEmbeddingsRepository creates a new repository instance
func NewFileEmbeddingsRepository(db *sqlx.DB, logger *logrus.Logger) *FileEmbeddingsRepository {
	return &FileEmbeddingsRepository{db: db, logger: logger}
}

// GetOrphanCandidates fetches paginated file entries for consistency checking
// Returns unique file_id entries with their file_path from metadata JSONB
func (r *FileEmbeddingsRepository) GetOrphanCandidates(ctx context.Context, limit, offset int) ([]FileEmbeddingEntry, error) {
	// Query unique file_ids with their file_path from metadata
	// Using DISTINCT ON to get one row per file_id (avoiding duplicates from chunks)
	query := `
		SELECT DISTINCT ON (file_id) 
			id,
			file_id,
			metadata->>'file_path' as file_path
		FROM file_embeddings
		ORDER BY file_id, created_at DESC
		LIMIT $1 OFFSET $2
	`

	var entries []FileEmbeddingEntry
	if err := r.db.SelectContext(ctx, &entries, query, limit, offset); err != nil {
		r.logger.WithError(err).Error("failed to fetch orphan candidates")
		return nil, fmt.Errorf("fetch orphan candidates: %w", err)
	}

	return entries, nil
}

// CountAll returns total count of unique file_ids in file_embeddings
func (r *FileEmbeddingsRepository) CountAll(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(DISTINCT file_id) FROM file_embeddings`
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		return 0, fmt.Errorf("count file embeddings: %w", err)
	}
	return count, nil
}

// DeleteByFileID removes all embeddings (all chunks) for a given file_id
// Returns number of rows deleted
func (r *FileEmbeddingsRepository) DeleteByFileID(ctx context.Context, fileID string) (int64, error) {
	query := `DELETE FROM file_embeddings WHERE file_id = $1`

	result, err := r.db.ExecContext(ctx, query, fileID)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"file_id": fileID,
			"error":   err.Error(),
		}).Error("failed to delete embeddings by file_id")
		return 0, fmt.Errorf("delete embeddings for %s: %w", fileID, err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}
