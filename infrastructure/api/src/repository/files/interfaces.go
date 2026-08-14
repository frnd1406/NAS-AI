package files_repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/nas-ai/api/src/domain/files"
)

// FileRepositoryInterface defines methods for file metadata operations
type FileRepositoryInterface interface {
	Save(ctx context.Context, file *files.File) error
	GetByID(ctx context.Context, id string) (*files.File, error)
	DeleteSoft(ctx context.Context, id string) error
	DeleteHard(ctx context.Context, id string) error
}

// HoneyfileRepositoryInterface defines methods for honeyfile operations
type HoneyfileRepositoryInterface interface {
	RecordEvent(ctx context.Context, honeyfileID uuid.UUID, event *HoneyfileEvent) error
	GetIDByPath(ctx context.Context, rawPath string) (uuid.UUID, error)
	IncrementTrigger(ctx context.Context, rawPath string) (uuid.UUID, error)
	GetAllPaths(ctx context.Context) ([]string, error)
	EnsureTable(ctx context.Context) error
	Create(ctx context.Context, filePath, fileType string, createdBy *uuid.UUID) (*Honeyfile, error)
	ListAll(ctx context.Context) ([]Honeyfile, error)
	Delete(ctx context.Context, filePath string) error
}

// FileEmbeddingRepositoryInterface defines methods for embedding consistency
type FileEmbeddingRepositoryInterface interface {
	GetOrphanCandidates(ctx context.Context, limit, offset int) ([]FileEmbeddingEntry, error)
	CountAll(ctx context.Context) (int, error)
	DeleteByFileID(ctx context.Context, fileID string) (int64, error)
}
