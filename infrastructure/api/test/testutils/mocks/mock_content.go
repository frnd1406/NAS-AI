package mocks

import (
	"context"
	"io"
	"mime/multipart"
	"os"

	"github.com/nas-ai/api/src/domain/files"
	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/content"
	"github.com/stretchr/testify/mock"
)

// ============================================================
// MockStorageService (implements content.StorageService)
// ============================================================

type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) Save(dir string, file multipart.File, fileHeader *multipart.FileHeader) (*content.SaveResult, error) {
	args := m.Called(dir, file, fileHeader)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*content.SaveResult), args.Error(1)
}

func (m *MockStorageService) SaveWithEncryption(ctx context.Context, dir string, file multipart.File, fileHeader *multipart.FileHeader, mode files.EncryptionMode, password string) (*content.SaveResult, error) {
	args := m.Called(ctx, dir, file, fileHeader, mode, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*content.SaveResult), args.Error(1)
}

func (m *MockStorageService) Open(relPath string) (*os.File, os.FileInfo, string, error) {
	args := m.Called(relPath)
	f, _ := args.Get(0).(*os.File)
	fi, _ := args.Get(1).(os.FileInfo)
	s := args.String(2)
	return f, fi, s, args.Error(3)
}

func (m *MockStorageService) List(relPath string) ([]content.StorageEntry, error) {
	args := m.Called(relPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]content.StorageEntry), args.Error(1)
}

func (m *MockStorageService) Delete(relPath string) error {
	args := m.Called(relPath)
	return args.Error(0)
}

func (m *MockStorageService) ListTrash() ([]content.TrashEntry, error) {
	args := m.Called()
	return args.Get(0).([]content.TrashEntry), args.Error(1)
}

func (m *MockStorageService) RestoreFromTrash(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStorageService) DeleteFromTrash(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStorageService) Rename(oldRel, newName string) error {
	args := m.Called(oldRel, newName)
	return args.Error(0)
}

func (m *MockStorageService) Move(srcRel, dstRel string) error {
	args := m.Called(srcRel, dstRel)
	return args.Error(0)
}

func (m *MockStorageService) Mkdir(relPath string) error {
	args := m.Called(relPath)
	return args.Error(0)
}

func (m *MockStorageService) GetFullPath(relPath string) (string, error) {
	args := m.Called(relPath)
	return args.String(0), args.Error(1)
}

// Convenience methods
func (m *MockStorageService) UploadFile(ctx context.Context, fileHeader *multipart.FileHeader) (interface{}, error) {
	args := m.Called(ctx, fileHeader)
	return args.Get(0), args.Error(1)
}

func (m *MockStorageService) DownloadFile(path string) (*os.File, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*os.File), args.Error(1)
}

func (m *MockStorageService) DeleteFile(path string) error {
	return m.Delete(path)
}

func (m *MockStorageService) ListFiles(path string) ([]interface{}, error) {
	args := m.Called(path)
	return args.Get(0).([]interface{}), args.Error(1)
}

// ============================================================
// MockHoneyfileService (implements content.HoneyfileServiceInterface)
// ============================================================

type MockHoneyfileService struct {
	mock.Mock
}

func (m *MockHoneyfileService) CheckAndTrigger(ctx context.Context, path string, meta content.RequestMetadata) bool {
	args := m.Called(ctx, path, meta)
	return args.Bool(0)
}

func (m *MockHoneyfileService) IsHoneyfile(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

// ============================================================
// MockEncryptedStorageService (implements content.EncryptedStorageServiceInterface)
// ============================================================

type MockEncryptedStorageService struct {
	mock.Mock
}

func (m *MockEncryptedStorageService) SaveEncrypted(dir string, file multipart.File, fileHeader *multipart.FileHeader) (*content.SaveResult, error) {
	args := m.Called(dir, file, fileHeader)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*content.SaveResult), args.Error(1)
}

func (m *MockEncryptedStorageService) OpenEncrypted(relPath string) (io.ReadCloser, os.FileInfo, string, error) {
	args := m.Called(relPath)
	if args.Get(0) == nil {
		return nil, nil, "", args.Error(3)
	}
	return args.Get(0).(io.ReadCloser), args.Get(1).(os.FileInfo), args.String(2), args.Error(3)
}

func (m *MockEncryptedStorageService) ListEncrypted(relPath string) ([]storage.StorageEntry, error) {
	args := m.Called(relPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]storage.StorageEntry), args.Error(1)
}

func (m *MockEncryptedStorageService) DeleteEncrypted(relPath string) error {
	args := m.Called(relPath)
	return args.Error(0)
}

func (m *MockEncryptedStorageService) IsEncryptionEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockEncryptedStorageService) GetEncryptedBasePath() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEncryptedStorageService) SetEncryptedBasePath(path string) error {
	args := m.Called(path)
	return args.Error(0)
}
