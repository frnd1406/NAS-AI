package content

import (
	"context"
	"io"
	"mime/multipart"
	"os"

	"github.com/nas-ai/api/src/domain/files"
	"github.com/nas-ai/api/src/drivers/storage"
)

// StorageService defines the interface for storage operations
type StorageService interface {
	Save(dir string, file multipart.File, fileHeader *multipart.FileHeader) (*SaveResult, error)
	SaveWithEncryption(ctx context.Context, dir string, file multipart.File, fileHeader *multipart.FileHeader, mode files.EncryptionMode, password string) (*SaveResult, error)
	List(relPath string) ([]StorageEntry, error)
	Open(relPath string) (*os.File, os.FileInfo, string, error)
	Delete(relPath string) error
	ListTrash() ([]TrashEntry, error)
	RestoreFromTrash(id string) error
	DeleteFromTrash(id string) error
	Rename(oldRel, newName string) error
	Move(srcRel, dstRel string) error
	Mkdir(relPath string) error
	GetFullPath(relPath string) (string, error)
}

// HoneyfileService defines the interface for honeyfile operations
type HoneyfileServiceInterface interface {
	CheckAndTrigger(ctx context.Context, path string, meta RequestMetadata) bool
}

// EncryptedStorageServiceInterface defines contract for encrypted storage operations
type EncryptedStorageServiceInterface interface {
	SaveEncrypted(dir string, file multipart.File, fileHeader *multipart.FileHeader) (*SaveResult, error)
	OpenEncrypted(relPath string) (io.ReadCloser, os.FileInfo, string, error)
	ListEncrypted(relPath string) ([]storage.StorageEntry, error)
	DeleteEncrypted(relPath string) error
	IsEncryptionEnabled() bool
	GetEncryptedBasePath() string
	SetEncryptedBasePath(path string) error
}
